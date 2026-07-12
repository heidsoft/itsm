'use client';

import React from 'react';
import { App, Card, Empty, Select, Spin } from 'antd';

import CIRelationshipManager from '@/components/cmdb/CIRelationshipManager';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { CMDBApi } from '@/lib/api/cmdb-api';

type CiOption = {
  id: number;
  name: string;
  type: string;
};

export default function RelationshipsPage() {
  const { message } = App.useApp();
  const [loading, setLoading] = React.useState(true);
  const [cis, setCis] = React.useState<CiOption[]>([]);
  const [selectedCiId, setSelectedCiId] = React.useState<number | undefined>(undefined);

  React.useEffect(() => {
    let mounted = true;

    const load = async () => {
      setLoading(true);
      try {
        const response = await CMDBApi.getCIs({ limit: 200 });
		const items = response?.items ?? [];
        const options = items.map((item) => ({
          id: item.id,
          name: item.name,
		  type: item.type || '配置项',
        }));
        if (!mounted) return;
        setCis(options);
        if (options.length > 0) {
          setSelectedCiId(options[0].id);
        }
      } catch (error) {
        message.error('加载配置项失败');
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    };

    load();
    return () => {
      mounted = false;
    };
  }, [message]);

  const selectedCi = cis.find(item => item.id === selectedCiId) || null;

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="关系管理"
        description="围绕 CI 的依赖、托管、影响和包含关系进行建模，这是 Service Graph 的骨架。"
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
