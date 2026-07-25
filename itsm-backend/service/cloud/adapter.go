package cloud

import (
	"context"

	"itsm-backend/ent"
)

// CloudDiscoveryAdapter 云发现适配器接口
type CloudDiscoveryAdapter interface {
	Provider() string
	ServiceCode() string
	InitClients(ctx context.Context, account *ent.CloudAccount, regions []string) (map[string]Client, error)
	ListRegions(ctx context.Context, account *ent.CloudAccount) ([]string, error)
	DiscoverRegion(ctx context.Context, account *ent.CloudAccount, region string, client Client, nextToken string) (*PageResult, error)
	Close()
	ValidateCredential(ctx context.Context, account *ent.CloudAccount) error
}

// Client 云厂商 SDK 客户端抽象
type Client interface {
	Close() error
}
