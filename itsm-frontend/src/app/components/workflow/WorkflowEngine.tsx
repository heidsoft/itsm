'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Button,
  Table,
  Modal,
  Form,
  Input,
  Select,
  Space,
  Tag,
  message,
  Popconfirm,
  Tooltip,
  Drawer,
  Typography,
  Row,
  Col,
  Timeline,
  Badge,
  Progress,
  Alert,
  Divider,
  Switch,
} from 'antd';
import {
  Plus,
  Edit,
  Delete,
  Play,
  Pause,
  Square,
  Eye,
  Settings,
  Workflow,
  CheckCircle,
  Clock,
  AlertTriangle,
  ArrowRight,
  Users,
  FileText,
  GitBranch,
  GitCommit,
  GitPullRequest,
} from 'lucide-react';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { LoadingSkeleton } from '@/components/ui/LoadingSkeleton';

const { TextArea } = Input;
const { Title, Text } = Typography;
const { Option } = Select;

interface WorkflowStep {
  id: string;
  name: string;
  type: 'start' | 'task' | 'approval' | 'condition' | 'end';
  description: string;
  assignee?: string;
  assigneeType: 'user' | 'role' | 'group' | 'auto';
  conditions?: WorkflowCondition[];
  actions?: WorkflowAction[];
  order: number;
  estimatedTime?: number; // 分钟
  sla?: number; // 分钟
  isRequired: boolean;
  canSkip: boolean;
  nextSteps: string[]; // 下一步骤ID数组
}

interface WorkflowCondition {
  id: string;
  field: string;
  operator: 'equals' | 'not_equals' | 'contains' | 'greater_than' | 'less_than';
  value: any;
  logic: 'and' | 'or';
}

interface WorkflowAction {
  id: string;
  type: 'update_field' | 'send_notification' | 'create_task' | 'webhook';
  target: string;
  value: any;
  description: string;
}

interface Workflow {
  id: string;
  name: string;
  description: string;
  category: string;
  version: string;
  isActive: boolean;
  isDefault: boolean;
  steps: WorkflowStep[];
  triggers: string[];
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

interface WorkflowInstance {
  id: string;
  workflowId: string;
  workflowName: string;
  ticketId: string;
  currentStep: string;
  status: 'running' | 'paused' | 'completed' | 'cancelled' | 'error';
  startedAt: string;
  completedAt?: string;
  currentAssignee?: string;
  progress: number;
  estimatedCompletion?: string;
}

interface WorkflowEngineProps {
  mode?: 'design' | 'monitor' | 'manage';
}

export const WorkflowEngine: React.FC<WorkflowEngineProps> = ({ mode = 'manage' }) => {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [instances, setInstances] = useState<WorkflowInstance[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [editingWorkflow, setEditingWorkflow] = useState<Workflow | null>(null);
  const [form] = Form.useForm();

  // 模拟加载工作流数据
  useEffect(() => {
    loadWorkflows();
    if (mode === 'monitor') {
      loadInstances();
    }
  }, [mode]);

  const loadWorkflows = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));

