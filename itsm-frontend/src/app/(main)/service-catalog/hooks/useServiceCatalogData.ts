'use client';

import { useState, useEffect } from 'react';
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

  useEffect(() => {
    loadServiceCatalogs();
  }, []);

  useEffect(() => {
    setStats({
      total: catalogs.length,
      cloudServices: catalogs.filter(c => String(c.category) === '云资源服务').length,
      accountServices: catalogs.filter(c => String(c.category) === '账号与权限').length,
      securityServices: catalogs.filter(c => String(c.category) === '安全服务').length,
    });
  }, [catalogs]);

  const loadServiceCatalogs = async () => {
    try {
      setLoading(true);
      const data = await ServiceCatalogApi.getServices();
      setCatalogs(data.services || []);
    } catch (error) {
      console.error('加载服务目录失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const filteredCatalogs = catalogs.filter(catalog => {
    const desc = (catalog.shortDescription || (catalog as any).description || '').toLowerCase();
    const matchesSearch = searchText
      ? catalog.name.toLowerCase().includes(searchText.toLowerCase()) ||
        desc.includes(searchText.toLowerCase())
      : true;
    const categoryLabel = String(catalog.category);
    const matchesCategory = categoryFilter ? categoryLabel === categoryFilter : true;
    const matchesPriority = priorityFilter ? (catalog as any).priority === priorityFilter : true;
    const matchesCIType = ciTypeFilter ? catalog.ciTypeId === ciTypeFilter : true;
    const matchesCloudService = cloudServiceFilter ? catalog.cloudServiceId === cloudServiceFilter : true;
    return matchesSearch && matchesCategory && matchesPriority && matchesCIType && matchesCloudService;
  });

  return {
    catalogs: filteredCatalogs,
    loading,
    stats,
    setSearchText,
    setCategoryFilter,
    setPriorityFilter,
    setCiTypeFilter,
    setCloudServiceFilter,
    loadServiceCatalogs,
  };
};
