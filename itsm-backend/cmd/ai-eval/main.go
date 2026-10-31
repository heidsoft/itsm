// Command ai-eval 是 AI 质量验收的**可执行入口**。
//
// ## 为什么是一个命令而不是环境变量
//
// 此前 ai/eval 曾提供过 ITSM_AI_EVAL_USE_LLM=1 的「生产验收模式」开关——
// 但它只会切换日志标签，候选仍来自降级路径，等于对外宣称一个不存在的
// 验收模式。真实的 LLM 质量验收需要真实语料与真实候选生成器，落地方式
// 必须是「人主动执行、看得见输出」的命令，而不是 CI 里悄悄生效的开关。
//
// ## 用法
//
//	# 降级档（无 LLM，验证评测框架本身与降级路径的基线）：
//	go run ./cmd/ai-eval
//
//	# LLM 档（生产验收，读 llm.* 配置构造真实 provider）：
//	go run ./cmd/ai-eval -mode llm
//
//	# 指定输出目录（默认 ./ai-eval-report）：
//	go run ./cmd/ai-eval -mode llm -out /tmp/report
//
// 输出为 JSON 报告：每个评测项给出指标、样本数、模式与候选来源说明，
// 并附机器可读的结论（passed / mode / basis）。报告落在输出目录，供
// 验收归档；CI 不消费 LLM 档（外部依赖不稳定，不该挡合并）。
//
// ## 退出码
//
//	0 —— 报告产出（不代表指标达标；是否达标由验收人对照阈值判断）
//	1 —— 无法产出报告（数据集缺失、LLM 配置缺失等）
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ai/eval"
	"itsm-backend/service"
)

