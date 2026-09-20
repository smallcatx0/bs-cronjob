package v1

import (
	"time"

	"cron-job/internal/tasks"
	"cron-job/middleware/resp"
	"cron-job/models/dao"
	"cron-job/models/dao/rds"
	"cron-job/models/valid"
	"cron-job/pkg/glog"

	"github.com/gin-gonic/gin"
)

type Jobs struct{}

// List 列出所有 定时任务
func (Jobs) List(c *gin.Context) {
	p := valid.JobQuery{}
	err := valid.BindQueryAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	q := dao.MysqlCli.Model(&rds.Job{})
	// 查询条件
	if p.Name != "" {
		q = q.Where("name LIKE ?", "%"+p.Name+"%")
	}
	if p.Type != "" {
		q = q.Where("type = ?", p.Type)
	}
	if p.Status != nil {
		q = q.Where("status = ?", *p.Status)
	}
	if p.ScheduleType != "" {
		q = q.Where("schedule_type = ?", p.ScheduleType)
	}
	q = q.Order("id DESC")

	pg := resp.NewPage(c)
	q, err = pg.Paginate(q)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if pg.Total == 0 {
		resp.Paginate(c, pg, nil)
		return
	}
	jobs := make([]rds.Job, 0, pg.Limit)
	err = q.Find(&jobs).Error
	if err != nil {
		resp.Fail(c, err)
		return
	}

	// 附带每个任务最近一次执行摘要(辅助信息, 查询失败不阻断列表)
	jobIDs := make([]int64, 0, len(jobs))
	for _, j := range jobs {
		jobIDs = append(jobIDs, j.ID)
	}
	lastLogs, lerr := rds.LastRunBriefs(jobIDs)
	if lerr != nil {
		glog.Error("LastRunBriefs", lerr.Error())
		lastLogs = nil
	}
	type row struct {
		rds.Job
		LastLog *rds.LastRunBrief `json:"last_log"`
	}
	out := make([]row, 0, len(jobs))
	for _, j := range jobs {
		r := row{Job: j}
		if lb, ok := lastLogs[j.ID]; ok {
			brief := lb
			r.LastLog = &brief
		}
		out = append(out, r)
	}
	resp.Paginate(c, pg, out)
}

