export type CMDBCapability = {
  key: string;
  title: string;
  description: string;
  href: string;
  capabilityKey: string;
};

/**
 * CMDB 展示元数据。成熟度和 readiness 只来自后端 capability 控制平面。
 */
export const CMDB_CAPABILITIES = [
  {
    key: 'configuration-items',
    title: '配置项管理',
    description: '配置项录入、查询、编辑、生命周期与历史审计。',
    href: '/cmdb/ci',
    capabilityKey: 'cmdb.configurationItems',
  },
  {
    key: 'ci-types',
    title: 'CI 类型建模',
    description: '维护类型、字段模板和数据约束。',
    href: '/admin/cmdb-types',
    capabilityKey: 'cmdb.ciTypes',
  },
  {
    key: 'relationships',
    title: '关系管理',
    description: '维护依赖、托管和影响关系。',
    href: '/cmdb/relationships',
    capabilityKey: 'cmdb.relationships',
  },
  {
    key: 'topology',
    title: '拓扑与影响分析',
    description: '查看上下游拓扑，并支撑事件和变更影响分析。',
    href: '/cmdb/topology',
    capabilityKey: 'cmdb.topology',
  },
  {
    key: 'cloud-discovery',
    title: '云资源自动发现',
    description: '等待真实连接测试、任务执行、重试与审计闭环验收。',
    href: '/cmdb/registry',
    capabilityKey: 'cmdbDiscovery',
  },
  {
    key: 'cloud-reconciliation',
    title: '云资源对账',
    description: '等待真实发现数据源和冲突处置闭环验收。',
    href: '/cmdb/reconciliation',
    capabilityKey: 'cmdbReconciliation',
  },
] as const satisfies readonly CMDBCapability[];
