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

}
