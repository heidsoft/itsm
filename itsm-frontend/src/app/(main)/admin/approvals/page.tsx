'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Typography,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  message,
  Popconfirm,
  Tag,
  Row,
  Col,
  Statistic,
  InputNumber,
  Alert,
} from 'antd';
import {
  Plus,
  Edit,
  Trash2,
  GitMerge,
  RefreshCw,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { httpClient } from '@/lib/api/http-client';
import { WorkflowDefinitionApi } from '@/lib/api/workflow-definition-api';

const { Title, Text } = Typography;
const { TextArea } = Input;

interface ApprovalWorkflow {
  id: number;
  name: string;
  description?: string;
  ticketType?: string;
  priority?: string;
  isActive: boolean;
  workflowId?: string;
  nodes?: ApprovalNode[];
  createdAt?: string;
  updatedAt?: string;
}

interface ApprovalNode {
  level: number;
  name: string;
  approverType?: string;
  approverIds?: number[];
  assigneeType?: string;
  assigneeValue?: string;
  approvalMode?: string;
  timeoutHours?: number;
  allowReject?: boolean;
  allowDelegate?: boolean;
  rejectAction?: string;
}

export default function ApprovalManagement() {
  const [workflows, setWorkflows] = useState<ApprovalWorkflow[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState<ApprovalWorkflow | null>(null);
  const [form] = Form.useForm();
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });
  const [bpmnWorkflows, setBpmnWorkflows] = useState<{ id: string; name: string }[]>([]);

  // 加载审批工作流数据
  const loadWorkflows = useCallback(async () => {
    setFetching(true);
    try {
      const response = await httpClient.get<{ items: ApprovalWorkflow[]; total: number }>(
        '/api/v1/approval-workflows',
        { page: 1, pageSize: 100 }
      );
      const items = (response.items || []).map(normalizeWorkflow);
      setWorkflows(items);
      setStats({
        total: response.total || 0,
        active: items.filter(w => w.isActive).length,
        inactive: items.filter(w => !w.isActive).length,
      });
    } catch (error) {
      console.error('Failed to load workflows:', error);
      message.error('加载审批工作流失败');
    } finally {
      setFetching(false);
    }
  }, []);

  // 加载 BPMN 工作流定义列表
  const loadBpmnWorkflows = useCallback(async () => {
    try {
      const result = await WorkflowDefinitionApi.getWorkflows({ page: 1, pageSize: 100 });
      const list = (result.workflows || []).map(w => ({
        id: w.id,
        name: w.name,
      }));
      setBpmnWorkflows(list);
    } catch (error) {
      console.error('Failed to load BPMN workflows:', error);
      setBpmnWorkflows([]);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    loadWorkflows();
    loadBpmnWorkflows();
  }, [loadWorkflows, loadBpmnWorkflows]);

  // 处理保存
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      const data = {
        ...values,
        isActive: values.isActive ?? true,
        nodes: normalizeNodes(values.nodes || []),
      };

      if (selectedWorkflow) {
        // 更新
        await httpClient.put(`/api/v1/approval-workflows/${selectedWorkflow.id}`, data);
        message.success('审批工作流更新成功');
      } else {
        // 创建
        await httpClient.post('/api/v1/approval-workflows', data);
        message.success('审批工作流创建成功');
      }

      setShowModal(false);
      form.resetFields();
      setSelectedWorkflow(null);
      loadWorkflows();
    } catch (error) {
      console.error('Failed to save workflow:', error);
      message.error('保存审批工作流失败');
    } finally {
      setLoading(false);
    }
  };

  // 处理删除
  const handleDelete = async (id: number) => {
    try {
      await httpClient.delete(`/api/v1/approval-workflows/${id}`);
      message.success('审批工作流删除成功');
      loadWorkflows();
    } catch (error) {
      console.error('Failed to delete workflow:', error);
      message.error('删除审批工作流失败');
    }
  };

  // 处理编辑
  const handleEdit = (record: ApprovalWorkflow) => {
    setSelectedWorkflow(record);
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      ticketType: record.ticketType || record.ticketType,
      priority: record.priority,
      isActive: record.isActive ?? record.isActive,
      workflowId: record.workflowId || record.workflowId,
      nodes: record.nodes?.length ? record.nodes : [defaultApprovalNode()],
    });
    setShowModal(true);
  };

  // 处理切换状态
  const handleToggleStatus = async (record: ApprovalWorkflow) => {
    try {
      await httpClient.patch(`/api/v1/approval-workflows/${record.id}`, {
        isActive: !record.isActive,
      });
      message.success(`${record.isActive ? '停用' : '启用'}成功`);
      loadWorkflows();
    } catch (error) {
      console.error('Failed to toggle status:', error);
      message.error('状态切换失败');
    }
  };

  // 表格列定义
  const columns: ColumnsType<ApprovalWorkflow> = [
    {
      title: '工作流名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <Space>
          <GitMerge size={16} />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    {
      title: '工单类型',
      dataIndex:'ticketType',
      key:'ticketType',
      render: (text: string) => <Tag color="blue">{text || '-'}</Tag>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      render: (text: string) => <Tag color="purple">{text || '-'}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'status',
      render: (isActive: boolean) => (
        <Tag color={isActive ? 'success' : 'default'}>
          {isActive ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: 'BPMN工作流',
      dataIndex:'workflowId',
      key:'workflowId',
      render: (workflowId: string) => (
        workflowId ? (
          <Tag color="blue">已关联</Tag>
        ) : (
          <Tag>未关联</Tag>
        )
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_: unknown, record: ApprovalWorkflow) => (
        <Space size="small">
          <Button
            type="text"
            icon={<Edit size={16} />}
            onClick={() => handleEdit(record)}
          />
          <Button
            type="text"
            onClick={() => handleToggleStatus(record)}
          >
            {record.isActive ? '停用' : '启用'}
          </Button>
          <Popconfirm
            title="确认删除"
            description={`确定要删除工作流"${record.name}"吗？`}
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button type="text" danger icon={<Trash2 size={16} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      {/* 迁移提示 */}
      <Alert
        message="审批配置已迁移"
        description="审批流程配置已迁移到「工作流管理」页面。本页面仅作历史数据查询，新流程请在工作流管理中创建。"
        type="warning"
        showIcon
        closable
        className="mb-6"
        action={
          <Button size="small" type="primary" href="/workflow">
            前往工作流管理
          </Button>
        }
      />

      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <GitMerge className="mr-2" />
          审批管理
        </Title>
        <Text type="secondary">管理审批工作流和审批规则</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="工作流总数"
              value={stats.total}
              prefix={<GitMerge />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="启用"
              value={stats.active}
              prefix={<GitMerge />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="禁用"
              value={stats.inactive}
              prefix={<GitMerge />}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作栏 */}
      <Card className="mb-6">
        <Space>
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={() => {
              setSelectedWorkflow(null);
              form.setFieldsValue({ isActive: true, nodes: [defaultApprovalNode()] });
              setShowModal(true);
            }}
          >
            新建工作流
          </Button>
          <Button
            icon={<RefreshCw size={16} />}
            onClick={() => loadWorkflows()}
            loading={fetching}
          >
            刷新
          </Button>
        </Space>
      </Card>

      {/* 工作流列表 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={workflows}
          rowKey="id"
          loading={fetching}
          pagination={{
            total: workflows.length,
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `共 ${total} 条记录`,
          }}
          className="enterprise-table"
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={
          <span>
            <Edit className="w-4 h-4 mr-2" />
            {selectedWorkflow ? '编辑工作流' : '新建工作流'}
          </span>
        }
        open={showModal}
        onOk={handleSave}
        onCancel={() => {
          setShowModal(false);
          setSelectedWorkflow(null);
          form.resetFields();
        }}
        width={600}
        confirmLoading={loading}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label="工作流名称"
            name="name"
            rules={[{ required: true, message: '请输入工作流名称' }]}
          >
            <Input placeholder="请输入工作流名称" />
          </Form.Item>
          <Form.Item
            label="工单类型"
            name="ticket_type"
          >
            <Select
              placeholder="选择工单类型"
              allowClear
              options={[
                { label: '服务请求', value: 'service_request' },
                { label: '事件', value: 'incident' },
                { label: '问题', value: 'problem' },
                { label: '变更', value: 'change' },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="优先级"
            name="priority"
          >
            <Select
              placeholder="选择优先级"
              allowClear
              options={[
                { label: '紧急', value: 'urgent' },
                { label: '高', value: 'high' },
                { label: '中', value: 'medium' },
                { label: '低', value: 'low' },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="描述"
            name="description"
          >
            <TextArea rows={3} placeholder="请输入工作流描述" />
          </Form.Item>
          <Form.Item
            label="关联BPMN工作流"
            name="workflow_id"
            tooltip="选择关联的BPMN工作流，用于自动化流程编排"
          >
            <Select
              placeholder="选择BPMN工作流（可选）"
              allowClear
              showSearch
              optionFilterProp="children"
              options={bpmnWorkflows.map(w => ({
                label: w.name,
                value: w.id,
              }))}
            />
          </Form.Item>
          <Form.Item
            label="状态"
            name="is_active"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" defaultChecked />
          </Form.Item>
          <Form.List name="nodes" rules={[{ validator: async (_, nodes) => {
            if (!nodes || nodes.length < 1) {
              return Promise.reject(new Error('至少需要一个审批节点'));
            }
          } }]}>
            {(fields, { add, remove }, { errors }) => (
              <div>
                <div className="mb-2 flex items-center justify-between">
                  <Text strong>审批节点</Text>
                  <Button size="small" icon={<Plus size={14} />} onClick={() => add(defaultApprovalNode(fields.length + 1))}>
                    添加节点
                  </Button>
                </div>
                {fields.map((field, index) => (
                  <Card key={field.key} size="small" className="mb-3">
                    <Row gutter={12}>
                      <Col span={6}>
                        <Form.Item name={[field.name, 'level']} label="级别" initialValue={index + 1}>
                          <InputNumber min={1} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col span={18}>
                        <Form.Item name={[field.name, 'name']} label="节点名称" rules={[{ required: true }]}>
                          <Input placeholder="直属主管审批" />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Row gutter={12}>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'approver_type']} label="审批人类型" rules={[{ required: true }]}>
                          <Select options={approverTypeOptions} />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'assignee_type']} label="动态解析类型">
                          <Select allowClear options={dynamicApproverOptions} />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'assignee_value']} label="解析值">
                          <Input placeholder="部门/团队/项目ID，或金额阈值" />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Row gutter={12}>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'approver_ids']} label="固定审批人ID">
                          <Select mode="tags" tokenSeparators={[',']} placeholder="输入用户ID" />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'approval_mode']} label="审批模式" initialValue="any">
                          <Select options={[
                            { value: 'any', label: '任一通过' },
                            { value: 'all', label: '全部通过' },
                            { value: 'sequential', label: '顺序审批' },
                            { value: 'parallel', label: '并行审批' },
                          ]} />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'timeout_hours']} label="超时小时">
                          <InputNumber min={0} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Row gutter={12}>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'allow_reject']} label="允许拒绝" valuePropName="checked" initialValue>
                          <Switch />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'allow_delegate']} label="允许委派" valuePropName="checked">
                          <Switch />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name={[field.name, 'reject_action']} label="拒绝动作" initialValue="end">
                          <Select options={[
                            { value: 'end', label: '结束' },
                            { value: 'return', label: '退回' },
                            { value: 'custom', label: '自定义' },
                          ]} />
                        </Form.Item>
                      </Col>
                    </Row>
                    {fields.length > 1 && (
                      <Button danger type="link" onClick={() => remove(field.name)}>
                        删除节点
                      </Button>
                    )}
                  </Card>
                ))}
                <Form.ErrorList errors={errors} />
              </div>
            )}
          </Form.List>
        </Form>
      </Modal>
    </div>
  );
}

const approverTypeOptions = [
  { value: 'user', label: '指定用户' },
  { value: 'role', label: '指定角色' },
  { value: 'dept_manager', label: '部门负责人' },
  { value: 'team_leader', label: '团队负责人' },
  { value: 'project_manager', label: '项目负责人' },
  { value: 'temp_team_leader', label: '临时团队负责人' },
  { value: 'amount_based', label: '按金额阈值' },
];

const dynamicApproverOptions = [
  { value: 'dept_manager', label: '部门负责人' },
  { value: 'team_leader', label: '团队负责人' },
  { value: 'project_manager', label: '项目负责人' },
  { value: 'temp_team_leader', label: '临时团队负责人' },
  { value: 'amount_based', label: '按金额阈值' },
];

function defaultApprovalNode(level = 1): ApprovalNode {
  return {
    level,
    name: `审批节点 ${level}`,
    approverType: 'role',
    approverIds: [],
    approvalMode: 'any',
    allowReject: true,
    allowDelegate: false,
    rejectAction: 'end',
  };
}

function normalizeNodes(nodes: ApprovalNode[]): ApprovalNode[] {
  return nodes.map((node, index) => {
    const approverIds = node.approverIds || node.approverIds || [];
    return {
      ...defaultApprovalNode(index + 1),
      ...node,
      approverType: node.approverType || node.approverType || 'role',
      approverIds: approverIds
      .map(id => Number(id))
      .filter(id => Number.isInteger(id) && id > 0),
      assigneeType: node.assigneeType || node.assigneeType,
      assigneeValue: node.assigneeValue || node.assigneeValue,
      approvalMode: node.approvalMode || node.approvalMode || 'any',
      timeoutHours: node.timeoutHours || node.timeoutHours,
      allowReject: node.allowReject ?? node.allowReject ?? true,
      allowDelegate: node.allowDelegate ?? node.allowDelegate ?? false,
      rejectAction: node.rejectAction || node.rejectAction || 'end',
      level: Number(node.level || index + 1),
    };
  });
}

function normalizeWorkflow(workflow: ApprovalWorkflow): ApprovalWorkflow {
  return {
    ...workflow,
    ticketType: workflow.ticketType || workflow.ticketType,
    isActive: workflow.isActive ?? workflow.isActive ?? false,
    nodes: workflow.nodes ? normalizeNodes(workflow.nodes) : [],
  };
}
