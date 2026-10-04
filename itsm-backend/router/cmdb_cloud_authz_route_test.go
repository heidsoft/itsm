package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-04 边缘功能收口 E4-20b/20c）：/api/v1/cmdb/cloud-* 这 15 条活表面路由
// 曾经在每个维度上都比被删掉的 /api/v1/cloud/* 那一侧更弱，收敛后必须补齐：
//
//  1. 租户身份 fail-closed：handler 直接 c.GetInt("tenant_id")，缺上下文时以 tenant=0 进查询。
//     真实链路上 TenantMiddleware 会先拦（401/2001），所以这里断言的是「认证但无租户」
//     绝不能拿到任何数据；handler 自身的兜底分支由 handlers/cmdb 的单测覆盖。
//  2. 跨租户与「不存在」同义：Get/Update 走 ent 的 tenant 谓词 → 404/4004；
//     Delete 曾把「按 tenant+id 命中 0 行」当删除成功返回 code:0，现在必须 404，
//     并证明归属租户侧的行仍在（不是级联误删）。
//  3. provider 枚举：写入体与三条列表查询都只认六个规范值，别名（alibaba/qcloud 等）
//     只在适配器边界由 NormalizeProvider 归一化。
//  4. 同租户内账号与服务厂商矛盾是 400（数据自相矛盾），不再是从 fmt.Errorf 落成的 500。
//  5. 任何失败路径都不得外露 SQL/驱动/内部错误文案。

// cloudSurfaceFailureMessage 断言失败响应的公开契约：业务码、稳定文案、零泄漏。
// HTTP 状态由调用方断言（Fail 会按业务码反查状态，两者各自独立成立才是完整契约）。
func cloudSurfaceFailureMessage(t *testing.T, body string, env cmdbListEnvelope, wantCode int, wantMessage string) {
	t.Helper()
	assert.Equal(t, wantCode, env.Code, "body="+body)
	assert.Equal(t, wantMessage, env.Message, "失败路径必须给稳定公开文案: body="+body)
	for _, leak := range []string{"sql", "Error 1", "pq:", "sqlite3", "no such", "driver:", "runtime error", "fmt.Errorf"} {
		assert.NotContains(t, body, leak, "响应不得外露底层错误细节: body="+body)
	}
}

