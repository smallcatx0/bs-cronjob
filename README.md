# cron-job 定时任务管理平台

将日志系统中硬编码的定时任务统一管理:动态启停、手动运行、查看运行记录与日志。
所有任务调度均基于 **Asynq**(Redis)。

## 架构

- 周期任务(cron):`asynq.Scheduler` 注册 cronspec;任务启用/停用/更新后自动重建 Scheduler
- 一次性任务(once):`client.Enqueue + ProcessAt(execute_at)`,固定 TaskID(`job:once:{id}`),停用/删除时撤销
- 手动运行:立即入队,`trigger_type=manual`,不影响任务状态
- 执行入口:`internal/tasks.HandleJobExec`,按类型分发 http / shell / gofunc,前后写 `bs_job_log`

## 启动

```bash
# 1. 初始化数据库表
mysql -h10.2.3.18 -ugobs -p bs < sql/schema.sql

# 2. 拉取依赖(新增了 github.com/hibiken/asynq)
go mod tidy

# 3. 启动后端 HTTP 服务 + 调度器(:8083,只负责接口/入队/cron 触发,不消费任务)
go run ./cmd

# 4. 启动任务消费端(独立进程,监听 asynq 队列并执行任务)
go run ./cmd/worker

# 5. 启动前端(:5173, 已配置代理到后端)
cd web && npm install && npm run dev
```

> 编译独立二进制:`go build -o bin/cron-server ./cmd && go build -o bin/cron-worker ./cmd/worker`
> 生产环境可分别部署 HTTP 服务与消费端,消费端可水平扩容多个实例。

## 任务类型与 payload 格式

| 类型 | payload 示例 |
|---|---|
| http | `{"method":"GET","url":"http://x/api","headers":{"Token":"t"},"body":"","expect_status":200}` |
| shell | `{"cmd":"/data/scripts/backup.sh","args":["-full"],"dir":"/data"}`(cmd 必须在 `shell.whitelist` 内) |
| gofunc | `{"func":"demo.hello","args":{"name":"x"}}`(函数需 `tasks.Register` 注册) |

注册 go func(参照原 `AddCronFunc` 迁移方式):

```go
import "cron-job/internal/tasks"

tasks.Register("check_v3_domain_list", func(ctx context.Context, args json.RawMessage) (string, error) {
    v3log_metric.CheckV3DomainList()
    return "ok", nil
})
```

## API(前缀 /admin/jobs)

| 接口 | 方法 | 说明 |
|---|---|---|
| /list | GET | 分页查询(name/type/status/schedule_type 过滤) |
| /add | POST | 新建(默认停用) |
| /detail | GET | 详情(?id=) |
| /update | POST | 更新(仅停用状态,按提交字段更新) |
| /delete | POST | 删除(启用中不可删) |
| /run | POST | 手动立即执行一次 |
| /toggle | POST | 启用/停用 `{"id":1,"status":1}` |
| /log | POST | 运行记录分页(job_id/job_name/status/trigger_type 过滤) |
| /gofuncs | GET | 已注册 go func 名称列表 |

## 配置(conf/app.yaml 新增)

```yaml
asynq:
  queue: default     # 队列名
  concurrency: 10    # worker 并发数
shell:
  whitelist:         # shell 任务白名单(基名或全路径)
    - echo
```

## 状态机

- 任务:`0 停用 → 1 启用`;once 任务执行完成后置 `2 已过期`
- 运行记录:`running → success / failed`
