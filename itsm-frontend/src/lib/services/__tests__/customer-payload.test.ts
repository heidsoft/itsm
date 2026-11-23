/**
 * 客户/分支提交体的类型契约（回归锁）。
 *
 * 后端 `customerRequest.Aliases` 是 `[]string`（`handlers/email_intake/handler.go:76`），
 * 但页面表单曾用普通 Input 收「逗号分隔」的字符串直传。Gin 的 `ShouldBindJSON`
 * 遇到字符串到 []string 的类型不匹配会直接 400，所以**新增/编辑客户必然失败**，
 * 且错误信息只有一句笼统的「保存失败」。
 *
 * 修法是把别名定义成 `string[]`（Select tags 模式收集），提交体只取接口声明字段。
 * 本测试锁三件事：
 *   ① 别名始终是数组（空表单得到 []，不是 undefined/空串）；
 *   ② 提交体不含 id/createdAt/updatedAt 这类回读字段；
 *   ③ 编辑时历史名原样带回（接口是整体替换语义，漏传等于清空）。
 *
 * 反向验证：把别名改回 `values.aliases ?? ''`，①②会红。
 */

import {
  CUSTOMER_FORM_EMPTY_HINT,
  toCustomerPayload,
  toBranchPayload,
} from '../customer-payload';

describe('customer payload', () => {
  it('别名空表单时提交空数组而非 undefined / 空串', () => {
    const payload = toCustomerPayload({ name: '北京某某科技有限公司' }, 'active');
    expect(Array.isArray(payload.aliases)).toBe(true);
    expect(payload.aliases).toEqual([]);
    // 后端 bind 要求 []string；空串会被判成类型错误 400
    expect(payload.aliases).not.toBe('');
  });

  it('别名正常透出，且 name/shortName 做去空白', () => {
    const payload = toCustomerPayload(
      { name: '  某某科技  ', shortName: ' 简称 ', aliases: ['某某', '某科技'] },
      'active'
    );
    expect(payload.name).toBe('某某科技');
    expect(payload.shortName).toBe('简称');
    expect(payload.aliases).toEqual(['某某', '某科技']);
  });

  it('提交体不得包含 id/createdAt/updatedAt 回读字段', () => {
    const payload = toCustomerPayload({ name: 'A' }, 'active');
    expect(Object.keys(payload).sort()).toEqual(
      ['aliases', 'historicalNames', 'name', 'shortName', 'status'].sort()
    );
  });

  it('编辑时历史名原样带回（整体替换语义下漏传等于清空）', () => {
    const payload = toCustomerPayload({ name: 'A' }, 'active', ['旧名一', '旧名二']);
    expect(payload.historicalNames).toEqual(['旧名一', '旧名二']);
    // 新建时不传 = 空数组，同样是数组类型
    expect(toCustomerPayload({ name: 'B' }, 'active').historicalNames).toEqual([]);
  });

  it('分支提交体同样保证别名为数组且带 customerId', () => {
    const payload = toBranchPayload({ name: '上海分公司' }, 42, 'active');
    expect(payload.aliases).toEqual([]);
    expect(payload.customerId).toBe(42);
    expect(payload.status).toBe('active');
  });

  it('别名输入提示指向「回车/粘贴多值」，而不是让用户手写分隔符再靠 split', () => {
    // 提示若退化成「请输入别名」之类，用户不知道可以一次录入多条；
    // 这里只需要保证提示本身表达了多值录入方式（粘贴逗号是允许的便捷路径，
    // 但控件收集的仍是数组，不再由前端 split 字符串）
    expect(CUSTOMER_FORM_EMPTY_HINT).toContain('回车');
  });
});
