package service

import (
	"errors"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"strconv"
	"strings"
)

/*
GroupResponse 是用户组列表返回给前端的聚合结构。
除用户组基础信息外，还包含已授权数据源、审批人和成员 ID 列表。
*/
type GroupResponse struct {
	ID            uint     `json:"id"`
	Name          string   `json:"name"`
	Code          string   `json:"code"`
	Status        string   `json:"status"`
	DataSourceIDs []string `json:"datasourceIds"`
	ReviewerIDs   []string `json:"reviewerIds"`
	MemberIDs     []string `json:"memberIds"`
	Description   string   `json:"description"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

/*
GroupService 定义用户组管理业务能力。
它负责用户组基础信息维护，以及审批人和成员关联同步。
*/
type GroupService interface {
	ListAll() ([]GroupResponse, error)
	Create(name, code, status, description string, reviewerIDs, memberIDs []uint) error
	Update(id uint, name, code, status, description string, reviewerIDs, memberIDs []uint) error
	Delete(id uint) error
}

/*
groupService 是 GroupService 的默认实现。
它通过 group repository 维护用户组，通过 permission repository 维护组关联关系。
*/
type groupService struct {
	repo     repository.GroupRepository
	userRepo repository.UserRepository
	permRepo repository.PermissionRepository
}

/*
NewGroupService 创建用户组业务服务。
userRepo 用于查找默认超级管理员审批人，permRepo 用于同步成员和审批人关联。
*/
func NewGroupService(repo repository.GroupRepository, userRepo repository.UserRepository, permRepo repository.PermissionRepository) GroupService {
	return &groupService{
		repo:     repo,
		userRepo: userRepo,
		permRepo: permRepo,
	}
}

/*
ListAll 返回全部用户组及其关联信息。
每个用户组都会额外查询授权数据源、审批人和成员，并转换成前端需要的字符串 ID 列表。
*/
func (s *groupService) ListAll() ([]GroupResponse, error) {
	groups, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}

	responses := make([]GroupResponse, 0, len(groups))
	for _, group := range groups {
		datasources, err := s.permRepo.GetGroupDataSources(group.ID)
		if err != nil {
			return nil, err
		}
		approvers, err := s.permRepo.GetGroupApprovers(group.ID)
		if err != nil {
			return nil, err
		}
		members, err := s.permRepo.GetGroupMembers(group.ID)
		if err != nil {
			return nil, err
		}

		responses = append(responses, GroupResponse{
			ID:            group.ID,
			Name:          group.Name,
			Code:          normalizeGroupCode(group.Code, group.Name),
			Status:        normalizeGroupStatus(group.Status),
			DataSourceIDs: dataSourceIDsToStrings(datasources),
			ReviewerIDs:   userIDsToStrings(approvers),
			MemberIDs:     userIDsToStrings(members),
			Description:   normalizeGroupDescription(group.Description),
			CreatedAt:     group.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:     group.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return responses, nil
}

/*
Create 创建用户组并同步审批人和成员关系。
如果未传审批人，会自动使用第一个超级管理员作为默认审批人。
*/
func (s *groupService) Create(name, code, status, description string, reviewerIDs, memberIDs []uint) error {
	group := &model.Group{
		Name:        name,
		Code:        normalizeGroupCode(code, name),
		Status:      normalizeGroupStatus(status),
		Description: description,
		Source:      "local",
	}
	if err := s.repo.Create(group); err != nil {
		return err
	}
	normalizedReviewerIDs, err := s.ensureReviewerIDs(reviewerIDs)
	if err != nil {
		return err
	}
	return s.syncGroupUsers(group.ID, normalizedReviewerIDs, memberIDs)
}

/*
Update 更新用户组基础信息并同步审批人和成员关系。
用户组不存在时返回 group not found，避免静默创建或误更新。
*/
func (s *groupService) Update(id uint, name, code, status, description string, reviewerIDs, memberIDs []uint) error {
	group, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	group.Name = name
	group.Code = normalizeGroupCode(code, name)
	group.Status = normalizeGroupStatus(status)
	group.Description = description
	if err := s.repo.Update(group); err != nil {
		return err
	}
	normalizedReviewerIDs, err := s.ensureReviewerIDs(reviewerIDs)
	if err != nil {
		return err
	}
	return s.syncGroupUsers(id, normalizedReviewerIDs, memberIDs)
}

/*
ensureReviewerIDs 保证用户组至少拥有一个审批人。
当请求未指定审批人时，会使用系统中的第一个超级管理员兜底。
*/
func (s *groupService) ensureReviewerIDs(reviewerIDs []uint) ([]uint, error) {
	if len(reviewerIDs) > 0 {
		return reviewerIDs, nil
	}

	superAdmin, err := s.userRepo.FindFirstSuperAdmin()
	if err != nil {
		return nil, err
	}
	if superAdmin == nil {
		return nil, errors.New("用户组至少需要一个审核人，且当前系统未找到超级管理员")
	}
	return []uint{superAdmin.ID}, nil
}

/*
syncGroupUsers 同步用户组审批人和成员关联。
当前采用先删除原有关联、再按目标 ID 集重新创建的方式，确保前端提交列表就是最终状态。
*/
func (s *groupService) syncGroupUsers(groupID uint, reviewerIDs, memberIDs []uint) error {
	if reviewerIDs != nil {
		existingApprovers, err := s.permRepo.GetGroupApprovers(groupID)
		if err != nil {
			return err
		}
		wantedApprovers := uintSet(reviewerIDs)
		for _, approver := range existingApprovers {
			if err := s.permRepo.RemoveApprover(groupID, approver.ID); err != nil {
				return err
			}
		}
		for reviewerID := range wantedApprovers {
			if err := s.permRepo.AddApprover(groupID, reviewerID); err != nil {
				return err
			}
		}
	}

	if memberIDs != nil {
		existingMembers, err := s.permRepo.GetGroupMembers(groupID)
		if err != nil {
			return err
		}
		wantedMembers := uintSet(memberIDs)
		for _, member := range existingMembers {
			if err := s.permRepo.RemoveUserFromGroup(member.ID, groupID); err != nil {
				return err
			}
		}
		for memberID := range wantedMembers {
			if err := s.permRepo.AddUserToGroup(memberID, groupID); err != nil {
				return err
			}
		}
	}

	return nil
}

/*
uintSet 将 ID 列表转换为集合。
用于同步关联时去重，避免重复添加同一个成员或审批人。
*/
func uintSet(ids []uint) map[uint]bool {
	set := make(map[uint]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

/*
Delete 删除指定用户组。
删除前会确认用户组存在，并阻止删除内置用户组 NextMeta_Groups。
*/
func (s *groupService) Delete(id uint) error {
	group, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	// 内置用户组用于系统初始化和默认权限兜底，不允许被删除。
	if group.Name == "NextMeta_Groups" {
		return errors.New("内置用户组不可删除")
	}

	return s.repo.Delete(id)
}

/*
normalizeGroupCode 规范化用户组编码。
前端未传 code 时使用 name 兜底，避免返回空编码影响列表展示。
*/
func normalizeGroupCode(code string, name string) string {
	trimmedCode := strings.TrimSpace(code)
	if trimmedCode != "" {
		return trimmedCode
	}
	return strings.TrimSpace(name)
}

/*
normalizeGroupStatus 规范化用户组状态。
disabled 和 inactive 都视为停用，其余值默认归一为 enabled。
*/
func normalizeGroupStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "disabled", "inactive":
		return "disabled"
	default:
		return "enabled"
	}
}

/*
normalizeGroupDescription 规范化用户组描述。
描述为空时返回短横线，避免前端列表展示空白。
*/
func normalizeGroupDescription(description string) string {
	if strings.TrimSpace(description) == "" {
		return "-"
	}
	return description
}

/*
dataSourceIDsToStrings 将数据源模型列表转换为字符串 ID 列表。
转换过程中会按数据源 ID 去重，避免前端多选控件出现重复值。
*/
func dataSourceIDsToStrings(datasources []model.DataSource) []string {
	seen := make(map[uint]bool, len(datasources))
	ids := make([]string, 0, len(datasources))
	for _, datasource := range datasources {
		if seen[datasource.ID] {
			continue
		}
		seen[datasource.ID] = true
		ids = append(ids, strconv.FormatUint(uint64(datasource.ID), 10))
	}
	return ids
}

/*
userIDsToStrings 将用户模型列表转换为字符串 ID 列表。
审批人和成员列表共用该转换逻辑，并按用户 ID 去重。
*/
func userIDsToStrings(users []model.User) []string {
	seen := make(map[uint]bool, len(users))
	ids := make([]string, 0, len(users))
	for _, user := range users {
		if seen[user.ID] {
			continue
		}
		seen[user.ID] = true
		ids = append(ids, strconv.FormatUint(uint64(user.ID), 10))
	}
	return ids
}
