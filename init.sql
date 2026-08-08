/*
 Navicat Premium Dump SQL

 Source Server         : nextmeta
 Source Server Type    : MySQL
 Source Server Version : 80046 (8.0.46)
 Source Host           : localhost:3306
 Source Schema         : nextmeta

 Target Server Type    : MySQL
 Target Server Version : 80046 (8.0.46)
 File Encoding         : 65001

 Date: 08/08/2026 22:19:22
*/

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
  `username` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '操作用户名',
  `action` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '操作类型',
  `query_session_id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '查询说明会话ID',
  `ip` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '客户端IP',
  `details` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '详情',
  `status` tinyint DEFAULT '1' COMMENT '状态(1:成功, 0:失败)',
  `data_source_id` bigint unsigned DEFAULT '0' COMMENT '数据源ID',
  `data_source` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '数据源名称',
  `database` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '数据库名',
  `sql_content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'SQL内容',
  `duration_ms` bigint DEFAULT '0' COMMENT '执行耗时毫秒',
  `row_count` bigint DEFAULT '0' COMMENT '查询返回行数',
  `exported` tinyint(1) DEFAULT '0' COMMENT '是否导出',
  PRIMARY KEY (`id`),
  KEY `idx_audit_logs_deleted_at` (`deleted_at`),
  KEY `idx_audit_logs_query_session_id` (`query_session_id`)
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
  `code` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `description` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `severity` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'warning',
  `type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT '1',
  `explanation` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
  `example` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
  `config` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_audit_rules_code` (`code`),
  KEY `idx_audit_rules_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=96 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of audit_rules
-- ----------------------------
BEGIN;
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (24, '2026-05-12 08:40:51.924', '2026-05-12 08:40:51.924', NULL, 'NM_SQL_PARSE_ERROR', 'SQL语法错误', 'SQL语句必须能被解析', 'error', 'common', 1, 'SQL语句存在语法错误或无法被解析时会阻断提交。', 'SELEC * FROM users', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (25, '2026-05-12 08:40:51.930', '2026-05-12 08:40:51.930', NULL, 'NM_SQL_EMPTY_STATEMENT', '空SQL语句', 'SQL内容不能为空', 'error', 'common', 1, '提交审核的 SQL 内容不能为空。', '', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (26, '2026-05-12 08:40:51.942', '2026-05-12 08:40:51.942', NULL, 'NM_SQL_ONLY_DDL_DML', '仅允许DDL或DML', '工单SQL仅允许DDL或DML语句', 'error', 'common', 1, '工单流程只处理 DDL/DML 变更类 SQL，查询窗口不使用该规则。', 'SELECT * FROM users', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (27, '2026-05-12 08:40:51.951', '2026-05-12 08:40:51.951', NULL, 'NM_SQL_TICKET_TYPE_MATCH', '工单类型与SQL类型匹配', 'DDL和DML不允许混合执行', 'error', 'common', 1, 'DDL 工单只能包含 DDL，DML 工单只能包含 DML。', 'CREATE TABLE t(id INT); INSERT INTO t(id) VALUES (1);', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (28, '2026-05-12 08:40:51.957', '2026-05-12 08:40:51.957', NULL, 'NM_DML_UPDATE_REQUIRE_WHERE', 'UPDATE必须包含WHERE', 'UPDATE语句必须包含WHERE条件', 'error', 'DML', 1, '避免无条件 UPDATE 影响整表数据。', 'UPDATE users SET name = \'test\'', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (29, '2026-05-12 08:40:51.962', '2026-05-14 12:25:48.914', NULL, 'NM_DML_DELETE_REQUIRE_WHERE', 'DELETE必须包含WHERE', 'DELETE语句必须包含WHERE条件', 'error', 'DML', 1, '避免无条件 DELETE 删除整表数据。', 'DELETE FROM users', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (30, '2026-05-12 08:40:51.965', '2026-05-12 08:40:51.965', NULL, 'NM_DML_UPDATE_DELETE_FORBID_LIMIT', 'UPDATE/DELETE禁止LIMIT', 'UPDATE或DELETE语句禁止使用LIMIT', 'error', 'DML', 1, 'LIMIT 可能导致变更范围不稳定，建议使用明确 WHERE 条件。', 'DELETE FROM users WHERE status = 0 LIMIT 10', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (31, '2026-05-12 08:40:51.969', '2026-05-12 08:40:51.969', NULL, 'NM_DML_UPDATE_DELETE_FORBID_ORDER_BY', 'UPDATE/DELETE禁止ORDER BY', 'UPDATE或DELETE语句禁止使用ORDER BY', 'error', 'DML', 1, 'ORDER BY 搭配变更语句容易造成执行不确定性。', 'UPDATE users SET status = 1 ORDER BY id', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (32, '2026-05-12 08:40:51.973', '2026-05-12 08:40:51.973', NULL, 'NM_DML_INSERT_REQUIRE_COLUMNS', 'INSERT必须显式指定列', 'INSERT语句必须显式指定列名', 'error', 'DML', 1, '显式指定列名可以避免表结构变化导致插入错位。', 'INSERT INTO users VALUES (1, \'test\')', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (33, '2026-05-12 08:40:51.982', '2026-05-12 08:40:51.982', NULL, 'NM_DML_INSERT_DUPLICATE_COLUMNS', 'INSERT列名不能重复', 'INSERT语句中列名不能重复', 'error', 'DML', 1, '重复列名会导致 SQL 语义不明确。', 'INSERT INTO users(id, id) VALUES (1, 2)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (34, '2026-05-12 08:40:51.985', '2026-05-12 08:40:51.985', NULL, 'NM_DML_INSERT_COLUMNS_VALUES_MATCH', 'INSERT列和值数量匹配', 'INSERT列数量必须和值数量一致', 'error', 'DML', 1, '列数量和值数量不一致会导致执行失败或数据错位。', 'INSERT INTO users(id, name) VALUES (1)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (35, '2026-05-12 08:40:51.989', '2026-07-23 11:07:38.202', NULL, 'NM_DML_INSERT_MAX_ROWS', 'INSERT行数限制', '单条INSERT语句写入行数不能超过限制', 'error', 'DML', 1, '限制单条 INSERT 写入规模，降低大批量变更风险。', 'INSERT INTO users(id) VALUES (1), (2), ...', '{\"max_rows\":2}');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (36, '2026-05-12 08:40:51.992', '2026-05-12 08:40:51.992', NULL, 'NM_DML_FORBID_REPLACE', '禁止REPLACE', '禁止使用REPLACE语句', 'error', 'DML', 1, 'REPLACE 可能隐式删除再插入数据，风险较高。', 'REPLACE INTO users(id, name) VALUES (1, \'test\')', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (37, '2026-05-12 08:40:51.996', '2026-05-12 08:40:51.996', NULL, 'NM_DDL_FORBID_DROP_DATABASE', '禁止DROP DATABASE', '禁止执行DROP DATABASE', 'error', 'DDL', 1, 'DROP DATABASE 属于极高危操作，会删除整个数据库。', 'DROP DATABASE prod', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (38, '2026-05-12 08:40:51.999', '2026-05-12 08:40:51.999', NULL, 'NM_DDL_FORBID_TRUNCATE', '禁止TRUNCATE', '禁止执行TRUNCATE', 'error', 'DDL', 1, 'TRUNCATE 会快速清空整表数据，风险较高。', 'TRUNCATE TABLE users', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (39, '2026-05-12 08:40:52.002', '2026-05-12 08:40:52.002', NULL, 'NM_DDL_CREATE_TABLE_REQUIRE_PK', 'CREATE TABLE必须包含主键', 'CREATE TABLE语句必须定义主键', 'error', 'DDL', 1, '业务表应具备主键，便于数据定位和维护。', 'CREATE TABLE users (name VARCHAR(50))', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (40, '2026-05-12 08:40:52.005', '2026-07-22 11:57:20.577', NULL, 'NM_DDL_FORBID_FOREIGN_KEY', '禁止外键', '禁止创建外键约束', 'error', 'DDL', 1, '外键可能增加变更和维护复杂度，建议由业务层保证约束。', 'CREATE TABLE orders (user_id INT, FOREIGN KEY (user_id) REFERENCES users(id))', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (41, '2026-05-12 08:40:52.008', '2026-05-12 08:40:52.008', NULL, 'NM_DDL_FORBID_CREATE_TABLE_AS', '禁止CREATE TABLE AS SELECT', '禁止使用CREATE TABLE AS SELECT', 'error', 'DDL', 1, 'CTAS 可能产生大量数据复制，影响线上稳定性。', 'CREATE TABLE users_bak AS SELECT * FROM users', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (42, '2026-05-12 08:40:52.015', '2026-05-12 08:40:52.015', NULL, 'NM_DDL_OBJECT_NAME_PATTERN', '对象命名规范', '数据库对象名称必须符合命名规范', 'warning', 'DDL', 1, '对象名称建议使用小写字母、数字和下划线。', 'CREATE TABLE UserInfo (id INT PRIMARY KEY)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (43, '2026-05-12 08:40:52.020', '2026-05-12 08:40:52.020', NULL, 'NM_DDL_OBJECT_NAME_MAX_LENGTH', '库/表/字段/索引名长度限制', '库名、表名、字段名、索引名不能超过长度限制', 'warning', 'DDL', 1, '该规则限制库名、表名、字段名、索引名的最大长度，过长对象名不利于维护和识别。', 'CREATE TABLE very_very_very_very_very_very_long_table_name (id INT PRIMARY KEY)', '{\"max_length\":32}');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (44, '2026-05-12 09:16:45.151', '2026-05-12 09:16:45.151', NULL, 'NM_SQL_MULTI_STATEMENT_LIMIT', '多语句数量限制', '单个工单SQL语句数量不能超过限制', 'warning', 'common', 1, '限制单个工单内提交过多 SQL 语句，降低批量变更风险。当前默认限制为 100 条。', 'UPDATE users SET status = 1 WHERE id = 1; ...', '{\"max_statements\":100}');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (45, '2026-05-12 09:16:45.164', '2026-05-12 09:16:45.164', NULL, 'NM_DML_INSERT_FORBID_SELECT', '禁止INSERT SELECT', '禁止使用INSERT SELECT写入数据', 'error', 'DML', 1, 'INSERT SELECT 可能产生不可控的大批量写入，当前阶段默认阻断。', 'INSERT INTO users_bak(id, name) SELECT id, name FROM users', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (46, '2026-05-12 09:16:45.171', '2026-05-12 09:16:45.171', NULL, 'NM_DML_WHERE_FORBID_ALWAYS_TRUE', '禁止恒真WHERE条件', 'WHERE条件中不允许使用恒真表达式', 'error', 'DML', 1, 'WHERE 1=1 等恒真条件容易放大变更范围，应使用明确业务条件。', 'UPDATE users SET status = 0 WHERE 1 = 1', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (47, '2026-05-12 09:16:45.178', '2026-05-12 09:16:45.178', NULL, 'NM_DDL_CREATE_TABLE_REQUIRE_COMMENT', 'CREATE TABLE必须有表注释', 'CREATE TABLE语句必须包含表注释', 'warning', 'DDL', 1, '表注释有助于说明业务含义和维护责任。', 'CREATE TABLE users (id INT PRIMARY KEY)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (48, '2026-05-12 09:16:45.182', '2026-05-12 09:16:45.182', NULL, 'NM_DDL_COLUMN_REQUIRE_COMMENT', '字段必须有注释', 'CREATE TABLE字段必须包含COMMENT注释', 'warning', 'DDL', 1, '字段注释有助于降低误用和维护成本。', 'CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (49, '2026-05-12 09:58:53.668', '2026-05-13 18:23:57.584', NULL, 'NM_DDL_COLUMN_REQUIRE_NOT_NULL', '字段默认要求NOT NULL', '字段要求显式声明NOT NULL', 'warning', 'DDL', 1, 'CREATE TABLE和可解析的ALTER TABLE字段定义中，普通字段应显式声明NOT NULL，避免NULL值带来查询和业务语义风险。主键和自增字段会被排除。', 'CREATE TABLE users (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(50) NULL)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (50, '2026-05-12 09:58:53.688', '2026-05-13 18:23:57.591', NULL, 'NM_DDL_COLUMN_REQUIRE_DEFAULT', '字段要求默认值', '字段要求设置DEFAULT默认值', 'warning', 'DDL', 1, 'CREATE TABLE和可解析的ALTER TABLE字段定义中，普通字段应设置DEFAULT默认值，降低写入时隐式NULL或隐式默认行为的风险。主键和自增字段会被排除。', 'CREATE TABLE users (id BIGINT PRIMARY KEY AUTO_INCREMENT, status TINYINT NOT NULL)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (51, '2026-05-12 09:58:53.696', '2026-05-13 18:23:57.592', NULL, 'NM_DDL_FORBID_BLOB_TEXT', '禁止BLOB/TEXT类型', '禁止使用BLOB或TEXT类型', 'warning', 'DDL', 1, 'BLOB/TEXT类型容易带来存储、索引和查询性能风险。当前规则会检查BLOB、TEXT、TINYTEXT、MEDIUMTEXT、LONGTEXT等类型。', 'CREATE TABLE posts (id BIGINT PRIMARY KEY, content TEXT)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (52, '2026-05-12 09:58:53.704', '2026-05-13 18:23:57.593', NULL, 'NM_DDL_FORBID_JSON', '禁止JSON类型', '禁止使用JSON类型', 'warning', 'DDL', 1, 'JSON字段会弱化结构约束并增加查询和索引复杂度，默认不建议在变更工单中新增。', 'CREATE TABLE configs (id BIGINT PRIMARY KEY, data JSON)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (53, '2026-05-12 09:58:53.710', '2026-05-13 18:23:57.594', NULL, 'NM_DDL_CHAR_LENGTH_LIMIT', 'CHAR长度限制', 'CHAR长度超过64建议改用VARCHAR', 'warning', 'DDL', 1, 'CHAR适合较短且固定长度的字段。长度超过64时，建议改用VARCHAR以减少空间浪费。', 'CREATE TABLE users (id BIGINT PRIMARY KEY, code CHAR(128) NOT NULL)', '{\"max_length\":64}');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (54, '2026-05-12 09:58:53.716', '2026-05-13 18:23:57.597', NULL, 'NM_DDL_DECIMAL_PRECISION_LIMIT', 'DECIMAL精度限制', 'DECIMAL精度或小数位不能超过限制', 'warning', 'DDL', 1, 'DECIMAL默认限制为precision <= 38、scale <= 18，避免过高精度带来存储和计算成本。', 'CREATE TABLE orders (id BIGINT PRIMARY KEY, amount DECIMAL(65,30) NOT NULL)', '{\"max_precision\":38,\"max_scale\":18}');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (55, '2026-05-12 09:58:53.720', '2026-05-13 18:23:57.597', NULL, 'NM_DDL_TIMESTAMP_DEFAULT_REQUIRED', 'TIMESTAMP要求默认值', 'TIMESTAMP字段要求设置DEFAULT默认值', 'warning', 'DDL', 1, 'TIMESTAMP字段应显式设置DEFAULT，避免不同MySQL版本或SQL模式下出现隐式默认值差异。', 'CREATE TABLE logs (id BIGINT PRIMARY KEY, created_at TIMESTAMP NOT NULL)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (56, '2026-05-12 09:58:53.724', '2026-05-13 18:23:57.598', NULL, 'NM_DDL_DATETIME_DEFAULT_REQUIRED', 'DATETIME要求默认值', 'DATETIME字段要求设置DEFAULT默认值', 'warning', 'DDL', 1, 'DATETIME字段应显式设置DEFAULT，避免写入时出现非预期NULL或隐式默认行为。', 'CREATE TABLE logs (id BIGINT PRIMARY KEY, created_at DATETIME NOT NULL)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (57, '2026-05-12 09:58:53.729', '2026-05-13 18:23:57.598', NULL, 'NM_DDL_FORBID_MULTI_TIMESTAMP_AUTO_UPDATE', '禁止多个自动更新时间字段', '同一张表不允许多个ON UPDATE CURRENT_TIMESTAMP字段', 'warning', 'DDL', 1, '多个自动更新时间字段会让更新时间语义不清晰，默认只允许一个ON UPDATE CURRENT_TIMESTAMP字段。', 'CREATE TABLE logs (id BIGINT PRIMARY KEY, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (58, '2026-05-12 09:58:53.735', '2026-05-13 18:23:57.599', NULL, 'NM_DDL_FORBID_PARTITION_TABLE', '禁止或限制分区表', '禁止或限制使用分区表', 'warning', 'DDL', 1, '分区表会增加DDL、备份、查询计划和运维复杂度，默认在审核中提示风险。', 'CREATE TABLE logs (id BIGINT PRIMARY KEY) PARTITION BY HASH(id) PARTITIONS 4', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (59, '2026-05-12 09:58:53.741', '2026-05-13 18:23:57.599', NULL, 'NM_DDL_INDEX_NAME_REQUIRED', '索引必须显式命名', '索引必须显式指定名称', 'warning', 'DDL', 1, '显式索引名便于后续维护、排查和变更。主键索引不受此规则限制。', 'CREATE TABLE users (id BIGINT PRIMARY KEY, KEY (name))', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (60, '2026-05-12 09:58:53.746', '2026-05-13 18:23:57.599', NULL, 'NM_DDL_INDEX_PREFIX_REQUIRED', '索引命名前缀规范', '普通索引建议idx_前缀，唯一索引建议uniq_前缀', 'warning', 'DDL', 1, '索引命名使用统一前缀可以提升可读性。普通索引建议idx_，唯一索引建议uniq_。', 'CREATE TABLE users (id BIGINT PRIMARY KEY, KEY name_index (name), UNIQUE KEY email_index (email))', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (61, '2026-05-12 09:58:53.751', '2026-05-13 18:23:57.599', NULL, 'NM_DDL_INDEX_MAX_COLUMNS', '联合索引字段数限制', '联合索引字段数不能超过5', 'warning', 'DDL', 1, '过长联合索引维护成本高，写入开销大，也容易无法被有效使用。当前默认限制为5个字段。', 'CREATE INDEX idx_many_cols ON users(a,b,c,d,e,f)', '{\"max_columns\":5}');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (62, '2026-05-12 09:58:53.762', '2026-05-13 18:23:57.600', NULL, 'NM_DDL_INDEX_MAX_COUNT', '单表索引数量限制', 'CREATE TABLE中的索引数量不能超过8', 'warning', 'DDL', 1, '当前静态规则只统计CREATE TABLE语句内定义的索引数量，已有表索引总数需要后续动态审核通过information_schema补齐。', 'CREATE TABLE users (id BIGINT PRIMARY KEY, KEY idx_a(a), KEY idx_b(b), KEY idx_c(c), KEY idx_d(d), KEY idx_e(e), KEY idx_f(f), KEY idx_g(g), KEY idx_h(h), KEY idx_i(i))', '{\"max_count\":8}');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (63, '2026-05-12 09:58:53.777', '2026-05-13 18:23:57.600', NULL, 'NM_DDL_INDEX_DUPLICATE_COLUMNS', '索引内字段不能重复', '同一个索引内不能重复出现字段', 'warning', 'DDL', 1, '索引字段重复没有实际收益，会造成SQL语义不清晰。', 'CREATE INDEX idx_dup ON users(name, name)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (64, '2026-05-12 09:58:53.787', '2026-05-13 18:23:57.600', NULL, 'NM_DDL_INDEX_FORBID_BLOB_TEXT', 'BLOB/TEXT字段不允许建索引', 'CREATE TABLE内BLOB/TEXT字段不允许建索引', 'warning', 'DDL', 1, '当前静态规则只覆盖CREATE TABLE中字段定义和索引定义在同一条SQL内的场景，已有表CREATE INDEX需要后续动态审核通过information_schema补齐。', 'CREATE TABLE posts (id BIGINT PRIMARY KEY, content TEXT, KEY idx_content(content))', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (67, '2026-05-12 09:58:53.810', '2026-05-13 18:54:49.654', NULL, 'NM_DML_UPDATE_FORBID_PK_CHANGE', '禁止更新主键', 'UPDATE不允许修改主键字段', 'error', 'DML', 1, '动态审核会读取目标表主键信息，阻断UPDATE直接修改主键字段的高风险操作。', 'UPDATE users SET id = 2 WHERE id = 1', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (68, '2026-05-12 09:58:53.816', '2026-05-13 18:54:49.654', NULL, 'NM_DML_WHERE_REQUIRE_INDEX_COLUMN', 'WHERE至少包含索引字段', 'WHERE条件至少应包含一个索引字段', 'warning', 'DML', 1, '动态审核会读取目标表索引字段，检查UPDATE/DELETE的WHERE条件是否至少包含一个索引字段，降低全表扫描和大范围锁定风险。', 'UPDATE users SET status = 1 WHERE name = \'alice\'', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (69, '2026-05-12 09:58:53.821', '2026-05-13 18:23:57.601', NULL, 'NM_DML_FORBID_SUBQUERY', '禁止复杂子查询', 'DML语句中禁止使用子查询', 'warning', 'DML', 1, 'DML中的子查询可能造成执行计划复杂、锁范围扩大或影响行数难以评估，默认在审核中提示风险。', 'UPDATE users SET status = 1 WHERE id IN (SELECT user_id FROM orders)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (70, '2026-05-12 09:58:53.828', '2026-05-13 20:09:56.889', NULL, 'NM_META_DATABASE_EXISTS', '数据库必须存在', '目标数据库必须存在', 'error', 'META', 1, '动态审核会通过information_schema.SCHEMATA检查目标数据库是否存在。目标库来自请求database字段，若为空则使用数据源默认数据库。连接失败仍由NM_META_CONNECTION_FAILED处理。', 'UPDATE users SET status = 1 WHERE id = 1', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (71, '2026-05-12 09:58:53.832', '2026-05-13 18:54:49.651', NULL, 'NM_META_TABLE_EXISTS', '表必须存在', 'SQL引用的表必须存在', 'error', 'META', 1, '动态审核会连接目标数据源读取information_schema元数据，校验DML或DDL引用的目标表是否存在。', 'UPDATE users SET status = 1 WHERE id = 1', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (72, '2026-05-12 09:58:53.836', '2026-05-13 18:54:49.653', NULL, 'NM_META_COLUMN_EXISTS', '字段必须存在', 'SQL引用的字段必须存在', 'error', 'META', 1, '动态审核会基于目标表真实字段信息，校验SQL中引用的字段是否存在。当前优先覆盖单表DML场景。', 'UPDATE users SET status = 1 WHERE missing_col = 1', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (73, '2026-05-12 09:58:53.842', '2026-05-13 20:09:56.903', NULL, 'NM_META_INDEX_EXISTS', '索引必须存在或不存在', '创建索引时不能重复，删除索引时目标索引必须存在', 'warning', 'META', 1, '动态审核会读取information_schema.STATISTICS，校验CREATE INDEX、DROP INDEX以及ALTER TABLE ADD/DROP INDEX的常见语法。创建已存在索引或删除不存在索引会触发提示。', 'DROP INDEX idx_name ON users', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (74, '2026-05-12 09:58:53.857', '2026-05-13 18:54:49.654', NULL, 'NM_META_INSERT_COLUMN_EXISTS', 'INSERT字段必须存在', 'INSERT字段必须存在于目标表', 'error', 'META', 1, '动态审核会校验INSERT显式字段列表中的字段是否存在于目标表。', 'INSERT INTO users(missing_col) VALUES (1)', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (75, '2026-05-12 09:58:53.871', '2026-05-13 18:54:49.654', NULL, 'NM_META_UPDATE_COLUMN_EXISTS', 'UPDATE字段必须存在', 'UPDATE SET字段必须存在于目标表', 'error', 'META', 1, '动态审核会校验UPDATE SET中的目标字段是否存在于目标表。', 'UPDATE users SET missing_col = 1 WHERE id = 1', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (76, '2026-05-12 09:58:53.879', '2026-05-13 20:09:56.905', NULL, 'NM_META_ALTER_COLUMN_EXISTS', 'ALTER目标字段符合当前表结构', 'ALTER字段变更必须符合当前表结构', 'error', 'META', 1, '动态审核会读取information_schema.COLUMNS，校验ALTER TABLE ADD/MODIFY/CHANGE/DROP COLUMN常见语法。ADD字段不能已存在，MODIFY/DROP字段必须存在，CHANGE原字段必须存在且新字段不能冲突。', 'ALTER TABLE users MODIFY COLUMN missing_col INT', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (77, '2026-05-12 09:58:53.888', '2026-05-13 19:45:46.162', NULL, 'NM_META_EXPLAIN_AFFECT_ROWS', 'EXPLAIN预估影响行数限制', 'EXPLAIN预估扫描行数不能超过10000', 'warning', 'META', 1, '动态审核会对UPDATE、DELETE、INSERT SELECT、REPLACE SELECT和SELECT执行EXPLAIN。当前默认阈值为10000行；当预估扫描行数超过阈值，或访问类型为ALL且未命中索引时，会提示执行计划风险。不使用EXPLAIN ANALYZE。', 'UPDATE users SET status = 1 WHERE created_at < \'2020-01-01\'', '{\"max_rows\":10000}');
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (78, '2026-05-12 09:58:53.901', '2026-05-13 20:09:56.907', NULL, 'NM_META_IMPLICIT_TYPE_CONVERSION', 'WHERE隐式类型转换检测', 'WHERE条件存在可能的隐式类型转换时提示', 'warning', 'META', 1, '动态审核会基于目标表字段类型检查单表UPDATE/DELETE/SELECT的WHERE条件。字符串字段与数字字面量比较，或数值字段与非数字字符串比较时，会提示隐式类型转换风险。函数表达式、CAST和多表字段归属暂不覆盖。', 'SELECT * FROM users WHERE phone = 13800138000', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (79, '2026-05-13 18:54:49.614', '2026-05-13 18:54:49.614', NULL, 'NM_META_CONNECTION_FAILED', '动态审核连接失败', '动态审核连接目标库或读取元数据失败', 'error', 'META', 1, '动态审核需要连接目标数据源读取information_schema元数据。如果连接失败、超时或权限不足，将阻断提交。', '目标库连接失败或information_schema权限不足', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (80, '2026-05-14 15:04:54.860', '2026-07-22 16:05:50.000', NULL, 'NM_DML_UPDATE_DELETE_JOIN_RISK', '禁止多表UPDATE/DELETE', '单条UPDATE/DELETE不允许包含多表JOIN或多目标写入', 'error', 'DML', 1, '当前动态审核无法可靠确认多表写入中的目标表、字段归属、索引使用和隐式类型转换，因此单条多表 UPDATE/DELETE 直接阻断。一个工单包含多条单表 UPDATE/DELETE 不受影响。', 'UPDATE users u JOIN orders o ON u.id = o.user_id SET u.status = 1 WHERE o.order_no = \'x\'', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (81, '2026-05-14 15:04:54.860', '2026-05-14 15:04:54.860', NULL, 'NM_DML_WHERE_INDEX_COLUMN_FUNCTION_RISK', '索引字段函数包裹风险', 'WHERE条件中索引字段被函数包裹，可能导致索引失效', 'warning', 'DML', 1, 'WHERE 条件中对索引字段使用函数或表达式包装，可能导致 MySQL 无法有效使用普通 BTree 索引。默认提示警告；如需严格管控，可在审核规则管理页面将严重级别调整为 error。', 'UPDATE users SET status = 1 WHERE ABS(age) = 18', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (82, '2026-07-22 15:48:50.000', '2026-07-22 15:48:50.000', NULL, 'NM_DML_ON_DUPLICATE_FORBID_NON_IDEMPOTENT', '禁止ON DUPLICATE非幂等更新', 'ON DUPLICATE KEY UPDATE不允许基于字段当前值进行累加或计算', 'error', 'DML', 1, '字段自增、递减或基于当前值计算的表达式在重复执行时会继续变化，无法保证幂等性。', 'INSERT INTO users(id, retry_count) VALUES (1, 1) ON DUPLICATE KEY UPDATE retry_count = retry_count + 1;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (83, '2026-07-23 00:00:00.000', '2026-07-23 00:00:00.000', NULL, 'NM_SQL_CROSS_DATABASE_FORBIDDEN', '禁止跨数据库执行', 'SQL只能访问工单选择的目标数据库', 'warning', 'common', 1, '当工单选择数据库 db_a 时，SQL 中显式操作 db_b.table_name 这类其他库对象会触发提示。当前默认只警告不阻断；如需严格限制，可在审核规则管理页面将严重级别调整为 error。开启跨库后，部分动态元数据审核仍以工单选择库为主。', '工单选择 db_a，但提交 SQL：UPDATE db_b.users SET status = 0 WHERE id = 1;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (84, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_DDL_DROP_TABLE_RISK', 'DROP TABLE风险提醒', 'DROP TABLE可能造成数据永久丢失，默认仅提醒风险、不阻断执行', 'warning', 'DDL', 1, '检测到DROP TABLE时提示确认目标表、备份和恢复方案。该规则默认仅作风险提醒，不阻断工单。', 'DROP TABLE audit_logs;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (85, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_DDL_DROP_COLUMN_RISK', 'DROP COLUMN风险提醒', '删除字段可能造成数据永久丢失，默认仅提醒风险、不阻断执行', 'warning', 'DDL', 1, '检测到ALTER TABLE DROP COLUMN时提示确认字段数据、依赖关系和恢复方案。该规则默认仅作风险提醒，不阻断工单。', 'ALTER TABLE users DROP COLUMN legacy_code;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (86, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_DDL_RENAME_TABLE_RISK', 'RENAME TABLE风险提醒', '重命名表可能影响现有业务访问，默认仅提醒风险、不阻断执行', 'warning', 'DDL', 1, '检测到RENAME TABLE时提示确认应用引用、权限和上下游依赖是否已同步调整。该规则默认仅作风险提醒，不阻断工单。', 'RENAME TABLE users TO app_users;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (87, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_DDL_ALTER_COLUMN_CHANGE_RISK', '字段定义变更风险提醒', 'MODIFY/CHANGE COLUMN可能导致数据截断或不兼容，默认仅提醒风险、不阻断执行', 'warning', 'DDL', 1, '检测到MODIFY COLUMN或CHANGE COLUMN时提示确认类型范围、字符集和应用兼容性。该规则默认仅作风险提醒，不阻断工单。', 'ALTER TABLE users MODIFY COLUMN nickname VARCHAR(16) NOT NULL;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (88, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_DDL_CONVERT_CHARSET_RISK', '字符集转换风险提醒', '转换表字符集可能锁表或改变数据编码，默认仅提醒风险、不阻断执行', 'warning', 'DDL', 1, '检测到ALTER TABLE CONVERT TO CHARACTER SET时提示评估锁表时间、数据编码兼容性和回滚方案。该规则默认仅作风险提醒，不阻断工单。', 'ALTER TABLE users CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (89, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_DML_ON_DUPLICATE_UPDATE_RISK', 'ON DUPLICATE KEY UPDATE风险提醒', '该写法可能隐藏重复键冲突并更新已有数据，默认仅提醒风险、不阻断执行', 'warning', 'DML', 1, '检测到ON DUPLICATE KEY UPDATE时提示确认冲突键、更新字段及重复执行结果。该规则默认仅作风险提醒，不阻断工单。', 'INSERT INTO users(id, nickname) VALUES (1, \'alice\') ON DUPLICATE KEY UPDATE nickname = VALUES(nickname);', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (90, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_DML_SENSITIVE_LITERAL_WRITE_RISK', '敏感字段字面量写入风险提醒', '敏感字段写入字符串字面量时需确认已加密或哈希，默认仅提醒风险、不阻断执行', 'warning', 'DML', 1, '检测到INSERT向password、token、secret、api_key、email、phone等敏感字段写入字符串字面量时提示人工确认；该检测不能判断内容是否为明文，只要求确认已按业务要求加密或哈希。该规则默认不阻断工单。', 'INSERT INTO users(username, password) VALUES (\'alice\', \'$2b$12$hashed_value\');', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (91, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_DML_SENSITIVE_FUNCTION_UPDATE_RISK', '敏感字段函数更新风险提醒', '敏感字段使用函数表达式更新时需确认结果已正确加密或哈希，默认仅提醒风险、不阻断执行', 'warning', 'DML', 1, '检测到UPDATE通过函数或转换表达式更新password、token、secret、api_key、email、phone等敏感字段时提示人工确认；规则不判断结果一定是明文，只要求确认结果符合加密或哈希要求。该规则默认不阻断工单。', 'UPDATE users SET password = SHA2(\'new-password\', 256) WHERE id = 1;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (92, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_SQL_DISABLE_FOREIGN_KEY_CHECKS_RISK', '关闭外键检查风险提醒', '关闭外键检查可能写入不一致数据，默认仅提醒风险、不阻断执行', 'warning', 'common', 1, '检测到SET FOREIGN_KEY_CHECKS = 0时提示确认数据导入顺序、约束恢复和一致性校验方案。该规则默认仅作风险提醒，不阻断工单。', 'SET FOREIGN_KEY_CHECKS = 0;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (93, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_SQL_LOCK_TABLES_RISK', 'LOCK TABLES风险提醒', '显式锁表可能阻塞其他业务请求，默认仅提醒风险、不阻断执行', 'warning', 'common', 1, '检测到LOCK TABLE或LOCK TABLES时提示评估锁持有时间、并发影响及解锁安排。该规则默认仅作风险提醒，不阻断工单。', 'LOCK TABLES users WRITE;', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (94, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_SQL_PRIVILEGE_CHANGE_RISK', '权限变更风险提醒', 'GRANT/REVOKE会改变数据库访问权限，默认仅提醒风险、不阻断执行', 'warning', 'common', 1, '检测到GRANT或REVOKE时提示按最小权限原则确认授权对象、权限范围和回收方案。该规则默认仅作风险提醒，不阻断工单。', 'GRANT SELECT ON app.* TO \'report_user\'@\'%\';', NULL);
INSERT INTO `audit_rules` (`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `description`, `severity`, `type`, `enabled`, `explanation`, `example`, `config`) VALUES (95, '2026-05-13 00:00:00.000', '2026-05-13 00:00:00.000', NULL, 'NM_SQL_ROUTINE_OBJECT_RISK', '例程对象变更风险提醒', '触发器、存储过程、函数或事件可能引入隐式行为，默认仅提醒风险、不阻断执行', 'warning', 'common', 1, '检测到CREATE、ALTER或DROP触发器、存储过程、函数、事件时提示确认隐式副作用、权限和运行周期。该规则默认仅作风险提醒，不阻断工单。', 'CREATE EVENT cleanup_logs ON SCHEDULE EVERY 1 DAY DO DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL 30 DAY;', NULL);
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
  `pattern` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '匹配模式(正则或通配符)',
  `rule_type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '脱敏类型(mask_middle, mask_all, mask_left, mask_right)',
  `description` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '规则描述',
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
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '数据源名称',
  `type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '数据库类型(MySQL)',
  `host` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '主机地址',
  `port` bigint NOT NULL COMMENT '端口',
  `database` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '数据库名',
  `username` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户名',
  `password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '密码(加密)',
  `environment` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '生产' COMMENT '环境',
  `execution_timeout_seconds` int DEFAULT '30' COMMENT '执行超时时间(秒)',
  `query_timeout_seconds` int DEFAULT '30' COMMENT '查询超时时间(秒)',
  `connect_timeout` bigint DEFAULT '10' COMMENT '连接超时时间(秒)',
  `description` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述',
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'active' COMMENT '状态(active/inactive)',
  PRIMARY KEY (`id`),
  KEY `idx_data_sources_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of data_sources
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for feishu_config
-- ----------------------------
DROP TABLE IF EXISTS `feishu_config`;
CREATE TABLE `feishu_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `enabled` tinyint(1) NOT NULL DEFAULT '0',
  `app_id` varchar(100) NOT NULL DEFAULT '',
  `app_secret` varchar(255) NOT NULL DEFAULT '',
  `redirect_uri` varchar(500) NOT NULL DEFAULT '',
  `default_role` varchar(20) NOT NULL DEFAULT 'developer',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of feishu_config
-- ----------------------------
BEGIN;
INSERT INTO `feishu_config` (`id`, `enabled`, `app_id`, `app_secret`, `redirect_uri`, `default_role`, `created_at`, `updated_at`) VALUES (1, 0, 'cli_xxxxxxxxxxxxxxxx', 'your-app-secret', 'https://nextmeta.example.com/api/v1/auth/feishu/callback', 'readonly', '2026-08-07 14:19:45.798', '2026-08-08 22:04:12.729');
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
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of group_approvers
-- ----------------------------
BEGIN;
INSERT INTO `group_approvers` (`id`, `created_at`, `updated_at`, `deleted_at`, `group_id`, `user_id`) VALUES (6, '2026-08-08 21:31:03.073', '2026-08-08 21:31:03.073', NULL, 1, 1);
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
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '组名',
  `code` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '组编码',
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'enabled' COMMENT '状态(enabled/disabled)',
  `description` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述',
  `source` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'local' COMMENT '来源(local/ldap)',
  `dn` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'LDAP DN',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_groups_name` (`name`),
  KEY `idx_groups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of groups
-- ----------------------------
BEGIN;
INSERT INTO `groups` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `code`, `status`, `description`, `source`, `dn`) VALUES (1, '2025-12-03 17:52:57.616', '2026-08-07 20:28:51.955', NULL, 'NextMeta_Groups', 'NextMeta_Groups', 'enabled', '-', 'local', '');
COMMIT;

-- ----------------------------
-- Table structure for ldap_config
-- ----------------------------
DROP TABLE IF EXISTS `ldap_config`;
CREATE TABLE `ldap_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `enabled` tinyint(1) NOT NULL DEFAULT '0',
  `url` varchar(255) NOT NULL DEFAULT '',
  `base_dn` varchar(255) NOT NULL DEFAULT '',
  `group_base_dn` varchar(255) NOT NULL DEFAULT '',
  `bind_dn` varchar(255) NOT NULL DEFAULT '',
  `bind_pass` varchar(255) NOT NULL DEFAULT '',
  `user_filter` varchar(500) NOT NULL DEFAULT '(objectClass=person)',
  `group_filter` varchar(500) NOT NULL DEFAULT '(objectClass=*)',
  `mapping_username` varchar(100) NOT NULL DEFAULT 'uid',
  `mapping_real_name` varchar(100) NOT NULL DEFAULT 'cn',
  `mapping_email` varchar(100) NOT NULL DEFAULT 'mail',
  `sync_interval` int NOT NULL DEFAULT '30',
  `exclude_keywords` varchar(255) NOT NULL DEFAULT 'admin',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of ldap_config
