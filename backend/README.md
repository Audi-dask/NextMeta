# 后端项目索引
后端快速定位手册，只包含功能业务逻辑对应文件映射管理、以及实现效果。

## P0
- backend/cmd/server/main.go # 启动目录、接口注册、依赖注入、数据库初始化、路由注册和服务启动
- backend/configs/config.go  # 配置获取相关，从 config.yaml 读取 server/database/jwt 配置；JWT 有效期统一使用分钟
- backend/config.yaml # 后端运行配置文件，包含服务端口、MySQL、JWT 有效期（分钟）配置
- backend/go.mod # Go模块依赖，目前主项目使用 gin、gorm、mysql、jwt、zap、testify、vitess sqlparser 等
- backend/internal/router/router.go # Gin 路由集中注册，包含 /api/health、/api/v1 接口、JWT鉴权分组、管理员权限分组、License 访问控制、前端静态资源 SPA 回退
- backend/internal/router/middleware.go # Gin 中间件，包含日志、异常恢复、跨域、JWT鉴权、AdminOnly权限判断
- backend/internal/api/dto # API 入参/出参结构定义，负责前后端 JSON 字段协议和基础校验；不是数据库模型，数据源、用户、查询审计等接口会用这里的 DTO

- backend/internal/api/v1/user_handler.go # 用户接口，登录、注册、个人资料、本地用户管理
- backend/internal/api/v1/group_handler.go # 用户组接口，组列表、新增、更新、删除
- backend/internal/api/v1/permission_handler.go # 权限接口，维护用户组成员、组数据源、组审批人，以及用户可访问数据源查询
- backend/internal/api/v1/datasource_handler.go # 数据源接口，数据源增删改查、复制、连接测试、库表字段读取、SQL查询执行；查询窗口只允许 SELECT/EXPLAIN，会注入执行人注释并记录查询审计日志
- backend/internal/api/v1/ticket_handler.go # 工单接口，工单创建、语法检测、审批、撤回、我的工单、待审批工单、审批历史、工单详情、执行结果导出
- backend/internal/api/v1/audit_log_handler.go # 审计日志接口，目前主要查询 SQL 查询审计日志
- backend/internal/api/v1/audit_rule_handler.go # 审核规则接口，规则列表、规则状态更新、规则配置更新，直接依赖 BaseRepository 操作规则表
- backend/internal/api/v1/dashboard_handler.go # 首页看板接口，统计工单、数据源、用户、审计日志等概览数据
- backend/internal/api/v1/system_setting_handler.go # 系统设置接口，配置列表、配置更新、通知测试、LDAP连接测试、历史数据清理；LDAP测试使用 config.yaml 固定连接和前端提交的过滤/映射配置
- backend/internal/api/v1/oauth_handler.go # 飞书 OAuth 登录接口，返回授权地址、处理回调、一次性 ticket 兑换 JWT，并记录登录审计
- backend/internal/api/v1/login_audit_handler.go # 登录审计接口，分页查询本地、LDAP、飞书等登录记录
- backend/internal/api/v1/snippet_handler.go # SQL片段接口，片段新增、列表、更新、删除

- backend/internal/service/user_service.go # 用户业务逻辑，本地用户、登录认证、资料更新等
- backend/internal/service/group_service.go # 用户组业务逻辑，组管理、组成员关联处理
- backend/internal/service/permission_service.go # 权限业务逻辑，判断用户对数据源的访问权限、审批权限、可访问数据源列表
- backend/internal/service/datasource_service.go # 数据源业务逻辑，数据源 CRUD、连接测试、库表字段查询、SQL执行；查询走 global_sql_limit、QueryTimeoutSeconds、max_execution_time，工单执行走 ExecutionTimeoutSeconds

