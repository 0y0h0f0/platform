package biz

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"task-platform/internal/task/data"
	"task-platform/pkg/xerr"
)

const (
	projectCacheTTL    = 5 * time.Minute
	projectCachePrefix = "project:"
)

type UserServiceClient interface {
	GetUser(ctx context.Context, userID string) (userExists bool, userActive bool, err error)
}

type ProjectBiz struct {
	db          *gorm.DB
	projectRepo data.ProjectRepository
	memberRepo  data.MemberRepository
	userClient  UserServiceClient
	logWriter   *LogWriter
	rdb         *redis.Client
}

func NewProjectBiz(db *gorm.DB, projectRepo data.ProjectRepository, memberRepo data.MemberRepository, userClient UserServiceClient, logWriter *LogWriter, rdb *redis.Client) *ProjectBiz {
	return &ProjectBiz{
		db:          db,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		userClient:  userClient,
		logWriter:   logWriter,
		rdb:         rdb,
	}
}

func (b *ProjectBiz) CreateProject(ctx context.Context, callerID, name, description string) (*data.Project, error) {
	if !data.IsValidProjectName(name) {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "project name must be 1-100 characters")
	}

	var project *data.Project
	err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p := &data.Project{
			Name:        name,
			Description: description,
			OwnerID:     callerID,
		}
		if err := data.NewProjectRepository(tx).Create(ctx, p); err != nil {
			return err
		}

		m := &data.ProjectMember{
			ProjectID: p.ID,
			UserID:    callerID,
			Role:      data.RoleOwner,
		}
		if err := data.NewMemberRepository(tx).Add(ctx, m); err != nil {
			return err
		}

		project = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &project.ID,
		OperatorID: callerID,
		Action:     data.ActionProjectCreate,
		Detail:     `{}`,
	})
	return project, nil
}

func (b *ProjectBiz) GetProject(ctx context.Context, projectID, callerID string) (*data.Project, error) {
	member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, xerr.NewError(xerr.CodeNotFound, "project not found")
	}

	if b.rdb != nil {
		if p := b.getCachedProject(ctx, projectID); p != nil {
			return p, nil
		}
	}

	project, err := b.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if b.rdb != nil {
		b.cacheProject(ctx, project)
	}

	return project, nil
}

func (b *ProjectBiz) UpdateProject(ctx context.Context, projectID, callerID, name, description string, version int64) (*data.Project, error) {
	if name != "" && !data.IsValidProjectName(name) {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "project name must be 1-100 characters")
	}

	project, member, err := b.getProjectAndMember(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}

	if member.Role != data.RoleOwner {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "only owner can update project")
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}

	updates := make(map[string]any)
	if name != "" {
		updates["name"] = name
	}
	updates["description"] = description
	updates["version"] = project.Version + 1

	updated, err := b.projectRepo.Update(ctx, projectID, version, updates)
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &updated.ID,
		OperatorID: callerID,
		Action:     data.ActionProjectUpdate,
		Detail:     `{}`,
	})
	b.invalidateProjectCache(ctx, updated.ID)
	return updated, nil
}

func (b *ProjectBiz) ArchiveProject(ctx context.Context, projectID, callerID string) (*data.Project, error) {
	_, member, err := b.getProjectAndMember(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if member.Role != data.RoleOwner {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "only owner can archive project")
	}

	project, err := b.setProjectStatus(ctx, projectID, data.ProjectStatusArchived)
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &project.ID,
		OperatorID: callerID,
		Action:     data.ActionProjectArchive,
		Detail:     `{}`,
	})
	b.invalidateProjectCache(ctx, project.ID)
	return project, nil
}

func (b *ProjectBiz) UnarchiveProject(ctx context.Context, projectID, callerID string) (*data.Project, error) {
	_, member, err := b.getProjectAndMember(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if member.Role != data.RoleOwner {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "only owner can unarchive project")
	}

	project, err := b.setProjectStatus(ctx, projectID, data.ProjectStatusActive)
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &project.ID,
		OperatorID: callerID,
		Action:     data.ActionProjectUnarchive,
		Detail:     `{}`,
	})
	b.invalidateProjectCache(ctx, project.ID)
	return project, nil
}

