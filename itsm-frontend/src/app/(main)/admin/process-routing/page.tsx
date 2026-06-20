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

const { Option } = Select;

interface ProcessRoutingRule {
  id: number;
  business_type: string;
  business_sub_type?: string;
  department_id?: number;
  department_name?: string;
  team_id?: number;
  team_name?: string;
  scenario?: string;
  category?: string;
  process_definition_key: string;
  priority: number;
  is_active: boolean;
  conditions?: Record<string, any>;
  approval_chain_id?: string;
  sla_policy_id?: string;
}

interface Department {
  id: number;
  name: string;
  code: string;
}

interface Team {
  id: number;
  name: string;
  code: string;
}

export default function ProcessRoutingPage() {
  const [rules, setRules] = useState<ProcessRoutingRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedRule, setSelectedRule] = useState<ProcessRoutingRule | null>(null);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [processDefinitions, setProcessDefinitions] = useState<any[]>([]);
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
      // TODO: Replace with actual API call
      const response = await fetch('/api/v1/process-bindings');
      if (response.ok) {
        const data = await response.json();
        setRules(data.data || []);
      }
    } catch (error) {
      console.error('Failed to load routing rules:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadDepartments = async () => {
    try {
      const response = await fetch('/api/v1/departments');
      if (response.ok) {
        const data = await response.json();
        setDepartments(data.data || []);
      }
    } catch (error) {
      console.error('Failed to load departments:', error);
    }
  };

  const loadTeams = async () => {
    try {
      const response = await fetch('/api/v1/teams');
      if (response.ok) {
        const data = await response.json();
        setTeams(data.data || []);
      }
    } catch (error) {
      console.error('Failed to load teams:', error);
    }
  };

  const loadProcessDefinitions = async () => {
    try {
      const response = await fetch('/api/v1/bpmn/process-definitions');
      if (response.ok) {
        const data = await response.json();
        setProcessDefinitions(data.data?.data || []);
      }
    } catch (error) {
      console.error('Failed to load process definitions:', error);
    }
  };

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      const payload = {
        ...values,
        conditions: parseJSONField(values.conditions, 'Conditions'),
      };
      const response = await fetch(
        selectedRule ? `/api/v1/process-bindings/${selectedRule.id}` : '/api/v1/process-bindings',
        {
          method: selectedRule ? 'PUT' : 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        }
      );
      if (!response.ok) {
        throw new Error('Failed to save routing rule');
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
    const response = await fetch(`/api/v1/process-bindings/${id}`, { method: 'DELETE' });
    if (!response.ok) {
      message.error('Failed to delete routing rule');
      return;
    }
    message.success('Routing rule deleted');
    loadRules();
  };

  const handleDuplicate = (record: ProcessRoutingRule) => {
    const { id, ...rest } = record;
    form.setFieldsValue({
      ...rest,
      conditions: record.conditions ? JSON.stringify(record.conditions, null, 2) : '',
      priority: record.priority + 1,
    });
    setShowModal(true);
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
      dataIndex: 'business_type',
      key: 'business_type',
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
      dataIndex: 'business_sub_type',
      key: 'business_sub_type',
    },
    {
      title: 'Department',
      dataIndex: 'department_name',
      key: 'department_name',
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
      dataIndex: 'process_definition_key',
      key: 'process_definition_key',
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
      dataIndex: 'is_active',
      key: 'is_active',
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
    active: rules.filter(r => r.is_active).length,
    departmentSpecific: rules.filter(r => r.department_id && r.department_id > 0).length,
    global: rules.filter(r => !r.department_id || r.department_id === 0).length,
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
