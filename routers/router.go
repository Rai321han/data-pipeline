package routers

import (
	"content_pipeline/controllers"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	beego.Router("/api/v1/job", &controllers.JobController{}, "post:CreateJob")
}
