'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Select,
  Input,
  InputNumber,
  Space,
  Tag,
  App,
  Tabs,
  Descriptions,
  Switch,
  Tooltip,
  Popconfirm,
  Row,
  Col,
  Statistic,
  Alert,
} from 'antd';
import Link from 'next/link';
import { Plus, Edit, Delete, Copy, Search, Settings } from 'lucide-react';
import type {
  ProcessBinding,
  ProcessBindingPayload} from '@/lib/api/process-binding-api';
import {
  ProcessBindingApi
} from '@/lib/api/process-binding-api';
import { WorkflowApi } from '@/lib/api/workflow-api';
import type { Department } from '@/lib/services/department-service';
import { departmentService } from '@/lib/services/department-service';
import type { Team } from '@/lib/services/team-service';
import { teamService } from '@/lib/services/team-service';



type ProcessRoutingRule = ProcessBinding & {
  departmentName?: string;
  teamName?: string;
};

export default function ProcessRoutingPage() {
  const { message } = App.useApp();
  const [rules, setRules] = useState<ProcessRoutingRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedRule, setSelectedRule] = useState<ProcessRoutingRule | null>(null);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [processDefinitions, setProcessDefinitions] = useState<Array<{ key: string; name: string }>>([]);
  const [form] = Form.useForm();

  // Load data
  useEffect(() => {
    loadRules();
    loadDepartments();
    loadTeams();
    loadProcessDefinitions();
  }, []);

  const loadRules = async () => {
    setLoading(true);
    try {
      const data = await ProcessBindingApi.list();
      setRules(data);
    } catch (error) {
      console.error('Failed to load routing rules:', error);
      message.error('加载流程路由规则失败');
    } finally {
      setLoading(false);
    }
  };

  const loadDepartments = async () => {
    try {
      const data = await departmentService.getDepartmentTree();
      setDepartments(flattenDepartments(data));
    } catch (error) {
      console.error('Failed to load departments:', error);
    }
  };

  const loadTeams = async () => {
    try {
      const data = await teamService.listTeams();
      setTeams(data);
    } catch (error) {
      console.error('Failed to load teams:', error);
    }
  };

  const loadProcessDefinitions = async () => {
    try {
      const response = await WorkflowApi.getWorkflows({ page: 1, pageSize: 100 });
      setProcessDefinitions(response.workflows.map(workflow => ({
        key: workflow.code,
        name: workflow.name,
      })));
    } catch (error) {
      console.error('Failed to load process definitions:', error);
    }
  };

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      const payload: ProcessBindingPayload = {
        ...values,
        conditions: parseJSONField(values.conditions, 'Conditions'),
        priority: values.priority ?? 0,
        isActive: values.isActive ?? true,
      };
      if (selectedRule) {
        await ProcessBindingApi.update(selectedRule.id, payload);
      } else {
        await ProcessBindingApi.create(payload);
      }
      message.success('路由规则已保存');
      setShowModal(false);
      form.resetFields();
      setSelectedRule(null);
      loadRules();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  const handleEdit = (record: ProcessRoutingRule) => {
    setSelectedRule(record);
    form.setFieldsValue({
      ...record,
      conditions: record.conditions ? JSON.stringify(record.conditions, null, 2) : '',
    });
    setShowModal(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await ProcessBindingApi.delete(id);
    } catch {
      message.error('删除路由规则失败');
      return;
    }
    message.success('路由规则已删除');
    loadRules();
  };

  const handleDuplicate = (record: ProcessRoutingRule) => {
    const { id: _id, tenantId: _tenantId, createdAt: _createdAt, updatedAt: _updatedAt, ...rest } = record;
    form.setFieldsValue({
      ...rest,
      conditions: record.conditions ? JSON.stringify(record.conditions, null, 2) : '',
      priority: record.priority + 1,
    });
    setShowModal(true);
  };

  const flattenDepartments = (items: Department[]): Department[] => {
    return items.flatMap(item => [item, ...flattenDepartments(item.children || [])]);
  };

  // 条件 JSON 实时校验（onChange 触发，非法时红字提示但不阻塞输入）
  const jsonValidator = (_: unknown, value: unknown) => {
    if (!value || typeof value !== 'string' || !value.trim()) return Promise.resolve();
    try {
      JSON.parse(value);
      return Promise.resolve();
    } catch {
      return Promise.reject(new Error('JSON 格式不合法，请检查引号、逗号与括号'));
    }
  };

  const parseJSONField = (value: unknown, label: string) => {
    if (!value || typeof value !== 'string' || value.trim() === '') {
      return {};
    }
    try {
      return JSON.parse(value);
    } catch {
      throw new Error(`${label} must be valid JSON`);
    }
  };

  // Scenario options
  const scenarioOptions = [
    { value: 'alert_handling', label: '告警处理' },
    { value: 'change_release', label: '变更发布' },
    { value: 'emergency_change', label: '紧急变更' },
    { value: 'code_release', label: '代码发布' },
    { value: 'expense_approval', label: '费用审批' },
    { value: 'leave_approval', label: '请假审批' },
    { value: 'recruitment_approval', label: '招聘审批' },
  ];

  // Category options
  const categoryOptions = [
    { value: 'operations', label: '运维' },
    { value: 'rd', label: '研发' },
    { value: 'finance', label: '财务' },
    { value: 'hr', label: '人力' },
    { value: 'general', label: '综合' },
  ];

  // Table columns
  const columns = [
    {
      title: '业务类型',
      dataIndex:'businessType',
      key:'businessType',
      render: (type: string) => (
        <Tag color={
          type === 'incident' ? 'red' :
          type === 'change' ? 'blue' :
          type === 'service_request' ? 'green' :
          'default'
        }>
          {type === 'incident' ? '事件' : type === 'change' ? '变更' : type === 'service_request' ? '服务请求' : '其他'}
        </Tag>
      ),
    },
    {
      title: '子类型',
      dataIndex:'businessSubType',
      key:'businessSubType',
    },
    {
      title: '部门',
      dataIndex:'departmentName',
      key:'departmentName',
      render: (name: string) => name || <Tag>全局</Tag>,
    },
    {
      title: '场景',
      dataIndex: 'scenario',
      key: 'scenario',
      render: (scenario: string) => scenario && <Tag color="purple">{scenario}</Tag>,
    },
    {
      title: '流程',
      dataIndex:'processDefinitionKey',
      key:'processDefinitionKey',
      render: (key: string) => <Tag color="cyan">{key}</Tag>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      sorter: (a: ProcessRoutingRule, b: ProcessRoutingRule) => a.priority - b.priority,
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'red'}>
          {active ? '启用' : '停用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: ProcessRoutingRule) => (
        <Space>
          <Tooltip title="编辑">
            <Button size="small" icon={<Edit />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Tooltip title="复制">
            <Button size="small" icon={<Copy />} onClick={() => handleDuplicate(record)} />
          </Tooltip>
          <Popconfirm
            title="确定删除这条规则吗？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Tooltip title="删除">
              <Button size="small" danger icon={<Delete />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // Statistics
  const stats = {
    total: rules.length,
    active: rules.filter(r => r.isActive).length,
    departmentSpecific: rules.filter(r => r.departmentId && r.departmentId > 0).length,
    global: rules.filter(r => !r.departmentId || r.departmentId === 0).length,
  };

  return (
    <div className="space-y-6">
      {/* 关联页面入口 */}
      <Alert
        type="info"
        showIcon
        message="流程路由规则决定业务单据匹配到哪个工作流"
        description={
          <Space wrap>
            <span>相关配置：</span>
            <Link href="/admin/department-processes">部门流程配置</Link>
            <Link href="/admin/workflows">工作流管理</Link>
            <Link href="/workflow/designer">流程设计器</Link>
          </Space>
        }
      />
      {/* Statistics */}
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic title="规则总数" value={stats.total} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="已启用" value={stats.active} valueStyle={{ color: '#3f8600' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="部门专用" value={stats.departmentSpecific} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="全局规则" value={stats.global} />
          </Card>
        </Col>
      </Row>

      {/* Rules Table */}
      <Card
        title={
          <Space>
            <Settings />
            <span>流程路由规则</span>
          </Space>
        }
        extra={
          <Space>
            <Input
              placeholder="搜索规则..."
              prefix={<Search />}
              style={{ width: 200 }}
            />
            <Button type="primary" icon={<Plus />} onClick={() => setShowModal(true)}>
              新增规则
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={rules}
          loading={loading}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `共 ${total} 条记录`,
          }}
        />
      </Card>

      {/* Add/Edit Modal */}
      <Modal
        title={selectedRule ? '编辑路由规则' : '新增路由规则'}
        open={showModal}
        onCancel={() => {
          setShowModal(false);
          setSelectedRule(null);
          form.resetFields();
        }}
        onOk={handleSave}
        width={800}
      >
        <Form form={form} layout="vertical">
          <Tabs
            items={[
              {
                key: 'basic',
                label: '基本信息',
                children: (
                  <>
                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item
                        name="businessType"
                        label="业务类型"
                        rules={[{ required: true }]}
                        >
                        <Select options={[{ value: 'ticket', label: '工单' }, { value: 'incident', label: '事件' }, { value: 'change', label: '变更' }, { value: 'service_request', label: '服务请求' }, { value: 'problem', label: '问题' }, { value: 'release', label: '发布' }]} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="businessSubType" label="子类型">
                          <Input />
                        </Form.Item>
                      </Col>
                    </Row>

                    <Form.Item
                      name="processDefinitionKey"
                      label="流程定义"
                      rules={[{ required: true }]}
                    >
                      <Select showSearch optionFilterProp="children" options={processDefinitions.map(pd => ({ value: pd.key, label: `${pd.name} (${pd.key})` }))} />
                    </Form.Item>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item
                          name="priority"
                          label="优先级"
                          rules={[{ required: true }]}
                          tooltip="数字越小优先级越高"
                        >
                          <InputNumber min={0} max={1000} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item
                          name="isActive"
                          label="启用"
                          valuePropName="checked"
                        >
                          <Switch />
                        </Form.Item>
                      </Col>
                    </Row>
                  </>
                ),
              },
              {
                key: 'scope',
                label: '适用范围',
                children: (
                  <>
                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="departmentId" label="部门">
                          <Select
                            allowClear
                            placeholder="全局（所有部门）"
                            showSearch
                            optionFilterProp="children"
                            options={departments.map(dept => ({ value: dept.id, label: `${dept.name} (${dept.code})` }))}
                          />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="teamId" label="团队">
                          <Select
                            allowClear
                            placeholder="所有团队"
                            showSearch
                            optionFilterProp="children"
                            options={teams.map(team => ({ value: team.id, label: `${team.name} (${team.code})` }))}
                          />
                        </Form.Item>
                      </Col>
                    </Row>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="scenario" label="场景">
                          <Select allowClear placeholder="全部场景" options={scenarioOptions} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="category" label="部门类别">
                          <Select allowClear placeholder="全部类别" options={categoryOptions} />
                        </Form.Item>
                      </Col>
                    </Row>
                  </>
                ),
              },
              {
                key: 'advanced',
                label: '高级设置',
                children: (
                  <>
                    <Form.Item name="approvalChainId" label="审批链 ID（可选）">
                      <Input placeholder="覆盖默认审批链，留空则使用流程内置" />
                    </Form.Item>
                    <Form.Item name="slaPolicyId" label="SLA 策略 ID（可选）">
                      <Input placeholder="覆盖默认 SLA 策略，留空则使用流程内置" />
                    </Form.Item>
                    <Form.Item
                      name="conditions"
                      label="条件配置（JSON 格式）"
                      validateTrigger="onChange"
                      rules={[{ validator: jsonValidator }]}
                    >
                      <Input.TextArea
                        rows={5}
                        placeholder='例如：{"severity":"p0","min_amount":100000}'
                      />
                    </Form.Item>
                  </>
                ),
              },
            ]}
          />
        </Form>
      </Modal>
    </div>
  );
}
