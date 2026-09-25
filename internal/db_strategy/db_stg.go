package dbstrategy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"cron-job/internal/conf"
	"cron-job/internal/tasks"
	"cron-job/models/dao"
	rds "cron-job/models/dao/rds"
	"cron-job/pkg/glog"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	sqlTimeOut = 60 // 单批 sql 执行超时(秒)
	LockKeyTpl = "bs:dbauto:%s"

	// strategyTaskTimeout 单个策略任务的整体超时上限, 入队时通过 asynq.Timeout 显式设置
	// (asynq 默认 30 分钟), 超时后 asynq 取消 ctx, 执行链需全程透传该 ctx 以便删除循环/SQL 被取消,
	// 不再留下僵尸 goroutine 与永久 running 的策略日志; 单批超时(sqlTimeOut)须小于该值
	strategyTaskTimeout = 30 * time.Minute

	lockTTL           = 300 * time.Second // 锁有效期
	lockRenewInterval = 60 * time.Second  // 执行期间续租间隔(约 lockTTL/5)

	logPre = "[db_strategy] "
)

// unlockScript 仅当锁仍为自己持有时删除(compare-and-delete), 避免误删其他实例重新抢到的锁
var unlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

// renewScript 仅当锁仍为自己持有时续期(compare-and-pexpire), 避免给别人的锁续命
var renewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

// asynq 任务类型, 冒号后为策略 kind, 与 Payload.Kind 一致
const (
	TypeTtlStrategy   = "dbstrategy:ttl"
	TypeRetryStrategy = "dbstrategy:retry"

	KindTtl   = "ttl"
	KindRetry = "retry"
)

// Payload db_strategy asynq 任务负载, 仅携带策略 Kind+ID, 消费端按此查配置库获取最新完整配置
type Payload struct {
	Kind string `json:"kind"` // ttl / retry
	ID   int64  `json:"id"`   // 策略配置行主键(tabledata_ttl/tabledata_retry.id)
}

type DbStrategy struct {
	Logger   *zap.Logger
	sched    *asynq.Scheduler // 按 cronspec 触发策略任务入队(生产端)
	client   *asynq.Client    // 供测试/运行期手动入队使用
	db       *gorm.DB         // 配置表所在的数据库链接
	redisCli *redis.Client
	ttl      *rds.TabledataTtl
	retry    *rds.TabledataRetry
	entries  map[string]string // 策略任务名 -> asynq entryID
	mu       sync.Mutex
	Debug    bool
}

var (
	instance   *DbStrategy
	instanceMu sync.Mutex
)

// Instance 返回最近一次构造的 DbStrategy 单例, 供消费端注册 handler 与执行任务使用
func Instance() *DbStrategy {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	if instance == nil {
		instance, _ = NewDbStrategy(dao.MysqlCli, dao.RedisCli)
	}
	return instance
}

// InitInstance 消费端启动时预建单例(不启动调度器, 但需备好配置库供执行时查询)
func InitInstance(db *gorm.DB, redisCli *redis.Client) *DbStrategy {
	s, _ := NewDbStrategy(db, redisCli)
	return s
}

func NewDbStrategy(db *gorm.DB, redisCli *redis.Client) (*DbStrategy, error) {
	if redisCli == nil {
		redisCli = dao.RedisCli // 构造时未传入则复用全局 redis 连接
	}
	s := &DbStrategy{
		Logger:   glog.Z(),
		db:       db,
		redisCli: redisCli,
		entries:  make(map[string]string),
		ttl:      new(rds.TabledataTtl),
		retry:    new(rds.TabledataRetry),
	}
	if redisCli != nil {
		s.sched = asynq.NewScheduler(tasks.RedisOpt(),
			&asynq.SchedulerOpts{Location: time.Local, Logger: tasks.NewZapLogger(s.Logger)})
		s.client = asynq.NewClient(tasks.RedisOpt())
	}
	instanceMu.Lock()
	instance = s
	instanceMu.Unlock()
	return s, nil
}

