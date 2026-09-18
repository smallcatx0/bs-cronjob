package valid

import (
	"cron-job/middleware/resp"
	"cron-job/models/dao/rds"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// cronParser 与 asynq scheduler 一致: 秒可选, 支持 quartz 风格 `?`
var cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour |
	cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// CheckCronExpr 校验 cron 表达式
func CheckCronExpr(expr string) error {
	if expr == "" {
		return fmt.Errorf("cron_expr 不能为空")
	}
	_, err := cronParser.Parse(expr)
	if err != nil {
		return fmt.Errorf("cron_expr 无效: %w", err)
	}
	return nil
}

type JobQuery struct {
	Name         string `form:"name" binding:"omitempty,max=128"`
	Type         string `form:"type" binding:"omitempty,oneof=http shell gofunc"`
	Status       *int   `form:"status" binding:"omitempty,oneof=0 1 2"`
	ScheduleType string `form:"schedule_type" binding:"omitempty,oneof=once cron"`
}

type JobAdd struct {
	Name         string `json:"name" binding:"required,max=128"`
	Type         string `json:"type" binding:"required,oneof=http shell gofunc"`
	ScheduleType string `json:"schedule_type" binding:"required,oneof=once cron"`
	CronExpr     string `json:"cron_expr" binding:"omitempty,max=128"`
	ExecuteAt    string `json:"execute_at" binding:"omitempty,datetime=2006-01-02 15:04:05"`
	Payload      string `json:"payload" binding:"omitempty"`
	TimeoutSec   int    `json:"timeout_sec" binding:"omitempty,min=1"`
	// ---
	ExecuteAtTime *time.Time `json:"-"` // 仅用于 once 任务, 解析 execute_at 后的时间
}

func (p *JobAdd) Valid() error {
	switch p.ScheduleType {
	case rds.SchedCron:
		if err := CheckCronExpr(p.CronExpr); err != nil {
			return resp.ParamInValid(err.Error())
		}
	case rds.SchedOnce:
		if p.ExecuteAt == "" {
			return resp.ParamInValid("once 任务 execute_at 必填")
		}
		executeAt, err := time.ParseInLocation("2006-01-02 15:04:05", p.ExecuteAt, time.Local)
		if err != nil {
			return resp.ParamInValid("execute_at 格式错误")
		}
		p.ExecuteAtTime = &executeAt
	}
	return nil
}

type JobUpdate struct {
	ID         int64  `json:"id" binding:"required"`
	Name       string `json:"name" binding:"omitempty,max=128"`
	CronExpr   string `json:"cron_expr" binding:"omitempty,max=128"`
	ExecuteAt  string `json:"execute_at" binding:"omitempty,datetime=2006-01-02 15:04:05"`
	Payload    string `json:"payload" binding:"omitempty"`
	TimeoutSec int    `json:"timeout_sec" binding:"omitempty,min=1"`
	// ---
	ExecuteAtTime *time.Time `json:"-"` // 仅用于 once 任务, 解析 execute_at 后的时间
}

func (p *JobUpdate) Valid() error {
	if p.CronExpr != "" {
		if err := CheckCronExpr(p.CronExpr); err != nil {
			return resp.ParamInValid(err.Error())
		}
	}
	if p.ExecuteAt != "" {
		executeAt, err := time.ParseInLocation("2006-01-02 15:04:05", p.ExecuteAt, time.Local)
		if err != nil {
			return resp.ParamInValid("execute_at 格式错误")
		}
		p.ExecuteAtTime = &executeAt
	}
	return nil
}

type JobLogQuery struct {
	JobID       int64  `form:"job_id"`
	JobName     string `form:"job_name" binding:"omitempty,max=128"`
	Status      string `form:"status" binding:"omitempty,oneof=running success failed"`
	TriggerType string `form:"trigger_type" binding:"omitempty,oneof=cron once manual"`
}
