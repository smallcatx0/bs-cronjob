package valid

import (
	"cron-job/internal/conf"
	"cron-job/middleware/resp"
	"cron-job/models/dao/rds"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// cronParser 与 asynq scheduler 保持一致: 标准 5 段(分 时 日 月 周), 支持 @every 等描述符, 不支持秒字段
var cronParser = cron.NewParser(cron.Minute | cron.Hour |
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

// validateAlarm 校验告警配置(空跳过): 解析 JSON 并按 type 分发, 预定义需命中配置机器人, 自定义需合法 webhook+secret
// 命中脱敏占位的自定义值放行(由 controller Update 跳过覆盖, 避免前端回传脱敏值覆盖真实配置)
func validateAlarm(s string) error {
	if s == "" {
		return nil
	}
	c, err := rds.ParseAlarm(s)
	if err != nil {
		return resp.ParamInValid("alarm 格式错误: " + err.Error())
	}
	switch c.Type {
	case rds.AlarmTypeDing:
		if c.Name != "" && c.Webhook == "" {
			if !conf.HasDingRobot(c.Name) {
				return resp.ParamInValid("alarm 机器人不存在: " + c.Name)
			}
			return nil
		}
		if c.Webhook != "" {
			if rds.IsAlarmMasked(c) {
				return nil // 脱敏占位, 保留原值
			}
			if !strings.HasPrefix(c.Webhook, "https://oapi.dingtalk.com/robot/send") {
				return resp.ParamInValid("alarm webhook 非法")
			}
			if c.Secret == "" {
				return resp.ParamInValid("alarm secret 必填")
			}
			return nil
		}
		return resp.ParamInValid("alarm 需配置 name 或 webhook")
	default:
		return resp.ParamInValid("alarm type 不支持: " + c.Type)
	}
}

type JobQuery struct {
	Name         string `form:"name" binding:"omitempty,max=128"`
	Type         string `form:"type" binding:"omitempty,oneof=http shell gofunc"`
	Status       *int   `form:"status" binding:"omitempty,oneof=0 1 2"`
	ScheduleType string `form:"schedule_type" binding:"omitempty,oneof=once cron"`
}

type JobAdd struct {
	Name         string `json:"name" binding:"required,max=128"`
	Description  string `json:"description" binding:"omitempty,max=255"`
	Type         string `json:"type" binding:"required,oneof=http shell gofunc"`
	ScheduleType string `json:"schedule_type" binding:"required,oneof=once cron"`
	CronExpr     string `json:"cron_expr" binding:"omitempty,max=128"`
	ExecuteAt    string `json:"execute_at" binding:"omitempty,datetime=2006-01-02 15:04:05"`
	Payload      string `json:"payload" binding:"omitempty"`
	TimeoutSec   int    `json:"timeout_sec" binding:"omitempty,min=1"`
	Alarm        string `json:"alarm" binding:"omitempty,max=512"`
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
	return validateAlarm(p.Alarm)
}

type JobUpdate struct {
	ID          int64  `json:"id" binding:"required"`
	Description string `json:"description" binding:"omitempty,max=255"`
	CronExpr    string `json:"cron_expr" binding:"omitempty,max=128"`
	ExecuteAt   string `json:"execute_at" binding:"omitempty,datetime=2006-01-02 15:04:05"`
	Payload     string `json:"payload" binding:"omitempty"`
	TimeoutSec  int    `json:"timeout_sec" binding:"omitempty,min=1"`
	Alarm       string `json:"alarm" binding:"omitempty,max=512"`
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
	return validateAlarm(p.Alarm)
}

type JobLogQuery struct {
	JobID       int64  `form:"job_id"`
	JobName     string `form:"job_name" binding:"omitempty,max=128"`
	Status      string `form:"status" binding:"omitempty,oneof=running success failed"`
	TriggerType string `form:"trigger_type" binding:"omitempty,oneof=cron once manual"`
}
