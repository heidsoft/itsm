package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件守卫「评测门禁本身是否有效」。
//
// ## 为什么需要这份文件
//
// 评测门禁最危险的失效模式不是「阈值太松」，而是**自我实现**：候选直接来自
// 期望答案，于是「用正确答案考自己」必然满分，指标 1.00 看起来很漂亮，但把
// 真实实现改坏（摘要返回空串、检索返回随机）它照样绿。
//
// 这样的门禁不是宽松，是**没有门禁**——它让「AI 质量有 CI 保护」变成不实陈述，
// 比没有门禁更危险：没人会去补真正的验收。
//
// 因此这里守三件事：
//
//  1. **候选真实性**：门禁用的候选不得是期望答案；
//  2. **退化敏感性**：候选退化时指标必须真的下降（门禁能感知坏）；
//  3. **源码形态**：门禁测试里不得再出现「stub 直接返回期望答案」的写法。
//
// 第 3 条用源码扫描实现：这种写法可以出现在任何地方，逐个断言会永远追在
// 后面；扫源码才是对的地方。

// TestGateCandidatesAreNotGoldenAnswers 守第 1 条：门禁实际使用的候选
// 不得就是期望答案。
func TestGateCandidatesAreNotGoldenAnswers(t *testing.T) {
	// 摘要：候选来自生产降级路径，不得等于期望 topics 的拼接。
	sumCases := LoadSummarizeCases(t)
	gen := SummarizeCandidateGenerator(nil)
	for _, tc := range sumCases {
		candidate := gen(strings.Join(tc.Messages, " "))
		report := DetectGoldenText(candidate, tc.ExpectedTopics)
		require.Falsef(t, report.UsedGolden,
			"case %s 的摘要候选就是期望答案：%s（候选=%q）", tc.ID, report.Detail, candidate)
	}

	// RAG：降级候选不得等于期望文档集合。
	ragCases := LoadRAGCases(t)
	for _, tc := range ragCases {
		candidate := DegradedRAGCandidates(tc)
		report := DetectGoldenDocIDs(candidate, tc.ExpectedDocIDs)
		require.Falsef(t, report.UsedGolden,
			"case %s 的检索候选就是期望答案：%s", tc.ID, report.Detail)
	}
}

// TestDetectGoldenDetectsSelfImplementation 验证检测器本身有效：
// 自我实现的候选必须被识别出来，正常候选不得被误判。
func TestDetectGoldenDetectsSelfImplementation(t *testing.T) {
	// 自我实现：候选就是期望答案 → 必须识别
	report := DetectGoldenText("CDN 缓存 登录页", []string{"CDN", "缓存", "登录页"})
	assert.True(t, report.UsedGolden, "完全相同的候选必须被识别为自我实现")
	assert.NotEmpty(t, report.Detail)

	// 正常候选：覆盖期望主题但不相同 → 不得误判
	report = DetectGoldenText("用户反馈 CDN 缓存了旧版页面，已强制刷新解决", []string{"CDN", "缓存", "登录页"})
	assert.False(t, report.UsedGolden, "正常的高分候选不得被误判为自我实现")

	// 文档集合同理
	report = DetectGoldenDocIDs([]string{"KB-001", "KB-002"}, []string{"KB-002", "KB-001"})
	assert.True(t, report.UsedGolden, "集合相同（顺序无关）必须被识别")

	report = DetectGoldenDocIDs([]string{"KB-999"}, []string{"KB-001", "KB-002"})
	assert.False(t, report.UsedGolden, "不同集合不得被误判")
}

