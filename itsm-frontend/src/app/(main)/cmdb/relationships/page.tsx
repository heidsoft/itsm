'use client';

import React from 'react';
import { App, Card, Empty, Select, Spin } from 'antd';

import CIRelationshipManager from '@/components/cmdb/CIRelationshipManager';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { useCIsQuery } from '@/lib/hooks/useCMDB';

type CiOption = {
  id: number;
  name: string;
  type: string;
};

export default function RelationshipsPage() {
  const { message } = App.useApp();
  // 根 CI 候选集。页长上限 100，超过会被后端回落成默认 20。
  const cisQuery = useCIsQuery({ pageSize: 100 });
  const cis: CiOption[] = React.useMemo(() => {
    const items = (cisQuery.data?.items ?? []) as Array<{ id: number; name: string; type?: string }>;
    return items.map(item => ({ id: item.id, name: item.name, type: item.type || 'Configuration item' }));
  }, [cisQuery.data]);

  const [selectedCiId, setSelectedCiId] = React.useState<number | undefined>(undefined);

  // 默认选中第一条
  React.useEffect(() => {
    if (selectedCiId === undefined && cis.length > 0) {
      setSelectedCiId(cis[0].id);
    }
  }, [cis, selectedCiId]);

  React.useEffect(() => {
    if (cisQuery.isError) message.error('Failed to load configuration items');
  }, [cisQuery.isError, message]);

  const selectedCi = cis.find(item => item.id === selectedCiId) || null;
  const loading = cisQuery.isPending;

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="Relationship Management"
        description="View and manage dependencies, hosting, influences, and inclusion relationships between configuration items."
      />

      <Card
        title="Select root configuration item"
        extra={
          <Select
            showSearch
            placeholder="Select a configuration item"
            value={selectedCiId}
            style={{ width: 320 }}
            loading={loading}
            optionFilterProp="label"
            onChange={value => setSelectedCiId(value)}
            options={cis.map(item => ({
              value: item.id,
              label: `${item.name} (${item.type})`,
            }))}
          />
        }
      >
        {loading ? (
          <div className="flex min-h-[240px] items-center justify-center">
            <Spin size="large" />
          </div>
        ) : selectedCi ? (
          <CIRelationshipManager ciId={selectedCi.id} ciName={selectedCi.name} />
        ) : (
          <Empty description="Please select a configuration item to view the relationship diagram" />
        )}
      </Card>
    </div>
  );
}
