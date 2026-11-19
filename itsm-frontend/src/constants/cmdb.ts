/**
 * CMDB 资产管理相关常量定义
 */

// 配置项状态
export enum CIStatus {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  MAINTENANCE = 'maintenance',
  DECOMMISSIONED = 'decommissioned',
}

export const CIStatusLabels: Record<CIStatus, string> = {
  [CIStatus.ACTIVE]: '使用中',
  [CIStatus.INACTIVE]: '未激活',
  [CIStatus.MAINTENANCE]: '维护中',
  [CIStatus.DECOMMISSIONED]: '已报废',
};

// 常见 CI 类型 (演示用，实际从 API 获取)
export enum CITypeEnum {
  SERVER = 'server',
  DATABASE = 'database',
  APPLICATION = 'application',
  NETWORK = 'network',
  STORAGE = 'storage',
  BUSINESS_SERVICE = 'business_service',
}

export const CITypeLabels: Record<string, string> = {
  [CITypeEnum.SERVER]: '服务器',
  [CITypeEnum.DATABASE]: '数据库',
  [CITypeEnum.APPLICATION]: '应用系统',
  [CITypeEnum.NETWORK]: '网络设备',
  [CITypeEnum.STORAGE]: '存储设备',
  [CITypeEnum.BUSINESS_SERVICE]: '业务服务',
  // 与 pkg/seeder 预置类型对齐（ciTypeDefinitions）：
  // middleware / cloud_vm / kubernetes 此前缺词条，展示层会把代码直接透出。
  middleware: '中间件',
  cloud_vm: '云虚拟机',
  kubernetes: 'Kubernetes资源',
};

/**
 * CI 类型代码 → 中文展示文案（单一入口）。
 *
 * ci_types.name 是租户内唯一代码（server/database/...），中文名在 description。
 * 展示层没有 CIType 列表可查时（如搜索结果、拓扑节点、下拉选项），
 * 用这份静态词表兜底；未知代码原样返回，不做枚举断言以免吞掉自建类型。
 *
 * 有 CIType 列表时优先用 description（租户可自定义），见 CIList.getCITypeLabel。
 */
export const ciTypeCodeLabel = (code?: string | null): string =>
  code ? CITypeLabels[code] || code : '';

/**
 * 有 CIType 列表时：优先中文 description，回退代码、再回退 ID。
 * 与 CIList.getCITypeLabel 同口径，抽到此处避免多处各自实现。
 */
export const resolveCITypeLabel = (
  types: { id: number; name: string; description?: string }[] | undefined,
  ciTypeId?: number,
  fallbackCode?: string
): string => {
  const type = types?.find(item => item.id === ciTypeId);
  return type?.description?.trim() || ciTypeCodeLabel(type?.name) || ciTypeCodeLabel(fallbackCode) || '';
};
