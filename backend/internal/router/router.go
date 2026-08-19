package router

import (
	"io/fs"
	"net/http"
	v1 "nextmeta-backend/internal/api/v1"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/web"
	"strings"

	"github.com/gin-gonic/gin"
)

/*
InitRouter 集中创建 Gin Engine 并注册后端所有 HTTP 路由。
这里接收 main.go 注入完成的各类 Handler，统一挂载全局中间件、API 分组、
鉴权边界、管理员权限边界，以及前端静态资源和 SPA 回退处理。
*/
func InitRouter(
	userHandler *v1.UserHandler,
	groupHandler *v1.GroupHandler,
	dsHandler *v1.DataSourceHandler,
	permHandler *v1.PermissionHandler,
	ticketHandler *v1.TicketHandler,
	auditHandler *v1.AuditLogHandler,
	dashboardHandler *v1.DashboardHandler,
	auditRuleHandler *v1.AuditRuleHandler,
	systemSettingHandler *v1.SystemSettingHandler,
	snippetHandler *v1.SnippetHandler,
	oauthHandler *v1.OAuthHandler,
	loginAuditHandler *v1.LoginAuditHandler,
	licenseSvc service.LicenseService,
) *gin.Engine {
	r := gin.New()

	r.Use(Logger())
	r.Use(Recovery())
	r.Use(CORS())

	// ---------------------------------------------------------
	// 1. Health Check (用于 Docker 健康检查)
	// ---------------------------------------------------------
	r.Any("/api/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// ---------------------------------------------------------
	// 2. API Routes
	// ---------------------------------------------------------
	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			// 注册接口仅允许管理员调用，避免生产环境开放任意用户自注册。
			auth.POST("/register", JWTAuth(), AdminOnly(), userHandler.Register)
			auth.POST("/login", userHandler.Login)
			auth.POST("/refresh", userHandler.RefreshToken)

			// 飞书 OAuth 公开接口
			auth.GET("/feishu/authorize", oauthHandler.FeishuAuthorize)
			auth.GET("/feishu/callback", oauthHandler.FeishuCallback)
			auth.POST("/feishu/exchange", oauthHandler.FeishuExchange)
			// 登录页前置状态
			auth.GET("/status", oauthHandler.LoginStatus)
		}

		// 首页看板接口需要登录后访问，用于返回系统概览统计数据。
		dashboard := api.Group("/dashboard")
		dashboard.Use(JWTAuth())
		{
			dashboard.GET("/stats", dashboardHandler.GetStats)
		}

		// 审核规则接口允许登录用户查看规则，规则状态和配置修改仅限管理员。
		rules := api.Group("/rules")
		rules.Use(JWTAuth())
		{
			rules.GET("", auditRuleHandler.ListRules)
			rules.PUT("/:id/status", AdminOnly(), auditRuleHandler.UpdateRuleStatus)
			rules.PUT("/:id", AdminOnly(), auditRuleHandler.UpdateRule)
		}

		user := api.Group("/user")
		user.Use(JWTAuth())
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			// 用户修改自己的密码：userID 取自 JWT，禁止从请求体指定其他用户。
			user.PUT("/password", userHandler.ChangePassword)
			user.GET("/list", AdminOnly(), userHandler.ListLocalUsers)
			user.POST("", AdminOnly(), userHandler.CreateUser)
			// 该接口可改写他人的角色和状态，仅限管理员；普通用户改自己资料/密码请走 /user/profile 和 /user/password。
			user.PUT("", AdminOnly(), userHandler.UpdateUser)
			user.DELETE("", AdminOnly(), userHandler.DeleteUser)
			user.GET("/datasources", permHandler.GetUserAccessibleDataSources)
		}

		// 用户组接口维护组基础信息，以及成员、数据源、审批人三类关联关系。
		groups := api.Group("/groups")
		groups.Use(JWTAuth())
		{
			groups.GET("", AdminOnly(), groupHandler.ListAll)
			groups.POST("", AdminOnly(), groupHandler.Create)
			groups.PUT("", AdminOnly(), groupHandler.Update)
			groups.DELETE("/:id", AdminOnly(), groupHandler.Delete)
			groups.POST("/:id/members", AdminOnly(), permHandler.AddGroupMember)
			groups.DELETE("/:id/members/:userId", AdminOnly(), permHandler.RemoveGroupMember)
			groups.GET("/:id/members", AdminOnly(), permHandler.GetGroupMembers)
			groups.POST("/:id/datasources", AdminOnly(), permHandler.AddGroupDataSource)
			groups.DELETE("/:id/datasources/:dsId", AdminOnly(), permHandler.RemoveGroupDataSource)
			groups.GET("/:id/datasources", AdminOnly(), permHandler.GetGroupDataSources)
			groups.POST("/:id/approvers", AdminOnly(), permHandler.AddGroupApprover)
			groups.DELETE("/:id/approvers/:userId", AdminOnly(), permHandler.RemoveGroupApprover)
			groups.GET("/:id/approvers", AdminOnly(), permHandler.GetGroupApprovers)
		}

		// 数据源接口覆盖数据源 CRUD、连接测试、库表元数据读取和查询窗口 SQL 执行。
		// License 失效时（未上传/过期/验签失败）整组拒绝访问，保证 SQL 执行入口与授权状态强绑定。
		ds := api.Group("/datasources")
		ds.Use(JWTAuth(), LicenseGuardForDataSource(licenseSvc))
		{
			ds.POST("", AdminOnly(), dsHandler.Create)
			ds.PUT("", AdminOnly(), dsHandler.Update)
			ds.GET("", dsHandler.List)
			ds.POST("/test", AdminOnly(), dsHandler.TestConnectionConfig)
			ds.DELETE("/:id", AdminOnly(), dsHandler.Delete)
			ds.POST("/:id/copy", AdminOnly(), dsHandler.Copy)
			ds.POST("/:id/test", AdminOnly(), dsHandler.TestConnection)
			ds.GET("/:id/schemas", dsHandler.FetchSchemas)
			ds.GET("/:id/schemas/columns", dsHandler.FetchColumns)
			ds.POST("/:id/query", dsHandler.ExecuteSQL)
		}

		// 工单接口负责 SQL 工单创建、语法检测、审批流、列表查询和执行结果导出。
		// 工单提交和审批最终会触发数据源上的 SQL，License 失效时同样整组拒绝。
		tickets := api.Group("/tickets")
		tickets.Use(JWTAuth(), LicenseGuardForDataSource(licenseSvc))
		{
			tickets.POST("", ticketHandler.CreateTicket)
			tickets.POST("/check-syntax", ticketHandler.CheckSyntax)
			tickets.GET("/approvers", ticketHandler.GetDatasourceApprovers)
			tickets.GET("/audit-logs", AdminOnly(), ticketHandler.GetTicketAuditLogs)
			tickets.POST("/:id/approve", ticketHandler.ApproveTicket)
			tickets.POST("/:id/withdraw", ticketHandler.WithdrawTicket)
			tickets.GET("/my", ticketHandler.GetMyTickets)
			tickets.GET("/approver", ticketHandler.GetApproverTickets)
			tickets.GET("/pending", ticketHandler.GetPendingTickets)
			tickets.GET("/approval-history", ticketHandler.GetApprovalHistory)
			tickets.GET("/:id", ticketHandler.GetTicketDetail)
			tickets.GET("/:id/result", ticketHandler.ExportTicketResult)
		}

		// 审计日志接口当前主要面向管理员查询 SQL 查询窗口产生的审计记录。
		audit := api.Group("/audit")
		audit.Use(JWTAuth())
		{
			audit.GET("/query-logs", AdminOnly(), auditHandler.GetQueryAuditLogs)
		}

		// 系统设置接口用于读取和更新通知等运行配置，修改和测试通知仅限管理员。
		// License 相关接口刻意不挂 LicenseGuard，保证授权失效时管理员仍能查看状态并上传新 license。
		settings := api.Group("/settings")
		settings.Use(JWTAuth())
		{
			settings.GET("", AdminOnly(), systemSettingHandler.List)
			settings.PUT("", AdminOnly(), systemSettingHandler.Update)
			settings.GET("/notifications", AdminOnly(), systemSettingHandler.ListNotifications)
			settings.POST("/ldap/test", AdminOnly(), systemSettingHandler.TestLDAP)
			settings.POST("/ldap/sync", AdminOnly(), systemSettingHandler.SyncLDAP)
			settings.GET("/ldap", AdminOnly(), systemSettingHandler.GetLdapConfig)
			settings.PUT("/ldap", AdminOnly(), systemSettingHandler.UpdateLdapConfig)
			settings.GET("/feishu", AdminOnly(), systemSettingHandler.GetFeishuConfig)
			settings.PUT("/feishu", AdminOnly(), systemSettingHandler.UpdateFeishuConfig)
			settings.POST("/test-notify", AdminOnly(), systemSettingHandler.TestNotify)
			settings.GET("/license", AdminOnly(), systemSettingHandler.GetLicense)
			settings.POST("/license", AdminOnly(), systemSettingHandler.UploadLicense)
		}

		// 登录审计
		loginAudit := api.Group("/login-audit")
		loginAudit.Use(JWTAuth())
		{
			loginAudit.GET("", AdminOnly(), loginAuditHandler.List)
		}

		// SQL 片段接口提供用户常用 SQL 片段的新增、查询、更新和删除能力。
		snippets := api.Group("/snippets")
		snippets.Use(JWTAuth())
		{
			snippets.POST("", snippetHandler.Create)
			snippets.GET("", snippetHandler.List)
			snippets.PUT("", snippetHandler.Update)
			snippets.DELETE("/:id", snippetHandler.Delete)
		}
	}

	// ---------------------------------------------------------
	// 3. Static Files (SPA Frontend)
	// ---------------------------------------------------------
	// 从 embed.FS 中获取前端构建产物 dist 目录，供后续静态资源服务和 SPA 回退使用。
	distFS, _ := fs.Sub(web.StaticFS, "dist")

	// /assets 必须挂到 dist/assets，否则请求 /assets/xxx.js 时会错误地到 dist 根目录查找。
	assetsFS, _ := fs.Sub(distFS, "assets")
	r.StaticFS("/assets", http.FS(assetsFS))

	// 统一接管未命中路由，根据路径区分 API 404、静态资源请求和前端 SPA 路由。
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 请求保持 JSON 格式返回，避免被前端 index.html 回退吞掉真实接口错误。
		if strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{"code": 404, "msg": "API route not found"})
			return
		}

		// 非 assets 前缀的静态文件尝试直接从 dist 中读取，例如 favicon 或 manifest。
		cleanPath := strings.TrimPrefix(path, "/")
		if file, err := distFS.Open(cleanPath); err == nil {
			stat, _ := file.Stat()
			file.Close()
			if !stat.IsDir() {
				c.FileFromFS(cleanPath, http.FS(distFS))
				return
			}
		}

		// 前端 SPA 路由统一回退到 index.html，由浏览器端路由继续解析页面。
		// 这里直接读取文件内容，避免 c.FileFromFS 在部分路径下触发重定向循环。
		if f, err := distFS.Open("index.html"); err == nil {
			defer f.Close()
			stat, _ := f.Stat()
			content := make([]byte, stat.Size())
			f.Read(content)
			c.Data(200, "text/html; charset=utf-8", content)
		} else {
			c.String(404, "Page not found")
		}
	})

	return r
}
