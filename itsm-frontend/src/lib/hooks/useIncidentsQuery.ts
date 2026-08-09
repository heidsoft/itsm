'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { App } from 'antd';

import { IncidentAPI } from '@/lib/api/incident-api';
import type { Incident, ListIncidentsRequest } from '@/lib/api/incident-api';
import { UserApi } from '@/lib/api/user-api';

export interface IncidentFilters {
  readonly search: string;
  readonly status?: string;
  readonly priority?: string;
  readonly source?: string;
}

export interface UseIncidentsQueryOptions {
  /** Localized toast shown when the fetch fails. Defaults to a zh-CN string. */
  readonly errorMessage?: string;
}

export interface UseIncidentsQueryResult {
  readonly incidents: readonly Incident[];
  readonly total: number;
  readonly loading: boolean;
  readonly loadError: boolean;
  readonly page: number;
  readonly pageSize: number;
  readonly setPage: (page: number, pageSize?: number) => void;
  readonly refresh: () => Promise<void>;
}

const DEFAULT_PAGE_SIZE = 10;
const DEFAULT_ERROR_MESSAGE = '加载事件列表失败，请稍后重试';

/**
 * Manages the incident list query lifecycle: pagination, filter inputs,
 * loading/error state, and reporter-name enrichment.
 *
 * The hook owns no UI state. It re-fetches whenever any dependency in the
 * supplied filters changes. Callers can drive a retry by calling `refresh()`
 * (also called by the hook itself after a successful retry from an error).
 *
 * `errorMessage` is read through a ref so passing a fresh localized string
 * (e.g. from `t()`) never re-creates `fetchIncidents` and re-triggers a fetch.
 */
export function useIncidentsQuery(
  filters: IncidentFilters,
  options?: UseIncidentsQueryOptions
): UseIncidentsQueryResult {
  const { message } = App.useApp();
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [page, setPageInternal] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);

  const errorMessageRef = useRef(options?.errorMessage ?? DEFAULT_ERROR_MESSAGE);
  errorMessageRef.current = options?.errorMessage ?? DEFAULT_ERROR_MESSAGE;

  const setPage = useCallback((next: number, nextSize?: number) => {
    setPageInternal(next);
    if (nextSize !== undefined) {
      setPageSize(nextSize);
    }
  }, []);

  // Guards against out-of-order responses: when filters change rapidly, a
  // slower earlier request must not overwrite a faster later one.
  const requestIdRef = useRef(0);

  const fetchIncidents = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    setLoadError(false);
    try {
      const response = await IncidentAPI.listIncidents({
        page,
        pageSize,
        keyword: filters.search || undefined,
        status: filters.status,
        priority: filters.priority,
        source: filters.source,
      });
      if (requestIdRef.current !== requestId) return;
      const items = response.incidents || response.data || [];

      // H3: only fetch the user directory when at least one row has a
      // reporterId to enrich. Skips the round-trip entirely for lists of
      // unassigned incidents (high-volume tenants, repeated refresh).
      const needsUserEnrichment = items.some(
        i => i.reporterId !== undefined && i.reporterId !== null
      );
      const userMap = new Map<number, string>();
      if (needsUserEnrichment) {
        try {
          const usersResponse = await UserApi.getUsers({ pageSize: 100 });
          if (requestIdRef.current !== requestId) return;
          usersResponse.users.forEach(user => userMap.set(user.id, user.name));
        } catch {
          // intentionally swallowed — UI degrades to id-only reporter
        }
      }

      if (requestIdRef.current !== requestId) return;

      const enriched = items.map(inc => {
        const reporterId = inc.reporterId;
        const reporterName = reporterId ? userMap.get(reporterId) : undefined;
        return {
          ...inc,
          ...(reporterId !== undefined && reporterName
            ? { reporter: { id: reporterId, name: reporterName } }
            : {}),
        };
      });

      setIncidents(enriched);
      setTotal(response.total ?? enriched.length);
    } catch (error) {
      if (requestIdRef.current !== requestId) return;
      console.error('Failed to fetch incidents:', error);
      message.error(errorMessageRef.current);
      setLoadError(true);
    } finally {
      if (requestIdRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [page, pageSize, filters.search, filters.status, filters.priority, filters.source, message]);

  useEffect(() => {
    void fetchIncidents();
  }, [fetchIncidents]);

  return {
    incidents,
    total,
    loading,
    loadError,
    page,
    pageSize,
    setPage,
    refresh: fetchIncidents,
  };
}
