package controller

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// uniqueTestID 生成唯一的测试ID，用于避免测试间的唯一约束冲突
func uniqueTestID() string {
	return fmt.Sprintf("%d", rand.Int31())
}

// SetupTestDB 为每个测试创建一个独立的内存数据库，避免测试间冲突
func SetupTestDB(t *testing.T) *ent.Client {
	// 为每个测试使用唯一的数据库文件名
	dbName := fmt.Sprintf("file:ent_%s_%d?mode=memory&cache=shared&_fk=1", t.Name(), rand.Int31())
	client := enttest.Open(t, "sqlite3", dbName)
	return client
}

// SetupTestLogger 为测试创建 logger
func SetupTestLogger(t *testing.T) *zap.SugaredLogger {
	return zaptest.NewLogger(t).Sugar()
}

// RequireTestDB 创建一个测试数据库并在测试结束时关闭
func RequireTestDB(t *testing.T) *ent.Client {
	client := SetupTestDB(t)
	t.Cleanup(func() {
		client.Close()
	})
	return client
}

// CreateTestTenantWithID 使用唯一ID创建测试租户
func CreateTestTenantWithID(t *testing.T, client *ent.Client, prefix string) *ent.Tenant {
	uniqueID := uniqueTestID()
	tenant, err := client.Tenant.Create().
		SetName(prefix + " Tenant " + uniqueID).
		SetCode(prefix + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(t.Context())
	require.NoError(t, err)
	return tenant
}
