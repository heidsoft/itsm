/**
 * 筛选条件持久化工具
 *
 * 功能：
 * 1. 将筛选条件保存到 localStorage
 * 2. 从 localStorage 恢复筛选条件
 * 3. 清除筛选条件
 * 4. 支持页面级别的独立存储
 */

import { logger } from '@/lib/env';

// 筛选条件存储接口
export interface FilterState {
  [key: string]: unknown;
}

// 存储键前缀
const STORAGE_PREFIX = 'itsm_filter_';

/**
 * 生成存储键
 * @param pageKey 页面唯一标识
 */
function getStorageKey(pageKey: string): string {
  return `${STORAGE_PREFIX}${pageKey}`;
}

/**
 * 保存筛选条件到 localStorage
 * @param pageKey 页面唯一标识（如：'tickets', 'incidents', 'problems'）
 * @param filters 筛选条件对象
 */
export function saveFilters(pageKey: string, filters: FilterState): void {
  try {
    const key = getStorageKey(pageKey);
    const serialized = JSON.stringify(filters);
    localStorage.setItem(key, serialized);
    logger.debug(`[FilterPersistence] Saved filters for ${pageKey}:`, filters);
  } catch (error) {
    logger.error(`[FilterPersistence] Failed to save filters for ${pageKey}:`, error);
  }
}

/**
 * 从 localStorage 恢复筛选条件
 * @param pageKey 页面唯一标识
 * @param defaults 默认筛选条件（当 localStorage 中没有数据时使用）
 * @returns 恢复的筛选条件
 */
export function restoreFilters(pageKey: string, defaults: FilterState = {}): FilterState {
  try {
    const key = getStorageKey(pageKey);
    const serialized = localStorage.getItem(key);

    if (!serialized) {
      logger.debug(`[FilterPersistence] No saved filters for ${pageKey}, using defaults`);
      return defaults;
    }

    const filters = JSON.parse(serialized) as FilterState;
    logger.debug(`[FilterPersistence] Restored filters for ${pageKey}:`, filters);
    return { ...defaults, ...filters };
  } catch (error) {
    logger.error(`[FilterPersistence] Failed to restore filters for ${pageKey}:`, error);
    return defaults;
  }
}

/**
 * 清除筛选条件
 * @param pageKey 页面唯一标识
 */
export function clearFilters(pageKey: string): void {
  try {
    const key = getStorageKey(pageKey);
    localStorage.removeItem(key);
    logger.debug(`[FilterPersistence] Cleared filters for ${pageKey}`);
  } catch (error) {
    logger.error(`[FilterPersistence] Failed to clear filters for ${pageKey}:`, error);
  }
}

/**
 * 清除所有筛选条件
 */
export function clearAllFilters(): void {
  try {
    const keysToRemove: string[] = [];

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith(STORAGE_PREFIX)) {
        keysToRemove.push(key);
      }
    }

    keysToRemove.forEach(key => localStorage.removeItem(key));
    logger.debug(`[FilterPersistence] Cleared all filters (${keysToRemove.length} items)`);
  } catch (error) {
    logger.error(`[FilterPersistence] Failed to clear all filters:`, error);
  }
}

/**
 * 获取所有已保存筛选条件的页面列表
 * @returns 页面标识列表
 */
export function getSavedFilterPages(): string[] {
  try {
    const pages: string[] = [];

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith(STORAGE_PREFIX)) {
        const pageKey = key.substring(STORAGE_PREFIX.length);
        pages.push(pageKey);
      }
    }

    return pages;
  } catch (error) {
    logger.error(`[FilterPersistence] Failed to get saved filter pages:`, error);
    return [];
  }
}

/**
 * 筛选条件持久化 Hook 辅助函数
 *
 * 使用示例：
 * ```tsx
 * const [filters, setFilters] = useState<FilterState>(() =>
 *   restoreFilters('tickets', { status: 'open', priority: 'high' })
 * );
 *
 * const updateFilters = (newFilters: FilterState) => {
 *   const updated = { ...filters, ...newFilters };
 *   setFilters(updated);
 *   saveFilters('tickets', updated);
 * };
 *
 * const resetFilters = () => {
 *   clearFilters('tickets');
 *   setFilters({ status: 'open', priority: 'high' });
 * };
 * ```
 */

// 常用的默认筛选条件模板
export const DEFAULT_FILTERS = {
  // 工单列表
  tickets: {},
  // 事件列表
  incidents: {
    status: 'all',
    priority: 'all',
    category: 'all',
    dateRange: null,
  },
  // 问题列表
  problems: {
    status: 'all',
    priority: 'all',
    dateRange: null,
  },
  // 变更列表
  changes: {
    status: 'all',
    type: 'all',
    risk: 'all',
    dateRange: null,
  },
  // 知识库列表
  knowledge: {
    category: 'all',
    status: 'published',
    dateRange: null,
  },
  // 资产列表
  assets: {
    type: 'all',
    status: 'all',
    department: 'all',
  },
  // SLA 列表
  sla: {
    status: 'all',
    serviceType: 'all',
  },
} as const;

/**
 * 获取页面的默认筛选条件
 * @param pageKey 页面唯一标识
 * @returns 默认筛选条件
 */
export function getDefaultFilters(pageKey: string): FilterState {
  const key = pageKey.toLowerCase() as keyof typeof DEFAULT_FILTERS;
  return (DEFAULT_FILTERS[key] as FilterState) || {};
}
