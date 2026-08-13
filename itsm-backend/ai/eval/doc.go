// Package eval provides the AI evaluation harness (Stage 2 of the ITSM test
// improvement plan). The harness reads golden cases from JSONL files under
// ./datasets/ and computes evaluation metrics against the deterministic
// eval-mode LLM gateway.
//
// 当前骨架（v1.1 收尾）：
//   - Triage / Summarize / RAG / Prediction 四类 golden case 已 seed
//   - Eval 模式：使用 deterministic fixture 替代真实 LLM，避免外部依赖
//   - 关键指标：top-1 accuracy / ROUGE-L / hit-rate / ROC AUC
//
// 后续 PR（v1.5）：
//   - 用 LLM gateway --eval-mode 替换占位 fixture
//   - CI 接入 --fail-under=85% 门禁
package eval
