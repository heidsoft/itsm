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
  message,
  Tabs,
  Descriptions,
  Switch,
  Tooltip,
  Popconfirm,
  Row,
  Col,
  Statistic,
} from 'antd';
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

const { Option } = Select;

type ProcessRoutingRule = ProcessBinding & {
  departmentName?: string;
  teamName?: string;
};

export default function ProcessRoutingPage() {
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
      message.success('Routing rule saved successfully');
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
      message.error('Failed to delete routing rule');
      return;
    }
    message.success('Routing rule deleted');
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
    { value: 'alert_handling', label: 'Alert Handling' },
    { value: 'change_release', label: 'Change Release' },
    { value: 'emergency_change', label: 'Emergency Change' },
    { value: 'code_release', label: 'Code Release' },
    { value: 'expense_approval', label: 'Expense Approval' },
    { value: 'leave_approval', label: 'Leave Approval' },
    { value: 'recruitment_approval', label: 'Recruitment Approval' },
  ];

  // Category options
  const categoryOptions = [
    { value: 'operations', label: 'Operations' },
    { value: 'rd', label: 'R&D' },
    { value: 'finance', label: 'Finance' },
    { value: 'hr', label: 'HR' },
    { value: 'general', label: 'General' },
  ];

  // Table columns
  const columns = [
    {
      title: 'Business Type',
      dataIndex:'businessType',
      key:'businessType',
      render: (type: string) => (
        <Tag color={
          type === 'incident' ? 'red' :
          type === 'change' ? 'blue' :
          type === 'service_request' ? 'green' :
          'default'
        }>
          {type}
        </Tag>
      ),
    },
    {
      title: 'Sub Type',
      dataIndex:'businessSubType',
      key:'businessSubType',
    },
    {
      title: 'Department',
      dataIndex:'departmentName',
      key:'departmentName',
      render: (name: string) => name || <Tag>Global</Tag>,
    },
    {
      title: 'Scenario',
      dataIndex: 'scenario',
      key: 'scenario',
      render: (scenario: string) => scenario && <Tag color="purple">{scenario}</Tag>,
    },
    {
      title: 'Process',
      dataIndex:'processDefinitionKey',
      key:'processDefinitionKey',
      render: (key: string) => <Tag color="cyan">{key}</Tag>,
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      sorter: (a: ProcessRoutingRule, b: ProcessRoutingRule) => a.priority - b.priority,
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'red'}>
          {active ? 'Active' : 'Inactive'}
        </Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: ProcessRoutingRule) => (
        <Space>
          <Tooltip title="Edit">
            <Button size="small" icon={<Edit />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Tooltip title="Duplicate">
            <Button size="small" icon={<Copy />} onClick={() => handleDuplicate(record)} />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this rule?"
            onConfirm={() => handleDelete(record.id)}
          >
            <Tooltip title="Delete">
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
    <div style={{ padding: 24 }}>
      {/* Statistics */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic title="Total Rules" value={stats.total} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Active Rules" value={stats.active} valueStyle={{ color: '#3f8600' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Department-Specific" value={stats.departmentSpecific} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Global Rules" value={stats.global} />
          </Card>
        </Col>
      </Row>

      {/* Rules Table */}
      <Card
        title={
          <Space>
            <Settings />
            <span>Process Routing Rules</span>
          </Space>
        }
        extra={
          <Space>
            <Input
              placeholder="Search rules..."
              prefix={<Search />}
              style={{ width: 200 }}
            />
            <Button type="primary" icon={<Plus />} onClick={() => setShowModal(true)}>
              Add Rule
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
            showTotal: (total) => `Total ${total} rules`,
          }}
        />
      </Card>

      {/* Add/Edit Modal */}
      <Modal
        title={selectedRule ? 'Edit Routing Rule' : 'New Routing Rule'}
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
                label: 'Basic Info',
                children: (
                  <>
                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item
                          name="business_type"
                          label="Business Type"
                          rules={[{ required: true }]}
                        >
                          <Select>
                            <Option value="ticket">Ticket</Option>
                            <Option value="incident">Incident</Option>
                            <Option value="change">Change</Option>
                            <Option value="service_request">Service Request</Option>
                            <Option value="problem">Problem</Option>
                            <Option value="release">Release</Option>
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="business_sub_type" label="Sub Type">
                          <Input />
                        </Form.Item>
                      </Col>
                    </Row>

                    <Form.Item
                      name="process_definition_key"
                      label="Process Definition"
                      rules={[{ required: true }]}
                    >
                      <Select showSearch optionFilterProp="children">
                        {processDefinitions.map((pd: any) => (
                          <Option key={pd.key} value={pd.key}>
                            {pd.name} ({pd.key})
                          </Option>
                        ))}
                      </Select>
                    </Form.Item>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item
                          name="priority"
                          label="Priority"
                          rules={[{ required: true }]}
                          tooltip="Higher priority = checked first"
                        >
                          <InputNumber min={0} max={1000} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item
                          name="is_active"
                          label="Active"
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
                label: 'Scope',
                children: (
                  <>
                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="department_id" label="Department">
                          <Select
                            allowClear
                            placeholder="Global (All Departments)"
                            showSearch
                            optionFilterProp="children"
                          >
                            {departments.map(dept => (
                              <Option key={dept.id} value={dept.id}>
                                {dept.name} ({dept.code})
                              </Option>
                            ))}
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="team_id" label="Team">
                          <Select
                            allowClear
                            placeholder="All Teams"
                            showSearch
                            optionFilterProp="children"
                          >
                            {teams.map(team => (
                              <Option key={team.id} value={team.id}>
                                {team.name} ({team.code})
                              </Option>
                            ))}
                          </Select>
                        </Form.Item>
                      </Col>
                    </Row>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="scenario" label="Scenario">
                          <Select allowClear placeholder="Any Scenario">
                            {scenarioOptions.map(opt => (
                              <Option key={opt.value} value={opt.value}>
                                {opt.label}
                              </Option>
                            ))}
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="category" label="Category">
                          <Select allowClear placeholder="Any Category">
                            {categoryOptions.map(opt => (
                              <Option key={opt.value} value={opt.value}>
                                {opt.label}
                              </Option>
                            ))}
                          </Select>
                        </Form.Item>
                      </Col>
                    </Row>
                  </>
                ),
              },
              {
                key: 'advanced',
                label: 'Advanced',
                children: (
                  <>
                    <Form.Item name="approval_chain_id" label="Approval Chain ID">
                      <Input placeholder="Optional approval chain override" />
                    </Form.Item>
                    <Form.Item name="sla_policy_id" label="SLA Policy ID">
                      <Input placeholder="Optional SLA policy override" />
                    </Form.Item>
                    <Form.Item name="conditions" label="Conditions JSON">
                      <Input.TextArea
                        rows={5}
                        placeholder='{"severity":"p0","min_amount":100000}'
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
