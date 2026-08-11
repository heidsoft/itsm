import { capabilityAllows, type ProductCapabilityState } from '../capability-api';

const capability = (patch: Partial<ProductCapabilityState> = {}): ProductCapabilityState => ({
  key: 'marketplace',
  maturity: 'pilot',
  buildAvailable: true,
  deploymentReady: true,
  tenantReady: true,
  allowedActions: ['read'],
  dependencies: [],
  acceptanceVersion: 'test',
  ...patch,
});

describe('capabilityAllows', () => {
  it('allows only an action granted by the backend control plane', () => {
    expect(capabilityAllows([capability()], 'marketplace', 'read')).toBe(true);
    expect(capabilityAllows([capability()], 'marketplace', 'manage')).toBe(false);
  });

  it.each([
    { maturity: 'disabled' as const },
    { buildAvailable: false },
    { deploymentReady: false },
    { tenantReady: false },
  ])('fails closed when capability is not product-ready: %o', patch => {
    expect(capabilityAllows([capability(patch)], 'marketplace')).toBe(false);
  });

  it('fails closed when capability state is missing', () => {
    expect(capabilityAllows([], 'marketplace')).toBe(false);
  });
});
