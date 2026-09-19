# AI服务精准测试计划

> **目标**: 为3个AI核心服务（triage/rag/summarize）编写高质量测试，聚焦关键路径，覆盖fallback逻辑

**测试原则**:
- 每个服务8-10个关键测试场景
- 使用Mock隔离外部依赖（LLM Gateway）
- 覆盖：正常路径、fallback路径、边界条件
- 测试纯函数优先（如keywordBasedSuggest、simpleTruncate）

---

## 一、测试文件结构

```
itsm-backend/service/
├── triage_service_test.go       # AI分流测试
├── rag_service_test.go           # RAG检索测试
├── summarize_service_test.go     # 摘要服务测试
```

---

## 二、TriageService 测试设计

### 测试场景（共9个）

| 场景 | 描述 | 难度 |
|------|------|------|
| TestTriage_Suggest_LLM_Success | LLM正常返回有效JSON | 中 |
| TestTriage_Suggest_LLM_ParseError | LLM返回非法JSON，使用fallback | 中 |
| TestTriage_Suggest_LLM_EmptyResponse | LLM返回空分类，使用默认值 | 易 |
| TestTriage_Suggest_NoGateway_KeywordOnly | 无LLM网关，使用keyword fallback | 易 |
| TestTriage_Suggest_Keyword_Database | 关键词"mysql"匹配database | 易 |
| TestTriage_Suggest_Keyword_Network | 关键词"wifi"匹配network | 易 |
| TestTriage_Suggest_Keyword_PriorityEscalation | 关键词"down"提升优先级 | 易 |
| TestTriage_Suggest_EmptyInput | 空输入返回默认值 | 易 |
| TestTriage_containsAny_Match | containsAny辅助函数测试 | 易 |
| TestTriage_BatchSuggest | 批量处理测试 | 易 |

### Mock设计

```go
// MockLLMGateway 实现 LLMProvider 接口
type MockLLMGateway struct {
    MockChat func(ctx context.Context, model string, messages []LLMMessage) (string, error)
}

func (m *MockLLMGateway) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
    return m.MockChat(ctx, model, messages)
}
```

### 测试代码框架

```go
package service

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "go.uber.org/zap"
)

func TestTriage_Suggest_Keyword_Database(t *testing.T) {
    // 无LLM网关，强制使用keyword fallback
    svc := NewTriageService(nil, zap.NewNop())

    result := svc.Suggest(context.Background(), "MySQL连接失败", "数据库无法访问")

    assert.Equal(t, "database", result.Category)
    assert.Equal(t, 101, result.AssigneeID)
    assert.Greater(t, result.Confidence, 0.6)
}

func TestTriage_Suggest_LLM_Success(t *testing.T) {
    mockGateway := &MockLLMGateway{
        MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
            return `{"category":"security","priority":"critical","confidence":0.9,"explanation":"sql injection"}`, nil
        },
    }
    svc := NewTriageService(mockGateway, zap.NewNop())

    result := svc.Suggest(context.Background(), "SQL注入攻击", "发现恶意请求")

    assert.Equal(t, "security", result.Category)
    assert.Equal(t, "critical", result.Priority)
    assert.Equal(t, 100, result.AssigneeID)
}

func TestTriage_Suggest_LLM_ParseError_Fallback(t *testing.T) {
    mockGateway := &MockLLMGateway{
        MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
            return "invalid json response", nil  // 返回非法JSON
        },
    }
    svc := NewTriageService(mockGateway, zap.NewNop())

    result := svc.Suggest(context.Background(), "服务器宕机", "")

    // 应该fallback到keyword，匹配server
    assert.Equal(t, "server", result.Category)
}
```

---

## 三、RAGService 测试设计

### 测试场景（共8个）

| 场景 | 描述 | 难度 |
|------|------|------|
| TestRAG_Ask_HybridSearch_Deduplicates | 混合搜索去重 | 中 |
| TestRAG_Ask_VectorFails_FallbackKeyword | 向量搜索失败，降级到keyword | 中 |
| TestRAG_Ask_KeywordOnly_NoVector | 纯关键词搜索模式 | 易 |
| TestRAG_Ask_LimitParameter | limit参数控制结果数量 | 易 |
| TestRAG_VectorSearch_EmbbedingError | Embedding失败处理 | 中 |
| TestRAG_keywordSearch_MultipleMatches | 多匹配返回正确数量 | 易 |
| TestRAG_snippet_Truncation | snippet函数截断长文本 | 易 |
| TestRAG_Ask_EmptyQuery | 空查询处理 | 易 |