// 根据数据库配置注册任务并启动调度器(生产端)
func (s *DbStrategy) Regist() {
	// 查询配置注册到 asynq Scheduler 中
	ttls, err := s.ttl.GetCfgs()
	if err != nil {
		s.Logger.Error(logPre + fmt.Sprintf(
			"查询数据%s库配置失败, err=%s",
			s.ttl.TableName(), err.Error(),
		))
	} else {
		// 依次加入调度器中(仅 online 状态的策略注册到调度器, 运行期增删改同步走 toggle 接口)
		for i := range ttls {
			cfg := ttls[i]
			if cfg.Status != rds.StrategyOnline {
				continue
			}
			s.AddCronTask(cfg.Spec, TypeTtlStrategy, "ttl:"+cfg.UnKey, &Payload{Kind: KindTtl, ID: cfg.ID})
		}
	}
	retrys, err := s.retry.GetCfgs()
	if err != nil {
		s.Logger.Error(logPre + fmt.Sprintf(
			"查询数据%s库配置失败, err=%s",
			s.retry.TableName(), err.Error(),
		))
	} else {
		for i := range retrys {
			cfg := retrys[i]
			if cfg.Status != rds.StrategyOnline {
				continue // 仅 online 状态的策略注册到调度器
			}
			s.AddCronTask(cfg.Spec, TypeRetryStrategy, "retry:"+cfg.Unkey, &Payload{Kind: KindRetry, ID: cfg.ID})
		}
	}
	// 启动调度器(非阻塞, 内部 goroutine 运行)
	if s.sched != nil {
		go func() {
			if err := s.sched.Run(); err != nil {
				s.Logger.Error(logPre + "scheduler exit: " + err.Error())
			}
		}()
	}
}

// AddCronTask 按 cronspec(标准 5 段)注册策略任务到调度器, 触发时入队策略任务。
// 重复添加同名任务定为更新; spec 由 Scheduler.Register 校验(asynq 仅支持 5 段表达式)。
// 注册失败时返回 error, 供运行期切换(online)调用方回滚状态。
func (s *DbStrategy) AddCronTask(spec, taskType, funName string, p *Payload) error {
	s.Logger.Info(logPre + fmt.Sprintf(
		"注册任务 %s(%s)", funName, spec,
	))
	b, err := json.Marshal(p)
	if err != nil {
		s.Logger.Error(logPre + fmt.Sprintf("序列化任务%s负载失败, err=%s", funName, err.Error()))
		return err
	}
	task := asynq.NewTask(taskType, b)
	// 不设置固定 TaskID: asynq 的 TaskID 语义是任务键存续期内不可重复入队,
	// 失败任务归档后任务键仍保留, 后续触发将一直 ErrTaskIDConflict 导致策略永久停摆;
	// 重叠触发防护交由消费端 Redis 执行锁, 失败不重试则下一轮 cron 触发即补偿
	opts := []asynq.Option{
		asynq.Queue(tasks.StrategyQueue()), // 策略任务走独立队列, 与业务 job:exec 隔离, worker 可拆分消费
		asynq.MaxRetry(0),
		asynq.Timeout(strategyTaskTimeout), // 显式整体超时, 到期取消 ctx 使执行链可被优雅中断
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sched == nil {
		s.Logger.Error(logPre + "scheduler 未初始化, 无法注册任务 " + funName)
		return errors.New("scheduler 未初始化, 无法注册任务 " + funName)
	}
	// 重复添加任务定为更新: 先注销旧注册项
	if eid, ok := s.entries[funName]; ok {
		if err := s.sched.Unregister(eid); err != nil {
			s.Logger.Error(logPre + fmt.Sprintf("注销旧任务%s注册项失败, err=%s", funName, err.Error()))
		}
		delete(s.entries, funName)
	}
	eid, err := s.sched.Register(spec, task, opts...)
	if err != nil {
		s.Logger.Error(logPre + fmt.Sprintf("%s 注册定时任务%s失败, err=%s", conf.HostName(), funName, err.Error()))
		return err
	}
	s.entries[funName] = eid
	return nil
}

// RemoveCronTask 按任务名从调度器注销策略任务注册项(策略切换为 offline 时调用),
// 仅撤销后续调度, 已入队待执行任务不回收(与 jobs 周期任务停用语义一致)
func (s *DbStrategy) RemoveCronTask(funName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	eid, ok := s.entries[funName]
	if !ok {
		return
	}
	if err := s.sched.Unregister(eid); err != nil {
		s.Logger.Error(logPre + fmt.Sprintf("注销任务%s注册项失败, err=%s", funName, err.Error()))
		return
	}
	delete(s.entries, funName)
	s.Logger.Info(logPre + fmt.Sprintf("注销任务 %s 成功", funName))
}

// Shutdown 停止调度器并释放客户端
func (s *DbStrategy) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sched != nil {
		s.sched.Shutdown()
		s.sched = nil
	}
	if s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}
	s.entries = map[string]string{}
}

