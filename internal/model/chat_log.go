// internal/model/chat_log.go
package model

import "gorm.io/gorm"

// ChatLog 对话持久化模型
type ChatLog struct {
	gorm.Model
	SessionID string `gorm:"type:varchar(100);index;comment:会话唯一标识" json:"sessionId"`
	UserID    uint   `gorm:"index;comment:所属用户ID" json:"userId"`
	Role      string `gorm:"type:varchar(20);comment:角色: system/user/assistant" json:"role"`
	Content   string `gorm:"type:longtext;comment:消息原文" json:"content"`
	Tokens    int    `gorm:"comment:消耗Token数(可选)" json:"tokens"`
}

func (ChatLog) TableName() string {
	return "chat_log"
}
