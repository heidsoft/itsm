// 创建数据库schema
if err := client.Schema.Create(context.Background()); err != nil {
	log.Fatalf("Failed to create schema resources: %v", err)
}