package service_request

import (
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/service"
)

// newTestService 构造带真实 BPMN 桥接的服务请求域服务。
//
// 改造前本域在 NewService 内部构造共享层桥接（entClient 非 nil 即非 nil），
// 构造上移组合根后夹具必须显式注入——否则 approvalBridge 为 nil，
// 「完成业务作业节点 / 终止残留流程实例」等路径会 panic 或被跳过，
// 测试断言的流程语义会静默消失。本辅助函数让所有夹具保持与改造前一致的装配口径。
func newTestService(repo Repository, scRepo service_catalog.Repository, cmdbRepo cmdb.Repository, entClient *ent.Client, logger *zap.SugaredLogger, approvalChain ApprovalChainResolver) *Service {
	var opts []ServiceOption
	if entClient != nil {
		opts = append(opts, WithApprovalBridge(service.NewBPMNApprovalBridge(entClient, logger)))
	}
	return NewService(repo, scRepo, cmdbRepo, entClient, logger, approvalChain, opts...)
}
