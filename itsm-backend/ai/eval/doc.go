// Package eval provides the AI evaluation harness (Stage 2 of the ITSM test
// improvement plan). The harness reads golden cases from JSONL files under
// ./datasets/ and computes evaluation metrics against the deterministic
// eval-mode LLM gateway.
//
// 当前状态（2026-10-31）：
//   - Triage / Summarize / RAG / Prediction 四类 golden case 已 seed
//   - 关键指标：top-1 accuracy / ROUGE-L / hit-rate / ROC AUC
//
// ## 评测模式
//
// 只有一种 CI 评测模式（degraded）：候选来自**生产代码路径**——摘要走
// SummarizeService 的无 LLM 降级路径（截断）。不需要外部依赖，且测的是真实
// 的生产行为。
//
// RAG 一项已升级为**真实检索**：语料（真实知识文章）建进内存库，用生产的
// service.RAGService 关键词路径检索（真实 SQL + 真实租户/发布/软删/时效
// 过滤）。此前期望写的是虚构 ID（KB-001 等，库里不存在），任何真实检索器
// 的命中率必然为 0；而用期望答案当候选又让指标恒满分——两头都不成立。
// 现在期望按**文章标题**引用（与库的 ID 解耦），同一份数据集既能在 CI 内存库
// 跑，也能在接了真实知识库的生产验收里跑。实测召回率 0.80。
//
// 不提供“设置环境变量即生产验收”的开关：此前 ITSM_AI_EVAL_USE_LLM 只会
// 切换日志标签，候选仍来自降级路径，等于对外宣称一个不存在的验收模式。
// 真实 LLM 质量验收需要真实语料与真实候选生成器，落地后再引入明确的开关。
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
