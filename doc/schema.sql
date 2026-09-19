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
  `task_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'asynq待执行任务ID',
  `next_run` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务';

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
