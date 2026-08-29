package main

// ITSM Swagger / OpenAPI 元信息。swaggo 在 `swag init` 时扫描本文件以提供
// @title / @version / @BasePath 等全局信息。本文件不带构建标签，确保常规构建与
// migrate / create_user 构建都能将其纳入。

// @title ITSM API
// @version 1.6.9
// @description AI-Native ITSM 平台 API：ITIL v4 核心流程（工单/事件/问题/变更/发布/服务请求）、
//   CMDB、BPMN 工作流、知识库 RAG、SLA、RBAC 与多租户隔离。
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @security BearerAuth
import (
	_ "itsm-backend/docs"
)
