package main

import (
	"context"
	"database/sql"
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

	// First, connect to postgres database to drop and recreate our database
	postgresDSN := fmt.Sprintf("host=%s port=%d user=%s dbname=postgres sslmode=%s password=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.SSLMode, cfg.Database.Password)

	postgresDB, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer postgresDB.Close()

	// Drop existing database if it exists
	fmt.Printf("Dropping database %s if it exists...\n", cfg.Database.DBName)
	_, err = postgresDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.Database.DBName))
	if err != nil {
		log.Fatalf("failed to drop database: %v", err)
	}

	// Create fresh database
	fmt.Printf("Creating fresh database %s...\n", cfg.Database.DBName)
	_, err = postgresDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.Database.DBName))
	if err != nil {
		log.Fatalf("failed to create database: %v", err)
	}

	postgresDB.Close()

	// Now connect to our fresh database
	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("failed connecting to database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Run schema migration on fresh database
	fmt.Println("Running schema migration on fresh database...")
	err = client.Schema.Create(ctx)
	if err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}
	fmt.Println("Schema migration completed successfully!")

	// Create default tenant
	defaultTenant, err := createDefaultTenant(ctx, client)
	if err != nil {
		log.Fatalf("failed to create default tenant: %v", err)
	}

	fmt.Printf("Created default tenant with ID: %d\n", defaultTenant.ID)
	fmt.Println("Fresh migration completed successfully!")
	fmt.Println("You can now run 'go run main.go' to start the server.")
}

func createDefaultTenant(ctx context.Context, client *ent.Client) (*ent.Tenant, error) {
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
