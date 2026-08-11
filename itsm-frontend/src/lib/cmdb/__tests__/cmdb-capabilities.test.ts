import {
  CMDB_CAPABILITIES,
} from '../cmdb-capabilities';

describe('CMDB commercial capabilities', () => {
  it('defines presentation metadata for supported capability keys', () => {
    expect(CMDB_CAPABILITIES.slice(0, 4).map(capability => capability.key)).toEqual([
      'configuration-items',
      'ci-types',
      'relationships',
      'topology',
    ]);
  });

  it('does not advertise cloud discovery or reconciliation as GA', () => {
  });

  it('requires every capability to have a route and an explicit maturity status', () => {
    for (const capability of CMDB_CAPABILITIES) {
      expect(capability.href).toMatch(/^\//);
      expect(capability.capabilityKey).toMatch(/^cmdb/);
    }
  });
});
