package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ticketRefMockClient 用 sqlmock 构造 Ent client，注入自定义字段引用校验的 DB 故障。
func ticketRefMockClient(t *testing.T) (*ent.Client, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	return client, mock
}

// ticketRefCases 覆盖 3 类引用校验字段：用户 / 部门 / CI。
var ticketRefCases = []struct {
	name  string
	field dto.CustomFieldDefinition
	value interface{}
}{
	{
		name:  "用户",
		field: dto.CustomFieldDefinition{Label: "负责人", Type: dto.CustomFieldTypeUser},
		value: 42,
	},
	{
		name:  "部门",
		field: dto.CustomFieldDefinition{Label: "归属部门", Type: dto.CustomFieldTypeDepartment},
		value: 7,
	},
	{
		name:  "CI",
		field: dto.CustomFieldDefinition{Label: "关联配置项", Type: dto.CustomFieldTypeCI},
		value: 13,
	},
}

// TestValidateCustomFieldValue_DBErrorIsNotReportedAsNotFound 是「撤掉守卫即失败」的回归锁：
// 旧实现用 `exists, _ := ...Exist(ctx)` 吞掉 DB 错误，err!=nil 时 exists=false，
// 于是把服务端 DB 故障误报成 ParamErrorCode(1001) 的「字段 X 的用户/部门/CI 不存在
// 或无权引用」，既误导客户端改参数重试，又丢失 driver 错误日志。
// 正确语义：DB 故障必须包装成 500 类错误（classifyError → 5001），底层错误原样上抛。
func TestValidateCustomFieldValue_DBErrorIsNotReportedAsNotFound(t *testing.T) {
	for _, tc := range ticketRefCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := ticketRefMockClient(t)
			injected := errors.New("injected DB failure for " + tc.name)
			mock.ExpectQuery("SELECT .*LIMIT 1").WillReturnError(injected)

			svc := &TicketService{client: client}
			err := svc.validateCustomFieldValue(context.Background(), 1, tc.field, tc.value)

			require.Error(t, err, "DB 故障不得被吞成校验通过")
			assert.ErrorIs(t, err, injected, "底层 driver 错误必须原样上抛（%w），不能丢失")
			var bizErr *common.BusinessError
			assert.False(t, errors.As(err, &bizErr) && bizErr.Code == common.ParamErrorCode,
				"绝不能把 DB 故障伪装成 1001『不存在或无权引用』的参数错误")

			// 走生产同款 RespondError 出口：未分类的 driver 错误必须落为 5001/500。
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tickets", nil)
			common.RespondError(c, err, "创建工单失败")
			assert.Equal(t, http.StatusInternalServerError, w.Code, "DB 故障的 HTTP 状态必须是 500")
			var resp common.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, common.InternalErrorCode, resp.Code, "业务码必须是 5001")
			assert.NotContains(t, resp.Message, "不存在或无权引用", "响应文案不得误导客户端改参数")
		})
	}
}

// TestValidateCustomFieldValue_GenuineNotFoundStillYieldsParamError 锁定正确分支不被过度修正：
// DB 查询成功但记录确实不存在（或跨租户）时，仍应返回 1001「不存在或无权引用」。
func TestValidateCustomFieldValue_GenuineNotFoundStillYieldsParamError(t *testing.T) {
	for _, tc := range ticketRefCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := ticketRefMockClient(t)
			// Exist() 内部走 FirstID：零行 → NotFoundError → false,nil；
			// 不是返回一个含 false 的行。
			mock.ExpectQuery("SELECT .*LIMIT 1").
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			svc := &TicketService{client: client}
			err := svc.validateCustomFieldValue(context.Background(), 1, tc.field, tc.value)

			require.Error(t, err)
			var bizErr *common.BusinessError
			require.ErrorAs(t, err, &bizErr, "真实不存在应返回 BusinessError")
			assert.Equal(t, common.ParamErrorCode, bizErr.Code)
			assert.Contains(t, bizErr.Message, "不存在或无权引用")
		})
	}
}
