import {
  CMDB_CAPABILITIES,
  CMDB_GA_CAPABILITIES,
  isCMDBCapabilityGA,
} from '../cmdb-capabilities';

describe('CMDB commercial capabilities', () => {
  it('only exposes production-accepted capabilities as GA', () => {
    expect(CMDB_GA_CAPABILITIES.map(capability => capability.key)).toEqual([
      'configuration-items',
      'ci-types',
      'relationships',
      'topology',
    ]);
  });

  it('does not advertise cloud discovery or reconciliation as GA', () => {
    expect(isCMDBCapabilityGA('cloud-discovery')).toBe(false);
    expect(isCMDBCapabilityGA('cloud-reconciliation')).toBe(false);
  });

  it('requires every capability to have a route and an explicit maturity status', () => {
    for (const capability of CMDB_CAPABILITIES) {
      expect(capability.href).toMatch(/^\//);
      expect(['ga', 'pilot', 'disabled']).toContain(capability.status);
    }
  });
});
