package main

import (
	_ "content_pipeline/routers"
	beego "github.com/beego/beego/v2/server/web"
)

func main() {
	beego.Run()
}

