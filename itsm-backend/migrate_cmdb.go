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

	// Run schema migration first
	fmt.Println("Running schema migration...")
	err = client.Schema.Create(ctx)
	if err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}
	fmt.Println("Schema migration completed successfully!")

	// Get default tenant
	defaultTenant, err := getOrCreateDefaultTenant(ctx, client)
	if err != nil {
		log.Fatalf("failed to get default tenant: %v", err)
	}

	// Initialize CMDB data
	err = initializeCMDBData(ctx, client, defaultTenant.ID)
	if err != nil {
		log.Fatalf("failed to initialize CMDB data: %v", err)
	}

	fmt.Println("CMDB migration completed successfully!")
	fmt.Println("You can now run 'go run main.go' to start the server.")
}

func getOrCreateDefaultTenant(ctx context.Context, client *ent.Client) (*ent.Tenant, error) {
	// Check if default tenant already exists
	existing, err := client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err == nil {
		fmt.Println("Using existing default tenant...")
		return existing, nil
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

func initializeCMDBData(ctx context.Context, client *ent.Client, tenantID int) error {
	fmt.Println("Initializing CMDB data...")

	// Create default CI types
	err := createDefaultCITypes(ctx, client, tenantID)
	if err != nil {
		return fmt.Errorf("failed to create CI types: %w", err)
	}

	// Create default relationship types
	err = createDefaultRelationshipTypes(ctx, client, tenantID)
	if err != nil {
		return fmt.Errorf("failed to create relationship types: %w", err)
	}

	fmt.Println("CMDB data initialization completed!")
	return nil
}

func createDefaultCITypes(ctx context.Context, client *ent.Client, tenantID int) error {
	fmt.Println("Creating default CI types...")

	ciTypes := []struct {
		Name            string
		DisplayName     string
		Description     string
		Category        string
		Icon            string
		AttributeSchema map[string]interface{}
	}{
		{
			Name:        "server",
			DisplayName: "服务器",
			Description: "物理服务器或虚拟机",
			Category:    "infrastructure",
			Icon:        "server",
			AttributeSchema: map[string]interface{}{
				"cpu_cores":  map[string]interface{}{"type": "integer", "required": false},
				"memory_gb":  map[string]interface{}{"type": "integer", "required": false},
				"storage_gb": map[string]interface{}{"type": "integer", "required": false},
				"os":         map[string]interface{}{"type": "string", "required": false},
				"ip_address": map[string]interface{}{"type": "string", "required": false},
				"hostname":   map[string]interface{}{"type": "string", "required": true},
			},
		},
		{
			Name:        "database",
			DisplayName: "数据库",
			Description: "数据库实例",
			Category:    "application",
			Icon:        "database",
			AttributeSchema: map[string]interface{}{
				"db_type":      map[string]interface{}{"type": "string", "required": true},
				"version":      map[string]interface{}{"type": "string", "required": false},
				"port":         map[string]interface{}{"type": "integer", "required": false},
				"schema_count": map[string]interface{}{"type": "integer", "required": false},
			},
		},
		{
			Name:        "application",
			DisplayName: "应用程序",
			Description: "业务应用程序",
			Category:    "application",
			Icon:        "application",
			AttributeSchema: map[string]interface{}{
				"app_type": map[string]interface{}{"type": "string", "required": false},
				"version":  map[string]interface{}{"type": "string", "required": false},
				"port":     map[string]interface{}{"type": "integer", "required": false},
				"url":      map[string]interface{}{"type": "string", "required": false},
			},
		},
		{
			Name:        "network_device",
			DisplayName: "网络设备",
			Description: "路由器、交换机等网络设备",
			Category:    "infrastructure",
			Icon:        "network",
			AttributeSchema: map[string]interface{}{
				"device_type":   map[string]interface{}{"type": "string", "required": true},
				"model":         map[string]interface{}{"type": "string", "required": false},
				"ports":         map[string]interface{}{"type": "integer", "required": false},
				"management_ip": map[string]interface{}{"type": "string", "required": false},
			},
		},
		{
			Name:        "service",
			DisplayName: "业务服务",
			Description: "业务服务定义",
			Category:    "service",
			Icon:        "service",
			AttributeSchema: map[string]interface{}{
				"service_level": map[string]interface{}{"type": "string", "required": false},
				"availability":  map[string]interface{}{"type": "string", "required": false},
				"criticality":   map[string]interface{}{"type": "string", "required": false},
			},
		},
	}

	for _, ciType := range ciTypes {
		_, err := client.CIType.Create().
			SetName(ciType.Name).
			SetDisplayName(ciType.DisplayName).
			SetDescription(ciType.Description).
			SetCategory(ciType.Category).
			SetIcon(ciType.Icon).
			SetAttributeSchema(ciType.AttributeSchema).
			SetIsSystem(true).
			SetTenantID(tenantID).
			Save(ctx)

		if err != nil {
			// 如果已存在则跳过
			fmt.Printf("CI Type %s already exists or error: %v\n", ciType.Name, err)
			continue
		}
		fmt.Printf("Created CI Type: %s\n", ciType.DisplayName)
	}

	return nil
}

func createDefaultRelationshipTypes(ctx context.Context, client *ent.Client, tenantID int) error {
	fmt.Println("Creating default relationship types...")

	relationshipTypes := []struct {
		Name          string
		DisplayName   string
		Description   string
		Direction     string
		Cardinality   string
		SourceCITypes []string
		TargetCITypes []string
	}{
		{
			Name:        "depends_on",
			DisplayName: "依赖于",
			Description: "表示一个CI依赖于另一个CI",
			Direction:   "unidirectional",
			Cardinality: "many_to_many",
		},
		{
			Name:          "hosts",
			DisplayName:   "托管",
			Description:   "表示一个CI托管另一个CI",
			Direction:     "unidirectional",
			Cardinality:   "one_to_many",
			SourceCITypes: []string{"server"},
			TargetCITypes: []string{"application", "database"},
		},
		{
			Name:          "connects_to",
			DisplayName:   "连接到",
			Description:   "表示网络连接关系",
			Direction:     "bidirectional",
			Cardinality:   "many_to_many",
			SourceCITypes: []string{"server", "network_device"},
			TargetCITypes: []string{"server", "network_device"},
		},
		{
			Name:          "uses",
			DisplayName:   "使用",
			Description:   "表示一个CI使用另一个CI的服务",
			Direction:     "unidirectional",
			Cardinality:   "many_to_many",
			SourceCITypes: []string{"application"},
			TargetCITypes: []string{"database", "service"},
		},
		{
			Name:          "provides",
			DisplayName:   "提供",
			Description:   "表示CI提供某种服务",
			Direction:     "unidirectional",
			Cardinality:   "one_to_many",
			SourceCITypes: []string{"application", "server"},
			TargetCITypes: []string{"service"},
		},
		{
			Name:        "contains",
			DisplayName: "包含",
			Description: "表示层次包含关系",
			Direction:   "unidirectional",
			Cardinality: "one_to_many",
		},
	}

	for _, relType := range relationshipTypes {
		_, err := client.CIRelationshipType.Create().
			SetName(relType.Name).
			SetDisplayName(relType.DisplayName).
			SetDescription(relType.Description).
			SetDirection(relType.Direction).
			SetCardinality(relType.Cardinality).
			SetSourceCiTypes(relType.SourceCITypes).
			SetTargetCiTypes(relType.TargetCITypes).
			SetIsSystem(true).
			SetTenantID(tenantID).
			Save(ctx)

		if err != nil {
			// 如果已存在则跳过
			fmt.Printf("Relationship Type %s already exists or error: %v\n", relType.Name, err)
			continue
		}
		fmt.Printf("Created Relationship Type: %s\n", relType.DisplayName)
	}

	return nil
}
