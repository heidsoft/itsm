'use client';

import AdvancedReporting from '@/components/business/AdvancedReporting';
import ServiceQualityOverview from '@/components/business/reports/ServiceQualityOverview';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';

export default function ReportsPage() {
  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="Report Center"
        description="ITSM Business Report"
      />
      <ServiceQualityOverview />
      <AdvancedReporting />
    </div>
  );
}
