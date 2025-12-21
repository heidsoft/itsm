//go:build !migrate && !create_user
// +build !migrate,!create_user

// 构建标签：指定在什么条件下编译这个文件
// !migrate && !create_user 表示当不执行数据库迁移和创建用户时编译此文件
// 这样可以避免在运行迁移脚本时启动完整的Web服务器

package main

import (
	"itsm-backend/internal/bootstrap"
)

// main函数：Go程序的入口点
// 当程序启动时，首先执行这个函数
func main() {
	app := bootstrap.NewApplication()
	app.Run()
}