func TestCMDBCloudSurfaceTenantAndEnumRouteContract(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_cmdb_cloud_authz_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	r := setupCMDBListRouter(t, client, logger)

	a := seedCloudSurfaceTenant(ctx, t, client, "cloud-authz-a", "aliyun", 3)
	b := seedCloudSurfaceTenant(ctx, t, client, "cloud-authz-b", "aws", 3)
	require.NotEqual(t, a.tenantID, b.tenantID)

	// 同租户内故意做一个厂商矛盾的云服务类型：A 的云账号是 aliyun，这个服务是 aws。
	conflictingService, err := client.CloudService.Create().
		SetProvider("aws").
		SetServiceCode("s3").
		SetServiceName("Object Storage").
		SetResourceTypeCode("bucket").
		SetResourceTypeName("Bucket").
		SetTenantID(a.tenantID).
		SetCreatedAt(cloudSurfaceTime).
		SetUpdatedAt(cloudSurfaceTime).
		Save(ctx)
	require.NoError(t, err)

	call := func(t *testing.T, method, path string, tenant cloudSurfaceTenant, tenantOverride *int, body interface{}) (int, string, cmdbListEnvelope) {
		t.Helper()
		tenantID := tenant.tenantID
		if tenantOverride != nil {
			tenantID = *tenantOverride
		}
		token, err := middleware.GenerateAccessToken(tenant.adminID, tenant.adminName, "super_admin", tenantID, cmdbListSecret, time.Hour)
		require.NoError(t, err)

		payload := strings.NewReader("")
		if body != nil {
			raw, marshalErr := json.Marshal(body)
			require.NoError(t, marshalErr)
			payload = strings.NewReader(string(raw))
		}
		req := httptest.NewRequest(method, path, payload)
		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var env cmdbListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), env
	}
	as := func(t *testing.T, method, path string, tenant cloudSurfaceTenant, body interface{}) (int, string, cmdbListEnvelope) {
		t.Helper()
		return call(t, method, path, tenant, nil, body)
	}
	expectFailure := func(t *testing.T, method, path string, tenant cloudSurfaceTenant, body interface{}, wantStatus, wantCode int, wantMessage string) string {
		t.Helper()
		status, raw, env := as(t, method, path, tenant, body)
		assert.Equal(t, wantStatus, status, "path=%s body=%s", path, raw)
		cloudSurfaceFailureMessage(t, raw, env, wantCode, wantMessage)
		return raw
	}

	t.Run("认证但无租户上下文时拿不到任何云数据", func(t *testing.T) {
		// 修复前：handler 用 c.GetInt("tenant_id")，缺上下文会以 tenant=0 进查询，
		// 表现为「这个租户没有云资源」的空列表假成功，而不是拒绝。
		zero := 0
		for _, path := range []string{
			"/api/v1/cmdb/cloud-services",
			"/api/v1/cmdb/cloud-accounts",
			"/api/v1/cmdb/cloud-resources",
			fmt.Sprintf("/api/v1/cmdb/cloud-services/%d", a.serviceIDs[0]),
			fmt.Sprintf("/api/v1/cmdb/cloud-resources/%d", a.resourceIDs[0]),
		} {
			status, raw, env := call(t, http.MethodGet, path, a, &zero, nil)
			assert.Equal(t, http.StatusUnauthorized, status, "path=%s body=%s", path, raw)
			assert.Equal(t, 2001, env.Code, "path=%s body=%s", path, raw)
			assert.Equal(t, "租户信息缺失", env.Message, raw)
			assert.NotContains(t, raw, `"items"`, "无租户时绝不能返回列表载荷: path="+path+" body="+raw)
		}
	})

	t.Run("跨租户读取统一 404 且按资源给文案", func(t *testing.T) {
		for _, tc := range []struct{ name, path, message string }{
			{"云服务", fmt.Sprintf("/api/v1/cmdb/cloud-services/%d", b.serviceIDs[0]), "cloud service not found"},
			{"云账号", fmt.Sprintf("/api/v1/cmdb/cloud-accounts/%d", b.accountID), "cloud account not found"},
			{"云资源", fmt.Sprintf("/api/v1/cmdb/cloud-resources/%d", b.resourceIDs[0]), "cloud resource not found"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				raw := expectFailure(t, http.MethodGet, tc.path, a, nil, http.StatusNotFound, 4004, tc.message)
				assert.NotContains(t, raw, "cloud-b", "响应不得回露归属租户标识: body="+raw)
			})
		}
	})

	t.Run("本租户读取仍然正常", func(t *testing.T) {
		status, body, env := as(t, http.MethodGet, fmt.Sprintf("/api/v1/cmdb/cloud-services/%d", a.serviceIDs[0]), a, nil)
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		assert.Contains(t, body, `"serviceCode":"ecs"`, body)
	})

	t.Run("跨租户更新被拒且目标行未被改写", func(t *testing.T) {
		expectFailure(t, http.MethodPut, fmt.Sprintf("/api/v1/cmdb/cloud-services/%d", b.serviceIDs[0]), a,
			cloudServiceBody("aliyun", "hijacked", "Hijacked"), http.StatusNotFound, 4004, "cloud service not found")

		stored, err := client.CloudService.Get(ctx, b.serviceIDs[0])
		require.NoError(t, err)
		assert.Equal(t, "ecs", stored.ServiceCode, "跨租户 PUT 不得改写归属租户的行")
	})

	t.Run("跨租户删除返回 404 而不是删除成功", func(t *testing.T) {
		for _, tc := range []struct{ name, path, message string }{
			{"云服务", fmt.Sprintf("/api/v1/cmdb/cloud-services/%d", b.serviceIDs[0]), "cloud service not found"},
			{"云账号", fmt.Sprintf("/api/v1/cmdb/cloud-accounts/%d", b.accountID), "cloud account not found"},
			{"云资源", fmt.Sprintf("/api/v1/cmdb/cloud-resources/%d", b.resourceIDs[0]), "cloud resource not found"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				expectFailure(t, http.MethodDelete, tc.path, a, nil, http.StatusNotFound, 4004, tc.message)
			})
		}

		// 归属租户侧的行必须仍在：证明 404 来自租户谓词，不是把别人的数据删了。
		_, err := client.CloudService.Get(ctx, b.serviceIDs[0])
		require.NoError(t, err, "租户 B 的云服务必须仍然存在")
		_, err = client.CloudAccount.Get(ctx, b.accountID)
		require.NoError(t, err, "租户 B 的云账号必须仍然存在")
		_, err = client.CloudResource.Get(ctx, b.resourceIDs[0])
		require.NoError(t, err, "租户 B 的云资源必须仍然存在")
	})

	t.Run("本租户删除真实生效", func(t *testing.T) {
		victim, err := client.CloudResource.Create().
			SetCloudAccountID(a.accountID).
			SetServiceID(a.serviceIDs[0]).
			SetResourceID("i-to-delete").
			SetTenantID(a.tenantID).
			SetCreatedAt(cloudSurfaceTime).
			SetUpdatedAt(cloudSurfaceTime).
			Save(ctx)
		require.NoError(t, err)

		status, body, env := as(t, http.MethodDelete, fmt.Sprintf("/api/v1/cmdb/cloud-resources/%d", victim.ID), a, nil)
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)

		_, err = client.CloudResource.Get(ctx, victim.ID)
		assert.True(t, ent.IsNotFound(err), "本租户删除必须真的落库，err=%v", err)
	})

	t.Run("provider 枚举在三条列表上一致生效", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/cmdb/cloud-services",
			"/api/v1/cmdb/cloud-accounts",
			"/api/v1/cmdb/cloud-resources",
		} {
			// 修复前：任意字符串直接进精确匹配，拼错的厂商名表现为空列表而不是请求错误。
			expectFailure(t, http.MethodGet, path+"?provider=alibaba", a, nil, http.StatusBadRequest, 1001, "请求参数错误")

			status, body, env := as(t, http.MethodGet, path+"?provider=aliyun", a, nil)
			assert.Equal(t, http.StatusOK, status, "%s: %s", path, body)
			assert.Equal(t, 0, env.Code, body)
		}
	})

	t.Run("写入请求体的 provider 必须是规范值", func(t *testing.T) {
		// bind 阶段的失败文案沿用 handler 既有的 "Invalid request body"（全仓一致），
		// 本用例断言的是它没有被降级成 500，也没有把 validator 原文外露。
		expectFailure(t, http.MethodPost, "/api/v1/cmdb/cloud-services", a,
			cloudServiceBody("alibaba", "slb", "Load Balancer"), http.StatusBadRequest, 1001, "Invalid request body")
		expectFailure(t, http.MethodPost, "/api/v1/cmdb/cloud-accounts", a,
			cloudAccountBody("qcloud", "acct-illegal"), http.StatusBadRequest, 1001, "Invalid request body")

		count, err := client.CloudService.Query().Where(cloudservice.TenantIDEQ(a.tenantID)).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, count, "非法写入不得留下任何行（2 个种子 + 1 个厂商矛盾服务）")
	})

	t.Run("合法 provider 写入成功并落在本租户", func(t *testing.T) {
		status, body, env := as(t, http.MethodPost, "/api/v1/cmdb/cloud-services", a,
			cloudServiceBody("aliyun", "oss", "Object Storage Service"))
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		var created struct {
			ID       int    `json:"id"`
			Provider string `json:"provider"`
			TenantID int    `json:"tenantId"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &created), body)
		assert.Equal(t, "aliyun", created.Provider, body)
		assert.Equal(t, a.tenantID, created.TenantID, "归属租户必须来自认证上下文，不是请求体")

		expectFailure(t, http.MethodGet, fmt.Sprintf("/api/v1/cmdb/cloud-services/%d", created.ID), b, nil,
			http.StatusNotFound, 4004, "cloud service not found")
	})

	t.Run("引用其他租户的云账号/云服务登记资源为 404", func(t *testing.T) {
		expectFailure(t, http.MethodPost, "/api/v1/cmdb/cloud-resources", a,
			cloudResourceBody(a.accountID, b.serviceIDs[0], "i-hijack-svc"),
			http.StatusNotFound, 4004, "cloud service not found")
		expectFailure(t, http.MethodPost, "/api/v1/cmdb/cloud-resources", a,
			cloudResourceBody(b.accountID, a.serviceIDs[0], "i-hijack-acct"),
			http.StatusNotFound, 4004, "cloud account not found")
	})

	t.Run("同租户但厂商矛盾是 400 而不是 500", func(t *testing.T) {
		// A 的账号是 aliyun，conflictingService 是 aws：请求数据自相矛盾。
		// 修复前是裸 fmt.Errorf，被归类成 500/5001「操作失败」。
		// 现在它是 NewValidationError，由 common.classifyError 按 HTTPStatus 反查成
		// 4000（BadRequestCode），文案就是这条领域规则本身。
		// 注意与 bind 阶段的 1001 分属两层：1001 是「格式不对」，4000 是「格式对但语义矛盾」。
		expectFailure(t, http.MethodPost, "/api/v1/cmdb/cloud-resources", a,
			cloudResourceBody(a.accountID, conflictingService.ID, "i-mismatch"),
			http.StatusBadRequest, 4000, "cloud account and cloud service belong to different providers")
	})

	t.Run("手工登记云资源成功并回填派生 identity", func(t *testing.T) {
		status, body, env := as(t, http.MethodPost, "/api/v1/cmdb/cloud-resources", a,
			cloudResourceBody(a.accountID, a.serviceIDs[0], "i-manual"))
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		assert.Contains(t, body, `"provider":"aliyun"`, "provider 由云账号边派生: body="+body)
		assert.Contains(t, body, `"canonicalAccountId":"cloud-authz-a-acct"`, body)
		assert.Contains(t, body, `"resourceScope":"global"`, body)
		assert.Contains(t, body, `"identityVersion":1`, body)
	})
}

// cloudServiceBody 构造最小云服务写入体；只有 provider 可变，用于枚举正/反例。
func cloudServiceBody(provider, code, name string) map[string]interface{} {
	return map[string]interface{}{
		"provider":         provider,
		"serviceCode":      code,
		"serviceName":      name,
		"resourceTypeCode": code + "-type",
		"resourceTypeName": name + " type",
	}
}

func cloudAccountBody(provider, accountID string) map[string]interface{} {
	return map[string]interface{}{
		"provider":    provider,
		"accountId":   accountID,
		"accountName": accountID + " production",
	}
}

func cloudResourceBody(accountID, serviceID int, resourceID string) map[string]interface{} {
	return map[string]interface{}{
		"cloudAccountId": accountID,
		"serviceId":      serviceID,
		"resourceId":     resourceID,
		"resourceName":   "manual " + resourceID,
	}
}
