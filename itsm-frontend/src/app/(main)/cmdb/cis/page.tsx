'use client';

import React from 'react';

import CIList from '@/components/cmdb/CIList';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';

export default function CIListPage() {
  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="配置项工作台"
        description="查看和管理所有配置项，支持按类型、状态、关键程度等维度筛选。"
      />
      <CIList />
    </div>
  );
}
