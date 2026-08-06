package integration

// TestDynamicFields 端到端集成测试：动态自定义字段（field_definitions / field_values）
// 的完整链路 —— 模板创建 -> 工单提交 -> 工单详情回显 -> 工单列表不携带自定义字段。
//
// 这是本次 dynamic-custom-fields 计划的最后一道防线：此前同一项目里发生过
// "三个独立的 Ticket -> DTO 转换函数彼此走漏" 的事故，只有真正打到 HTTP 路由层
// 的测试才能发现——单纯的 service 层单测各自 mock，测不出三份拼接逻辑分叉。
// Task 4 已经把三份转换收敛成 service.ToTicketResponse /
// ToTicketResponseWithCustomFields 两个函数（分别对应"列表，不查字段值，避免
// N+1"和"详情/创建，查一次字段值"），这个测试就是钉住这条设计决策不被后续修改
// 悄悄破坏。
//
// 脚手架说明：仓库里 itsm-backend/tests/integration/ 目录在本测试新增之前并不
// 存在（历史上只有顶层 itsm-backend/integration/，且那里的用例只直接操作 ent
// client，完全不经过 gin router）。真正"起一个真实 gin router + 真实
// controller/service + httptest 打请求 + 解析 JSON 响应"的既有范式来自
// controller/ticket_controller_test.go 的 setupTestTicketController：用一个
// 轻量 middleware 把 tenant_id/user_id/role 直接注入 gin.Context（不跑完整的
// JWT/RBAC 中间件链），然后挂载真实的 TicketController + 真实
// service.NewTicketServiceForTest（backed by enttest sqlite）。这个测试沿用同
// 一种搭法，因为它才是唯一能触发 controller 里实际 DTO 映射代码路径的现有模式；
// tests/contract 和 tests/rbac 目录下的用例则是手写内联 handler 模拟响应，并不
// 会经过真实 controller，测不出映射分叉问题。
import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// apiEnvelope 对应 common.Response 的 {code, message, data} 信封。用
// json.RawMessage 延迟解析 data，方便每一步用不同的目标类型解码。
type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupDynamicFieldsRouter(t *testing.T) (*gin.Engine, *ent.Client, *ent.Tenant, *ent.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:dynamic_fields_e2e?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()

	ticketService := service.NewTicketServiceForTest(client, logger)
	ticketController := controller.NewTicketController(ticketService, nil, nil, client, logger)

	ctx := t.Context()
	tenant, err := client.Tenant.Create().
		SetName("Dynamic Fields Tenant").
		SetCode("DYNFIELDS").
		SetDomain("dynfields.test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("dynfields_user").
		SetEmail("dynfields@test.com").
		SetPasswordHash("hashedpassword").
		SetName("Dynamic Fields User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetRole("admin").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		// 跟随 controller/ticket_controller_test.go 的既有范式：直接把
		// tenant_id/user_id/role 塞进 gin.Context，绕开完整的 JWT/RBAC
		// 中间件链——这个测试要验证的是 controller -> service -> DTO 的
		// HTTP 层映射，不是认证/鉴权本身（那部分有 tests/rbac 单独覆盖）。
		c.Set("tenant_id", tenant.ID)
		c.Set("user_id", user.ID)
		c.Set("role", "admin") // admin -> DataScopeAll，列表查询不会被行级权限收窄
		c.Next()
	})

	r.POST("/api/v1/tickets/templates", ticketController.CreateTicketTemplate)
	r.POST("/api/v1/tickets", ticketController.CreateTicket)
	r.GET("/api/v1/tickets/:id", ticketController.GetTicket)
	r.GET("/api/v1/tickets", ticketController.ListTickets)

	return r, client, tenant, user
}

func doDynamicFieldsRequest(t *testing.T, r http.Handler, method, path string, body interface{}) (apiEnvelope, int) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var env apiEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "response body=%s", w.Body.String())
	return env, w.Code
}

