import { KPI_UNKNOWN, kpiValue } from '../certification-kpi';

/**
 * 数据认证 KPI 的「未知 vs 0」区分（回归锁）。
 *
 * 反向验证：把 kpiValue 改回 `value ?? 0`，前两条会红（undefined/null 会返回 0），
 * 而「真实 0 显示 0」仍绿——三条断言合起来才能表达「失败不等于 0 条待办」。
 */
describe('certification kpiValue', () => {
  it('未取到数据时返回未知占位，不得回落到 0', () => {
    expect(kpiValue(undefined)).toBe(KPI_UNKNOWN);
    expect(kpiValue(null)).toBe(KPI_UNKNOWN);
    expect(kpiValue(undefined)).not.toBe(0);
  });

  it('后端真实下发 0 时显示 0（与「不知道」区分开）', () => {
    expect(kpiValue(0)).toBe(0);
  });

  it('正常数值原样透出', () => {
    expect(kpiValue(7)).toBe(7);
    expect(kpiValue(1203)).toBe(1203);
  });
});
