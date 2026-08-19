package repository

import (
	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

/*
SystemSettingRepository 定义系统设置表的数据访问能力。
配置项按 key-value 存储，当前用于通知和查询限制等运行配置。
*/
type SystemSettingRepository interface {
	Get(key string) (string, error)
	Set(key, value string) error
	GetAll() ([]model.SystemSetting, error)
}

/*
systemSettingRepository 是 SystemSettingRepository 的 GORM 实现。
所有系统配置读写都通过注入的 *gorm.DB 执行。
*/
type systemSettingRepository struct {
	db *gorm.DB
}

/*
NewSystemSettingRepository 创建系统设置仓储。
db 由 main.go 初始化并注入。
*/
func NewSystemSettingRepository(db *gorm.DB) SystemSettingRepository {
	return &systemSettingRepository{db: db}
}

/*
Get 按 key 读取系统配置值。
未找到时返回空字符串和 nil，便于调用方使用默认配置兜底。
*/
func (r *systemSettingRepository) Get(key string) (string, error) {
	var setting model.SystemSetting
	// 使用 Find 避免 First/Take 在配置不存在时触发 GORM record not found 日志。
	result := r.db.Where("`key` = ?", key).Limit(1).Find(&setting)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		return "", nil
	}
	return setting.Value, nil
}

/*
Set 创建或更新系统配置项。
配置不存在时创建新记录，存在时只更新 Value 字段。
*/
func (r *systemSettingRepository) Set(key, value string) error {
	var setting model.SystemSetting
	// 先按 key 查询配置是否存在，再决定创建或更新。
	err := r.db.Where("`key` = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			setting = model.SystemSetting{
				Key:   key,
				Value: value,
			}
			return r.db.Create(&setting).Error
		}
		return err
	}

	setting.Value = value
	return r.db.Save(&setting).Error
}

/*
GetAll 返回全部系统配置项。
系统设置页面会把该列表转换为 key-value map 后展示。
*/
func (r *systemSettingRepository) GetAll() ([]model.SystemSetting, error) {
	var settings []model.SystemSetting
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}
