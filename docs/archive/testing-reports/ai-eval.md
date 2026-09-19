# AI Evaluation Harness (Stage 2)

本文档定义 ITSM AI 评估器的离线指标集（golden cases + 评估函数）。
对应 v1.1 收尾计划阶段 2：PR-2.1（骨架）、PR-2.2（指标）、PR-2.3（RBAC 测试）。

## 当前状态（v1.1 收尾）

| 阶段 | 任务 | 状态 |
|------|------|------|
| 2.1 | `itsm-backend/ai/eval/` 评估集骨架 | ✅ 完成 |
| 2.2 | Triage / Summarize / RAG / Prediction 评估器 | 🟡 骨架完成（占位 fixture） |
| 2.3 | AI 工具权限与审计契约测试 | ✅ 已有 (`handlers/ai/service_rbac_test.go`) |

## 目录结构

```
itsm-backend/ai/eval/
├── doc.go              # 包文档
├── eval_test.go        # 评估入口（go test ./ai/eval/...）
└── datasets/           # golden case JSONL
    ├── triage.jsonl     # 10 cases
    ├── summarize.jsonl  # 5 cases
    ├── rag.jsonl        # 10 cases
    └── prediction.jsonl # 5 cases
```

## 指标与门槛（v1.5 目标）

| 评估项 | 指标 | 门槛 | 当前 |
|--------|------|------|------|
| Triage | top-1 accuracy | ≥85% | 100% (eval-mode fixture) |
| Triage | top-3 accuracy | ≥95% | N/A (待 fixture 升级) |
| Triage | priority 偏差 | ≤1 | N/A (待 fixture 升级) |
| Summarize | ROUGE-L | ≥0.6 | 0.93 (topic coverage 简化) |
| Summarize | hallucination ratio | ≤5% | N/A (待 fixture 升级) |
| RAG | hit-rate | ≥70% | 100% (占位) |
| RAG | MRR | ≥0.5 | N/A (待 fixture 升级) |
| RAG | tenant 隔离违规 | =0 | ✅ (强制 tenant_id) |
| Prediction | ROC AUC | ≥0.75 | SKIP (占位) |

## 运行方式

```bash
# 在 itsm-backend/ 目录
GOTOOLCHAIN=auto go test -v ./ai/eval/...
```

输出示例：

```
=== RUN   TestEval_Triage_Top1Accuracy
    eval_test.go:259: triage top-1 accuracy: 1.00 (cases=10)
--- PASS: TestEval_Triage_Top1Accuracy
=== RUN   TestEval_Summarize_ROUGE
    eval_test.go:269: summarize ROUGE-L (topic coverage): 0.93 (cases=5)
--- PASS: TestEval_Summarize_ROUGE
=== RUN   TestEval_RegressionSnapshots
    eval_test.go:318:   triage       cases=10  top1     = 1.000
    eval_test.go:318:   summarize    cases=5   rougeL   = 0.933
    eval_test.go:318:   rag          cases=10  hitRate  = 1.000
    eval_test.go:318:   prediction   cases=5   rocAuc   = 0.000
--- PASS: TestEval_RegressionSnapshots
```

## 扩 case 流程

1. 在对应 `.jsonl` 文件追加一行 JSON（保持与现有 schema 一致）
2. 跑 `go test ./ai/eval/...` 验证
3. 若新增指标维度，更新 `eval_test.go` 中的 `TriageCase`/`SummarizeCase` 等 struct
4. 把修改后输出 attach 到 PR description

## 接入 LLM gateway（v1.5 计划）

当前 `EvalTriage` 使用 deterministic keyword 匹配；v1.5 接入：
- `service/llm_gateway.go` 引入 `--eval-mode`
- `--eval-mode=true` 时：`EvalTriage` 改为 `llm_gateway.Chat(model="triage-eval", ...)`
- 真实 LLM 输出与 golden 期望对比：top-1 / ROUGE / hit-rate / ROC AUC

## CI 接入

```yaml
# .github/workflows/ai-eval.yml（计划中）
- name: AI eval regression
  run: GOTOOLCHAIN=auto go test -v ./ai/eval/... --fail-under=85%
```

门禁 `--fail-under=85%` 要求所有任务指标 ≥0.85（除已被 `t.Skip` 的占位 prediction 外）。

## 与 ROADMAP 对齐

- v1.1 收尾：triage=200 / summarize=100 / rag=150 / prediction=80 cases
- 当前 seed：triage=10 / summarize=5 / rag=10 / prediction=5（骨架用）
- 扩到 ROADMAP 目标量是 PR-2.2 后续工作（v1.5 窗口）

## 失败模式

| 现象 | 排查 |
|------|------|
| `triage top-1 accuracy` < 0.85 | 关键词表覆盖不足 → 扩 `EvalTriage` switch case |
| `summarize ROUGE` < 0.6 | 期望 topic 列表过长 → 调整 max_length |
| `RAG hit-rate` < 0.7 | expected_doc_ids 与 fixture 不匹配 → 检查 stub |
| `prediction ROC AUC` 实际 0.0 | 已知占位 → 等 prediction service 接入后启用 |