- backend/internal/service/ticket_service.go # 工单核心业务逻辑，创建工单、审批流、撤回审核中工单、审批人查询、状态流转、执行结果导出、通知触发
- backend/internal/service/audit_service.go # SQL审核业务入口，工单提交和语法检测会走这里；先执行静态审核，DataSourceID 存在时叠加动态元数据和 EXPLAIN 审核
- backend/internal/service/audit_log_service.go # 审计日志业务逻辑，查询审计日志保存和列表查询
- backend/internal/service/dashboard_service.go # 看板统计业务逻辑，聚合 repository 数据给首页接口使用
- backend/internal/service/system_setting  # 没有单独 service 文件，系统设置当前主要由 repository + handler + notification service 组合处理
- backend/internal/service/cleanup_service.go # 历史数据清理服务，按截止时间分批物理删除工单审批、工单和查询审计记录
- backend/internal/service/notification_service.go # 通知服务，目前从系统设置读取通知配置，用于工单等场景消息通知/测试通知
- backend/internal/service/ldap_service.go # LDAP服务，目前只提供连接测试、用户查询和组归属预览；固定连接参数来自 config.yaml，过滤规则和字段映射来自系统设置页请求
- backend/internal/service/ldap_sync_service.go # LDAP同步服务，读取系统设置后定时或手动同步 LDAP 用户/组到本地缓存；默认间隔 30 分钟，local 用户不受影响
- backend/internal/service/feishu_oauth_service.go # 飞书 OAuth 登录业务，生成授权地址、消费回调、换取飞书用户信息、创建一次性登录票据、签发 JWT
- backend/internal/service/license_service.go # License 状态服务，负责加载、刷新和判定登录/数据源访问权限
- backend/internal/service/snippet_service.go # SQL片段业务逻辑，片段 CRUD 的 service 层


- backend/internal/repository/base_repository.go # 通用 Repository，持有 *gorm.DB，当前审核规则等少量场景直接使用
- backend/internal/repository/user_repo.go # 用户表数据访问，本地用户、用户列表、用户删除等
- backend/internal/repository/group_repo.go # 用户组表数据访问，组 CRUD 等
- backend/internal/repository/permission_repo.go # 权限关系数据访问，用户组成员、组数据源、组审批人关联维护
- backend/internal/repository/datasource_repo.go # 数据源表数据访问，数据源 CRUD、复制、列表查询
- backend/internal/repository/ticket_repo.go # 工单表数据访问，工单创建、审批流查询、状态更新、详情查询、历史查询
- backend/internal/repository/audit_log_repo.go # 审计日志表数据访问，保存查询日志、分页查询日志
- backend/internal/repository/dashboard_repo  # 没有单独文件，看板统计目前复用 ticket/datasource/user/audit_log repository
- backend/internal/repository/system_setting_repo.go # 系统设置表数据访问，配置读取、批量更新、单项查询
- backend/internal/repository/cleanup_repo.go # 历史数据清理数据访问，按表名白名单分批物理删除 created_at 早于截止时间的记录
- backend/internal/repository/ldap_sync_repo.go # LDAP同步落库逻辑，事务内 upsert LDAP 用户/组、刷新 LDAP 组成员关系，并禁用 LDAP 中已不存在的用户/组
- backend/internal/repository/login_audit_repository.go # 登录审计数据访问，保存和分页查询本地/LDAP/飞书登录记录
- backend/internal/repository/feishu_config_repo.go # 飞书配置表数据访问，读取和更新 App ID、App Secret、回调地址和默认角色
- backend/internal/repository/ldap_config_repo.go # LDAP 配置表数据访问，读取和更新 LDAP 连接、过滤和映射配置
- backend/internal/repository/oauth_state_repo.go # 飞书 OAuth state 数据访问，生成、消费和清理一次性 state
- backend/internal/repository/oauth_login_ticket_repo.go # 飞书 OAuth 一次性登录票据数据访问，生成、消费和清理 ticket
- backend/internal/repository/user_oauth_binding_repo.go # 用户 OAuth 绑定数据访问，维护飞书用户标识与本地用户的绑定关系
- backend/internal/repository/snippet_repo.go # SQL片段表数据访问，片段 CRUD

