/**
 * ITSM前端架构 - 组件架构系统
 * 
 * 组件分层原则：
 * 1. 原子组件 (Atoms) - 最小可复用单元
 * 2. 分子组件 (Molecules) - 原子组件组合
 * 3. 有机体组件 (Organisms) - 复杂业务组件
 * 4. 模板组件 (Templates) - 页面布局模板
 * 5. 页面组件 (Pages) - 具体业务页面
 */

import { ReactNode, ComponentType } from 'react';

// 组件类型枚举
export enum ComponentType {
  ATOM = 'atom',
  MOLECULE = 'molecule',
  ORGANISM = 'organism',
  TEMPLATE = 'template',
  PAGE = 'page',
}

// 组件元数据接口
export interface ComponentMetadata {
  name: string;
  type: ComponentType;
  module: string;
  version: string;
  description: string;
  props: ComponentProp[];
  examples: ComponentExample[];
  dependencies: string[];
  tags: string[];
}

// 组件属性定义
export interface ComponentProp {
  name: string;
  type: string;
  required: boolean;
  defaultValue?: any;
  description: string;
  examples: any[];
}

// 组件示例
export interface ComponentExample {
  title: string;
  description: string;
  code: string;
  props: Record<string, any>;
}

// 组件注册表
export const COMPONENT_REGISTRY = new Map<string, ComponentMetadata>();

// 组件注册函数
export function registerComponent(
  componentName: string,
  metadata: ComponentMetadata
): void {
  COMPONENT_REGISTRY.set(componentName, metadata);
}

// 获取组件元数据
export function getComponentMetadata(componentName: string): ComponentMetadata | undefined {
  return COMPONENT_REGISTRY.get(componentName);
}

// 按类型获取组件
export function getComponentsByType(type: ComponentType): ComponentMetadata[] {
  return Array.from(COMPONENT_REGISTRY.values()).filter(comp => comp.type === type);
}

// 按模块获取组件
export function getComponentsByModule(module: string): ComponentMetadata[] {
  return Array.from(COMPONENT_REGISTRY.values()).filter(comp => comp.module === module);
}

// 组件工厂接口
export interface ComponentFactory<T = any> {
  create: (props: T) => ReactNode;
  validate: (props: T) => boolean;
  getDefaultProps: () => Partial<T>;
}

// 组件工厂注册表
export const COMPONENT_FACTORY_REGISTRY = new Map<string, ComponentFactory>();

// 注册组件工厂
export function registerComponentFactory<T>(
  componentName: string,
  factory: ComponentFactory<T>
): void {
  COMPONENT_FACTORY_REGISTRY.set(componentName, factory);
}

// 创建组件实例
export function createComponent<T>(
  componentName: string,
  props: T
): ReactNode | null {
  const factory = COMPONENT_FACTORY_REGISTRY.get(componentName) as ComponentFactory<T>;
  if (!factory) return null;

  if (!factory.validate(props)) {
    console.error(`Invalid props for component ${componentName}:`, props);
    return null;
  }

  return factory.create(props);
}

// 组件依赖解析
export function resolveComponentDependencies(componentName: string): string[] {
  const metadata = getComponentMetadata(componentName);
  if (!metadata) return [];

  const dependencies = new Set<string>();
  
  function resolveDeps(compName: string) {
    const meta = getComponentMetadata(compName);
    if (!meta) return;

    meta.dependencies.forEach(dep => {
      if (!dependencies.has(dep)) {
        dependencies.add(dep);
        resolveDeps(dep);
      }
    });
  }

  resolveDeps(componentName);
  return Array.from(dependencies);
}

// 组件使用统计
export interface ComponentUsageStats {
  componentName: string;
  usageCount: number;
  lastUsed: Date;
  errorCount: number;
  performanceMetrics: {
    averageRenderTime: number;
    memoryUsage: number;
  };
}

export const COMPONENT_USAGE_STATS = new Map<string, ComponentUsageStats>();

// 记录组件使用
export function recordComponentUsage(componentName: string, renderTime: number): void {
  const stats = COMPONENT_USAGE_STATS.get(componentName) || {
    componentName,
    usageCount: 0,
    lastUsed: new Date(),
    errorCount: 0,
    performanceMetrics: {
      averageRenderTime: 0,
      memoryUsage: 0,
    },
  };

  stats.usageCount++;
  stats.lastUsed = new Date();
  stats.performanceMetrics.averageRenderTime = 
    (stats.performanceMetrics.averageRenderTime * (stats.usageCount - 1) + renderTime) / stats.usageCount;

  COMPONENT_USAGE_STATS.set(componentName, stats);
}

// 获取组件使用统计
export function getComponentUsageStats(componentName: string): ComponentUsageStats | undefined {
  return COMPONENT_USAGE_STATS.get(componentName);
}

// 组件热重载支持
export interface HotReloadConfig {
  enabled: boolean;
  modules: string[];
  exclude: string[];
}

export const HOT_RELOAD_CONFIG: HotReloadConfig = {
  enabled: process.env.NODE_ENV === 'development',
  modules: [],
  exclude: ['node_modules', '.next'],
};

// 组件热重载
export function hotReloadComponent(componentName: string): void {
  if (!HOT_RELOAD_CONFIG.enabled) return;

  const metadata = getComponentMetadata(componentName);
  if (!metadata) return;

  // 清除组件缓存
  if (typeof window !== 'undefined' && (window as any).__REACT_HOT_LOADER__) {
    (window as any).__REACT_HOT_LOADER__.invalidate();
  }

  console.log(`Hot reloading component: ${componentName}`);
}

export default {
  ComponentType,
  registerComponent,
  getComponentMetadata,
  getComponentsByType,
  getComponentsByModule,
  registerComponentFactory,
  createComponent,
  resolveComponentDependencies,
  recordComponentUsage,
  getComponentUsageStats,
  hotReloadComponent,
};
