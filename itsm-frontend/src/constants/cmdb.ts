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
  [CIStatus.ACTIVE]: 'In Use',
  [CIStatus.INACTIVE]: 'Inactive',
  [CIStatus.MAINTENANCE]: 'Maintenance',
  [CIStatus.DECOMMISSIONED]: 'scrapped',
};

// 常见 CI 类型 (演示用，实际从 API 获取)
export enum CITypeEnum {
  SERVER = 'server',
  DATABASE = 'database',
  APPLICATION = 'application',
  NETWORK = 'network',
  STORAGE = 'storage',
}

export const CITypeLabels: Record<string, string> = {
  [CITypeEnum.SERVER]: 'Servers',
  [CITypeEnum.DATABASE]: 'Database',
  [CITypeEnum.APPLICATION]: 'Application Systems',
  [CITypeEnum.NETWORK]: 'Network Devices',
  [CITypeEnum.STORAGE]: 'Storage device',
};
