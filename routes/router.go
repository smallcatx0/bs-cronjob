package routes

import (
	controller "cron-job/controller"
	"cron-job/internal/conf"

	"github.com/gin-gonic/gin"
)

// Register http路由总入口
func Register(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    conf.AppConf.GetString("base.name"),
			"desc":    conf.AppConf.GetString("base.describe"),
			"version": conf.AppConf.GetString("base.version"),
		})
	}) // version
	health := controller.Health{}
	{
		r.GET("/healthz", health.Healthz)
		r.GET("/ready", health.Ready)
		r.GET("/reload", health.ReloadConf)
		r.GET("/test", health.Test)
	}

	// 注册admin接口
	registAdmin(r)
}
