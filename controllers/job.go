package controllers

import (
	"content_pipeline/services"
	"errors"
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
//	@param title formData string true "A prompt for optimizing a title"
//	@param description formData string true "A prompt for optimizing a description"
//	@param site formData string true "The URL of the site to be processed"
//	@param file formData file true "The CSV file containing the data for the job"
//
// Responses:
//
//   - 204: Job created successfully, no content returned.
//   - 400: Bad request, invalid input parameters or file format.
//   - 500: Internal server error, an unexpected error occurred while processing the job.
func (c *JobController) CreateJob() {
	title := c.GetString("title")
	description := c.GetString("description")
	siteUrl := c.GetString("site")

	file, _, err := c.GetFile("file")

	if err != nil {
		c.writeError(400, "csv file is required")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)

	if err != nil {
		c.writeError(400, "failed to read csv file")
		return
	}

	validationService := services.ValidationService{}
	err = validationService.ValidateJob(title, description, siteUrl, fileBytes)

	if err != nil {
		c.writeMappedError(err)
		return
	}

	jobService := services.JobService{}
	ctx := c.Ctx.Request.Context()
	err = jobService.ProcessJob(ctx, title, description, siteUrl, fileBytes)

	if err != nil {
		c.writeMappedError(err)
		return
	}

	c.Ctx.ResponseWriter.WriteHeader(204)
}

// writeMappedError maps a service error to an appropriate HTTP status code and error message, and writes the response back to the client.
//
// Parameters:
//
//   - err: The error object to be mapped and written to the response. This error is expected to be of type *services.ServiceError, which contains additional context about the error type and message.
//
// The function checks the type of the error and sets the HTTP status code and error message accordingly.
// For validation errors, it returns a 400 status code with a specific message. For configuration, storage, LLM, processing, or serialization errors, it returns a 500 status code. If the error does not match any known types, it defaults to a 500 status code with a generic internal server error message.
func (c *JobController) writeMappedError(err error) {
	status := 500
	message := "internal server error"

	var serviceErr *services.ServiceError
	hasServiceErr := errors.As(err, &serviceErr)

	switch {
	case errors.Is(err, services.ErrValidation):
		status = 400
		if hasServiceErr {
			message = serviceErr.Message
		} else {
			message = "invalid request"
		}
	case errors.Is(err, services.ErrConfiguration), errors.Is(err, services.ErrStorage), errors.Is(err, services.ErrLLM), errors.Is(err, services.ErrProcessing), errors.Is(err, services.ErrSerialization):
		status = 500
	}

	c.writeError(status, message)
}

// writeError sets the HTTP status code and writes a JSON response containing the error message.
func (c *JobController) writeError(status int, message string) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = map[string]string{"error": message}
	c.ServeJSON()
}
