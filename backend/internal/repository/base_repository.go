package repository

import "gorm.io/gorm"

/*
BaseRepository 保存通用的 GORM 数据库连接。
少量不需要独立 repository 的场景会直接通过它访问 DB，例如审核规则配置。
*/
type BaseRepository struct {
	DB *gorm.DB
}
