package main

import (
	"context"
	"fmt"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"log"
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

	// Run incremental schema migration
	fmt.Println("Running incremental schema migration for CMDB...")
	err = client.Schema.Create(ctx)
	if err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}
	fmt.Println("Schema migration completed successfully!")

	// Get default tenant
	defaultTenant, err := client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		log.Fatalf("default tenant not found, please run basic migration first: %v", err)
	}

	// Initialize CMDB data only
	err = initializeCMDBData(ctx, client, defaultTenant.ID)
	if err != nil {
		log.Fatalf("failed to initialize CMDB data: %v", err)
	}

	fmt.Println("CMDB incremental migration completed successfully!")
}

// 这里复用上面的 initializeCMDBData, createDefaultCITypes, createDefaultRelationshipTypes 函数
// [函数内容与上面相同，为了简洁这里省略]
