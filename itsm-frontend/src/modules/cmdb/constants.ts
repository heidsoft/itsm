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
}

export const CITypeLabels: Record<string, string> = {
    [CITypeEnum.SERVER]: '服务器',
    [CITypeEnum.DATABASE]: '数据库',
    [CITypeEnum.APPLICATION]: '应用系统',
    [CITypeEnum.NETWORK]: '网络设备',
    [CITypeEnum.STORAGE]: '存储设备',
};
