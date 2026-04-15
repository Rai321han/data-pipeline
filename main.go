package main

import (
	"content_pipeline/pkg/geminiai"
	"content_pipeline/pkg/minio"
	_ "content_pipeline/routers"

	beego "github.com/beego/beego/v2/server/web"
)

func main() {
	minioEndpoint, err := beego.AppConfig.String("app::minio_endpoint")
	if err != nil {
		panic("Minio endpoint is required")
	}

	minioAccessKey, err := beego.AppConfig.String("app::minio_access_key")
	if err != nil {
		panic("Minio access key is required")
	}

	minioSecretKey, err := beego.AppConfig.String("app::minio_secret_key")
	if err != nil {
		panic("Minio secret key is required")
	}

	minioUseSSL, err := beego.AppConfig.Bool("app::minio_use_ssl")
	if err != nil {
		panic("Minio use SSL configuration is required")
	}

	err = minio.Init(minioEndpoint, minioAccessKey, minioSecretKey, minioUseSSL)
	if err != nil {
		panic(err)
	}

	llm_key, err := beego.AppConfig.String("app::llm_key")

	if err != nil {
		panic("LLM API key is required")
	}

	err = geminiai.InitGeminiAIClient(llm_key)
	if err != nil {
		panic(err)
	}

	beego.Run()
}