// RegisterHandlers 将 db_strategy 任务 handler 注册到 asynq 消费端, 在启动消费端前调用
func RegisterHandlers() {
	h := func(ctx context.Context, t *asynq.Task) error {
		return Instance().Handle(ctx, t)
	}
	tasks.RegisterHandler(TypeTtlStrategy, h)
	tasks.RegisterHandler(TypeRetryStrategy, h)
}

// Handle asynq 任务执行入口(消费端): 按 payload 的 Kind+ID 查询配置库拿到最新策略配置,
// 竞争 Redis 分布式锁(补偿 asynq 缺失的 SkipIfStillRunning)后分发到 TTL 清理 / Retry 重试。
// asynq 传入的 ctx(携带整体超时/取消信号)全程透传到执行链, 任务被取消时策略日志同步闭环
func (s *DbStrategy) Handle(ctx context.Context, t *asynq.Task) (err error) {
	var p Payload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("%w: payload解析失败 %v", asynq.SkipRetry, err)
	}
	if s.db == nil {
		return fmt.Errorf("%w: 配置库连接为空, 无法查询策略配置", asynq.SkipRetry)
	}
	if s.redisCli == nil {
		return fmt.Errorf("%w: redis连接为空, 无法竞争策略执行锁", asynq.SkipRetry)
	}
	// 按 Kind+ID 查配置库: 取运行期最新策略配置(改配置立即生效), 并拿到锁唯一名、策略名与执行闭包
	var (
		unkey string
		name  string
		runFn func(ctx context.Context) (string, error)
	)
	switch p.Kind {
	case KindTtl:
		cfg, err := s.ttl.GetByID(p.ID)
		if err != nil {
			s.Logger.Error(logPre + fmt.Sprintf("查询ttl策略失败 id=%d err=%s", p.ID, err.Error()))
			return fmt.Errorf("%w: 查询ttl策略失败 id=%d err=%v", asynq.SkipRetry, p.ID, err)
		}
		unkey, name, runFn = cfg.UnKey, cfg.UnKey, func(ctx context.Context) (string, error) { return s.deleteTableRecord(ctx, *cfg) }
	case KindRetry:
		cfg, err := s.retry.GetByID(p.ID)
		if err != nil {
			s.Logger.Error(logPre + fmt.Sprintf("查询retry策略失败 id=%d err=%s", p.ID, err.Error()))
			return fmt.Errorf("%w: 查询retry策略失败 id=%d err=%v", asynq.SkipRetry, p.ID, err)
		}
		unkey, name, runFn = cfg.Unkey, cfg.Unkey, func(ctx context.Context) (string, error) { return s.updateTableRecord(ctx, *cfg) }
	default:
		return fmt.Errorf("%w: 未知策略类型 kind=%s", asynq.SkipRetry, p.Kind)
	}
	funName := p.Kind + ":" + unkey
	// 竞争分布式锁(带 owner token): 三态区分——抢到锁执行; 竞争失败说明其他实例正在执行, 跳过且不触发重试;
	// redis 异常不能当作"未抢到锁"静默跳过(否则 redis 抖动期间策略无声空转), 返回错误让任务 failed
	token, ok, lerr := s.acquireLock(ctx, funName)
	if lerr != nil {
		s.Logger.Error(logPre + fmt.Sprintf("竞争 %s 任务锁异常, err=%s", funName, lerr.Error()))
		// 同步写一条 failed 策略日志, 保证审计可见(下轮 cron 触发即补偿)
		jl := rds.StartStrategyLog(p.Kind, p.ID, name)
		rds.FinishStrategyLog(jl, "", lerr)
		return lerr
	}
	if !ok {
		s.Logger.Warn(logPre + fmt.Sprintf("%s 未抢到 %s 任务锁, 跳过执行", conf.HostName(), funName))
		return nil
	}
	// 先停续租再释放锁(defer LIFO), 避免删除后仍尝试续期
	defer s.releaseLock(funName, token)
	defer s.startLockRenew(funName, token)()

	// 参照 tasks 包 JobLog 机制: 执行开始写 running 日志, 结束回写结果;
	// 用命名返回值 + defer 闭环: 任务超时被取消/执行 panic 时也能回写终态, 不留永久 running 记录
	jl := rds.StartStrategyLog(p.Kind, p.ID, name)
	var output string
	defer func() {
		if r := recover(); r != nil {
			output, err = "", fmt.Errorf("panic: %v", r)
			s.Logger.Error(logPre + fmt.Sprintf("%s 执行panic: %v", funName, r))
		}
		if jl != nil {
			rds.FinishStrategyLog(jl, output, err)
		}
	}()
	output, err = runFn(ctx)
	return err
}

