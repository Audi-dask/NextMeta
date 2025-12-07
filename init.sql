-- ========================================
-- NextMeta 数据库初始化脚本
-- 生成时间: 2025-12-07 14:23:30
-- 用途: Docker 部署的数据库初始化
-- ========================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;


-- ----------------------------
-- Table structure for audit_logs
-- ----------------------------
DROP TABLE IF EXISTS `audit_logs`;
CREATE TABLE `audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned NOT NULL COMMENT '操作用户ID',
  `username` varchar(50) COLLATE utf8mb4_general_ci NOT NULL COMMENT '操作用户名',
  `action` varchar(50) COLLATE utf8mb4_general_ci NOT NULL COMMENT '操作类型',
  `ip` varchar(50) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '客户端IP',
  `details` text COLLATE utf8mb4_general_ci COMMENT '详情',
  `status` tinyint DEFAULT '1' COMMENT '状态(1:成功, 0:失败)',
  PRIMARY KEY (`id`),
  KEY `idx_audit_logs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for audit_rules
-- ----------------------------
DROP TABLE IF EXISTS `audit_rules`;
CREATE TABLE `audit_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `code` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
  `name` varchar(100) COLLATE utf8mb4_general_ci NOT NULL,
  `description` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `severity` varchar(20) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'warning',
  `type` varchar(50) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT '1',
  `explanation` text COLLATE utf8mb4_general_ci,
  `example` text COLLATE utf8mb4_general_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_audit_rules_code` (`code`),
  KEY `idx_audit_rules_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for data_source_masking_rules
-- ----------------------------
DROP TABLE IF EXISTS `data_source_masking_rules`;
CREATE TABLE `data_source_masking_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `data_source_id` bigint unsigned NOT NULL COMMENT '关联数据源ID',
  `pattern` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT '匹配模式(正则或通配符)',
  `rule_type` varchar(50) COLLATE utf8mb4_general_ci NOT NULL COMMENT '脱敏类型(mask_middle, mask_all, mask_left, mask_right)',
  `description` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '规则描述',
  PRIMARY KEY (`id`),
  KEY `idx_data_source_masking_rules_deleted_at` (`deleted_at`),
  KEY `idx_data_source_masking_rules_data_source_id` (`data_source_id`),
  CONSTRAINT `fk_data_sources_masking_rules` FOREIGN KEY (`data_source_id`) REFERENCES `data_sources` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for data_sources
-- ----------------------------
DROP TABLE IF EXISTS `data_sources`;
CREATE TABLE `data_sources` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT '数据源名称',
  `type` varchar(20) COLLATE utf8mb4_general_ci NOT NULL COMMENT '数据库类型(MySQL/PostgreSQL/Redis...)',
  `host` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '主机地址',
  `port` bigint NOT NULL COMMENT '端口',
  `database` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '数据库名',
  `username` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户名',
  `password` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '密码(加密)',
  `group` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '分组/项目',
  `connect_timeout` bigint DEFAULT '10' COMMENT '连接超时时间(秒)',
  `sql_timeout` bigint DEFAULT '60' COMMENT 'SQL执行超时时间(秒)',
  `description` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述',
  `status` varchar(20) COLLATE utf8mb4_general_ci DEFAULT 'active' COMMENT '状态(active/inactive)',
  PRIMARY KEY (`id`),
  KEY `idx_data_sources_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for group_approvers
-- ----------------------------
DROP TABLE IF EXISTS `group_approvers`;
CREATE TABLE `group_approvers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `group_id` bigint unsigned NOT NULL COMMENT '组ID',
  `user_id` bigint unsigned NOT NULL COMMENT '审批人用户ID',
  PRIMARY KEY (`id`),
  KEY `idx_group_approvers_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for group_datasources
-- ----------------------------
DROP TABLE IF EXISTS `group_datasources`;
CREATE TABLE `group_datasources` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `group_id` bigint unsigned NOT NULL COMMENT '组ID',
  `data_source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  PRIMARY KEY (`id`),
  KEY `idx_group_datasources_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for groups
-- ----------------------------
DROP TABLE IF EXISTS `groups`;
CREATE TABLE `groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT '组名',
  `description` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述',
  `source` varchar(20) COLLATE utf8mb4_general_ci DEFAULT 'local' COMMENT '来源(local/ldap)',
  `dn` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'LDAP DN',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_groups_name` (`name`),
  KEY `idx_groups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for sql_snippets
