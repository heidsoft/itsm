'use client';

import { RefreshCw, AlertTriangle, PlusCircle, Eye, Edit, MoreHorizontal } from 'lucide-react';

import React, { useState, useEffect } from 'react';
import { Card, Table, Button, Space, Row, Col, Select, Input, message, Tag, Badge } from 'antd';
import {
  problemService,
  Problem,
  ProblemStatus,
  ProblemPriority,
  ListProblemsParams,
} from '@/lib/services/problem-service';
// AppLayout is handled by layout.tsx
import { useRouter } from 'next/navigation';

const { Search: SearchInput } = Input;
const { Option } = Select;

const ProblemListPage = () => {
  const [problems, setProblems] = useState<Problem[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<string>('');
  const [searchText, setSearchText] = useState('');
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const router = useRouter();

  useEffect(() => {
    fetchProblems();
    fetchStats();
  }, [pagination.current, pagination.pageSize, filter, searchText]);

  const fetchProblems = async () => {
    setLoading(true);
    try {
      const params: ListProblemsParams = {
        page: pagination.current,
        page_size: pagination.pageSize,
      };

      if (filter) {
        Object.assign(params, { status: filter as ProblemStatus });
      }

      if (searchText) {
        Object.assign(params, { keyword: searchText });
      }

      const response = await problemService.listProblems(params);
      setProblems(response.problems);
      setPagination(prev => ({ ...prev, total: response.total }));
    } catch {
      message.error('获取问题列表失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      // TODO: 实现统计数据获取
      console.log('获取统计数据');
    } catch {
      message.error('获取统计数据失败');
    }
  };

  const handleTableChange = (page: number, pageSize?: number) => {
    setPagination(prev => ({
      ...prev,
      current: page,
      pageSize: pageSize || prev.pageSize,
    }));
  };

  // 刷新数据
  const handleCreateProblem = () => {
    router.push('/problems/new');
  };

  const statusConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    open: {
      color: '#fa8c16',
      text: '待处理',
      backgroundColor: '#fff7e6',
    },
    'in-progress': {
      color: '#1890ff',
      text: '处理中',
      backgroundColor: '#e6f7ff',
    },
    resolved: {
      color: '#52c41a',
      text: '已解决',
      backgroundColor: '#f6ffed',
    },
    closed: {
      color: '#00000073',
      text: '已关闭',
      backgroundColor: '#fafafa',
    },
  };

  const priorityConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    low: {
      color: '#52c41a',
      text: '低',
      backgroundColor: '#f6ffed',
    },
    medium: {
      color: '#1890ff',
      text: '中',
      backgroundColor: '#e6f7ff',
    },
    high: {
      color: '#fa8c16',
      text: '高',
      backgroundColor: '#fff7e6',
    },
    critical: {
      color: '#ff4d4f',
      text: '紧急',
      backgroundColor: '#fff2f0',
    },
  };

  // 渲染统计卡片
  const renderStatsCards = () => (
    <div className='mb-4'>
      <Row gutter={[12, 12]}>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card 
            className='text-center hover:shadow-lg transition-all duration-300 border-0 bg-gradient-to-br from-blue-500 to-blue-600 text-white'
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <AlertTriangle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>0</div>
            <div className='text-blue-100 font-medium text-xs'>总问题数</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card 
            className='text-center hover:shadow-lg transition-all duration-300 border-0 bg-gradient-to-br from-orange-500 to-orange-600 text-white'
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <AlertTriangle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>0</div>
            <div className='text-orange-100 font-medium text-xs'>待处理</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card 
            className='text-center hover:shadow-lg transition-all duration-300 border-0 bg-gradient-to-br from-cyan-500 to-cyan-600 text-white'
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <RefreshCw className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>0</div>
            <div className='text-cyan-100 font-medium text-xs'>处理中</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card 
            className='text-center hover:shadow-lg transition-all duration-300 border-0 bg-gradient-to-br from-green-500 to-green-600 text-white'
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <AlertTriangle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>0</div>
            <div className='text-green-100 font-medium text-xs'>已解决</div>
          </Card>
        </Col>
      </Row>
    </div>
  );

  // 渲染筛选器
  const renderFilters = () => (
    <Card
      className='mb-6 bg-white shadow-sm border border-gray-200 rounded-lg'
      style={{ marginBottom: 24 }}
    >
      <div className='mb-4'>
        <h3 className='text-lg font-semibold text-gray-900 mb-1'>筛选器</h3>
        <p className='text-sm text-gray-500'>使用筛选条件快速查找问题</p>
      </div>
      <Row gutter={20} align='middle'>
        <Col xs={24} sm={12} md={8}>
          <div className='space-y-2'>
            <label className='block text-sm font-medium text-gray-700'>搜索</label>
            <SearchInput
              placeholder='搜索问题标题、ID或描述...'
              allowClear
              onSearch={value => setSearchText(value)}
              size='large'
              enterButton
              className='rounded-md'
              style={{
                borderRadius: '6px',
              }}
            />
          </div>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <div className='space-y-2'>
            <label className='block text-sm font-medium text-gray-700'>状态</label>
            <Select
              placeholder='状态筛选'
              size='large'
              allowClear
              value={filter}
              onChange={value => setFilter(value)}
              style={{ width: '100%' }}
              className='rounded-md'
            >
              <Option value={ProblemStatus.OPEN}>
                <div className='flex items-center gap-2'>
                  <div className='w-2 h-2 bg-orange-500 rounded-full'></div>
                  待处理
                </div>
              </Option>
              <Option value={ProblemStatus.IN_PROGRESS}>
                <div className='flex items-center gap-2'>
                  <div className='w-2 h-2 bg-blue-500 rounded-full'></div>
                  处理中
                </div>
              </Option>
              <Option value={ProblemStatus.RESOLVED}>
                <div className='flex items-center gap-2'>
                  <div className='w-2 h-2 bg-green-500 rounded-full'></div>
                  已解决
                </div>
              </Option>
              <Option value={ProblemStatus.CLOSED}>
                <div className='flex items-center gap-2'>
                  <div className='w-2 h-2 bg-gray-500 rounded-full'></div>
                  已关闭
                </div>
              </Option>
            </Select>
          </div>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <div className='space-y-2'>
            <label className='block text-sm font-medium text-gray-700'>操作</label>
            <Button
              icon={<RefreshCw size={20} />}
              onClick={fetchProblems}
              loading={loading}
              size='large'
              style={{ width: '100%' }}
              className='flex items-center justify-center gap-2 bg-blue-50 border-blue-200 text-blue-700 hover:bg-blue-100 hover:border-blue-300 rounded-md transition-colors duration-200'
            >
              刷新
            </Button>
          </div>
        </Col>
      </Row>
    </Card>
  );

  // 渲染问题列表
  const renderProblemList = () => (
    <Card className='bg-white shadow-sm border border-gray-200 rounded-lg'>
      <div className='mb-6 flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4'>
        <div className='flex flex-col sm:flex-row sm:items-center gap-4'>
          <div>
            <h2 className='text-xl font-semibold text-gray-900'>问题列表</h2>
            <p className='text-sm text-gray-500 mt-1'>管理和跟踪系统问题</p>
          </div>
          {selectedRowKeys.length > 0 && (
            <Badge
              count={selectedRowKeys.length}
              className='bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm font-medium'
            >
              <span className='text-sm text-gray-600'>已选中 {selectedRowKeys.length} 项</span>
            </Badge>
          )}
        </div>
        <div className='flex gap-3'>
          <Button
            type='primary'
            icon={<PlusCircle size={16} />}
            onClick={handleCreateProblem}
            className='bg-blue-600 hover:bg-blue-700 border-blue-600 hover:border-blue-700 rounded-md shadow-sm transition-colors duration-200 flex items-center gap-2'
          >
            创建问题
          </Button>
        </div>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        columns={columns}
        dataSource={problems}
        rowKey='id'
        loading={loading}
        pagination={{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          onChange: handleTableChange,
        }}
        scroll={{ x: 1200 }}
        className='[&_.ant-table-thead>tr>th]:bg-gray-50 [&_.ant-table-thead>tr>th]:border-b-2 [&_.ant-table-thead>tr>th]:border-gray-200 [&_.ant-table-thead>tr>th]:font-semibold [&_.ant-table-thead>tr>th]:text-gray-900 [&_.ant-table-tbody>tr:hover>td]:bg-blue-50 [&_.ant-table-tbody>tr>td]:border-b [&_.ant-table-tbody>tr>td]:border-gray-100'
      />
    </Card>
  );

  // 表格列定义
  const columns = [
    {
      title: '问题信息',
      key: 'problem_info',
      width: 300,
      render: (_: unknown, record: Problem) => (
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <div
            style={{
              width: 40,
              height: 40,
              backgroundColor: '#e6f7ff',
              borderRadius: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginRight: 12,
            }}
          >
            <AlertTriangle size={20} style={{ color: '#1890ff' }} />
          </div>
          <div>
            <div style={{ fontWeight: 'medium', color: '#000', marginBottom: 4 }}>
              {record.title}
            </div>
            <div style={{ fontSize: 'small', color: '#666' }}>
              #{record.id} • {record.category}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: ProblemStatus) => {
        const config = statusConfig[status];
        return (
          <span
            style={{
              padding: '4px 12px',
              borderRadius: 16,
              fontSize: 'small',
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: ProblemPriority) => {
        const config = priorityConfig[priority];
        return (
          <span
            style={{
              padding: '4px 12px',
              borderRadius: 16,
              fontSize: 'small',
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: '影响范围',
      dataIndex: 'impact',
      key: 'impact',
      width: 120,
      render: (impact: string) => {
        const impactConfig: Record<string, { color: string; text: string }> = {
          low: { color: 'green', text: '低' },
          medium: { color: 'orange', text: '中' },
          high: { color: 'red', text: '高' },
        };
        const config = impactConfig[impact] || { color: 'default', text: impact };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '负责人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 150,
      render: (assignee: { name: string }) => (
        <div style={{ fontSize: 'small' }}>{assignee?.name || '未分配'}</div>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (created_at: string) => (
        <div style={{ fontSize: 'small', color: '#666' }}>
          {new Date(created_at).toLocaleDateString('zh-CN')}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Problem) => (
        <Space size='small'>
          <Button
            type='text'
            size='small'
            icon={<Eye size={16} />}
            onClick={() => router.push(`/problems/${record.id}`)}
            className='text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-md transition-colors duration-200'
            title='查看详情'
          />
          <Button
            type='text'
            size='small'
            icon={<Edit size={16} />}
            onClick={() => router.push(`/problems/${record.id}/edit`)}
            className='text-green-600 hover:text-green-800 hover:bg-green-50 rounded-md transition-colors duration-200'
            title='编辑问题'
          />
          <Button
            type='text'
            size='small'
            icon={<MoreHorizontal size={16} />}
            className='text-gray-600 hover:text-gray-800 hover:bg-gray-50 rounded-md transition-colors duration-200'
            title='更多操作'
          />
        </Space>
      ),
    },
  ];

  return (
    <div>
      {renderStatsCards()}
      {renderFilters()}
      {renderProblemList()}
    </div>
  );
};

export default ProblemListPage;
