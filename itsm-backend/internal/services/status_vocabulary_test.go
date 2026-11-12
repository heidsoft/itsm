package service

import (
	"testing"

	"itsm-backend/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChangeStatusMachinesMatchVocabulary 锁定三种变更类型的迁移表与数据库
// CHECK 词表一致。
//
// changes.status 是唯一同时存在 pending（dto/API）与 submitted（service
// 状态机）两个待审批写法的列：normalizeChangeTransitionStatus 在入口归一，
// 但迁移表本身用 submitted，数据库两种都接受（存量数据两种都有）。
// 本测试保证归一逻辑之外的任何新增值都必须先进 CHECK 词表。
func TestChangeStatusMachinesMatchVocabulary(t *testing.T) {
	machines := map[string]map[string][]string{
		"base":      changeBaseTransitions,
		"standard":  changeStandardTransitions,
		"emergency": changeEmergencyTransitions,
		"normal":    changeNormalTransitions,
	}

	for name, transitions := range machines {
		require.NotEmpty(t, transitions, "%s 变更状态机迁移表为空", name)
		for from, targets := range transitions {
			assert.True(t, common.StatusValueAllowed("changes", "status", from),
				"%s 变更状态机以 %q 为源状态，但该值不在 CHECK 词表内", name, from)
			for _, to := range targets {
				assert.True(t, common.StatusValueAllowed("changes", "status", to),
					"%s 变更状态机允许迁移到 %q，但该值不在 CHECK 词表内（生产会 23514）", name, to)
			}
		}
	}
}

// TestChangePendingAndSubmittedAreBothAccepted 记录 pending/submitted 的
// 双写法事实：两者都必须被 CHECK 接受，否则存量变更读取/迁移会失败。
func TestChangePendingAndSubmittedAreBothAccepted(t *testing.T) {
	for _, v := range []string{"pending", "submitted"} {
		assert.True(t, common.StatusValueAllowed("changes", "status", v),
			"changes.status 必须同时接受 %q（pending=API 词表，submitted=状态机词表）", v)
	}
}
