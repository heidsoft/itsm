package knowledge

import (
	"context"
	"os"
	"testing"

	"itsm-backend/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 共享层 RAG 实现仍满足域包契约：抽接口不改变装配行为。
var (
	_ RAGService   = (*service.RAGService)(nil)
	_ RAGRetriever = (*service.RAGService)(nil)
	_ RAGIndexer   = (*service.RAGService)(nil)
)

// fakeRAG 是纯域内实现，不依赖共享 service 包。
// 回归锁意义：若把 Service.rag / NewVectorIndexCommandHandler 改回共享层
// 具体指针类型，本文件立即无法编译 —— 能区分新旧实现，不是假绿。
type fakeRAG struct {
	asked   int
	indexed int
	removed int
}

func (f *fakeRAG) Ask(_ context.Context, _ int, _ string, _ int) ([]map[string]any, error) {
	f.asked++
	return nil, nil
}
func (f *fakeRAG) HasVectorBackend() bool { return true }
func (f *fakeRAG) VectorIndexReady() bool { return true }
func (f *fakeRAG) IndexArticle(_ context.Context, _, _ int, _, _ string) error {
	f.indexed++
	return nil
}
func (f *fakeRAG) RemoveArticle(_ context.Context, _, _ int) error {
	f.removed++
	return nil
}

func TestKnowledgeServiceAcceptsDomainLocalRAG(t *testing.T) {
	fake := &fakeRAG{}
	svc := &Service{}
	svc.SetRAG(fake)
	require.NotNil(t, svc.rag, "域内实现必须能注入 KnowledgeService")
	assert.False(t, svc.vectorIndexBlocked(), "后端已就绪时不应把索引判为阻塞")

	worker := NewVectorIndexCommandHandler(nil, fake)
	require.NotNil(t, worker)
	require.NotNil(t, worker.rag, "域内实现必须能注入 outbox worker")

	// 窄契约可单独使用：worker 只需要 RAGIndexer 子集。
	var indexer RAGIndexer = fake
	assert.True(t, indexer.VectorIndexReady())

	svc.SetRAG(nil)
	assert.Nil(t, svc.rag, "nil 注入应清空依赖，不得留下 typed-nil 接口")
}

// 源码守卫：知识域生产文件不得重新引入共享 service 包，
// 也不得把 RAG 字段写回具体指针类型（L.3 只挡 import，挡不住同包别名回退）。
func TestKnowledgeDomainHasNoSharedServiceType(t *testing.T) {
	for _, name := range []string{"rag_contract.go", "service.go", "outbox.go"} {
		body, err := os.ReadFile(name)
		require.NoError(t, err, "读取 %s", name)
		assert.NotContains(t, string(body), `"itsm-backend/internal/services"`,
			"%s 不得 import 共享 service 包", name)
		assert.NotContains(t, string(body), "*service.RAGService",
			"%s 不得持有共享层 RAG 具体类型", name)
	}
}
