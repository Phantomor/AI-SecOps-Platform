// internal/model/user.go
package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户实体模型
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserAccount  string         `gorm:"type:varchar(50);uniqueIndex;not null;comment:登录账号" json:"userAccount"`
	UserPassword string         `gorm:"type:varchar(255);not null;comment:加密后的密码" json:"-"` // json:"-" 防止密码意外泄露给前端
	UserName     string         `gorm:"type:varchar(50);comment:用户昵称" json:"userName"`
	UserRole     string         `gorm:"type:varchar(20);default:'user';comment:角色 admin/user" json:"userRole"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "user"
}
