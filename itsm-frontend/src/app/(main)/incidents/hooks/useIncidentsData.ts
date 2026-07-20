'use client';

import { useState, useEffect } from 'react';
import type { Incident } from '@/lib/api/incident-api';
import { IncidentAPI } from '@/lib/api/incident-api';
import { message } from 'antd';
import { useI18n } from '@/lib/i18n';

export const useIncidentsData = () => {
  const { t } = useI18n();
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState({
    status: '',
    priority: '',
    source: '',
    keyword: '',
  });
  const [metrics, setMetrics] = useState({
    totalIncidents: 0,
    criticalIncidents: 0,
    majorIncidents: 0,
    avgResolutionTime: 0,
  });

  useEffect(() => {
    loadIncidents();
    loadMetrics();
  }, [currentPage, pageSize, filters]);

  const loadIncidents = async () => {
    setLoading(true);
    try {
      const response = await IncidentAPI.listIncidents({
        page: currentPage,
        pageSize: pageSize,
        status: filters.status,
        priority: filters.priority,
        source: filters.source,
        keyword: filters.keyword,
      });
      setIncidents(response.incidents);
      setTotal((response as any).total ?? response.incidents?.length ?? 0);
    } catch (error) {
      console.error('Failed to load incidents:', error);
      message.error(t('incidents.loadDataError'));
    } finally {
      setLoading(false);
    }
  };

  const loadMetrics = async () => {
    try {
      const response = await IncidentAPI.getIncidentMetrics();
      setMetrics({
        totalIncidents: response.totalIncidents ?? 0,
        criticalIncidents: response.criticalIncidents ?? 0,
        majorIncidents: response.majorIncidents ?? 0,
        avgResolutionTime: response.avgResolutionTime ?? 0,
      });
    } catch (error) {
      console.error('Failed to load metrics:', error);
      setMetrics({
        totalIncidents: 0,
        criticalIncidents: 0,
        majorIncidents: 0,
        avgResolutionTime: 0,
      });
    }
  };

  const handleSearch = (value: string) => {
    setFilters({ ...filters, keyword: value });
    setCurrentPage(1);
  };

  const handleFilterChange = (key: string, value: string) => {
    setFilters({ ...filters, [key]: value });
    setCurrentPage(1);
  };

  return {
    incidents,
    loading,
    total,
    currentPage,
    pageSize,
    filters,
    metrics,
    setCurrentPage,
    setPageSize,
    handleSearch,
    handleFilterChange,
    loadIncidents,
  };
};
