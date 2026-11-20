package service_request

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// readbackFailRepo 包装真实仓储，制造「写成功后读回失败」的窄窗口故障：
// Update 内部先 GetWithApprovals 读旧值（放行），再 Update 落库（放行后置位），
// 之后 handler 的读回 Get → GetWithApprovals 命中故障返回错误。
// 这精确模拟了 DB 瞬时故障：写已提交，但读回拿不到——正是假成功的高发场景。
type readbackFailRepo struct {
	Repository
	mu       sync.Mutex
	failNext bool
}

func (r *readbackFailRepo) Update(ctx context.Context, req *ServiceRequest) error {
	err := r.Repository.Update(ctx, req)
	if err == nil {
		r.mu.Lock()
		r.failNext = true
		r.mu.Unlock()
	}
	return err
}

func (r *readbackFailRepo) GetWithApprovals(ctx context.Context, id, tenantID int) (*ServiceRequest, []*ServiceRequestApproval, error) {
	r.mu.Lock()
	fail := r.failNext
	r.mu.Unlock()
	if fail {
		return nil, nil, errors.New("injected read-back failure after commit")
	}
	return r.Repository.GetWithApprovals(ctx, id, tenantID)
}

// TestServiceRequestHandler_Update_ReadbackFailureIsNotFakeSuccess 回归：
// Update 端点写成功后读回 Get 的错误曾被 `_` 吞掉，toDTO(nil) 返回 nil →
// HTTP 200 且 data=null 的假成功。客户端以为更新成功却拿不到任何实体，
// 也无法判断审批列表是否已刷新。修复后必须与 Create 端点同口径：
// 读回失败时退回 Update 返回的权威实体（updated），保证 data 有主数据。
func TestServiceRequestHandler_Update_ReadbackFailureIsNotFakeSuccess(t *testing.T) {
	baseRouter, client, tenantID, uid, catID := srSetup(t)
	id := srCreateOne(t, baseRouter, catID)

	realRepo := NewEntRepository(client)
	wrapped := &readbackFailRepo{Repository: realRepo}
	logger := zaptest.NewLogger(t).Sugar()
	svc := newTestService(wrapped, nil, nil, client, logger, nil)
	h := NewHandler(svc)

	r := gin.New()
	r.Use(srAuth(tenantID, uid))
	r.PUT("/api/v1/service-requests/:id", h.Update)

	path := "/api/v1/service-requests/" + strconv.Itoa(id)
	code, resp := srDoHTTPReq(t, r, http.MethodPut, path,
		dto.UpdateServiceRequestRequest{Title: "读回失败也要返回实体"})

	require.Equal(t, common.SuccessCode, resp.Code, "写已提交，仍应成功响应: body=%s", srStr(resp))
	assert.Equal(t, http.StatusOK, code)
	require.NotNil(t, resp.Data, "读回失败不得返回 data=null 的假成功: body=%s", srStr(resp))
	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "data 必须是实体对象: %T", resp.Data)
	assert.Equal(t, "读回失败也要返回实体", data["title"], "必须退回 Update 返回的权威实体")
	assert.Equal(t, float64(id), data["id"], "实体 id 必须与更新目标一致")

	// 数据库里的写入确实生效——确认不是「只在响应里伪造」。
	got, err := realRepo.Get(context.Background(), id, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "读回失败也要返回实体", got.Title)
}