// TestGateDetectsDegradation 守第 2 条：门禁必须对退化敏感。
//
// 这是「门禁有效」的判据本身：如果候选变空时指标不降，那这个门禁测不出任何
// 东西——它只是在校验一个恒定值。
func TestGateDetectsDegradation(t *testing.T) {
	sumCases := LoadSummarizeCases(t)
	require.NotEmpty(t, sumCases, "摘要数据集不应为空")

	// 正常候选（生产降级路径）
	gen := SummarizeCandidateGenerator(nil)
	good := SummarizeROUGE(t, sumCases, gen)

	// 退化候选：返回空串
	bad := SummarizeROUGE(t, sumCases, func(string) string { return "" })
	assert.Less(t, bad, good,
		"候选退化为空时 ROUGE 必须下降——若不降，说明该门禁测不出退化")

	// 退化候选：返回无关文本
	irrelevant := SummarizeROUGE(t, sumCases, func(string) string { return "zzz" })
	assert.Less(t, irrelevant, good,
		"候选退化为无关文本时 ROUGE 必须下降")

	ragCases := LoadRAGCases(t)
	require.NotEmpty(t, ragCases, "RAG 数据集不应为空")
	goodHit := RAGHitRate(t, ragCases, func(c RAGCase) []string { return DegradedRAGCandidates(c) })
	badHit := RAGHitRate(t, ragCases, func(c RAGCase) []string { return nil })
	assert.Less(t, badHit, goodHit,
		"候选退化为空时 hit-rate 必须下降——若不降，说明该门禁测不出退化")
}

// TestGateTestsDoNotUseGoldenStubs 守第 3 条：门禁测试里不得再出现
// 「stub 直接返回期望答案」的写法。
//
// 扫描范围是本包的所有 .go 文件；命中的是「返回 ExpectedDocIDs / ExpectedTopics
// 的字面量」这一形态。它可以出现在单元测试里（那是刻意的构造），但绝不能出现在
// 门禁路径上——所以本检查用「命中即报告 + 人工确认」的方式，而不是粗暴禁止。
//
// 更严格的做法是逐个函数检查，这里用文件级扫描 + 明确列出允许的位置：
// 允许出现在 gate_guard_test.go（本文件，构造反例用）与 eval_test.go 的
// 单元测试段。门禁函数 TestEval_* 内不允许。
func TestGateTestsDoNotUseGoldenStubs(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	var offenders []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(".", name))
		require.NoError(t, err)
		content := string(raw)

		// 只看门禁函数体：TestEval_ 开头的函数。
		for _, fn := range splitGoFuncs(content) {
			if !strings.HasPrefix(fn.name, "TestEval_") {
				continue
			}
			if strings.Contains(fn.body, "ExpectedDocIDs") && strings.Contains(fn.body, "return c.ExpectedDocIDs") {
				offenders = append(offenders, name+":"+fn.name+" 返回 ExpectedDocIDs 作为候选")
			}
			if strings.Contains(fn.body, "ExpectedTopics") && strings.Contains(fn.body, "return m") {
				offenders = append(offenders, name+":"+fn.name+" 把输入原样当候选")
			}
		}
	}
	assert.Empty(t, offenders,
		"门禁函数不得用期望答案充当候选（自我实现会让门禁恒满分）：\n"+strings.Join(offenders, "\n"))
}

// goFunc 是从源码里切出的一个顶层函数。
type goFunc struct {
	name string
	body string
}

// splitGoFuncs 按顶层函数切分源码（按行首的 "func " 判定）。
func splitGoFuncs(content string) []goFunc {
	var out []goFunc
	lines := strings.Split(content, "\n")
	var cur *goFunc
	var buf []string
	for _, ln := range lines {
		if strings.HasPrefix(ln, "func ") {
			if cur != nil {
				cur.body = strings.Join(buf, "\n")
				out = append(out, *cur)
			}
			name := ln
			if i := strings.Index(name, "("); i > 0 {
				name = name[5:i]
			}
			cur = &goFunc{name: name}
			buf = []string{ln}
			continue
		}
		if cur != nil {
			buf = append(buf, ln)
		}
	}
	if cur != nil {
		cur.body = strings.Join(buf, "\n")
		out = append(out, *cur)
	}
	return out
}
