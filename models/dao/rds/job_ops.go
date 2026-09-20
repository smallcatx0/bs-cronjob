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
func SetTaskInfo(id int64, nextRun *time.Time) error {
	return dao.MysqlCli.Model(&Job{}).Where("id = ?", id).
		Updates(map[string]interface{}{"next_run": nextRun}).Error
}

// ClearTaskInfo 清除待执行任务信息
func ClearTaskInfo(id int64) error {
	return dao.MysqlCli.Model(&Job{}).Where("id = ?", id).
		Updates(map[string]interface{}{"task_id": "", "next_run": nil}).Error
}

// SetNextRun 更新周期任务下次执行时间
func SetNextRun(id int64, nextRun *time.Time) error {
	return dao.MysqlCli.Model(&Job{}).Where("id = ?", id).
		Update("next_run", nextRun).Error
}

// LastRunBrief 任务最近一次执行摘要
type LastRunBrief struct {
	JobID      int64      `gorm:"column:job_id" json:"job_id"`
	Status     string     `gorm:"column:status" json:"status"` // running / success / failed
	StartedAt  *time.Time `gorm:"column:started_at" json:"started_at"`
	FinishedAt *time.Time `gorm:"column:finished_at" json:"finished_at"`
}

// LastRunBriefs 批量查询各任务最近一次执行摘要, 以 job_id 为 key 返回
func LastRunBriefs(jobIDs []int64) (map[int64]LastRunBrief, error) {
	out := make(map[int64]LastRunBrief, len(jobIDs))
	if len(jobIDs) == 0 {
		return out, nil
	}
	// 1. 取每个任务最近一条日志的 id (日志按启动顺序自增)
	type maxID struct {
		JobID int64 `gorm:"column:job_id"`
		ID    int64 `gorm:"column:id"`
	}
	var maxIDs []maxID
	err := dao.MysqlCli.Model(&JobLog{}).
		Select("job_id, MAX(id) AS id").
		Where("job_id IN ?", jobIDs).
		Group("job_id").
		Find(&maxIDs).Error
	if err != nil || len(maxIDs) == 0 {
		return out, err
	}
	// 2. 按 id 集合回查详情
	ids := make([]int64, 0, len(maxIDs))
	for _, m := range maxIDs {
		ids = append(ids, m.ID)
	}
	var briefs []LastRunBrief
	err = dao.MysqlCli.Model(&JobLog{}).
		Select("job_id, status, started_at, finished_at").
		Where("id IN ?", ids).
		Find(&briefs).Error
	if err != nil {
		return out, err
	}
	for _, b := range briefs {
		out[b.JobID] = b
	}
	return out, nil
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