type sectionResult struct {
	Task        string  `json:"task"`
	Mode        string  `json:"mode"`
	Basis       string  `json:"basis"`
	Cases       int     `json:"cases"`
	Metric      string  `json:"metric"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
}

type report struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	Mode        string          `json:"mode"`
	Sections    []sectionResult `json:"sections"`
	Notes       []string        `json:"notes"`
}

func main() {
	mode := flag.String("mode", "degraded", "评测档位：degraded（无 LLM）或 llm（真实 LLM，需 llm.* 配置）")
	outDir := flag.String("out", "ai-eval-report", "报告输出目录")
	datasets := flag.String("datasets", "ai/eval/datasets", "数据集目录（相对工作目录）")
	flag.Parse()

	if err := run(*mode, *outDir, *datasets); err != nil {
		fmt.Fprintf(os.Stderr, "ai-eval: %v\n", err)
		os.Exit(1)
	}
}

func run(mode, outDir, datasets string) error {
	logger := zap.NewNop()
	rep := report{
		GeneratedAt: time.Now(),
		Mode:        mode,
		Notes: []string{
			"本报告是 AI 质量验收的输入，不是自动结论：指标是否达标由验收人对照阈值判断。",
			"degraded 档测的是生产降级路径（无 LLM 时的真实行为），LLM 档测真实模型候选。",
		},
	}

	// ---- 摘要 ----
	var sumGen func(string) string
	basis := "生产 SummarizeService 无 LLM 降级路径（截断）"
	if mode == "llm" {
		gateway, err := newGatewayFromConfig()
		if err != nil {
			return fmt.Errorf("LLM 档需要可用的 llm.* 配置: %w", err)
		}
		svc := service.NewSummarizeService(gateway, logger)
		sumGen = eval.SummarizeCandidateGenerator(svc)
		basis = "生产 SummarizeService + 真实 LLM（llm.* 配置）"
	} else {
		svc := service.NewSummarizeService(nil, logger)
		sumGen = eval.SummarizeCandidateGenerator(svc)
	}

	sumCases := mustLoad[[]eval.SummarizeCase](filepath.Join(datasets, "summarize.jsonl"))
	rep.Sections = append(rep.Sections, sectionResult{
		Task: "summarize", Mode: mode, Basis: basis,
		Cases: len(sumCases), Metric: "rougeL(topic recall)",
		Value:       eval.SummarizeROUGE(sumCases, sumGen),
		Description: "摘要候选对期望主题的覆盖率（ROUGE-L 召回）。",
	})

	// ---- RAG（真实检索器 + 真实语料，与 CI 同一条链路） ----
	corpus, err := eval.LoadRAGCorpus(filepath.Join(datasets, "corpus", "rag_corpus.jsonl"))
	if err != nil {
		return fmt.Errorf("载入 RAG 语料: %w", err)
	}
	ragCases := mustLoad[[]eval.RAGCase](filepath.Join(datasets, "rag.jsonl"))
	retriever, err := eval.NewRealRAGRetriever("file:aieval_corpus?mode=memory&cache=shared&_fk=1", corpus, logger.Sugar())
	if err != nil {
		return fmt.Errorf("建 RAG 评测库: %w", err)
	}
	defer retriever.Close()

	ctx := context.Background()
	hits := 0
	for _, c := range ragCases {
		got := retriever.Retrieve(ctx, c)
		if len(got) == 0 || len(c.ExpectedRefs) == 0 {
			continue
		}
		expected := map[string]struct{}{}
		for _, ref := range c.ExpectedRefs {
			expected[eval.NormalizeRef(ref)] = struct{}{}
		}
		for _, id := range got {
			if _, ok := expected[eval.NormalizeRef(id)]; ok {
				hits++
				break
			}
		}
	}
	rep.Sections = append(rep.Sections, sectionResult{
		Task: "rag", Mode: "real-retrieval",
		Basis: "生产 RAGService 关键词检索 + 真实语料（corpus/rag_corpus.jsonl）",
		Cases: len(ragCases), Metric: "hitRate",
		Value:       float64(hits) / float64(len(ragCases)),
		Description: "真实检索在语料上的召回率（命中任一期望文章即计命中）。",
	})

	// ---- 预测（真实预测器） ----
	predCases := mustLoad[[]eval.PredictionCase](filepath.Join(datasets, "prediction.jsonl"))
	predictor := service.NewSLABreachPredictor()
	pos, neg, correctPairs := 0, 0, 0.0
	scores := make([]float64, 0, len(predCases))
	labels := make([]bool, 0, len(predCases))
	for _, c := range predCases {
		risk, err := predictor.Predict(predictFeatures(c))
		if err != nil {
			continue
		}
		scores = append(scores, risk.Probability)
		labels = append(labels, c.ExpectedBreachWithinSLA)
		if c.ExpectedBreachWithinSLA {
			pos++
		} else {
			neg++
		}
	}
	if pos > 0 && neg > 0 {
		correctPairs = auc(scores, labels)
	}
	rep.Sections = append(rep.Sections, sectionResult{
		Task: "prediction", Mode: "real-predictor",
		Basis: "生产 SLABreachPredictor（排序层）+ 人工标注（annotation_basis 留存）",
		Cases: len(predCases), Metric: "rocAuc",
		Value:       correctPairs,
		Description: "预测器排序与人工标注的一致性（AUC 只看排序，不看概率绝对值）。",
	})

	// ---- 输出 ----
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("建输出目录: %w", err)
	}
	raw, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(outDir, fmt.Sprintf("ai-eval-%s-%s.json", mode, rep.GeneratedAt.Format("20060102-150405")))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("写报告: %w", err)
	}

	fmt.Printf("ai-eval 报告已产出: %s\n", path)
	for _, s := range rep.Sections {
		fmt.Printf("  %-12s %-14s cases=%-3d %-22s = %.3f\n", s.Task, s.Mode, s.Cases, s.Metric, s.Value)
	}
	fmt.Println("注意：指标是否达标由验收人对照阈值判断；本命令不自动下结论。")
	return nil
}

// newGatewayFromConfig 按 llm.* 配置构造真实 LLM 网关。
//
// 配置缺失时报错而不是降级：验收人要的是「LLM 档」，悄悄降级成截断路径
// 会让报告看起来正常而实际没测到模型——那是假验收。
func newGatewayFromConfig() (*service.LLMGateway, error) {
	cfg := service.LoadLLMConfig()
	if cfg.APIKey == "" && cfg.Endpoint == "" {
		return nil, fmt.Errorf("llm.* 配置为空（provider=%q）", cfg.Provider)
	}
	provider := service.NewProviderFromConfig(cfg)
	if provider == nil {
		return nil, fmt.Errorf("无法按配置构造 provider（provider=%q）", cfg.Provider)
	}
	return service.NewLLMGateway(provider, nil, nil, cfg.Provider), nil
}

func mustLoad[T any](path string) T {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-eval: 读数据集 %s: %v\n", path, err)
		os.Exit(1)
	}
	var out T
	// JSONL：逐行解码后合并进切片（T 是切片类型时）。
	if err := json.Unmarshal(jsonlToArray(raw), &out); err != nil {
		fmt.Fprintf(os.Stderr, "ai-eval: 解析数据集 %s: %v\n", path, err)
		os.Exit(1)
	}
	return out
}

// jsonlToArray 把 JSONL 转成 JSON 数组（供 Unmarshal 进切片）。
func jsonlToArray(raw []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte('[')
	first := true
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !first {
			buf.WriteByte(',')
		}
		buf.WriteString(line)
		first = false
	}
	buf.WriteByte(']')
	return buf.Bytes()
}
