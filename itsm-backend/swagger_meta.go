package main

// swagger 服务入口：blank import 生成目录 docs/，使 swaggo/gin 能挂载
// /swagger/*any。全局 API 信息注解（@title/@version/@BasePath/
// @securityDefinitions）必须写在 main.go 的包注释中——swag init -g main.go
// 只从该文件读取全局信息（详见 main.go 顶部注释）。

import (
	_ "itsm-backend/docs"
)
