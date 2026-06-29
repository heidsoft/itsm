package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Menu struct {
	ID              int    `gorm:"primaryKey"`
	Name            string `gorm:"type:varchar(100)"`
	Path            string `gorm:"type:varchar(200)"`
	Icon            string `gorm:"type:varchar(50)"`
	ParentID        *int   `gorm:"column:parent_id"`
	PermissionCode  string `gorm:"type:varchar(100)"`
	SortOrder       int    `gorm:"column:sort_order"`
	TenantID        int    `gorm:"column:tenant_id"`
	IsVisible       bool   `gorm:"column:is_visible"`
	IsEnabled       bool   `gorm:"column:is_enabled"`
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	// 加载.env文件
	err := godotenv.Load("/Users/heidsoft/Downloads/research/itsm/itsm-backend/.env")
	if err != nil {
		log.Printf("警告：无法加载.env文件，使用环境变量: %v", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dbHost := getEnv("DB_HOST", "localhost")
		dbUser := getEnv("DB_USER", "itsm")
		dbPass := getEnv("DB_PASSWORD", "")
		dbName := getEnv("DB_NAME", "itsm")
		dbPort := getEnv("DB_PORT", "5432")
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			dbHost, dbUser, dbPass, dbName, dbPort)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 查询系统管理菜单的ID
	var adminMenu Menu
	result := db.Where("path = ? AND tenant_id = ?", "/admin", 1).First(&adminMenu)
	if result.Error != nil {
		log.Fatal("未找到系统管理菜单:", result.Error)
	}

	// 检查连接器菜单是否已存在
	var existingMenu Menu
	result = db.Where("path = ? AND tenant_id = ?", "/admin/connectors", 1).First(&existingMenu)
	if result.Error == nil {
		fmt.Println("ℹ️  连接器市场菜单已存在，无需重复添加")
		return
	}

	// 插入新菜单
	menu := Menu{
		Name:           "连接器市场",
		Path:           "/admin/connectors",
		Icon:           "Plug",
		ParentID:       &adminMenu.ID,
		PermissionCode: "connector:read",
		SortOrder:      317,
		TenantID:       1,
		IsVisible:      true,
		IsEnabled:      true,
	}

	if err := db.Create(&menu).Error; err != nil {
		log.Fatal("插入菜单失败:", err)
	}

	fmt.Println("✅ 连接器市场菜单添加成功！")
	fmt.Println("👉 请清理浏览器缓存，重新登录系统，在「系统管理」菜单下即可看到「连接器市场」入口")
}
