'use client';

import { useState, useEffect, useCallback } from 'react';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import type { ServiceItem } from '@/types/service-catalog';

export const useServiceCatalogData = () => {
  const [catalogs, setCatalogs] = useState<ServiceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchText, setSearchText] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('');
  const [ciTypeFilter, setCiTypeFilter] = useState<number | undefined>(undefined);
  const [cloudServiceFilter, setCloudServiceFilter] = useState<number | undefined>(undefined);
  const [stats, setStats] = useState({
    total: 0,
    cloudServices: 0,
    accountServices: 0,
    securityServices: 0,
  });

  const loadServiceCatalogs = useCallback(async () => {
    try {
      setLoading(true);
      const data = await ServiceCatalogApi.getServices({
        page: 1,
        pageSize: 100,
        search: searchText,
        status: 'published' as any,
      });
      setCatalogs(data.services || []);
    } catch (error) {
      console.error('加载服务目录失败:', error);
    } finally {
      setLoading(false);
    }
  }, [searchText]);

  useEffect(() => {
    loadServiceCatalogs();
  }, [loadServiceCatalogs]);

  useEffect(() => {
    // 与页面分类标签保持一致的过滤逻辑（使用部分匹配）
    const categoryCountMap: Record<string, string[]> = {
      cloud: ['云资源服务', 'Cloud Service'],
      account: ['账号与权限', 'Account Service'],
      security: ['安全服务', 'Security Service'],
    };

    const countByCategory = (keywords: string[]) => {
      return catalogs.filter(catalog =>
        keywords.some(keyword => String(catalog.category).includes(keyword))
      ).length;
    };

    setStats({
      total: catalogs.length,
      cloudServices: countByCategory(categoryCountMap.cloud),
      accountServices: countByCategory(categoryCountMap.account),
      securityServices: countByCategory(categoryCountMap.security),
    });
  }, [catalogs]);

  return {
    catalogs,
    loading,
    stats,
    searchText,
    setSearchText,
    setCategoryFilter,
    setPriorityFilter,
    setCiTypeFilter,
    setCloudServiceFilter,
    loadServiceCatalogs,
  };
};
