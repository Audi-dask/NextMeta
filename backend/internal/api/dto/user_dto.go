package dto

/*
RegisterRequest 是管理员创建注册用户时提交的请求体。
注册接口使用 real_name 字段作为用户显示姓名，账号和密码会按绑定规则做基础校验。
*/
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	RealName string `json:"real_name" binding:"required"`
	Email    string `json:"email" binding:"omitempty,email"`
}

/*
LoginRequest 是前端登录页提交的请求体。
user_handler.Login 会用用户名和密码完成本地用户认证。
*/
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

/*
RefreshTokenRequest 是 access token 过期后前端续期时提交的请求体。
refresh token 校验通过后会换取新的 token pair。
*/
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

/*
LoginResponse 是登录和续期接口返回给前端的 token 响应。
access token 用于后续 Authorization Bearer 请求，refresh token 用于续期。
*/
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// UserResponse 是历史用户响应结构，当前后端未发现直接引用。
// 先注释观察；如果后续编译或测试发现隐性引用，再恢复该结构。
// type UserResponse struct {
// 	ID       uint   `json:"id"`
// 	Username string `json:"username"`
// 	RealName string `json:"real_name"`
// 	Role     string `json:"role"`
// }

/*
CreateUserRequest 是管理员在用户管理页面创建本地用户时提交的请求体。
用户管理页面使用 realName 字段作为用户显示姓名，并要求显式指定用户角色。
*/
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	RealName string `json:"realName" binding:"required"`
	Email    string `json:"email" binding:"omitempty,email"`
	Role     string `json:"role" binding:"required,oneof=super_admin admin developer readonly"`
}

/*
UpdateUserRequest 是用户管理页面更新用户资料、角色、状态或密码时提交的请求体。
Password 为空时不修改原密码，Status 只允许 enabled 或 disabled。
*/
type UpdateUserRequest struct {
	ID       uint   `json:"id" binding:"required"`
	RealName string `json:"realName"`
	Email    string `json:"email" binding:"omitempty,email"`
	Role     string `json:"role" binding:"required,oneof=super_admin admin developer readonly"`
	Status   string `json:"status" binding:"required,oneof=enabled disabled"`
	Password string `json:"password"`
}

/*
UpdateProfileRequest 是当前登录用户更新个人资料时提交的请求体。
handler 会使用 token 中的 userID 限定只能更新当前用户自己。
*/
type UpdateProfileRequest struct {
	RealName string `json:"real_name" binding:"required"`
	Email    string `json:"email" binding:"omitempty,email"`
}

/*
ChangePasswordRequest 是当前登录用户修改自己密码时提交的请求体。
old_password 用于二次校验当前用户身份，new_password 是要替换的新密码；
handler 会使用 token 中的 userID 限定只能修改当前用户自己，不允许从请求体指定用户 ID。
*/
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
}
