package layering

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/handlers/vector_store"
	"itsm-backend/internal/vectorstore"
	"itsm-backend/internal/services"
)

// 向量/嵌入共享基础设施归位专项回归锁（2026-11-11）。
//
// 本批把 service/ 顶层四个生产实现（vector_store.go / embed_pipeline.go /
// embedder_openai.go / ai_services.go，共 381 行）git mv 进 internal/vectorstore，
// 按计划文档「共享基础设施迁 internal/」的收口方向落地。与 skill/escalation/sla
// 三批不同，本批的假绿风险不是「域包重新依赖共享层」，而是：
//
//   - 实现搬走后在 service 侧另起一份同形 struct（别名被换成独立副本），
//     类型身份与 RLS 会话口径会静默漂移，go build 完全绿；
//   - 旧文件名复活或基线行复活，L.1 / C.6.4 棘轮若不逐行锁定会留免检空间；
//   - 组合根被"顺手"改成直连 internal/vectorstore，绕过 service 侧过渡 API，
//     使 app.go（正带并行会话 WIP）产生不必要的装配改动。
//
// 因此锁同时检查「编译可过但语义全错」的回退：别名退化、实现复制、
// 文件/基线复活、上界回退。逐处比对不取并集，扫描器零命中直接失败。

// 编译期断言：service 侧必须是同一类型/同一签名，而不是同形副本。
var (
	_ *service.VectorStore     = (*vectorstore.VectorStore)(nil)
	_ *vectorstore.VectorStore = (*service.VectorStore)(nil)
	_ service.Embedder         = (*vectorstore.OpenAIEmbedder)(nil)
	_ vectorstore.Embedder     = (*service.OpenAIEmbedder)(nil)
	_ vector_store.VectorStore = (*service.VectorStore)(nil)

	_ func(*sql.DB) *service.VectorStore                   = service.NewVectorStore
	_ func(*sql.DB) *vectorstore.VectorStore               = vectorstore.NewVectorStore
	_ func() *service.OpenAIEmbedder                       = service.NewOpenAIEmbedder
	_ func(string, string, string) *service.OpenAIEmbedder = service.NewOpenAIEmbedderWithConfig
)

func vsReadBackend(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", ".."}, parts...)...)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", p, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：%s 仅 %d 字节", p, len(body))
	}
	return string(body)
}

func vsReadRepo(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", "..", ".."}, parts...)...)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", p, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：%s 仅 %d 字节", p, len(body))
	}
	return string(body)
}

// vsHasLine 逐行精确锚定声明，不用全文模糊匹配（避免同名参数假绿）。
func vsHasLine(src, decl string) bool {
	for _, line := range strings.Split(src, "\n") {
		if strings.TrimSpace(line) == strings.TrimSpace(decl) {
			return true
		}
	}
	return false
}

// vsHasAlias 逐行比对 type 块里的别名：lhs 必须是声明名本身，rhs 必须精确等于
// 目标限定符，gofmt 对齐空格不参与比较。别名被换成独立 struct 时立即失配。
func vsHasAlias(src, name, target string) bool {
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, name+" ") && t != name {
			continue
		}
		parts := strings.SplitN(t, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) == name && strings.TrimSpace(parts[1]) == target {
			return true
		}
	}
	return false
}

// vsServiceFilesCeilingOK 是 C.6.4 上界棘轮的纯函数，供正向/反向验证复用：
// 上界只允许继续下降（≤335），回退到 339 必红；继续降到 334 不误报。
func vsServiceFilesCeilingOK(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "service_go_files=") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "service_go_files="))
		if len(fields) == 0 {
			return false
		}
		v, err := strconv.Atoi(fields[0])
		if err != nil {
			return false
		}
		return v >= 0 && v <= 335
	}
	return false
}