-- ----------------------------
DROP TABLE IF EXISTS `sql_snippets`;
CREATE TABLE `sql_snippets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `title` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT '片段标题',
  `content` text COLLATE utf8mb4_general_ci NOT NULL COMMENT 'SQL内容',
  PRIMARY KEY (`id`),
  KEY `idx_sql_snippets_deleted_at` (`deleted_at`),
  KEY `idx_sql_snippets_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for sql_tickets
-- ----------------------------
DROP TABLE IF EXISTS `sql_tickets`;
CREATE TABLE `sql_tickets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `creator_id` bigint unsigned NOT NULL COMMENT '创建人ID',
  `group_id` bigint unsigned NOT NULL COMMENT '所属组ID',
  `data_source_id` bigint unsigned NOT NULL COMMENT '目标数据源ID',
  `title` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '工单标题',
  `sql_content` text COLLATE utf8mb4_general_ci NOT NULL COMMENT 'SQL内容',
  `ticket_type` varchar(20) COLLATE utf8mb4_general_ci NOT NULL COMMENT '工单类型:query/change',
  `status` varchar(20) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'pending' COMMENT '状态:pending/approved/rejected/executed/failed',
  `execute_result` text COLLATE utf8mb4_general_ci COMMENT '执行结果',
  `is_force` tinyint(1) DEFAULT '0' COMMENT '是否强制提交',
  PRIMARY KEY (`id`),
  KEY `idx_sql_tickets_deleted_at` (`deleted_at`),
  KEY `fk_sql_tickets_creator` (`creator_id`),
  CONSTRAINT `fk_sql_tickets_creator` FOREIGN KEY (`creator_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for system_settings
