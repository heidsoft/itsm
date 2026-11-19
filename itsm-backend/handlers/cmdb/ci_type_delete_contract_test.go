package cmdb

// CI 类型删除的 HTTP 契约锁。
//
// 锁的是「客户端从 HTTP 响应里看到什么」，而不是 service 返回值：
// 曾的缺陷是 DeleteCIType 走 Ent 预检（读拦截器已过滤 scrapped），
// 类型被已退役 CI 物理占用时预检放行，DELETE 撞库外键 23503，
// 客户端拿到 {"code":5001,"message":"删除CI类型失败"}——运维只能看到
// 「系统故障」，无法知道要去清理退役资产，于是反复重试。
//
// 本测试把两件事钉死：
//  1. 类型被占用时必须是 HTTP 409 + 业务码 4090，不是 500/5001；
//  2. message 必须点明「配置项引用」这一可执行原因，不得回落到调用点兜底文案。
//
// 反向验证（2026-11-19 已执行）：把 DeleteCIType 退回裸 fmt.Errorf 后，
// 本文件首个用例在 assert.Equal(4090) 处立即红（实际 5001），
// 恢复后转绿——红在断言而非编译。

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/database"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/services"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type ciTypeDeleteHandlerFixture struct {
	router   *gin.Engine
	tenantID int
	typeID   int
}

func newCITypeDeleteHandlerFixture(t *testing.T) *ciTypeDeleteHandlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	// 与生产同口径：读拦截器会让 Ent 看不到 scrapped 行。
	database.RegisterSecurityInterceptors(client, "off")

	ctx := t.Context()
	tenant, err := client.Tenant.Create().SetName("类型删除租户").SetCode("ci-type-del").
		SetDomain("ci-type-del.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	// 一条**已退役**的 CI：物理占用 ci_type_id，但读面不可见。
	_, err = client.ConfigurationItem.Create().
		SetName("retired-ci").SetCiNumber("CI-retired-01").
		SetCiTypeID(ciType.ID).SetCiType("server").
		SetStatus("retired").SetLifecycleStatus("scrapped").
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// 不能用 database.GetRawDB()：那是生产进程的全局单例，测试里为 nil，
	// 会让 service 退化到 Ent 分支——于是测试看不见退役行，压根不触发本用例
	// 要锁的 sentinel 文案（反向验证时它仍然通过，正是这个漏洞）。
	// 同一具名内存库另开连接：ent 不暴露底层 *sql.DB，而 rawDB 分支必须真的
	// 执行 SQL 才能证明它看得见退役行（cache=shared 保证两边是同一个库）。
	rawDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })
	require.NoError(t, rawDB.PingContext(ctx), "测试必须拿到可用连接，否则锁不住 rawDB 分支")
	ciTypeService := service.NewCITypeService(client, zap.NewNop().Sugar())
	ciTypeService.SetRawDB(rawDB)
	// 直接打到 gin handler，绕开 Service 的层层委托，贴住真实调用路径。
	productionSvc := &ProductionService{
		logger:        zap.NewNop().Sugar(),
		ciTypeService: ciTypeService,
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("user_id", 9)
		c.Next()
	})
	r.DELETE("/ci-types/:id", productionSvc.DeleteCIType)

	return &ciTypeDeleteHandlerFixture{router: r, tenantID: tenant.ID, typeID: ciType.ID}
}

func (f *ciTypeDeleteHandlerFixture) doDelete(t *testing.T, id int) (int, int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/ci-types/"+strconv.Itoa(id), nil)
	f.router.ServeHTTP(w, req)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应体不是 JSON: %s", w.Body.String())
	}
	return w.Code, resp.Code, resp.Message
}

// 退役 CI 占用类型时：HTTP 409 + 4090 + 可执行文案。
func TestDeleteCIType_ScrappedReferenceReturnsConflict(t *testing.T) {
	f := newCITypeDeleteHandlerFixture(t)

	status, code, msg := f.doDelete(t, f.typeID)

	assert.Equal(t, http.StatusConflict, status, "类型被占用是可预期冲突，不能报 500")
	assert.Equal(t, 4090, code, "业务码必须是 4090，不能是未分类的 5001")
	assert.Contains(t, msg, "引用", "文案必须点明阻塞原因是配置项引用")
	assert.NotEqual(t, "删除CI类型失败", msg,
		"不得回落到调用点兜底文案，否则客户端无法知道要清理哪些资产")
}
