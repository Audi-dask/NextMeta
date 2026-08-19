package model

import (
	"time"

	"gorm.io/gorm"
)

/*
SQLSnippet 是用户保存的 SQL 片段模型。
每条片段归属于一个用户，用于 SQL 编辑器中快速复用常用语句。
*/
type SQLSnippet struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	UserID    uint           `gorm:"not null;index;comment:用户ID"`
	Title     string         `gorm:"type:varchar(100);not null;comment:片段标题"`
	Content   string         `gorm:"type:text;not null;comment:SQL内容"`
}

/*
TableName 指定 SQL 片段模型对应 sql_snippets 表。
*/
func (SQLSnippet) TableName() string {
	return "sql_snippets"
}
