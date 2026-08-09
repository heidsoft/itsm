package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud/aliyun"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize logger")
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adapter := aliyun.NewAliyunECSAdapter(logger.Sugar())
	account := &ent.CloudAccount{Provider: "aliyun", CredentialRef: "env://ITSM_ALIYUN"}
	if err := adapter.ValidateCredential(ctx, account); err != nil {
		fmt.Fprintf(os.Stderr, "Aliyun credential validation failed: %v\n", err)
		os.Exit(1)
	}

	regions, err := adapter.ListRegions(ctx, account)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Aliyun region discovery failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Aliyun connection validated; accessible regions: %d\n", len(regions))
}