// newLockToken 生成锁 owner 标识: 主机名 + 随机串, 区分同机不同任务与不同实例
func newLockToken() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s:%d", conf.HostName(), time.Now().UnixNano())
	}
	return fmt.Sprintf("%s:%s", conf.HostName(), hex.EncodeToString(b))
}

// acquireLock 竞争分布式锁, 三态返回:
//   - 抢锁成功: (token, true, nil)
//   - 锁被其他实例持有: ("", false, nil), 调用方应跳过执行
//   - redis 异常: ("", false, err), 调用方应判定任务失败, 不可静默跳过
func (s *DbStrategy) acquireLock(ctx context.Context, funcName string) (string, bool, error) {
	key := fmt.Sprintf(LockKeyTpl, funcName)
	token := newLockToken()
	ok, err := s.redisCli.SetNX(ctx, key, token, lockTTL).Result()
	if err != nil {
		return "", false, fmt.Errorf("抢锁失败 redis_key=%s: %w", key, err)
	}
	if !ok {
		return "", false, nil
	}
	return token, true, nil
}

// releaseLock 释放锁, 仅删除自己持有的锁(Lua compare-and-delete)
func (s *DbStrategy) releaseLock(funcName, token string) {
	key := fmt.Sprintf(LockKeyTpl, funcName)
	n, err := unlockScript.Run(context.Background(), s.redisCli, []string{key}, token).Int64()
	if err != nil {
		s.Logger.Error(logPre + fmt.Sprintf("释放分布式锁失败, redis_key(%s) err=%s", key, err.Error()))
		return
	}
	if n == 0 {
		s.Logger.Warn(logPre + fmt.Sprintf("锁(%s)已非自己持有(可能超时被抢占), 跳过删除", key))
	}
}

// startLockRenew 启动续租 watchdog, 每 lockRenewInterval 仅在自己仍持有时延长 TTL,
// 防止长任务超过 lockTTL 后锁被其他实例抢占导致并发执行。返回停止函数。
func (s *DbStrategy) startLockRenew(funcName, token string) (stop func()) {
	key := fmt.Sprintf(LockKeyTpl, funcName)
	done := make(chan struct{})
	var once sync.Once
	go func() {
		ticker := time.NewTicker(lockRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				ok, err := renewScript.Run(
					context.Background(), s.redisCli, []string{key},
					token, lockTTL.Milliseconds(),
				).Int()
				if err != nil {
					s.Logger.Error(logPre + fmt.Sprintf("续租锁(%s)失败, err=%s", key, err.Error()))
					continue
				}
				if ok == 0 {
					s.Logger.Warn(logPre + fmt.Sprintf("锁(%s)已丢失(超时被抢占), 停止续租", key))
					return
				}
			}
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}

// withBatchTimeout 从任务 ctx 派生单批 SQL 执行超时: 任务整体取消与单批超时均能中断执行
func withBatchTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}

