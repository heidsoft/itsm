//go:build !migrate && !create_user && !seed_demo
// +build !migrate,!create_user,!seed_demo

// 构建标签：指定在什么条件下编译这个文件
// !migrate && !create_user && !seed_demo 表示当不执行数据库迁移、创建用户和
// 演示数据播种时编译此文件。这样可以避免在运行迁移脚本时启动完整的Web服务器
//
// Swagger / OpenAPI 全局信息（swag init -g main.go 只从本文件读取全局注解，
// 放在其他文件会被忽略——勿移回 swagger_meta.go）：
//
//	@title ITSM API
//	@version 1.6.9
//	@description AI-Native ITSM 平台 API：ITIL v4 核心流程（工单/事件/问题/变更/发布/服务请求）、CMDB、BPMN 工作流、知识库 RAG、SLA、RBAC 与多租户隔离。
//	@BasePath /api/v1
//	@securityDefinitions.apikey BearerAuth
//	@in header
//	@name Authorization

package main

import (
	"os"

	boot "itsm-backend/internal/bootstrap"
	"itsm-backend/middleware"
)

// main函数：Go程序的入口点
// 当程序启动时，首先执行这个函数
func main() {
	// 部署模式 gating:private 部署时关掉 /api/v1/msp/* 路由族。
	// 在 NewApplication 之前完成,确保 router 注册到的 msp 路径在请求进入时 404。
	middleware.ApplyDeploymentMode(os.Getenv("DEPLOYMENT_MODE"))

	if os.Getenv("ITSM_BOOTSTRAP_ONLY") == "true" {
		boot.RunInitialization()
		return
	}

	app := boot.NewApplication()
	app.Run()
}
