# cron-job 定时任务管理平台

将日志系统中硬编码的定时任务统一管理:动态启停、手动运行、查看运行记录与日志。
所有任务调度均基于 **Asynq**(Redis)。

## 架构

- 周期任务(cron):`asynq.Scheduler` 注册 cronspec;启用/停用增量同步 `RegisterCron` / `UnregisterCron`,仅进程启动时按 DB 全量重建
- 一次性任务(once):`client.Enqueue + ProcessAt(execute_at)`,固定 TaskID(`job:once:{id}`),停用/删除时撤销
- 手动运行:立即入队,`trigger_type=manual`,不影响任务状态
- 执行入口:`internal/tasks.HandleJobExec`,按类型分发 http / shell / gofunc,前后写 `bs_job_log`
- 下次执行时间(`next_run`):cron 任务启用时计算写入,每次执行完重新计算,停用时清除;once 任务入队时写入 `execute_at`,停用/执行完成后清除

> cron 表达式为标准 5 段(`分 时 日 月 周`),与 asynq 调度器解析器一致,支持 `@every` 等描述符,不支持秒字段/6 段 Quartz 格式。

## 任务字段

`bs_job` 主要字段:`name` 任务名、`description` 任务描述(可选,≤255)、`type`、`status`、`schedule_type`、`cron_expr` / `execute_at`、`payload`、`timeout_sec`、`next_run`。

## 启动

```bash
# 1. 初始化数据库表
mysql -h127.0.0.1 -ugobs -p bs < doc/schema.sql

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
| /detail | POST | 详情(query ?id=) |
| /update | POST | 更新(仅停用状态,按提交字段更新,含 description) |
| /delete | POST | 删除(启用中不可删) |
| /run | POST | 手动立即执行一次 |
| /toggle | POST | 启用/停用 `{"id":1,"status":1}`,同步维护 next_run |
| /log | POST | 运行记录分页(job_id/job_name/status/trigger_type 过滤) |
| /gofuncs | GET | 已注册 go func 名称列表 |


## 状态机

- 任务:`0 停用 → 1 启用`;once 任务执行完成后置 `2 已过期`
- 运行记录:`running → success / failed`
