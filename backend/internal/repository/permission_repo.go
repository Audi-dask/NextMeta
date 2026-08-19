package repository

import (
	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

/*
PermissionRepository 定义用户组权限关系的数据访问能力。
它维护三类关联：用户组成员、用户组数据源授权和用户组审批人。
*/
type PermissionRepository interface {
	AddUserToGroup(userID, groupID uint) error
	RemoveUserFromGroup(userID, groupID uint) error
	GetUserGroups(userID uint) ([]model.Group, error)
	GetGroupMembers(groupID uint) ([]model.User, error)
	IsUserInGroup(userID, groupID uint) (bool, error)

	AddDataSourceToGroup(groupID, datasourceID uint) error
	RemoveDataSourceFromGroup(groupID, datasourceID uint) error
	GetGroupDataSources(groupID uint) ([]model.DataSource, error)
	ReplaceGroupDataSources(groupID uint, datasourceIDs []uint) error
	GetDataSourceGroups(datasourceID uint) ([]model.Group, error)

	AddApprover(groupID, userID uint) error
	RemoveApprover(groupID, userID uint) error
	GetGroupApprovers(groupID uint) ([]model.User, error)
	IsGroupApprover(userID, groupID uint) (bool, error)
	HasApprovers(groupID uint) (bool, error)
}

/*
permissionRepository 是 PermissionRepository 的 GORM 实现。
所有权限关系读写都通过注入的 *gorm.DB 执行。
*/
type permissionRepository struct {
	db *gorm.DB
}

/*
NewPermissionRepository 创建权限关系仓储。
db 由 main.go 初始化并注入。
*/
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

/*
AddUserToGroup 新增用户和用户组的成员关系。
重复关系约束由数据库模型或调用方保证。
*/
func (r *permissionRepository) AddUserToGroup(userID, groupID uint) error {
	ug := &model.UserGroup{UserID: userID, GroupID: groupID}
	return r.db.Create(ug).Error
}

/*
RemoveUserFromGroup 移除用户和用户组的成员关系。
GORM 会按模型配置执行软删除。
*/
func (r *permissionRepository) RemoveUserFromGroup(userID, groupID uint) error {
	return r.db.Where("user_id = ? AND group_id = ?", userID, groupID).
		Delete(&model.UserGroup{}).Error
}

/*
GetUserGroups 返回用户所属且处于启用状态的用户组列表。
停用组不再授予数据源访问、工单提交或审批权限。
*/
func (r *permissionRepository) GetUserGroups(userID uint) ([]model.Group, error) {
	var groups []model.Group
	err := r.db.Table("groups").
		Joins("INNER JOIN user_groups ON user_groups.group_id = `groups`.id").
		Where("user_groups.user_id = ? AND user_groups.deleted_at IS NULL AND `groups`.deleted_at IS NULL AND `groups`.status = ?", userID, "enabled").
		Find(&groups).Error
	return groups, err
}

/*
GetGroupMembers 返回指定用户组的成员列表。
查询会通过 user_groups 关联 users 表，并过滤已删除关联。
*/
func (r *permissionRepository) GetGroupMembers(groupID uint) ([]model.User, error) {
	var users []model.User
	err := r.db.Table("users").
		Joins("INNER JOIN user_groups ON user_groups.user_id = users.id").
		Where("user_groups.group_id = ? AND user_groups.deleted_at IS NULL", groupID).
		Find(&users).Error
	return users, err
}

/*
IsUserInGroup 判断用户是否属于指定的启用用户组。
停用组中的成员关系仍保留用于管理展示，但不再参与授权判断。
*/
func (r *permissionRepository) IsUserInGroup(userID, groupID uint) (bool, error) {
	var count int64
	err := r.db.Table("user_groups").
		Joins("INNER JOIN `groups` ON `groups`.id = user_groups.group_id").
		Where("user_groups.user_id = ? AND user_groups.group_id = ? AND user_groups.deleted_at IS NULL AND `groups`.deleted_at IS NULL AND `groups`.status = ?", userID, groupID, "enabled").
		Count(&count).Error
	return count > 0, err
}

/*
AddDataSourceToGroup 新增用户组和数据源的授权关系。
授权后组内成员可通过权限服务访问该数据源。
*/
func (r *permissionRepository) AddDataSourceToGroup(groupID, datasourceID uint) error {
	gds := &model.GroupDataSource{GroupID: groupID, DataSourceID: datasourceID}
	return r.db.Create(gds).Error
}

/*
RemoveDataSourceFromGroup 移除用户组的数据源授权关系。
*/
func (r *permissionRepository) RemoveDataSourceFromGroup(groupID, datasourceID uint) error {
	return r.db.Where("group_id = ? AND data_source_id = ?", groupID, datasourceID).
		Delete(&model.GroupDataSource{}).Error
}

/*
GetGroupDataSources 返回指定用户组已授权的数据源列表。
查询会过滤已软删除的 group_datasources 关联记录。
*/
func (r *permissionRepository) GetGroupDataSources(groupID uint) ([]model.DataSource, error) {
	var datasources []model.DataSource
	err := r.db.Model(&model.DataSource{}).
		Joins("INNER JOIN group_datasources ON group_datasources.data_source_id = data_sources.id").
		Where("group_datasources.group_id = ? AND group_datasources.deleted_at IS NULL", groupID).
		Find(&datasources).Error
	return datasources, err
}

/*
ReplaceGroupDataSources 整体替换用户组的数据源授权列表。
它在事务中先删除原授权，再按去重后的 datasourceIDs 创建新授权。
*/
func (r *permissionRepository) ReplaceGroupDataSources(groupID uint, datasourceIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&model.GroupDataSource{}).Error; err != nil {
			return err
		}

		seen := make(map[uint]bool, len(datasourceIDs))
		for _, datasourceID := range datasourceIDs {
			if datasourceID == 0 || seen[datasourceID] {
				continue
			}
			seen[datasourceID] = true
			if err := tx.Create(&model.GroupDataSource{GroupID: groupID, DataSourceID: datasourceID}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

/*
GetDataSourceGroups 返回拥有指定数据源授权且处于启用状态的用户组列表。
停用组保留配置但不再形成有效权限范围。
*/
func (r *permissionRepository) GetDataSourceGroups(datasourceID uint) ([]model.Group, error) {
	var groups []model.Group
	err := r.db.Table("groups").
		Joins("INNER JOIN group_datasources ON group_datasources.group_id = `groups`.id").
		Where("group_datasources.data_source_id = ? AND group_datasources.deleted_at IS NULL AND `groups`.deleted_at IS NULL AND `groups`.status = ?", datasourceID, "enabled").
		Find(&groups).Error
	return groups, err
}

/*
AddApprover 新增用户组审批人关系。
审批人用于处理该组数据源相关工单。
*/
func (r *permissionRepository) AddApprover(groupID, userID uint) error {
	ga := &model.GroupApprover{GroupID: groupID, UserID: userID}
	return r.db.Create(ga).Error
}

/*
RemoveApprover 移除用户组审批人关系。
至少保留一个审批人的业务限制由 service 或 handler 层处理。
*/
func (r *permissionRepository) RemoveApprover(groupID, userID uint) error {
	return r.db.Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.GroupApprover{}).Error
}

/*
GetGroupApprovers 返回指定用户组的审批人列表。
查询会通过 group_approvers 关联 users 表，并过滤已删除关联。
*/
func (r *permissionRepository) GetGroupApprovers(groupID uint) ([]model.User, error) {
	var users []model.User
	err := r.db.Table("users").
		Joins("INNER JOIN group_approvers ON group_approvers.user_id = users.id").
		Where("group_approvers.group_id = ? AND group_approvers.deleted_at IS NULL", groupID).
		Find(&users).Error
	return users, err
}

/*
IsGroupApprover 判断用户是否是指定启用用户组的审批人。
停用用户组后，其审批人关系保留但不再具有审批权限。
*/
func (r *permissionRepository) IsGroupApprover(userID, groupID uint) (bool, error) {
	var count int64
	err := r.db.Table("group_approvers").
		Joins("INNER JOIN `groups` ON `groups`.id = group_approvers.group_id").
		Where("group_approvers.user_id = ? AND group_approvers.group_id = ? AND group_approvers.deleted_at IS NULL AND `groups`.deleted_at IS NULL AND `groups`.status = ?", userID, groupID, "enabled").
		Count(&count).Error
	return count > 0, err
}

/*
HasApprovers 判断指定启用用户组是否配置了审批人。
停用组即使保留审批人配置，也不能用于新工单提交。
*/
func (r *permissionRepository) HasApprovers(groupID uint) (bool, error) {
	var count int64
	err := r.db.Table("group_approvers").
		Joins("INNER JOIN `groups` ON `groups`.id = group_approvers.group_id").
		Where("group_approvers.group_id = ? AND group_approvers.deleted_at IS NULL AND `groups`.deleted_at IS NULL AND `groups`.status = ?", groupID, "enabled").
		Count(&count).Error
	return count > 0, err
}
