package common

import (
	"context"
	"net/http/httptest"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// TestGetDepartmentTree_IncludesAllLevels 锁住部门树深度。
// 历史实现只 WithChildren 预加载一层，toDeptDomain 递归读 Edges.Children 时
// 孙节点的 edges 为空，三层及以上部门在「部门流程」页直接消失。
func TestGetDepartmentTree_IncludesAllLevels(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dept_tree_depth?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	root, err := client.Department.Create().SetName("总部").SetCode("HQ").SetTenantID(1).Save(ctx)
	require.NoError(t, err)
	child, err := client.Department.Create().
		SetName("技术部").SetCode("ENG").SetTenantID(1).SetParentID(root.ID).Save(ctx)
	require.NoError(t, err)
	grandchild, err := client.Department.Create().
		SetName("平台组").SetCode("PLAT").SetTenantID(1).SetParentID(child.ID).Save(ctx)
	require.NoError(t, err)

	tree, err := NewEntRepository(client).GetDepartmentTree(ctx, 1)
	require.NoError(t, err)
	require.Len(t, tree, 1, "只应有一个根部门")
	require.Len(t, tree[0].Children, 1)
	require.Len(t, tree[0].Children[0].Children, 1, "第三层部门不得丢失")
	require.Equal(t, grandchild.ID, tree[0].Children[0].Children[0].ID)
}

func TestShouldUseSecureCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("localhost http does not force secure", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		req := httptest.NewRequest("POST", "http://localhost:8090/api/v1/auth/login", nil)
		req.Host = "localhost:8090"
		c.Request = req

		require.False(t, shouldUseSecureCookies(c))
		require.Equal(t, "", cookieDomain(c))
	})

	t.Run("forwarded https enables secure", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		req := httptest.NewRequest("POST", "http://example.com/api/v1/auth/login", nil)
		req.Host = "example.com"
		req.Header.Set("X-Forwarded-Proto", "https")
		c.Request = req

		require.True(t, shouldUseSecureCookies(c))
		require.Equal(t, "", cookieDomain(c))
	})
}
