/*
Copyright (year) Bytedance Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"douyincloud-gin-demo/redis"
	"douyincloud-gin-demo/service"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	redis.Init()
	service.Init()

	r := gin.Default()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "200")
	})
	r.POST("/api/open_api", service.RunOpenApi)
	r.GET("/ws", service.ConnectOrDisconnect)
	r.POST("/ws", service.Uplink)
	log.Println("Server init success")
	r.Run(":8000")
}
