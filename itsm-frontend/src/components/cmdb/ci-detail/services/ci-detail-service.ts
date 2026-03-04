/**
 * CI Detail 服务层
 * 封装与 CI 详情相关的 API 调用
 */

import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';
import { CMDBApi } from '@/lib/api/';

/**
 * 获取 CI 详情和类型列表
 */
export const fetchCIDetail = async (ciId: string | number): Promise<{
  ci: any; // ConfigurationItem 类型在不同模块中不兼容，使用 any
  types: any[]; // CIType[]
}> => {
  const [ciData, typeData] = await Promise.all([
    CMDBApi.getCI(String(ciId)),
    CMDBApi.getTypes(),
  ]);

  return {
    ci: ciData,
    types: typeData || [],
  };
};

/**
 * 获取 CI 影响分析
 */
export const fetchCIImpactAnalysis = async (ciId: string | number) => {
  return await CMDBApi.getCIImpactAnalysis(Number(ciId));
};

/**
 * 获取 CI 变更历史
 */
export const fetchCIChangeHistory = async (ciId: string | number) => {
  return await CMDBApi.getCIChangeHistory(Number(ciId));
};
