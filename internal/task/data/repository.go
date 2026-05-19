package data

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"task-platform/pkg/xerr"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	FindByID(ctx context.Context, id string) (*Project, error)
	Update(ctx context.Context, id string, version int64, updates map[string]any) (*Project, error)
	FindByMemberID(ctx context.Context, userID string, includeArchived bool, limit, offset int) ([]*Project, error)
}

type MemberRepository interface {
	Add(ctx context.Context, member *ProjectMember) error
	Remove(ctx context.Context, projectID, userID string) error
	FindByProjectAndUser(ctx context.Context, projectID, userID string) (*ProjectMember, error)
	ListByProject(ctx context.Context, projectID string) ([]*ProjectMember, error)
	UpdateRole(ctx context.Context, projectID, userID string, role int32) (*ProjectMember, error)
	FindOwnerByProject(ctx context.Context, projectID string) (*ProjectMember, error)
}

type projectRepo struct{ db *gorm.DB }
type memberRepo struct{ db *gorm.DB }

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepo{db: db}
}

func NewMemberRepository(db *gorm.DB) MemberRepository {
	return &memberRepo{db: db}
}

func (r *projectRepo) Create(ctx context.Context, project *Project) error {
	err := r.db.WithContext(ctx).Create(project).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return xerr.NewError(xerr.CodeAlreadyExists, "project name already exists for this user")
		}
		return xerr.NewError(xerr.CodeInternal, "create project failed")
	}
	return nil
}

func (r *projectRepo) FindByID(ctx context.Context, id string) (*Project, error) {
	var p Project
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NewError(xerr.CodeNotFound, "project not found")
	}
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "find project failed")
	}
	return &p, nil
}

func (r *projectRepo) Update(ctx context.Context, id string, version int64, updates map[string]any) (*Project, error) {
	result := r.db.WithContext(ctx).Model(&Project{}).
		Where("id = ? AND version = ?", id, version).
		Updates(updates)
	if result.Error != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "update project failed")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NewError(xerr.CodeAborted, "project was modified by another request, please retry")
	}
	return r.FindByID(ctx, id)
}

func (r *projectRepo) FindByMemberID(ctx context.Context, userID string, includeArchived bool, limit, offset int) ([]*Project, error) {
	query := r.db.WithContext(ctx).
		Joins("JOIN task_svc.project_members ON task_svc.project_members.project_id = task_svc.projects.id").
		Where("task_svc.project_members.user_id = ?", userID).
		Where("task_svc.projects.deleted_at IS NULL")

	if !includeArchived {
		query = query.Where("task_svc.projects.status = ?", ProjectStatusActive)
	}

	var projects []*Project
	err := query.Order("task_svc.projects.created_at DESC, task_svc.projects.id DESC").
		Limit(limit).Offset(offset).
		Find(&projects).Error
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "list projects failed")
	}
	return projects, nil
}

func (r *memberRepo) Add(ctx context.Context, member *ProjectMember) error {
	err := r.db.WithContext(ctx).Create(member).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return xerr.NewError(xerr.CodeAlreadyExists, "user is already a member of this project")
		}
		return xerr.NewError(xerr.CodeInternal, "add member failed")
	}
	return nil
}

func (r *memberRepo) Remove(ctx context.Context, projectID, userID string) error {
	result := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&ProjectMember{})
	if result.Error != nil {
		return xerr.NewError(xerr.CodeInternal, "remove member failed")
	}
	if result.RowsAffected == 0 {
		return xerr.NewError(xerr.CodeNotFound, "member not found")
	}
	return nil
}

func (r *memberRepo) FindByProjectAndUser(ctx context.Context, projectID, userID string) (*ProjectMember, error) {
	var m ProjectMember
	err := r.db.WithContext(ctx).Where("project_id = ? AND user_id = ?", projectID, userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "find member failed")
	}
	return &m, nil
}

func (r *memberRepo) ListByProject(ctx context.Context, projectID string) ([]*ProjectMember, error) {
	var members []*ProjectMember
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&members).Error
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "list members failed")
	}
	return members, nil
}

func (r *memberRepo) UpdateRole(ctx context.Context, projectID, userID string, role int32) (*ProjectMember, error) {
	result := r.db.WithContext(ctx).Model(&ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Update("role", role)
	if result.Error != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "update member role failed")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NewError(xerr.CodeNotFound, "member not found")
	}
	return r.FindByProjectAndUser(ctx, projectID, userID)
}

func (r *memberRepo) FindOwnerByProject(ctx context.Context, projectID string) (*ProjectMember, error) {
	var m ProjectMember
	err := r.db.WithContext(ctx).Where("project_id = ? AND role = ?", projectID, RoleOwner).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "find owner member failed")
	}
	return &m, nil
}
