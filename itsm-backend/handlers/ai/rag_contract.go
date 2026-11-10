package ai

import (
	"context"

	"itsm-backend/handlers/knowledge"
)

// RAGChatService 是 AI 域对知识检索能力的窄契约（依赖倒置）。
//
// 历史上 Service.rag 直接持有共享层 *service.RAGService 具体指针：共享层任何
// 改动都会波及 AI 域，域包也无法在不引入共享层的前提下做单元测试。这里按 AI
// 域真实消费面收敛为三个方法——SearchKnowledge 用 Ask、Chat 用 AskMulti、
// 决策底账用 Mode；Ask 直接复用知识域现成的 knowledge.RAGRetriever，保证两个
// 域对「一次检索」的口径只有一份定义，不另起一份同签名接口。
//
// 流式问答（AskWithLLMStreamWithToolsMulti）现以 dto 端口与 dto 值类型声明，
// 由可选能力接口 ragStreamer 做类型断言。该接口本批仍留在 service.go，
// 因为既有回归锁对声明位置有断言；AI 域整体归零批次再并入本契约文件。
// 注意 C.8 / L.3 守卫是全文 grep 计数：即使在注释里写出带引号的共享包
// 路径也会被计入，因此本文件连注释都不得出现该字面量。
//
// 契约边界：新增方法前先确认 AI 域确实消费，禁止把共享层大接口原样搬进域包。

// RAGChatService 是 AI 问答/检索读面所需的最小契约。
type RAGChatService interface {
	knowledge.RAGRetriever
	AskMulti(ctx context.Context, tenantID int, query string, supplemental []string, limit int) ([]map[string]any, error)
	Mode() string
}
