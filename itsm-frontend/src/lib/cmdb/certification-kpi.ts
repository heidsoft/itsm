/**
 * 数据认证页 KPI 数值的「未知」表达。
 *
 * 页面曾直接写 `data?.pending ?? 0`：加载失败时 `data` 为 null，四个卡片于是都显示 0，
 * 看起来像「一条待办都没有、认证全部做完」。对认证体系这是最危险的假绿——运维会
 * 因此不做任何核对就认为数据是新鲜的。
 *
 * 规则只有一条：**没拿到数就显示未知，不显示 0**。只有后端真实下发 0 才显示 0。
 * 抽成独立模块而不是页面内联，是为了能用行为测试锁住这条规则（页面内联时只能做
 * 源码字符串断言，任何一处写成 `?? 0` 都容易被「只要文件里出现过该词」之类的宽松
 * 写法放过）。
 */

/** Statistic 接受 number | string；未取到值返回全角破折号占位。 */
export const KPI_UNKNOWN = '—';

export function kpiValue(value: number | undefined | null): number | string {
  return value === undefined || value === null ? KPI_UNKNOWN : value;
}
