package main

import (
	"content_pipeline/pkg/groq"
	"content_pipeline/pkg/s3"
	_ "content_pipeline/routers"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/filter/cors"
)

func main() {
	awsEndpoint, err := beego.AppConfig.String("app::aws_endpoint")
	if err != nil {
		panic("S3 endpoint is required")
	}

	awsAccessKey, err := beego.AppConfig.String("app::aws_access_key")
	if err != nil {
		panic("S3 access key is required")
	}

	awsSecretKey, err := beego.AppConfig.String("app::aws_secret_key")
	if err != nil {
		panic("S3 secret key is required")
	}

	awsRegion, err := beego.AppConfig.String("app::aws_region")
	if err != nil {
		panic("S3 region is required")
	}

	err = s3.Init(awsEndpoint, awsAccessKey, awsSecretKey, awsRegion)

	// Init Groq after config is loaded
	groqAPIKey, err := beego.AppConfig.String("app::groq_api_key")
	if err != nil || groqAPIKey == "" {
		panic("GROQ API key is required")
	}

	groqBaseURL, err := beego.AppConfig.String("app::groq_base_url")
	if err != nil || groqBaseURL == "" {
		panic("GROQ base URL is required")
	}

	groq.Init(groqAPIKey, groqBaseURL)
	if err != nil {
		panic(err)
	}

	frontendURL, err := beego.AppConfig.String("app::frontend_url")
	if err != nil || frontendURL == "" {
		frontendURL = "http://localhost:3000" // default to localhost if not configured
	}

	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowOrigins: []string{
			frontendURL, // your React dev server
		},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Authorization",
			"Access-Control-Allow-Origin",
			"Content-Type",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Headers",
		},
		AllowCredentials: true,
	}))

	beego.Run()
}