-- ----------------------------
BEGIN;
INSERT INTO `ldap_config` (`id`, `enabled`, `url`, `base_dn`, `group_base_dn`, `bind_dn`, `bind_pass`, `user_filter`, `group_filter`, `mapping_username`, `mapping_real_name`, `mapping_email`, `sync_interval`, `exclude_keywords`, `created_at`, `updated_at`) VALUES (1, 0, 'ldap://127.0.0.1:3890', 'dc=example,dc=com', 'ou=groups,dc=example,dc=com', 'uid=admin,ou=people,dc=example,dc=com', 'your-password', '(objectClass=person)', '(objectClass=*)', 'uid', 'cn', 'mail', 30, 'admin,lldap', '2026-08-07 15:35:09.343', '2026-08-08 22:03:50.449');
COMMIT;

-- ----------------------------
-- Table structure for login_audits
-- ----------------------------
DROP TABLE IF EXISTS `login_audits`;
CREATE TABLE `login_audits` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL DEFAULT '0',
  `username` varchar(50) NOT NULL DEFAULT '' COMMENT '登录用户名',
  `login_method` varchar(20) NOT NULL DEFAULT '' COMMENT '登录方式(local/ldap/feishu)',
  `client_ip` varchar(45) NOT NULL DEFAULT '' COMMENT '客户端 IP',
  `user_agent` varchar(500) NOT NULL DEFAULT '' COMMENT '浏览器 User-Agent',
  `success` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否登录成功',
  `error_message` varchar(255) NOT NULL DEFAULT '' COMMENT '失败原因',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of login_audits
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for oauth_login_tickets
-- ----------------------------
DROP TABLE IF EXISTS `oauth_login_tickets`;
CREATE TABLE `oauth_login_tickets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ticket_hash` varchar(64) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `provider` varchar(20) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `consumed_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ticket_hash` (`ticket_hash`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of oauth_login_tickets
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for oauth_states
-- ----------------------------
DROP TABLE IF EXISTS `oauth_states`;
CREATE TABLE `oauth_states` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `state_hash` varchar(64) NOT NULL,
  `provider` varchar(20) NOT NULL,
  `purpose` varchar(20) NOT NULL,
  `redirect_uri` varchar(500) NOT NULL DEFAULT '',
  `client_ip` varchar(45) NOT NULL DEFAULT '',
  `expires_at` datetime(3) NOT NULL,
  `consumed_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_state_hash` (`state_hash`),
  KEY `idx_provider_purpose` (`provider`,`purpose`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of oauth_states
-- ----------------------------
BEGIN;
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
  `title` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '片段标题',
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'SQL内容',
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
  `title` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '工单标题',
  `sql_content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'SQL内容',
  `ticket_type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '工单类型:query/change',
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'pending' COMMENT '状态:pending/approved/rejected/executed/failed',
  `execute_result` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '执行结果',
  `executor_id` bigint unsigned DEFAULT '0' COMMENT '执行人ID',
  `executor_name` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '执行人名称',
  `executed_at` datetime(3) DEFAULT NULL COMMENT '执行时间',
  `affected_rows` bigint DEFAULT '0' COMMENT '影响行数',
  `execution_duration_ms` bigint DEFAULT '0' COMMENT '执行耗时毫秒',
  `is_force` tinyint(1) DEFAULT '0' COMMENT '是否强制提交',
  `database` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '目标数据库名',
  `approver_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '指定审核人ID',
  `statement_results` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '逐语句执行结果JSON',
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
  `key` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `value` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
  `description` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of system_settings
-- ----------------------------
BEGIN;
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('global_sql_limit', '1000', '全局 SQL 查询行数限制 (Global SQL Query Limit)', '2025-12-05 15:39:59.497', '2026-06-24 17:34:08.665');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('local_enabled', 'true', '', '2026-08-07 18:39:45.597', '2026-08-07 18:56:25.466');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_enabled', 'false', '是否启用系统通知 (Enable System Notification)', '2026-07-23 12:53:34.633', '2026-08-08 22:06:39.469');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_event_ticket_created', 'false', '工单创建通知开关 (Ticket Created Notification Switch)', '2026-07-23 12:53:34.555', '2026-08-08 22:06:39.436');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_event_ticket_executed', 'false', '工单执行成功通知开关 (Ticket Executed Notification Switch)', '2026-07-23 12:53:34.604', '2026-08-08 22:06:39.454');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_event_ticket_failed', 'false', '工单执行失败通知开关 (Ticket Failed Notification Switch)', '2026-07-23 12:53:34.617', '2026-08-08 22:06:39.459');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_event_ticket_rejected', 'false', '工单驳回通知开关 (Ticket Rejected Notification Switch)', '2026-07-23 12:53:34.595', '2026-08-08 22:06:39.448');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_template_ticket', '{STATUS}｜{TYPE}｜{DATABASE}\n\n工单：{TICKET_NO} - {TITLE}\n数据源：{DATASOURCE}\n操作人：{OPERATOR}\n时间：{OPERATION_TIME}\n\n处理说明：\n{REMARK}\n\n执行结果：\n{EXECUTE_RESULT}', '工单通用通知模板 (Ticket Notification Template)', '2026-07-23 12:53:34.627', '2026-08-08 22:06:39.464');
INSERT INTO `system_settings` (`key`, `value`, `description`, `created_at`, `updated_at`) VALUES ('notification_webhook_url', 'https://open.feishu.cn/open-apis/bot/v2/hook/*********************', '系统通知 Webhook 地址 (System Notification Webhook URL)', '2026-07-23 12:53:34.638', '2026-08-08 22:06:39.473');
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
  `action` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '操作:approve/reject',
  `comment` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '审批意见',
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
INSERT INTO `user_groups` (`id`, `created_at`, `updated_at`, `deleted_at`, `user_id`, `group_id`) VALUES (11, '2026-08-08 21:31:03.072', '2026-08-08 21:31:03.072', NULL, 1, 1);
COMMIT;

-- ----------------------------
-- Table structure for user_oauth_bindings
-- ----------------------------
DROP TABLE IF EXISTS `user_oauth_bindings`;
CREATE TABLE `user_oauth_bindings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `provider` varchar(20) NOT NULL,
  `provider_user_id` varchar(100) NOT NULL,
  `union_id` varchar(100) NOT NULL DEFAULT '',
  `open_id` varchar(100) NOT NULL DEFAULT '',
  `nickname` varchar(100) NOT NULL DEFAULT '',
  `avatar_url` varchar(500) NOT NULL DEFAULT '',
  `email` varchar(100) NOT NULL DEFAULT '',
  `raw_profile` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_provider_user` (`user_id`,`provider`),
  UNIQUE KEY `idx_provider_provider_user` (`provider`,`provider_user_id`),
  KEY `idx_union_id` (`union_id`),
  KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of user_oauth_bindings
-- ----------------------------
BEGIN;
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
  `username` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '用户名',
  `password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '加密密码',
  `real_name` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '真实姓名',
  `email` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '邮箱',
  `avatar_url` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '头像地址',
  `role` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'user' COMMENT '角色(admin/user)',
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'enabled' COMMENT '状态(enabled/disabled)',
  `source` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'local' COMMENT '来源(local/ldap)',
  `dn` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'LDAP DN',
  `last_login_at` datetime(3) DEFAULT NULL COMMENT '最后登录时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of users
-- ----------------------------
BEGIN;
INSERT INTO `users` (`id`, `created_at`, `updated_at`, `deleted_at`, `username`, `password`, `real_name`, `email`, `avatar_url`, `role`, `status`, `source`, `dn`, `last_login_at`) VALUES (1, '2025-12-03 14:38:26.235', '2026-08-07 21:25:06.877', NULL, 'NextMeta', '$2b$12$TXGL8rxggLW6y2uDk8xGCenIfpwEM28b9/md/8gpBPHiNzKQKL.ri', '超级管理员', 'admin@nextmeta.local', '', 'super_admin', 'enabled', 'local', '', NULL);
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
