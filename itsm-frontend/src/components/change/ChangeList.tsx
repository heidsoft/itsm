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
  App,
  Modal,
  Empty,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Search, Plus, Pencil, Trash2, Eye, RefreshCw } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ChangeApi } from '@/lib/api/';
import type {
  ChangeType,
  ChangePriority} from '@/constants/change';
import {
  ChangeStatus,
  ChangeImpact,
  ChangeRisk,
  ChangeStatusLabels,
  ChangeTypeLabels,
  ChangePriorityLabels,
} from '@/constants/change';
import type { Change, ChangeQuery } from '@/types/biz/change';

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

interface ChangeListProps {
  showHeader?: boolean;
  /** 外部传入的搜索关键词 */
  search?: string;
  /** 外部传入的状态筛选 */
  status?: string;
  /** 外部传入的风险等级筛选 */
  risk?: string;
}

const ChangeList: React.FC<ChangeListProps> = ({ showHeader = true, search, status, risk }) => {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Change[]>([]);
  const [total, setTotal] = useState(0);
  const [form] = Form.useForm();

  const [query, setQuery] = useState<ChangeQuery>({
    page: 1,
    pageSize: 10,
  });

  // 当外部筛选条件变化时，同步到表单并重新加载
  useEffect(() => {
    if (search !== undefined || status !== undefined || risk !== undefined) {
      form.setFieldsValue({
        search: search || undefined,
        status: status || undefined,
        type: undefined, // risk在API中可能对应type，这里先不处理
      });
      setQuery(prev => ({ ...prev, page: 1 }));
    }
  }, [search, status, risk, form]);

  const loadData = async () => {
    setLoading(true);
    try {
      const values = await form.validateFields().catch(() => ({}));
      const resp = await ChangeApi.getChanges({
        ...query,
        ...values,
      });
      // 支持 snake_case 和 camelCase 格式
      setData((resp as any).changes || (resp as any).items || [] as any);
      setTotal(resp.total || 0);
    } catch (error) {
      // 只在有实际错误时显示失败消息，不是因为表单验证导致的
      if (
        error &&
        typeof error === 'object' &&
        !('name' in error && (error as { name?: string }).name === 'ValidationError')
      ) {
        console.error('Failed to load changes:', error);
        message.error('加载变更列表失败');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
     
  }, [query]);

  const handleSearch = () => {
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleReset = () => {
    form.resetFields();
    setQuery({ page: 1, pageSize: 10 });
  };

  const handleDelete = (id: number, title: string) => {
    Modal.confirm({
      title: '确认删除',
      content: (
        <div>
          <p>确定要删除变更请求「{title}」吗？</p>
          <p style={{ color: '#ff4d4f' }}>此操作不可撤销。</p>
        </div>
      ),
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await ChangeApi.deleteChange(id);
          message.success('删除成功');
          loadData();
        } catch (error) {
          message.error('删除失败');
        }
      },
    });
  };

  const columns: ColumnsType<Change> = [
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
      dataIndex:'createdByName',
      width: 100,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: unknown, record: Change) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye />}
              onClick={() => router.push(`/changes/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Pencil />}
              onClick={() => router.push(`/changes/${record.id}/edit`)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Button
              type="text"
              danger
              icon={<Trash2 />}
              onClick={() => handleDelete(record.id, record.title)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      {showHeader && (
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">变更管理</h1>
            <p className="text-gray-500 mt-1">管理IT基础架构和服务的变更请求，最小化变更风险</p>
          </div>
          <Button
            type="primary"
            icon={<Plus />}
            onClick={() => router.push('/changes/new')}
            size="large"
          >
            新建变更
          </Button>
        </div>
      )}

      <Card className="rounded-lg shadow-sm border border-gray-200">
        <Form form={form} layout="inline" className="mb-6 flex-wrap gap-y-4">
          <Form.Item name="search" className="mb-0">
            <Input
              placeholder="搜索标题"
              allowClear
              prefix={<Search className="text-gray-400" />}
              className="w-64"
            />
          </Form.Item>
          <Form.Item name="status" className="mb-0">
            <Select placeholder="状态" className="w-32" allowClear>
              {Object.entries(ChangeStatusLabels).map(([value, label]) => (
                <Option key={value} value={value}>
                  {label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="type" className="mb-0">
            <Select placeholder="类型" className="w-32" allowClear>
              {Object.entries(ChangeTypeLabels).map(([value, label]) => (
                <Option key={value} value={value}>
                  {label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item className="mb-0">
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<RefreshCw />} onClick={loadData} />
            </Space>
          </Form.Item>
        </Form>

        {data.length === 0 && !loading ? (
          <Empty description="暂无变更记录" image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Button type="primary" onClick={() => router.push('/changes/new')}>
              创建第一个变更
            </Button>
          </Empty>
        ) : (
          <Table
            rowKey="id"
            columns={columns}
            dataSource={data}
            loading={loading}
            pagination={{
              current: query.page,
              pageSize: query.pageSize,
              total: total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `共 ${total} 条记录`,
              pageSizeOptions: ['10', '20', '50', '100'],
              onChange: (page, pageSize) => setQuery(prev => ({ ...prev, page, pageSize })),
            }}
            scroll={{ x: 1000 }}
            getPopupContainer={node => node.parentElement || document.body}
          />
        )}
      </Card>
    </div>
  );
};

export default ChangeList;
