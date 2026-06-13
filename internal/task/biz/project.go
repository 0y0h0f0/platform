package biz

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"task-platform/internal/task/data"
	"task-platform/pkg/xerr"
	"task-platform/pkg/xredis"
)

const (
	projectCacheTTL    = 5 * time.Minute
	projectCachePrefix = "project:"
)

// UserServiceClient is the task service's narrow dependency on user-service.
// Keeping only the validation method here avoids coupling task business rules
// to the generated user-service client.
type UserServiceClient interface {
	GetUser(ctx context.Context, userID string) (userExists bool, userActive bool, err error)
}

// ProjectBiz coordinates project ownership, membership and audit logging rules.
type ProjectBiz struct {
	db          *gorm.DB
	projectRepo data.ProjectRepository
	memberRepo  data.MemberRepository
	userClient  UserServiceClient
	logWriter   *LogWriter
	rdb         *redis.Client
	sf          singleflight.Group
}

// NewProjectBiz wires project business logic with repositories, user validation,
// operation logging and optional Redis-backed project caching.
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

// CreateProject creates a project and inserts the creator as the owner in one
// transaction so a project never exists without its owner membership.
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

// GetProject verifies membership before returning project data. Missing
// membership is reported as NOT_FOUND so private project existence is not leaked.
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

	// Singleflight collapses concurrent cache misses for the same project.
	v, err, _ := b.sf.Do(projectID, func() (any, error) {
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
	})
	if err != nil {
		return nil, err
	}
	return v.(*data.Project), nil
}

// UpdateProject changes editable project fields with optimistic locking.
// The caller must be the owner and archived projects remain read-only.
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

// ArchiveProject marks a project read-only. Only the owner can archive it.
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

// UnarchiveProject restores writes for an archived project. Only the owner can
// bring the project back to active state.
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

// TransferOwnership promotes an existing active member to owner and demotes the
// current owner to admin in the same transaction.
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

		// Re-read project inside transaction to get the latest version for
		// optimistic locking, preventing a TOCTOU race where the project is
		// modified between the initial read and the transfer.
		currentProject, err := projectRepo.FindByID(ctx, projectID)
		if err != nil {
			return err
		}

		// Re-verify both memberships inside the transaction to prevent TOCTOU
		// races where a member is removed between the initial checks and the transfer.
		callerMember, err := memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
		if err != nil {
			return err
		}
		if callerMember == nil {
			return xerr.NewError(xerr.CodeFailedPrecondition, "caller is no longer a project member")
		}
		if callerMember.Role != data.RoleOwner {
			return xerr.NewError(xerr.CodePermissionDenied, "caller is no longer the owner")
		}

		targetMember, err := memberRepo.FindByProjectAndUser(ctx, projectID, targetUserID)
		if err != nil {
			return err
		}
		if targetMember == nil {
			return xerr.NewError(xerr.CodeFailedPrecondition, "target user is no longer a project member")
		}

		if _, err := projectRepo.Update(ctx, projectID, currentProject.Version, map[string]any{
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
		Detail:     jsonDetail(map[string]string{"from_user_id": callerID, "to_user_id": targetUserID}),
	})
	b.invalidateProjectCache(ctx, projectID)
	return updated, nil
}

// AddProjectMember invites an active user into a project. Owners may add admins
// or members; admins may only add regular members.
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

	// Use a transaction with project version check to prevent writes on
	// archived projects between the status check and the member insert.
	var member *data.ProjectMember
	err = b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		projectRepo := data.NewProjectRepository(tx)
		memberRepo := data.NewMemberRepository(tx)

		current, err := projectRepo.FindByID(ctx, projectID)
		if err != nil {
			return err
		}
		if current.Status == data.ProjectStatusArchived {
			return xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
		}

		m := &data.ProjectMember{
			ProjectID: projectID,
			UserID:    targetUserID,
			Role:      role,
		}
		if err := memberRepo.Add(ctx, m); err != nil {
			return err
		}
		member = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		OperatorID: callerID,
		Action:     data.ActionMemberAdd,
		Detail:     jsonDetail(map[string]string{"user_id": targetUserID}),
	})
	return member, nil
}

