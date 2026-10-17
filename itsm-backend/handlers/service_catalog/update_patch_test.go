package service_catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 服务目录 PUT 的部分更新语义回归（2026-10-16 隔离栈实测发现的缺陷类）：
// 旧实现把可选字段折叠成零值后无条件覆盖，页面只发 {status} 的批量发布/停用
// 会把 requiresApproval 从 true 静默改成 false；同时 0/false 因为 DTO 上的
// binding 与 omitempty 既写不进去也传不出来，表单里的「清除」无法表达。

// scDoRaw 与 scDoReq 走同一条 handler，但保留 recorder：
// 契约测试必须能断言 HTTP 状态码和响应字节，而不只是业务 code。
func scDoRaw(t *testing.T, r *gin.Engine, method, path string, body any) (*httptest.ResponseRecorder, common.Response) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "body=%s", w.Body.String())
	return w, resp
}

type scFull struct {
	Name              string  `json:"name"`
	Category          string  `json:"category"`
	Description       string  `json:"description"`
	Icon              string  `json:"icon"`
	ServiceType       string  `json:"serviceType"`
	Price             float64 `json:"price"`
	DeliveryTime      string  `json:"deliveryTime"`
	Unit              string  `json:"unit"`
	RequiresApproval  bool    `json:"requiresApproval"`
	ApprovalLevel     int     `json:"approvalLevel"`
	SLAResponseTime   int     `json:"slaResponseTime"`
	SLAResolutionTime int     `json:"slaResolutionTime"`
	SortOrder         int     `json:"sortOrder"`
	Status            string  `json:"status"`
	IsActive          bool    `json:"isActive"`
}

// scCreateFull 建一条每个字段都偏离默认值的目录，这样「被清零」在断言里是可见的。
func scCreateFull(t *testing.T, r *gin.Engine) (int, scFull) {
	t.Helper()
	uid := scUID()
	approval := true
	resp := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name:              "Patch-" + uid,
		Category:          "it_service",
		Description:       "原描述",
		Icon:              "cloud",
		ServiceType:       "rds",
		Price:             100,
		DeliveryTime:      "3",
		Unit:              "月",
		RequiresApproval:  &approval,
		ApprovalLevel:     2,
		SLAResponseTime:   30,
		SLAResolutionTime: 240,
		SortOrder:         7,
		Status:            "enabled",
	})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	require.IsType(t, map[string]interface{}{}, resp.Data)
	raw, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var out scFull
	require.NoError(t, json.Unmarshal(raw, &out))
	id, ok := resp.Data.(map[string]interface{})["id"].(float64)
	require.True(t, ok, "body=%s", string(raw))
	return int(id), out
}

func scGetFull(t *testing.T, r *gin.Engine, id int) scFull {
	t.Helper()
	resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs/"+strconv.Itoa(id), nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	raw, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var out scFull
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// scDataKeys 返回响应 data 的实际 JSON 键，用于证明 false/0 真的被序列化出来了。
func scDataKeys(t *testing.T, w *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), "body=%s", w.Body.String())
	require.NotNil(t, envelope.Data, "body=%s", w.Body.String())
	return envelope.Data
}

func TestUpdate_StatusOnlyKeepsUnsubmittedFields(t *testing.T) {
	r, _, _ := scSetup(t)
	id, before := scCreateFull(t, r)
	path := "/api/v1/service-catalogs/" + strconv.Itoa(id)

	w, resp := scDoRaw(t, r, "PUT", path, map[string]any{"status": "disabled"})
	require.Equal(t, 200, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(&resp))

	// 修复前：requiresApproval 变成 false —— 批量发布/停用会静默改写审批开关。
	got := scGetFull(t, r, id)
	assert.Equal(t, "disabled", got.Status)
	assert.False(t, got.IsActive)
	assert.True(t, got.RequiresApproval, "只提交 status 不得改写审批开关")
	assert.Equal(t, before.Name, got.Name)
	assert.Equal(t, before.Category, got.Category)
	assert.Equal(t, before.Description, got.Description)
	assert.Equal(t, before.Icon, got.Icon)
	assert.Equal(t, before.ServiceType, got.ServiceType)
	assert.Equal(t, before.Price, got.Price)
	assert.Equal(t, before.DeliveryTime, got.DeliveryTime)
	assert.Equal(t, before.Unit, got.Unit)
	assert.Equal(t, before.ApprovalLevel, got.ApprovalLevel)
	assert.Equal(t, before.SLAResponseTime, got.SLAResponseTime)
	assert.Equal(t, before.SLAResolutionTime, got.SLAResolutionTime)
	assert.Equal(t, before.SortOrder, got.SortOrder)
}

