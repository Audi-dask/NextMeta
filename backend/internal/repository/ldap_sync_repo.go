package repository

import (
	"fmt"

	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

/*
LDAPSyncGroupInput 是 LDAP 组同步时需要落库的结构。
MemberDNs 保存该组当前的 LDAP 成员 DN 快照。
*/
type LDAPSyncGroupInput struct {
	DN        string
	Name      string
	MemberDNs []string
}

/*
LDAPSyncResult 汇总一次 LDAP 同步写入本地缓存的结果。
*/
type LDAPSyncResult struct {
	CreatedUsers      int
	UpdatedUsers      int
	DisabledUsers     int
	CreatedGroups     int
	UpdatedGroups     int
	DisabledGroups    int
	SyncedMemberships int
}

/*
LDAPSyncRepository 定义 LDAP 缓存同步的数据访问能力。
同步只处理 source=ldap 的用户和组，local 数据不会被修改。
*/
type LDAPSyncRepository interface {
	Sync(users []model.User, groups []LDAPSyncGroupInput) (LDAPSyncResult, error)
}

type ldapSyncRepository struct {
	db *gorm.DB
}

func NewLDAPSyncRepository(db *gorm.DB) LDAPSyncRepository {
	return &ldapSyncRepository{db: db}
}

func (r *ldapSyncRepository) Sync(users []model.User, groups []LDAPSyncGroupInput) (LDAPSyncResult, error) {
	var result LDAPSyncResult
	err := r.db.Transaction(func(tx *gorm.DB) error {
		seenUserDNs := make([]string, 0, len(users))
		for _, user := range users {
			if user.DN == "" || user.Username == "" {
				continue
			}
			user.Source = "ldap"
			user.Status = "enabled"
			if user.Role == "" {
				user.Role = string(model.UserRoleDeveloper)
			}
			if user.Password == "" {
				user.Password = "ldap"
			}
			if err := upsertLDAPUser(tx, &user, &result); err != nil {
				return err
			}
			seenUserDNs = append(seenUserDNs, user.DN)
		}

		if count, err := disableMissingLDAPUsers(tx, seenUserDNs); err != nil {
			return err
		} else {
			result.DisabledUsers = int(count)
		}

		seenGroupDNs := make([]string, 0, len(groups))
		for _, group := range groups {
			if group.DN == "" || group.Name == "" {
				continue
			}
			storedGroup, err := upsertLDAPGroup(tx, group, &result)
			if err != nil {
				return err
			}
			seenGroupDNs = append(seenGroupDNs, group.DN)
			if err := replaceLDAPGroupMembers(tx, storedGroup.ID, group.MemberDNs, &result); err != nil {
				return err
			}
		}

		if count, err := disableMissingLDAPGroups(tx, seenGroupDNs); err != nil {
			return err
		} else {
			result.DisabledGroups = int(count)
		}

		return nil
	})
	return result, err
}

func upsertLDAPUser(tx *gorm.DB, user *model.User, result *LDAPSyncResult) error {
	var existing model.User
	query := tx.Where("source = ? AND dn = ?", "ldap", user.DN).Limit(1).Find(&existing)
	if query.Error != nil {
		return query.Error
	}
	if query.RowsAffected > 0 {
		applyLDAPUser(&existing, user)
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}
		result.UpdatedUsers++
		return nil
	}

	conflictUser, err := findUserByUsernameUnscoped(tx, user.Username)
	if err != nil {
		return err
	}
	if conflictUser == nil {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		result.CreatedUsers++
		return nil
	}
	if !conflictUser.DeletedAt.Valid && conflictUser.Status == "enabled" {
		return fmt.Errorf("LDAP 用户名 %s 与激活中的现有用户冲突，请先禁用或删除该用户", user.Username)
	}

	if conflictUser.DeletedAt.Valid {
		conflictUser.Role = string(model.UserRoleReadonly)
	}
	applyLDAPUser(conflictUser, user)
	conflictUser.DeletedAt = gorm.DeletedAt{}
	if err := tx.Unscoped().Save(conflictUser).Error; err != nil {
		return err
	}
	result.UpdatedUsers++
	return nil
}

func findUserByUsernameUnscoped(tx *gorm.DB, username string) (*model.User, error) {
	var user model.User
	query := tx.Unscoped().Where("username = ?", username).Limit(1).Find(&user)
	if query.Error != nil {
		return nil, query.Error
	}
	if query.RowsAffected == 0 {
		return nil, nil
	}
	return &user, nil
}

func applyLDAPUser(target *model.User, source *model.User) {
	target.Username = source.Username
	target.Password = source.Password
	target.RealName = source.RealName
	target.Email = source.Email
	if target.Role == "" {
		target.Role = string(model.UserRoleReadonly)
	}
	target.Status = "enabled"
	target.Source = "ldap"
	target.DN = source.DN
}

func disableMissingLDAPUsers(tx *gorm.DB, seenDNs []string) (int64, error) {
	query := tx.Model(&model.User{}).Where("source = ? AND status <> ?", "ldap", "disabled")
	if len(seenDNs) > 0 {
		query = query.Where("dn NOT IN ?", seenDNs)
	}
	result := query.Update("status", "disabled")
	return result.RowsAffected, result.Error
}

func upsertLDAPGroup(tx *gorm.DB, group LDAPSyncGroupInput, result *LDAPSyncResult) (*model.Group, error) {
	name := group.Name + "_AD"
	var existing model.Group
	query := tx.Where("source = ? AND dn = ?", "ldap", group.DN).Limit(1).Find(&existing)
	if query.Error != nil {
		return nil, query.Error
	}
	if query.RowsAffected == 0 {
		created := model.Group{
			Name:        name,
			Code:        name,
			Status:      "enabled",
			Description: "Synced from LDAP",
			Source:      "ldap",
			DN:          group.DN,
		}
		if err := tx.Create(&created).Error; err != nil {
			return nil, err
		}
		result.CreatedGroups++
		return &created, nil
	}

	existing.Name = name
	existing.Code = name
	existing.Status = "enabled"
	existing.Description = "Synced from LDAP"
	if err := tx.Save(&existing).Error; err != nil {
		return nil, err
	}
	result.UpdatedGroups++
	return &existing, nil
}

func replaceLDAPGroupMembers(tx *gorm.DB, groupID uint, memberDNs []string, result *LDAPSyncResult) error {
	if err := tx.Unscoped().Where("group_id = ?", groupID).Delete(&model.UserGroup{}).Error; err != nil {
		return err
	}
	if len(memberDNs) == 0 {
		return nil
	}

	var users []model.User
	if err := tx.Where("source = ? AND status = ? AND dn IN ?", "ldap", "enabled", memberDNs).Find(&users).Error; err != nil {
		return err
	}
	seen := make(map[uint]bool, len(users))
	for _, user := range users {
		if seen[user.ID] {
			continue
		}
		seen[user.ID] = true
		if err := tx.Create(&model.UserGroup{UserID: user.ID, GroupID: groupID}).Error; err != nil {
			return err
		}
		result.SyncedMemberships++
	}
	return nil
}

func disableMissingLDAPGroups(tx *gorm.DB, seenDNs []string) (int64, error) {
	query := tx.Model(&model.Group{}).Where("source = ? AND status <> ?", "ldap", "disabled")
	if len(seenDNs) > 0 {
		query = query.Where("dn NOT IN ?", seenDNs)
	}
	result := query.Update("status", "disabled")
	return result.RowsAffected, result.Error
}
