import itsmModdleDescriptor from '../itsm-moddle-descriptor';

describe('ITSM BPMN moddle descriptor', () => {
  it('persists assignment, approval and countersign attributes', () => {
    const properties = itsmModdleDescriptor.types[0].properties.map(property => property.name);
    expect(properties).toEqual(expect.arrayContaining([
      'assignee', 'candidateUsers', 'candidateGroups', 'taskPurpose',
      'approvalMode', 'approvalThreshold', 'rejectStrategy', 'timeoutAction',
      'allowDelegate', 'allowAddApprover', 'commentRequiredOnReject',
    ]));
  });
});