- backend/internal/model/user.go # 用户模型，包含本地/LDAP 来源字段 Source 和 LDAP DN 字段；同时也包含 SQLTicket 工单模型和 TicketApproval 工单审批记录模型，后期如果整理模型可以考虑拆分
- backend/internal/model/user_role.go # 用户角色/权限相关模型
- backend/internal/model/datasource.go # 数据源模型，保存 MySQL 连接信息和基础元数据
- backend/internal/model/audit_log.go # 审计日志模型，记录查询 SQL、用户、数据源、耗时、结果等
- backend/internal/model/audit_rule.go # 审核规则模型，保存规则编码、名称、描述、级别、启用状态等
- backend/internal/model/dashboard.go # 看板统计返回模型
- backend/internal/model/masking_rule.go # 脱敏规则模型，配合 pkg/masking 使用
- backend/internal/model/login_audit.go # 登录审计模型，记录登录方式、状态、IP、错误信息和时间
- backend/internal/model/oauth_state.go # 飞书 OAuth state 模型，保存防重放和防 CSRF 的一次性状态
- backend/internal/model/oauth_login_ticket.go # 飞书 OAuth 一次性登录票据模型
- backend/internal/model/user_oauth_binding.go # 用户 OAuth 绑定模型，保存第三方身份与本地用户关联
- backend/internal/model/feishu_config.go # 飞书配置模型，保存 OAuth 应用配置
- backend/internal/model/ldap_config.go # LDAP 配置模型，保存连接和同步配置
- backend/internal/model/snippet.go # SQL片段模型
- backend/internal/model/system_setting.go # 系统设置模型，保存通知等系统配置项

- backend/internal/audit/README.md # 自研 SQL 审核核心说明，包含静态审核、动态元数据审核和规则清单，改审核规则前建议先读这里
- backend/internal/audit/engine.go # 审核规则执行引擎，负责组织解析结果和规则执行
- backend/internal/audit/parser.go # SQL拆分、Vitess AST解析、语句类型识别
- backend/internal/audit/report.go # 审核报告汇总、分数计算、阻断状态计算
- backend/internal/audit/rule.go # 审核规则接口、规则启用状态、严重级别适配
- backend/internal/audit/types.go # 审核核心 Request、Report、Suggestion 等核心结构
- backend/internal/audit/engine_test.go # 静态审核核心测试
- backend/internal/audit/dynamic_engine_test.go # 动态审核核心测试
- backend/internal/audit/rules/common.go # 通用审核规则，DDL/DML 都会执行，比如混合类型检测等
- backend/internal/audit/rules/ddl.go # DDL审核规则，建表、改表、删表等结构变更规则
- backend/internal/audit/rules/ddl_scope.go # DDL规则作用域判断，控制 DDL 规则命中范围
- backend/internal/audit/rules/dml.go # DML审核规则，INSERT/UPDATE/DELETE 等数据变更规则
- backend/internal/audit/rules/dml_scope.go # DML规则作用域判断，控制 DML 规则命中范围
- backend/internal/audit/rules/meta.go # 动态元数据规则，检查表、字段、主键和索引字段等真实库信息
- backend/internal/audit/rules/meta_remaining.go # 需要真实库元数据的补充规则，覆盖数据库、索引、ALTER字段和隐式类型转换等检查
- backend/internal/audit/rules/explain.go # EXPLAIN 预估影响行数和全表扫描风险规则
- backend/internal/audit/rules/scope.go # 通用规则作用域工具函数
- backend/internal/audit/static/defaults.go # 默认审核规则注册入口，新增静态规则后一般需要在这里挂载
- backend/internal/audit/dynamic/defaults.go # 动态规则注册入口，新增动态规则后一般需要在这里挂载

- backend/pkg/jwt/jwt.go # JWT生成、解析、从 gin.Context 获取用户信息；SecretKey 与有效期通过启动时 Configure 从 config.yaml 注入
- backend/pkg/logger/logger.go # 统一日志基础设施：Zap 单行 JSON 输出，将 message 收敛为 event 字段，并接管 GORM 与 MySQL driver 日志；GORM 忽略预期的 record not found
- backend/pkg/response/response.go # API统一响应封装，成功/失败响应格式从这里看
- backend/pkg/masking/engine.go # 数据脱敏引擎，按脱敏规则处理查询结果字段

- tmp/tools/cleanup/main.go # 清理/维护类工具，不属于主服务链路，执行前先确认用途
- tmp/tools/cleanup/README.md # cleanup 工具说明
- tmp/tools/inspect/main.go # 检查/探查类工具，不属于主服务链路，排查问题时再看
- backend/kill.sh # 本地停止后端进程脚本
- backend/server # 已构建出的二进制文件，不是源码，通常不需要读


## 日志当前边界

