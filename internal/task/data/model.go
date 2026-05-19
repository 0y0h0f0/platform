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
