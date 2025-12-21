/**
 * ITSM前端架构 - 领域驱动模块结构
 * 
 * 模块组织原则：
 * 1. 按业务领域划分模块
 * 2. 每个模块包含完整的业务逻辑
 * 3. 模块间通过接口通信
 * 4. 支持模块的独立开发和测试
 */

// 核心业务模块
export const CORE_MODULES = {
  // 认证授权模块
  AUTH: 'auth',
  // 用户管理模块
  USER: 'user',
  // 工单管理模块
  TICKET: 'ticket',
  // 事件管理模块
  INCIDENT: 'incident',
  // 问题管理模块
  PROBLEM: 'problem',
  // 变更管理模块
  CHANGE: 'change',
  // 配置管理模块
  CMDB: 'cmdb',
  // 知识库模块
  KNOWLEDGE: 'knowledge',
  // 服务目录模块
  SERVICE: 'service',
  // 报表分析模块
  ANALYTICS: 'analytics',
  // 工作流模块
  WORKFLOW: 'workflow',
  // 通知模块
  NOTIFICATION: 'notification',
} as const;

// 共享模块
export const SHARED_MODULES = {
  // UI组件库
  UI: 'ui',
  // 工具函数
  UTILS: 'utils',
  // 类型定义
  TYPES: 'types',
  // API客户端
  API: 'api',
  // 状态管理
  STORE: 'store',
  // 路由管理
  ROUTER: 'router',
  // 国际化
  I18N: 'i18n',
  // 主题系统
  THEME: 'theme',
  // 权限控制
  PERMISSION: 'permission',
} as const;

// 模块类型
export type CoreModule = typeof CORE_MODULES[keyof typeof CORE_MODULES];
export type SharedModule = typeof SHARED_MODULES[keyof typeof SHARED_MODULES];
export type ModuleType = CoreModule | SharedModule;

// 模块配置接口
export interface ModuleConfig {
  name: string;
  version: string;
  dependencies: ModuleType[];
  exports: string[];
  routes?: string[];
  permissions?: string[];
}

// 模块注册表
export const MODULE_REGISTRY = new Map<ModuleType, ModuleConfig>();

// 模块注册函数
export function registerModule(moduleType: ModuleType, config: ModuleConfig): void {
  MODULE_REGISTRY.set(moduleType, config);
}

// 获取模块配置
export function getModuleConfig(moduleType: ModuleType): ModuleConfig | undefined {
  return MODULE_REGISTRY.get(moduleType);
}

// 检查模块依赖
export function checkModuleDependencies(moduleType: ModuleType): boolean {
  const config = getModuleConfig(moduleType);
  if (!config) return false;

  return config.dependencies.every(dep => MODULE_REGISTRY.has(dep));
}

// 获取模块导出
export function getModuleExports(moduleType: ModuleType): string[] {
  const config = getModuleConfig(moduleType);
  return config?.exports || [];
}

export default {
  CORE_MODULES,
  SHARED_MODULES,
  registerModule,
  getModuleConfig,
  checkModuleDependencies,
  getModuleExports,
};
