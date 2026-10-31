package service

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 本文件守卫「AI 审计底账里的模型名必须是真的」这条性质。
//
// 背景：TriageService 曾用硬编码常量 "gpt-4o-mini" 填 model 字段，而生产配的
// 是另一家 provider——审计底账于是系统地指向一个错误的模型。假溯源比没有溯源
// 更危险：底账看起来权威，事后追责时却把人引向错误方向。修法是让 provider 自报
// 模型（ModelReportingProvider），取不到时如实报 "unknown"，绝不猜。
//
// 三层守卫：
//  1. EffectiveModel 的三条路径（显式 > provider 自报 > unknown）；
//  2. 分诊结果的 modelName 真的等于 provider 配置的模型，而不是任何硬编码值；
//  3. 源码扫描：生产代码里不得再出现硬编码的具体模型名（防回归）。

// modelReportingProvider 能自报模型的 provider（可报路径）。
type modelReportingProvider struct {
	model    string
	response string
}

func (p *modelReportingProvider) Chat(_ context.Context, _ string, _ []LLMMessage) (string, error) {
	if p.response != "" {
		return p.response, nil
	}
	return `{"category":"general","priority":"medium","confidence":0.8,"explanation":"stub"}`, nil
}

func (p *modelReportingProvider) Model() string { return p.model }

// silentProvider 不能自报模型的 provider（与本地 Ollama 同类：模型完全由调用方
// 每次入参决定）。它有意不实现 ModelReportingProvider。
type silentProvider struct{}

func (p *silentProvider) Chat(_ context.Context, _ string, _ []LLMMessage) (string, error) {
	return `{"category":"general","priority":"medium","confidence":0.8,"explanation":"stub"}`, nil
}

func TestEffectiveModelPrefersExplicitModel(t *testing.T) {
	g := NewLLMGateway(&modelReportingProvider{model: "provider-configured"}, nil, nil, "p")
	// 显式指定的模型是权威：调用方知道自己在调什么。
	assert.Equal(t, "caller-specified", g.EffectiveModel("caller-specified"))
}

func TestEffectiveModelFallsBackToProviderConfig(t *testing.T) {
	g := NewLLMGateway(&modelReportingProvider{model: "deepseek-chat"}, nil, nil, "p")
	// 调用方不指定时，用 provider 实际配置的那个——这正是此前被硬编码掩盖的信息。
	assert.Equal(t, "deepseek-chat", g.EffectiveModel(""))
}

func TestEffectiveModelUnknownWhenProviderCannotReport(t *testing.T) {
	g := NewLLMGateway(&silentProvider{}, nil, nil, "p")
	// 不能报就报 unknown。宁可让运维去补配置，也不填一个看起来像真的模型名。
	assert.Equal(t, "unknown", g.EffectiveModel(""))
}

// TestEffectiveModelIsNeverAFabricatedModelName 是反向性质的守卫：
// 无论走哪条路径，EffectiveModel 都不得凭空造出一个具体模型名。
func TestEffectiveModelIsNeverAFabricatedModelName(t *testing.T) {
	g := NewLLMGateway(&silentProvider{}, nil, nil, "p")
	got := g.EffectiveModel("")
	assert.NotEqual(t, "gpt-4o-mini", got, "不得回退到硬编码的模型名")
	assert.NotEmpty(t, got, "也不得回退到空串：空的溯源字段会被前端当成「未记录」而静默丢弃")
}

// TestTriageModelNameReflectsRealProviderModel 锁住端到端性质：分诊结果里的
// modelName 必须等于 provider 配置的模型，而不是任何写死的常量。
func TestTriageModelNameReflectsRealProviderModel(t *testing.T) {
	const configured = "deepseek-chat"
	g := NewLLMGateway(&modelReportingProvider{model: configured}, nil, nil, "deepseek")
	svc := NewTriageService(g, zap.NewNop())

	result := svc.SuggestForTenant(context.Background(), "数据库连不上", "应用报 connection refused", 1)

	require.Equal(t, "llm", result.Model, "model 字段表示决策链路，不应被模型名污染")
	assert.Equal(t, configured, result.ModelName, "modelName 必须是 provider 实际使用的模型")
	assert.NotEqual(t, "gpt-4o-mini", result.ModelName)
}

// TestTriageModelNameUnknownForSilentProvider：provider 报不出模型时，
// 底账如实写 unknown——不静默省略，也不编造。
func TestTriageModelNameUnknownForSilentProvider(t *testing.T) {
	g := NewLLMGateway(&silentProvider{}, nil, nil, "local")
	svc := NewTriageService(g, zap.NewNop())

	result := svc.SuggestForTenant(context.Background(), "数据库连不上", "应用报 connection refused", 1)

	require.Equal(t, "llm", result.Model)
	assert.Equal(t, "unknown", result.ModelName)
}

// TestNoHardcodedModelNameInProductionCode 是防回归的源码扫描守卫。
//
// 为什么用源码扫描而不是断言某个常量：硬编码可以出现在任何调用点，逐个断言
// 会永远追在后面。扫描 service/ 与 handlers/ 下所有非测试 .go 文件，命中即红。
//
// 只扫「代码行」：注释里提到 gpt-4o-mini 是在解释这个坑本身，不算违规。
// 测试文件里的固定值是为了断言，同样豁免。
func TestNoHardcodedModelNameInProductionCode(t *testing.T) {
	// 具体厂商模型名的前缀。配置里的默认值（如 config 包）不在此列——
	// 本次只收口 AI 决策链路，避免误伤无关代码。
	prefixes := []string{"gpt-", "claude-", "deepseek-", "qwen", "glm-", "moonshot-"}

	var offenders []string
	roots := []string{"..", "."}
	seen := map[string]bool{}
	for _, root := range roots {
		for _, dir := range []string{"service", "handlers"} {
			base := filepath.Join(root, dir)
			if _, err := os.Stat(base); err != nil {
				continue
			}
			_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}
				if strings.HasSuffix(path, "_test.go") || seen[path] {
					return nil
				}
				seen[path] = true
				b, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				for i, line := range strings.Split(string(b), "\n") {
					trimmed := strings.TrimSpace(line)
					// 跳过注释与纯文档行
					if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
						continue
					}
					for _, p := range prefixes {
						if strings.Contains(trimmed, `"`+p) {
							offenders = append(offenders,
								filepath.Base(path)+":"+strconv.Itoa(i+1)+" → "+trimmed)
						}
					}
				}
				return nil
			})
		}
	}

	assert.Empty(t, offenders,
		"生产代码不得硬编码具体模型名：模型应由 provider 配置或调用方显式传入。\n"+
			"硬编码会让 AI 审计底账在换模型之后继续指向旧模型。命中位置：\n"+
			strings.Join(offenders, "\n"))
}
