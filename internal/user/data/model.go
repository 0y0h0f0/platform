package data

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	Username     string         `gorm:"column:username;size:32;not null"`
	Email        string         `gorm:"column:email;size:320;not null"`
	PasswordHash string         `gorm:"column:password_hash;size:255;not null"`
	Nickname     string         `gorm:"column:nickname;size:64;not null;default:''"`
	AvatarURL    string         `gorm:"column:avatar_url;not null;default:''"`
	Status       int32          `gorm:"column:status;not null;default:0"`
	CreatedAt    time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (User) TableName() string {
	return "user_svc.users"
}
