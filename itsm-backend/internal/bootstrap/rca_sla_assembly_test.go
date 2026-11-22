package bootstrap

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRootCauseAnalysisServiceReceivesSLAService 锁定「事件转问题绑 SLA」的装配。
//
// 为什么需要源码级守卫：内部服务漏注入时编译、启动、以及「事件转问题」的成功用例
// 全都绿——CreateProblemFromIncident 里 slaSvc 为 nil 只是静默跳过绑定。唯一的
// 症状出现在产品语义上：问题管理页新建的问题 GET /problems/:id/sla 返回 bound=true，
// 事件详情页转出来的问题恒 bound=false。这种「两套事实」只有在真跑两条路径并比对
// 才看得见，因此把装配本身钉在测试里，让删这行的人必须解释一次。
func TestRootCauseAnalysisServiceReceivesSLAService(t *testing.T) {
	source, err := os.ReadFile("app.go")
	require.NoError(t, err)
	require.NotEmpty(t, source)

	body := functionSource(t, source, "NewApplication")

	// 必须把同一个 ticketSLAService 实例注入：另外构造一个 TicketSLAService
	// 会丢掉 SetSLAStore 注入的 store，绑定路径直接短路（SLAStore() 为 nil）。
	for _, want := range []string{
		"rootCauseAnalysisService.SetSLAService(ticketSLAService)",
		"ticketSLAService.SetSLAStore(slaStore)",
		"problemServiceDomain.SetSLAService(ticketSLAService)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("app.go 缺少装配 %q：事件转问题与问题管理页新建会给出两套 SLA 事实", want)
		}
	}

	// 反向：不得为 RCA 另造 SLA 服务实例（上面的 require 已钉住正解，
	// 这里防止后来者「顺手」再 new 一个把正解覆盖掉）。
	if strings.Contains(body, "rootCauseAnalysisService.SetSLAService(service.NewTicketSLAService") {
		t.Error("RCA 的 SLA 注入必须是共享的 ticketSLAService 实例，不能另建（会丢掉注入的 sla.Store）")
	}
}