### Mock设计

```go
// MockVectorStore
type MockVectorStore struct {
    MockSearchTopKByType func(ctx context.Context, tenantID int, objType string, embedding []float64, k int) (interface{ Close() error; Next() bool; Scan(...interface{}) error }, error)
}

// MockEmbedder
type MockEmbedder struct {
    MockEmbed func(text string) ([]float64, error)
}

func (m *MockEmbedder) Embed(text string) ([]float64, error) {
    return m.MockEmbed(text)
}
```

### 测试代码框架

```go
func TestRAG_Ask_KeywordOnly_NoVector(t *testing.T) {
    cfg := RAGConfig{
        UseVector:    false,
        UseKeyword:   true,
        HybridSearch: false,
        MaxResults:   5,
    }
    svc := NewRAGService(mockClient, nil, nil, logger, cfg)  // vectors和embedder为nil

    results, err := svc.Ask(context.Background(), 1, "mysql", 5)

    assert.NoError(t, err)
    assert.Equal(t, 5, len(results))
}

func TestRAG_snippet_Truncation(t *testing.T) {
    longText := strings.Repeat("a", 500)
    result := snippet(longText, 200)

    assert.Equal(t, 200, len(result))
    assert.True(t, strings.HasSuffix(result, "..."))
}
```

---

## 四、SummarizeService 测试设计

### 测试场景（共8个）

| 场景 | 描述 | 难度 |
|------|------|------|
| TestSummarize_TextShort_ReturnsAsIs | 文本小于maxLen直接返回 | 易 |
| TestSummarize_WithLLM_Success | LLM正常返回摘要 | 中 |
| TestSummarize_WithLLM_TrimsOutput | LLM返回带markdown格式被清理 | 易 |
| TestSummarize_NoGateway_Fallback | 无LLM网关使用simpleTruncate | 易 |
| TestSummarize_WithContext | 带上下文的摘要 | 中 |
| TestSummarize_GenerateActionItems | 提取行动项 | 中 |
| TestSummarize_simpleTruncate | simpleTruncate函数测试 | 易 |
| TestSummarize_EmptyInput | 空输入处理 | 易 |

### 测试代码框架

```go
func TestSummarize_TextShort_ReturnsAsIs(t *testing.T) {
    svc := NewSummarizeService(nil, zap.NewNop())

    text := "短文本"
    result, err := svc.Summarize(context.Background(), text, 200)

    assert.NoError(t, err)
    assert.Equal(t, text, result)
}

func TestSummarize_simpleTruncate(t *testing.T) {
    longText := strings.Repeat("a", 500)
    result := simpleTruncate(longText, 100)

    assert.Equal(t, 103, len(result))  // 100 + "..."
    assert.True(t, strings.HasSuffix(result, "..."))
}

func TestSummarize_WithLLM_Success(t *testing.T) {
    mockGateway := &MockLLMGateway{
        MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
            return "这是LLM生成的摘要内容", nil
        },
    }
    svc := NewSummarizeService(mockGateway, zap.NewNop())

    result, err := svc.Summarize(context.Background(), strings.Repeat("a", 1000), 200)

    assert.NoError(t, err)
    assert.Contains(t, result, "LLM生成的摘要")
}
```

---

## 五、执行顺序

1. **TriageService** - 纯函数多，fallback路径清晰，最容易测试
2. **SummarizeService** - 依赖Mock LLM，simpleTruncate可单独测试
3. **RAGService** - 最复杂，需要Mock VectorStore和Embedder

每个服务测试完成后：
- 运行 `go test ./service/triage_service_test.go -v` 验证
- 确保所有测试通过
- 提交到Git

---

## 六、验收标准

| AI服务 | 测试场景数 | 目标覆盖率 |
|--------|-----------|-----------|
| TriageService | 9 | ~70% |
| RAGService | 8 | ~60% |
| SummarizeService | 8 | ~75% |

### 质量检查
- [ ] 所有测试通过
- [ ] 无hardcoded mock数据泄露
- [ ] 断言充分（不只是`assert.NoError`）
- [ ] 覆盖fallback路径
- [ ] 测试独立运行（不依赖外部服务）