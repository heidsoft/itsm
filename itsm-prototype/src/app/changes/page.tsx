'use client';

import { RefreshCw, GitBranch, Eye, Edit, MoreHorizontal, AlertTriangle, Plus } from 'lucide-react';

import React, { useState, useEffect } from 'react';
import { Card, Table, Button, Space, Row, Col, Select, Input, Badge } from 'antd';
import { Change, ChangeStats } from '../lib/services/change-service';

const { Search: SearchInput } = Input;
const { Option } = Select;

const getChangeStatusColor = (status: string) => {
  switch (status) {
    case 'pending':
      return {
        color: '#fa8c16',
        text: '待审批',
        backgroundColor: '#fff7e6',
      };
    case 'approved':
      return {
        color: '#1890ff',
        text: '审批通过',
        backgroundColor: '#e6f7ff',
      };
    case 'implementing':
      return {
        color: '#722ed1',
        text: '实施中',
        backgroundColor: '#f9f0ff',
      };
    case 'completed':
      return {
        color: '#52c41a',
        text: '已完成',
        backgroundColor: '#f6ffed',
      };
    case 'cancelled':
      return {
        color: '#ff4d4f',
        text: '已取消',
        backgroundColor: '#fff2f0',
      };
    case 'draft':
      return {
        color: '#8c8c8c',
        text: '草稿',
        backgroundColor: '#f5f5f5',
      };
    case 'rejected':
      return {
        color: '#ff4d4f',
        text: '已拒绝',
        backgroundColor: '#fff2f0',
      };
    default:
      return {
        color: '#8c8c8c',
        text: status,
        backgroundColor: '#f5f5f5',
      };
  }
};

const getPriorityColor = (priority: string) => {
  switch (priority) {
    case 'urgent':
      return {
        color: '#ff4d4f',
        text: '紧急',
        backgroundColor: '#fff2f0',
      };
    case 'high':
      return {
        color: '#fa8c16',
        text: '高',
        backgroundColor: '#fff7e6',
      };
    case 'medium':
      return {
        color: '#1890ff',
        text: '中',
        backgroundColor: '#e6f7ff',
      };
    case 'low':
      return {
        color: '#52c41a',
        text: '低',
        backgroundColor: '#f6ffed',
      };
    default:
      return {
        color: '#8c8c8c',
        text: priority,
        backgroundColor: '#f5f5f5',
      };
  }
};

