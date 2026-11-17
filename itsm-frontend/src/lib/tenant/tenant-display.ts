/**
 * 租户类型展示映射 —— 租户管理页与回归锁的共用源。
 *
 * 缺陷 O8 的同源问题：映射表键写成 camelCase（saasCustomer / mspProvider），
 * 而后端 `/api/v1/tenants` 返回的 `type` 是 snake_case（`saas_customer` /
 * `msp_provider`）。`TENANT_TYPES[record.type]` 于是恒不命中，标签一直回退
 * 显示原始枚举；同时前端另立了一份本地 `Tenant` 类型并用 `as Tenant[]` 强转，
 * 把这处不一致（以及后端根本没有的 userCount 字段）完全掩盖掉了。
 *
 * 因此这里只保留**一张**定义表：值域键、配色、i18n 词条名都写在同一处。
 * 拆成「键表 + label 表」两张表正是漂移的来源，不再拆。
 */

export type TranslatorFn = (key: string, params?: Record<string, string | number>) => string;

export interface TenantTypeConfig {
  label: string;
  color: string;
}

interface TenantTypeDefinition {
  /** Antd Tag 配色 */
  color: string;
  /** i18n 词条名：tenants.typeLabels.<labelKey> */
  labelKey: string;
}

/**
 * 后端 `tenants.type` 的真实值域（snake_case）。
 *
 * 键必须是后端的原样值；labelKey 才允许是 camelCase，因为 i18n 词条本身
 * 用的是 camelCase 风格命名。这两件事不要混。
 */
const TENANT_TYPE_DEFINITIONS: Readonly<Record<string, TenantTypeDefinition>> = {
  standard: { color: 'blue', labelKey: 'standard' },
  internal: { color: 'cyan', labelKey: 'internal' },
  saas_customer: { color: 'green', labelKey: 'saasCustomer' },
  msp_provider: { color: 'gold', labelKey: 'mspProvider' },
  msp_customer: { color: 'purple', labelKey: 'mspCustomer' },
  msp: { color: 'orange', labelKey: 'msp' },
  customer: { color: 'default', labelKey: 'customer' },
};

/** 后端 `tenants.type` 的真实值域（供测试与类型收敛断言使用）。 */
export const TENANT_TYPE_VALUES = Object.keys(TENANT_TYPE_DEFINITIONS);

/** i18n 词条名清单（供测试断言「每个键都有对应词条」）。 */
export const TENANT_TYPE_LABEL_KEYS = Object.values(TENANT_TYPE_DEFINITIONS).map(item => item.labelKey);

export type TenantTypeConfigMap = Record<string, TenantTypeConfig>;

/**
 * 构造租户类型展示配置（label 走当前语言的 i18n）。
 */
export function getTenantTypeConfig(t: TranslatorFn): TenantTypeConfigMap {
  const config: TenantTypeConfigMap = {};
  for (const [type, definition] of Object.entries(TENANT_TYPE_DEFINITIONS)) {
    config[type] = {
      color: definition.color,
      label: t(`tenants.typeLabels.${definition.labelKey}`),
    };
  }
  return config;
}

/**
 * 解析租户类型的展示配置。
 *
 * 用显式查找而不是 `config[type]` 宽松索引：`api-config` 的 `Tenant.type`
 * 还含三个历史兼容值（trial / professional / enterprise，来自旧会话视图），
 * 这些值后端不会返回，但类型上合法。命中不到时按「原样显示枚举 + 默认灰色」
 * 处理，不静默归类成某一类——静默归类会让脏数据看起来像正常数据。
 */
export function resolveTenantType(config: TenantTypeConfigMap, type: string): TenantTypeConfig {
  if (Object.prototype.hasOwnProperty.call(config, type)) {
    return config[type];
  }
  return { color: 'default', label: type };
}
