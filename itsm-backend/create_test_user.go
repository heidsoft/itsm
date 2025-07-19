package main

import (
	"context"
	"fmt"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Connect to database
	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("failed connecting to database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Find default tenant
	defaultTenant, err := client.Tenant.Query().
		Where(tenant.CodeEQ("default")).
		First(ctx)
	if err != nil {
		log.Fatalf("failed to find default tenant: %v", err)
	}

	// Check if test user already exists
	existingUser, err := client.User.Query().
		Where(user.UsernameEQ("admin")).
		First(ctx)
	if err == nil {
		fmt.Printf("Test user already exists with ID: %d\n", existingUser.ID)
		return
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	// Create test user
	testUser, err := client.User.Create().
		SetUsername("admin").
		SetPasswordHash(string(passwordHash)).
		SetEmail("admin@example.com").
		SetName("管理员").
		SetDepartment("IT部门").
		SetActive(true).
		SetTenantID(defaultTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		log.Fatalf("failed to create test user: %v", err)
	}

	fmt.Printf("Created test user with ID: %d\n", testUser.ID)
	fmt.Println("Test credentials:")
	fmt.Println("Username: admin")
	fmt.Println("Password: admin123")
	fmt.Println("Tenant Code: default (可选)")
}
