export type CMDBCapability = {
  key: string;
  title: string;
  description: string;
  href: string;
  capabilityKey: string;
  /** 进入该能力页面所需的权限码；缺省表示 hub 页权限（cmdb:read）即可进入。 */
  requiredPermission?: string;
};

/**
 * CMDB 展示元数据。成熟度和 readiness 只来自后端 capability 控制平面。
 */
export const CMDB_CAPABILITIES = [
  {
    key: 'configuration-items',
    title: 'Configuration item management',
    description: 'Configuration item entry, query, editing, life cycle and historical audit.',
    href: '/cmdb/cis',
    capabilityKey: 'cmdb.configurationItems',
  },
  {
    key: 'ci-types',
    title: 'CI type modeling',
    description: 'Maintain types, field templates and data constraints.',
    href: '/admin/cmdb-types',
    capabilityKey: 'cmdb.ciTypes',
    requiredPermission: 'cmdb:write',
  },
  {
    key: 'relationships',
    title: 'Relationship Management',
    description: 'Maintain dependency, hosting, and impact relationships.',
    href: '/cmdb/relationships',
    capabilityKey: 'cmdb.relationships',
  },
  {
    key: 'topology',
    title: 'Topology and Impact Analysis',
    description: 'View upstream and downstream topology and support event and change impact analysis.',
    href: '/cmdb/topology',
    capabilityKey: 'cmdb.topology',
  },
  {
    key: 'cloud-discovery',
    title: 'Automatic discovery of cloud resources',
    description: 'Waiting for real connection testing, task execution, retry and audit closed-loop acceptance.',
    href: '/cmdb/registry',
    capabilityKey: 'cmdbDiscovery',
  },
  {
    key: 'cloud-reconciliation',
    title: 'Cloud resource reconciliation',
    description: 'Waiting for closed-loop acceptance of real data source discovery and conflict resolution.',
    href: '/cmdb/reconciliation',
    capabilityKey: 'cmdbReconciliation',
  },
] as const satisfies readonly CMDBCapability[];
