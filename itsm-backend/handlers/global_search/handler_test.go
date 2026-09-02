package global_search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

type searchEnvelope struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    SearchResponse `json:"data"`
}

func newSearchTestClient(t *testing.T) *ent.Client {
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return client
}

func createSearchTenantAndUser(t *testing.T, client *ent.Client, suffix string) (int, int) {
	tenant, err := client.Tenant.Create().SetName("Tenant " + suffix).SetCode("tenant-" + suffix).Save(context.Background())
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("user-" + suffix).
		SetEmail("user-" + suffix + "@example.com").
		SetName("User " + suffix).
		SetPasswordHash("not-a-real-password").
		SetTenantID(tenant.ID).
		Save(context.Background())
	require.NoError(t, err)
	return tenant.ID, user.ID
}

func newSearchRouter(client *ent.Client, tenantID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		ctx.Next()
	})
	NewHandler(NewService(client)).RegisterRoutes(router.Group("/api/v1"))
	return router
}

func createSearchTicket(t *testing.T, client *ent.Client, tenantID, requesterID int, title, description string) *ent.Ticket {
	item, err := client.Ticket.Create().
		SetTitle(title).
		SetDescription(description).
		SetTicketNumber(fmt.Sprintf("T-%d-%s", tenantID, title)).
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return item
}

func performSearch(t *testing.T, router http.Handler, path string) (int, searchEnvelope) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, request)

	var response searchEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return recorder.Code, response
}

func TestSearchRouteFindsTicketsByQAndEnforcesTenant(t *testing.T) {
	client := newSearchTestClient(t)
	tenantA, userA := createSearchTenantAndUser(t, client, "a")
	tenantB, userB := createSearchTenantAndUser(t, client, "b")
	wanted := createSearchTicket(t, client, tenantA, userA, "测试工单：服务器无法访问", "网络连接失败")
	createSearchTicket(t, client, tenantB, userB, "测试工单：服务器无法访问", "另一个租户")
	createSearchTicket(t, client, tenantA, userA, "其他工单", "服务器无法访问，需要排查")

	status, response := performSearch(t, newSearchRouter(client, tenantA), "/api/v1/global-search?q="+"服务器无法访问")

	require.Equal(t, http.StatusOK, status)
	require.Equal(t, common.SuccessCode, response.Code)
	require.Equal(t, "success", response.Message)
	require.Equal(t, 2, response.Data.Total)
	require.Len(t, response.Data.Results, 2)
	require.Equal(t, wanted.ID, response.Data.Results[0].ID)
	require.Equal(t, "ticket", response.Data.Results[0].Type)
}

func TestSearchRouteWithBlankQReturnsStableEmptyResults(t *testing.T) {
	client := newSearchTestClient(t)
	tenantID, userID := createSearchTenantAndUser(t, client, "blank")
	createSearchTicket(t, client, tenantID, userID, "test ticket", "test description")

	status, response := performSearch(t, newSearchRouter(client, tenantID), "/api/v1/global-search?q=")

	require.Equal(t, http.StatusOK, status)
	require.Equal(t, common.SuccessCode, response.Code)
	require.NotNil(t, response.Data.Results)
	require.Empty(t, response.Data.Results)
	require.Zero(t, response.Data.Total)
}

func TestSearchRouteFailsClosedWithoutTenantContext(t *testing.T) {
	client := newSearchTestClient(t)
	router := gin.New()
	NewHandler(NewService(client)).RegisterRoutes(router.Group("/api/v1"))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/global-search?q=test", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var response common.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, common.BadRequestCode, response.Code)
}