// 数据库删除逻辑, 返回执行摘要供策略日志记录; ctx 为 asynq 任务 ctx, 取消时中止分批循环
func (s *DbStrategy) deleteTableRecord(ctx context.Context, cfg rds.TabledataTtl) (string, error) {
	db, err := dao.ConnMysql(cfg.Dsn, true)
	if err != nil {
		s.Logger.Error(logPre + fmt.Sprintf(
			"数据库(%s)链接失败 err=%s",
			dao.DsnMask(cfg.Dsn), err.Error(),
		))
		return "", err
	}
	if s.Debug {
		db = db.Debug()
	}
	defer dao.CloseTmpMysql(db)
	st := time.Now()
	sql, err := rds.BuildTtlDeleteSql(cfg.Tablename, cfg.ColumnName, cfg.ColumnType, cfg.FindWh,
		time.Now().Add(-time.Second*time.Duration(cfg.TtlValue)), cfg.Limit)
	if err != nil {
		return "", err
	}
	deletedNum := int64(0)
	// 每批前检查任务 ctx: 整体超时/取消时立即停止, 不留下继续删数据的僵尸 goroutine
	for ctx.Err() == nil {
		rows, err := func() (int64, error) {
			timeout, cancel := withBatchTimeout(ctx, time.Second*sqlTimeOut)
			defer cancel()
			res := db.WithContext(timeout).Exec(sql)
			return res.RowsAffected, res.Error
		}()
		if err != nil {
			s.Logger.Error(logPre + fmt.Sprintf(
				"表(%s.%s); sql=%s, err=%s",
				cfg.DbName, cfg.Tablename, sql, err.Error(),
			))
			return "", err
		}
		deletedNum += rows
		if rows == 0 {
			break
		}
		// 批间等待也可被取消, 避免任务超时后仍卡在 Sleep
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
	if err := ctx.Err(); err != nil {
		s.Logger.Warn(logPre + fmt.Sprintf(
			"表(%s.%s) 删除任务被取消, 已删除%d条, err=%s",
			cfg.DbName, cfg.Tablename, deletedNum, err.Error(),
		))
		return "", err
	}
	cost := time.Since(st)

	out := fmt.Sprintf("表(%s.%s) sql=%s 删除%d条，耗时%dms",
		cfg.DbName, cfg.Tablename, sql, deletedNum, cost/time.Millisecond)
	s.Logger.Info(logPre + out)
	return out, nil
}

// 数据库重试逻辑, 返回执行摘要供策略日志记录

type CountRes struct {
	Count int64 `gorm:"column:c"`
}

func (s *DbStrategy) updateTableRecord(ctx context.Context, cfg rds.TabledataRetry) (string, error) {
	db, err := dao.ConnMysql(cfg.Dsn, true)
	if err != nil {
		s.Logger.Error(logPre + fmt.Sprintf(
			"数据库(%s)链接失败 err=%s",
			dao.DsnMask(cfg.Dsn), err.Error(),
		))
		return "", err
	}
	if s.Debug {
		db = db.Debug()
	}
	defer dao.CloseTmpMysql(db)
	curr := time.Now()
	findSql, updateSql, err := rds.BuildRetrySqls(cfg.Tablename, cfg.ColumnName, cfg.ColumnType,
		cfg.FindWh, cfg.SetFields, cfg.Before, cfg.Duration, cfg.Limit, curr)
	if err != nil {
		return "", err
	}
	s.Logger.Debug(logPre + "findSql: " + findSql)
	s.Logger.Debug(logPre + "updateSql: " + updateSql)
	res := CountRes{}
	err = db.WithContext(ctx).Raw(findSql).First(&res).Error
	if err != nil {
		s.Logger.Error(logPre + fmt.Sprintf(
			"表(%s.%s); sql=%s, err=%s",
			cfg.DbName, cfg.Tablename, findSql, err.Error(),
		))
		return "", err
	}
	if res.Count == 0 {
		// 不需要更新数据
		dt := time.Since(curr)
		out := fmt.Sprintf("表(%s.%s) sql=%s 无需更新，耗时%dms",
			cfg.DbName, cfg.Tablename, findSql, dt/time.Millisecond)
		s.Logger.Info(logPre + out)
		return out, nil
	}
	// 分批 UPDATE(每批 LIMIT cfg.Limit, 与 TTL 分批删除对齐), 避免单条语句无上限更新导致
	// 长事务/锁扩散/binlog 暴涨; 每批前检查任务 ctx, 批间间隔 1 秒可被取消
	updatedNum := int64(0)
	for ctx.Err() == nil {
		rows, err := func() (int64, error) {
			// 从任务 ctx 派生单批 UPDATE 超时, 任务整体取消与单批超时均能中断执行
			timeout, cancel := withBatchTimeout(ctx, 30*time.Second)
			defer cancel()
			res := db.WithContext(timeout).Exec(updateSql)
			return res.RowsAffected, res.Error
		}()
		if err != nil {
			s.Logger.Error(logPre + fmt.Sprintf(
				"表(%s.%s); sql=%s, err=%s",
				cfg.DbName, cfg.Tablename, updateSql, err.Error(),
			))
			return "", err
		}
		updatedNum += rows
		// RowsAffected < limit 说明命中行已更新完, 停止分批
		if rows < cfg.Limit {
			break
		}
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
	if err := ctx.Err(); err != nil {
		s.Logger.Warn(logPre + fmt.Sprintf(
			"表(%s.%s) 重试任务被取消, 已更新%d条, err=%s",
			cfg.DbName, cfg.Tablename, updatedNum, err.Error(),
		))
		return "", err
	}
	// TODO: 需要更新量，超过某阈值 告警

	cost := time.Since(curr)
	out := fmt.Sprintf("表(%s.%s) 命中%d条, 更新%d条, sql=%s, 耗时%dms",
		cfg.DbName, cfg.Tablename, res.Count, updatedNum, updateSql, cost/time.Millisecond)
	s.Logger.Info(logPre + out)
	return out, nil
}