// RemoveProjectMember removes another member according to role hierarchy.
// Self-removal is handled by LeaveProject so owner-transfer checks stay explicit.
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

	err = b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		memberRepo := data.NewMemberRepository(tx)
		projectRepo := data.NewProjectRepository(tx)

		// Re-verify the project is not archived inside the transaction.
		currentProject, err := projectRepo.FindByID(ctx, projectID)
		if err != nil {
			return err
		}
		if currentProject.Status == data.ProjectStatusArchived {
			return xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
		}

		// Re-verify the caller is still a member with sufficient role.
		currentCaller, err := memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
		if err != nil {
			return err
		}
		if currentCaller == nil {
			return xerr.NewError(xerr.CodeFailedPrecondition, "caller is no longer a project member")
		}

		switch currentCaller.Role {
		case data.RoleOwner:
		case data.RoleAdmin:
			currentTarget, err := memberRepo.FindByProjectAndUser(ctx, projectID, targetUserID)
			if err != nil {
				return err
			}
			if currentTarget == nil {
				return xerr.NewError(xerr.CodeNotFound, "target user is no longer a member")
			}
			if currentTarget.Role == data.RoleOwner || currentTarget.Role == data.RoleAdmin {
				return xerr.NewError(xerr.CodePermissionDenied, "admin cannot remove owner or other admins")
			}
		default:
			return xerr.NewError(xerr.CodePermissionDenied, "no permission to remove members")
		}

		if err := memberRepo.Remove(ctx, projectID, targetUserID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		OperatorID: callerID,
		Action:     data.ActionMemberRemove,
		Detail:     jsonDetail(map[string]string{"user_id": targetUserID}),
	})
	return targetMember, nil
}

// UpdateProjectMemberRole changes admin/member roles. Owner changes are routed
// through TransferOwnership to keep exactly one owner for the project.
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
		Detail:     jsonDetail(map[string]string{"user_id": targetUserID}),
	})
	return updated, nil
}

// LeaveProject lets a non-owner member leave an active project.
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

// ListProjectMembers returns members only after confirming the caller belongs
// to the project.
func (b *ProjectBiz) ListProjectMembers(ctx context.Context, projectID, callerID string) ([]*data.ProjectMember, error) {
	if _, err := b.requireMember(ctx, projectID, callerID); err != nil {
		return nil, err
	}
	return b.memberRepo.ListByProject(ctx, projectID)
}

// CheckProjectMember is used by callers that need a fast membership probe
// without surfacing NOT_FOUND as an error.
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

// ListProjects returns projects visible to the caller, capped to a conservative
// page size for gateway-driven lists.
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

	// Archived projects can still be read by members; write paths layer their own
	// read-only checks on top of this shared membership lookup.
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
		if !errors.Is(err, redis.Nil) {
			log.Printf("WARN: project cache get failed for %s: %v", projectID, err)
		} else {
			xredis.IncrCacheMiss()
		}
		return nil
	}
	xredis.IncrCacheHit()
	var p data.Project
	if err := json.Unmarshal(raw, &p); err != nil {
		log.Printf("WARN: project cache deserialize failed for %s: %v", projectID, err)
		return nil
	}
	return &p
}

func (b *ProjectBiz) cacheProject(ctx context.Context, p *data.Project) {
	raw, err := json.Marshal(p)
	if err != nil {
		log.Printf("WARN: project cache marshal failed for %s: %v", p.ID, err)
		return
	}
	if err := b.rdb.Set(ctx, projectCachePrefix+p.ID, raw, projectCacheTTL).Err(); err != nil {
		log.Printf("WARN: project cache set failed for %s: %v", p.ID, err)
	}
}

func (b *ProjectBiz) invalidateProjectCache(ctx context.Context, projectID string) {
	if b.rdb != nil {
		if err := b.rdb.Del(ctx, projectCachePrefix+projectID).Err(); err != nil {
			log.Printf("WARN: project cache invalidation failed for %s: %v", projectID, err)
		}
	}
}
