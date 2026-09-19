package main

import (
	bootstrap "cron-job/bootstrap"
	"cron-job/internal/conf"
	dbstrategy "cron-job/internal/db_strategy"
	"cron-job/internal/tasks"
	"cron-job/middleware/httpmd"
	routes "cron-job/routes"
)

func init() {
	bootstrap.InitFlag("conf/app.yaml")
}

func main() {
	if !bootstrap.Flag() {
		return
	}
	// 读取配置文件
	bootstrap.InitConf(&bootstrap.Param.C)
	bootstrap.InitLog()
	bootstrap.InitDB()
	// 初始化 任务调度生成端
	bootstrap.InitProducer()
	bootstrap.Heartbeat()

	app := bootstrap.NewApp(conf.IsDebug())
	app.GinEngibe.Use(httpmd.SetHeader)
	app.GinEngibe.Use(httpmd.ReqLog)
	// 注册路由
	app.RegisterRoutes(routes.Register)
	// 启动HTTP 服务
	app.Run(conf.HttpPort())
	// 等待退出
	app.WaitExit(shutdown)
}

// shutdown 优雅退出: 先停 db_strategy 调度器, 再停 asynq 生产/消费端
func shutdown() {
	dbstrategy.Instance().Shutdown()
	tasks.Shutdown()
}
