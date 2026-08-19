package model

/*
UserRole 表示系统内置用户角色。
角色值会写入 users.role，并用于路由权限和工单提交权限判断。
*/
type UserRole string

const (
	// UserRoleSuperAdmin 是超级管理员角色，拥有管理员权限。
	UserRoleSuperAdmin UserRole = "super_admin"
	// UserRoleAdmin 是管理员角色，拥有后台管理权限。
	UserRoleAdmin UserRole = "admin"
	// UserRoleDeveloper 是开发者角色，可以提交 SQL 工单。
	UserRoleDeveloper UserRole = "developer"
	// UserRoleReadonly 是只读角色，只能访问允许的查询和查看能力。
	UserRoleReadonly UserRole = "readonly"
)

/*
IsAdminRole 判断角色是否具备管理员权限。
路由层 AdminOnly 中间件会通过该函数限制管理类接口访问。
*/
func IsAdminRole(role string) bool {
	return role == string(UserRoleSuperAdmin) || role == string(UserRoleAdmin)
}

/*
CanSubmitTicket 判断角色是否允许提交 SQL 工单。
只读角色不能提交工单，管理员、超级管理员和开发者可以提交。
*/
func CanSubmitTicket(role string) bool {
	return role == string(UserRoleSuperAdmin) || role == string(UserRoleAdmin) || role == string(UserRoleDeveloper)
}
