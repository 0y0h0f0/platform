package data

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"task-platform/pkg/xerr"
	"task-platform/pkg/xcursor"
	"time"
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

type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	FindByID(ctx context.Context, id string) (*Task, error)
	Update(ctx context.Context, id string, version int64, updates map[string]any) (*Task, error)
	Delete(ctx context.Context, id string) (*Task, error)
	List(ctx context.Context, filter TaskFilter) (tasks []*Task, nextCursor string, err error)
}

type TaskFilter struct {
	ProjectID  string
	Status     *int32
	AssigneeID string
	Keyword    string
	Limit      int
	Cursor     string
	FilterHash string
}

type CommentRepository interface {
	Create(ctx context.Context, comment *TaskComment) error
	FindByID(ctx context.Context, id string) (*TaskComment, error)
	Delete(ctx context.Context, id string) (*TaskComment, error)
	ListByTask(ctx context.Context, taskID string, limit int32, afterID string) ([]*TaskComment, error)
}

type OperationLogRepository interface {
	CreateBatch(ctx context.Context, logs []*OperationLog) error
	ListByProject(ctx context.Context, projectID string, limit int, cursor string) ([]*OperationLog, string, error)
	ListByTask(ctx context.Context, taskID string, limit int, cursor string) ([]*OperationLog, string, error)
}

type projectRepo struct{ db *gorm.DB }
type memberRepo struct{ db *gorm.DB }
type taskRepo struct{ db *gorm.DB }
type commentRepo struct{ db *gorm.DB }
type operationLogRepo struct{ db *gorm.DB }

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepo{db: db}
}

func NewMemberRepository(db *gorm.DB) MemberRepository {
	return &memberRepo{db: db}
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepo{db: db}
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepo{db: db}
}

func NewOperationLogRepository(db *gorm.DB) OperationLogRepository {
	return &operationLogRepo{db: db}
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

// --- TaskRepository ---

func (r *taskRepo) Create(ctx context.Context, task *Task) error {
	err := r.db.WithContext(ctx).Create(task).Error
	if err != nil {
		return xerr.NewError(xerr.CodeInternal, "create task failed")
	}
	return nil
}

func (r *taskRepo) FindByID(ctx context.Context, id string) (*Task, error) {
	var t Task
	err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NewError(xerr.CodeNotFound, "task not found")
	}
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "find task failed")
	}
	return &t, nil
}

func (r *taskRepo) Update(ctx context.Context, id string, version int64, updates map[string]any) (*Task, error) {
	result := r.db.WithContext(ctx).Model(&Task{}).
		Where("id = ? AND version = ?", id, version).
		Updates(updates)
	if result.Error != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "update task failed")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NewError(xerr.CodeAborted, "task was modified by another request, please retry")
	}
	return r.FindByID(ctx, id)
}

func (r *taskRepo) Delete(ctx context.Context, id string) (*Task, error) {
	task, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&Task{})
	if result.Error != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "delete task failed")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NewError(xerr.CodeInternal, "delete task failed")
	}
	return task, nil
}

func (r *taskRepo) List(ctx context.Context, filter TaskFilter) ([]*Task, string, error) {
	query := r.db.WithContext(ctx).Model(&Task{}).
		Where("project_id = ?", filter.ProjectID)

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.AssigneeID != "" {
		query = query.Where("assignee_id = ?", filter.AssigneeID)
	}
	if filter.Keyword != "" {
		query = query.Where("title ILIKE ?", "%"+filter.Keyword+"%")
	}

	if filter.Cursor != "" {
		fields, err := xcursor.DecodeCursor(filter.Cursor)
		if err != nil {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor")
		}
		id, ok := fields["id"].(string)
		if !ok {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor fields")
		}
		createdAtStr, ok := fields["created_at"].(string)
		if !ok {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor fields")
		}
		createdAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
		if err != nil {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor timestamp")
		}
		query = query.Where("(created_at, id) < (?, ?)", createdAt, id)
	}

	if filter.Limit <= 0 || filter.Limit > 50 {
		filter.Limit = 20
	}
	query = query.Order("created_at DESC, id DESC").Limit(filter.Limit + 1)

	var tasks []*Task
	if err := query.Find(&tasks).Error; err != nil {
		return nil, "", xerr.NewError(xerr.CodeInternal, "list tasks failed")
	}

	var nextCursor string
	if len(tasks) > filter.Limit {
		last := tasks[filter.Limit-1]
		c, err := xcursor.EncodeCursor(map[string]any{
			"created_at": last.CreatedAt.Format(time.RFC3339Nano),
			"id":         last.ID,
		})
		if err != nil {
			nextCursor = ""
		} else {
			nextCursor = c
		}
		tasks = tasks[:filter.Limit]
	}

	return tasks, nextCursor, nil
}

