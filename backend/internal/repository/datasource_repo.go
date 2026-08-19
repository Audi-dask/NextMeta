package repository

import (
	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

/*
DataSourceRepository 定义数据源表的数据访问能力。
它同时负责数据源基础信息和脱敏规则关联的读取与保存。
*/
type DataSourceRepository interface {
	Create(ds *model.DataSource) error
	Update(ds *model.DataSource) error
	Delete(id uint) error
	FindByID(id uint) (*model.DataSource, error)
	FindAll() ([]model.DataSource, error)
	CountAll() (int64, error)
	CountByType() ([]model.DailyTrend, error)
}

/*
dataSourceRepository 是 DataSourceRepository 的 GORM 实现。
所有数据源数据访问都通过注入的 *gorm.DB 执行。
*/
type dataSourceRepository struct {
	db *gorm.DB
}

/*
NewDataSourceRepository 创建数据源仓储。
db 由 main.go 初始化并注入。
*/
func NewDataSourceRepository(db *gorm.DB) DataSourceRepository {
	return &dataSourceRepository{db: db}
}

/*
Create 创建数据源记录。
GORM 会按模型关联同时创建传入的脱敏规则。
*/
func (r *dataSourceRepository) Create(ds *model.DataSource) error {
	return r.db.Create(ds).Error
}

/*
Update 更新数据源基础信息并整体替换脱敏规则。
该方法使用事务保证基础信息和规则列表要么同时成功，要么同时回滚。
*/
func (r *dataSourceRepository) Update(ds *model.DataSource) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先取出规则并清空关联字段，避免 GORM 自动更新关联造成重复或残留。
		rules := ds.MaskingRules
		ds.MaskingRules = nil

		// 只保存数据源基础字段，脱敏规则后面单独整体替换。
		if err := tx.Omit("MaskingRules").Save(ds).Error; err != nil {
			return err
		}

		// 物理删除旧规则，保持脱敏规则表和当前配置一致。
		if err := tx.Unscoped().Where("data_source_id = ?", ds.ID).Delete(&model.DataSourceMaskingRule{}).Error; err != nil {
			return err
		}

		// 写入新的规则列表，并补齐 DataSourceID 关联字段。
		if len(rules) > 0 {
			for i := range rules {
				rules[i].DataSourceID = ds.ID
			}
			if err := tx.Create(&rules).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

/*
Delete 按主键删除数据源。
关联清理和连接池失效由 service 或模型关联策略处理。
*/
func (r *dataSourceRepository) Delete(id uint) error {
	return r.db.Delete(&model.DataSource{}, id).Error
}

/*
FindByID 按主键查询数据源。
查询时预加载 MaskingRules，便于 service 层执行脱敏配置和列表展示。
*/
func (r *dataSourceRepository) FindByID(id uint) (*model.DataSource, error) {
	var ds model.DataSource
	err := r.db.Preload("MaskingRules").First(&ds, id).Error
	return &ds, err
}

/*
FindAll 返回全部数据源。
查询时预加载 MaskingRules，避免列表转换时额外逐条查询。
*/
func (r *dataSourceRepository) FindAll() ([]model.DataSource, error) {
	var dss []model.DataSource
	err := r.db.Preload("MaskingRules").Find(&dss).Error
	return dss, err
}

/*
CountAll 统计数据源总数。
首页看板会通过该方法展示数据源概览数量。
*/
func (r *dataSourceRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&model.DataSource{}).Count(&count).Error
	return count, err
}

/*
CountByType 按数据源类型统计数量。
这里复用 DailyTrend 结构，Date 字段承载类型名称，Count 字段承载数量。
*/
func (r *dataSourceRepository) CountByType() ([]model.DailyTrend, error) {
	var results []model.DailyTrend
	err := r.db.Model(&model.DataSource{}).
		Select("type as date, count(*) as count").
		Group("type").
		Scan(&results).Error
	return results, err
}
