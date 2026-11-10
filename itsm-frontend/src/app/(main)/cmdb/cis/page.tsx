'use client';

import React from 'react';

import CIList from '@/components/cmdb/CIList';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';

export default function CIListPage() {
  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="CI Workbench"
        description="View and manage all configuration items, supporting filtering by name, CI type, and status."
      />
      <CIList />
    </div>
  );
}
