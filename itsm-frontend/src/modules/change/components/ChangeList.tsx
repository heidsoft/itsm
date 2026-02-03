'use client';

/**
 * 变更列表组件
 */

import React, { useState, useEffect } from 'react';
import {
  Table,
  Tag,
  Button,
  Card,
  Space,
  Tooltip,
  Input,
  Select,
  Form,
  message,
  Modal,
} from 'antd';
import {
  SearchOutlined,
  EyeOutlined,
  EditOutlined,
  PlusOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ChangeApi } from '../api';
import {
  ChangeStatus,
  ChangeType,
  ChangePriority,
  ChangeImpact,
  ChangeRisk,
  ChangeStatusLabels,
  ChangeTypeLabels,
  ChangePriorityLabels,
} from '../constants';
import type { Change, ChangeQuery } from '../types';

const { Option } = Select;

// 颜色映射
const statusColors: Record<string, string> = {
  [ChangeStatus.DRAFT]: 'default',
  [ChangeStatus.PENDING]: 'orange',
  [ChangeStatus.APPROVED]: 'cyan',
  [ChangeStatus.IN_PROGRESS]: 'blue',
  [ChangeStatus.COMPLETED]: 'green',
  [ChangeStatus.REJECTED]: 'red',
  [ChangeStatus.ROLLED_BACK]: 'magenta',
  [ChangeStatus.CANCELLED]: 'default',
};

const ChangeList: React.FC = () => {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Change[]>([]);
  const [total, setTotal] = useState(0);
  const [form] = Form.useForm();

  const [query, setQuery] = useState<ChangeQuery>({
    page: 1,
    page_size: 10,
  });

  const loadData = async () => {
    setLoading(true);
    try {
      const values = await form.validateFields();
      const resp = await ChangeApi.getChanges({
        ...query,
        ...values,
      });
      setData(resp.changes || []);
      setTotal(resp.total || 0);
    } catch (error) {
      // console.error('Failed to load changes:', error);
      message.error('加载变更列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query]);

  const handleSearch = () => {
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleReset = () => {
    form.resetFields();
    setQuery({ page: 1, page_size: 10 });
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: '模型',
      dataIndex: 'type',
      width: 100,
      render: (type: ChangeType) => ChangeTypeLabels[type] || type,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: ChangeStatus) => (
        <Tag color={statusColors[status]}>{ChangeStatusLabels[status]}</Tag>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      width: 80,
      render: (priority: ChangePriority) => <Tag>{ChangePriorityLabels[priority]}</Tag>,
    },
    {
      title: '创建人',
      dataIndex: 'created_by_name',
      width: 100,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: any, record: Change) => (
        <Space size='small'>
          <Tooltip title='查看详情'>
            <Button
              type='text'
              icon={<EyeOutlined />}
              onClick={() => router.push(`/changes/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title='编辑'>
            <Button
              type='text'
              icon={<EditOutlined />}
              onClick={() => router.push(`/changes/${record.id}/edit`)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">变更管理</h1>
          <p className="text-gray-500 mt-1">管理IT基础架构和服务的变更请求，最小化变更风险</p>
        </div>
        <Button
          type='primary'
          icon={<PlusOutlined />}
          onClick={() => router.push('/changes/create')}
          size="large"
        >
          新建变更
        </Button>
      </div>

      <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
        <Form form={form} layout='inline' className="mb-6 flex-wrap gap-y-4">
          <Form.Item name='search' className="mb-0">
            <Input 
              placeholder='搜索标题' 
              allowClear 
              prefix={<SearchOutlined className="text-gray-400" />} 
              className="w-64"
            />
          </Form.Item>
          <Form.Item name='status' className="mb-0">
            <Select placeholder='状态' className="w-32" allowClear>
              {Object.entries(ChangeStatusLabels).map(([value, label]) => (
                <Option key={value} value={value}>
                  {label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name='type' className="mb-0">
            <Select placeholder='类型' className="w-32" allowClear>
              {Object.entries(ChangeTypeLabels).map(([value, label]) => (
                <Option key={value} value={value}>
                  {label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item className="mb-0">
            <Space>
              <Button type='primary' ghost onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<SyncOutlined />} onClick={loadData} />
            </Space>
          </Form.Item>
        </Form>

        <Table
          rowKey='id'
          columns={columns as any}
          dataSource={data}
          loading={loading}
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total: total,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条记录`,
            onChange: (page, page_size) => setQuery(prev => ({ ...prev, page, page_size })),
          }}
          scroll={{ x: 1000 }}
        />
      </Card>
    </div>
  );
};

export default ChangeList;
