'use client';

/**
 * SLA 定义列表组件
 */

import React, { useState, useEffect } from 'react';
import {
  Table,
  Tag,
  Button,
  Card,
  Space,
  Tooltip,
  App,
  Modal,
  Breadcrumb,
  Switch,
  Empty,
} from 'antd';
import { Plus, Pencil, Trash2, RotateCcw, Bell } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { SLAApi } from '@/lib/api/sla-api';
import type { SLADefinition } from '@/lib/api/sla-api';
import { SLAPriorityLabels, SLAPriorityColors } from '@/constants/sla';

const SLAList: React.FC = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<SLADefinition[]>([]);
  const [total, setTotal] = useState(0);
  const [pagination, setPagination] = useState({ page: 1, size: 10 });

  const loadData = async () => {
    setLoading(true);
    try {
      const resp = await SLAApi.getDefinitions(pagination);
      setData(resp.items || []);
      setTotal(resp.total || 0);
    } catch (error) {
      message.error('加载 SLA 列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
     
  }, [pagination]);

  const handleToggleActive = async (record: SLADefinition, checked: boolean) => {
    try {
      await SLAApi.updateDefinition(record.id, { isActive: checked });
      message.success(`${checked ? '激活' : '禁用'}成功`);
      loadData();
    } catch (e) {
      message.error('操作失败');
    }
  };

  const handleDelete = (id: number) => {
    Modal.confirm({
      title: '确定要删除此 SLA 定义吗？',
      content: '删除后无法恢复，且可能影响相关合规性检查。',
      onOk: async () => {
        try {
          await SLAApi.deleteDefinition(id);
          message.success('删除成功');
          loadData();
        } catch (e) {
          message.error('删除失败');
        }
      },
    });
  };

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: SLADefinition) => (
        <a onClick={() => router.push(`/sla/definitions/${record.id}`)}>{text}</a>
      ),
    },
    {
      title: '适用优先级',
      dataIndex: 'priority',
      render: (priority: string) => (
        <Tag color={SLAPriorityColors[priority] || 'blue'}>
          {SLAPriorityLabels[priority] || priority}
        </Tag>
      ),
    },
    {
      title: '响应时间(分)',
      dataIndex:'responseTime',
      width: 120,
    },
    {
      title: '解决时间(分)',
      dataIndex:'resolutionTime',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      width: 100,
      render: (active: boolean, record: SLADefinition) => (
        <Switch
          size="small"
          checked={active}
          onChange={checked => handleToggleActive(record, checked)}
        />
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: SLADefinition) => (
        <Space size="small">
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Pencil />}
              onClick={() => router.push(`/sla/definitions/${record.id}/edit`)}
            />
          </Tooltip>
          <Tooltip title="预警规则">
            <Button
              type="text"
              icon={<Bell />}
              onClick={() => router.push(`/sla/definitions/${record.id}/alerts`)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Button
              type="text"
              danger
              icon={<Trash2 />}
              onClick={() => handleDelete(record.id)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <Card className="rounded-lg shadow-sm border border-gray-200">
      <Breadcrumb className="mb-4">
        <Breadcrumb.Item>首页</Breadcrumb.Item>
        <Breadcrumb.Item>服务级别管理</Breadcrumb.Item>
        <Breadcrumb.Item>SLA 定义</Breadcrumb.Item>
      </Breadcrumb>

      <div className="flex justify-between mb-4">
        <Space>
          <Button
            type="primary"
            icon={<Plus />}
            onClick={() => router.push('/sla/definitions/new')}
          >
            新建 SLA
          </Button>
        </Space>
        <Button icon={<RotateCcw />} onClick={loadData} />
      </div>

      <Table
        rowKey="id"
        columns={columns as any}
        dataSource={data}
        loading={loading}
        scroll={{ x: 'max-content' }}
        locale={{
          emptyText: (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无SLA数据">
              <Button type="primary" onClick={() => router.push('/sla/definitions/new')}>
                创建第一个SLA
              </Button>
            </Empty>
          ),
        }}
        pagination={{
          current: pagination.page,
          pageSize: pagination.size,
          total: total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: total => `共 ${total} 条记录`,
          pageSizeOptions: ['10', '20', '50', '100'],
          onChange: (page, size) => setPagination({ page, size }),
        }}
        getPopupContainer={node => node.parentElement || document.body}
      />
    </Card>
  );
};

export default SLAList;