const ChangeListPage = () => {
  const [changes, setChanges] = useState<Change[]>([]);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<ChangeStats>({
    total: 0,
    draft: 0,
    pending: 0,
    approved: 0,
    implementing: 0,
    completed: 0,
    cancelled: 0,
  });
  const [filter, setFilter] = useState('全部');
  const [searchText, setSearchText] = useState('');
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 模拟数据
  const mockChanges: Change[] = [
    {
      id: 1,
      title: '系统升级变更',
      description: '升级核心系统版本',
      justification: '提升系统性能和安全性',
      status: 'pending',
      priority: 'high',
      type: 'standard',
      impactScope: 'high',
      riskLevel: 'medium',
      createdBy: 1,
      createdByName: '张三',
      assigneeName: '李四',
      tenantId: 1,
      plannedStartDate: '2024-01-15T09:00:00Z',
      plannedEndDate: '2024-01-15T18:00:00Z',
      implementationPlan: '系统升级实施计划',
      rollbackPlan: '系统回滚计划',
      createdAt: '2024-01-10T10:00:00Z',
      updatedAt: '2024-01-10T10:00:00Z',
    },
    {
      id: 2,
      title: '数据库配置变更',
      description: '调整数据库连接池配置',
      justification: '优化数据库性能',
      status: 'approved',
      priority: 'medium',
      type: 'normal',
      impactScope: 'medium',
      riskLevel: 'low',
      createdBy: 2,
      createdByName: '王五',
      assigneeName: '赵六',
      tenantId: 1,
      plannedStartDate: '2024-01-16T14:00:00Z',
      plannedEndDate: '2024-01-16T16:00:00Z',
      implementationPlan: '数据库配置变更计划',
      rollbackPlan: '数据库配置回滚计划',
      createdAt: '2024-01-12T14:30:00Z',
      updatedAt: '2024-01-12T14:30:00Z',
    },
  ];

  const mockStats: ChangeStats = {
    total: 156,
    draft: 12,
    pending: 23,
    approved: 45,
    implementing: 18,
    completed: 88,
    cancelled: 5,
  };

  const fetchChanges = async () => {
    try {
      setLoading(true);
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      setChanges(mockChanges);
      setPagination(prev => ({ ...prev, total: mockChanges.length }));
    } catch (error) {
      console.error('获取变更列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      setStats(mockStats);
    } catch (error) {
      console.error('获取统计数据失败:', error);
    }
  };

  useEffect(() => {
    fetchChanges();
    fetchStats();
  }, []);

  const handleCreateChange = () => {
    console.log('创建变更');
  };

  const handleTableChange = (paginationInfo: {
    current?: number;
    pageSize?: number;
    total?: number;
  }) => {
    setPagination(prev => ({ ...prev, ...paginationInfo }));
    fetchChanges();
  };

  const columns = [
    {
      title: '变更信息',
      dataIndex: 'title',
      key: 'title',
      width: 300,
      render: (_: unknown, record: Change) => (
        <div>
          <div className='font-medium text-gray-900 mb-1'>{record.title}</div>
          <div className='text-sm text-gray-500 mb-2'>{record.description}</div>
          <div className='flex items-center space-x-2'>
            <Badge
              color={
                record.type === 'emergency' ? 'red' : record.type === 'standard' ? 'green' : 'blue'
              }
              text={
                record.type === 'emergency'
                  ? '紧急变更'
                  : record.type === 'standard'
                  ? '标准变更'
                  : '普通变更'
              }
            />
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const statusConfig = getChangeStatusColor(status);
        return <Badge color={statusConfig.color} text={statusConfig.text} />;
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: string) => {
        const priorityConfig = getPriorityColor(priority);
        return <Badge color={priorityConfig.color} text={priorityConfig.text} />;
      },
    },
    {
      title: '申请人',
      dataIndex: 'createdByName',
      key: 'createdByName',
      width: 120,
      render: (createdByName: string) => <div style={{ fontSize: 'small' }}>{createdByName}</div>,
    },
    {
      title: '审批人',
      dataIndex: 'assigneeName',
      key: 'assigneeName',
      width: 120,
      render: (assigneeName: string) => (
        <div style={{ fontSize: 'small' }}>{assigneeName || '未分配'}</div>
      ),
    },
    {
      title: '计划时间',
      key: 'plannedTime',
      width: 180,
      render: (_: unknown, record: Change) => (
        <div style={{ fontSize: 'small' }}>
          {record.plannedStartDate && (
            <>
              <div>{new Date(record.plannedStartDate).toLocaleDateString('zh-CN')}</div>
              <div style={{ color: '#666' }}>
                {new Date(record.plannedStartDate).toLocaleTimeString('zh-CN', {
                  hour: '2-digit',
                  minute: '2-digit',
                })}{' '}
                -{' '}
                {record.plannedEndDate &&
                  new Date(record.plannedEndDate).toLocaleTimeString('zh-CN', {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
              </div>
            </>
          )}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: Change) => (
        <Space size='small'>
          <Button
            type='text'
            icon={<Eye size={16} />}
            className='text-blue-600 hover:text-blue-800 hover:bg-blue-50'
            title='查看详情'
            onClick={() => console.log('查看变更', record.id)}
          />
          <Button
            type='text'
            icon={<Edit size={16} />}
            className='text-green-600 hover:text-green-800 hover:bg-green-50'
            title='编辑变更'
            onClick={() => console.log('编辑变更', record.id)}
          />
          <Button
            type='text'
            icon={<MoreHorizontal size={16} />}
            className='text-gray-600 hover:text-gray-800 hover:bg-gray-50'
            title='更多操作'
            onClick={() => console.log('更多操作', record.id)}
          />
        </Space>
      ),
    },
  ];

  return (
    <div className='p-6 bg-gray-50 min-h-screen'>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} md={6}>
          <Card className='bg-gradient-to-br from-blue-500 to-blue-600 border-0 shadow-lg hover:shadow-xl transition-all duration-300'>
            <div className='text-center'>
              <GitBranch className='mx-auto mb-3 text-white' size={32} />
              <div className='text-3xl font-bold text-white'>{stats.total}</div>
              <div className='text-blue-100'>总变更数</div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card className='bg-gradient-to-br from-orange-500 to-orange-600 border-0 shadow-lg hover:shadow-xl transition-all duration-300'>
            <div className='text-center'>
              <AlertTriangle className='mx-auto mb-3 text-white' size={32} />
              <div className='text-3xl font-bold text-white'>{stats.pending}</div>
              <div className='text-orange-100'>待审批</div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card className='bg-gradient-to-br from-green-500 to-green-600 border-0 shadow-lg hover:shadow-xl transition-all duration-300'>
            <div className='text-center'>
              <GitBranch className='mx-auto mb-3 text-white' size={32} />
              <div className='text-3xl font-bold text-white'>{stats.approved}</div>
              <div className='text-green-100'>已批准</div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card className='bg-gradient-to-br from-purple-500 to-purple-600 border-0 shadow-lg hover:shadow-xl transition-all duration-300'>
            <div className='text-center'>
              <GitBranch className='mx-auto mb-3 text-white' size={32} />
              <div className='text-3xl font-bold text-white'>{stats.completed}</div>
              <div className='text-purple-100'>已完成</div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 筛选器 */}
      <Card className='mb-6 shadow-sm border-0'>
        <div className='mb-4'>
          <h3 className='text-lg font-semibold text-gray-800 mb-2'>筛选条件</h3>
        </div>
        <Row gutter={[16, 16]} align='middle'>
          <Col xs={24} sm={12} md={8}>
            <SearchInput
              placeholder='搜索变更标题或描述'
              value={searchText}
              onChange={e => setSearchText(e.target.value)}
              onSearch={fetchChanges}
              className='rounded-lg'
              size='large'
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              value={filter}
              onChange={setFilter}
              className='w-full'
              size='large'
              placeholder='选择状态'
            >
              <Option value='全部'>
                <div className='flex items-center'>
                  <div className='w-2 h-2 rounded-full bg-gray-400 mr-2'></div>
                  全部状态
                </div>
              </Option>
              <Option value='pending'>
                <div className='flex items-center'>
                  <div className='w-2 h-2 rounded-full bg-orange-500 mr-2'></div>
                  待审批
                </div>
              </Option>
              <Option value='approved'>
                <div className='flex items-center'>
                  <div className='w-2 h-2 rounded-full bg-blue-500 mr-2'></div>
                  已批准
                </div>
              </Option>
              <Option value='completed'>
                <div className='flex items-center'>
                  <div className='w-2 h-2 rounded-full bg-green-500 mr-2'></div>
                  已完成
                </div>
              </Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Button
              icon={<RefreshCw size={20} />}
              onClick={fetchChanges}
              loading={loading}
              size='large'
              className='w-full bg-blue-50 border-blue-200 text-blue-600 hover:bg-blue-100 hover:border-blue-300'
            >
              刷新
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 变更列表 */}
      <Card className='shadow-sm border-0'>
        <div className='flex justify-between items-center mb-6'>
          <div>
            <h3 className='text-xl font-semibold text-gray-800 mb-1'>变更列表</h3>
            <p className='text-gray-600'>管理和跟踪所有变更请求</p>
          </div>
          <div className='flex items-center space-x-3'>
            {selectedRowKeys.length > 0 && (
              <Badge
                count={selectedRowKeys.length}
                className='bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm font-medium'
              >
                已选择 {selectedRowKeys.length} 项
              </Badge>
            )}
            <Button
              type='primary'
              icon={<Plus size={20} />}
              onClick={handleCreateChange}
              size='large'
              className='bg-blue-600 hover:bg-blue-700 border-blue-600 hover:border-blue-700 shadow-sm'
            >
              创建变更
            </Button>
          </div>
        </div>

        <Table
          columns={columns}
          dataSource={changes}
          rowKey='id'
          loading={loading}
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          onChange={handleTableChange}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          className='[&_.ant-table-thead>tr>th]:bg-gray-50 [&_.ant-table-thead>tr>th]:font-semibold [&_.ant-table-tbody>tr:hover>td]:bg-blue-50 [&_.ant-table-tbody>tr>td]:border-gray-100'
        />
      </Card>
    </div>
  );
};

export default ChangeListPage;