// --- CommentRepository ---

func (r *commentRepo) Create(ctx context.Context, comment *TaskComment) error {
	err := r.db.WithContext(ctx).Create(comment).Error
	if err != nil {
		return xerr.NewError(xerr.CodeInternal, "create comment failed")
	}
	return nil
}

func (r *commentRepo) FindByID(ctx context.Context, id string) (*TaskComment, error) {
	var c TaskComment
	err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NewError(xerr.CodeNotFound, "comment not found")
	}
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "find comment failed")
	}
	return &c, nil
}

func (r *commentRepo) Delete(ctx context.Context, id string) (*TaskComment, error) {
	comment, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&TaskComment{})
	if result.Error != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "delete comment failed")
	}
	return comment, nil
}

func (r *commentRepo) ListByTask(ctx context.Context, taskID string, limit int32, afterID string) ([]*TaskComment, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := r.db.WithContext(ctx).Where("task_id = ?", taskID)
	if afterID != "" {
		var anchor TaskComment
		if err := r.db.WithContext(ctx).Where("id = ?", afterID).First(&anchor).Error; err != nil {
			return nil, xerr.NewError(xerr.CodeNotFound, "after_id comment not found")
		}
		query = query.Where("(created_at, id) > (?, ?)", anchor.CreatedAt, anchor.ID)
	}
	var comments []*TaskComment
	err := query.Order("created_at ASC, id ASC").Limit(int(limit)).Find(&comments).Error
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "list comments failed")
	}
	return comments, nil
}

// --- OperationLogRepository ---

func (r *operationLogRepo) CreateBatch(ctx context.Context, logs []*OperationLog) error {
	if len(logs) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).Create(logs).Error
	if err != nil {
		return xerr.NewError(xerr.CodeInternal, "create operation logs failed")
	}
	return nil
}

func (r *operationLogRepo) ListByProject(ctx context.Context, projectID string, limit int, cursor string) ([]*OperationLog, string, error) {
	return r.listLogs(ctx, "project_id = ?", projectID, limit, cursor)
}

func (r *operationLogRepo) ListByTask(ctx context.Context, taskID string, limit int, cursor string) ([]*OperationLog, string, error) {
	return r.listLogs(ctx, "task_id = ?", taskID, limit, cursor)
}

func (r *operationLogRepo) listLogs(ctx context.Context, where string, arg string, limit int, cursor string) ([]*OperationLog, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := r.db.WithContext(ctx).Where(where, arg)

	if cursor != "" {
		fields, err := xcursor.DecodeCursor(cursor)
		if err != nil {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor")
		}
		id, ok := fields["id"].(string)
		if !ok {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor fields")
		}
		createdAtStr, ok := fields["created_at"].(string)
		if !ok {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor fields")
		}
		createdAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
		if err != nil {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor timestamp")
		}
		query = query.Where("(created_at, id) < (?, ?)", createdAt, id)
	}

	query = query.Order("created_at DESC, id DESC").Limit(limit + 1)

	var logs []*OperationLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, "", xerr.NewError(xerr.CodeInternal, "list operation logs failed")
	}

	var nextCursor string
	if len(logs) > limit {
		last := logs[limit-1]
		c, err := xcursor.EncodeCursor(map[string]any{
			"created_at": last.CreatedAt.Format(time.RFC3339Nano),
			"id":         last.ID,
		})
		if err != nil {
			nextCursor = ""
		} else {
			nextCursor = c
		}
		logs = logs[:limit]
	}

	return logs, nextCursor, nil
}
