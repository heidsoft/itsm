'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { ServiceStatus, type ServiceItem } from '@/types/service-catalog';

/** 门户一次拉满的目录上限：超过它必须显式提示，禁止把截断结果当成全量。 */
const PORTAL_FETCH_LIMIT = 500;
/** 输入停顿后再打服务端，避免每次按键都发一个请求。 */
const SEARCH_DEBOUNCE_MS = 300;

/** 分类卡片与页面分类标签共用的关键词归类口径。 */
const CATEGORY_KEYWORDS: Record<'cloud' | 'account' | 'security', string[]> = {
  cloud: ['云计算', '云资源'],
  account: ['支持', '账号'],
  security: ['安全'],
};

export const useServiceCatalogData = () => {
  const [catalogs, setCatalogs] = useState<ServiceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [truncated, setTruncated] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [ciTypeFilter, setCiTypeFilter] = useState<number | undefined>(undefined);
  const [cloudServiceFilter, setCloudServiceFilter] = useState<number | undefined>(undefined);
  const [stats, setStats] = useState({
    total: 0,
    cloudServices: 0,
    accountServices: 0,
    securityServices: 0,
  });
  const requestSeqRef = useRef(0);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchText.trim()), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [searchText]);

  // 关键词由服务端过滤（q），状态固定 published：门户不能展示已停用目录。
  // 分类/关联筛选作用在「已拉满的完整集合」上，所以计数与列表口径一致。
  const loadServiceCatalogs = useCallback(async () => {
    const seq = ++requestSeqRef.current;
    try {
      setLoading(true);
      setError(null);
      const result = await ServiceCatalogApi.getAllServices(
        {
          status: ServiceStatus.PUBLISHED,
          ...(debouncedSearch ? { search: debouncedSearch } : {}),
        },
        PORTAL_FETCH_LIMIT,
      );
      if (seq !== requestSeqRef.current) return;
      setCatalogs(result.services);
      setTotal(result.total);
      setTruncated(!result.complete);
    } catch (err) {
      if (seq !== requestSeqRef.current) return;
      setCatalogs([]);
      setTotal(0);
      setTruncated(false);
      setError(err instanceof Error ? err.message : '服务目录加载失败，请稍后重试');
    } finally {
      if (seq === requestSeqRef.current) setLoading(false);
    }
  }, [debouncedSearch]);

  useEffect(() => {
    loadServiceCatalogs();
  }, [loadServiceCatalogs]);

  useEffect(() => {
    const countByCategory = (keywords: string[]) =>
      catalogs.filter(catalog =>
        keywords.some(keyword => String(catalog.category).includes(keyword)),
      ).length;

    setStats({
      // total 用服务端计数：被截断时 catalogs.length 只是已加载的部分。
      total,
      cloudServices: countByCategory(CATEGORY_KEYWORDS.cloud),
      accountServices: countByCategory(CATEGORY_KEYWORDS.account),
      securityServices: countByCategory(CATEGORY_KEYWORDS.security),
    });
  }, [catalogs, total]);

  return {
    catalogs,
    loading,
    error,
    total,
    truncated,
    limit: PORTAL_FETCH_LIMIT,
    stats,
    searchText,
    setSearchText,
    categoryFilter,
    setCategoryFilter,
    ciTypeFilter,
    setCiTypeFilter,
    cloudServiceFilter,
    setCloudServiceFilter,
    loadServiceCatalogs,
  };
};
