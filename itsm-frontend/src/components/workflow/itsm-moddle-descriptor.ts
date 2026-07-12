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
  }],
} as const;

export default itsmModdleDescriptor;
