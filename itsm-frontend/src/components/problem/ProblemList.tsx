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
} from '@/constants/problem';
import type { Problem, ProblemQuery } from '@/types/biz/problem';

const { Option } = Select;

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
}

const ProblemList: React.FC<ProblemListProps> = ({
  showHeader = true,
  keyword,
  status,
  priority,
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
      setData(resp.problems || resp.items || []);
      setTotal(resp.total || 0);
    } catch (error) {
      // 只在有实际错误时显示失败消息，不是因为表单验证导致的
      if (
        error &&
        typeof error === 'object' &&
        !('name' in error && (error as { name?: string }).name === 'ValidationError')
      ) {
        console.error('Failed to load problems:', error);
        message.error('加载问题列表失败');
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

  const handleDelete = (id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除该问题吗？此操作不可撤销。',
      onOk: async () => {
        try {
          await ProblemApi.deleteProblem(id);
          message.success('删除成功');
          loadData();
        } catch (error) {
          message.error('删除失败');
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
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: ProblemStatus) => (
        <Tag color={statusColors[status]}>{ProblemStatusLabels[status]}</Tag>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      width: 100,
      render: (priority: ProblemPriority) => (
        <Tag color={priorityColors[priority]}>{ProblemPriorityLabels[priority]}</Tag>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      width: 120,
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: unknown, record: Problem) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye />}
              onClick={() => router.push(`/problems/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Pencil />}
              onClick={() => router.push(`/problems/${record.id}/edit`)}
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
    <div className="p-6">
      {showHeader && (
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">问题管理</h1>
            <p className="text-gray-500 mt-1">识别、分析和消除事件发生的根本原因</p>
          </div>
          <Button
            type="primary"
            icon={<Plus />}
            onClick={() => router.push('/problems/new')}
            size="large"
          >
            新建问题
          </Button>
        </div>
      )}

      <Card className="rounded-lg shadow-sm border border-gray-200">
        <Form form={form} layout="inline" className="mb-6 flex-wrap gap-y-4">
          <Form.Item name="keyword" className="mb-0">
            <Input
              placeholder="搜索标题或内容"
              allowClear
              prefix={<Search className="text-gray-400" />}
              className="w-64"
            />
          </Form.Item>
          <Form.Item name="status" className="mb-0">
            <Select placeholder="状态" className="w-32" allowClear>
              {Object.entries(ProblemStatusLabels).map(([value, label]) => (
                <Option key={value} value={value}>
                  {label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="priority" className="mb-0">
            <Select placeholder="优先级" className="w-32" allowClear>
              {Object.entries(ProblemPriorityLabels).map(([value, label]) => (
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
              <Button onClick={handleReset}>重置</Button>
            </Space>
          </Form.Item>
        </Form>

        {data.length === 0 && !loading ? (
          <Empty description="暂无问题记录" image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Button type="primary" onClick={() => router.push('/problems/new')}>
              创建第一个问题
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
              showTotal: total => `共 ${total} 条记录`,
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
