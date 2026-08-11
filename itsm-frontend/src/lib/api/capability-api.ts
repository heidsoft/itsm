import { httpClient } from './http-client';

export type CapabilityMaturity = 'ga' | 'pilot' | 'disabled';

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
