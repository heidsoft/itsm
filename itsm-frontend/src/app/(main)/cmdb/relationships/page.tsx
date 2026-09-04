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
  // React Query：CI 全量（relationships 页用作根 CI 选择）
  const cisQuery = useCIsQuery({ size: 200 });
  const cis: CiOption[] = React.useMemo(() => {
    const items = (cisQuery.data?.items ?? []) as Array<{ id: number; name: string; type?: string }>;
    return items.map(item => ({ id: item.id, name: item.name, type: item.type || '配置项' }));
  }, [cisQuery.data]);

  const [selectedCiId, setSelectedCiId] = React.useState<number | undefined>(undefined);

  // 默认选中第一条
  React.useEffect(() => {
    if (selectedCiId === undefined && cis.length > 0) {
      setSelectedCiId(cis[0].id);
    }
  }, [cis, selectedCiId]);

  React.useEffect(() => {
    if (cisQuery.isError) message.error('加载配置项失败');
  }, [cisQuery.isError, message]);

  const selectedCi = cis.find(item => item.id === selectedCiId) || null;
  const loading = cisQuery.isLoading;

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="关系管理"
        description="查看和管理配置项之间的依赖、托管、影响和包含关系。"
      />

      <Card
        title="选择根配置项"
        extra={
          <Select
            showSearch
            placeholder="选择一个配置项"
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
          <Empty description="请选择一个配置项后查看关系图谱" />
        )}
      </Card>
    </div>
  );
}
