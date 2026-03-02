'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Tag,
  Space,
  Tooltip,
  Modal,
  Form,
  Badge,
  Row,
  Col,
  Statistic,
  Timeline,
  Descriptions,
  Progress,
  Alert,
  App,
} from 'antd';
import {
  Search,
  Filter,
  PlayCircle,
  PauseCircle,
  StopCircle,
  Eye,
  Clock,
  User,
  Calendar,
  CheckCircle,
  AlertCircle,
  XCircle,
  RefreshCw,
  BarChart3,
  GitBranch,
  Activity,
  History,
  ThumbsUp,
  ThumbsDown,
} from 'lucide-react';
// AppLayout is handled by parent layout
import {
  WorkflowAPI,
  WorkflowInstance as ApiWorkflowInstance,
  WorkflowTask as ApiWorkflowTask,
} from '@/lib/api/workflow-api';

const { Option } = Select;

type WorkflowInstanceRecord = Omit<ApiWorkflowInstance, 'status'> & {
  status: string;
  instance_id: string;
  business_key?: string;
  workflow_name?: string;
  priority?: string;
  started_by?: string;
  started_at?: string;
  completed_at?: string;
  due_date?: string;
};

type WorkflowTaskRecord = Omit<ApiWorkflowTask, 'status'> & {
  status: string;
  name?: string;
  activity_id?: string;
  type?: string;
  assignee?: string;
  created_at?: string;
  due_date?: string;
};

