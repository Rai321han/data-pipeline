package controllers

import (
	"content_pipeline/services"
	"io"

	beego "github.com/beego/beego/v2/server/web"
)

type JobController struct {
	beego.Controller
}

// CreateJob handles the creation of a new job. It expects a multipart/form-data request with the following parameters:
//
// Request Parameters:
//
//	@param title formData string true "The title of the job"
//	@param description formData string true "A brief description of the job"
//	@param site formData string true "The URL of the site to be processed"
//	@param file formData file true "The CSV file containing the data for the job"
func (c *JobController) CreateJob() {
	title := c.GetString("title")
	description := c.GetString("description")
	siteUrl := c.GetString("site")

	file, _, err := c.GetFile("file")

	if err != nil {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "csv file is required"}
		c.ServeJSON()
		return
	}

	fileBytes, err := io.ReadAll(file)

	if err != nil {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "failed to read csv file"}
		c.ServeJSON()
		return
	}

	validationService := services.ValidationService{}
	err = validationService.ValidateJob(title, description, siteUrl, fileBytes)

	if err != nil {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": err.Error()}
		c.ServeJSON()
		return
	}

	defer file.Close()

	jobService := services.JobService{}
	err = jobService.CreateJob(title, description, siteUrl, fileBytes)

	if err != nil {
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "failed to create job"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]interface{}{
		"message": "job created successfully",
	}
	c.ServeJSON()
}
