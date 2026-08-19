package service

import (
	"nextmeta-backend/internal/repository"
)

/*
PermissionService 定义权限判断相关业务能力。
当前权限模型以用户组为核心，用户通过所属用户组获得数据源访问权和审批权限。
*/
type PermissionService interface {
	CanAccessDataSource(userID, datasourceID uint) (bool, error)
	IsGroupApprover(userID, groupID uint) (bool, error)
	HasGroupApprovers(groupID uint) (bool, error)
	GetUserAccessibleDataSources(userID uint) ([]uint, error)
	GetUserGroups(userID uint) ([]uint, error)
}

/*
permissionService 是 PermissionService 的默认实现。
permRepo 负责用户组关系查询，dsRepo 预留给需要读取数据源详情的权限场景。
*/
type permissionService struct {
	permRepo repository.PermissionRepository
	dsRepo   repository.DataSourceRepository
}

/*
NewPermissionService 创建权限业务服务。
Handler 和其他 service 通过该接口判断数据源访问权和审批权限。
*/
func NewPermissionService(permRepo repository.PermissionRepository, dsRepo repository.DataSourceRepository) PermissionService {
	return &permissionService{
		permRepo: permRepo,
		dsRepo:   dsRepo,
	}
}

/*
CanAccessDataSource 判断用户是否可以访问指定数据源。
它先读取用户所属用户组，再检查这些组是否授权了目标数据源。
*/
func (s *permissionService) CanAccessDataSource(userID, datasourceID uint) (bool, error) {
	// 用户的数据源权限来自其所属的所有用户组。
	groups, err := s.permRepo.GetUserGroups(userID)
	if err != nil {
		return false, err
	}

	// 任意一个用户组拥有该数据源授权，即认为用户可访问。
	for _, group := range groups {
		datasources, err := s.permRepo.GetGroupDataSources(group.ID)
		if err != nil {
			continue
		}
		for _, ds := range datasources {
			if ds.ID == datasourceID {
				return true, nil
			}
		}
	}

	return false, nil
}

/*
IsGroupApprover 判断用户是否是指定用户组的审批人。
工单审批流会通过该能力校验审批人身份。
*/
func (s *permissionService) IsGroupApprover(userID, groupID uint) (bool, error) {
	return s.permRepo.IsGroupApprover(userID, groupID)
}

/*
HasGroupApprovers 判断指定用户组是否存在审批人。
创建或维护工单审批配置时可用该能力做完整性校验。
*/
func (s *permissionService) HasGroupApprovers(groupID uint) (bool, error) {
	return s.permRepo.HasApprovers(groupID)
}

/*
GetUserAccessibleDataSources 返回用户可访问的数据源 ID 列表。
它会聚合用户所属所有用户组的数据源授权，并按数据源 ID 去重。
*/
func (s *permissionService) GetUserAccessibleDataSources(userID uint) ([]uint, error) {
	groups, err := s.permRepo.GetUserGroups(userID)
	if err != nil {
		return nil, err
	}

	dsMap := make(map[uint]bool)
	for _, group := range groups {
		datasources, err := s.permRepo.GetGroupDataSources(group.ID)
		if err != nil {
			continue
		}
		for _, ds := range datasources {
			dsMap[ds.ID] = true
		}
	}

	var dsIDs []uint
	for id := range dsMap {
		dsIDs = append(dsIDs, id)
	}

	return dsIDs, nil
}

/*
GetUserGroups 返回用户所属用户组 ID 列表。
repository 返回完整用户组模型，这里只提取业务层需要的 ID。
*/
func (s *permissionService) GetUserGroups(userID uint) ([]uint, error) {
	groups, err := s.permRepo.GetUserGroups(userID)
	if err != nil {
		return nil, err
	}

	var groupIDs []uint
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}

	return groupIDs, nil
}
