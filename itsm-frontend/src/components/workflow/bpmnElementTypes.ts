/**
 * BPMN 元素类型词表：画布类型（bpmn:UserTask）到展示名/搜索别名的唯一映射。
 */

const SHORT_TYPE_I18N_KEY: Record<string, string> = {
  StartEvent: 'bpmnDesigner.nodeTypes.startEvent',
  EndEvent: 'bpmnDesigner.nodeTypes.endEvent',
  IntermediateThrowEvent: 'bpmnDesigner.nodeTypes.intermediateThrowEvent',
  IntermediateCatchEvent: 'bpmnDesigner.nodeTypes.intermediateCatchEvent',
  BoundaryEvent: 'bpmnDesigner.nodeTypes.boundaryEvent',
  UserTask: 'bpmnDesigner.nodeTypes.userTask',
  ServiceTask: 'bpmnDesigner.nodeTypes.serviceTask',
  ScriptTask: 'bpmnDesigner.nodeTypes.scriptTask',
  ManualTask: 'bpmnDesigner.nodeTypes.manualTask',
  BusinessRuleTask: 'bpmnDesigner.nodeTypes.businessRuleTask',
  SendTask: 'bpmnDesigner.nodeTypes.sendTask',
  ReceiveTask: 'bpmnDesigner.nodeTypes.receiveTask',
  CallActivity: 'bpmnDesigner.nodeTypes.callActivity',
  SubProcess: 'bpmnDesigner.nodeTypes.subProcess',
  Transaction: 'bpmnDesigner.nodeTypes.transaction',
  AdHocSubProcess: 'bpmnDesigner.nodeTypes.adHocSubProcess',
  ExclusiveGateway: 'bpmnDesigner.nodeTypes.exclusiveGateway',
  ParallelGateway: 'bpmnDesigner.nodeTypes.parallelGateway',
  InclusiveGateway: 'bpmnDesigner.nodeTypes.inclusiveGateway',
  ComplexGateway: 'bpmnDesigner.nodeTypes.complexGateway',
  SequenceFlow: 'bpmnDesigner.nodeTypes.sequenceFlow',
};

export const toShortType = (elementType: string | undefined): string => {
  if (!elementType) return '';
  const idx = elementType.lastIndexOf(':');
  return idx >= 0 ? elementType.slice(idx + 1) : elementType;
};

export const typeSearchTerms = (elementType: string, translatedLabel: string): string[] => {
  const shortType = toShortType(elementType);
  const terms = [shortType.toLowerCase(), elementType.toLowerCase().replace(':', '')];
  if (translatedLabel) terms.push(translatedLabel.toLowerCase());
  return Array.from(new Set(terms.filter(Boolean)));
};

export const formatElementType = (
  elementType: string,
  translate: (key: string) => string
): string => {
  const shortType = toShortType(elementType);
  const key = SHORT_TYPE_I18N_KEY[shortType];
  if (!key) return shortType;
  const label = translate(key);
  // key 未注册时 useI18n 原样返回 key，此时展示英文名而不是点路径
  return label && label !== key ? label : shortType;
};
