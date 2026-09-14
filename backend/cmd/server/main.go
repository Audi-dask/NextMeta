package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nextmeta-backend/configs"
	v1 "nextmeta-backend/internal/api/v1"
	"nextmeta-backend/internal/datasource"
	"nextmeta-backend/internal/license"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/internal/router"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/jwt"
	"nextmeta-backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

/*
resolveLicensePath 返回 license.lic 的运行时路径。
生产容器（Dockerfile 设置 GIN_MODE=release）中放在持久化数据卷 /data 下，与二进制分离，重启/重建镜像不丢失。
本地开发仍回退到二进制所在目录或当前工作目录，避免在 macOS 上写入 /data 失败。
*/
func resolveLicensePath() string {
	if os.Getenv("GIN_MODE") == "release" {
		return "/data/license.lic"
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		// go run 把二进制编译到 /tmp/go-build* 或 macOS 的 /var/folders/...；这种情况下用 cwd 更顺手。
		if !strings.Contains(dir, "go-build") && !strings.Contains(dir, "/var/folders/") {
			return filepath.Join(dir, "license.lic")
		}
	}
	return "license.lic"
}

// @title           NextMeta API
// @version         v2.1.0
// @description     NextMeta SQL Audit Platform API

/*
后端服务启动入口，负责按顺序完成配置加载、JWT 生命周期配置、日志初始化、
数据库可用性检查、GORM 连接创建、Repository/Service/Handler 依赖注入，
最后注册 Gin 路由并监听配置文件中的服务端口。
*/
func exitWithError(event string, err error, fields ...zap.Field) {
	fields = append(fields, zap.Error(err))
	logger.Log.Error(event, fields...)
	_ = logger.Log.Sync()
	os.Exit(1)
}

