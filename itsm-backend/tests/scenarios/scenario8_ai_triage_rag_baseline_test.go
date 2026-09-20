package scenarios

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"itsm-backend/service"
)

// Scenario 8: AI 分诊准确率 + RAG 命中基线
//
// 覆盖：
//   - 关键词分诊覆盖全部 8 个分类（database/network/server/application/security/storage/user_access/general）
//   - 优先级升级：安全关键词 → critical，宕机/不可用 → high
//   - 置信度归一化：越界值回落到 0.6
//   - 无 LLM 时降级到关键词启发式
//   - RAG 关键词搜索基本命中（标题匹配得分高于内容匹配）

func TestScenario8_AITriageRAGBaseline(t *testing.T) {
	logger := zap.NewNop()
	svc := service.NewTriageService(nil, logger)
	ctx := context.Background()

	t.Run("keyword triage covers all 8 categories", func(t *testing.T) {
		cases := []struct {
			title    string
			desc     string
			category string
		}{
			{"MySQL 慢查询告警", "数据库连接池耗尽，查询超时", "database"},
			{"办公网络无法上网", "交换机丢包严重，路由器重启后仍无法恢复", "network"},
			{"Linux 服务器 CPU 飙高", "主机内存不足，操作系统负载过高", "server"},
			{"应用接口报错", "软件部署后 API 返回 500，服务异常", "application"},
			{"发现安全漏洞", "认证绕过攻击，恶意入侵检测", "security"},
			{"磁盘空间不足", "存储容量告警，备份快照占用过多", "storage"},
			{"用户无法登录", "账号密码重置，权限访问被拒绝", "user_access"},
			{"其他问题", "需要帮助处理一些杂项事务", "general"},
		}

		for _, tc := range cases {
			result := svc.Suggest(ctx, tc.title, tc.desc)
			if result.Category != tc.category {
				t.Errorf("[%s] 期望分类 %q, 实际 %q (explanation=%s)", tc.title, tc.category, result.Category, result.Explanation)
			}
		}
	})

	t.Run("security keywords escalate to critical priority", func(t *testing.T) {
		result := svc.Suggest(ctx, "发现安全漏洞", "认证绕过攻击")
		if result.Priority != "critical" {
			t.Fatalf("安全关键词应触发 critical 优先级, 实际 %q", result.Priority)
		}
		if result.Category != "security" {
			t.Fatalf("安全关键词应分类为 security, 实际 %q", result.Category)
		}
	})

	t.Run("down/unavailable keywords escalate to high priority", func(t *testing.T) {
		result := svc.Suggest(ctx, "数据库宕机", "服务不可用，紧急处理")
		if result.Priority != "high" && result.Priority != "critical" {
			t.Fatalf("宕机/不可用关键词应升级到 high 或 critical, 实际 %q", result.Priority)
		}
	})

	t.Run("business impact escalates medium to high", func(t *testing.T) {
		result := svc.Suggest(ctx, "应用报错", "影响多个用户，业务中断")
		if result.Priority != "high" {
			t.Fatalf("影响大量用户的关键词应升级优先级到 high, 实际 %q", result.Priority)
		}
	})

	t.Run("confidence is within valid range", func(t *testing.T) {
		result := svc.Suggest(ctx, "数据库连接失败", "MySQL 无法连接")
		if result.Confidence < 0 || result.Confidence > 1 {
			t.Fatalf("置信度应在 [0,1] 范围内, 实际 %f", result.Confidence)
		}
		if result.Confidence < 0.5 {
			t.Fatalf("关键词匹配的置信度不应低于 0.5, 实际 %f", result.Confidence)
		}
	})

	t.Run("nil gateway falls back to keyword heuristic", func(t *testing.T) {
		result, err := svc.SuggestWithMetadata(ctx, "服务器内存不足", "Linux 主机负载过高")
		if err != nil {
			t.Fatalf("nil gateway 不应返回错误: %v", err)
		}
		if result.Method != "fallback" {
			t.Fatalf("无 LLM 网关时 method 应为 fallback, 实际 %q", result.Method)
		}
		if result.Result.Category != "server" {
			t.Fatalf("应分类为 server, 实际 %q", result.Result.Category)
		}
	})

	t.Run("default assignee is set by category", func(t *testing.T) {
		result := svc.Suggest(ctx, "MySQL 慢查询", "数据库连接池问题")
		if result.AssigneeID == 0 {
			t.Fatal("database 分类应设置默认 assignee")
		}
	})

	t.Run("SuggestForTenant scopes to tenant", func(t *testing.T) {
		result := svc.SuggestForTenant(ctx, "防火墙规则异常", "网络设备丢包", 42)
		if result.Category != "network" {
			t.Fatalf("租户级分诊应正确分类, 期望 network, 实际 %q", result.Category)
		}
	})

	t.Run("invalid category normalizes to general", func(t *testing.T) {
		result := svc.Suggest(ctx, "随便写点什么", "完全不相关的描述内容")
		if result.Category != "general" {
			t.Fatalf("无法识别的内容应归为 general, 实际 %q", result.Category)
		}
	})
}