func TestUpdate_ExplicitFalseAndZeroAreWrittenAndTransmitted(t *testing.T) {
	r, _, _ := scSetup(t)
	id, before := scCreateFull(t, r)
	path := "/api/v1/service-catalogs/" + strconv.Itoa(id)

	w, resp := scDoRaw(t, r, "PUT", path, map[string]any{
		"requiresApproval":  false,
		"approvalLevel":     0,
		"slaResponseTime":   0,
		"slaResolutionTime": 0,
		"sortOrder":         0,
		"description":       "",
		"icon":              "",
		"unit":              "",
	})
	require.Equal(t, 200, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(&resp))

	// 布尔值上的 omitempty 会让「关闭审批」在 JSON 上退化成「没有这个字段」，
	// 前端只能靠硬编码补默认值 —— 这里必须真的出现 false。
	keys := scDataKeys(t, w)
	assert.Equal(t, "false", string(keys["requiresApproval"]), "body=%s", w.Body.String())
	assert.Contains(t, keys, "isActive", "body=%s", w.Body.String())

	got := scGetFull(t, r, id)
	assert.False(t, got.RequiresApproval)
	assert.Zero(t, got.ApprovalLevel)
	assert.Zero(t, got.SLAResponseTime)
	assert.Zero(t, got.SLAResolutionTime)
	assert.Zero(t, got.SortOrder)
	assert.Equal(t, "", got.Description)
	assert.Equal(t, "", got.Icon)
	assert.Equal(t, "", got.Unit)
	// 未提交的关键字段仍然不动
	assert.Equal(t, before.Name, got.Name)
	assert.Equal(t, before.DeliveryTime, got.DeliveryTime)
	assert.Equal(t, before.ServiceType, got.ServiceType)
	assert.Equal(t, before.Price, got.Price)
	assert.Equal(t, before.Status, got.Status)
}

// 表单里的「清除」必须能表达：显式提交空字符串是清空，未提交是不动。
func TestUpdate_ClearsOptionalEnumsOnlyWhenExplicitlyEmptied(t *testing.T) {
	r, _, _ := scSetup(t)
	id, before := scCreateFull(t, r)
	path := "/api/v1/service-catalogs/" + strconv.Itoa(id)

	w, resp := scDoRaw(t, r, "PUT", path, map[string]any{"unit": "", "serviceType": ""})
	require.Equal(t, 200, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(&resp))

	got := scGetFull(t, r, id)
	assert.Equal(t, "", got.Unit)
	assert.Equal(t, "", got.ServiceType)
	assert.Equal(t, before.Description, got.Description)
	assert.Equal(t, before.RequiresApproval, got.RequiresApproval)

	// 只改名称时，刚清空的字段不能被旧值「复活」
	renamed := "Renamed-" + scUID()
	w, resp = scDoRaw(t, r, "PUT", path, map[string]any{"name": renamed})
	require.Equal(t, 200, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(&resp))

	got = scGetFull(t, r, id)
	assert.Equal(t, renamed, got.Name)
	assert.Equal(t, "", got.Unit)
	assert.Equal(t, "", got.ServiceType)
}

