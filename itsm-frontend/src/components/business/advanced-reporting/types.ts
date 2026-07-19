export interface ReportSummary {
  id: string;
  name: string;
  path: string;
}

export interface AdvancedReportingViewProps {
  reports: ReportSummary[];
  loading: boolean;
  error: string | null;
  onReload: () => void;
  onOpenReport?: (report: ReportSummary) => void;
}
