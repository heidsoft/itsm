package ai

import (
	"context"
	"os"
	"strings"
	"testing"

	"itsm-backend/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 共享层 RAG 实现仍满足域包契约与可选流式能力：抽接口不改变装配行为。
var (
	_ RAGChatService = (*service.RAGService)(nil)
	_ ragStreamer    = (*service.RAGService)(nil)
)

// fakeRAGChat 是纯域内实现，不依赖共享 service 包。
// 回归锁意义：若把 Service.rag / NewService 改回 *service.RAGService，
// 本文件立即无法编译 —— 能区分新旧实现，不是假绿。
type fakeRAGChat struct {
	asked      int
	askedMulti int
}

func (f *fakeRAGChat) Ask(_ context.Context, _ int, _ string, _ int) ([]map[string]any, error) {
	f.asked++
	return []map[string]any{{"title": "from-domain-fake"}}, nil
}

func (f *fakeRAGChat) AskMulti(_ context.Context, _ int, _ string, _ []string, _ int) ([]map[string]any, error) {
	f.askedMulti++
	return nil, nil
}

func (f *fakeRAGChat) Mode() string { return "fake" }

func TestAIServiceAcceptsDomainLocalRAG(t *testing.T) {
	fake := &fakeRAGChat{}
	svc := NewService(nil, zap.NewNop().Sugar(), fake, nil, nil, nil, nil, nil, nil, nil)
	require.NotNil(t, svc.rag, "域内实现必须能注入 AI Service")

	// SearchKnowledge 只经窄契约的 Ask，不触碰 repo / LLM。
	res, err := svc.SearchKnowledge(context.Background(), 1, "q", "keyword", 5)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.asked, "检索必须走注入的域内实现")
	assert.Len(t, res, 1)
	assert.Equal(t, "fake", svc.rag.Mode(), "检索模式同样来自窄契约")

	// 只实现读面契约的实现不具备流式能力：明确降级，而不是 panic 或解引用具体指针。
	_, _, err = svc.ChatStream(context.Background(), 1, 1, "end_user", "q", 5, 0, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RAG service not initialized", "降级文案须与未装配口径一致")

	// 未装配时保持既有降级语义，且不得留下 typed-nil 接口。
	bare := NewService(nil, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil)
	require.Nil(t, bare.rag, "nil 注入应清空依赖，不得留下 typed-nil 接口")
	_, err = bare.SearchKnowledge(context.Background(), 1, "q", "keyword", 5)
	assert.Error(t, err, "未装配时不得 panic")
}

// stripLineComments 剔除 // 行注释，让源码守卫只检查可执行代码。
// 契约与实现文件都需要在注释里写明被禁止的旧写法（否则后来者不知道为什么），
// 全文字符串扫描会把这些说明误判成违规；剔除注释后，真正的 import 与
// 具体类型引用仍会被抓到，反向验证依旧有效。
func stripLineComments(src string) string {
	var out strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

// 源码守卫：L.3 只数 import，挡不住「同包别名 + 共享层具体类型」回退。
// 同时锁住契约文件本身零共享层依赖：若它 import service，ai 域计数会由 4 升到 5。
func TestAIRAGContractHasNoSharedServiceType(t *testing.T) {
	contract, err := os.ReadFile("rag_contract.go")
	require.NoError(t, err, "读取 rag_contract.go")
	require.Greater(t, len(contract), 200, "扫描器必须真的读到契约文件，零字节/空文件不算通过")
	contractCode := stripLineComments(string(contract))
	require.NotEmpty(t, strings.TrimSpace(contractCode), "剔除注释后仍须有可执行代码")
	assert.NotContains(t, contractCode, `"itsm-backend/service"`,
		"rag_contract.go 不得 import 共享 service 包，否则 L.3 的 ai 计数会上升")
	assert.Contains(t, contractCode, `"itsm-backend/handlers/knowledge"`,
		"必须复用知识域现成的 RAGRetriever 契约，不得另起同签名接口")
	assert.Contains(t, contractCode, "type RAGChatService interface")
	assert.Contains(t, contractCode, "AskMulti(", "契约须覆盖 AI 域真实消费面")
	assert.Contains(t, contractCode, "Mode() string", "契约须覆盖 AI 域真实消费面")

	body, err := os.ReadFile("service.go")
	require.NoError(t, err, "读取 service.go")
	require.Greater(t, len(body), 1000, "扫描器必须真的读到 service.go，零命中不能是空文件造成的假绿")
	bodyCode := stripLineComments(string(body))
	require.NotEmpty(t, strings.TrimSpace(bodyCode), "剔除注释后仍须有可执行代码")
	assert.NotContains(t, bodyCode, "*service.RAGService",
		"service.go 不得持有共享层 RAG 具体类型")
	assert.Contains(t, bodyCode, "rag RAGChatService", "Service.rag 必须是域内窄接口")
	assert.Contains(t, bodyCode, "type ragStreamer interface", "流式能力须显式声明为可选接口")
	assert.Contains(t, bodyCode, "s.rag.(ragStreamer)", "流式路径必须走类型断言")
	assert.Contains(t, bodyCode, "streamer.AskWithLLMStreamWithToolsMulti(",
		"流式调用不得退回 s.rag 的具体方法")
}
