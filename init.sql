-- ========================================
-- NextMeta 数据库初始化脚本
-- 生成时间: 08/12/2025 19:46:20
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
-- Records of audit_logs
-- ----------------------------
BEGIN;
COMMIT;

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
) ENGINE=InnoDB AUTO_INCREMENT=24 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of audit_rules
-- ----------------------------
BEGIN;
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (1, '2025-12-04 18:01:36.509', '2025-12-04 18:30:47.007', NULL, 'ERR.000', 'SQL语法错误', 'SQL语法必须正确', 'error', 'syntax', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (2, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'ERR.001', 'DDL执行错误', 'DDL语句在虚拟环境中执行失败', 'error', 'syntax', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (3, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'ALI.001', '建议使用AS关键字', '在列或表别名(如\"tbl AS alias\")中, 明确使用 AS 关键字比隐含别名(如\"tbl alias\")更易懂', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (4, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'ALI.002', '不建议给列起别名', '不建议对列使用别名', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (5, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'ALI.003', '别名与原名相同', '别名与原名相同，没有意义', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (6, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'ARG.001', '前缀通配符LIKE', 'LIKE查询不建议使用前缀通配符(\'%abc\')', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (7, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'ARG.002', '全通配符LIKE', 'LIKE查询不建议使用全通配符(\'%\')', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (8, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'ARG.003', '隐式类型转换', '检测到隐式类型转换，可能导致无法使用索引', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (9, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'ARG.008', '使用!=或<>操作符', '不建议使用!=或<>操作符，可能导致无法使用索引', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (10, '2025-12-04 18:01:36.509', '2025-12-04 19:23:51.733', NULL, 'CLA.001', '无WHERE条件的SELECT', 'SELECT语句必须包含WHERE条件', 'warning', 'heuristic', 1, '', '');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (11, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'CLA.002', 'ORDER BY RAND()', '禁止使用ORDER BY RAND()', 'error', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (12, '2025-12-04 18:01:36.509', '2025-12-04 20:03:08.460', NULL, 'CLA.003', '大OFFSET翻页', 'LIMIT OFFSET过大，建议优化', 'error', 'heuristic', 1, '', '');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (13, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'CLA.004', 'GROUP BY常量', 'GROUP BY常量没有意义', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (14, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'CLA.005', 'ORDER BY常量', 'ORDER BY常量没有意义', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (15, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'CLA.011', '建表未指定注释', '建议为表和列添加注释', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (16, '2025-12-04 18:01:36.509', '2025-12-04 18:11:50.867', NULL, 'CLA.013', 'DML语句缺少WHERE', 'UPDATE/DELETE语句必须包含WHERE条件', 'error', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (17, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'COL.001', 'SELECT *', '不建议使用SELECT *，请指定具体列', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (18, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'COL.002', 'INSERT未指定列', 'INSERT语句建议显式指定列名', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (19, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'COL.010', '使用了TEXT/BLOB类型', '不建议使用TEXT/BLOB类型', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (20, '2025-12-04 18:01:36.509', '2025-12-04 18:01:36.509', NULL, 'KWR.003', '使用OR条件', '建议将OR改写为IN或UNION', 'warning', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (21, '2025-12-04 18:01:36.509', '2025-12-05 19:47:18.995', NULL, 'FUN.001', '列上使用函数', '在WHERE条件列上使用函数会导致索引失效', 'error', 'heuristic', 1, '', '');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (22, '2025-12-04 18:30:37.891', '2025-12-04 18:30:37.891', NULL, 'SEC.005', '禁止 DROP TABLE', '生产环境中禁止直接执行 DROP TABLE 操作', 'error', 'heuristic', 1, NULL, NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`) VALUES (23, '2025-12-04 18:30:37.934', '2025-12-04 18:30:37.934', NULL, 'SEC.006', '禁止 DROP DATABASE', '生产环境中禁止直接执行 DROP DATABASE 操作', 'error', 'heuristic', 1, NULL, NULL);
COMMIT;

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
-- Records of data_source_masking_rules
-- ----------------------------
BEGIN;
COMMIT;

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
-- Records of data_sources
-- ----------------------------
BEGIN;
COMMIT;

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
-- Records of group_approvers
-- ----------------------------
BEGIN;
COMMIT;

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
-- Records of group_datasources
-- ----------------------------
BEGIN;
COMMIT;

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
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of groups
-- ----------------------------
BEGIN;
INSERT INTO `groups` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `description`, `source`, `dn`) VALUES (1, '2025-12-03 17:52:57.616', '2025-12-03 17:52:57.616', NULL, 'NextMeta_Groups', '', 'local', '');
COMMIT;

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
-- Records of sql_snippets
-- ----------------------------
BEGIN;
COMMIT;

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
  `database` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '目标数据库名',
  PRIMARY KEY (`id`),
  KEY `idx_sql_tickets_deleted_at` (`deleted_at`),
  KEY `fk_sql_tickets_creator` (`creator_id`),
  CONSTRAINT `fk_sql_tickets_creator` FOREIGN KEY (`creator_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of sql_tickets
-- ----------------------------
BEGIN;
COMMIT;

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
-- Records of system_settings
-- ----------------------------
BEGIN;
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('', '【SQL审计平台】工单审批结果通知\n工单标题: {title}\n审批状态: {status}\n处理人: {operator}\n处理时间: {updated_at}\n备注: {remark}', '审批结果通知模版 (Result Notification Template)', '2025-12-08 19:05:14.293', '2025-12-08 19:05:14.293');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('global_sql_limit', '1000', '全局 SQL 查询行数限制 (Global SQL Query Limit)', '2025-12-05 15:39:59.497', '2025-12-08 19:33:39.026');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('key', 'notification_template_result', '', '2025-12-06 14:33:29.147', '2025-12-06 14:37:55.131');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('ldap_auto_sync_cron', '0 * * * *', '', '2025-12-08 19:02:31.242', '2025-12-08 19:02:40.297');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('ldap_auto_sync_enabled', 'false', '', '2025-12-08 19:02:31.159', '2025-12-08 19:33:39.165');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_template_pending', '【SQL审计平台】您有新的工单待审批\n工单标题: {title}\n工单类型: {type}\n提交人: {creator}\n提交时间: {created_at}', '待审批通知模版 (Pending Notification Template)', '2025-12-06 14:20:19.374', '2025-12-06 14:43:03.207');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_template_result', '【SQL审计平台】工单审批结果通知\n工单标题: {title}\n审批状态: {status}\n处理人: {operator}\n处理时间: {updated_at}\n备注: {remark}', '审批结果通知模版 (Result Notification Template)', '2025-12-06 14:20:19.426', '2025-12-06 14:43:03.237');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_webhook', 'https://open.feishu.cn/open-apis/bot/v2/hook/', '通知 Webhook 地址 (Notification Webhook URL)', '2025-12-06 14:20:19.340', '2025-12-06 14:43:03.177');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('ticket_affected_rows_threshold', '1000', '', '2025-12-06 11:32:59.893', '2025-12-08 19:33:39.086');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('value', '【SQL审计平台】工单审批结果通知\n工单标题: {title}\n审批状态: {status}\n处理人: {operator}\n处理时间: {updated_at}\n备注: {remark}', '', '2025-12-06 14:33:29.182', '2025-12-06 14:37:55.161');
COMMIT;

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
-- Records of ticket_approvals
-- ----------------------------
BEGIN;
COMMIT;

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
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of user_groups
-- ----------------------------
BEGIN;
INSERT INTO `user_groups` (`id`, `created_at`, `updated_at`, `deleted_at`, `user_id`, `group_id`) VALUES (1, '2025-12-08 19:36:32.807', '2025-12-08 19:36:32.807', NULL, 1, 1);
COMMIT;

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
  `dn` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'LDAP DN',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of users
-- ----------------------------
BEGIN;
INSERT INTO `users` (`id`, `created_at`, `updated_at`, `deleted_at`, `username`, `password`, `real_name`, `email`, `role`, `status`, `source`, `dn`) VALUES (1, '2025-12-03 14:38:26.235', '2025-12-07 12:30:11.085', NULL, 'NextMeta', '$2a$10$mrEhG98f0rIsPJ/8lg4Hde5mQCK.YK8k6T.gbtTmsvUajINHn8.WC', 'NextMeta', 'test@NextMetacom.com', 'admin', 1, 'local', NULL);
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
