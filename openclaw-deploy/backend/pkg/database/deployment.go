package database

import (
	"time"

	"gorm.io/gorm"
)

// Deployment 部署实例
type Deployment struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       string    `gorm:"type:varchar(36);index" json:"user_id"`
	Plan         string    `gorm:"type:varchar(20)" json:"plan"` // community, pro, enterprise
	InstanceID   string    `gorm:"type:varchar(50);uniqueIndex" json:"instance_id"`
	InstanceName string    `gorm:"type:varchar(100)" json:"instance_name"`
	Domain       string    `gorm:"type:varchar(200)" json:"domain"`
	Status       string    `gorm:"type:varchar(20)" json:"status"` // deploying, running, stopped, error
	Region       string    `gorm:"type:varchar(50)" json:"region"`
	IPAddress    string    `gorm:"type:varchar(50)" json:"ip_address"`
	Username     string    `gorm:"type:varchar(100)" json:"username"`
	Password     string    `gorm:"type:varchar(200)" json:"-"` // 加密存储
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// GetDeployments 获取部署列表
func GetDeployments(page, pageSize int) ([]Deployment, int64, error) {
	var deployments []Deployment
	var total int64

	offset := (page - 1) * pageSize

	err := db.Model(&Deployment{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = db.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&deployments).Error
	if err != nil {
		return nil, 0, err
	}

	return deployments, total, nil
}

// CreateDeployment 创建部署
func CreateDeployment(deployment *Deployment) error {
	return db.Create(deployment).Error
}

// GetDeployment 获取部署详情
func GetDeployment(id string) (*Deployment, error) {
	var deployment Deployment
	err := db.First(&deployment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

// UpdateDeployment 更新部署
func UpdateDeployment(deployment *Deployment) error {
	return db.Save(deployment).Error
}

// DeleteDeployment 删除部署
func DeleteDeployment(id string) error {
	return db.Delete(&Deployment{}, "id = ?", id).Error
}

// GetDeploymentsByUser 获取用户的部署
func GetDeploymentsByUser(userID string) ([]Deployment, error) {
	var deployments []Deployment
	err := db.Where("user_id = ?", userID).Find(&deployments).Error
	return deployments, err
}

// db 全局数据库实例
var db *gorm.DB

// Init 初始化数据库
func Init() error {
	dsn := getEnv("DB_DSN", "host=localhost user=postgres password=password dbname=openclaw_deploy port=5432 sslmode=disable TimeZone=Asia/Shanghai")
	
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// 自动迁移
	err = db.AutoMigrate(&Deployment{})
	if err != nil {
		return err
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
