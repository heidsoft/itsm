'use client';

import React from 'react';

import CIList from '@/components/cmdb/CIList';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';

export default function CIPage() {
  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="配置项工作台"
        description="面向 CSDM 的配置项清单，不是纯资产表。这里保留清单管理，但它只是 Service Graph 的一个视图。"
      />
      <CIList />
    </div>
  );
}
