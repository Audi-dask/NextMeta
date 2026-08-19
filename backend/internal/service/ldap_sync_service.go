package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/pkg/logger"

	"go.uber.org/zap"
)

const defaultLDAPSyncIntervalMinutes = 30

type LDAPSyncService interface {
	SyncNow() (repository.LDAPSyncResult, error)
	Start(ctx context.Context)
}

type ldapSyncService struct {
	ldapSvc        LDAPService
	ldapConfigRepo repository.LdapConfigRepository
	syncRepo       repository.LDAPSyncRepository
	mu             sync.Mutex
}

func NewLDAPSyncService(
	ldapSvc LDAPService,
	ldapConfigRepo repository.LdapConfigRepository,
	syncRepo repository.LDAPSyncRepository,
) LDAPSyncService {
	return &ldapSyncService{
		ldapSvc:        ldapSvc,
		ldapConfigRepo: ldapConfigRepo,
		syncRepo:       syncRepo,
	}
}

func (s *ldapSyncService) SyncNow() (repository.LDAPSyncResult, error) {
	if !s.mu.TryLock() {
		return repository.LDAPSyncResult{}, errors.New("LDAP 同步正在执行中")
	}
	defer s.mu.Unlock()

	cfg, err := s.ldapConfigRepo.Get()
	if err != nil {
		return repository.LDAPSyncResult{}, err
	}
	if !cfg.Enabled {
		return repository.LDAPSyncResult{}, errors.New("LDAP 未启用，无法执行同步")
	}

	req := s.buildRequest(cfg)

	directory, err := s.ldapSvc.Fetch(req)
	if err != nil {
		return repository.LDAPSyncResult{}, err
	}
	users := make([]model.User, 0, len(directory.Users))
	for _, ldapUser := range directory.Users {
		users = append(users, model.User{
			Username: ldapUser.Username,
			Password: "ldap",
			RealName: ldapUser.RealName,
			Email:    ldapUser.Email,
			Role:     string(model.UserRoleReadonly),
			Status:   "enabled",
			Source:   "ldap",
			DN:       ldapUser.DN,
		})
	}

	groups := make([]repository.LDAPSyncGroupInput, 0, len(directory.Groups))
	for _, ldapGroup := range directory.Groups {
		groups = append(groups, repository.LDAPSyncGroupInput{
			DN:        ldapGroup.DN,
			Name:      ldapGroup.Name,
			MemberDNs: ldapGroup.MemberDNs,
		})
	}
	return s.syncRepo.Sync(users, groups)
}

func (s *ldapSyncService) Start(ctx context.Context) {
	go func() {
		for {
			cfg, err := s.ldapConfigRepo.Get()
			interval := defaultLDAPSyncIntervalMinutes
			if err == nil && cfg.SyncInterval >= 1 {
				interval = cfg.SyncInterval
			}

			timer := time.NewTimer(time.Duration(interval) * time.Minute)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				cfg, err := s.ldapConfigRepo.Get()
				if err != nil {
					logger.Log.Warn("Failed to read LDAP config", zap.Error(err))
					continue
				}
				if !cfg.Enabled {
					continue
				}
				if _, err := s.SyncNow(); err != nil {
					logger.Log.Warn("LDAP scheduled sync failed", zap.Error(err))
				}
			}
		}
	}()
}

func (s *ldapSyncService) buildRequest(cfg *model.LdapConfig) *dto.LDAPTestRequest {
	mapping, _ := json.Marshal(map[string]string{
		"username":  cfg.MappingUsername,
		"real_name": cfg.MappingRealName,
		"email":     cfg.MappingEmail,
	})
	return &dto.LDAPTestRequest{
		Enabled:         cfg.Enabled,
		UserFilter:      cfg.UserFilter,
		GroupFilter:     cfg.GroupFilter,
		MappingJSON:     string(mapping),
		ExcludeKeywords: cfg.ExcludeKeywords,
	}
}
