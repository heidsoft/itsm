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
    setStats({
      total: catalogs.length,
      cloudServices: catalogs.filter(c => String(c.category) === '云资源服务').length,
      accountServices: catalogs.filter(c => String(c.category) === '账号与权限').length,
      securityServices: catalogs.filter(c => String(c.category) === '安全服务').length,
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
