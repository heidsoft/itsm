// Package eval provides the AI evaluation harness (Stage 2 of the ITSM test
// improvement plan). The harness reads golden cases from JSONL files under
// ./datasets/ and computes evaluation metrics against the deterministic
// eval-mode LLM gateway.
//
// 当前状态（2026-10-31）：
//   - Triage / Summarize / RAG / Prediction 四类 golden case 已 seed
//   - 关键指标：top-1 accuracy / ROUGE-L / hit-rate / ROC AUC
//
// ## 两种评测模式（EvalMode）
//
//	degraded —— CI 默认。候选来自**生产代码路径**：摘要走 SummarizeService 的
//	            无 LLM 降级路径（截断），检索走 DegradedRAGCandidates（部分命中
//	            期望的确定性基线）。不需要外部依赖，且测的是真实的生产行为。
//	llm     —— 生产验收。设 ITSM_AI_EVAL_USE_LLM=1 后候选来自真实 LLM，
//	            产出真实质量分。
//
// ## 门禁有效性（2026-10-31 修复）
//
// 此前摘要与 RAG 两类门禁是**自我实现**的：摘要候选 = 输入拼接、RAG 候选 =
// 期望文档 ID。实测三项指标恒为满分（ROUGE 0.84 / hit-rate 1.00 / AUC 1.00），
// 且把真实实现改坏门禁照样绿——那不是宽松的门禁，是**没有门禁**：它让
// 「AI 质量有 CI 保护」变成不实陈述。
//
// 现在 harness.go 提供候选真实性检测（DetectGoldenText / DetectGoldenDocIDs），
// gate_guard_test.go 守三件事：候选不得是期望答案、候选退化时指标必须下降、
// 门禁函数不得用期望答案充当候选（源码扫描）。
//
// ## 仍需真实语料验收
//
// RAG 的 expected_doc_ids 是虚构文档 ID（KB-001 等），库里不存在——任何真实
// 检索器在 CI 里命中率必然为 0。因此 CI 只做候选真实性检查，质量验收留给
// 接了真实语料的生产环境。
package eval
