package data

import (
	"time"

	"gorm.io/gorm"
)

// Project is the persisted project aggregate. Version is used for optimistic
// locking on write APIs exposed through the gateway.
type Project struct {
	ID          string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	Name        string         `gorm:"column:name;size:100;not null"`
	Description string         `gorm:"column:description;not null;default:''"`
	OwnerID     string         `gorm:"column:owner_id;not null"`
	Status      int32          `gorm:"column:status;not null;default:0"`
	Version     int64          `gorm:"column:version;not null;default:0"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the model to the task service schema.
func (Project) TableName() string {
	return "task_svc.projects"
}

// ProjectMember links users to projects and stores their project-scoped role.
type ProjectMember struct {
	ID        string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	ProjectID string    `gorm:"column:project_id;not null"`
	UserID    string    `gorm:"column:user_id;not null"`
	Role      int32     `gorm:"column:role;not null"`
	JoinedAt  time.Time `gorm:"column:joined_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName pins the model to the task service schema.
func (ProjectMember) TableName() string {
	return "task_svc.project_members"
}

const (
	// ProjectStatusActive allows project and task writes.
	ProjectStatusActive int32 = 0
	// ProjectStatusArchived keeps the project readable but blocks mutations.
	ProjectStatusArchived int32 = 1

	// RoleOwner has full control and is unique per project.
	RoleOwner int32 = 0
	// RoleAdmin can manage regular members and tasks.
	RoleAdmin int32 = 1
	// RoleMember can work with tasks under member-level restrictions.
	RoleMember int32 = 2
)

// IsValidProjectName enforces the storage/API length limit for project names.
func IsValidProjectName(name string) bool {
	l := len(name)
	return l >= 1 && l <= 100
}

// Task is the persisted task record. DueTime is stored as a string to match the
// current API contract without adding timezone conversion in the data layer.
type Task struct {
	ID         string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	ProjectID  string         `gorm:"column:project_id;not null"`
	Title      string         `gorm:"column:title;size:200;not null"`
	Content    string         `gorm:"column:content;not null;default:''"`
	Status     int32          `gorm:"column:status;not null;default:0"`
	Priority   int32          `gorm:"column:priority;not null;default:1"`
	AssigneeID *string        `gorm:"column:assignee_id"`
	CreatorID  string         `gorm:"column:creator_id;not null"`
	DueTime    *string        `gorm:"column:due_time"`
	Extra      string         `gorm:"column:extra;type:jsonb;not null;default:'{}'"`
	Version    int64          `gorm:"column:version;not null;default:0"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the model to the task service schema.
func (Task) TableName() string {
	return "task_svc.tasks"
}

const (
	// TaskStatusTodo is the initial state for new tasks.
	TaskStatusTodo      int32 = 0
	TaskStatusDoing     int32 = 1
	TaskStatusDone      int32 = 2
	TaskStatusCancelled int32 = 3

	// PriorityLow through PriorityUrgent mirror the frontend select values.
	PriorityLow    int32 = 0
	PriorityNormal int32 = 1
	PriorityHigh   int32 = 2
	PriorityUrgent int32 = 3
)

// IsValidTaskTitle enforces the storage/API length limit for task titles.
func IsValidTaskTitle(title string) bool {
	l := len(title)
	return l >= 1 && l <= 200
}

// TaskComment stores append-only task discussion entries.
type TaskComment struct {
	ID        string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	TaskID    string    `gorm:"column:task_id;not null"`
	UserID    string    `gorm:"column:user_id;not null"`
	Content   string    `gorm:"column:content;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

// TableName pins the model to the task service schema.
func (TaskComment) TableName() string {
	return "task_svc.task_comments"
}

// OperationLog records auditable project/task mutations. ProjectID or TaskID may
// be nil depending on the action scope.
type OperationLog struct {
	ID         string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	ProjectID  *string   `gorm:"column:project_id"`
	TaskID     *string   `gorm:"column:task_id"`
	OperatorID string    `gorm:"column:operator_id;not null"`
	Action     string    `gorm:"column:action;size:50;not null"`
	Detail     string    `gorm:"column:detail;type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

// TableName pins the model to the task service schema.
func (OperationLog) TableName() string {
	return "task_svc.operation_logs"
}

const (
	// Action* constants are stable audit-log action names consumed by the gateway
	// and frontend operation-log views.
	ActionTaskCreate           = "task.create"
	ActionTaskUpdate           = "task.update"
	ActionTaskAssign           = "task.assign"
	ActionTaskStatusChange     = "task.status_change"
	ActionTaskDelete           = "task.delete"
	ActionCommentCreate        = "comment.create"
	ActionCommentDelete        = "comment.delete"
	ActionMemberAdd            = "member.add"
	ActionMemberRemove         = "member.remove"
	ActionMemberRoleChange     = "member.role_change"
	ActionMemberLeave          = "member.leave"
	ActionProjectCreate        = "project.create"
	ActionProjectUpdate        = "project.update"
	ActionProjectArchive       = "project.archive"
	ActionProjectUnarchive     = "project.unarchive"
	ActionProjectTransferOwner = "project.transfer_ownership"
)
