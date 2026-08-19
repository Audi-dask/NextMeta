package dto

/*
LDAPTestRequest 是系统设置页触发 LDAP 测试时提交的动态策略配置。
固定连接参数来自 config.yaml，这里只接收前端当前表单中的过滤规则和字段映射。
*/
type LDAPTestRequest struct {
	Enabled         bool   `json:"enabled"`
	URL             string `json:"url"`
	BaseDN          string `json:"base_dn"`
	GroupBaseDN     string `json:"group_base_dn"`
	BindDN          string `json:"bind_dn"`
	BindPass        string `json:"bind_pass"`
	UserFilter      string `json:"user_filter" binding:"required"`
	GroupFilter     string `json:"group_filter" binding:"required"`
	MappingJSON     string `json:"mapping_json" binding:"required"`
	ExcludeKeywords string `json:"exclude_keywords"`
}

/*
LDAPTestUserResponse 是 LDAP 测试接口返回的用户预览项。
展示用户名、显示名、邮箱以及所属组，供系统设置页测试结果表格使用。
*/
type LDAPTestUserResponse struct {
	Username string   `json:"username"`
	RealName string   `json:"real_name"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
}

/*
LDAPTestResponse 是 LDAP 测试接口返回结构。
当前只返回用户列表，组信息按用户所属组的形式内嵌展示。
*/
type LDAPTestResponse struct {
	Users []LDAPTestUserResponse `json:"users"`
}

/*
LDAPSyncResponse 是 LDAP 同步接口返回的汇总结果。
前端会基于这些数量展示手动同步结果。
*/
type LDAPSyncResponse struct {
	CreatedUsers      int `json:"created_users"`
	UpdatedUsers      int `json:"updated_users"`
	DisabledUsers     int `json:"disabled_users"`
	CreatedGroups     int `json:"created_groups"`
	UpdatedGroups     int `json:"updated_groups"`
	DisabledGroups    int `json:"disabled_groups"`
	SyncedMemberships int `json:"synced_memberships"`
}
