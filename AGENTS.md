
这是 cron-job 项目的项目级代理说明，后续所有代码修改都应遵循本文件与现有仓库实现保持一致。

## 项目定位

这是一个基于 Go + Gin + GORM + Asynq 的定时任务管理平台，核心功能包括：

- 定时任务 CRUD
- cron / once 两种调度类型
- 任务启停、手动执行、日志查询
- 异步任务执行与状态同步
- 统一响应、分页、参数校验与错误处理

## 实际启动链路

使用代码中的真实入口作为准绳：

1. 启动入口：cmd/main.go
2. 配置/启动初始化：bootstrap/init.go
3. 路由注册：routes/router.go
4. 接口实现：controller/v1/jobs.go
5. 任务执行：internal/tasks
6. 持久化模型：models/dao/rds
7. 参数校验：models/valid
8. 统一响应：middleware/resp

关键约定：

- 先走 bootstrap 初始化，再注册路由
- 任务调度与执行逻辑放在 internal/tasks
- HTTP 接口逻辑优先参考 controller/v1/jobs.go
- 统一返回格式和错误码使用 middleware/resp

## 代码规范

遵循现有接口风格，以 controller/v1/jobs.go 为主线。

- 所有参数校验逻辑都写在 models/valid 下
- 优先使用 gin 的 validator 能力，例如 required、omitempty、oneof、min、max、datetime
- 涉及多字段联动校验的复杂逻辑，写在 func (p *Type) Valid() error {} 中
- 简单入参模型（4 个字段以下）可直接在 controller 内使用匿名结构体，无须额外定义在 models/valid
- 简单 CRUD 逻辑直接写在 controller 层，不要无必要地增加 service 层
- 数据库模型定义在 models/dao/rds，模型相关逻辑也放在该包下
- 数据响应模型、分页、错误码统一使用 middleware/resp
- 任务状态、调度类型、任务类型等枚举/常量优先复用 models/dao/rds 中现有定义，而非重复声明

如果新增接口，应该同步检查：

- 校验模型是否在 models/valid
- 相关 controller 是否遵循现有 CRUD 结构
- 数据库操作是否落在 models/dao/rds
- 返回值是否统一走 middleware/resp


## 工作流程要求

- 修改前先定位同类接口和同类模型，优先复用现有实现
- 对非 trivial 的需求，先规划方案，再写入 doc/xxxx-plan.md，最后逐步实现
- 优先保持现有命名、参数结构和返回结构一致，不要为了“更抽象”而引入额外的框架或层次
- 若修改逻辑涉及任务运行、持久化和 HTTP 接口，需同时检查这三者是否保持一致

## 参考入口

- 接口实现：controller/v1/jobs.go
- 校验实现：models/valid/job.go
- 模型与状态：models/dao/rds
- 响应封装：middleware/resp
- 任务调度：internal/tasks
