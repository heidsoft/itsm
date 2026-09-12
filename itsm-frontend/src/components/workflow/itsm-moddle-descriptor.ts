const itsmModdleDescriptor = {
  name: 'ITSM',
  uri: 'https://github.com/heidsoft/itsm/schema/bpmn',
  prefix: 'itsm',
  xml: { tagAlias: 'lowerCase' },
  types: [{
    name: 'ApprovalUserTask',
    extends: ['bpmn:UserTask'],
    properties: [
      { name: 'assignee', isAttr: true, type: 'String' },
      { name: 'candidateUsers', isAttr: true, type: 'String' },
      { name: 'candidateGroups', isAttr: true, type: 'String' },
      { name: 'priority', isAttr: true, type: 'String' },
      { name: 'formKey', isAttr: true, type: 'String' },
      { name: 'dueDate', isAttr: true, type: 'String' },
      { name: 'taskPurpose', isAttr: true, type: 'String' },
      { name: 'approvalMode', isAttr: true, type: 'String' },
      { name: 'approvalThreshold', isAttr: true, type: 'Integer' },
      { name: 'rejectStrategy', isAttr: true, type: 'String' },
      { name: 'timeoutAction', isAttr: true, type: 'String' },
      { name: 'allowDelegate', isAttr: true, type: 'Boolean' },
      { name: 'allowAddApprover', isAttr: true, type: 'Boolean' },
      { name: 'commentRequiredOnReject', isAttr: true, type: 'Boolean' },
    ],
  }, {
    // 后端 BPMNServiceTask 已按局部名解析这些属性。不在此声明时 bpmn-moddle 会把它们
    // 归入 businessObject.$attrs，面板回读恒为 undefined（配置看似丢失）。
    // 禁止声明 operationRef 等 BPMN 标准属性名：重名会让整个元素解析失败。
    name: 'NotifyServiceTask',
    extends: ['bpmn:ServiceTask'],
    properties: [
      { name: 'ccType', isAttr: true, type: 'String' },
      { name: 'ccUserIds', isAttr: true, type: 'String' },
      { name: 'ccGroupIds', isAttr: true, type: 'String' },
      { name: 'ccRoleIds', isAttr: true, type: 'String' },
      { name: 'ccVariable', isAttr: true, type: 'String' },
      { name: 'ccNotify', isAttr: true, type: 'String' },
      { name: 'notifyChannels', isAttr: true, type: 'String' },
    ],
  }],
} as const;

export default itsmModdleDescriptor;
