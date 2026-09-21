# db_strategy 说明

`dbstrategy`是一个**基于数据库配置的表数据自动维护策略引擎**：把「过期数据清理（TTL）」和「失败数据重试（Retry）」两类维护动作以配置行的形式存进数据库，进程启动时读取配置并注册到 **asynq Scheduler**，按 cron 触发入队、由 worker 消费执行，与项目任务调度（`internal/tasks`）共用同一套 asynq 链路。

## 能力概览

| 策略       | 配置表               | asynq 任务类型         | 执行函数                | 作用                                      |
| -------- | ----------------- | ------------------ | ------------------- | --------------------------------------- |
| TTL 清理   | `tabledata_ttl`   | `dbstrategy:ttl`   | `deleteTableRecord` | 按时间列删除过期记录，分批 `DELETE ... LIMIT` 循环直到删完 |
| Retry 重试 | `tabledata_retry` | `dbstrategy:retry` | `updateTableRecord` | 在时间窗口内查找满足条件的记录，批量 `UPDATE` 重置状态        |

## 文件结构

- `db_conn.go`：临时数据库连接的建立与关闭（GORM + MySQL）、SQL 日志接入 zap、DSN 脱敏
- `db_stg.go`：核心逻辑 `DbStrategy`，含 asynq Scheduler 注册（生产端）、任务 handler 与分布式锁（消费端）、TTL 删除与 Retry 更新
- 配置模型（`TabledataTtl` / `TabledataRetry`）、时间列类型常量与增删改查接口位于 `models/dao/rds/tabledata.go`，包内通过别名引用

## 架构与调度链路

```
HTTP 服务(cmd/main.go)                       worker(cmd/worker/main.go)
bootstrap.InitDbStrategyProducer             bootstrap.InitDbStrategyConsumer + InitConsumer
  └─ DbStrategy.Regist()                       └─ RegisterHandlers() 注册 dbstrategy:* handler
      ├─ 读取 tabledata_ttl / tabledata_retry       ↓
      ├─ Scheduler.Register(spec, task)      asynq Server 领取任务
      └─ sched.Run() 按 cron 入队 ──────────▶  Handle(): 按 Kind+ID 查配置库 -> 竞争
     (队列 tasks.StrategyQueue())      Redis 锁 -> 分发到 deleteTableRecord / updateTableRecord
```

- 策略任务使用独立队列 `asynq.strategy_queue`(默认 `dbstrategy`), 与业务 `job:exec` 队列隔离, worker 可通过 `asynq.queues` 配置拆分消费

- 任务负载 `Payload{Kind, ID}` 仅携带策略主键，消费端按 `Kind+ID` 回查配置库获取**执行时最新**配置（运行期改配置立即生效，无需重建调度器中的 payload）
- 代价：worker 消费端必须能访问配置库（已在 `InitInstance` 传入 `dao.MysqlCli`），且每次执行前多一次主键查询
- 入队不使用固定 TaskID：asynq 的 TaskID 语义是任务键存续期内不可重复入队，失败任务归档后任务键仍保留，后续触发将一直 `ErrTaskIDConflict` 导致策略永久停摆；重叠触发防护由消费端 Redis 执行锁承担
- `MaxRetry(0)`：失败不重试（直接归档），下一轮 cron 触发即补偿

## 核心类型与使用方式

生产端（HTTP 服务，已接入 `bootstrap.InitDbStrategyProducer`）：

```go
stg, err := dbstrategy.NewDbStrategy(dao.MysqlCli, dao.RedisCli) // 存放策略配置表的库连接
if err != nil {
    panic(err)
}
stg.Debug = false // 打开后打印执行的 SQL
stg.Regist()      // 读取配置表 -> 注册 asynq Scheduler -> Run() 按 cron 入队
```

消费端（worker，已接入 `bootstrap.InitDbStrategyConsumer`，需先于 `InitConsumer` 调用）：

```go
dbstrategy.InitInstance(dao.MysqlCli, dao.RedisCli) // 预建单例(备好配置库+redisCli, 不启动调度器)
dbstrategy.RegisterHandlers()                       // 注册 dbstrategy:ttl / dbstrategy:retry handler
tasks.ConsumerClient()                              // 随后正常启动 asynq 消费端
```

### DbStrategy 字段

- `Logger`：zap 日志，默认复用 `glog.Z()`
- `db`：策略配置表所在的数据库连接（生产端读取注册、消费端执行时按 ID 回查均依赖此库）
- `redisCli`：竞争执行锁用的 Redis 客户端，传 nil 时回落到 `dao.RedisCli`
- `sched` / `client`：asynq 调度器与入队客户端，Redis 参数复用 `tasks.RedisOpt()`
- `entries`：策略任务名 -> asynq entryID，重复注册同名任务会先注销旧注册项（即"更新"）
- `Debug`：为 true 时目标库连接开启 GORM Debug 模式

## 配置模型

### TabledataTtl（表 `tabledata_ttl`）

| 字段                            | 说明                                |
| ----------------------------- | --------------------------------- |
| `unkey`                       | 策略唯一 key，任务名格式为 `ttl:<unkey>`     |
| `dsn`                         | 目标数据库连接串                          |
| `db_name` / `table_name`      | 目标库名 / 表名（仅用于日志与 SQL 拼接）          |
| `column_name` / `column_type` | 依据的时间列名 / 列类型                     |
| `ttl_value`                   | 过期秒数，早于 `now - ttl_value` 的记录会被删除 |
| `limit`                       | 每批 DELETE 的条数                     |
| `spec`                        | cron 表达式（标准 5 段）                  |

### TabledataRetry（表 `tabledata_retry`）