// Add 新建任务(默认停用, 待第一次测试后才可启用)
func (Jobs) Add(c *gin.Context) {
	p := valid.JobAdd{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	job := rds.Job{
		Name:         p.Name,
		Description:  p.Description,
		Type:         p.Type,
		Status:       rds.StatusOff,
		ScheduleType: p.ScheduleType,
		CronExpr:     p.CronExpr,
		Payload:      p.Payload,
		TimeoutSec:   p.TimeoutSec,
		CreatedAt:    time.Now(),
	}
	if job.TimeoutSec <= 0 {
		job.TimeoutSec = 300
	}
	if p.ScheduleType == rds.SchedOnce {
		job.ExecuteAt = p.ExecuteAtTime
	}

	err = dao.MysqlCli.Debug().Create(&job).Error
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Succ(c, job)
}

// Detail 任务详情
func (Jobs) Detail(c *gin.Context) {
	p := struct {
		ID int64 `form:"id" binding:"required"`
	}{}
	err := valid.BindQueryAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	job, err := rds.GetJob(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Succ(c, job)
}

// Update 更新任务(仅停用状态可更新, 只更新提交的字段)
func (Jobs) Update(c *gin.Context) {
	p := valid.JobUpdate{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	job, err := rds.GetJob(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if job.Status == rds.StatusOn {
		resp.Fail(c, resp.ParamInValid("仅有停用的任务才可更新"))
		return
	}

	updates := map[string]interface{}{}
	if p.Description != "" {
		updates["description"] = p.Description
	}
	if p.Payload != "" {
		updates["payload"] = p.Payload
	}
	if p.TimeoutSec > 0 {
		updates["timeout_sec"] = p.TimeoutSec
	}
	if p.CronExpr != "" {
		if job.ScheduleType != rds.SchedCron {
			resp.Fail(c, resp.ParamInValid("非 cron 任务不可设置 cron_expr"))
			return
		}
		updates["cron_expr"] = p.CronExpr
	}
	if p.ExecuteAt != "" {
		if job.ScheduleType != rds.SchedOnce {
			resp.Fail(c, resp.ParamInValid("非 once 任务不可设置 execute_at"))
			return
		}
		updates["execute_at"] = p.ExecuteAtTime
	}

	if len(updates) > 0 {
		err = dao.MysqlCli.Model(&rds.Job{}).Where("id = ?", job.ID).Updates(updates).Error
		if err != nil {
			resp.Fail(c, err)
			return
		}
	}
	job, _ = rds.GetJob(job.ID)
	resp.Succ(c, job)
}

// Delete 删除任务(启用中不可删除)
func (Jobs) Delete(c *gin.Context) {
	p := struct {
		ID int64 `json:"id" binding:"required"`
	}{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	job, err := rds.GetJob(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if job.Status == rds.StatusOn {
		resp.Fail(c, resp.ParamInValid("启用中的任务不可删除, 请先停用"))
		return
	}
	// 撤销可能存在的待执行 asynq 任务
	tasks.DeletePendingOnce(job)
	err = dao.MysqlCli.Delete(job).Error
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Succ(c, job)
}

// Run 手动立即执行一次(不影响任务状态)
func (Jobs) Run(c *gin.Context) {
	p := struct {
		ID int64 `json:"id" binding:"required"`
	}{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	job, err := rds.GetJob(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	err = tasks.EnqueueManual(job)
	if err != nil {
		resp.Fail(c, resp.ParamInValid("任务入队失败", err.Error()))
		return
	}
	resp.Succ(c, job)
}

// Toggle 启用/停用任务
func (Jobs) Toggle(c *gin.Context) {
	p := struct {
		ID     int64 `json:"id" binding:"required"`
		Status int   `json:"status" binding:"oneof=0 1"` // 1=启用 0=停用
	}{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	job, err := rds.GetJob(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if job.Status == p.Status {
		resp.Succ(c, job)
		return
	}

	if p.Status == rds.StatusOn { // 启用
		switch job.ScheduleType {
		case rds.SchedCron:
			if err = valid.CheckCronExpr(job.CronExpr); err != nil {
				resp.Fail(c, resp.ParamInValid(err.Error()))
				return
			}
			if err = tasks.RegisterCron(job); err != nil {
				resp.Fail(c, resp.ParamInValid("任务注册失败", err.Error()))
				return
			}
			if err = rds.SetStatus(job.ID, rds.StatusOn); err != nil {
				tasks.UnregisterCron(job.ID) // 落库失败回滚注册
				resp.Fail(c, err)
				return
			}
			// 启用时计算下次执行时间并写入 next_run
			if next, cerr := tasks.NextCronRun(job.CronExpr, time.Now()); cerr == nil {
				_ = rds.SetNextRun(job.ID, &next)
			}
		case rds.SchedOnce:
			if job.ExecuteAt == nil || !job.ExecuteAt.After(time.Now()) {
				resp.Fail(c, resp.ParamInValid("执行时间已过期, 无法启用"))
				return
			}
			if err = tasks.EnqueueOnce(job); err != nil {
				resp.Fail(c, resp.ParamInValid("任务入队失败", err.Error()))
				return
			}
			if err = rds.SetStatus(job.ID, rds.StatusOn); err != nil {
				resp.Fail(c, err)
				return
			}
		}
	} else { // 停用
		if err = rds.SetStatus(job.ID, rds.StatusOff); err != nil {
			resp.Fail(c, err)
			return
		}
		switch job.ScheduleType {
		case rds.SchedCron:
			tasks.UnregisterCron(job.ID)
		case rds.SchedOnce:
			tasks.DeletePendingOnce(job)
		}
		// 停用时清除下次执行时间
		_ = rds.SetNextRun(job.ID, nil)
	}
	job, _ = rds.GetJob(job.ID)
	resp.Succ(c, job)
}

// Log 任务运行记录
func (Jobs) Log(c *gin.Context) {
	p := valid.JobLogQuery{}
	err := valid.BindQueryAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	q := dao.MysqlCli.Model(&rds.JobLog{})
	if p.JobID > 0 {
		q = q.Where("job_id = ?", p.JobID)
	}
	if p.JobName != "" {
		q = q.Where("job_name LIKE ?", "%"+p.JobName+"%")
	}
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	if p.TriggerType != "" {
		q = q.Where("trigger_type = ?", p.TriggerType)
	}
	q = q.Order("started_at DESC, id DESC")

	pg := resp.NewPage(c)
	q, err = pg.Paginate(q)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if pg.Total == 0 {
		resp.Paginate(c, pg, nil)
		return
	}
	logs := make([]rds.JobLog, 0, pg.Limit)
	err = q.Find(&logs).Error
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Paginate(c, pg, logs)
}

// GoFuncs 已注册的 go func 任务名(供前端下拉)
func (Jobs) GoFuncs(c *gin.Context) {
	resp.Succ(c, tasks.RegisteredFuncs())
}
