//go:build migrate
// +build migrate

package main

import (
	"context"
	"fmt"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"log"
	"time"

	_ "github.com/lib/pq"
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

	// 1. First run schema migration to create all tables
	fmt.Println("Running schema migration...")
	err = client.Schema.Create(ctx)
	if err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}
	fmt.Println("Schema migration completed successfully!")

	// 2. Then create default tenant
	defaultTenant, err := createDefaultTenant(ctx, client)
	if err != nil {
		log.Fatalf("failed to create default tenant: %v", err)
	}

	fmt.Printf("Created/Found default tenant with ID: %d\n", defaultTenant.ID)
	fmt.Println("Migration completed successfully!")
	fmt.Println("You can now run 'go run main.go' to start the server.")
}

func createDefaultTenant(ctx context.Context, client *ent.Client) (*ent.Tenant, error) {
	// Check if default tenant already exists
	existingTenant, err := client.Tenant.Query().
		Where(tenant.CodeEQ("default")).
		First(ctx)
	if err == nil {
		// Default tenant already exists
		fmt.Println("Default tenant already exists")
		return existingTenant, nil
	}

	// Create default tenant
	fmt.Println("Creating default tenant...")
	return client.Tenant.Create().
		SetName("Default Tenant").
		SetCode("default").
		SetDomain("localhost").
		SetStatus(tenant.StatusActive).
		SetType(tenant.TypeEnterprise).
		SetSettings(map[string]interface{}{
			"max_users":   1000,
			"max_tickets": 10000,
		}).
		SetQuota(map[string]interface{}{
			"storage":   "10GB",
			"bandwidth": "100GB",
		}).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}
