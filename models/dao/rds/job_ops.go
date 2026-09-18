package rds

import (
	"time"

	"cron-job/models/dao"
)

// GetJob 按ID查询任务
func GetJob(id int64) (*Job, error) {
	var job Job
	err := dao.MysqlCli.First(&job, id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// EnabledCronJobs 所有启用的周期任务(供调度器注册)
func EnabledCronJobs() ([]Job, error) {
	var jobs []Job
	err := dao.MysqlCli.
		Where("status = ? AND schedule_type = ?", StatusOn, SchedCron).
		Find(&jobs).Error
	return jobs, err
}

// SetStatus 更新任务状态
func SetStatus(id int64, status int) error {
	return dao.MysqlCli.Model(&Job{}).Where("id = ?", id).
		Update("status", status).Error
}

// SetTaskInfo 记录 asynq 待执行任务ID与下次运行时间(once任务)
func SetTaskInfo(id int64, taskID string, nextRun *time.Time) error {
	return dao.MysqlCli.Model(&Job{}).Where("id = ?", id).
		Updates(map[string]interface{}{"task_id": taskID, "next_run": nextRun}).Error
}

// ClearTaskInfo 清除待执行任务信息
func ClearTaskInfo(id int64) error {
	return dao.MysqlCli.Model(&Job{}).Where("id = ?", id).
		Updates(map[string]interface{}{"task_id": "", "next_run": nil}).Error
}

// StartJobLog 任务开始执行,写 running 日志
func StartJobLog(job *Job, triggerType string) *JobLog {
	now := time.Now()
	jl := &JobLog{
		JobID:       job.ID,
		JobName:     job.Name,
		TriggerType: triggerType,
		Status:      LogRunning,
		StartedAt:   &now,
	}
	dao.MysqlCli.Create(jl)
	return jl
}

// FinishJobLog 任务执行结束,回写结果
func FinishJobLog(jl *JobLog, output string, runErr error) {
	now := time.Now()
	jl.FinishedAt = &now
	jl.Output = truncate(output, 8192)
	if runErr != nil {
		jl.Status = LogFailed
		jl.Error = truncate(runErr.Error(), 2048)
	} else {
		jl.Status = LogSuccess
	}
	dao.MysqlCli.Save(jl)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
