import {
  mergeLintIssues,
  validationUnavailableIssue,
  isValidationUnavailable,
} from '../workflowValidation';

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

describe('校验不可用的 fail-closed 语义', () => {
  it('校验服务不可用产生 error 级 issue，部署门禁 some(type===error) 必然拦截', () => {
    const issues = [validationUnavailableIssue('校验服务暂不可用')];

    // 部署/发布门禁就是这句判定；空数组会误放行，error issue 才会拦截。
    expect(issues.some(i => i.type === 'error')).toBe(true);
    expect(isValidationUnavailable(issues)).toBe(true);
  });

  it('普通流程错误不会被误标成校验不可用', () => {
    const issues = [{ type: 'error' as const, message: '节点未配置', elementId: 'Task_1' }];
    expect(issues.some(i => i.type === 'error')).toBe(true);
    expect(isValidationUnavailable(issues)).toBe(false);
  });

  it('空数组既非不可用也不应被当成校验通过的替代品（门禁调用方不得再返回 []）', () => {
    expect(isValidationUnavailable([])).toBe(false);
    expect([].some(i => (i as { type: string }).type === 'error')).toBe(false);
  });
});
