'use client';

/**
 * 问题列表组件
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

import { ProblemApi } from '@/lib/api/';
import {
  ProblemStatus,
  ProblemPriority,
  ProblemStatusLabels,
  ProblemPriorityLabels,
  problemStatusLabel,
  problemPriorityLabel,
} from '@/constants/problem';
import type { Problem, ProblemQuery } from '@/types/biz/problem';



// 状态和优先级颜色映射
const statusColors: Record<string, string> = {
  [ProblemStatus.OPEN]: 'red',
  [ProblemStatus.IN_PROGRESS]: 'blue',
  [ProblemStatus.RESOLVED]: 'green',
  [ProblemStatus.CLOSED]: 'default',
};

const priorityColors: Record<string, string> = {
  [ProblemPriority.CRITICAL]: 'magenta',
  [ProblemPriority.HIGH]: 'red',
  [ProblemPriority.MEDIUM]: 'orange',
  [ProblemPriority.LOW]: 'blue',
};

interface ProblemListProps {
  showHeader?: boolean;
  /** 外部传入的关键词筛选 */
  keyword?: string;
  /** 外部传入的状态筛选 */
  status?: string;
  /** 外部传入的优先级筛选 */
  priority?: string;
  /** 受控分页：嵌入模式（showHeader=false）下由父级 Pagination 驱动 */
  page?: number;
  pageSize?: number;
}

const ProblemList: React.FC<ProblemListProps> = ({
  showHeader = true,
  keyword,
  status,
  priority,
  page,
  pageSize,
}) => {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Problem[]>([]);
  const [total, setTotal] = useState(0);
  const [form] = Form.useForm();

  const [query, setQuery] = useState<ProblemQuery>({
    page: 1,
    pageSize: 10,
  });

  // 当外部筛选条件变化时，同步到表单并重新加载
  useEffect(() => {
    if (keyword !== undefined || status !== undefined || priority !== undefined) {
      form.setFieldsValue({
        keyword: keyword || undefined,
        status: status || undefined,
        priority: priority || undefined,
      });
      setQuery(prev => ({ ...prev, page: 1 }));
    }
  }, [keyword, status, priority, form]);

  const loadData = async () => {
    setLoading(true);
    try {
      const values = await form.validateFields().catch(() => ({}));
      const resp = await ProblemApi.getProblems({
        ...query,
        ...values,
      });
      // 信封只有一个 items 键，实体只有一份 Problem，
      // 因此这里既不需要多字段 fallback，也不需要 as unknown as 断言。
      setData(resp.items);
      setTotal(resp.total);
    } catch (error) {
      // 只在有实际错误时显示失败消息，不是因为表单验证导致的
      if (
        error &&
        typeof error === 'object' &&
        !('name' in error && (error as { name?: string }).name === 'ValidationError')
      ) {
        console.error('Failed to load problems:', error);
        message.error('Failed to load problem list');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();

  }, [query]);

  // 受控分页同步：嵌入模式下表格自身 pagination={false}，分页控件在父级。
  // 2026-10-14 修复：此前不接收父级页码，父级 Pagination 翻页只改父 state，
  // 表格停在第 1 页；此处把父级 page/pageSize 同步进内部 query 触发重取。
  useEffect(() => {
    if (showHeader) return; // 独立模式自管分页，不受外部控制
    setQuery(prev => ({
      ...prev,
      page: page ?? prev.page,
      pageSize: pageSize ?? prev.pageSize,
    }));
  }, [showHeader, page, pageSize]);

  const handleSearch = () => {
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleReset = () => {
    form.resetFields();
    setQuery({ page: 1, pageSize: 10 });
  };

  const handleDelete = (id: number) => {
    Modal.confirm({
      title: 'Confirm Delete',
      content: 'Are you sure you want to delete this question? This action cannot be undone.',
      onOk: async () => {
        try {
          await ProblemApi.deleteProblem(id);
          message.success('Deleted');
          loadData();
        } catch (error) {
          message.error('Delete failed');
        }
      },
    });
  };

  const columns: ColumnsType<Problem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: 'Title',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status]}>{problemStatusLabel(status)}</Tag>
      ),
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      width: 100,
      render: (priority: string) => (
        <Tag color={priorityColors[priority]}>{problemPriorityLabel(priority)}</Tag>
      ),
    },
    {
      title: 'Category',
      dataIndex: 'category',
      width: 120,
    },
    {
      title: 'Updated At',
      dataIndex: 'updatedAt',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 150,
      render: (_: unknown, record: Problem) => (
        <Space size="small">
          <Tooltip title="View Details">
            <Button
              type="text"
              aria-label={`Inspect issues ${record.id}`}
              icon={<Eye />}
              onClick={() => router.push(`/problems/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="Edit">
            <Button
              type="text"
              aria-label={`Edit Problem ${record.id}`}
              icon={<Pencil />}
              onClick={() => router.push(`/problems/${record.id}/edit`)}
            />
          </Tooltip>
          <Tooltip title="Delete">
            <Button
              type="text"
              danger
              aria-label={`Delete issue ${record.id}`}
              icon={<Trash2 />}
              onClick={() => handleDelete(record.id)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  // When embedded in BusinessPageTemplate (showHeader=false), render a bare table
  // without Card wrapper, search form, or internal pagination — the parent
  // provides all of those. This eliminates duplicate filters and pagination.
  if (!showHeader) {
    return (
      <Table
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={false}
        scroll={{ x: 1000 }}
        getPopupContainer={node => node.parentElement || document.body}
      />
    );
  }

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Problems</h1>
          <p className="text-gray-500 mt-1">Identify, analyze and eliminate root causes of incidents</p>
        </div>
        <Button
          type="primary"
          icon={<Plus />}
          onClick={() => router.push('/problems/new')}
          size="large"
        >

          New Problem
        </Button>
      </div>

      <Card className="rounded-lg shadow-sm border border-[var(--color-border-primary)]">
        <Form form={form} layout="inline" className="mb-6 flex-wrap gap-y-4">
          <Form.Item name="keyword" className="mb-0">
            <Input
              placeholder="Search title or content"
              allowClear
              prefix={<Search className="text-gray-400" />}
              className="w-64"
            />
          </Form.Item>
          <Form.Item name="status" className="mb-0">
            <Select
              placeholder="Status"
              className="w-32"
              allowClear
              options={Object.entries(ProblemStatusLabels).map(([value, label]) => ({
                value,
                label,
              }))}
            />
          </Form.Item>
          <Form.Item name="priority" className="mb-0">
            <Select
              placeholder="Priority"
              className="w-32"
              allowClear
              options={Object.entries(ProblemPriorityLabels).map(([value, label]) => ({
                value,
                label,
              }))}
            />
          </Form.Item>
          <Form.Item className="mb-0">
            <Space>
              <Button type="primary" onClick={handleSearch}>

                Query
              </Button>
              <Button onClick={handleReset}>Reset</Button>
            </Space>
          </Form.Item>
        </Form>

        {data.length === 0 && !loading ? (
          <Empty description="No problem records" image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Button type="primary" onClick={() => router.push('/problems/new')}>

              Create issue
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
              onChange: (page, pageSize) => setQuery(prev => ({ ...prev, page, pageSize })),
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `Total ${total} records`,
              pageSizeOptions: ['10', '20', '50', '100'],
            }}
            scroll={{ x: 1000 }}
            getPopupContainer={node => node.parentElement || document.body}
          />
        )}
      </Card>
    </div>
  );
};

export default ProblemList;
