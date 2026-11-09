package vector_store

// 域内窄契约：状态端点只需要租户维度的向量计数。
// 连通性探测走配置驱动的独立连接器，不属于本契约。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import "context"

type VectorStore interface {
	CountByTenant(ctx context.Context, tenantID int) (int64, error)
}
