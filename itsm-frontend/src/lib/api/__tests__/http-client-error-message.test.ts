/**
 * `serverErrorMessage` 契约：服务端可行动原因优先，未知错误回退本地文案。
 *
 * 这条规则来自一个真实缺陷：变更详情页的审批/流转失败一律提示「操作失败」，
 * 后端已经返回的 409「该变更未绑定运行中的审批流程实例…请配置审批流程模板」
 * 被前端覆盖掉，使用者无从判断该改配置还是该重试。
 */

import { ApiRequestError, serverErrorMessage, isPermissionDeniedError } from '../http-client';

describe('serverErrorMessage', () => {
  it('透传后端业务拒绝的 message（409 会签缺流程实例）', () => {
    const err = new ApiRequestError(
      'approval quorum requires a running workflow instance: 该变更未绑定运行中的审批流程实例，审批决定无法持久化',
      409,
      4090
    );
    expect(serverErrorMessage(err, '操作失败')).toBe(
      'approval quorum requires a running workflow instance: 该变更未绑定运行中的审批流程实例，审批决定无法持久化'
    );
  });

  it('透传 403 非审批人提示，而不是折叠成本地文案', () => {
    const err = new ApiRequestError('您不是该变更的待办审批人', 403, 2003);
    expect(serverErrorMessage(err, '审批失败')).toBe('您不是该变更的待办审批人');
  });

  it('message 为空的 ApiRequestError 回退本地文案（避免提示空白）', () => {
    const err = new ApiRequestError('', 500);
    expect(serverErrorMessage(err, '操作失败')).toBe('操作失败');
  });

  it('普通 Error 与未知值一律回退本地文案（网络错误没有业务原因）', () => {
    expect(serverErrorMessage(new Error('Failed to fetch'), '操作失败')).toBe('操作失败');
    expect(serverErrorMessage('boom', '操作失败')).toBe('操作失败');
    expect(serverErrorMessage(undefined, '操作失败')).toBe('操作失败');
  });

  it('不改变既有权限判定的语义', () => {
    expect(isPermissionDeniedError(new ApiRequestError('无权限', 403))).toBe(true);
    expect(isPermissionDeniedError(new ApiRequestError('无权限', 200, 2003))).toBe(true);
    expect(isPermissionDeniedError(new ApiRequestError('冲突', 409, 4090))).toBe(false);
  });
});