func (b *ProjectBiz) TransferOwnership(ctx context.Context, projectID, callerID, targetUserID string) (*data.Project, error) {
	if callerID == targetUserID {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "cannot transfer ownership to yourself")
	}

	project, member, err := b.getProjectAndMember(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}
	if member.Role != data.RoleOwner {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "only owner can transfer ownership")
	}

	targetMember, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetMember == nil {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "target user is not a project member")
	}

	exists, active, err := b.userClient.GetUser(ctx, targetUserID)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "validate target user failed")
	}
	if !exists || !active {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "target user does not exist or is disabled")
	}

	err = b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		projectRepo := data.NewProjectRepository(tx)
		memberRepo := data.NewMemberRepository(tx)

		if _, err := projectRepo.Update(ctx, projectID, project.Version, map[string]any{
			"owner_id": targetUserID,
			"version":  project.Version + 1,
		}); err != nil {
			return err
		}

		if _, err := memberRepo.UpdateRole(ctx, projectID, callerID, data.RoleAdmin); err != nil {
			return err
		}

		if _, err := memberRepo.UpdateRole(ctx, projectID, targetUserID, data.RoleOwner); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	updated, err := b.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		OperatorID: callerID,
		Action:     data.ActionProjectTransferOwner,
		Detail:     `{"from_user_id":"` + callerID + `","to_user_id":"` + targetUserID + `"}`,
	})
	b.invalidateProjectCache(ctx, projectID)
	return updated, nil
}

func (b *ProjectBiz) AddProjectMember(ctx context.Context, projectID, callerID, targetUserID string, role int32) (*data.ProjectMember, error) {
	if role != data.RoleAdmin && role != data.RoleMember {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "invalid role: can only add admin or member")
	}

	project, callerMember, err := b.getProjectAndMember(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}

	switch callerMember.Role {
	case data.RoleOwner:
	case data.RoleAdmin:
		if role != data.RoleMember {
			return nil, xerr.NewError(xerr.CodePermissionDenied, "admin can only invite members")
		}
	default:
		return nil, xerr.NewError(xerr.CodePermissionDenied, "no permission to add members")
	}

	exists, active, err := b.userClient.GetUser(ctx, targetUserID)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "validate target user failed")
	}
	if !exists || !active {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "target user does not exist or is disabled")
	}

	member := &data.ProjectMember{
		ProjectID: projectID,
		UserID:    targetUserID,
		Role:      role,
	}
	if err := b.memberRepo.Add(ctx, member); err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		OperatorID: callerID,
		Action:     data.ActionMemberAdd,
		Detail:     `{"user_id":"` + targetUserID + `"}`,
	})
	return member, nil
}

func (b *ProjectBiz) RemoveProjectMember(ctx context.Context, projectID, callerID, targetUserID string) (*data.ProjectMember, error) {
	project, callerMember, err := b.getProjectAndMember(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}

	if callerID == targetUserID {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "cannot remove yourself; use leave instead")
	}

	targetMember, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetMember == nil {
		return nil, xerr.NewError(xerr.CodeNotFound, "target user is not a member")
	}

	switch callerMember.Role {
	case data.RoleOwner:
	case data.RoleAdmin:
		if targetMember.Role == data.RoleOwner || targetMember.Role == data.RoleAdmin {
			return nil, xerr.NewError(xerr.CodePermissionDenied, "admin cannot remove owner or other admins")
		}
	default:
		return nil, xerr.NewError(xerr.CodePermissionDenied, "no permission to remove members")
	}

	if err := b.memberRepo.Remove(ctx, projectID, targetUserID); err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		OperatorID: callerID,
		Action:     data.ActionMemberRemove,
		Detail:     `{"user_id":"` + targetUserID + `"}`,
	})
	return targetMember, nil
}

