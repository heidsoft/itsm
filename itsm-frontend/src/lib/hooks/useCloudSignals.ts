/**
 * 云信号 React Query Hooks：云监控告警、安全漏洞与同步状态。
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import CloudSignalApi, {
  type AlertListQuery,
  type FindingListQuery,
} from '@/lib/api/cloudsignal-api';

export const CLOUD_SIGNAL_KEYS = {
  all: ['cloud-signals'] as const,
  alerts: (query: AlertListQuery) => [...CLOUD_SIGNAL_KEYS.all, 'alerts', query] as const,
  findings: (query: FindingListQuery) => [...CLOUD_SIGNAL_KEYS.all, 'findings', query] as const,
  findingSummary: () => [...CLOUD_SIGNAL_KEYS.all, 'findings', 'summary'] as const,
  syncStatus: () => [...CLOUD_SIGNAL_KEYS.all, 'sync-status'] as const,
};

export function useCloudAlertsQuery(query: AlertListQuery) {
  return useQuery({
    queryKey: CLOUD_SIGNAL_KEYS.alerts(query),
    queryFn: () => CloudSignalApi.listAlerts(query),
    staleTime: 30000,
  });
}

export function useSecurityFindingsQuery(query: FindingListQuery) {
  return useQuery({
    queryKey: CLOUD_SIGNAL_KEYS.findings(query),
    queryFn: () => CloudSignalApi.listFindings(query),
    staleTime: 30000,
  });
}

export function useFindingSummaryQuery() {
  return useQuery({
    queryKey: CLOUD_SIGNAL_KEYS.findingSummary(),
    queryFn: () => CloudSignalApi.getFindingSummary(),
    staleTime: 30000,
  });
}

export function useCloudSignalSyncStatusQuery() {
  return useQuery({
    queryKey: CLOUD_SIGNAL_KEYS.syncStatus(),
    queryFn: () => CloudSignalApi.getSyncStatus(),
    staleTime: 30000,
  });
}

// 动作成功后统一失效云信号缓存：告警与漏洞的状态变化也会影响统计和同步面板。
function useSignalMutation<TVariables, TResult>(
  mutationFn: (variables: TVariables) => Promise<TResult>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CLOUD_SIGNAL_KEYS.all });
    },
  });
}

export function useAcknowledgeAlertMutation() {
  return useSignalMutation((id: number) => CloudSignalApi.acknowledgeAlert(id));
}

export function useResolveAlertMutation() {
  return useSignalMutation((id: number) => CloudSignalApi.resolveAlert(id));
}

export function useIgnoreFindingMutation() {
  return useSignalMutation((id: number) => CloudSignalApi.ignoreFinding(id));
}

export function useReopenFindingMutation() {
  return useSignalMutation((id: number) => CloudSignalApi.reopenFinding(id));
}