// TestDynamicFields 跑通"模板(2个字段) -> 工单创建(提交2个字段值) -> 工单详情
// (customFields 长度2，label 非空) -> 工单列表(不带 customFields key)"的完整
// HTTP 路由链路。
func TestDynamicFields(t *testing.T) {
	r, client, _, _ := setupDynamicFieldsRouter(t)
	defer client.Close()

	// ---- Step 1: 创建带 2 个字段定义的工单模板 ----
	type templateFieldReq struct {
		Name     string `json:"name"`
		Label    string `json:"label"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
	}
	createTemplateReq := struct {
		Name        string             `json:"name"`
		Description string             `json:"description"`
		Category    string             `json:"category"`
		Priority    string             `json:"priority"`
		Fields      []templateFieldReq `json:"fields"`
	}{
		Name:        "动态字段集成测试模板",
		Description: "端到端集成测试专用模板",
		Category:    "incident",
		Priority:    "medium",
		Fields: []templateFieldReq{
			{Name: "department", Label: "部门", Type: "text", Required: true},
			{Name: "urgencyReason", Label: "紧急原因", Type: "text", Required: false},
		},
	}

	env, status := doDynamicFieldsRequest(t, r, http.MethodPost, "/api/v1/tickets/templates", createTemplateReq)
	require.Equal(t, http.StatusOK, status, "创建模板应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code, "创建模板 code 应为成功, message=%s", env.Message)

	var templateResp struct {
		ID     int                      `json:"id"`
		Fields []map[string]interface{} `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &templateResp))
	require.Len(t, templateResp.Fields, 2, "模板响应里的 fields 长度应为 2")
	require.NotZero(t, templateResp.ID)
	templateID := templateResp.ID

	// ---- Step 2: 用该模板 ID 创建工单，提交 2 个字段值 ----
	createTicketReq := struct {
		Title       string                 `json:"title"`
		Description string                 `json:"description"`
		Priority    string                 `json:"priority"`
		TemplateID  int                    `json:"templateId"`
		FormFields  map[string]interface{} `json:"formFields"`
	}{
		Title:       "打印机无法连接网络",
		Description: "三楼打印机从今天早上开始无法连接到公司网络，需要紧急处理。",
		Priority:    "medium",
		TemplateID:  templateID,
		FormFields: map[string]interface{}{
			"presetTypeId": "incident",
			"values": map[string]interface{}{
				"department":    "IT",
				"urgencyReason": "影响全楼层打印",
			},
		},
	}

	env, status = doDynamicFieldsRequest(t, r, http.MethodPost, "/api/v1/tickets", createTicketReq)
	require.Equal(t, http.StatusOK, status, "创建工单应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code, "创建工单 code 应为成功, message=%s", env.Message)

	var createdTicket struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &createdTicket))
	require.NotZero(t, createdTicket.ID)
	ticketID := createdTicket.ID

	// ---- Step 3: 获取工单详情，断言 customFields 长度2，且每项 label 非空 ----
	env, status = doDynamicFieldsRequest(t, r, http.MethodGet, "/api/v1/tickets/"+strconv.Itoa(ticketID), nil)
	require.Equal(t, http.StatusOK, status, "获取工单详情应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code)

	var ticketDetail struct {
		ID           int `json:"id"`
		CustomFields []struct {
			Name  string      `json:"name"`
			Label string      `json:"label"`
			Value interface{} `json:"value"`
		} `json:"customFields"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &ticketDetail))
	require.Len(t, ticketDetail.CustomFields, 2, "工单详情 customFields 长度应为 2")
	for _, f := range ticketDetail.CustomFields {
		assert.NotEmpty(t, f.Label, "customFields[%q] 的 label 不应为空（不能只是原始 name）", f.Name)
		assert.NotEqual(t, f.Name, f.Label, "customFields[%q] 的 label 不应该退化成原始字段名", f.Name)
	}

	// ---- Step 4: 获取工单列表，断言列表项不带 customFields key ----
	env, status = doDynamicFieldsRequest(t, r, http.MethodGet, "/api/v1/tickets?page=1&pageSize=20&templateId="+strconv.Itoa(templateID), nil)
	require.Equal(t, http.StatusOK, status, "获取工单列表应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code)

	var listResp struct {
		Tickets []map[string]interface{} `json:"tickets"`
		Total   int                      `json:"total"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &listResp))
	require.NotEmpty(t, listResp.Tickets, "列表里应该能查到刚创建的工单")

	found := false
	for _, item := range listResp.Tickets {
		idVal, _ := item["id"].(float64)
		if int(idVal) == ticketID {
			found = true
		}
		_, hasCustomFields := item["customFields"]
		assert.False(t, hasCustomFields, "列表响应项不应该带 customFields key（Task 4 的设计决策：列表不查字段值，避免 N+1）")
	}
	assert.True(t, found, "列表里应该包含 Step 2 创建的工单")
}

