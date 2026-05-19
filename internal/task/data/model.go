package data

import (
	"time"

	"gorm.io/gorm"
)

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

func (Project) TableName() string {
	return "task_svc.projects"
}

type ProjectMember struct {
	ID        string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	ProjectID string    `gorm:"column:project_id;not null"`
	UserID    string    `gorm:"column:user_id;not null"`
	Role      int32     `gorm:"column:role;not null"`
	JoinedAt  time.Time `gorm:"column:joined_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (ProjectMember) TableName() string {
	return "task_svc.project_members"
}

const (
	ProjectStatusActive   int32 = 0
	ProjectStatusArchived int32 = 1

	RoleOwner  int32 = 0
	RoleAdmin  int32 = 1
	RoleMember int32 = 2
)

func IsValidProjectName(name string) bool {
	l := len(name)
	return l >= 1 && l <= 100
}

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

func (Task) TableName() string {
	return "task_svc.tasks"
}

const (
	TaskStatusTodo      int32 = 0
	TaskStatusDoing     int32 = 1
	TaskStatusDone      int32 = 2
	TaskStatusCancelled int32 = 3

	PriorityLow    int32 = 0
	PriorityNormal int32 = 1
	PriorityHigh   int32 = 2
	PriorityUrgent int32 = 3
)

func IsValidTaskTitle(title string) bool {
	l := len(title)
	return l >= 1 && l <= 200
}

type TaskComment struct {
	ID        string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	TaskID    string    `gorm:"column:task_id;not null"`
	UserID    string    `gorm:"column:user_id;not null"`
	Content   string    `gorm:"column:content;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (TaskComment) TableName() string {
	return "task_svc.task_comments"
}

type OperationLog struct {
	ID         string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	ProjectID  *string   `gorm:"column:project_id"`
	TaskID     *string   `gorm:"column:task_id"`
	OperatorID string    `gorm:"column:operator_id;not null"`
	Action     string    `gorm:"column:action;size:50;not null"`
	Detail     string    `gorm:"column:detail;type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (OperationLog) TableName() string {
	return "task_svc.operation_logs"
}

const (
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
