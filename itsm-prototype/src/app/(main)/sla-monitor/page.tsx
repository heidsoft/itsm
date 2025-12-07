'use client';

import React from 'react';
import { SLAMonitorDashboard } from '@/components/business/SLAMonitorDashboard';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';

export default function SLAMonitorPage() {
  return (
    <ErrorBoundary>
      <div className='min-h-screen'>
        <SLAMonitorDashboard autoRefresh={true} refreshInterval={30} />
      </div>
    </ErrorBoundary>
  );
}

