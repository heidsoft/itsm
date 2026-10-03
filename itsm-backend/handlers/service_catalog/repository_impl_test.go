package service_catalog

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntRepository_Search_IncludesLegacyActiveStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	repo := NewEntRepository(client)

	_, err := client.ServiceCatalog.Create().
		SetName("VM Service").
		SetCategory("it_service").
		SetDescription("virtual machine").
		SetDeliveryTime(1).
		SetStatus("active").
		SetTenantID(1).
		Save(ctx)
	require.NoError(t, err)

	list, total, err := repo.Search(ctx, 1, "vm", ListFilters{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, list, 1)
	require.Equal(t, "active", list[0].Status)
}

// TestEntRepository_ListZeroValueFiltersCannotReadWholeTable 锁住 repository 侧的兜底：
// 非 HTTP 调用方（tests/scenarios 直接 repo.List(...)）传零值分页时，Ent 的 Limit(0)
// 等于不加 LIMIT，一次调用就会把该租户整表读出来。
//
// 这里断言的是「有界」而不是具体页长：repository 兜底走 common.ValidatePagination
// （DefaultPageSize=10），HTTP 入口走 common.GetPaginationFromQuery（缺省 20）。
// 两套缺省页长的统一属于账本 E4-9，不在本片范围内。
func TestEntRepository_ListZeroValueFiltersCannotReadWholeTable(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_list_zero_value?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	repo := NewEntRepository(client)

	const tenantID = 7
	for i := 0; i < 25; i++ {
		_, err := client.ServiceCatalog.Create().
			SetName(fmt.Sprintf("Catalog %02d", i)).
			SetCategory("compute").
			SetStatus("enabled").
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}

	list, total, err := repo.List(ctx, tenantID, ListFilters{})
	require.NoError(t, err)
	assert.Equal(t, 25, total, "total 是全量条数，不受页长影响")
	assert.Len(t, list, common.DefaultPageSize, "零值分页必须落到有界缺省页长，而不是整表")

	searchList, searchTotal, err := repo.Search(ctx, tenantID, "", ListFilters{})
	require.NoError(t, err)
	assert.Equal(t, 25, searchTotal)
	assert.Len(t, searchList, common.DefaultPageSize, "Search 必须与 List 共用同一条兜底")
}

// TestEntRepository_ListPaginationDoesNotDuplicateOrDrop 证明 created_at 同秒时
// ID 并列键能兜住页边界：三页拼接必须等于全序，不重不漏。
func TestEntRepository_ListPaginationDoesNotDuplicateOrDrop(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_list_paging?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	repo := NewEntRepository(client)

	const tenantID = 8
	for i := 0; i < 25; i++ {
		_, err := client.ServiceCatalog.Create().
			SetName(fmt.Sprintf("Page %02d", i)).
			SetCategory("compute").
			SetStatus("enabled").
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}

	all, _, err := repo.List(ctx, tenantID, ListFilters{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Len(t, all, 25)
	allIDs := make([]int, 0, len(all))
	for _, item := range all {
		allIDs = append(allIDs, item.ID)
	}

	var paged []int
	for page := 1; page <= 3; page++ {
		list, _, err := repo.List(ctx, tenantID, ListFilters{Page: page, PageSize: 10})
		require.NoError(t, err)
		want := 10
		if page == 3 {
			want = 5
		}
		require.Len(t, list, want)
		for _, item := range list {
			paged = append(paged, item.ID)
		}
	}
	assert.Equal(t, allIDs, paged, "created_at 同秒时必须由 ID 并列键兜住页边界")

	// 越界页码必须是空集，而不是回绕或重复最后一页。
	overflow, _, err := repo.List(ctx, tenantID, ListFilters{Page: 4, PageSize: 10})
	require.NoError(t, err)
	assert.Empty(t, overflow)
}
