package capability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNonGADependencies(t *testing.T) {
	defs := []definition{
		{key: "ga", maturity: MaturityGA},
		{key: "pilot", maturity: MaturityPilot},
		{key: "off", maturity: MaturityDisabled},
	}

	require.Empty(t, nonGADependencies([]string{"ga"}, defs), "GA 依赖不构成传导风险")
	require.Equal(t, []string{"off", "pilot"},
		nonGADependencies([]string{"pilot", "off", "pilot", "ga"}, defs),
		"pilot/disabled 必须上榜，且去重排序保证响应稳定")
	require.Equal(t, []string{"missing"},
		nonGADependencies([]string{"missing"}, defs),
		"registry 里查不到的依赖不得默认安全（fail-closed）")
	require.NotNil(t, nonGADependencies(nil, defs), "空依赖必须返回空切片而非 nil")
}

// TestHandlerDisclosesPilotDependencyRisks 是依赖成熟度传导的运行时回归锁：
// 旧实现把 dependencies 原样下发、不做传导，GA 的 incident/problem/change
// 仍报 "ga" 却跑在 Pilot 的 workflow 上。删掉 handler 里的 dependencyRisks
// 赋值（或改回硬编码空切片）本用例必红，且不依赖编译失败。
func TestHandlerDisclosesPilotDependencyRisks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/capabilities", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("role", "admin")
		NewHandler(nil)(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.NotEmpty(t, response.Data.Items)

	byKey := make(map[string]Capability, len(response.Data.Items))
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}

	// 业务语义：三个 GA 能力都依赖仍是 Pilot 的编排层，矩阵不得让 "ga" 掩盖它。
	for _, key := range []string{"incident", "problem", "change"} {
		require.Equal(t, MaturityGA, byKey[key].Maturity, "%s 自身成熟度不变", key)
		require.Contains(t, byKey[key].DependencyRisks, "workflow",
			"%s 依赖的 workflow 仍是 Pilot，必须如实披露", key)
	}
	require.Equal(t, []string{"workflow"}, byKey["change"].DependencyRisks)
	require.Empty(t, byKey["ticket"].DependencyRisks, "无依赖的能力不得有噪声")
	require.Empty(t, byKey["cmdb.configurationItems"].DependencyRisks, "依赖全 GA 时不得误报")
	require.Equal(t, []string{"workflow"}, byKey["serviceRequest"].DependencyRisks,
		"Pilot 能力自身也要披露其未验收依赖")

	// 全量一致性：每个条目下发的风险集必须等于纯函数对 registry 的计算结果，
	// 防止只给几个样例硬编码而其余条目悄悄退回「不传导」。
	for _, def := range registry {
		want := nonGADependencies(def.dependencies, registry)
		got := byKey[def.key].DependencyRisks
		if len(want) == 0 {
			require.Empty(t, got, "%s 不应下发依赖风险", def.key)
			continue
		}
		require.Equal(t, want, got, "%s 的依赖风险集必须与 registry 传导结果一致", def.key)
	}
}
