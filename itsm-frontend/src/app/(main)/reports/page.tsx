'use client';

import AdvancedReporting from '@/components/business/AdvancedReporting';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';

export default function ReportsPage() {
  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="报表中心"
        description="查看当前后端已支持的 ITSM 业务报表。"
      />
      <AdvancedReporting />
    </div>
  );
}
