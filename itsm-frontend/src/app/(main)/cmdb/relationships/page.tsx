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
  const loading = cisQuery.isPending;

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="关系管理"
        description="按配置项查看它的出入向关系，并维护依赖、承载、包含、影响、拥有、使用等关系类型。"
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
          <Empty description="请选择一个配置项后查看它的关系列表" />
        )}
      </Card>
    </div>
  );
}
