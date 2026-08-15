package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID       string `gorm:"primaryKey;type:varchar(36)"`
	Username string `gorm:"uniqueIndex;type:varchar(100)"`
	Email    string `gorm:"uniqueIndex;type:varchar(200)"`
	Password string `gorm:"type:varchar(200)"`
	Role     string `gorm:"type:varchar(20);default:'user'"`
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
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

	// 从环境变量获取管理员密码，避免硬编码
	adminPassword := getEnv("ADMIN_PASSWORD", "")
	if adminPassword == "" {
		log.Fatal("ADMIN_PASSWORD 环境变量未设置，请设置管理员初始密码")
	}

	// 创建默认管理员账号
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("密码哈希失败:", err)
	}

	adminUser := User{
		ID:       "admin-001",
		Username: "admin",
		Email:    "admin@itsm.com",
		Password: string(hashedPassword),
		Role:     "admin",
	}

	// 检查是否已存在
	var existing User
	result := db.Where("username = ?", "admin").First(&existing)

	if result.Error != nil {
		// 不存在则创建
		if err := db.Create(&adminUser).Error; err != nil {
			log.Fatal("创建管理员账号失败:", err)
		}
		fmt.Println("✅ 默认管理员账号创建成功！")
		fmt.Println("用户名：admin")
		// P0-6 修复：禁止将明文密码写入 stdout（会进入容器/CI 日志）。
		fmt.Printf("密码：使用 ADMIN_PASSWORD 环境变量指定的值（长度 %d）\n", len(adminPassword))
	} else {
		fmt.Println("ℹ️  管理员账号已存在")
	}

	// 仅在显式要求时重置密码（RESET_ADMIN_PASSWORD=true）。
	// P0-6 修复：原实现每次运行都会覆盖管理员密码，导致运营中修改过的密码被静默重置。
	if result.Error == nil && strings.EqualFold(strings.TrimSpace(os.Getenv("RESET_ADMIN_PASSWORD")), "true") {
		newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("密码哈希失败:", err)
		}
		if err := db.Model(&User{}).Where("username = ?", "admin").Update("password", string(newHashedPassword)).Error; err != nil {
			log.Fatal("重置管理员密码失败:", err)
		}
		fmt.Println("✅ 管理员密码已重置为 ADMIN_PASSWORD 指定的值")
	}

	fmt.Println("\n数据库初始化完成！")
}
