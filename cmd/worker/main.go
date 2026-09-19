package main

import (
	bootstrap "cron-job/bootstrap"
	"cron-job/internal/tasks"
)

func init() {
	bootstrap.InitFlag("conf/worker.yaml")
}

func main() {
	if !bootstrap.Flag() {
		return
	}
	// 读取配置文件
	bootstrap.InitConf(&bootstrap.Param.C)
	bootstrap.InitLog()
	bootstrap.InitDB()
	// 仅启动 asynq 消费端(server),监听队列并执行任务
	bootstrap.InitConsumer()
	bootstrap.Heartbeat()
	// 等待退出
	bootstrap.WaitingExit(tasks.Shutdown)
}
