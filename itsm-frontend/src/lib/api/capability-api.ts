import { httpClient } from './http-client';

export type CapabilityMaturity = 'ga' | 'pilot' | 'disabled';

/**
 * 能力健康信号。
 *
 * 与 lastHealthCheckAt 的区别：后者回答「何时探测」，这里回答「探测结果」。
 * 后端在没有探测维度时省略整个字段（不伪造健康），因此可选。
 */
export interface CapabilityHealth {
  ok: boolean;
  reason?: string;
  checkedAt: string;
}

export interface ProductCapabilityState {
  key: string;
  maturity: CapabilityMaturity;
  buildAvailable: boolean;
  deploymentReady: boolean;
  tenantReady: boolean;
  allowedActions: string[];
  dependencies: string[];
  degradedReason?: string;
  lastHealthCheckAt?: string;
  health?: CapabilityHealth;
  /**
   * 为什么还不是 GA，以及晋升需要补哪块可验证证据。
   * 只有 maturity=pilot 时下发；GA 条目带理由会自相矛盾，后端不下发。
   */
  pilotReason?: string;
  acceptanceVersion: string;
}

export interface CapabilityResponse {
  items: ProductCapabilityState[];
  acceptanceVersion: string;
}

export const getCapabilities = (): Promise<CapabilityResponse> =>
  httpClient.get<CapabilityResponse>('/api/v1/capabilities');

/**
 * 能力可见性统一判据（fail-closed）：侧栏菜单与顶栏入口共用。
 *
 * 与 capabilityAllows 的差别：这里只回答「这个部署形态有没有这个能力」，
 * 不判具体 action。缺条目 / maturity=disabled / 任一 ready 位为 false 都视为不可见，
 * 避免「侧栏整条消失、顶栏按钮却永远亮着」的双口径。
 */
export const isCapabilityVisible = (capability: ProductCapabilityState | undefined): boolean =>
  Boolean(
    capability &&
      capability.maturity !== 'disabled' &&
      capability.buildAvailable &&
      capability.deploymentReady &&
      capability.tenantReady
  );

export const capabilityAllows = (
  capabilities: ProductCapabilityState[],
  key: string,
  action = 'read'
): boolean => {
  const capability = capabilities.find(item => item.key === key);
  return isCapabilityVisible(capability) && Boolean(capability?.allowedActions.includes(action));
};