- `pkg/logger` 是后端统一日志出口，应用日志使用 Zap 单行 JSON 编码；原日志调用的 message 会自动收敛到 `event` 字段，例如 `logger.Log.Info("oauth state consumed", ...)` 输出 `{"event":"oauth state consumed", ...}`。
- 每条日志统一包含 `timestamp`、`level`、`caller`、`event` 和业务字段，不再输出 caller 与 JSON 之间的游离消息文本。
- GORM 使用项目自定义 Zap adapter：普通 SQL 默认不输出，数据库执行错误输出 `gorm_query_failed`；`gorm.ErrRecordNotFound` 属于预期查询结果，不输出 ORM 错误 SQL，业务层按需要记录一次 Warn。
- GORM SQL 日志只保留带占位符的 SQL 模板，不展开查询参数，避免 OAuth state、用户条件等敏感值进入日志。
- MySQL driver 内部错误通过同一 Zap 出口输出，事件名为 `mysql_driver_error`，不再出现独立的 `[mysql] YYYY/MM/DD` 文本格式。
- 配置加载和服务启动不再使用 Go 标准库 `log.Printf/log.Fatal`，启动错误由 main 统一输出一次结构化日志。
- 当前尚未加入 request_id，也未调整 OAuth 失败时仍返回 HTTP 200 的响应策略；这两项属于后续链路追踪和接口语义改造，不包含在本次日志格式统一中。
- `closing bad idle connection` 和 `broken pipe` 虽已统一日志格式，但根因仍需通过 MySQL `wait_timeout`、网络空闲超时和连接池生命周期参数共同治理。


## LDAP 当前边界
- 固定连接配置在 backend/config.yaml 的 ldap 段，包括 url/base_dn/group_base_dn/bind_dn/bind_pass
- 动态策略配置由系统设置页提交，包括 user_filter、group_filter、mapping_json、ldap_sync_interval_minutes、ldap_exclude_keywords
- 测试入口是 POST /api/v1/settings/ldap/test，需要管理员权限，只返回用户和组预览，不落库
- 手动同步入口是 POST /api/v1/settings/ldap/sync，需要管理员权限；后台定时同步默认间隔 30 分钟，可通过系统设置调整
- 用户查询以 base_dn 为搜索根，组查询以 group_base_dn 为搜索根，组成员通过 member 或 uniqueMember 中的用户 DN 匹配
- LDAP 用户同步到 users 表，source=ldap；LDAP 删除或查询不到的用户本地保留但 status=disabled，登录时拒绝
- LDAP 新用户首次同步默认 role=readonly；后续同步不覆盖 role，平台角色由本地管理员维护
- local 用户不受 LDAP 同步影响，除非管理员手动删除或禁用
- LDAP 组同步到 groups 表，source=ldap，组名按 LDAP 组名拼接 _AD；LDAP 查询不到的组本地保留但 status=disabled
- LDAP 排除关键字默认 admin，支持英文逗号或中文逗号分隔；用户按 DN/username/real_name/email 匹配，组按 DN/name 匹配，命中后测试和同步都会跳过
- 字段映射 JSON 必须包含非空 username、real_name、email；LDAP 登录用户使用本地缓存用户记录和 LDAP DN bind 认证


## 飞书登录当前边界
- 飞书登录入口是 GET /api/v1/auth/feishu/authorize，前端先拿授权地址再跳转飞书开放平台
- 回调入口是 GET /api/v1/auth/feishu/callback，服务端消费 state 后生成一次性 login ticket，再跳转前端回调页
- 票据兑换入口是 POST /api/v1/auth/feishu/exchange，前端回调页拿 oauth_ticket 换取 JWT token pair
- 登录页状态接口是 GET /api/v1/auth/status，用于判断本地账号登录是否开放
- 飞书配置由系统设置页维护，包含 App ID、App Secret、回调地址和首次登录默认角色
- 飞书登录成功会写入登录审计，审计记录可通过 /api/v1/login-audit 查询


## 工单当前边界
- 新建工单后状态为 pending，前端展示为审核中
- 只有提交人本人可以撤回 pending 状态工单，撤回接口是 POST /api/v1/tickets/:id/withdraw
- 撤回后状态为 withdrawn，前端展示为已撤回；已审批通过、已执行、执行失败、已驳回、已撤回的工单不能撤回


## P1 
- backend/web/embed.go # 前端 dist 静态资源 embed 入口，router 里通过它提供 SPA 页面 无需过多关注
- backend/internal/license/loader.go # license 解析、验签、状态快照定义，license_service.go 依赖这里完成加载和判定

## P3
- reference/sql-audit/goInception  # goInception规则引擎源码，已移出后端目录，只做审核规则参考，非规则改造不必大量读取此路径
