CREATE TABLE `bs_job` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(128) NOT NULL COMMENT '任务名称',
  `type` varchar(32) NOT NULL COMMENT '任务类型：http / shell / gofunc',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '状态：0=停用, 1=启用',
  `schedule_type` varchar(32) NOT NULL COMMENT '调度类型：once / interval / cron',
  `cron_expr` varchar(128) DEFAULT NULL COMMENT 'Cron表达式',
  `execute_at` datetime DEFAULT NULL COMMENT '指定执行时间',
  `payload` text COMMENT 'JSON配置',
  `timeout_sec` int NOT NULL DEFAULT 300 COMMENT '超时秒数',
  `next_run` datetime DEFAULT NULL COMMENT '下次执行时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`),
  KEY `idx_next_run` (`next_run`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='定时任务表';

CREATE TABLE `bs_job_log` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `job_id` bigint NOT NULL COMMENT '任务ID',
  `job_name` varchar(128) DEFAULT NULL COMMENT '任务名称冗余',
  `trigger_type` varchar(32) DEFAULT NULL COMMENT '触发类型：cron / manual',
  `status` varchar(32) DEFAULT NULL COMMENT '执行状态：running / success / failed',
  `output` text COMMENT '标准输出',
  `error` text COMMENT '错误信息',
  `started_at` datetime DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime DEFAULT NULL COMMENT '结束时间',
  PRIMARY KEY (`id`),
  KEY `idx_job_id` (`job_id`),
  KEY `idx_started_at` (`started_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='任务执行日志表';