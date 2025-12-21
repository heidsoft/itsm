'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { Card, Space, Typography, Button, Table, Tag, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';

import { serviceRequestAPI, ServiceRequest } from '@/lib/api/service-request-api';

const { Title, Text } = Typography;

function statusTag(status: string) {
  const map: Record<string, { color: string; label: string }> = {
    submitted: { color: 'gold', label: '已提交（待主管）' },
    manager_approved: { color: 'blue', label: '主管已批（待IT）' },
    it_approved: { color: 'geekblue', label: 'IT已批（待安全）' },
    security_approved: { color: 'green', label: '安全已批' },
    provisioning: { color: 'cyan', label: '交付中' },
    delivered: { color: 'green', label: '已交付' },
    failed: { color: 'red', label: '交付失败' },
    rejected: { color: 'red', label: '已拒绝' },
    cancelled: { color: 'default', label: '已取消' },
  };
  const cfg = map[status] || { color: 'default', label: status || '-' };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

export default function PendingApprovalsPage() {
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
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Card>
          <Space align="center" style={{ width: '100%', justifyContent: 'space-between' }}>
            <div>
              <Title level={2} style={{ marginBottom: 0 }}>
                我的待办审批
              </Title>
              <Text type="secondary">V0：按角色返回当前需要我处理的服务请求</Text>
            </div>
            <Button icon={<ReloadOutlined />} onClick={() => load(page, size)} loading={loading}>
              刷新
            </Button>
          </Space>
        </Card>

        <Card>
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
                render: (v: string) => statusTag(String(v)),
              },
              {
                title: '申请人',
                key: 'requester',
                width: 160,
                render: (_, r) => r.requester?.name || r.requester?.username || String(r.requester_id || '-'),
              },
              {
                title: '创建时间',
                dataIndex: 'created_at',
                width: 220,
                render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
              },
              {
                title: '操作',
                key: 'actions',
                width: 160,
                render: (_, r) => (
                  <Link href={`/my-requests/${r.id}`}>
                    查看并处理
                  </Link>
                ),
              },
            ]}
          />
        </Card>
      </Space>
    </div>
  );
}


