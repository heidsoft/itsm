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
      setError(err instanceof Error ? err.message : 'Failed to load SLA details');
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
          title={error ? 'SLA details loading failed' : 'The SLA definition was not found'}
          description={error ?? `SLA Definition #${id ?? ''} Does not exist or does not belong to the current tenant.`}
          action={
            <Space>
              {error && (
                <Button size="small" icon={<RefreshCw className="w-4 h-4" />} onClick={loadDetail}>

                  Reload
                </Button>
              )}
              <Button type="primary" onClick={() => router.push('/sla')}>

                Back to List
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
          { title: 'Home page' },
          { title: 'Service Level Management' },
          { title: 'SLA Definition', onClick: () => router.push('/sla') },
          { title: 'Details' },
        ]}
      />

      <Card
        className="rounded-lg shadow-sm border border-gray-200"
        title={<span className="text-lg font-bold">{data.name}</span>}
        extra={
          <Button icon={<ArrowLeft />} onClick={() => router.push('/sla')}>

            Back to List
          </Button>
        }
      >
        <Descriptions
          bordered
          className="mb-6"
          column={2}
          items={[
            { key: 'name', label: 'Name', children: data.name },
            { key: 'serviceType', label: 'Service Type', children: data.serviceType || '-' },
            {
              key: 'priority',
              label: 'Priority',
              children: (
                <Tag color={getPriorityColor(data.priority)}>{getPriorityLabel(data.priority)}</Tag>
              ),
            },
            {
              key: 'status',
              label: 'Status',
              children: data.isActive ? <Tag color="green">Enabled</Tag> : <Tag color="red">Disable</Tag>,
            },
            { key: 'response', label: 'Response time limit', children: formatDuration(data.responseTime) },
            { key: 'resolution', label: 'Resolution time limit', children: formatDuration(data.resolutionTime) },
            { key: 'window', label: 'Service window', children: formatBusinessHours(data.businessHours) },
            { key: 'updated', label: 'Updated At', children: displayDateTime(data.updatedAt) },
            { key: 'description', label: 'Description', span: 2, children: data.description || 'None' },
          ]}
        />

        <Divider>Applicable conditions</Divider>
        {isEmptyRecord(data.conditions) ? (
          <p className="text-gray-500 mb-6">No applicable conditions are configured.</p>
        ) : (
          <pre className="bg-gray-50 p-3 rounded text-sm overflow-auto mb-6">
            {JSON.stringify(data.conditions, null, 2)}
          </pre>
        )}

        <Divider>EscalationRule</Divider>
        {!data.escalationLevels || data.escalationLevels.length === 0 ? (
          <p className="text-gray-500">No upgrade level configured.</p>
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
