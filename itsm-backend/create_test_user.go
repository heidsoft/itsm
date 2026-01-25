//go:build create_user
// +build create_user

package main

import (
	"context"
	"fmt"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"log"
	"os"
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

	// Get password from environment or generate random one
	password := os.Getenv("INIT_USER_PASSWORD")
	if password == "" {
		// Generate a random secure password
		generatedPassword, err := generateSecurePassword(16)
		if err != nil {
			log.Fatalf("failed to generate password: %v", err)
		}
		password = generatedPassword
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	// Create test user
	testUser, err := client.User.Create().
		SetUsername("admin").
		SetRole("admin").
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
	fmt.Printf("Password: %s\n", password)
	fmt.Println("Tenant Code: default (可选)")
	fmt.Println("")
	fmt.Println("Note: Set INIT_USER_PASSWORD environment variable to use a custom password")
}

func generateSecurePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	for i := range b {
		randomByte := make([]byte, 1)
		_, err := os.Read(randomByte)
		if err != nil {
			return "", err
		}
		b[i] = charset[int(randomByte[0])%len(charset)]
	}
	return string(b), nil
}