func TestUpdate_RejectsInvalidValuesWithFieldSpecificMessage(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]any
		wantMsg string
	}{
		{"交付天数非数字", map[string]any{"deliveryTime": "tomorrow"}, "deliveryTime 必须是正整数天数"},
		{"交付天数为零", map[string]any{"deliveryTime": "0"}, "交付天数必须在 1 到 3650 之间"},
		{"交付天数越界", map[string]any{"deliveryTime": "99999"}, "交付天数必须在 1 到 3650 之间"},
		{"审批级别越界", map[string]any{"approvalLevel": 9}, "审批级别必须在 0 到 3 之间（0 表示未配置）"},
		{"审批级别负数", map[string]any{"approvalLevel": -1}, "审批级别必须在 0 到 3 之间（0 表示未配置）"},
		{"状态非法", map[string]any{"status": "archived"}, "状态只能是 enabled 或 disabled"},
		{"状态清空", map[string]any{"status": ""}, "状态只能是 enabled 或 disabled"},
		{"服务类型非法", map[string]any{"serviceType": "gpu"}, "服务类型只能是 vm/rds/oss/network/storage/security/custom"},
		{"计价单位非法", map[string]any{"unit": "年"}, "计价单位只能是 月/次/用户"},
		{"名称清空", map[string]any{"name": "   "}, "服务名称不能为空"},
		{"分类清空", map[string]any{"category": ""}, "服务分类不能为空"},
		{"SLA 响应负数", map[string]any{"slaResponseTime": -5}, "SLA 响应时限不能为负数"},
		{"SLA 解决负数", map[string]any{"slaResolutionTime": -5}, "SLA 解决时限不能为负数"},
		{"排序负数", map[string]any{"sortOrder": -1}, "排序不能为负数"},
		{"价格负数", map[string]any{"price": -1}, "价格不能为负数"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, _, _ := scSetup(t)
			id, before := scCreateFull(t, r)
			path := "/api/v1/service-catalogs/" + strconv.Itoa(id)

			w, resp := scDoRaw(t, r, "PUT", path, tc.body)
			assert.Equal(t, 400, w.Code, "body=%s", w.Body.String())
			assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustSC(&resp))
			assert.Equal(t, tc.wantMsg, resp.Message, "必须给页面可直接定位的文案")

			// 拒绝后不得留下半写状态
			got := scGetFull(t, r, id)
			assert.Equal(t, before, got)
		})
	}
}

func TestUpdate_NameConflictIs409NotInternalError(t *testing.T) {
	r, _, _ := scSetup(t)
	idA, before := scCreateFull(t, r)
	idB, _ := scCreateFull(t, r)
	require.NotEqual(t, before.Name, scGetFull(t, r, idB).Name, "两条目录不能同名")

	w, resp := scDoRaw(t, r, "PUT", "/api/v1/service-catalogs/"+strconv.Itoa(idB),
		map[string]any{"name": before.Name})
	assert.Equal(t, 409, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, common.ConflictCode, resp.Code, "body=%s", mustSC(&resp))
	assert.Equal(t, "服务目录名称已存在，请换一个名称", resp.Message)
	assert.NotContains(t, strings.ToLower(w.Body.String()), "sql", "驱动错误不得透给页面")

	// 失败的那一行不受影响
	assert.Equal(t, before, scGetFull(t, r, idA))
	assert.NotEqual(t, before.Name, scGetFull(t, r, idB).Name)
}

func TestUpdate_CrossTenantRejectedAndRowUntouched(t *testing.T) {
	r, client, tenantA := scSetup(t)
	id, before := scCreateFull(t, r)

	tenantB := scTenant(t, client)
	require.NotEqual(t, tenantA, tenantB)
	engineB := scEngine(t, client, tenantB)

	w, resp := scDoRaw(t, engineB, "PUT", "/api/v1/service-catalogs/"+strconv.Itoa(id),
		map[string]any{"status": "disabled"})
	assert.Equal(t, 404, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, common.NotFoundErrorCode, resp.Code, "body=%s", mustSC(&resp))

	// A 租户的读写面也拿不到 B 的行
	assert.Equal(t, before, scGetFull(t, r, id))
}

// TestCreate_ApprovalLevelZeroMeansUnconfigured 证明新建目录可以停在「未配置审批」。
// 此前请求 DTO 上的 min=1 把 0 拦成通用「请求参数错误」，页面只能谎报级别或强制选 1。
func TestCreate_ApprovalLevelZeroMeansUnconfigured(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	requires := false

	w, resp := scDoRaw(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "Level0-" + uid, Category: "it_service",
		RequiresApproval: &requires, ApprovalLevel: 0,
	})
	require.Equal(t, 200, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(&resp))

	// 数值 0 会被响应 DTO 的 omitempty 省略，所以这里断言「没有谎报成非 0」。
	keys := scDataKeys(t, w)
	if raw, ok := keys["approvalLevel"]; ok {
		var level int
		require.NoError(t, json.Unmarshal(raw, &level))
		assert.Zero(t, level)
	}
	assert.Equal(t, json.RawMessage("false"), keys["requiresApproval"])
}

// scGetInt 读取响应 data 里的整数字段；响应 DTO 对 0 带 omitempty，键缺席即为 0。
func scGetInt(t *testing.T, w *httptest.ResponseRecorder, key string) int {
	t.Helper()
	raw, ok := scDataKeys(t, w)[key]
	if !ok {
		return 0
	}
	var value int
	require.NoError(t, json.Unmarshal(raw, &value), "key=%s", key)
	return value
}

