'use client';

import { useCallback, useEffect, useState } from 'react';
import { httpClient } from '@/lib/api/http-client';
import type { ReportSummary } from '../types';

interface ReportSummaryResponse {
  reports: ReportSummary[];
}

export function useReportSummaries() {
  const [reports, setReports] = useState<ReportSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await httpClient.get<ReportSummaryResponse>('/api/v1/reports');
      setReports(response.reports ?? []);
    } catch (requestError) {
      setReports([]);
      setError(requestError instanceof Error ? requestError.message : '报表目录加载失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { reports, loading, error, reload };
}
