'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Space,
  Typography,
  Tag,
  Badge,
  Timeline,
  Statistic,
  Row,
  Col,
  Tooltip,
  Popconfirm,
  message,
  App,
  Divider,
  Radio,
  InputNumber,
  Tabs,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  UserOutlined,
  SettingOutlined,
  HistoryOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { Ticket } from '@/lib/services/ticket-service';
import { TicketApprovalApi } from '@/lib/api/ticket-approval-api';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

interface ApprovalNode {
  id: string;
  level: number;
  name: string;
  approverType: 'user' | 'role' | 'department' | 'dynamic';
  approverIds: number[];
  approverNames: string[];
  approvalMode: 'sequential' | 'parallel' | 'any' | 'all';
  minimumApprovals?: number;
  timeoutHours?: number;
  conditions?: Array<{
    field: string;
    operator: 'equals' | 'not_equals' | 'greater_than' | 'less_than' | 'contains';
    value: any;
  }>;
  allowReject: boolean;
  allowDelegate: boolean;
  rejectAction: 'end' | 'return' | 'custom';
  returnToLevel?: number;
}

interface ApprovalWorkflow {
  id: number;
  name: string;
  description: string;
  ticketType?: string;
  priority?: string;
  nodes: ApprovalNode[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface ApprovalRecord {
  id: number;
  ticketId: number;
  ticketNumber: string;
  ticketTitle: string;
  workflowId: number;
  workflowName: string;
  currentLevel: number;
  totalLevels: number;
  approverId: number;
  approverName: string;
  status: 'pending' | 'approved' | 'rejected' | 'delegated' | 'timeout';
  action?: string;
  comment?: string;
  createdAt: string;
  processedAt?: string;
}

interface TicketMultiLevelApprovalProps {
  ticket?: Ticket;
  workflowId?: number;
  onWorkflowChange?: (workflowId: number) => void;
  canManage?: boolean;
}

export const TicketMultiLevelApproval: React.FC<TicketMultiLevelApprovalProps> = ({
  ticket,
  workflowId,
  onWorkflowChange,
  canManage = true,
}) => {
  const { message: antMessage } = App.useApp();
  const [workflows, setWorkflows] = useState<ApprovalWorkflow[]>([]);
  const [currentWorkflow, setCurrentWorkflow] = useState<ApprovalWorkflow | null>(null);
  const [approvalRecords, setApprovalRecords] = useState<ApprovalRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [workflowModalVisible, setWorkflowModalVisible] = useState(false);
  const [editingWorkflow, setEditingWorkflow] = useState<ApprovalWorkflow | null>(null);
  const [nodeModalVisible, setNodeModalVisible] = useState(false);
  const [editingNode, setEditingNode] = useState<ApprovalNode | null>(null);
  const [activeTab, setActiveTab] = useState<'workflow' | 'history' | 'stats'>('workflow');
  const [form] = Form.useForm();
  const [nodeForm] = Form.useForm();

  // 加载审批工作流
  const loadWorkflows = useCallback(async () => {
    try {
      setLoading(true);
      // 调用实际API
      const data = await TicketApprovalApi.getWorkflows({
        page: 1,
        pageSize: 100,
      });

      if (data && data.items && data.items.length > 0) {
        setWorkflows(data.items);
        if (workflowId) {
          const workflow = data.items.find(w => w.id === workflowId);
          if (workflow) {
            setCurrentWorkflow(workflow);
          }
        }
        return;
      }

      // 如果API返回空，使用模拟数据
      const mockWorkflows: ApprovalWorkflow[] = [
        {
          id: 1,
          name: '标准工单审批流程',
          description: '适用于一般工单的标准审批流程',
          ticketType: 'service_request',
          nodes: [
            {
              id: 'node1',
              level: 1,
              name: '直属主管审批',
              approverType: 'role',
              approverIds: [1],
              approverNames: ['直属主管'],
              approvalMode: 'any',
              timeoutHours: 24,
              allowReject: true,
              allowDelegate: true,
              rejectAction: 'end',
            },
            {
              id: 'node2',
              level: 2,
              name: '部门经理审批',
              approverType: 'role',
              approverIds: [2],
              approverNames: ['部门经理'],
              approvalMode: 'any',
              timeoutHours: 48,
              allowReject: true,
              allowDelegate: false,
              rejectAction: 'return',
              returnToLevel: 1,
            },
          ],
          isActive: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ];
      setWorkflows(mockWorkflows);

      if (workflowId) {
        const workflow = mockWorkflows.find(w => w.id === workflowId);
        if (workflow) {
          setCurrentWorkflow(workflow);
        }
      }
    } catch (error) {
      console.error('Failed to load workflows:', error);
      antMessage.error('加载审批工作流失败');
    } finally {
      setLoading(false);
    }
  }, [workflowId, antMessage]);

  // 加载审批记录
  const loadApprovalRecords = useCallback(async () => {
    if (!ticket?.id) return;

    try {
      // 调用实际API
      const data = await TicketApprovalApi.getApprovalRecords({
        ticketId: ticket.id,
        page: 1,
        pageSize: 100,
      });

      if (data && data.items && data.items.length > 0) {
        setApprovalRecords(data.items);
        return;
      }

      // 如果API返回空，使用模拟数据
      const mockRecords: ApprovalRecord[] = [
        {
          id: 1,
          ticketId: ticket.id,
          ticketNumber: ticket.ticketNumber || `T-${ticket.id}`,
          ticketTitle: ticket.title,
          workflowId: 1,
          workflowName: '标准工单审批流程',
          currentLevel: 1,
          totalLevels: 2,
          approverId: 1,
          approverName: '张三',
          status: 'pending',
          createdAt: new Date().toISOString(),
        },
      ];
      setApprovalRecords(mockRecords);
    } catch (error) {
      console.error('Failed to load approval records:', error);
    }
  }, [ticket]);

  useEffect(() => {
    loadWorkflows();
    if (ticket) {
      loadApprovalRecords();
    }
  }, [loadWorkflows, loadApprovalRecords, ticket]);

  // 保存工作流
  const handleSaveWorkflow = useCallback(async () => {
    try {
      const values = await form.validateFields();
      const workflowData: Partial<ApprovalWorkflow> = {
        ...values,
        nodes: currentWorkflow?.nodes || [],
      };

      if (editingWorkflow) {
        // 更新工作流
        await TicketApprovalApi.updateWorkflow(editingWorkflow.id, workflowData);
        antMessage.success('审批工作流已更新');
      } else {
        // 创建工作流
        await TicketApprovalApi.createWorkflow(workflowData);
        antMessage.success('审批工作流已创建');
      }

      setWorkflowModalVisible(false);
      setEditingWorkflow(null);
      form.resetFields();
      loadWorkflows();
    } catch (error) {
      console.error('Failed to save workflow:', error);
      antMessage.error('保存失败');
    }
  }, [form, editingWorkflow, currentWorkflow, antMessage, loadWorkflows]);

  // 保存审批节点
  const handleSaveNode = useCallback(async () => {
    try {
      const values = await nodeForm.validateFields();
      const nodeData: ApprovalNode = {
        id: editingNode?.id || `node_${Date.now()}`,
        ...values,
      };

      if (currentWorkflow) {
        const updatedNodes = editingNode
          ? currentWorkflow.nodes.map(n => (n.id === editingNode.id ? nodeData : n))
          : [...currentWorkflow.nodes, nodeData];

        setCurrentWorkflow({
          ...currentWorkflow,
          nodes: updatedNodes,
        });
      }

      setNodeModalVisible(false);
      setEditingNode(null);
      nodeForm.resetFields();
    } catch (error) {
      console.error('Failed to save node:', error);
      antMessage.error('保存节点失败');
    }
  }, [nodeForm, editingNode, currentWorkflow, antMessage]);

  // 删除节点
  const handleDeleteNode = useCallback(
    (nodeId: string) => {
      if (currentWorkflow) {
        setCurrentWorkflow({
          ...currentWorkflow,
          nodes: currentWorkflow.nodes.filter(n => n.id !== nodeId),
        });
      }
    },
    [currentWorkflow]
  );

  // 审批工作流表格列
  const workflowColumns: ColumnsType<ApprovalWorkflow> = [
    {
      title: '工作流名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '审批级别',
      key: 'levels',
      render: (_: any, record: ApprovalWorkflow) => (
        <Badge count={record.nodes.length} showZero>
          <Tag color='blue'>{record.nodes.length} 级</Tag>
        </Badge>
      ),
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (isActive: boolean) => (
        <Tag color={isActive ? 'green' : 'default'}>{isActive ? '启用' : '禁用'}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: ApprovalWorkflow) => (
        <Space>
          <Button
            type='link'
            size='small'
            icon={<EyeOutlined />}
            onClick={() => {
              setCurrentWorkflow(record);
              setActiveTab('workflow');
            }}
          >
            查看
          </Button>
          {canManage && (
            <>
              <Button
                type='link'
                size='small'
                icon={<EditOutlined />}
                onClick={() => {
                  setEditingWorkflow(record);
                  setCurrentWorkflow(record);
                  form.setFieldsValue(record);
                  setWorkflowModalVisible(true);
                }}
              >
                编辑
              </Button>
              <Popconfirm
                title='确定要删除这个工作流吗？'
                onConfirm={async () => {
                  // 注意：删除工作流API尚未实现
                  antMessage.success('工作流已删除（模拟）');
                  loadWorkflows();
                }}
              >
                <Button type='link' size='small' danger icon={<DeleteOutlined />}>
                  删除
                </Button>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  // 审批记录表格列
  const recordColumns: ColumnsType<ApprovalRecord> = [
    {
      title: '工单编号',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
      render: (text: string) => (
        <Text strong style={{ color: '#1890ff' }}>
          {text}
        </Text>
      ),
    },
    {
      title: '工单标题',
      dataIndex: 'ticketTitle',
      key: 'ticketTitle',
    },
    {
      title: '工作流',
      dataIndex: 'workflowName',
      key: 'workflowName',
    },
    {
      title: '审批级别',
      key: 'level',
      render: (_: any, record: ApprovalRecord) => (
        <Text>
          {record.currentLevel} / {record.totalLevels}
        </Text>
      ),
    },
    {
      title: '审批人',
      dataIndex: 'approverName',
      key: 'approverName',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const statusConfig: Record<string, { color: string; text: string; icon: React.ReactNode }> =
          {
            pending: { color: 'orange', text: '待审批', icon: <ClockCircleOutlined /> },
            approved: { color: 'green', text: '已批准', icon: <CheckCircleOutlined /> },
            rejected: { color: 'red', text: '已拒绝', icon: <CloseCircleOutlined /> },
            delegated: { color: 'blue', text: '已委派', icon: <UserOutlined /> },
            timeout: { color: 'default', text: '超时', icon: <ClockCircleOutlined /> },
          };
        const config = statusConfig[status] || { color: 'default', text: status, icon: null };
        return (
          <Tag color={config.color} icon={config.icon}>
            {config.text}
          </Tag>
        );
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (date: string) => format(new Date(date), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN }),
    },
  ];

  // 计算统计信息
  const stats = React.useMemo(() => {
    const total = approvalRecords.length;
    const pending = approvalRecords.filter(r => r.status === 'pending').length;
    const approved = approvalRecords.filter(r => r.status === 'approved').length;
    const rejected = approvalRecords.filter(r => r.status === 'rejected').length;
    const approvalRate = total > 0 ? (approved / total) * 100 : 0;
    return { total, pending, approved, rejected, approvalRate };
  }, [approvalRecords]);

  return (
    <div className='space-y-4'>
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={key => setActiveTab(key as 'workflow' | 'history' | 'stats')}
          type='card'
          size='large'
          items={[
            {
              key: 'workflow',
              label: (
                <span>
                  <SettingOutlined /> 审批流程
                </span>
              ),
              children: (
                <div className='space-y-4'>
                  {/* 工作流列表 */}
                  <div className='flex items-center justify-between mb-4'>
                    <Title level={5} style={{ margin: 0 }}>
                      审批工作流列表
                    </Title>
                    {canManage && (
                      <Button
                        type='primary'
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingWorkflow(null);
                          setCurrentWorkflow(null);
                          form.resetFields();
                          setWorkflowModalVisible(true);
                        }}
                      >
                        创建工作流
                      </Button>
                    )}
                  </div>
                  <Table
                    columns={workflowColumns}
                    dataSource={workflows}
                    rowKey='id'
                    loading={loading}
                    pagination={false}
                    onRow={record => ({
                      onClick: () => {
                        setCurrentWorkflow(record);
                        onWorkflowChange?.(record.id);
                      },
                    })}
                  />

                  {/* 当前工作流详情 */}
                  {currentWorkflow && (
                    <Card
                      title={
                        <div className='flex items-center justify-between'>
                          <span>工作流详情: {currentWorkflow.name}</span>
                          {canManage && (
                            <Button
                              type='primary'
                              size='small'
                              icon={<PlusOutlined />}
                              onClick={() => {
                                setEditingNode(null);
                                nodeForm.resetFields();
                                nodeForm.setFieldsValue({
                                  level: currentWorkflow.nodes.length + 1,
                                  approvalMode: 'any',
                                  allowReject: true,
                                  allowDelegate: false,
                                  rejectAction: 'end',
                                });
                                setNodeModalVisible(true);
                              }}
                            >
                              添加审批节点
                            </Button>
                          )}
                        </div>
                      }
                      className='mt-4'
                    >
                      <Timeline>
                        {currentWorkflow.nodes.map((node, index) => (
                          <Timeline.Item
                            key={node.id}
                            color={
                              index === 0
                                ? 'green'
                                : index === currentWorkflow.nodes.length - 1
                                  ? 'red'
                                  : 'blue'
                            }
                          >
                            <div className='flex items-start justify-between'>
                              <div className='flex-1'>
                                <div className='flex items-center gap-2 mb-2'>
                                  <Text strong>
                                    级别 {node.level}: {node.name}
                                  </Text>
                                  <Tag color='blue'>
                                    {node.approvalMode === 'sequential'
                                      ? '串行'
                                      : node.approvalMode === 'parallel'
                                        ? '并行'
                                        : node.approvalMode === 'any'
                                          ? '任一'
                                          : '全部'}
                                  </Tag>
                                  {node.timeoutHours && (
                                    <Tag color='orange'>超时: {node.timeoutHours}小时</Tag>
                                  )}
                                </div>
                                <div className='text-sm text-gray-600 space-y-1'>
                                  <div>
                                    审批人类型:{' '}
                                    <Tag>
                                      {node.approverType === 'user'
                                        ? '用户'
                                        : node.approverType === 'role'
                                          ? '角色'
                                          : node.approverType === 'department'
                                            ? '部门'
                                            : '动态'}
                                    </Tag>
                                  </div>
                                  <div>审批人: {node.approverNames.join(', ') || '未配置'}</div>
                                  {node.conditions && node.conditions.length > 0 && (
                                    <div>条件: {node.conditions.length} 个条件</div>
                                  )}
                                </div>
                              </div>
                              {canManage && (
                                <Space>
                                  <Button
                                    type='link'
                                    size='small'
                                    icon={<EditOutlined />}
                                    onClick={() => {
                                      setEditingNode(node);
                                      nodeForm.setFieldsValue(node);
                                      setNodeModalVisible(true);
                                    }}
                                  >
                                    编辑
                                  </Button>
                                  <Popconfirm
                                    title='确定要删除这个节点吗？'
                                    onConfirm={() => handleDeleteNode(node.id)}
                                  >
                                    <Button
                                      type='link'
                                      size='small'
                                      danger
                                      icon={<DeleteOutlined />}
                                    >
                                      删除
                                    </Button>
                                  </Popconfirm>
                                </Space>
                              )}
                            </div>
                          </Timeline.Item>
                        ))}
                      </Timeline>
                    </Card>
                  )}
                </div>
              ),
            },
            {
              key: 'history',
              label: (
                <span>
                  <HistoryOutlined /> 审批历史
                </span>
              ),
              children: (
                <Table
                  columns={recordColumns}
                  dataSource={approvalRecords}
                  rowKey='id'
                  pagination={{
                    pageSize: 10,
                    showSizeChanger: true,
                    showTotal: total => `共 ${total} 条记录`,
                  }}
                />
              ),
            },
            {
              key: 'stats',
              label: (
                <span>
                  <BarChartOutlined /> 审批统计
                </span>
              ),
              children: (
                <div className='space-y-4'>
                  <Row gutter={16}>
                    <Col xs={24} sm={12} md={6}>
                      <Card>
                        <Statistic
                          title='总审批数'
                          value={stats.total}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card>
                        <Statistic
                          title='待审批'
                          value={stats.pending}
                          styles={{ content: { color: '#faad14' } }}
                          prefix={<ClockCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card>
                        <Statistic
                          title='已批准'
                          value={stats.approved}
                          styles={{ content: { color: '#3f8600' } }}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card>
                        <Statistic
                          title='批准率'
                          value={stats.approvalRate}
                          precision={1}
                          suffix='%'
                          styles={{ content: { color: '#1890ff' } }}
                          prefix={<BarChartOutlined />}
                        />
                      </Card>
                    </Col>
                  </Row>
                </div>
              ),
            },
          ]}
        />
      </Card>

      {/* 工作流编辑模态框 */}
      <Modal
        title={editingWorkflow ? '编辑审批工作流' : '创建审批工作流'}
        open={workflowModalVisible}
        onOk={handleSaveWorkflow}
        onCancel={() => {
          setWorkflowModalVisible(false);
          setEditingWorkflow(null);
          form.resetFields();
        }}
        width={700}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label='工作流名称'
            rules={[{ required: true, message: '请输入工作流名称' }]}
          >
            <Input placeholder='例如：标准工单审批流程' />
          </Form.Item>
          <Form.Item name='description' label='描述'>
            <TextArea rows={3} placeholder='工作流描述（可选）' />
          </Form.Item>
          <Form.Item name='ticketType' label='适用工单类型'>
            <Select placeholder='请选择工单类型' allowClear>
              <Option value='incident'>事件</Option>
              <Option value='service_request'>服务请求</Option>
              <Option value='problem'>问题</Option>
              <Option value='change'>变更</Option>
            </Select>
          </Form.Item>
          <Form.Item name='priority' label='适用优先级'>
            <Select placeholder='请选择优先级' allowClear>
              <Option value='low'>低</Option>
              <Option value='medium'>中</Option>
              <Option value='high'>高</Option>
              <Option value='urgent'>紧急</Option>
            </Select>
          </Form.Item>
          <Form.Item name='isActive' label='启用状态' valuePropName='checked' initialValue={true}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 审批节点编辑模态框 */}
      <Modal
        title={editingNode ? '编辑审批节点' : '添加审批节点'}
        open={nodeModalVisible}
        onOk={handleSaveNode}
        onCancel={() => {
          setNodeModalVisible(false);
          setEditingNode(null);
          nodeForm.resetFields();
        }}
        width={800}
      >
        <Form form={nodeForm} layout='vertical'>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='level'
                label='审批级别'
                rules={[{ required: true, message: '请输入审批级别' }]}
              >
                <InputNumber min={1} max={10} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='name'
                label='节点名称'
                rules={[{ required: true, message: '请输入节点名称' }]}
              >
                <Input placeholder='例如：直属主管审批' />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name='approverType'
            label='审批人类型'
            rules={[{ required: true, message: '请选择审批人类型' }]}
          >
            <Radio.Group>
              <Radio value='user'>用户</Radio>
              <Radio value='role'>角色</Radio>
              <Radio value='department'>部门</Radio>
              <Radio value='dynamic'>动态</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item
            name='approvalMode'
            label='审批模式'
            rules={[{ required: true, message: '请选择审批模式' }]}
          >
            <Select>
              <Option value='sequential'>串行（按顺序审批）</Option>
              <Option value='parallel'>并行（同时审批）</Option>
              <Option value='any'>任一（任意一人通过即可）</Option>
              <Option value='all'>全部（所有人都需通过）</Option>
            </Select>
          </Form.Item>
          <Form.Item name='timeoutHours' label='超时时间（小时）'>
            <InputNumber min={1} max={720} style={{ width: '100%' }} />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='allowReject'
                label='允许拒绝'
                valuePropName='checked'
                initialValue={true}
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='allowDelegate'
                label='允许委派'
                valuePropName='checked'
                initialValue={false}
              >
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name='rejectAction'
            label='拒绝后的操作'
            rules={[{ required: true, message: '请选择拒绝后的操作' }]}
          >
            <Select>
              <Option value='end'>结束流程</Option>
              <Option value='return'>返回上一级</Option>
              <Option value='custom'>自定义</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
