package service

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"

	"github.com/go-ldap/ldap/v3"
)

type LDAPTestMapping struct {
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Email    string `json:"email"`
}

type LDAPPreviewUser struct {
	DN       string
	Username string
	RealName string
	Email    string
	Groups   []string
}

type LDAPDirectoryGroup struct {
	DN        string
	Name      string
	MemberDNs []string
}

type LDAPDirectory struct {
	Users  []LDAPPreviewUser
	Groups []LDAPDirectoryGroup
}

type LDAPService interface {
	Test(req *dto.LDAPTestRequest) ([]LDAPPreviewUser, error)
	Fetch(req *dto.LDAPTestRequest) (*LDAPDirectory, error)
	Authenticate(userDN, password string) error
}

type ldapService struct {
	ldapConfigRepo repository.LdapConfigRepository
}

func NewLDAPService(ldapConfigRepo repository.LdapConfigRepository) LDAPService {
	return &ldapService{ldapConfigRepo: ldapConfigRepo}
}

func (s *ldapService) Test(req *dto.LDAPTestRequest) ([]LDAPPreviewUser, error) {
	directory, err := s.Fetch(req)
	if err != nil {
		return nil, err
	}
	return directory.Users, nil
}

func (s *ldapService) Fetch(req *dto.LDAPTestRequest) (*LDAPDirectory, error) {
	// 测试时前端传连接字段用前端值，同步时留空则从数据库读
	cfg, err := s.ldapConfigRepo.Get()
	if err != nil {
		return nil, err
	}
	if req.URL != "" || req.BaseDN != "" || req.BindDN != "" {
		cfg.URL = req.URL
		cfg.BaseDN = req.BaseDN
		cfg.GroupBaseDN = req.GroupBaseDN
		cfg.BindDN = req.BindDN
		if req.BindPass != "" {
			cfg.BindPass = req.BindPass
		}
	}
	if err := validateLDAPConnConfig(cfg); err != nil {
		return nil, err
	}

	mapping, err := parseLDAPMapping(req.MappingJSON)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.UserFilter) == "" || strings.TrimSpace(req.GroupFilter) == "" {
		return nil, errors.New("LDAP 配置未完成，缺少过滤规则")
	}
	excludeKeywords := parseLDAPExcludeKeywords(req.ExcludeKeywords)

	conn, err := s.connect(cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	usersByDN, err := s.loadUsers(conn, cfg.BaseDN, req.UserFilter, mapping, excludeKeywords)
	if err != nil {
		return nil, err
	}
	groups, err := s.loadGroups(conn, cfg.GroupBaseDN, req.GroupFilter, usersByDN, excludeKeywords)
	if err != nil {
		return nil, err
	}

	users := make([]LDAPPreviewUser, 0, len(usersByDN))
	for _, user := range usersByDN {
		sort.Strings(user.Groups)
		users = append(users, *user)
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	return &LDAPDirectory{Users: users, Groups: groups}, nil
}

func (s *ldapService) Authenticate(userDN, password string) error {
	cfg, err := s.ldapConfigRepo.Get()
	if err != nil {
		return err
	}
	if err := validateLDAPConnConfig(cfg); err != nil {
		return err
	}
	if strings.TrimSpace(userDN) == "" || strings.TrimSpace(password) == "" {
		return errors.New("invalid credentials")
	}
	conn, err := ldap.DialURL(cfg.URL)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Bind(userDN, password)
}

func validateLDAPConnConfig(cfg *model.LdapConfig) error {
	if strings.TrimSpace(cfg.URL) == "" ||
		strings.TrimSpace(cfg.BaseDN) == "" ||
		strings.TrimSpace(cfg.GroupBaseDN) == "" ||
		strings.TrimSpace(cfg.BindDN) == "" ||
		strings.TrimSpace(cfg.BindPass) == "" {
		return errors.New("LDAP 配置未完成，缺少固定连接配置")
	}
	return nil
}

func (s *ldapService) connect(cfg *model.LdapConfig) (*ldap.Conn, error) {
	conn, err := ldap.DialURL(cfg.URL)
	if err != nil {
		return nil, err
	}
	if err := conn.Bind(cfg.BindDN, cfg.BindPass); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func parseLDAPMapping(raw string) (LDAPTestMapping, error) {
	var mapping LDAPTestMapping
	if err := json.Unmarshal([]byte(raw), &mapping); err != nil {
		return LDAPTestMapping{}, errors.New("字段映射 JSON 不合法")
	}
	mapping.Username = strings.TrimSpace(mapping.Username)
	mapping.RealName = strings.TrimSpace(mapping.RealName)
	mapping.Email = strings.TrimSpace(mapping.Email)
	if mapping.Username == "" || mapping.RealName == "" || mapping.Email == "" {
		return LDAPTestMapping{}, errors.New("字段映射 JSON 必须包含非空的 username、real_name、email")
	}
	return mapping, nil
}

func parseLDAPExcludeKeywords(raw string) []string {
	replaced := strings.ReplaceAll(raw, "，", ",")
	parts := strings.Split(replaced, ",")
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		keyword := strings.ToLower(strings.TrimSpace(part))
		if keyword == "" {
			continue
		}
		keywords = append(keywords, keyword)
	}
	return keywords
}

func (s *ldapService) loadUsers(conn *ldap.Conn, baseDN, userFilter string, mapping LDAPTestMapping, excludeKeywords []string) (map[string]*LDAPPreviewUser, error) {
	attributes := uniqueStrings([]string{mapping.Username, mapping.RealName, mapping.Email})
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		userFilter,
		attributes,
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	usersByDN := make(map[string]*LDAPPreviewUser)
	for _, entry := range searchResult.Entries {
		username := strings.TrimSpace(entry.GetAttributeValue(mapping.Username))
		if username == "" {
			continue
		}
		user := &LDAPPreviewUser{
			DN:       entry.DN,
			Username: username,
			RealName: strings.TrimSpace(entry.GetAttributeValue(mapping.RealName)),
			Email:    strings.TrimSpace(entry.GetAttributeValue(mapping.Email)),
			Groups:   []string{},
		}
		if ldapUserMatchesExclude(user, excludeKeywords) {
			continue
		}
		usersByDN[entry.DN] = user
	}
	return usersByDN, nil
}

func (s *ldapService) loadGroups(conn *ldap.Conn, groupBaseDN, groupFilter string, usersByDN map[string]*LDAPPreviewUser, excludeKeywords []string) ([]LDAPDirectoryGroup, error) {
	searchRequest := ldap.NewSearchRequest(
		groupBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		groupFilter,
		[]string{"cn", "member", "uniqueMember"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	groups := make([]LDAPDirectoryGroup, 0, len(searchResult.Entries))
	for _, entry := range searchResult.Entries {
		groupName := strings.TrimSpace(entry.GetAttributeValue("cn"))
		if groupName == "" {
			groupName = entry.DN
		}
		if ldapGroupMatchesExclude(entry.DN, groupName, excludeKeywords) {
			continue
		}
		members := entry.GetAttributeValues("member")
		if len(members) == 0 {
			members = entry.GetAttributeValues("uniqueMember")
		}
		group := LDAPDirectoryGroup{DN: entry.DN, Name: groupName, MemberDNs: members}
		groups = append(groups, group)
		for _, memberDN := range members {
			if user, ok := usersByDN[memberDN]; ok {
				user.Groups = append(user.Groups, groupName)
			}
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})
	return groups, nil
}

func ldapUserMatchesExclude(user *LDAPPreviewUser, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}
	return containsLDAPExcludeKeyword([]string{user.DN, user.Username, user.RealName, user.Email}, keywords)
}

func ldapGroupMatchesExclude(dn, name string, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}
	return containsLDAPExcludeKeyword([]string{dn, name}, keywords)
}

func containsLDAPExcludeKeyword(values []string, keywords []string) bool {
	for _, value := range values {
		lowerValue := strings.ToLower(value)
		for _, keyword := range keywords {
			if strings.Contains(lowerValue, keyword) {
				return true
			}
		}
	}
	return false
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func MapLDAPPreviewUsers(users []LDAPPreviewUser) dto.LDAPTestResponse {
	items := make([]dto.LDAPTestUserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, dto.LDAPTestUserResponse{
			Username: user.Username,
			RealName: user.RealName,
			Email:    user.Email,
			Groups:   user.Groups,
		})
	}
	return dto.LDAPTestResponse{Users: items}
}
