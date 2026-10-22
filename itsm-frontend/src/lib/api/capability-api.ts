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

export const capabilityAllows = (
  capabilities: ProductCapabilityState[],
  key: string,
  action = 'read'
): boolean => {
  const capability = capabilities.find(item => item.key === key);
  return Boolean(
    capability &&
      capability.maturity !== 'disabled' &&
      capability.buildAvailable &&
      capability.deploymentReady &&
      capability.tenantReady &&
      capability.allowedActions.includes(action)
  );
};