func TestVectorStoreImplementationIsInfraOwned(t *testing.T) {
	oldNames := []string{"vector_store.go", "embed_pipeline.go", "embedder_openai.go", "ai_services.go"}
	newNames := []string{"store.go", "embedding_pipeline.go", "embedder_openai.go", "similar_incidents.go"}

	// 1) 旧文件名在 service/ 顶层零复活（编译可过，只有源码守卫能挡）。
	for _, fn := range oldNames {
		if _, err := os.Stat(filepath.Join("..", "..", "internal", "services", fn)); !os.IsNotExist(err) {
			t.Errorf("internal/services/%s 复活：实现本体必须只在 internal/vectorstore", fn)
		}
	}

	// 2) 四个实现归位 internal/vectorstore，包名正确，且不得反向依赖上层。
	for _, fn := range newNames {
		src := vsReadBackend(t, "internal", "vectorstore", fn)
		if !strings.HasPrefix(src, "package vectorstore\n") {
			t.Errorf("internal/vectorstore/%s 包名漂移", fn)
		}
		if strings.Contains(src, `"itsm-backend/internal/services"`) {
			t.Errorf("internal/vectorstore/%s 反向依赖 service 包", fn)
		}
		if strings.Contains(src, `"itsm-backend/handlers/`) {
			t.Errorf("internal/vectorstore/%s 反向依赖 handlers 域包", fn)
		}
	}

	// 3) service 侧只保留别名 + 薄委托，禁止复制实现。
	rag := vsReadBackend(t, "internal", "services", "rag_service.go")
	for _, a := range []struct{ name, target string }{
		{"VectorStore", "vectorstore.VectorStore"},
		{"VectorSearchResult", "vectorstore.VectorSearchResult"},
		{"Embedder", "vectorstore.Embedder"},
		{"EmbeddingReadiness", "vectorstore.EmbeddingReadiness"},
		{"EmbeddingPipeline", "vectorstore.EmbeddingPipeline"},
		{"OpenAIEmbedder", "vectorstore.OpenAIEmbedder"},
	} {
		if !vsHasAlias(rag, a.name, a.target) {
			t.Errorf("internal/services/rag_service.go 缺少类型别名 %s = %s", a.name, a.target)
		}
	}
	for _, decl := range []string{
		"func NewVectorStore(db *sql.DB) *VectorStore { return vectorstore.NewVectorStore(db) }",
		"func NewOpenAIEmbedder() *OpenAIEmbedder { return vectorstore.NewOpenAIEmbedder() }",
		"return vectorstore.NewOpenAIEmbedderWithConfig(apiKey, endpoint, model)",
		"return vectorstore.NewEmbeddingPipeline(client, e, logger, vectors)",
		"return vectorstore.SimilarIncidents(ctx, vectors, embedder, tenantID, query, k)",
	} {
		if !vsHasLine(rag, decl) {
			t.Errorf("internal/services/rag_service.go 缺少薄委托 %q", decl)
		}
	}
	if strings.Contains(rag, "type VectorStore struct") || strings.Contains(rag, "func (s *VectorStore)") {
		t.Error("service 侧复制了 VectorStore 实现，类型身份已分裂")
	}

	// 4) 运行时类型身份：别名不是同形副本。
	pairs := []struct {
		name      string
		got, want reflect.Type
	}{
		{"VectorStore", reflect.TypeOf(service.VectorStore{}), reflect.TypeOf(vectorstore.VectorStore{})},
		{"VectorSearchResult", reflect.TypeOf(service.VectorSearchResult{}), reflect.TypeOf(vectorstore.VectorSearchResult{})},
		{"OpenAIEmbedder", reflect.TypeOf(service.OpenAIEmbedder{}), reflect.TypeOf(vectorstore.OpenAIEmbedder{})},
		{"EmbeddingPipeline", reflect.TypeOf(service.EmbeddingPipeline{}), reflect.TypeOf(vectorstore.EmbeddingPipeline{})},
		{"Embedder", reflect.TypeOf((*service.Embedder)(nil)).Elem(), reflect.TypeOf((*vectorstore.Embedder)(nil)).Elem()},
		{"EmbeddingReadiness", reflect.TypeOf((*service.EmbeddingReadiness)(nil)).Elem(), reflect.TypeOf((*vectorstore.EmbeddingReadiness)(nil)).Elem()},
	}
	for _, p := range pairs {
		if p.got != p.want {
			t.Errorf("service.%s 与 vectorstore.%s 不再是同一类型", p.name, p.name)
		}
	}

	// 5) 组合根不得绕开 service 侧过渡 API（app.go 本批零改动）。
	app := vsReadBackend(t, "internal", "bootstrap", "app.go")
	if strings.Contains(app, `"itsm-backend/internal/vectorstore"`) {
		t.Error("app.go 直连 internal/vectorstore，绕过 service 侧过渡 API")
	}
	if !strings.Contains(app, "service.NewVectorStore(") {
		t.Error("app.go 缺少 service.NewVectorStore 装配（别名/薄委托未生效）")
	}

	// 6) 域包窄契约仍以同一具体类型满足（防止改名后编译期断言被悄悄删除）。
	contracts := vsReadBackend(t, "handlers", "vector_store", "contracts.go")
	if !vsHasLine(contracts, "type VectorStore interface {") {
		t.Error("handlers/vector_store/contracts.go 窄契约接口名漂移")
	}

	// 7) L.1 旧行零复活；C.6.4 上界棘轮（回退必红、继续下降不误报）。
	baseline := vsReadRepo(t, "scripts", "docs-gate", "service-prod-baseline.txt")
	for _, line := range strings.Split(baseline, "\n") {
		trimmed := strings.TrimSpace(line)
		for _, fn := range oldNames {
			if trimmed == fn {
				t.Errorf("L.1 基线复活已迁出文件 %q", fn)
			}
		}
	}
	drift := vsReadRepo(t, "scripts", "docs-gate", "product-surface-baseline.txt")
	if !vsServiceFilesCeilingOK(drift) {
		t.Error("C.6.4 service_go_files 上界必须 ≤335（339 → 335，后续只许继续下降）")
	}
	if !vsServiceFilesCeilingOK("service_go_files=334 # 继续下降") {
		t.Error("棘轮正向验证失败：继续下降 335→334 不应误报")
	}
	if vsServiceFilesCeilingOK("service_go_files=339 # 回退") {
		t.Error("棘轮反向验证失败：回退到 339 必须报红")
	}
}
