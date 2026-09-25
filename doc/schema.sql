-- 定时任务管理平台 表结构
CREATE TABLE IF NOT EXISTS `bs_job` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL COMMENT '任务名',
  `description` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '任务描述',
  `type` VARCHAR(32) NOT NULL COMMENT 'http/shell/gofunc',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0=停用 1=启用 2=已过期',
  `schedule_type` VARCHAR(32) NOT NULL COMMENT 'once/cron',
  `cron_expr` VARCHAR(128) NOT NULL DEFAULT '',
  `execute_at` DATETIME NULL COMMENT 'once任务执行时间',
  `payload` TEXT COMMENT 'JSON配置',
  `timeout_sec` INT NOT NULL DEFAULT 300,
  `alarm` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '告警配置JSON: 空=关闭; {"type":"ding_alarm","name":..}预定义 / {"type":"ding_alarm","webhook":..,"secret":..}自定义',
  `task_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'asynq待执行任务ID',
  `next_run` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务';

-- 任务运行记录表
CREATE TABLE IF NOT EXISTS `bs_job_log` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `job_id` BIGINT NOT NULL,
  `job_name` VARCHAR(128) NOT NULL DEFAULT '',
  `trigger_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'cron/once/manual',
  `status` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'running/success/failed',
  `output` TEXT,
  `error` TEXT,
  `started_at` DATETIME NULL,
  `finished_at` DATETIME NULL,
  PRIMARY KEY (`id`),
  KEY `idx_job_id` (`job_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务运行记录';

-- Retry 策略配置表
CREATE TABLE IF NOT EXISTS `bs_tabledata_retry` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `unkey` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '任务唯一名',
  `dsn` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '数据库链接',
  `db_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '数据库名',
  `table_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '表名',
  `column_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '依据字段名',
  `column_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '依据字段类型 unix/timestamp/datetime',
  `find_wh` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '查找条件',
  `set_fields` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '更新的字段',
  `before` BIGINT NOT NULL DEFAULT 0 COMMENT '从当前时间之前多少秒',
  `duration` BIGINT NOT NULL DEFAULT 0 COMMENT '时间间隔',
  `limit` BIGINT NOT NULL DEFAULT 0 COMMENT '一次执行条数',
  `spec` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'cron表达式',
  `status` VARCHAR(16) NOT NULL DEFAULT 'offline' COMMENT '状态 offline/online, 仅online参与调度',
  `desc` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '描述',
  PRIMARY KEY (`id`),
  KEY `idx_unkey` (`unkey`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Retry策略配置';

-- TTL 策略配置表
CREATE TABLE IF NOT EXISTS `bs_tabledata_ttl` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `unkey` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '策略唯一key',
  `dsn` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '数据库链接',
  `db_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '数据库名',
  `table_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '表名',
  `column_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '依据字段名',
  `column_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '依据字段类型 unix/timestamp/datetime',
  `find_wh` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '附加筛选条件(可选)',
  `ttl_value` BIGINT NOT NULL DEFAULT 0 COMMENT 'TTL过期时间',
  `limit` BIGINT NOT NULL DEFAULT 0 COMMENT '一次执行条数',
  `spec` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'cron表达式',
  `status` VARCHAR(16) NOT NULL DEFAULT 'offline' COMMENT '状态 offline/online, 仅online参与调度',
  `desc` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '描述',
  PRIMARY KEY (`id`),
  KEY `idx_unkey` (`unkey`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='TTL策略配置';

-- TTL/Retry 策略执行日志表
CREATE TABLE IF NOT EXISTS `bs_dbstrategy_log` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `kind` VARCHAR(16) NOT NULL DEFAULT '' COMMENT '策略类型 ttl/retry',
  `strategy_id` BIGINT NOT NULL DEFAULT 0 COMMENT '策略配置行主键',
  `strategy_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '策略唯一名 unkey',
  `status` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'running/success/failed',
  `output` TEXT,
  `error` TEXT,
  `started_at` DATETIME NULL,
  `finished_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_kind` (`kind`),
  KEY `idx_strategy_id` (`strategy_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='TTL/Retry策略执行日志';
