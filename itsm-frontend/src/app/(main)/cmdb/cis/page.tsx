'use client';

import React from 'react';

import CIList from '@/components/cmdb/CIList';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';

export default function CIListPage() {
  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="配置项工作台"
        description="查看和管理所有配置项，支持按名称、CI 类型、状态筛选。"
      />
      <CIList />
    </div>
  );
}
