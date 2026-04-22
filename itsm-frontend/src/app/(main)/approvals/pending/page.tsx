'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { Card, Space, Button, Table, Tag, App, Empty } from 'antd';
import { ReloadOutlined, CheckCircleOutlined } from '@ant-design/icons';

import { serviceRequestAPI, ServiceRequest } from '@/lib/api/service-request-api';
import { useI18n } from '@/lib/i18n';

function statusTag(status: string, t: (key: string) => string) {
  const cfg: Record<string, { color: string; label: string }> = {
    submitted: { color: 'gold', label: t('serviceRequestStatus.submitted') },
    manager_approved: { color: 'blue', label: t('serviceRequestStatus.manager_approved') },
    it_approved: { color: 'geekblue', label: t('serviceRequestStatus.it_approved') },
    security_approved: { color: 'green', label: t('serviceRequestStatus.security_approved') },
    provisioning: { color: 'cyan', label: t('serviceRequestStatus.provisioning') },
    delivered: { color: 'green', label: t('serviceRequestStatus.delivered') },
    failed: { color: 'red', label: t('serviceRequestStatus.failed') },
    rejected: { color: 'red', label: t('serviceRequestStatus.rejected') },
    cancelled: { color: 'default', label: t('serviceRequestStatus.cancelled') },
  };
  const map = cfg[status] || { color: 'default', label: status || '-' };
  return <Tag color={map.color}>{map.label}</Tag>;
}

export default function PendingApprovalsPage() {
  const { message } = App.useApp();
  const { t: i18nT } = useI18n();
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<ServiceRequest[]>([]);
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(10);
  const [total, setTotal] = useState(0);

  const load = async (p = page, s = size) => {
    try {
      setLoading(true);
      const data = await serviceRequestAPI.getPendingApprovals({ page: p, size: s });
      setRows(data.requests || []);
      setTotal(data.total || 0);
      setPage(data.page || p);
      setSize(data.size || s);
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '加载待办失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load(1, size);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="max-w-6xl mx-auto p-6">
      <Space orientation="vertical" size={16} className="w-full">
        <Card className="rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center justify-between w-full">
            <div>
              <h2 className="text-2xl font-bold text-gray-900 mb-0">我的待办审批</h2>
              <p className="text-gray-500 mt-1 mb-0">V0：按角色返回当前需要我处理的服务请求</p>
            </div>
            <Button icon={<ReloadOutlined />} onClick={() => load(page, size)} loading={loading}>
              刷新
            </Button>
          </div>
        </Card>

        <Card className="rounded-lg shadow-sm border border-gray-200">
          {rows.length === 0 && !loading ? (
            <Empty description="暂无待审批事项" image={Empty.PRESENTED_IMAGE_SIMPLE}>
              <div className="text-center">
                <CheckCircleOutlined className="text-4xl text-green-500 mb-4" />
                <p className="text-gray-500">所有审批事项已处理完成</p>
              </div>
            </Empty>
          ) : (
            <Table<ServiceRequest>
              rowKey="id"
              loading={loading}
              dataSource={rows}
              pagination={{
                current: page,
                pageSize: size,
                total,
                showSizeChanger: true,
                onChange: (p, s) => load(p, s),
              }}
              columns={[
                {
                  title: 'ID',
                  dataIndex: 'id',
                  width: 90,
                },
                {
                  title: '标题',
                  key: 'title',
                  render: (_, r) => r.title || r.catalog?.name || '-',
                },
                {
                  title: '状态',
                  dataIndex: 'status',
                  width: 180,
                  render: (v: string) => statusTag(String(v), i18nT),
                },
                {
                  title: '申请人',
                  key: 'requester',
                  width: 160,
                  render: (_, r) =>
                    r.requester?.name || r.requester?.username || String(r.requesterId || '-'),
                },
                {
                  title: '创建时间',
                  dataIndex: 'createdAt',
                  width: 220,
                  render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
                },
                {
                  title: '操作',
                  key: 'actions',
                  width: 160,
                  render: (_, r) => <Link href={`/my-requests/${r.id}`}>查看并处理</Link>,
                },
              ]}
            />
          )}
        </Card>
      </Space>
    </div>
  );
}
