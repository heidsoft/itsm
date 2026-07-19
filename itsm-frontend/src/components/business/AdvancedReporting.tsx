'use client';

import { useRouter } from 'next/navigation';
import { AdvancedReportingView } from './advanced-reporting/AdvancedReportingView';
import { useReportSummaries } from './advanced-reporting/hooks/useReportSummaries';

export default function AdvancedReporting() {
  const router = useRouter();
  const state = useReportSummaries();

  return (
    <AdvancedReportingView
      reports={state.reports}
      loading={state.loading}
      error={state.error}
      onReload={state.reload}
      onOpenReport={report => router.push(report.path)}
    />
  );
}
