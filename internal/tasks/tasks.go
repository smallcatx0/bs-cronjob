// Package tasks 基于 asynq 的任务调度与执行。
// 所有调度都走 asynq:
//   - 周期任务: asynq.Scheduler 注册 cronspec,任务变更时重建 Scheduler
//   - 一次性任务: client.Enqueue + ProcessAt(execute_at),固定 TaskID 便于撤销
//   - 手动运行: 立即 Enqueue
//
// 执行统一走 job:exec handler,按 http/shell/gofunc 分发。
package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"cron-job/internal/conf"
	"cron-job/models/dao/rds"
	"cron-job/pkg/glog"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// TypeJobExec asynq 任务类型
const TypeJobExec = "job:exec"

// cronParser 与调度器/校验保持一致: 标准 5 段(分 时 日 月 周), 支持 @every 等描述符, 不支持秒字段
var cronParser = cron.NewParser(cron.Minute | cron.Hour |
	cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// NextCronRun 计算 cron 表达式在 from 之后的下一次执行时间
func NextCronRun(expr string, from time.Time) (time.Time, error) {
	sch, err := cronParser.Parse(expr)
	if err != nil {
		return time.Time{}, err
	}
	return sch.Next(from), nil
}

// Payload asynq 任务负载
type Payload struct {
	JobID       int64  `json:"job_id"`
	TriggerType string `json:"trigger_type"` // cron / once / manual
}

var (
	Client *asynq.Client

	server *asynq.Server
	sched  *asynq.Scheduler
	mu     sync.Mutex

	// cronEntries 记录已注册周期任务的 asynq entryID(jobID -> entryID)。
	// 仅在当前调度器实例生命周期内有效, 进程重启后由 InitScheduler 全量重建。
	cronEntries = map[int64]string{}
)

// Queue 队列名
func Queue() string {
	q := conf.AppConf.GetString("asynq.queue")
	if q == "" {
		q = "default"
	}
	return q
}

func redisOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     conf.AppConf.GetString("redis.addr"),
		DB:       conf.AppConf.GetInt("redis.db"),
		Password: conf.AppConf.GetString("redis.pwd"),
	}
}

// InitClient 初始化 asynq client(入队端: 手动/一次性任务入队)
func InitClient() {
	Client = asynq.NewClient(redisOpt())
}

// InitScheduler 启动调度器(生产端: 按 cronspec 触发周期任务入队)。
// HTTP 服务侧使用,启动时按 DB 中启用的 cron 任务重建 Scheduler。
func InitScheduler() {
	ReloadScheduler()
	glog.Z().Info("[tasks] asynq scheduler started, queue=" + Queue())
}

// ConsumerClient 启动消费端(asynq server),监听队列并执行 job:exec。
// 独立消费程序使用,非阻塞(内部 goroutine 运行)。
func ConsumerClient() {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeJobExec, HandleJobExec)

	concurrency := conf.AppConf.GetInt("asynq.concurrency")
	if concurrency <= 0 {
		concurrency = 10
	}
	server = asynq.NewServer(redisOpt(), asynq.Config{
		Concurrency: concurrency,
		Logger:      &zapLogger{s: glog.Z().Sugar()},
	})
	go func() {
		if err := server.Run(mux); err != nil {
			glog.Z().Error("[tasks] asynq server exit: " + err.Error())
		}
	}()
	glog.Z().Info("[tasks] asynq consumer started, queue=" + Queue())
}

// Shutdown 优雅退出
func Shutdown() {
	mu.Lock()
	if sched != nil {
		sched.Shutdown()
		sched = nil
	}
	cronEntries = map[int64]string{}
	mu.Unlock()
	if server != nil {
		server.Shutdown()
	}
	if Client != nil {
		Client.Close()
	}
}