      // 模拟工作流数据
      const mockWorkflows: Workflow[] = [
        {
          id: 'wf-001',
          name: '标准工单处理流程',
          description: '处理一般性工单的标准流程',
          category: '工单处理',
          version: '1.0',
          isActive: true,
          isDefault: true,
          steps: [
            {
              id: 'step-1',
              name: '工单接收',
              type: 'start',
              description: '系统自动接收工单',
              assigneeType: 'auto',
              order: 1,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-2'],
            },
            {
              id: 'step-2',
              name: '初步分析',
              type: 'task',
              description: '分析工单内容，确定处理方案',
              assignee: '张三',
              assigneeType: 'user',
              order: 2,
              estimatedTime: 30,
              sla: 60,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-3'],
            },
            {
              id: 'step-3',
              name: '问题分类',
              type: 'condition',
              description: '根据问题类型决定处理路径',
              assigneeType: 'auto',
              order: 3,
              conditions: [
                {
                  id: 'cond-1',
                  field: 'priority',
                  operator: 'equals',
                  value: 'high',
                  logic: 'and',
                },
              ],
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-4', 'step-5'],
            },
            {
              id: 'step-4',
              name: '紧急处理',
              type: 'task',
              description: '高优先级问题的紧急处理',
              assignee: '李四',
              assigneeType: 'user',
              order: 4,
              estimatedTime: 120,
              sla: 240,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-6'],
            },
            {
              id: 'step-5',
              name: '常规处理',
              type: 'task',
              description: '常规问题的标准处理',
              assignee: '王五',
              assigneeType: 'user',
              order: 5,
              estimatedTime: 60,
              sla: 120,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-6'],
            },
            {
              id: 'step-6',
              name: '质量检查',
              type: 'approval',
              description: '检查处理结果质量',
              assignee: '质量管理员',
              assigneeType: 'role',
              order: 6,
              estimatedTime: 15,
              sla: 30,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-7'],
            },
            {
              id: 'step-7',
              name: '工单关闭',
              type: 'end',
              description: '完成工单处理',
              assigneeType: 'auto',
              order: 7,
              isRequired: true,
              canSkip: false,
              nextSteps: [],
            },
          ],
          triggers: ['工单创建'],
          createdBy: '系统管理员',
          createdAt: '2024-01-01 00:00:00',
          updatedAt: '2024-01-15 10:00:00',
        },
        {
          id: 'wf-002',
          name: '变更审批流程',
          description: '处理系统变更的审批流程',
          category: '变更管理',
          version: '1.0',
          isActive: true,
          isDefault: false,
          steps: [
            {
              id: 'step-1',
              name: '变更申请',
              type: 'start',
              description: '提交变更申请',
              assigneeType: 'auto',
              order: 1,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-2'],
            },
            {
              id: 'step-2',
              name: '技术评估',
              type: 'approval',
              description: '技术团队评估变更可行性',
              assignee: '技术专家',
              assigneeType: 'role',
              order: 2,
              estimatedTime: 60,
              sla: 120,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-3'],
            },
            {
              id: 'step-3',
              name: '风险评估',
              type: 'approval',
              description: '风险评估团队评估变更风险',
              assignee: '风险管理员',
              assigneeType: 'role',
              order: 3,
              estimatedTime: 30,
              sla: 60,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-4'],
            },
            {
              id: 'step-4',
              name: '最终审批',
              type: 'approval',
              description: '管理层最终审批',
              assignee: '部门经理',
              assigneeType: 'role',
              order: 4,
              estimatedTime: 15,
              sla: 30,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-5'],
            },
            {
              id: 'step-5',
              name: '变更实施',
              type: 'task',
              description: '实施变更',
              assignee: '实施团队',
              assigneeType: 'group',
              order: 5,
              estimatedTime: 240,
              sla: 480,
              isRequired: true,
              canSkip: false,
              nextSteps: ['step-6'],
            },
            {
              id: 'step-6',
              name: '变更完成',
              type: 'end',
              description: '变更实施完成',
              assigneeType: 'auto',
              order: 6,
              isRequired: true,
              canSkip: false,
              nextSteps: [],
            },
          ],
          triggers: ['变更申请'],
          createdBy: '系统管理员',
          createdAt: '2024-01-01 00:00:00',
          updatedAt: '2024-01-15 10:00:00',
        },
      ];

