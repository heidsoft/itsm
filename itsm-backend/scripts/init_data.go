package main

import (
	"fmt"
	"log"
	"os"

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

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=itsm password=itsm_password_2026 dbname=itsm port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 创建默认管理员账号
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)

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
		fmt.Println("密码：admin123")
	} else {
		fmt.Println("ℹ️  管理员账号已存在")
	}

	// 更新密码（如果需要）
	if result.Error == nil {
		newHashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		db.Model(&User{}).Where("username = ?", "admin").Update("password", string(newHashedPassword))
		fmt.Println("✅ 管理员密码已重置为：admin123")
	}

	fmt.Println("\n数据库初始化完成！")
}