// TestUpdate_ClearsAssociationsOnlyWhenExplicitlyZero 锁死关联字段的 PATCH 语义。
// 修复前仓储层用 `> 0` 守卫 SetCiTypeID/SetCloudServiceID：显式提交 0 返回 200，
// 旧关联却原样留在库里，页面显示「未关联」而新申请仍按旧 CI 类型履约。
func TestUpdate_ClearsAssociationsOnlyWhenExplicitlyZero(t *testing.T) {
	r, client, tenantID := scSetup(t)
	ctx := context.Background()
	ci, err := client.CIType.Create().SetName("PatchCI-" + scUID()).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	cloud, err := client.CloudService.Create().
		SetProvider("aliyun").SetServiceCode("oss").SetServiceName("OSS").
		SetResourceTypeCode("bucket").SetResourceTypeName("Bucket").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	resp := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "Assoc-" + scUID(), Category: "cloud", CITypeID: ci.ID, CloudServiceID: cloud.ID,
	})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	id := int(resp.Data.(map[string]interface{})["id"].(float64))
	path := "/api/v1/service-catalogs/" + strconv.Itoa(id)

	w, _ := scDoRaw(t, r, "GET", path, nil)
	require.Equal(t, ci.ID, scGetInt(t, w, "ciTypeId"))
	require.Equal(t, cloud.ID, scGetInt(t, w, "cloudServiceId"))

	// 只解绑 CI 会让已关联的云服务失去归属：必须拒绝，且一行都不改。
	w, denied := scDoRaw(t, r, "PUT", path, map[string]any{"ciTypeId": 0})
	require.Equal(t, 400, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.ParamErrorCode, denied.Code)
	w, _ = scDoRaw(t, r, "GET", path, nil)
	require.Equal(t, ci.ID, scGetInt(t, w, "ciTypeId"), "被拒绝的 PATCH 不得留下半成品")
	require.Equal(t, cloud.ID, scGetInt(t, w, "cloudServiceId"))

	// 成对清空才是合法表达，并且必须真的落到库里。
	w, cleared := scDoRaw(t, r, "PUT", path, map[string]any{"ciTypeId": 0, "cloudServiceId": 0})
	require.Equal(t, 200, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.SuccessCode, cleared.Code, "body=%s", mustSC(&cleared))
	require.Equal(t, 0, scGetInt(t, w, "ciTypeId"))
	require.Equal(t, 0, scGetInt(t, w, "cloudServiceId"))

	// 关联字段恒定出现在响应里：客户端把 GET 的整份对象回贴成 PATCH 时，
	// 「未关联」必须带着 0 回来，省略字段会让 PATCH 变成「不修改」，解除关联就永远做不到。
	keys := scDataKeys(t, w)
	require.Contains(t, keys, "ciTypeId", "未关联必须显式返回 0，不能靠字段缺席")
	require.Contains(t, keys, "cloudServiceId")

	w, _ = scDoRaw(t, r, "GET", path, nil)
	require.Equal(t, 0, scGetInt(t, w, "ciTypeId"), "响应说清除了，库里也必须是 0")
	require.Equal(t, 0, scGetInt(t, w, "cloudServiceId"))
	require.Contains(t, scDataKeys(t, w), "ciTypeId")
	require.Contains(t, scDataKeys(t, w), "cloudServiceId")
}

// TestCreate_ValidationMessagesComeFromService 锁定创建路径的中文契约。
// 此前 handler 里另写了一份「关联云服务必须选 CI 类型」并硬编码 code 1001，
// 和 service 的同名规则只差一个空格；交付天数超限则直接把英文原文回给用户。
func TestCreate_ValidationMessagesComeFromService(t *testing.T) {
	r, client, tenantID := scSetup(t)
	ctx := context.Background()
	cloud, err := client.CloudService.Create().
		SetProvider("aliyun").SetServiceCode("oss").SetServiceName("OSS").
		SetResourceTypeCode("bucket").SetResourceTypeName("Bucket").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	w, resp := scDoRaw(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "Msg-" + scUID(), Category: "cloud", CloudServiceID: cloud.ID,
	})
	require.Equal(t, 400, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.ParamErrorCode, resp.Code)
	assert.Equal(t, "关联云服务时必须选择 CI 类型", resp.Message)

	w, resp = scDoRaw(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "Msg-" + scUID(), Category: "cloud", DeliveryTime: "4000",
	})
	require.Equal(t, 400, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.ParamErrorCode, resp.Code)
	assert.Equal(t, "交付天数必须在 1 到 3650 之间", resp.Message)
}