      setWorkflows(mockWorkflows);
    } catch (error) {
      message.error('加载工作流失败');
    } finally {
      setLoading(false);
    }
  };

  const loadInstances = async () => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));

      // 模拟工作流实例数据
      const mockInstances: WorkflowInstance[] = [
        {
          id: 'inst-001',
          workflowId: 'wf-001',
          workflowName: '标准工单处理流程',
          ticketId: 'T-2024-001',
          currentStep: 'step-2',
          status: 'running',
          startedAt: '2024-01-15 09:00:00',
          currentAssignee: '张三',
          progress: 28,
          estimatedCompletion: '2024-01-15 16:00:00',
        },
        {
          id: 'inst-002',
          workflowId: 'wf-002',
          workflowName: '变更审批流程',
          ticketId: 'C-2024-001',
          currentStep: 'step-3',
          status: 'running',
          startedAt: '2024-01-15 08:00:00',
          currentAssignee: '风险管理员',
          progress: 50,
          estimatedCompletion: '2024-01-15 14:00:00',
        },
      ];

      setInstances(mockInstances);
    } catch (error) {
      message.error('加载实例失败');
    }
  };

  const handleCreateWorkflow = () => {
    setEditingWorkflow(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEditWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);
    form.setFieldsValue(workflow);
    setModalVisible(true);
  };

  const handleDeleteWorkflow = async (id: string) => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));

      setWorkflows(prev => prev.filter(w => w.id !== id));
      message.success('工作流删除成功');
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleSaveWorkflow = async () => {
    try {
      const values = await form.validateFields();

      if (editingWorkflow) {
        // 更新工作流
        const updatedWorkflow = {
          ...editingWorkflow,
          ...values,
          updatedAt: new Date().toLocaleString(),
        };

        setWorkflows(prev => prev.map(w => (w.id === editingWorkflow.id ? updatedWorkflow : w)));
        message.success('工作流更新成功');
      } else {
        // 创建新工作流
        const newWorkflow: Workflow = {
          id: `wf-${Date.now()}`,
          ...values,
          version: '1.0',
          isActive: true,
          isDefault: false,
          steps: [],
          triggers: values.triggers || [],
          createdBy: '当前用户',
          createdAt: new Date().toLocaleString(),
          updatedAt: new Date().toLocaleString(),
        };

        setWorkflows(prev => [newWorkflow, ...prev]);
        message.success('工作流创建成功');
      }

      setModalVisible(false);
      form.resetFields();
    } catch (error) {
      console.error('保存工作流失败:', error);
    }
  };

  const getStepTypeIcon = (type: string) => {
    switch (type) {
      case 'start':
        return <Play size={16} />;
      case 'task':
        return <FileText size={16} />;
      case 'approval':
        return <CheckCircle size={16} />;
      case 'condition':
        return <GitBranch size={16} />;
      case 'end':
        return <Square size={16} />;
      default:
        return <Settings size={16} />;
    }
  };

  const getStepTypeColor = (type: string) => {
    switch (type) {
      case 'start':
        return 'green';
      case 'task':
        return 'blue';
      case 'approval':
        return 'orange';
      case 'condition':
        return 'purple';
      case 'end':
        return 'red';
      default:
        return 'default';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'green';
      case 'paused':
        return 'orange';
      case 'completed':
        return 'blue';
      case 'cancelled':
        return 'red';
      case 'error':
        return 'red';
      default:
        return 'default';
    }
  };

  const workflowColumns = [
    {
      title: '工作流名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Workflow) => (
        <div>
          <div className='font-medium'>{text}</div>
          <div className='text-sm text-gray-500'>{record.description}</div>
        </div>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      render: (category: string) => <Tag color='blue'>{category}</Tag>,
    },
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
      render: (version: string) => <Tag color='cyan'>v{version}</Tag>,
    },
    {
      title: '步骤数',
      key: 'stepsCount',
      render: (record: Workflow) => (
        <div className='flex items-center gap-2'>
          <Workflow size={16} className='text-gray-400' />
          <span>{record.steps.length}</span>
        </div>
      ),
    },
    {
      title: '状态',
      key: 'status',
      render: (record: Workflow) => (
        <div className='flex items-center gap-2'>
          <Switch
            checked={record.isActive}
            size='small'
            onChange={checked => {
              setWorkflows(prev =>
                prev.map(w => (w.id === record.id ? { ...w, isActive: checked } : w))
              );
            }}
          />
          {record.isDefault && <Tag color='green'>默认</Tag>}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (record: Workflow) => (
        <Space>
          <Tooltip title='查看详情'>
            <Button
              size='small'
              icon={<Eye size={14} />}
              onClick={() => {
                setEditingWorkflow(record);
                setDrawerVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title='编辑'>
            <Button
              size='small'
              icon={<Edit size={14} />}
              onClick={() => handleEditWorkflow(record)}
            />
          </Tooltip>
          <Popconfirm
            title='确定要删除这个工作流吗？'
            onConfirm={() => handleDeleteWorkflow(record.id)}
            okText='确定'
            cancelText='取消'
          >
            <Button size='small' danger icon={<Delete size={14} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const instanceColumns = [
    {
      title: '工作流',
      dataIndex: 'workflowName',
      key: 'workflowName',
      render: (text: string, record: WorkflowInstance) => (
        <div>
          <div className='font-medium'>{text}</div>
          <div className='text-sm text-gray-500'>工单: {record.ticketId}</div>
        </div>
      ),
    },
    {
      title: '当前步骤',
      key: 'currentStep',
      render: (record: WorkflowInstance) => (
        <div className='flex items-center gap-2'>
          <Badge status='processing' />
          <span className='text-sm'>{record.currentStep}</span>
        </div>
      ),
    },
    {
      title: '处理人',
      dataIndex: 'currentAssignee',
      key: 'currentAssignee',
      render: (assignee: string) => (
        <div className='flex items-center gap-2'>
          <Users size={14} className='text-gray-400' />
          <span>{assignee || '未分配'}</span>
        </div>
      ),
    },
    {
      title: '进度',
      key: 'progress',
      render: (record: WorkflowInstance) => (
        <div className='w-32'>
          <Progress
            percent={record.progress}
            size='small'
            status={record.status === 'error' ? 'exception' : 'normal'}
          />
        </div>
      ),
    },
    {
      title: '状态',
      key: 'status',
      render: (record: WorkflowInstance) => (
        <Tag color={getStatusColor(record.status)}>
          {record.status === 'running' && '运行中'}
          {record.status === 'paused' && '已暂停'}
          {record.status === 'completed' && '已完成'}
          {record.status === 'cancelled' && '已取消'}
          {record.status === 'error' && '错误'}
        </Tag>
      ),
    },
    {
      title: '预计完成',
      key: 'estimatedCompletion',
      render: (record: WorkflowInstance) => (
        <div className='text-sm text-gray-500'>{record.estimatedCompletion}</div>
      ),
    },
  ];

  if (loading) {
    return <LoadingSkeleton type='table' rows={5} columns={6} />;
  }

  if (workflows.length === 0) {
    return (
      <LoadingEmptyError
        state='empty'
        empty={{
          title: '暂无工作流',
          description: '创建第一个工作流来标准化业务流程',
          actionText: '创建工作流',
          onAction: handleCreateWorkflow,
        }}
      />
    );
  }

  return (
    <div className='space-y-6'>
      {/* 头部操作区 */}
      <Card>
        <div className='flex justify-between items-center'>
          <div>
            <Title level={4} className='mb-1'>
              工作流引擎
            </Title>
            <Text type='secondary'>
              {mode === 'design' && '设计和配置业务流程工作流'}
              {mode === 'monitor' && '监控工作流执行状态和进度'}
              {mode === 'manage' && '管理工作流定义和配置'}
            </Text>
          </div>
          {mode === 'manage' && (
            <Button type='primary' icon={<Plus size={16} />} onClick={handleCreateWorkflow}>
              创建工作流
            </Button>
          )}
        </div>
      </Card>

      {/* 工作流列表 */}
      {mode === 'manage' && (
        <Card>
          <Table
            columns={workflowColumns}
            dataSource={workflows}
            rowKey='id'
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            }}
          />
        </Card>
      )}

      {/* 工作流实例监控 */}
      {mode === 'monitor' && (
        <Card>
          <div className='mb-4'>
            <Title level={5}>工作流实例监控</Title>
            <Text type='secondary'>实时监控工作流执行状态和进度</Text>
          </div>
          <Table
            columns={instanceColumns}
            dataSource={instances}
            rowKey='id'
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showQuickJumper: true,
            }}
          />
        </Card>
      )}

      {/* 创建/编辑工作流模态框 */}
      <Modal
        title={editingWorkflow ? '编辑工作流' : '创建工作流'}
        open={modalVisible}
        onOk={handleSaveWorkflow}
        onCancel={() => setModalVisible(false)}
        okText='保存'
        cancelText='取消'
        width={800}
      >
        <Form form={form} layout='vertical'>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='name'
                label='工作流名称'
                rules={[{ required: true, message: '请输入工作流名称' }]}
              >
                <Input placeholder='请输入工作流名称' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='category'
                label='分类'
                rules={[{ required: true, message: '请选择分类' }]}
              >
                <Select placeholder='请选择分类'>
                  <Option value='工单处理'>工单处理</Option>
                  <Option value='变更管理'>变更管理</Option>
                  <Option value='问题管理'>问题管理</Option>
                  <Option value='发布管理'>发布管理</Option>
                  <Option value='其他'>其他</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name='description' label='描述'>
            <TextArea rows={3} placeholder='请输入工作流描述' />
          </Form.Item>

          <Form.Item name='triggers' label='触发条件'>
            <Select mode='tags' placeholder='请输入触发条件'>
              <Option value='工单创建'>工单创建</Option>
              <Option value='工单状态变更'>工单状态变更</Option>
              <Option value='变更申请'>变更申请</Option>
              <Option value='问题升级'>问题升级</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 工作流详情抽屉 */}
      <Drawer
        title='工作流详情'
        placement='right'
        width={800}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        {editingWorkflow && (
          <div className='space-y-6'>
            <div>
              <Title level={5}>基本信息</Title>
              <div className='bg-gray-50 p-4 rounded-lg space-y-2'>
                <div className='flex justify-between'>
                  <span className='text-gray-600'>工作流名称:</span>
                  <span className='font-medium'>{editingWorkflow.name}</span>
                </div>
                <div className='flex justify-between'>
                  <span className='text-gray-600'>分类:</span>
                  <span>{editingWorkflow.category}</span>
                </div>
                <div className='flex justify-between'>
                  <span className='text-gray-600'>版本:</span>
                  <span>v{editingWorkflow.version}</span>
                </div>
                <div className='flex justify-between'>
                  <span className='text-gray-600'>状态:</span>
                  <Tag color={editingWorkflow.isActive ? 'green' : 'red'}>
                    {editingWorkflow.isActive ? '启用' : '禁用'}
                  </Tag>
                </div>
              </div>
            </div>

            <div>
              <Title level={5}>工作流步骤</Title>
              <Timeline>
                {editingWorkflow.steps.map((step, index) => (
                  <Timeline.Item
                    key={step.id}
                    color={getStepTypeColor(step.type)}
                    dot={getStepTypeIcon(step.type)}
                  >
                    <div className='mb-2'>
                      <div className='flex items-center gap-2 mb-1'>
                        <span className='font-medium'>{step.name}</span>
                        <Tag color={getStepTypeColor(step.type)}>
                          {step.type === 'start' && '开始'}
                          {step.type === 'task' && '任务'}
                          {step.type === 'approval' && '审批'}
                          {step.type === 'condition' && '条件'}
                          {step.type === 'end' && '结束'}
                        </Tag>
                        {step.isRequired && <Tag color='red'>必填</Tag>}
                        {step.canSkip && <Tag color='orange'>可跳过</Tag>}
                      </div>
                      <div className='text-sm text-gray-600 mb-2'>{step.description}</div>

                      {step.assignee && (
                        <div className='text-sm text-gray-500 mb-1'>
                          处理人: {step.assignee} ({step.assigneeType})
                        </div>
                      )}

                      {step.estimatedTime && (
                        <div className='text-sm text-gray-500 mb-1'>
                          预计时间: {step.estimatedTime}分钟
                        </div>
                      )}

                      {step.sla && (
                        <div className='text-sm text-gray-500 mb-1'>SLA: {step.sla}分钟</div>
                      )}

                      {step.nextSteps.length > 0 && (
                        <div className='text-sm text-gray-500'>
                          下一步: {step.nextSteps.join(', ')}
                        </div>
                      )}
                    </div>
                  </Timeline.Item>
                ))}
              </Timeline>
            </div>

            <div>
              <Title level={5}>其他信息</Title>
              <div className='bg-gray-50 p-4 rounded-lg space-y-2'>
                <div className='flex justify-between'>
                  <span className='text-gray-600'>触发条件:</span>
                  <span>{editingWorkflow.triggers.join(', ') || '无'}</span>
                </div>
                <div className='flex justify-between'>
                  <span className='text-gray-600'>创建人:</span>
                  <span>{editingWorkflow.createdBy}</span>
                </div>
                <div className='flex justify-between'>
                  <span className='text-gray-600'>创建时间:</span>
                  <span>{editingWorkflow.createdAt}</span>
                </div>
                <div className='flex justify-between'>
                  <span className='text-gray-600'>更新时间:</span>
                  <span>{editingWorkflow.updatedAt}</span>
                </div>
              </div>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default WorkflowEngine;
