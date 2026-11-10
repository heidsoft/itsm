package change

import (
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service"
)

// newTestService 构造带真实 BPMN 桥接与审批链收口函数的变更域服务。
//
// 改造前本域在 NewService 内部构造共享层桥接（entClient 非 nil 即非 nil），
// 构造上移组合根后夹具必须显式注入——否则 approvalBridge 为 nil，桥接路径
// 被跳过，测试断言的「业务写入与流程推进同事务」语义会静默消失。
// 本辅助函数让所有夹具保持与改造前一致的装配口径。
func newTestService(repo Repository, entClient *ent.Client, logger *zap.SugaredLogger, approvalChain ApprovalChainResolver) *Service {
	var opts []ServiceOption
	if entClient != nil {
		opts = append(opts, WithApprovalBridge(service.NewBPMNApprovalBridge(entClient, logger)))
	}
	opts = append(opts, WithApprovalChainCloser(service.CloseChangeApprovalChains))
	return NewService(repo, entClient, logger, approvalChain, opts...)
}