// ReloadScheduler 依据DB全量重建 asynq Scheduler(异步)。
// 仅在进程启动(InitScheduler)时调用一次; 运行期任务启停请使用 RegisterCron / UnregisterCron 增量同步。
func ReloadScheduler() {
	go func() {
		mu.Lock()
		defer mu.Unlock()
		if sched != nil {
			sched.Shutdown()
			sched = nil
		}
		s := asynq.NewScheduler(redisOpt(), &asynq.SchedulerOpts{
			Location: time.Local,
			Logger:   &zapLogger{s: glog.Z().Sugar()},
		})
		jobs, err := rds.EnabledCronJobs()
		if err != nil {
			glog.Z().Error("[tasks] load cron jobs fail: " + err.Error())
			return
		}
		entries := make(map[int64]string, len(jobs))
		for i := range jobs {
			j := jobs[i]
			task := newTask(j.ID, rds.TriggerCron)
			if eid, err := s.Register(j.CronExpr, task,
				asynq.Queue(Queue()), asynq.MaxRetry(0)); err != nil {
				glog.Z().Error(fmt.Sprintf("[tasks] register cron job fail, id=%d name=%s expr=%s err=%v",
					j.ID, j.Name, j.CronExpr, err))
			} else {
				entries[j.ID] = eid
			}
		}
		go func() {
			if err := s.Run(); err != nil {
				glog.Z().Error("[tasks] scheduler exit: " + err.Error())
			}
		}()
		sched = s
		cronEntries = entries
		glog.Z().Info(fmt.Sprintf("[tasks] scheduler reloaded, cron jobs=%d", len(jobs)))
	}()
}

// RegisterCron 增量注册(或更新)单个周期任务到运行中的调度器, 避免全量重建。
// 任务启用后调用。调度器未就绪时安全跳过, 由下次启动全量加载兜底。
func RegisterCron(job *rds.Job) error {
	mu.Lock()
	defer mu.Unlock()
	if sched == nil {
		glog.Z().Warn(fmt.Sprintf("[tasks] scheduler not ready, skip register, id=%d", job.ID))
		return nil
	}
	// 表达式等可能变更, 先注销历史注册项
	if eid, ok := cronEntries[job.ID]; ok {
		if err := sched.Unregister(eid); err != nil {
			glog.Z().Error(fmt.Sprintf("[tasks] unregister stale cron entry fail, id=%d err=%v", job.ID, err))
		}
		delete(cronEntries, job.ID)
	}
	eid, err := sched.Register(job.CronExpr, newTask(job.ID, rds.TriggerCron),
		asynq.Queue(Queue()), asynq.MaxRetry(0))
	if err != nil {
		return err
	}
	cronEntries[job.ID] = eid
	return nil
}

// UnregisterCron 从运行中的调度器增量注销单个周期任务。
// 任务停用/删除后调用。
func UnregisterCron(jobID int64) {
	mu.Lock()
	defer mu.Unlock()
	eid, ok := cronEntries[jobID]
	if !ok {
		return
	}
	if sched != nil {
		if err := sched.Unregister(eid); err != nil {
			glog.Z().Error(fmt.Sprintf("[tasks] unregister cron entry fail, id=%d err=%v", jobID, err))
		}
	}
	delete(cronEntries, jobID)
}

// OnceTaskID 一次性任务的固定 asynq TaskID(用于撤销/幂等)
func OnceTaskID(jobID int64) string {
	return fmt.Sprintf("job:once:%d", jobID)
}

// EnqueueOnce 入队一次性任务,在 execute_at 时刻执行
func EnqueueOnce(job *rds.Job) error {
	if job.ExecuteAt == nil {
		return errors.New("execute_at 为空")
	}
	taskID := OnceTaskID(job.ID)
	info, err := Client.EnqueueContext(context.Background(), newTask(job.ID, rds.TriggerOnce),
		asynq.TaskID(taskID),
		asynq.ProcessAt(*job.ExecuteAt),
		asynq.Queue(Queue()),
		asynq.MaxRetry(0),
	)
	if err != nil {
		// 已存在同样的待执行任务,视为成功
		if errors.Is(err, asynq.ErrDuplicateTask) {
			return nil
		}
		return err
	}
	nextRun := info.NextProcessAt
	return rds.SetTaskInfo(job.ID, &nextRun)
}

