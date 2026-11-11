package service

import "itsm-backend/internal/bpmncontracts"

// BPMN 纯契约过渡 API（G0 归位，2026-11-11）。
//
// 实现本体已迁入 internal/bpmncontracts：值类型、XML 标签、GetID/GetName/GetType
// 方法，以及四个流程服务接口都只在契约包里定义一份。本文件只保留类型别名，
// 让 service 包内既有调用点与组合根零改动，并保证类型身份不分裂。
// 禁止在此重新定义 struct/interface、复制实现或另起一份常量口径。
type (
	BPMNElement                     = bpmncontracts.BPMNElement
	BPMNProcess                     = bpmncontracts.BPMNProcess
	BPMNOutgoingRef                 = bpmncontracts.BPMNOutgoingRef
	BPMNStartEvent                  = bpmncontracts.BPMNStartEvent
	BPMNEndEvent                    = bpmncontracts.BPMNEndEvent
	BPMNCompensateEventDefinition   = bpmncontracts.BPMNCompensateEventDefinition
	BPMNTimerEventDefinition        = bpmncontracts.BPMNTimerEventDefinition
	BPMNUserTask                    = bpmncontracts.BPMNUserTask
	BPMNServiceTask                 = bpmncontracts.BPMNServiceTask
	BPMNScriptTask                  = bpmncontracts.BPMNScriptTask
	BPMNExclusiveGateway            = bpmncontracts.BPMNExclusiveGateway
	BPMNParallelGateway             = bpmncontracts.BPMNParallelGateway
	BPMNInclusiveGateway            = bpmncontracts.BPMNInclusiveGateway
	BPMNSequenceFlow                = bpmncontracts.BPMNSequenceFlow
	BPMNConditionExpression         = bpmncontracts.BPMNConditionExpression
	BPMNSubProcess                  = bpmncontracts.BPMNSubProcess
	BPMNBusinessRuleTask            = bpmncontracts.BPMNBusinessRuleTask
	BPMNManualTask                  = bpmncontracts.BPMNManualTask
	BPMNCallActivity                = bpmncontracts.BPMNCallActivity
	BPMNDataObject                  = bpmncontracts.BPMNDataObject
	BPMNDataStore                   = bpmncontracts.BPMNDataStore
	BPMNBoundaryEvent               = bpmncontracts.BPMNBoundaryEvent
	BPMNIntermediateEvent           = bpmncontracts.BPMNIntermediateEvent
	BPMNDefinitions                 = bpmncontracts.BPMNDefinitions
	ProcessTriggerServiceInterface  = bpmncontracts.ProcessTriggerServiceInterface
	ProcessBindingServiceInterface  = bpmncontracts.ProcessBindingServiceInterface
	ServiceTaskHandlerInterface     = bpmncontracts.ServiceTaskHandlerInterface
	ProcessCallbackServiceInterface = bpmncontracts.ProcessCallbackServiceInterface
)
