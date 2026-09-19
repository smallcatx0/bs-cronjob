package rds

import "time"

// 任务状态
const (
	StatusOff     = 0 // 停用
	StatusOn      = 1 // 启用
	StatusExpired = 2 // 已过期(一次性任务执行完成后)
)

// 调度类型
const (
	SchedOnce = "once" // 一次性任务 At
	SchedCron = "cron" // 周期任务
)

// 任务类型
const (
	TypeHTTP   = "http"
	TypeShell  = "shell"
	TypeGoFunc = "gofunc"
)

// JobLog 状态
const (
	LogRunning = "running"
	LogSuccess = "success"
	LogFailed  = "failed"
)

// 触发类型
const (
	TriggerCron   = "cron"
	TriggerOnce   = "once"
	TriggerManual = "manual"
)

type Job struct {
	ID           int64      `gorm:"primaryKey; column:id" json:"id"`
	Name         string     `gorm:"size:128;not null; column:name" json:"name"`
	Type         string     `gorm:"size:32;not null; column:type" json:"type"`                   // http / shell / gofunc
	Status       int        `gorm:"default:0; column:status" json:"status"`                      // 0=停用, 1=启用, 2=已过期
	ScheduleType string     `gorm:"size:32;not null; column:schedule_type" json:"schedule_type"` // once / cron
	CronExpr     string     `gorm:"size:128; column:cron_expr" json:"cron_expr"`
	ExecuteAt    *time.Time `gorm:"column:execute_at" json:"execute_at"`
	Payload      string     `gorm:"type:text; column:payload" json:"payload"` // JSON 配置
	TimeoutSec   int        `gorm:"default:300; column:timeout_sec" json:"timeout_sec"`
	NextRun      *time.Time `gorm:"column:next_run" json:"next_run"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (Job) TableName() string { return "bs_job" }

type JobLog struct {
	ID          int64      `gorm:"primaryKey; column:id" json:"id"`
	JobID       int64      `gorm:"index; column:job_id" json:"job_id"`
	JobName     string     `gorm:"size:128; column:job_name" json:"job_name"`
	TriggerType string     `gorm:"size:32; column:trigger_type" json:"trigger_type"` // cron / once / manual
	Status      string     `gorm:"size:32; column:status" json:"status"`             // running / success / failed
	Output      string     `gorm:"type:text; column:output" json:"output"`
	Error       string     `gorm:"type:text; column:error" json:"error"`
	StartedAt   *time.Time `gorm:"column:started_at" json:"started_at"`
	FinishedAt  *time.Time `gorm:"column:finished_at" json:"finished_at"`
}

func (JobLog) TableName() string { return "bs_job_log" }