// DeletePendingOnce 删除一次性任务的待执行 asynq 任务(停用/删除时调用)
func DeletePendingOnce(job *rds.Job) {
	if job.ID == 0 {
		return
	}
	insp := asynq.NewInspector(redisOpt())
	defer insp.Close()
	if err := insp.DeleteTask(Queue(), OnceTaskID(job.ID)); err != nil && !errors.Is(err, asynq.ErrTaskNotFound) {
		glog.Z().Error(fmt.Sprintf("[tasks] delete pending task fail, id=%d task=%s err=%v",
			job.ID, OnceTaskID(job.ID), err))
	}
	_ = rds.ClearTaskInfo(job.ID)
}

// EnqueueManual 手动立即执行一次
func EnqueueManual(job *rds.Job) error {
	_, err := Client.EnqueueContext(context.Background(), newTask(job.ID, rds.TriggerManual),
		asynq.Queue(Queue()),
		asynq.MaxRetry(0),
	)
	return err
}

func newTask(jobID int64, triggerType string) *asynq.Task {
	b, _ := json.Marshal(Payload{JobID: jobID, TriggerType: triggerType})
	return asynq.NewTask(TypeJobExec, b)
}

// HandleJobExec 统一任务执行入口
func HandleJobExec(ctx context.Context, t *asynq.Task) error {
	var p Payload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("%w: payload解析失败 %v", asynq.SkipRetry, err)
	}
	job, err := rds.GetJob(p.JobID)
	if err != nil {
		return fmt.Errorf("%w: 任务不存在 id=%d", asynq.SkipRetry, p.JobID)
	}

	timeout := job.TimeoutSec
	if timeout <= 0 {
		timeout = 300
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	jl := rds.StartJobLog(job, p.TriggerType)

	var output string
	switch job.Type {
	case rds.TypeHTTP:
		output, err = RunHTTP(runCtx, job.Payload)
	case rds.TypeShell:
		output, err = RunShell(runCtx, job.Payload)
	case rds.TypeGoFunc:
		output, err = RunGoFunc(runCtx, job.Payload)
	default:
		err = fmt.Errorf("未知任务类型: %s", job.Type)
	}
	rds.FinishJobLog(jl, output, err)

	// 一次性任务执行完立即过期(手动测试不影响状态)
	if job.ScheduleType == rds.SchedOnce && p.TriggerType != rds.TriggerManual {
		_ = rds.SetStatus(job.ID, rds.StatusExpired)
		_ = rds.ClearTaskInfo(job.ID)
	}
	if job.ScheduleType == rds.SchedCron {
		// 周期任务: 根据 cron 表达式算出下次执行时间, 写入 next_run
		if next, cerr := NextCronRun(job.CronExpr, time.Now()); cerr != nil {
			glog.Z().Warn(fmt.Sprintf("[tasks] calc next run fail, id=%d name=%s expr=%s err=%v",
				job.ID, job.Name, job.CronExpr, cerr))
		} else {
			_ = rds.SetNextRun(job.ID, &next)
		}
	}

	if err != nil {
		glog.Z().Warn(fmt.Sprintf("[tasks] job fail, id=%d name=%s trigger=%s err=%v",
			job.ID, job.Name, p.TriggerType, err))
		return err
	}
	return nil
}

// zapLogger 将 asynq 日志接到 zap
type zapLogger struct {
	s *zap.SugaredLogger
}

func (l *zapLogger) Debug(args ...interface{}) {
	l.s.Debug(append([]interface{}{"[asynq] "}, args...)...)
}
func (l *zapLogger) Info(args ...interface{}) {
	l.s.Info(append([]interface{}{"[asynq] "}, args...)...)
}
func (l *zapLogger) Warn(args ...interface{}) {
	l.s.Warn(append([]interface{}{"[asynq] "}, args...)...)
}
func (l *zapLogger) Error(args ...interface{}) {
	l.s.Error(append([]interface{}{"[asynq] "}, args...)...)
}
func (l *zapLogger) Fatal(args ...interface{}) {
	l.s.Fatal(append([]interface{}{"[asynq] "}, args...)...)
}
