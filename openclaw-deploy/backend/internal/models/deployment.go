package models

import (
	"time"

	"gorm.io/gorm"
)

type Deployment struct {
	ID           string           `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       string           `gorm:"type:varchar(36);index" json:"user_id"`
	Plan         string           `gorm:"type:varchar(20)" json:"plan"`
	InstanceID   string           `gorm:"type:varchar(50);uniqueIndex" json:"instance_id"`
	InstanceName string           `gorm:"type:varchar(100)" json:"instance_name"`
	Domain       string           `gorm:"type:varchar(200)" json:"domain"`
	Status       string           `gorm:"type:varchar(20)" json:"status"`
	Region       string           `gorm:"type:varchar(50)" json:"region"`
	IPAddress    string           `gorm:"type:varchar(50)" json:"ip_address"`
	Username     string           `gorm:"type:varchar(100)" json:"username"`
	Password     string           `gorm:"type:varchar(200)" json:"-"`
	Config       *DeploymentConfig `gorm:"foreignKey:DeploymentID" json:"config,omitempty"`
	CreatedAt    time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

type DeploymentConfig struct {
	ID          string `gorm:"primaryKey;type:varchar(36)"`
	DeploymentID string `gorm:"uniqueIndex"`
	CPU         int    `json:"cpu"`
	Memory      int    `json:"memory"`
	Disk        int    `json:"disk"`
	Bandwidth   int    `json:"bandwidth"`
}

type DeploymentMetrics struct {
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	DiskUsage     float64 `json:"disk_usage"`
	NetworkIn     int64   `json:"network_in"`
	NetworkOut    int64   `json:"network_out"`
	QPS           int64   `json:"qps"`
	ResponseTime  int     `json:"response_time"`
	Timestamp     int64   `json:"timestamp"`
}

func (Deployment) TableName() string {
	return "deployments"
}

func CreateDeployment(db *gorm.DB, deployment *Deployment) error {
	return db.Create(deployment).Error
}

func GetDeployments(db *gorm.DB, page, pageSize int) ([]Deployment, int64, error) {
	var deployments []Deployment
	var total int64

	offset := (page - 1) * pageSize

	if err := db.Model(&Deployment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("Config").Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&deployments).Error
	if err != nil {
		return nil, 0, err
	}

	return deployments, total, nil
}

func GetDeployment(db *gorm.DB, id string) (*Deployment, error) {
	var deployment Deployment
	err := db.Preload("Config").First(&deployment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

func UpdateDeployment(db *gorm.DB, deployment *Deployment) error {
	return db.Save(deployment).Error
}

func DeleteDeployment(db *gorm.DB, id string) error {
	return db.Delete(&Deployment{}, "id = ?", id).Error
}

func GetDeploymentsByUser(db *gorm.DB, userID string) ([]Deployment, error) {
	var deployments []Deployment
	err := db.Preload("Config").Where("user_id = ?", userID).Find(&deployments).Error
	return deployments, err
}
