package main

import (
	"content_pipeline/pkg/geminiai"
	"content_pipeline/pkg/minio"
	_ "content_pipeline/routers"

	beego "github.com/beego/beego/v2/server/web"
)

func main() {
	err := minio.Init("localhost:9000", "minioadmin", "minioadmin", false)
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
