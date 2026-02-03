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
  message,
  Modal,
} from 'antd';
import {
  SearchOutlined,
  EyeOutlined,
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ProblemApi } from '../api';
import {
  ProblemStatus,
  ProblemPriority,
  ProblemStatusLabels,
  ProblemPriorityLabels,
} from '../constants';
import type { Problem, ProblemQuery } from '../types';

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

const ProblemList: React.FC = () => {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Problem[]>([]);
  const [total, setTotal] = useState(0);
  const [form] = Form.useForm();

  const [query, setQuery] = useState<ProblemQuery>({
    page: 1,
    page_size: 10,
  });

  const loadData = async () => {
    setLoading(true);
    try {
      const values = await form.validateFields();
      const resp = await ProblemApi.getProblems({
        ...query,
        ...values,
      });
      setData(resp.problems || []);
      setTotal(resp.total || 0);
    } catch (error) {
      // console.error('Failed to load problems:', error);
      message.error('加载问题列表失败');
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

  const columns = [
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
      render: (_: any, record: Problem) => (
        <Space size='small'>
          <Tooltip title='查看详情'>
            <Button
              type='text'
              icon={<EyeOutlined />}
              onClick={() => router.push(`/problems/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title='编辑'>
            <Button
              type='text'
              icon={<EditOutlined />}
              onClick={() => router.push(`/problems/${record.id}/edit`)}
            />
          </Tooltip>
          <Tooltip title='删除'>
            <Button
              type='text'
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record.id)}
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
          <h1 className="text-2xl font-bold text-gray-900">问题管理</h1>
          <p className="text-gray-500 mt-1">识别、分析和消除事件发生的根本原因</p>
        </div>
        <Button
          type='primary'
          icon={<PlusOutlined />}
          onClick={() => router.push('/problems/create')}
          size="large"
        >
          新建问题
        </Button>
      </div>

      <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
        <Form form={form} layout='inline' className="mb-6 flex-wrap gap-y-4">
          <Form.Item name='keyword' className="mb-0">
            <Input 
              placeholder='搜索标题或内容' 
              allowClear 
              prefix={<SearchOutlined className="text-gray-400" />} 
              className="w-64"
            />
          </Form.Item>
          <Form.Item name='status' className="mb-0">
            <Select placeholder='状态' className="w-32" allowClear>
              {Object.entries(ProblemStatusLabels).map(([value, label]) => (
                <Option key={value} value={value}>
                  {label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name='priority' className="mb-0">
            <Select placeholder='优先级' className="w-32" allowClear>
              {Object.entries(ProblemPriorityLabels).map(([value, label]) => (
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
              <Button onClick={handleReset}>重置</Button>
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
            onChange: (page, page_size) => setQuery(prev => ({ ...prev, page, page_size })),
            showSizeChanger: true,
            showTotal: t => `共 ${t} 条记录`,
          }}
          scroll={{ x: 1000 }}
        />
      </Card>
    </div>
  );
};

export default ProblemList;
