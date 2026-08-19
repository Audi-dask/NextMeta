package service

import (
	"errors"
	"time"

	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

/*
UserService 定义用户模块的业务能力。
Handler 层通过该接口完成注册登录、本地用户管理、个人资料维护和审批人身份查询。
*/
type UserService interface {
	Register(req *dto.RegisterRequest) error
	Login(username, password string) (*jwt.TokenPair, string, uint, error)
	ListLocalUsers() ([]model.User, error)
	CreateLocalUser(req *dto.CreateUserRequest) error
	UpdateUser(req *dto.UpdateUserRequest) error
	UpdateProfile(userID uint, req *dto.UpdateProfileRequest) error
	ChangePassword(userID uint, oldPassword, newPassword string) error
	DeleteUser(id uint) error
	GetByID(id uint) (*model.User, error)
	IsApprover(userID uint) (bool, error)
}

/*
userService 是 UserService 的默认实现。
用户数据通过 UserRepository 持久化，系统设置仓储预留给后续用户相关配置读取。
*/
type userService struct {
	repo        repository.UserRepository
	settingRepo repository.SystemSettingRepository
	ldapSvc     LDAPService
}

/*
NewUserService 创建用户业务服务。
返回接口类型，便于 Handler 层依赖抽象而不是具体实现。
*/
func NewUserService(repo repository.UserRepository, settingRepo repository.SystemSettingRepository, ldapSvc LDAPService) UserService {
	return &userService{repo: repo, settingRepo: settingRepo, ldapSvc: ldapSvc}
}

/*
Register 注册一个默认开发者角色的本地用户。
创建前会检查用户名唯一性，并使用 bcrypt 存储密码哈希。
*/
func (s *userService) Register(req *dto.RegisterRequest) error {
	existingUser, err := s.repo.FindByUsername(req.Username)
	if err != nil {
		return err
	}
	if existingUser != nil && existingUser.Status == "enabled" {
		return errors.New("username already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		RealName: req.RealName,
		Email:    req.Email,
		Role:     string(model.UserRoleDeveloper),
		Status:   "enabled",
	}

	return s.repo.Create(user)
}

/*
Login 校验本地用户名和密码并生成 token pair。
用户名不存在、密码为空或密码校验失败时统一返回 invalid credentials，避免暴露账号状态。
返回值额外暴露 user.Role，便于 Handler 层在 license 失效时根据角色决定是否放行登录。
*/
func (s *userService) Login(username, password string) (*jwt.TokenPair, string, uint, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, "", 0, err
	}
	if user == nil {
		return nil, "", 0, errors.New("invalid credentials")
	}
	if user.Status != "" && user.Status != "enabled" {
		return nil, "", 0, errors.New("user disabled")
	}
	if user.Source == "ldap" {
		if err := s.ldapSvc.Authenticate(user.DN, password); err != nil {
			return nil, "", 0, errors.New("invalid credentials")
		}
		if err := s.repo.UpdateLastLoginAt(user.ID, time.Now()); err != nil {
			return nil, "", 0, err
		}
		tokens, err := jwt.GenerateTokenPair(user.ID, user.Username, user.Role)
		if err != nil {
			return nil, "", 0, err
		}
		return tokens, user.Role, user.ID, nil
	}
	if user.Password == "" {
		return nil, "", 0, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", 0, errors.New("invalid credentials")
	}
	if err := s.repo.UpdateLastLoginAt(user.ID, time.Now()); err != nil {
		return nil, "", 0, err
	}

	tokens, err := jwt.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, "", 0, err
	}
	return tokens, user.Role, user.ID, nil
}

/*
ListLocalUsers 返回本地用户列表。
具体查询条件由 repository 层决定，当前用于管理员用户管理页面。
*/
func (s *userService) ListLocalUsers() ([]model.User, error) {
	return s.repo.FindAll()
}

/*
CreateLocalUser 创建管理员在用户管理页面新增的本地用户。
该流程要求显式传入角色，并默认启用用户状态和 local 来源。
*/
func (s *userService) CreateLocalUser(req *dto.CreateUserRequest) error {
	existingUser, err := s.repo.FindByUsername(req.Username)
	if err != nil {
		return err
	}
	if existingUser != nil && existingUser.Status == "enabled" {
		return errors.New("username already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		RealName: req.RealName,
		Email:    req.Email,
		Role:     req.Role,
		Status:   "enabled",
		Source:   "local",
	}

	return s.repo.Create(user)
}

/*
UpdateUser 更新用户管理页面提交的用户信息。
基础资料、角色和状态会直接覆盖；Password 为空时保留原密码。
*/
func (s *userService) UpdateUser(req *dto.UpdateUserRequest) error {
	user, err := s.repo.FindByID(req.ID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	user.RealName = req.RealName
	user.Email = req.Email

	user.Role = req.Role
	user.Status = req.Status

	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user.Password = string(hashedPassword)
	}

	return s.repo.Update(user)
}

/*
UpdateProfile 更新当前登录用户的个人资料。
该方法只接收调用方传入的 userID，不允许通过请求体指定要修改的用户。
*/
func (s *userService) UpdateProfile(userID uint, req *dto.UpdateProfileRequest) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	user.RealName = req.RealName
	user.Email = req.Email
	return s.repo.Update(user)
}

/*
ChangePassword 修改当前登录用户的密码。
调用方必须通过 token 解析出的 userID 调用本方法，避免从请求体指定用户 ID 越权改他人密码；
LDAP 用户的密码不在本系统维护，明确拒绝修改避免误导。
*/
func (s *userService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if user.Source == "ldap" {
		return errors.New("LDAP 用户请到目录服务侧修改密码")
	}
	if user.Password == "" {
		return errors.New("当前账号未设置本地密码，无法通过该入口修改")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("原密码不正确")
	}
	if oldPassword == newPassword {
		return errors.New("新密码不能与原密码相同")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashed)
	return s.repo.Update(user)
}

/*
DeleteUser 删除指定本地用户。
删除前会确认用户存在，并阻止删除内置管理员 NextMeta。
*/
func (s *userService) DeleteUser(id uint) error {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// 内置管理员账号用于系统初始化和兜底登录，不允许被删除。
	if user.Username == "NextMeta" {
		return errors.New("内置管理员用户不可删除")
	}

	return s.repo.Delete(id)
}

/*
GetByID 按用户 ID 查询用户详情。
Handler 获取个人资料等场景会通过该方法读取用户模型。
*/
func (s *userService) GetByID(id uint) (*model.User, error) {
	return s.repo.FindByID(id)
}

/*
IsApprover 判断用户是否被配置为任意用户组的审批人。
该标记用于前端决定是否展示审批相关入口。
*/
func (s *userService) IsApprover(userID uint) (bool, error) {
	return s.repo.IsApprover(userID)
}
