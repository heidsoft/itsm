import { mergeLintIssues } from '../workflowValidation';

describe('mergeLintIssues', () => {
  it('keeps local issues and appends backend warnings', () => {
    const merged = mergeLintIssues(
      [{ type: 'error', message: '本地错误', elementId: 'Task_1' }],
      [{ severity: 'warning', category: 'tasks', message: '未声明 taskPurpose', elementId: 'Task_2' }]
    );

    expect(merged).toHaveLength(2);
    expect(merged[1]).toEqual({
      type: 'warning',
      message: '未声明 taskPurpose',
      elementId: 'Task_2',
      elementName: undefined,
    });
  });

  it('dedupes identical element and message pairs', () => {
    const merged = mergeLintIssues(
      [{ type: 'warning', message: '重复问题', elementId: 'Task_1' }],
      [{ severity: 'warning', category: 'tasks', message: '重复问题', elementId: 'Task_1' }]
    );

    expect(merged).toHaveLength(1);
  });

  it('maps unknown severities to info', () => {
    const merged = mergeLintIssues([], [
      { severity: 'notice', category: 'tasks', message: '提示' },
    ]);

    expect(merged[0].type).toBe('info');
  });
});
