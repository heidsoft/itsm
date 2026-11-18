// 设计器新拖入的节点默认没有 name，待办列表会退化成英文 taskDefinitionKey 直出。
// 这里按 BPMN 类型给一个自然的中文默认名，用户随后可在属性面板改名。
// 仅写 name（展示用），不改节点 ID，引擎/会签/连线引用不受影响。

const defaultNames: Record<string, string> = {
  'bpmn:UserTask': '人工任务',
  'bpmn:ServiceTask': '自动任务',
  'bpmn:ScriptTask': '脚本任务',
  'bpmn:SendTask': '发送任务',
  'bpmn:ReceiveTask': '接收任务',
  'bpmn:ManualTask': '手动任务',
  'bpmn:BusinessRuleTask': '规则任务',
  'bpmn:CallActivity': '调用活动',
  'bpmn:SubProcess': '子流程',
  'bpmn:Transaction': '事务',
  'bpmn:ExclusiveGateway': '排他网关',
  'bpmn:ParallelGateway': '并行网关',
  'bpmn:InclusiveGateway': '包容网关',
  'bpmn:ComplexGateway': '复杂网关',
  'bpmn:EventBasedGateway': '事件网关',
  'bpmn:StartEvent': '开始',
  'bpmn:EndEvent': '结束',
  'bpmn:IntermediateCatchEvent': '中间事件',
  'bpmn:IntermediateThrowEvent': '中间事件',
  'bpmn:BoundaryEvent': '边界事件',
};

export function defaultNodeName(type: string): string | null {
  return defaultNames[type] ?? null;
}