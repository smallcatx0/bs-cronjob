package bootstrap

/**
初始化文件
*/
import (
	"cron-job/internal/conf"
	"cron-job/internal/tasks"
	"cron-job/models/dao"
	"cron-job/pkg/glog"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// InitConf 配置文件初始化
func InitConf(filePath *string) {
	err := conf.InitAppConf(filePath)
	if err != nil {
		panic(err)
	}
}

// initLog 初始化日志
func InitLog() {
	c := conf.AppConf
	if c.GetString("log.type") == "file" {
		glog.InitLog2file(
			c.GetString("log.path"),
			c.GetString("log.level"),
		)
	} else {
		glog.InitLog2std(c.GetString("log.level"))
	}
}

// InitDB 初始化db
func InitDB() {
	dao.MustInitMysql()
	dao.MustInitRedis()
}

// InitProducer 初始化 asynq 生产端(client + scheduler),HTTP 服务侧使用
func InitProducer() {
	tasks.InitClient()
	tasks.InitScheduler()
}

// InitConsumer 初始化 asynq 消费端(server),独立消费程序使用
func InitConsumer() {
	tasks.Serve()
}

// 心跳&状态检测
func Heartbeat() {
	defer func() {
		if r := recover(); r != nil {
			glog.Error("Heartbeat goroutine run panic " + fmt.Sprint(r))
		}
	}()
	if !conf.AppConf.GetBool("metrics") {
		return
	}
	dt := conf.AppConf.GetInt("metrics_dt")
	if dt == 0 {
		dt = 10
	}
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(dt))
		defer ticker.Stop()
		for range ticker.C {
			glog.SysStatInfo()
		}
	}()
}

func WaitingExit(funs ...func()) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	switch <-quit {
	case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
		for _, f := range funs {
			if f != nil {
				f()
			}
		}
		log.Printf("Shutdown quickly, bye...")
	case syscall.SIGHUP:
		// 处理各种服务的优雅关闭
		for _, f := range funs {
			if f != nil {
				f()
			}
		}
		log.Printf("Shutdown gracefully, bye...")
	}
	os.Exit(0)
}
