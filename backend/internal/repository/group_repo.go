package repository

import (
	"errors"
	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

/*
GroupRepository 定义用户组表的数据访问能力。
service 层通过它维护用户组基础信息，关联关系由权限仓储配合处理。
*/
type GroupRepository interface {
	Create(group *model.Group) error
	FindAll() ([]model.Group, error)
	FindByID(id uint) (*model.Group, error)
	FindByDN(dn string) (*model.Group, error)
	Update(group *model.Group) error
	Delete(id uint) error
}

/*
groupRepository 是 GroupRepository 的 GORM 实现。
所有用户组数据访问都通过注入的 *gorm.DB 执行。
*/
type groupRepository struct {
	db *gorm.DB
}

/*
NewGroupRepository 创建用户组仓储。
db 由 main.go 初始化并注入。
*/
func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

/*
Create 创建用户组。
创建前会物理清理同名且已软删除的历史记录，避免唯一约束或重名数据影响重新创建。
*/
func (r *groupRepository) Create(group *model.Group) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("name = ? AND deleted_at IS NOT NULL", group.Name).Delete(&model.Group{}).Error; err != nil {
			return err
		}
		return tx.Create(group).Error
	})
}

/*
FindAll 返回全部未删除用户组。
GORM 默认会自动过滤软删除记录。
*/
func (r *groupRepository) FindAll() ([]model.Group, error) {
	var groups []model.Group
	err := r.db.Find(&groups).Error
	return groups, err
}

/*
FindByID 按主键查询用户组。
未找到时返回 nil, nil，便于 service 层输出业务错误。
*/
func (r *groupRepository) FindByID(id uint) (*model.Group, error) {
	var group model.Group
	err := r.db.First(&group, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

/*
FindByDN 按外部目录 DN 查询用户组。
当前本地用户组仍保留该查询能力，便于兼容历史或目录同步场景。
*/
func (r *groupRepository) FindByDN(dn string) (*model.Group, error) {
	var group model.Group
	err := r.db.Where("dn = ?", dn).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

/*
Update 保存用户组模型变更。
调用方需要传入已读取并修改后的完整用户组模型。
*/
func (r *groupRepository) Update(group *model.Group) error {
	return r.db.Save(group).Error
}

/*
Delete 删除用户组及其关联关系。
删除时会在事务中清理成员、数据源授权和审批人关联，再物理删除用户组记录。
*/
func (r *groupRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("group_id = ?", id).Delete(&model.UserGroup{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("group_id = ?", id).Delete(&model.GroupDataSource{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("group_id = ?", id).Delete(&model.GroupApprover{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&model.Group{}, id).Error
	})
}
