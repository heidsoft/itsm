export type CMDBCapabilityStatus = 'ga' | 'pilot' | 'disabled';

export type CMDBCapability = {
  key: string;
  title: string;
  description: string;
  href: string;
  status: CMDBCapabilityStatus;
};

/**
 * CMDB 商业能力清单。
 *
 * 只有 GA 能力可以出现在正式导航和 CMDB 首页。pilot/disabled 能力保留
 * 研发路由，但不得作为已交付能力展示，直至通过生产验收。
 */
export const CMDB_CAPABILITIES = [
  {
    key: 'configuration-items',
    title: '配置项管理',
    description: '配置项录入、查询、编辑、生命周期与历史审计。',
    href: '/cmdb/ci',
    status: 'ga',
  },
  {
    key: 'ci-types',
    title: 'CI 类型建模',
    description: '维护类型、字段模板和数据约束。',
    href: '/admin/cmdb-types',
    status: 'ga',
  },
  {
    key: 'relationships',
    title: '关系管理',
    description: '维护依赖、托管和影响关系。',
    href: '/cmdb/relationships',
    status: 'ga',
  },
  {
    key: 'topology',
    title: '拓扑与影响分析',
    description: '查看上下游拓扑，并支撑事件和变更影响分析。',
    href: '/cmdb/topology',
    status: 'ga',
  },
  {
    key: 'cloud-discovery',
    title: '云资源自动发现',
    description: '等待真实连接测试、任务执行、重试与审计闭环验收。',
    href: '/cmdb/registry',
    status: 'pilot',
  },
  {
    key: 'cloud-reconciliation',
    title: '云资源对账',
    description: '等待真实发现数据源和冲突处置闭环验收。',
    href: '/cmdb/reconciliation',
    status: 'pilot',
  },
] as const satisfies readonly CMDBCapability[];

export const CMDB_GA_CAPABILITIES = CMDB_CAPABILITIES.filter(
  capability => capability.status === 'ga',
);

export const isCMDBCapabilityGA = (key: string): boolean =>
  CMDB_CAPABILITIES.some(capability => capability.key === key && capability.status === 'ga');