// TestDynamicFields_ArrayFormatWithUnderscoreNames 跑通 Task 11/12 修的那个 bug 本身：
// 前端 http-client.ts 会对请求体的 object key 做一次全局 snake_case->camelCase 转换，
// 如果 formFields.values 用 map 形状传（字段名是 object key），带下划线的字段名会被
// 悄悄改写、值静默丢失。Task 12 把 values 改成 [{name,value}] 数组形状规避这个问题——
// 这里用真实的 HTTP body（而不是 service 层直接构造 Go map）走一遍模板 -> 工单创建
// -> 详情回显，确认数组形状的带下划线字段名真的能落库、能读回。
func TestDynamicFields_ArrayFormatWithUnderscoreNames(t *testing.T) {
	r, client, _, _ := setupDynamicFieldsRouter(t)
	defer client.Close()

	type templateFieldReq struct {
		Name     string `json:"name"`
		Label    string `json:"label"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
	}
	createTemplateReq := struct {
		Name        string             `json:"name"`
		Description string             `json:"description"`
		Category    string             `json:"category"`
		Priority    string             `json:"priority"`
		Fields      []templateFieldReq `json:"fields"`
	}{
		Name:        "下划线字段名集成测试模板",
		Description: "验证数组形状的 values 能在真实 HTTP 请求体里存活",
		Category:    "incident",
		Priority:    "medium",
		Fields: []templateFieldReq{
			{Name: "current_replicas", Label: "当前副本数", Type: "number", Required: true},
		},
	}

	env, status := doDynamicFieldsRequest(t, r, http.MethodPost, "/api/v1/tickets/templates", createTemplateReq)
	require.Equal(t, http.StatusOK, status, "创建模板应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code)

	var templateResp struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &templateResp))
	templateID := templateResp.ID

	// formFields.values 用 [{name,value}] 数组形状提交，name 是数组元素里的字符串值
	// 而不是 object key，所以不会被请求体的驼峰转换改写。
	createTicketReq := struct {
		Title       string      `json:"title"`
		Description string      `json:"description"`
		Priority    string      `json:"priority"`
		TemplateID  int         `json:"templateId"`
		FormFields  interface{} `json:"formFields"`
	}{
		Title:       "扩容申请",
		Description: "生产环境副本数需要扩容",
		Priority:    "medium",
		TemplateID:  templateID,
		FormFields: map[string]interface{}{
			"presetTypeId": "incident",
			"values": []map[string]interface{}{
				{"name": "current_replicas", "value": 5},
			},
		},
	}

	env, status = doDynamicFieldsRequest(t, r, http.MethodPost, "/api/v1/tickets", createTicketReq)
	require.Equal(t, http.StatusOK, status, "创建工单应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code, "创建工单 code 应为成功, message=%s", env.Message)

	var createdTicket struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &createdTicket))
	ticketID := createdTicket.ID

	env, status = doDynamicFieldsRequest(t, r, http.MethodGet, "/api/v1/tickets/"+strconv.Itoa(ticketID), nil)
	require.Equal(t, http.StatusOK, status, "获取工单详情应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code)

	var ticketDetail struct {
		CustomFields []struct {
			Name  string      `json:"name"`
			Label string      `json:"label"`
			Value interface{} `json:"value"`
		} `json:"customFields"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &ticketDetail))
	require.Len(t, ticketDetail.CustomFields, 1, "工单详情 customFields 长度应为 1")
	assert.Equal(t, "current_replicas", ticketDetail.CustomFields[0].Name, "下划线字段名应该完整存活，不被驼峰转换破坏")
	assert.EqualValues(t, 5, ticketDetail.CustomFields[0].Value)
}

// TestDynamicFields_AdHocFieldsWithoutTemplate 跑通无模板（静态预设）的即席字段值路径：
// 前端对静态预设的工单类型没有对应的 field_definitions 行，会在 formFields.fieldDefs
// 里内联提交 {name,label} 定义，配合 formFields.values 一起交给
// extractAdHocFieldValues / CreateAdHocValues。这条路径完全不经过 TemplateID 分支，
// Task 11 新增，此前没有端到端测试覆盖过真实 HTTP 请求体。
func TestDynamicFields_AdHocFieldsWithoutTemplate(t *testing.T) {
	r, client, _, _ := setupDynamicFieldsRouter(t)
	defer client.Close()

	createTicketReq := struct {
		Title       string      `json:"title"`
		Description string      `json:"description"`
		Priority    string      `json:"priority"`
		FormFields  interface{} `json:"formFields"`
	}{
		Title:       "静态预设即席字段申请",
		Description: "没有数据库模板，字段定义随请求内联提交",
		Priority:    "medium",
		FormFields: map[string]interface{}{
			"presetTypeId": "incident",
			"fieldDefs": []map[string]interface{}{
				{"name": "affected_region", "label": "受影响地域"},
			},
			"values": []map[string]interface{}{
				{"name": "affected_region", "value": "cn-north"},
			},
		},
	}

	env, status := doDynamicFieldsRequest(t, r, http.MethodPost, "/api/v1/tickets", createTicketReq)
	require.Equal(t, http.StatusOK, status, "创建工单应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code, "创建工单 code 应为成功, message=%s", env.Message)

	var createdTicket struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &createdTicket))
	require.NotZero(t, createdTicket.ID)

	env, status = doDynamicFieldsRequest(t, r, http.MethodGet, "/api/v1/tickets/"+strconv.Itoa(createdTicket.ID), nil)
	require.Equal(t, http.StatusOK, status, "获取工单详情应返回200, message=%s", env.Message)
	require.Equal(t, 0, env.Code)

	var ticketDetail struct {
		CustomFields []struct {
			Name  string      `json:"name"`
			Label string      `json:"label"`
			Value interface{} `json:"value"`
		} `json:"customFields"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &ticketDetail))
	require.Len(t, ticketDetail.CustomFields, 1, "即席字段值应该被保存并在详情里回显")
	assert.Equal(t, "affected_region", ticketDetail.CustomFields[0].Name)
	assert.Equal(t, "受影响地域", ticketDetail.CustomFields[0].Label, "label 应该来自 fieldDefs 里内联提交的定义")
	assert.Equal(t, "cn-north", ticketDetail.CustomFields[0].Value)
}