func (b *ProjectBiz) UpdateProjectMemberRole(ctx context.Context, projectID, callerID, targetUserID string, role int32) (*data.ProjectMember, error) {
	if role != data.RoleAdmin && role != data.RoleMember {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "invalid role")
	}

	project, callerMember, err := b.getProjectAndMember(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}
	if callerMember.Role != data.RoleOwner {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "only owner can change member roles")
	}

	targetMember, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetMember == nil {
		return nil, xerr.NewError(xerr.CodeNotFound, "target user is not a member")
	}
	if targetMember.Role == data.RoleOwner {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "cannot change owner's role; use transfer ownership")
	}

	updated, err := b.memberRepo.UpdateRole(ctx, projectID, targetUserID, role)
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		OperatorID: callerID,
		Action:     data.ActionMemberRoleChange,
		Detail:     `{"user_id":"` + targetUserID + `"}`,
	})
	return updated, nil
}

func (b *ProjectBiz) LeaveProject(ctx context.Context, projectID, callerID string) (*data.ProjectMember, error) {
	member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, xerr.NewError(xerr.CodeNotFound, "project not found")
	}
	if member.Role == data.RoleOwner {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "owner cannot leave; transfer ownership first")
	}

	project, err := b.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}

	if err := b.memberRepo.Remove(ctx, projectID, callerID); err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		OperatorID: callerID,
		Action:     data.ActionMemberLeave,
		Detail:     `{}`,
	})
	return member, nil
}

func (b *ProjectBiz) ListProjectMembers(ctx context.Context, projectID, callerID string) ([]*data.ProjectMember, error) {
	if _, err := b.requireMember(ctx, projectID, callerID); err != nil {
		return nil, err
	}
	return b.memberRepo.ListByProject(ctx, projectID)
}

func (b *ProjectBiz) CheckProjectMember(ctx context.Context, projectID, userID string) (bool, int32, error) {
	member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		return false, 0, err
	}
	if member == nil {
		return false, 0, nil
	}
	return true, member.Role, nil
}

func (b *ProjectBiz) ListProjects(ctx context.Context, callerID string, includeArchived bool, limit, offset int) ([]*data.Project, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return b.projectRepo.FindByMemberID(ctx, callerID, includeArchived, limit, offset)
}

func (b *ProjectBiz) getProjectAndMember(ctx context.Context, projectID, callerID string) (*data.Project, *data.ProjectMember, error) {
	project, err := b.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}

	if project.Status == data.ProjectStatusArchived {
		member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
		if err != nil {
			return nil, nil, err
		}
		if member == nil {
			return nil, nil, xerr.NewError(xerr.CodeNotFound, "project not found")
		}
		return project, member, nil
	}

	member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
	if err != nil {
		return nil, nil, err
	}
	if member == nil {
		return nil, nil, xerr.NewError(xerr.CodeNotFound, "project not found")
	}

	return project, member, nil
}

func (b *ProjectBiz) requireMember(ctx context.Context, projectID, callerID string) (*data.ProjectMember, error) {
	_, member, err := b.getProjectAndMember(ctx, projectID, callerID)
	return member, err
}

func (b *ProjectBiz) setProjectStatus(ctx context.Context, projectID string, status int32) (*data.Project, error) {
	project, err := b.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return b.projectRepo.Update(ctx, projectID, project.Version, map[string]any{
		"status":  status,
		"version": project.Version + 1,
	})
}

func (b *ProjectBiz) getCachedProject(ctx context.Context, projectID string) *data.Project {
	raw, err := b.rdb.Get(ctx, projectCachePrefix+projectID).Bytes()
	if err != nil {
		return nil
	}
	var p data.Project
	if json.Unmarshal(raw, &p) != nil {
		return nil
	}
	return &p
}

func (b *ProjectBiz) cacheProject(ctx context.Context, p *data.Project) {
	raw, err := json.Marshal(p)
	if err != nil {
		return
	}
	b.rdb.Set(ctx, projectCachePrefix+p.ID, raw, projectCacheTTL)
}

func (b *ProjectBiz) invalidateProjectCache(ctx context.Context, projectID string) {
	if b.rdb != nil {
		b.rdb.Del(ctx, projectCachePrefix+projectID)
	}
}
