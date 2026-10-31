package service_request

import (
	"testing"

	"itsm-backend/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceRequestStatusMachineMatchesVocabulary 锁定服务请求运营状态机与
// 数据库 CHECK 词表一致。
//
// 注意：ent/schema/servicerequest.go 的字段注释写的是
// "submitted|approved|rejected|in_progress|completed|cancelled"，
// 与实际词表（审批链 9 态）不符——那段注释是错的，以 domain/servicerequest
// 常量与 common.StatusVocabulary 为准。本测试防止再按错误注释改状态机。
func TestServiceRequestStatusMachineMatchesVocabulary(t *testing.T) {
	require.NotEmpty(t, serviceRequestStatusTransitions)

	for from, targets := range serviceRequestStatusTransitions {
		assert.True(t, common.StatusValueAllowed("service_requests", "status", from),
			"服务请求状态机以 %q 为源状态，但该值不在 CHECK 词表内", from)
		for to := range targets {
			assert.True(t, common.StatusValueAllowed("service_requests", "status", to),
				"服务请求状态机允许迁移到 %q，但该值不在 CHECK 词表内（生产会 23514）", to)
		}
	}
}