| 字段                            | 说明                                                |
| ----------------------------- | ------------------------------------------------- |
| `unkey`                       | 策略唯一 key，任务名格式为 `retry:<unkey>`                   |
| `dsn`                         | 目标数据库连接串                                          |
| `column_name` / `column_type` | 依据的时间列名 / 列类型                                     |
| `before`                      | 时间窗口起点：当前时间往前 `before` 秒                          |
| `duration`                    | 时间窗口长度（秒），窗口为 `[now-before, now-before+duration)` |
| `find_wh`                     | 附加 WHERE 条件，如 `` `status`=30 ``                   |
| `set_fields`                  | UPDATE 的 SET 片段，如 `` `status`=1 ``                |
| `spec`                        | cron 表达式（标准 5 段）                                  |

### 时间列类型（column_type）

- `unix`：整数时间戳（秒）
- `timestamp`：时间戳类型
- `datetime`：日期时间类型

不支持的取值会直接返回错误。

## 执行机制

### cron 表达式

与项目任务平台对齐，使用 asynq Scheduler 内建的 cron 解析：**仅支持标准 5 段表达式（分 时 日 月 周）** 及 `@every` 等描述符，最小粒度为分钟。

### 并发与幂等保障

1. **执行锁（owner token + 续租）**：消费端执行前竞争 Redis 分布式锁，防止上一轮未跑完时下一轮/多副本重复投递并发执行（补偿 asynq 缺失的 `SkipIfStillRunning` 语义）
   - key 模板：`bs:dbauto:<ttl|retry>:<unkey>`，value 为 owner token（`主机名:随机串`），初始 TTL 300 秒
   - 执行期间由 watchdog 每 60 秒续租一次，且**仅在自己仍持有时**才延长 TTL（Lua `compare-and-pexpire`），支持任意长任务不丢锁
   - 释放使用 Lua `compare-and-delete`，只删自己持有的锁，不会误删超时后被其他实例抢占的锁
   - 未抢到锁（锁被其他实例持有）记 Warn 日志后返回 nil（不触发 asynq 重试）；**redis 异常与竞争失败三态区分**，异常时写一条 failed 策略日志并返回错误让任务 failed，避免 redis 抖动期间策略无声空转
2. **不使用固定 TaskID**：入队不设 `asynq.TaskID`，避免失败归档后任务键残留导致后续触发永久 `ErrTaskIDConflict`；重复入队由上述执行锁兜底（未抢到锁直接跳过）

> 注意：asynq Scheduler **没有**跨实例选主/激活锁（仅心跳上报），多副本部署 HTTP 服务时各副本都会按 cron 入队，需在部署层约束生产端仅单实例运行 `Regist()`。

### SQL 生成与执行

- TTL：按 `column_type` 拼出 `DELETE FROM <table> WHERE <col> < 阈值 LIMIT n`，循环执行直到 `RowsAffected == 0`，每批间隔 1 秒，单批超时 60 秒，避免大事务
- Retry：先 `SELECT count(*)` 探测，命中数为 0 直接返回；否则按批循环执行 `UPDATE <table> SET <set_fields> WHERE <时间窗口> AND <find_wh> LIMIT n`（n 为配置的 `limit`，与 TTL 分批删除对齐，避免单条语句无上限更新），单批超时 30 秒、批间间隔 1 秒（均可被任务 ctx 取消），`RowsAffected < limit` 时停止；前端 SQL 预览（`ParseSql`）与执行使用同一生成逻辑，预览 SQL 即执行 SQL（含 LIMIT）
- **ctx 全程透传**：asynq 任务 ctx（携带整体超时 `strategyTaskTimeout` = 30 分钟，入队时显式 `asynq.Timeout` 设置）透传到删除循环与所有 SQL：每批前检查 `ctx.Err()`、单批超时从任务 ctx 派生、批间等待可被取消；任务超时/取消时执行链立即中止，策略日志由 `Handle` 的 defer 闭环回写终态，不留永久 `running` 记录
- 目标库连接为**每次执行时临时建立、执行完关闭**（`connDb` / `CloseDb`），DSN 出错日志经 `DsnMask` 脱敏密码

## 单元测试

`db_stg_test.go` 为集成测试，依赖真实 MySQL 与 Redis（`tasks.RedisOpt()` 读取的配置需可用），运行前需填写：

- `test_db_dsn`：测试库 DSN
- `test_redis_addr`：Redis 地址

覆盖用例：

- `Test_deleteTableRecord`：TTL 删除逻辑
- `Test_updateTableRecord`：Retry 更新逻辑
- `Test_cronFun`：asynq 直接入队 + Scheduler 5 段表达式注册验证

## 已知限制与注意事项

1. **调度层热更新未实现**：`Regist()` 只在启动时读取一次配置，新增/删除策略、修改 `spec`（cron 表达式）需重启进程（代码已留 TODO，可参考 `tasks.RegisterCron` / `tasks.UnregisterCron` 的增量同步模式）。但因 payload 仅携带 ID、消费端执行前回查配置库，**已存在策略行的执行参数（dsn / ttl_value / limit / find_wh 等）修改会在下一轮立即生效**
2. **重叠执行防护依赖执行锁**：asynq 无 `SkipIfStillRunning` 语义，上一轮未跑完时下一轮 cron 仍会入队新任务；实际并发由消费端「owner token + 续租」执行锁拦截（抢不到锁则跳过）
3. **锁续租依赖进程存活**：若持锁进程崩溃，watchdog 停止续租，锁在 `lockTTL`（默认 300s）后自动释放，期间该策略不会被其他实例执行（故障转移有最长一个 TTL 的延迟）
4. **SQL 拼接**：表名、列名、`find_wh`、`set_fields` 均来自配置表直接拼接，配置表只对运维人员开放，不可接入不可信输入
