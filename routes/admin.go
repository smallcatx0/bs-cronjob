package routes

import (
	v1 "cron-job/controller/v1"

	"github.com/gin-gonic/gin"
)

func registAdmin(r *gin.Engine) {
	root := r.Group("/admin")

	jobRout := root.Group("/jobs")
	jobs := v1.Jobs{}
	jobRout.GET("/list", jobs.List)
	jobRout.POST("/add", jobs.Add)
	jobRout.POST("/detail", jobs.Detail)
	jobRout.POST("/update", jobs.Update)
	jobRout.POST("/delete", jobs.Delete)

	jobRout.POST("/run", jobs.Run)
	jobRout.POST("/toggle", jobs.Toggle)

	jobRout.POST("/log", jobs.Log)
	jobRout.GET("/gofuncs", jobs.GoFuncs)

	// db_strategy 表数据维护策略配置(TTL 清理 / Retry 重试)
	td := v1.Tabledata{}
	ttlRout := root.Group("/tabledata/ttl")
	ttlRout.GET("/list", td.TtlList)
	ttlRout.POST("/add", td.TtlAdd)
	ttlRout.GET("/detail", td.TtlDetail)
	ttlRout.POST("/update", td.TtlUpdate)
	ttlRout.POST("/delete", td.TtlDelete)
	ttlRout.POST("/toggle", td.TtlToggle)

	retryRout := root.Group("/tabledata/retry")
	retryRout.GET("/list", td.RetryList)
	retryRout.POST("/add", td.RetryAdd)
	retryRout.GET("/detail", td.RetryDetail)
	retryRout.POST("/update", td.RetryUpdate)
	retryRout.POST("/delete", td.RetryDelete)
	retryRout.POST("/toggle", td.RetryToggle)
}
