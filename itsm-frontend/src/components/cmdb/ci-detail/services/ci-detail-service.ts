/**
 * CI Detail 服务层
 * 封装与 CI 详情相关的 API 调用
 */

import type { CIType } from '@/types/biz/cmdb';
import type { ConfigurationItem } from '@/lib/api/cmdb-api';
import { CMDBApi } from '@/lib/api/cmdb-api';

/**
 * 获取 CI 详情和类型列表
 */
export const fetchCIDetail = async (
  ciId: string | number
): Promise<{
  ci: ConfigurationItem;
  types: CIType[];
}> => {
  const [ciData, typeData] = await Promise.all([CMDBApi.getCI(String(ciId)), CMDBApi.getTypes()]);

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
