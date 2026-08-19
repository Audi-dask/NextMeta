package repository

import (
	"errors"
	"time"

	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

/*
UserRepository 定义用户表和用户审批人关系的访问能力。
service 层通过该接口完成本地用户查询、创建、更新、删除和统计。
*/
type UserRepository interface {
	Create(user *model.User) error
	FindByUsername(username string) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	FindFirstSuperAdmin() (*model.User, error)
	FindByID(id uint) (*model.User, error)
	FindAll() ([]model.User, error)
	Update(user *model.User) error
	Delete(id uint) error
	CountAll() (int64, error)
	IsApprover(userID uint) (bool, error)
	UpdateLastLoginAt(userID uint, loginAt time.Time) error
}

/*
userRepository 是 UserRepository 的 GORM 实现。
所有用户相关 SQL 都通过注入的 *gorm.DB 执行。
*/
type userRepository struct {
	db *gorm.DB
}

/*
NewUserRepository 创建用户仓储。
db 由 main.go 初始化并注入。
*/
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

/*
Create 创建用户记录。
调用方需要提前完成密码哈希、角色和状态等字段赋值。
*/
func (r *userRepository) Create(user *model.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing model.User
		query := tx.Unscoped().Where("username = ?", user.Username).Limit(1).Find(&existing)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected == 0 {
			return tx.Create(user).Error
		}
		if existing.DeletedAt.Valid || existing.Status != "enabled" {
			existing.Username = user.Username
			existing.Password = user.Password
			existing.RealName = user.RealName
			existing.Email = user.Email
			existing.Role = user.Role
			existing.Status = user.Status
			existing.Source = user.Source
			existing.DN = user.DN
			existing.DeletedAt = gorm.DeletedAt{}
			if err := tx.Unscoped().Save(&existing).Error; err != nil {
				return err
			}
			user.ID = existing.ID
			return nil
		}
		return tx.Create(user).Error
	})
}

/*
FindByUsername 按用户名查询用户。
未找到时返回 nil, nil，方便 service 层区分不存在和数据库错误。
*/
func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

/*
FindByEmail 按邮箱查询用户。
未找到时返回 nil, nil，方便 service 层区分不存在和数据库错误。
*/
func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

/*
FindFirstSuperAdmin 查询系统中的第一个超级管理员。
用户组未指定审批人时会用该用户作为默认审批人兜底。
*/
func (r *userRepository) FindFirstSuperAdmin() (*model.User, error) {
	var user model.User
	err := r.db.Where("role = ?", string(model.UserRoleSuperAdmin)).Order("id ASC").First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

/*
FindByID 按主键查询用户。
未找到时返回 nil, nil，避免上层把不存在误判为数据库异常。
*/
func (r *userRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

/*
FindAll 返回全部用户。
当前用于管理员用户管理页面和相关配置页面。
*/
func (r *userRepository) FindAll() ([]model.User, error) {
	var users []model.User
	err := r.db.Find(&users).Error
	return users, err
}

/*
Update 保存用户模型变更。
该方法使用 GORM Save，调用方需传入完整用户模型。
*/
func (r *userRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

/*
Delete 按主键删除用户。
删除保护和业务校验由 service 层负责。
*/
func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}

/*
CountAll 统计用户总数。
首页看板会通过该方法展示用户概览数量。
*/
func (r *userRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

/*
IsApprover 判断用户是否配置为任意用户组审批人。
只统计 group_approvers 关联表，不关心具体用户组。
*/
func (r *userRepository) IsApprover(userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.GroupApprover{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

/*
UpdateLastLoginAt 更新用户最后登录时间。
登录成功后调用该方法，失败不应生成 token。
*/
func (r *userRepository) UpdateLastLoginAt(userID uint, loginAt time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("last_login_at", loginAt).Error
}