func main() {
	// 1. 初始化日志
	logger.InitLogger()
	defer func() {
		_ = logger.Log.Sync()
	}()

	// 2. 初始化配置
	cfg, err := configs.LoadConfig()
	if err != nil {
		exitWithError("config_load_failed", err)
	}

	jwtCfg := cfg.JWT
	if err := jwt.Configure(jwtCfg.Secret, time.Duration(jwtCfg.Expires)*time.Minute, time.Duration(jwtCfg.Refresh)*time.Minute); err != nil {
		exitWithError("jwt_configure_failed", err)
	}

	// 3. 初始化数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	if err := ensureDatabaseExists(cfg.Database); err != nil {
		exitWithError("database_validation_failed", err)
	}

	db, err := gorm.Open(mysql.Open(dsn), logger.GormConfig())
	if err != nil {
		exitWithError("database_connection_failed", err)
	}
	if !db.Migrator().HasColumn(&model.SQLTicket{}, "StatementResults") {
		if err := db.Migrator().AddColumn(&model.SQLTicket{}, "StatementResults"); err != nil {
			exitWithError("database_migration_failed", err, zap.String("column", "statement_results"))
		}
	}

	// 4. 初始化应用层 (依赖注入)
	licenseSvc, err := service.NewLicenseService(license.EmbeddedPublicKeyPEM, resolveLicensePath())
	if err != nil {
		exitWithError("license_service_initialization_failed", err)
	}

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSystemSettingRepository(db)
	ldapConfigRepo := repository.NewLdapConfigRepository(db)
	feishuConfigRepo := repository.NewFeishuConfigRepository(db)
	stateRepo := repository.NewOAuthStateRepository(db)
	oauthTicketRepo := repository.NewOAuthLoginTicketRepository(db)
	bindingRepo := repository.NewUserOAuthBindingRepository(db)
	ldapSyncRepo := repository.NewLDAPSyncRepository(db)

	ldapSvc := service.NewLDAPService(ldapConfigRepo)
	ldapSyncSvc := service.NewLDAPSyncService(ldapSvc, ldapConfigRepo, ldapSyncRepo)
	loginAuditRepo := repository.NewLoginAuditRepository(db)
	userService := service.NewUserService(userRepo, settingsRepo, ldapSvc)
	userHandler := v1.NewUserHandler(userService, licenseSvc, loginAuditRepo, settingsRepo)

	groupRepo := repository.NewGroupRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	groupService := service.NewGroupService(groupRepo, userRepo, permRepo)
	groupHandler := v1.NewGroupHandler(groupService)

	dsRepo := repository.NewDataSourceRepository(db)
	dsService := service.NewDataSourceService(dsRepo, settingsRepo, datasource.DefaultRegistry)
	auditLogRepo := repository.NewAuditLogRepository(db)
	auditLogService := service.NewAuditLogService(auditLogRepo)
	auditLogHandler := v1.NewAuditLogHandler(auditLogService)

	// 审计服务
	baseRepo := &repository.BaseRepository{DB: db}
	auditService := service.NewAuditService(auditLogRepo, baseRepo)

	permService := service.NewPermissionService(permRepo, dsRepo)

	// 数据源 Handler
	dsHandler := v1.NewDataSourceHandler(dsService, permService, auditService, auditLogService)

	// 权限 & 工单
	permHandler := v1.NewPermissionHandler(permRepo)

	// 通知服务
	notificationSvc := service.NewNotificationService(settingsRepo)

	ticketRepo := repository.NewTicketRepository(db)
	const interruptedExecutionReason = "服务在工单执行期间重启，无法确认 SQL 最终执行状态，请人工核对数据库后处理"
	recoveredCount, err := ticketRepo.FailExecutingTickets(interruptedExecutionReason)
	if err != nil {
		exitWithError("ticket_recovery_failed", err)
	}
	if recoveredCount > 0 {
		logger.Log.Warn("interrupted_tickets_recovered", zap.Int64("count", recoveredCount))
	}
	ticketService := service.NewTicketService(ticketRepo, userRepo, permRepo, permService, dsService, auditLogService, settingsRepo, auditService, notificationSvc)
	ticketHandler := v1.NewTicketHandler(ticketService, auditService)

	dashboardService := service.NewDashboardService(ticketRepo, dsRepo, userRepo, auditLogRepo)
	dashboardHandler := v1.NewDashboardHandler(dashboardService)

	// 审计规则 Handler
	auditRuleHandler := v1.NewAuditRuleHandler(baseRepo)

	// 历史数据清理服务
	cleanupRepo := repository.NewCleanupRepository(db)
	cleanupSvc := service.NewCleanupService(cleanupRepo)

	systemSettingHandler := v1.NewSystemSettingHandler(settingsRepo, ldapSvc, ldapSyncSvc, notificationSvc, licenseSvc, ldapConfigRepo, feishuConfigRepo, userRepo, cleanupSvc)

	snippetExpo := repository.NewSnippetRepository(db)
	snippetSvc := service.NewSnippetService(snippetExpo)
	snippetHandler := v1.NewSnippetHandler(snippetSvc)

	feishuOAuthSvc := service.NewFeishuOAuthService(
		feishuConfigRepo, stateRepo, oauthTicketRepo, bindingRepo, userRepo,
	)
	oauthHandler := v1.NewOAuthHandler(feishuOAuthSvc, loginAuditRepo, settingsRepo, userService)
	loginAuditHandler := v1.NewLoginAuditHandler(loginAuditRepo)

	// 初始化路由
	r := router.InitRouter(
		userHandler,
		groupHandler,
		dsHandler,
		permHandler,
		ticketHandler,
		auditLogHandler,
		dashboardHandler,
		auditRuleHandler,
		systemSettingHandler,
		snippetHandler,
		oauthHandler,
		loginAuditHandler,
		licenseSvc,
		userService,
	)

	ldapSyncSvc.Start(context.Background())

	logger.Log.Info("server_starting", zap.String("address", cfg.Server.Port))
	if err := r.Run(cfg.Server.Port); err != nil {
		exitWithError("server_start_failed", err)
	}
}

/*
启动前检查配置中的 MySQL 数据库是否已经存在。
该函数先使用不带具体库名的 DSN 连接 MySQL 实例，再查询 information_schema.schemata
确认目标库名存在，避免后续 GORM 使用完整 DSN 连接时才暴露数据库缺失问题。
*/
func ensureDatabaseExists(database configs.DatabaseConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local", database.User, database.Password, database.Host, database.Port)

	db, err := gorm.Open(mysql.Open(dsn), logger.GormConfig())
	if err != nil {
		return err
	}

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = ?", database.DBName).Scan(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("database %s does not exist", database.DBName)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