const WorkflowInstancesPage = () => {
  const { message } = App.useApp();
  const [instances, setInstances] = useState<WorkflowInstanceRecord[]>([]);
  const [tasks, setTasks] = useState<WorkflowTaskRecord[]>([]);
  const [counterSignStatus, setCounterSignStatus] = useState<
    Record<
      string,
      {
        parent_task_id: string;
        total: number;
        completed: number;
        approved: number;
        rejected: number;
        pending: number;
        status: 'pending' | 'approved' | 'rejected';
      }
    >
  >({});
  const [loading, setLoading] = useState(false);
  const [votingTaskId, setVotingTaskId] = useState<string | null>(null);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedInstance, setSelectedInstance] = useState<WorkflowInstanceRecord | null>(null);
  const [filters, setFilters] = useState({
    status: '',
    workflow_id: '',
    business_key: '',
    started_by: '',
  });
  const [stats, setStats] = useState({
    total: 0,
    running: 0,
    completed: 0,
    suspended: 0,
    terminated: 0,
  });

  // 分页状态
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  useEffect(() => {
    loadInstances(1, pagination.pageSize);
    loadStats();
  }, [filters]); // eslint-disable-line react-hooks/exhaustive-deps

  // 分页加载实例列表
  const loadInstances = async (page: number = 1, pageSize: number = 10) => {
    setLoading(true);
    try {
      const response = await WorkflowAPI.listWorkflowInstances({
        page,
        page_size: pageSize,
        ...filters,
      });
      const normalized = response.instances.map((instance: any) => ({
        ...instance,
        instance_id: instance.instance_id || instance.id,
        business_key: instance.business_key || instance.businessKey || '',
        workflow_name: instance.workflow_name || instance.workflowName || '',
        priority: instance.priority || 'normal',
        started_by: instance.started_by || instance.startedByName || '',
        started_at: instance.started_at || instance.startTime,
        completed_at: instance.completed_at || instance.endTime,
        due_date: instance.due_date,
      })) as WorkflowInstanceRecord[];
      setInstances(normalized);
      // 更新分页信息
      setPagination(prev => ({
        ...prev,
        current: page,
        pageSize,
        total: response.total || 0,
      }));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      console.error('加载工作流实例失败:', error);
      message.error('加载工作流实例失败: ' + errorMessage);
    } finally {
      setLoading(false);
    }
  };

  // 分页/排序/筛选变化处理
  const handleTableChange = async (newPagination: any) => {
    await loadInstances(newPagination.current, newPagination.pageSize);
    // 分页变化后更新统计数据
    loadStats();
  };

  const loadStats = async () => {
    try {
      // 调用后端API获取真实统计数据
      const instanceStats = await WorkflowAPI.getInstanceStats({
        process_definition_key: filters.workflow_id || undefined,
        status: filters.status || undefined,
      });
      setStats({
        total: instanceStats.total,
        running: instanceStats.running,
        completed: instanceStats.completed,
        suspended: instanceStats.suspended,
        terminated: instanceStats.terminated,
      });
    } catch (error) {
      console.error('加载统计数据失败:', error);
      // 降级使用本地过滤
      setStats({
        total: pagination.total,
        running: instances.filter(i => String(i.status) === 'running').length,
        completed: instances.filter(i => String(i.status) === 'completed').length,
        suspended: instances.filter(i => String(i.status) === 'suspended').length,
        terminated: instances.filter(i => String(i.status) === 'terminated').length,
      });
    }
  };

  const handleViewDetail = async (instance: WorkflowInstanceRecord) => {
    setSelectedInstance(instance);
    setDetailVisible(true);

    try {
      const tasksResponse = await WorkflowAPI.listWorkflowTasks(instance.id);
      const normalized = (tasksResponse as any[]).map(task => ({
        ...task,
        name: task.name || task.nodeName,
        activity_id: task.activity_id || task.nodeId,
        type: task.type || task.nodeType || 'task',
        assignee: task.assigneeName || task.assignee || '',
        created_at: task.created_at || task.startTime,
        due_date: task.due_date,
      })) as WorkflowTaskRecord[];
      setTasks(normalized);
    } catch (error) {
      message.error('加载任务列表失败');
    }
  };

  // 加载会签状态
  const loadCounterSignStatus = async (taskId: string) => {
    try {
      const status = await WorkflowAPI.getCounterSignStatus(taskId);
      setCounterSignStatus(prev => ({
        ...prev,
        [taskId]: status,
      }));
    } catch (error) {
      console.error('加载会签状态失败:', error);
    }
  };

  // 创建会签任务
  const handleCreateCounterSign = async (
    taskId: string,
    approvers: string[],
    approvalType: 'serial' | 'parallel' = 'parallel'
  ) => {
    try {
      await WorkflowAPI.createCounterSignTasks(taskId, approvers, approvalType);
      message.success('会签任务创建成功');
      // 刷新会签状态
      await loadCounterSignStatus(taskId);
      // 刷新任务列表
      if (selectedInstance) {
        const tasksResponse = await WorkflowAPI.listWorkflowTasks(selectedInstance.id);
        const normalized = (tasksResponse as any[]).map(task => ({
          ...task,
          name: task.name || task.nodeName,
          activity_id: task.activity_id || task.nodeId,
          type: task.type || task.nodeType || 'task',
          assignee: task.assigneeName || task.assignee || '',
          created_at: task.created_at || task.startTime,
          due_date: task.due_date,
        })) as WorkflowTaskRecord[];
        setTasks(normalized);
      }
    } catch (error) {
      message.error('创建会签任务失败');
    }
  };

  // 投票（通过/拒绝）
  const handleVote = async (taskId: string, approved: boolean, comment?: string) => {
    try {
      setVotingTaskId(taskId);
      await WorkflowAPI.vote(taskId, approved, comment);
      message.success(approved ? '已同意' : '已拒绝');

      // 查找父任务ID并刷新会签状态
      const task = tasks.find(t => t.id === taskId);
      if (task?.activity_id) {
        await loadCounterSignStatus(task.activity_id);
      }

      // 刷新任务列表
      if (selectedInstance) {
        const tasksResponse = await WorkflowAPI.listWorkflowTasks(selectedInstance.id);
        const normalized = (tasksResponse as any[]).map(t => ({
          ...t,
          name: t.name || t.nodeName,
          activity_id: t.activity_id || t.nodeId,
          type: t.type || t.nodeType || 'task',
          assignee: t.assigneeName || t.assignee || '',
          created_at: t.created_at || t.startTime,
          due_date: t.due_date,
        })) as WorkflowTaskRecord[];
        setTasks(normalized);
      }
    } catch (error) {
      message.error('投票失败');
    } finally {
      setVotingTaskId(null);
    }
  };

  const handleSuspendInstance = async (instanceId: string) => {
    try {
      await WorkflowAPI.suspendWorkflow(instanceId);
      message.success('实例暂停成功');
      loadInstances();
    } catch (error) {
      message.error('暂停失败');
    }
  };

  const handleResumeInstance = async (instanceId: string) => {
    try {
      await WorkflowAPI.resumeWorkflow(instanceId);
      message.success('实例恢复成功');
      loadInstances();
    } catch (error) {
      message.error('恢复失败');
    }
  };

  const handleTerminateInstance = async (instanceId: string) => {
    try {
      await WorkflowAPI.terminateWorkflow(instanceId, '手动终止');
      message.success('实例终止成功');
      loadInstances();
    } catch (error) {
      message.error('终止失败');
    }
  };

  const getStatusColor = (status: string) => {
    const colors = {
      running: 'green',
      completed: 'blue',
      suspended: 'orange',
      terminated: 'red',
    };
    return colors[status as keyof typeof colors] || 'default';
  };

  const getStatusText = (status: string) => {
    const texts = {
      running: '运行中',
      completed: '已完成',
      suspended: '已暂停',
      terminated: '已终止',
    };
    return texts[status as keyof typeof texts] || status;
  };

  const getPriorityColor = (priority: string) => {
    const colors = {
      low: 'green',
      normal: 'blue',
      high: 'orange',
      critical: 'red',
    };
    return colors[priority as keyof typeof colors] || 'default';
  };

  const columns = [
    {
      title: '实例ID',
      dataIndex: 'instance_id',
      key: 'instance_id',
      width: 150,
      render: (instanceId: string) => <span className='font-mono text-sm'>{instanceId}</span>,
    },
    {
      title: '业务键',
      dataIndex: 'business_key',
      key: 'business_key',
      width: 150,
      render: (businessKey: string) => <span className='text-sm'>{businessKey || '-'}</span>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: string) => <Tag color={getPriorityColor(priority)}>{priority}</Tag>,
    },
    {
      title: '启动人',
      dataIndex: 'started_by',
      key: 'started_by',
      width: 120,
      render: (startedBy: string) => (
        <div className='flex items-center'>
          <User className='w-4 h-4 mr-1' />
          <span>{startedBy}</span>
        </div>
      ),
    },
    {
      title: '启动时间',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 150,
      render: (date: string) => (
        <div className='text-sm'>{new Date(date).toLocaleString('zh-CN')}</div>
      ),
    },
    {
      title: '到期时间',
      dataIndex: 'due_date',
      key: 'due_date',
      width: 150,
      render: (date: string) => (
        <div className='text-sm'>{date ? new Date(date).toLocaleString('zh-CN') : '-'}</div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (record: WorkflowInstanceRecord) => (
        <Space>
          <Tooltip title='查看详情'>
            <Button
              type='text'
              icon={<Eye className='w-4 h-4' />}
              onClick={() => handleViewDetail(record)}
            />
          </Tooltip>
          {record.status === 'running' && (
            <>
              <Tooltip title='暂停'>
                <Button
                  type='text'
                  icon={<PauseCircle className='w-4 h-4' />}
                  onClick={() => handleSuspendInstance(record.instance_id)}
                />
              </Tooltip>
              <Tooltip title='终止'>
                <Button
                  type='text'
                  danger
                  icon={<StopCircle className='w-4 h-4' />}
                  onClick={() => handleTerminateInstance(record.instance_id)}
                />
              </Tooltip>
            </>
          )}
          {record.status === 'suspended' && (
            <Tooltip title='恢复'>
              <Button
                type='text'
                icon={<PlayCircle className='w-4 h-4' />}
                onClick={() => handleResumeInstance(record.instance_id)}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  const taskColumns = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: WorkflowTaskRecord) => (
        <div>
          <div className='font-medium'>{name}</div>
          <div className='text-sm text-gray-500'>{record.activity_id}</div>
        </div>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => <Tag color='blue'>{type}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>,
    },
    {
      title: '处理人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 120,
      render: (assignee: string) => <span>{assignee || '-'}</span>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (date: string) => (
        <div className='text-sm'>{new Date(date).toLocaleString('zh-CN')}</div>
      ),
    },
    {
      title: '到期时间',
      dataIndex: 'due_date',
      key: 'due_date',
      width: 150,
      render: (date: string) => (
        <div className='text-sm'>{date ? new Date(date).toLocaleString('zh-CN') : '-'}</div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: any, record: WorkflowTaskRecord) => {
        const csStatus = counterSignStatus[record.activity_id || record.id];

        // 如果是会签任务，显示投票按钮
        if (record.activity_id && record.status === 'pending') {
          return (
            <Space direction='vertical' size='small'>
              {/* 会签状态显示 */}
              {csStatus && (
                <div className='text-xs'>
                  <Tag
                    color={
                      csStatus.status === 'approved'
                        ? 'green'
                        : csStatus.status === 'rejected'
                          ? 'red'
                          : 'blue'
                    }
                  >
                    {csStatus.status === 'approved'
                      ? '已通过'
                      : csStatus.status === 'rejected'
                        ? '已拒绝'
                        : '会签中'}
                  </Tag>
                  <span className='ml-1'>
                    {csStatus.completed}/{csStatus.total}
                  </span>
                </div>
              )}
              {/* 投票按钮 */}
              <Space>
                <Button
                  type='primary'
                  size='small'
                  icon={<ThumbsUp className='w-3 h-3' />}
                  onClick={() => handleVote(record.id, true)}
                  loading={votingTaskId === record.id}
                  disabled={csStatus?.status !== 'pending'}
                >
                  通过
                </Button>
                <Button
                  size='small'
                  danger
                  icon={<ThumbsDown className='w-3 h-3' />}
                  onClick={() => handleVote(record.id, false)}
                  loading={votingTaskId === record.id}
                  disabled={csStatus?.status !== 'pending'}
                >
                  拒绝
                </Button>
              </Space>
            </Space>
          );
        }

        // 普通任务，显示完成按钮
        return (
          <Space>
            {record.status === 'pending' && (
              <Button
                type='link'
                size='small'
                onClick={() => {
                  // TODO: 完成任务的处理
                  message.info('完成任务功能开发中');
                }}
              >
                处理
              </Button>
            )}
          </Space>
        );
      },
    },
  ];

  return (
    <>
      {/* 页面头部 */}
      <div className='mb-6'>
        <h1 className='text-2xl font-bold text-gray-900'>工作流实例</h1>
        <p className='text-gray-600 mt-1'>管理工作流实例的执行状态和生命周期</p>
      </div>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='总实例'
              value={stats.total}
              prefix={<Activity className='w-5 h-5' />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='运行中'
              value={stats.running}
              prefix={<PlayCircle className='w-5 h-5' />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='已完成'
              value={stats.completed}
              prefix={<CheckCircle className='w-5 h-5' />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='已暂停'
              value={stats.suspended}
              prefix={<PauseCircle className='w-5 h-5' />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className='enterprise-card mb-6'>
        <Row gutter={[16, 16]} align='middle'>
          <Col xs={24} sm={12} md={6}>
            <Input
              placeholder='搜索实例ID或业务键...'
              prefix={<Search className='w-4 h-4' />}
              value={filters.business_key}
              onChange={e =>
                setFilters(prev => ({
                  ...prev,
                  business_key: e.target.value,
                }))
              }
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder='状态筛选'
              value={filters.status}
              onChange={value => setFilters(prev => ({ ...prev, status: value }))}
              allowClear
              style={{ width: '100%' }}
            >
              <Option value='running'>运行中</Option>
              <Option value='completed'>已完成</Option>
              <Option value='suspended'>已暂停</Option>
              <Option value='terminated'>已终止</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder='启动人筛选'
              value={filters.started_by}
              onChange={value => setFilters(prev => ({ ...prev, started_by: value }))}
              allowClear
              style={{ width: '100%' }}
            >
              <Option value='admin'>管理员</Option>
              <Option value='user1'>用户1</Option>
              <Option value='user2'>用户2</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={10}>
            <Space>
              <Button
                icon={<RefreshCw className='w-4 h-4' />}
                onClick={() => loadInstances(pagination.current, pagination.pageSize)}
              >
                刷新
              </Button>
              <Button icon={<BarChart3 className='w-4 h-4' />}>统计报告</Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 实例表格 */}
      <Card className='enterprise-card'>
        <Table
          columns={columns}
          dataSource={instances}
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
        />
      </Card>

      {/* 实例详情模态框 */}
      <Modal
        title={`实例详情 - ${selectedInstance?.instance_id}`}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={800}
        destroyOnHidden
      >
        {selectedInstance && (
          <div className='space-y-6'>
            {/* 基本信息 */}
            <Card title='基本信息' size='small'>
              <Descriptions column={2}>
                <Descriptions.Item label='实例ID'>{selectedInstance.instance_id}</Descriptions.Item>
                <Descriptions.Item label='业务键'>
                  {selectedInstance.business_key || '-'}
                </Descriptions.Item>
                <Descriptions.Item label='状态'>
                  <Tag color={getStatusColor(selectedInstance.status)}>
                    {getStatusText(selectedInstance.status)}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label='优先级'>
                  <Tag color={getPriorityColor(selectedInstance.priority ?? 'medium')}>
                    {selectedInstance.priority || '-'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label='启动人'>
                  {selectedInstance.started_by || '-'}
                </Descriptions.Item>
                <Descriptions.Item label='启动时间'>
                  {selectedInstance.started_at
                    ? new Date(selectedInstance.started_at).toLocaleString('zh-CN')
                    : '-'}
                </Descriptions.Item>
                {selectedInstance.completed_at && (
                  <Descriptions.Item label='完成时间'>
                    {new Date(selectedInstance.completed_at).toLocaleString('zh-CN')}
                  </Descriptions.Item>
                )}
                {selectedInstance.due_date && (
                  <Descriptions.Item label='到期时间'>
                    {new Date(selectedInstance.due_date).toLocaleString('zh-CN')}
                  </Descriptions.Item>
                )}
              </Descriptions>
            </Card>

            {/* 任务列表 */}
            <Card title='任务列表' size='small'>
              <Table
                columns={taskColumns}
                dataSource={tasks}
                rowKey='id'
                pagination={false}
                size='small'
              />
            </Card>

            {/* 流程变量 */}
            {selectedInstance.variables && Object.keys(selectedInstance.variables).length > 0 && (
              <Card title='流程变量' size='small'>
                <pre className='bg-gray-100 p-4 rounded text-sm overflow-auto'>
                  {JSON.stringify(selectedInstance.variables, null, 2)}
                </pre>
              </Card>
            )}
          </div>
        )}
      </Modal>
    </>
  );
};

export default WorkflowInstancesPage;
