'use client';

import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Breadcrumb,
  Button,
  Card,
  Descriptions,
  Divider,
  Skeleton,
  Space,
  Tag,
} from 'antd';
import { ArrowLeft, RefreshCw } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';

import { SLAApi } from '@/lib/api/';
import type { SLADefinition } from '@/lib/api/sla-api';
import { formatDateTime } from '@/lib/formatters';
import {
  formatBusinessHours,
  formatDuration,
  formatEscalationLevel,
  getPriorityColor,
  getPriorityLabel,
} from '@/lib/utils/sla-format';

const isEmptyRecord = (value?: Record<string, unknown>): boolean =>
  !value || Object.keys(value).length === 0;

// formatters.formatDateTime 对缺失或非法时间返回空串，详情页统一显示为 -
const displayDateTime = (value?: string) => formatDateTime(value) || '-';

const SLADetail: React.FC = () => {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<SLADefinition | null>(null);
  const [error, setError] = useState<string | null>(null);

  const id = params?.id;

  const loadDetail = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      setData(await SLAApi.getSLADefinition(Number(id)));
    } catch (err) {
      setData(null);
      setError(err instanceof Error ? err.message : '加载 SLA 详情失败');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  if (loading) {
    return (
      <Card className="rounded-lg shadow-sm border border-gray-200">
        <Skeleton active />
      </Card>
    );
  }

  if (error || !data) {
    return (
      <Card className="rounded-lg shadow-sm border border-gray-200">
        <Alert
          type={error ? 'error' : 'warning'}
          showIcon
          title={error ? 'SLA 详情加载失败' : '未找到该 SLA 定义'}
          description={error ?? `SLA 定义 #${id ?? ''} 不存在或不属于当前租户。`}
          action={
            <Space>
              {error && (
                <Button size="small" icon={<RefreshCw className="w-4 h-4" />} onClick={loadDetail}>
                  重新加载
                </Button>
              )}
              <Button type="primary" onClick={() => router.push('/sla')}>
                返回列表
              </Button>
            </Space>
          }
        />
      </Card>
    );
  }

  return (
    <div className="pb-6">
      <Breadcrumb
        className="mb-4"
        items={[
          { title: '首页' },
          { title: '服务级别管理' },
          { title: 'SLA 定义', onClick: () => router.push('/sla') },
          { title: '详情' },
        ]}
      />

      <Card
        className="rounded-lg shadow-sm border border-gray-200"
        title={<span className="text-lg font-bold">{data.name}</span>}
        extra={
          <Button icon={<ArrowLeft />} onClick={() => router.push('/sla')}>
            返回列表
          </Button>
        }
      >
        <Descriptions
          bordered
          className="mb-6"
          column={2}
          items={[
            { key: 'name', label: '名称', children: data.name },
            { key: 'serviceType', label: '服务类型', children: data.serviceType || '-' },
            {
              key: 'priority',
              label: '优先级',
              children: (
                <Tag color={getPriorityColor(data.priority)}>{getPriorityLabel(data.priority)}</Tag>
              ),
            },
            {
              key: 'status',
              label: '状态',
              children: data.isActive ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>,
            },
            { key: 'response', label: '响应时限', children: formatDuration(data.responseTime) },
            { key: 'resolution', label: '解决时限', children: formatDuration(data.resolutionTime) },
            { key: 'window', label: '服务窗口', children: formatBusinessHours(data.businessHours) },
            { key: 'updated', label: '更新时间', children: displayDateTime(data.updatedAt) },
            { key: 'description', label: '描述', span: 2, children: data.description || '无' },
          ]}
        />

        <Divider>适用条件</Divider>
        {isEmptyRecord(data.conditions) ? (
          <p className="text-gray-500 mb-6">未配置适用条件。</p>
        ) : (
          <pre className="bg-gray-50 p-3 rounded text-sm overflow-auto mb-6">
            {JSON.stringify(data.conditions, null, 2)}
          </pre>
        )}

        <Divider>升级规则</Divider>
        {!data.escalationLevels || data.escalationLevels.length === 0 ? (
          <p className="text-gray-500">未配置升级级别。</p>
        ) : (
          <Space wrap size={8}>
            {data.escalationLevels.map(level => (
              <Tag key={level.level} color="blue">
                {formatEscalationLevel(level)}
              </Tag>
            ))}
          </Space>
        )}
      </Card>
    </div>
  );
};

export default SLADetail;
