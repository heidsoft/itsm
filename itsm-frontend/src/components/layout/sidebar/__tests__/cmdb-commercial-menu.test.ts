import { getMenuConfig } from '../menu-config';

describe('CMDB commercial menu', () => {
  it('only exposes production-ready CMDB routes', () => {
    const cmdb = getMenuConfig().main.find(item => item.key === '/cmdb');
    const paths = cmdb?.children?.map(item => item.path);

    expect(paths).toEqual([
      '/cmdb/cis',
      '/cmdb/cis/create',
      '/cmdb/relationships',
      '/cmdb/topology',
    ]);
    expect(paths).not.toContain('/cmdb/cloud-resources');
    expect(paths).not.toContain('/cmdb/cloud-accounts');
    expect(paths).not.toContain('/cmdb/reconciliation');
  });
});