-- ----------------------------
DROP TABLE IF EXISTS `system_settings`;
CREATE TABLE `system_settings` (
  `key` varchar(100) COLLATE utf8mb4_general_ci NOT NULL,
  `value` text COLLATE utf8mb4_general_ci,
  `description` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for ticket_approvals
-- ----------------------------
DROP TABLE IF EXISTS `ticket_approvals`;
CREATE TABLE `ticket_approvals` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `ticket_id` bigint unsigned NOT NULL COMMENT '工单ID',
  `approver_id` bigint unsigned NOT NULL COMMENT '审批人ID',
  `action` varchar(20) COLLATE utf8mb4_general_ci NOT NULL COMMENT '操作:approve/reject',
  `comment` text COLLATE utf8mb4_general_ci COMMENT '审批意见',
  `approved_at` datetime(3) DEFAULT NULL COMMENT '审批时间',
  PRIMARY KEY (`id`),
  KEY `idx_ticket_approvals_deleted_at` (`deleted_at`),
  KEY `fk_ticket_approvals_approver` (`approver_id`),
  KEY `fk_sql_tickets_approvals` (`ticket_id`),
  CONSTRAINT `fk_sql_tickets_approvals` FOREIGN KEY (`ticket_id`) REFERENCES `sql_tickets` (`id`),
  CONSTRAINT `fk_ticket_approvals_approver` FOREIGN KEY (`approver_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for user_groups
-- ----------------------------
DROP TABLE IF EXISTS `user_groups`;
CREATE TABLE `user_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `group_id` bigint unsigned NOT NULL COMMENT '组ID',
  PRIMARY KEY (`id`),
  KEY `idx_user_groups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for users
-- ----------------------------
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `username` varchar(50) COLLATE utf8mb4_general_ci NOT NULL COMMENT '用户名',
  `password` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '加密密码',
  `real_name` varchar(50) COLLATE utf8mb4_general_ci NOT NULL COMMENT '真实姓名',
  `email` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '邮箱',
  `role` varchar(20) COLLATE utf8mb4_general_ci DEFAULT 'user' COMMENT '角色(admin/user)',
  `status` tinyint DEFAULT '1' COMMENT '状态(1:正常, 2:禁用)',
  `source` varchar(20) COLLATE utf8mb4_general_ci DEFAULT 'local' COMMENT '来源(local/ldap)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


-- ========================================
-- 初始数据
-- ========================================


-- 默认管理员账号: NextMeta / password123
INSERT INTO `users` (`id`, `created_at`, `updated_at`, `deleted_at`, `username`, `password`, `real_name`, `email`, `role`, `status`, `source`)
VALUES (1, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'NextMeta', '$2b$12$MZlhUO1LQf16ZOtGtNO.jebUgXQFW1U9fm7jLINOHaHTKvXXcbU2K', '系统管理员', 'admin@nextmeta.com', 'admin', 1, 'local');


-- 默认审计规则
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (1, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'ERR.000', 'SQL语法错误', 'SQL语法必须正确', 'error', 'syntax', 1, 'SQL语句存在语法错误，无法被数据库解析。', 'SELEC * FROM table');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (2, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'ERR.001', 'DDL执行错误', 'DDL语句在虚拟环境中执行失败', 'error', 'syntax', 1, 'DDL语句在模拟执行时失败，可能是因为表不存在或权限不足。', 'ALTER TABLE non_existent_table ADD COLUMN c1 INT');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (3, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'ALI.001', '建议使用AS关键字', '在列或表别名中, 明确使用 AS 关键字比隐含别名更易懂', 'warning', 'heuristic', 1, '使用 AS 关键字可以提高 SQL 的可读性，避免歧义。', 'SELECT col1 AS c1 FROM table AS t');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (4, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'ALI.003', '别名与原名相同', '别名与原名相同，没有意义', 'warning', 'heuristic', 1, '别名如果和原名一样，则是多余的。', 'SELECT col1 AS col1 FROM table');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (5, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'ARG.001', '前缀通配符LIKE', 'LIKE查询不建议使用前缀通配符(\'%abc\')', 'warning', 'heuristic', 1, '前缀通配符会导致索引失效，进行全表扫描。', 'SELECT * FROM table WHERE col LIKE \'%abc\'');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (6, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'ARG.002', '全通配符LIKE', 'LIKE查询不建议使用全通配符(\'%\')', 'warning', 'heuristic', 1, '全通配符没有过滤意义，且效率极低。', 'SELECT * FROM table WHERE col LIKE \'%\'');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (7, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'ARG.003', '隐式类型转换', '检测到隐式类型转换，可能导致无法使用索引', 'warning', 'heuristic', 1, '列类型与比较值类型不一致会导致隐式转换，从而使索引失效。', 'SELECT * FROM table WHERE varchar_col = 123');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (8, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'ARG.008', '使用!=或<>操作符', '不建议使用!=或<>操作符，可能导致无法使用索引', 'warning', 'heuristic', 1, '不等号查询通常无法使用索引。', 'SELECT * FROM table WHERE col != 1');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (9, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'CLA.001', '无WHERE条件的SELECT', 'SELECT语句必须包含WHERE条件', 'error', 'heuristic', 1, '全表扫描会消耗大量资源，甚至拖垮数据库。', 'SELECT * FROM table');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (10, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'CLA.002', 'ORDER BY RAND()', '禁止使用ORDER BY RAND()', 'error', 'heuristic', 1, 'ORDER BY RAND() 效率极低，不适合大数据量场景。', 'SELECT * FROM table ORDER BY RAND()');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (11, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'CLA.003', '大OFFSET翻页', 'LIMIT OFFSET过大，建议优化', 'warning', 'heuristic', 1, '大 OFFSET 会导致数据库扫描大量无用数据。', 'SELECT * FROM table LIMIT 100000, 10');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (12, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'CLA.011', '建表未指定注释', '建议为表和列添加注释', 'warning', 'heuristic', 1, '注释有助于理解表结构和业务含义。', 'CREATE TABLE t (id INT) COMMENT \'test table\'');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (13, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'CLA.013', 'DML语句缺少WHERE', 'UPDATE/DELETE语句必须包含WHERE条件', 'error', 'heuristic', 1, '没有 WHERE 条件的 UPDATE/DELETE 会影响全表数据，极其危险。', 'DELETE FROM table');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (14, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'COL.001', 'SELECT *', '不建议使用SELECT *，请指定具体列', 'warning', 'heuristic', 1, 'SELECT * 会查询不需要的列，增加网络和IO开销，且表结构变更可能导致应用报错。', 'SELECT * FROM table');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (15, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'COL.002', 'INSERT未指定列', 'INSERT语句建议显式指定列名', 'warning', 'heuristic', 1, '显式指定列名可以避免表结构变更导致的插入错误。', 'INSERT INTO table VALUES (1, \'a\')');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (16, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'COL.010', '使用了TEXT/BLOB类型', '不建议使用TEXT/BLOB类型', 'warning', 'heuristic', 1, '大字段会影响查询性能和磁盘空间。', 'CREATE TABLE t (c TEXT)');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (17, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'KWR.003', '使用OR条件', '建议将OR改写为IN或UNION', 'warning', 'heuristic', 1, 'OR 条件可能导致索引失效。', 'SELECT * FROM table WHERE a=1 OR a=2');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (18, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'FUN.001', '列上使用函数', '在WHERE条件列上使用函数会导致索引失效', 'error', 'heuristic', 1, '对索引列进行函数运算会导致无法使用索引。', 'SELECT * FROM table WHERE DATE(create_time) = \'2023-01-01\'');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (19, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'SEC.005', '禁止 DROP TABLE', '生产环境中禁止直接执行 DROP TABLE 操作', 'error', 'heuristic', 1, 'DROP TABLE 是高危操作，可能导致数据丢失。', 'DROP TABLE users');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`)
VALUES (20, '2025-12-07 14:23:30', '2025-12-07 14:23:30', NULL, 'SEC.006', '禁止 DROP DATABASE', '生产环境中禁止直接执行 DROP DATABASE 操作', 'error', 'heuristic', 1, 'DROP DATABASE 是极高危操作，可能导致整个库数据丢失。', 'DROP DATABASE test_db');

-- 默认系统设置
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`)
VALUES ('global_sql_limit', '1000', '全局 SQL 查询行数限制 (Global SQL Query Limit)', '2025-12-07 14:23:30', '2025-12-07 14:23:30');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`)
VALUES ('ticket_affected_rows_threshold', '5000', '工单影响行数安全阈值 (Ticket Affected Rows Safety Threshold)', '2025-12-07 14:23:30', '2025-12-07 14:23:30');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`)
VALUES ('notification_webhook', '', '通知 Webhook 地址 (Notification Webhook URL)', '2025-12-07 14:23:30', '2025-12-07 14:23:30');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`)
VALUES ('notification_template_pending', '【SQL审计平台】您有新的工单待审批\n工单标题: {title}\n工单类型: {type}\n提交人: {creator}\n提交时间: {created_at}', '待审批通知模版 (Pending Notification Template)', '2025-12-07 14:23:30', '2025-12-07 14:23:30');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`)
VALUES ('notification_template_result', '【SQL审计平台】工单审批结果通知\n工单标题: {title}\n审批状态: {status}\n处理人: {operator}\n处理时间: {updated_at}\n备注: {remark}', '审批结果通知模版 (Result Notification Template)', '2025-12-07 14:23:30', '2025-12-07 14:23:30');


SET FOREIGN_KEY_CHECKS = 1;