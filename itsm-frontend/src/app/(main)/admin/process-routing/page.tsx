'use client';

import React, { useState, useEffect, useMemo } from 'react';
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
  const [searchText, setSearchText] = useState('');
  const [form] = Form.useForm();

  // 搜索：规则为全量客户端加载（ProcessBindingApi.list 无分页），本地过滤口径透明
  const filteredRules = useMemo(() => {
    const keyword = searchText.trim().toLowerCase();
    if (!keyword) return rules;
    return rules.filter(rule => {
      const haystack = [
        rule.businessType,
        rule.businessSubType,
        rule.processDefinitionKey,
        rule.scenario,
        rule.category,
        rule.departmentName,
        rule.teamName,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();
      return haystack.includes(keyword);
    });
  }, [rules, searchText]);

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
      message.error('Failed to load process routing rules');
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
      const data = await teamService.listTeams({ page: 1, pageSize: 100 });
      setTeams(data.items);
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
      // 防御性剔除服务端字段：类型层面已 Omit，但 form values spread 可能
      // 在运行期把 tenantId/createdAt/updatedAt 漏进来（这些由 JWT/后端管理）。
      const safeValues = { ...values };
      delete (safeValues as Record<string, unknown>).tenantId;
      delete (safeValues as Record<string, unknown>).createdAt;
      delete (safeValues as Record<string, unknown>).updatedAt;
      const payload: ProcessBindingPayload = {
        ...safeValues,
        conditions: parseJSONField(values.conditions, 'Conditions'),
        priority: values.priority ?? 0,
        isActive: values.isActive ?? true,
      };
      if (selectedRule) {
        await ProcessBindingApi.update(selectedRule.id, payload);
      } else {
        await ProcessBindingApi.create(payload);
      }
      message.success('Routing rule saved');
      setShowModal(false);
      form.resetFields();
      setSelectedRule(null);
      loadRules();
    } catch (error) {
      // validateFields 拒绝时错误已在表单项内联展示，不重复弹提示；
      // API 失败必须让用户看到（2026-10-14 审查：原仅 console.error 静默）
      if (error && typeof error === 'object' && 'errorFields' in error) {
        return;
      }
      console.error('Failed to save routing rule:', error);
      message.error(error instanceof Error ? error.message : 'Failed to save routing rules');
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
      message.error('Failed to delete routing rules');
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

  // 条件 JSON 实时校验（onChange 触发，非法时红字提示但不阻塞输入）
  const jsonValidator = (_: unknown, value: unknown) => {
    if (!value || typeof value !== 'string' || !value.trim()) return Promise.resolve();
    try {
      JSON.parse(value);
      return Promise.resolve();
    } catch {
      return Promise.reject(new Error('Invalid JSON format, please check quotes, commas and brackets'));
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
    { value: 'alert_handling', label: 'Alarm handling' },
    { value: 'change_release', label: 'Change release' },
    { value: 'emergency_change', label: 'Emergency Change' },
    { value: 'code_release', label: 'Code release' },
    { value: 'expense_approval', label: 'Expense Approval' },
    { value: 'leave_approval', label: 'Leave Approval' },
    { value: 'recruitment_approval', label: 'Recruitment Approval' },
  ];

  // Category options
  const categoryOptions = [
    { value: 'operations', label: 'Operation and maintenance' },
    { value: 'rd', label: 'Research and Development' },
    { value: 'finance', label: 'Finance' },
    { value: 'hr', label: 'Manpower' },
    { value: 'general', label: 'Comprehensive' },
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
          {type === 'incident' ? 'Incident' : type === 'change' ? 'Change' : type === 'service_request' ? 'Service Request' : 'Other'}
        </Tag>
      ),
    },
    {
      title: 'Subtype',
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
      title: 'Scene',
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
          {active ? 'Enabled' : 'Disable'}
        </Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: unknown, record: ProcessRoutingRule) => (
        <Space>
          <Tooltip title="Edit">
            <Button aria-label="Edit" size="small" icon={<Edit />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Tooltip title="Copy">
            <Button aria-label="Copy" size="small" icon={<Copy />} onClick={() => handleDuplicate(record)} />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this rule?"
            onConfirm={() => handleDelete(record.id)}
          >
            <Tooltip title="Delete">
              <Button aria-label="Delete" size="small" danger icon={<Delete />} />
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
        title="Process routing rules determine which workflow a business document matches"
      />
      {/* Statistics */}
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic title="Total number of rules" value={stats.total} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Enabled" value={stats.active} styles={{ content: { color: '#3f8600' } }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Department only" value={stats.departmentSpecific} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Global rules" value={stats.global} />
          </Card>
        </Col>
      </Row>

      {/* Rules Table */}
      <Card
        title={
          <Space>
            <Settings />
            <span>Process routing rules</span>
          </Space>
        }
        extra={
          <Space>
            <Input
              placeholder="Search rules..."
              prefix={<Search />}
              style={{ width: 200 }}
              value={searchText}
              onChange={e => setSearchText(e.target.value)}
              allowClear
            />
            <Button
              type="primary"
              icon={<Plus />}
              onClick={() => {
                // 新增前清空表单与选中态：修复「编辑后点新增带回上一次值」
                setSelectedRule(null);
                form.resetFields();
                setShowModal(true);
              }}
            >

              Add new rule
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={filteredRules}
          loading={loading}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `Total ${total} records`,
          }}
        />
      </Card>

      {/* Add/Edit Modal */}
      <Modal
        title={selectedRule ? 'Edit routing rules' : 'Add routing rule'}
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
                        name="businessType"
                        label="Business Type"
                        rules={[{ required: true }]}
                        >
                        <Select options={[{ value: 'ticket', label: 'Ticket' }, { value: 'incident', label: 'Incident' }, { value: 'change', label: 'Change' }, { value: 'service_request', label: 'Service Request' }, { value: 'problem', label: 'Problem' }, { value: 'release', label: 'Release' }]} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="businessSubType" label="Subtype">
                          <Input />
                        </Form.Item>
                      </Col>
                    </Row>

                    <Form.Item
                      name="processDefinitionKey"
                      label="Process Definitions"
                      rules={[{ required: true }]}
                    >
                      <Select showSearch optionFilterProp="children" options={processDefinitions.map(pd => ({ value: pd.key, label: `${pd.name} (${pd.key})` }))} />
                    </Form.Item>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item
                          name="priority"
                          label="Priority"
                          rules={[{ required: true }]}
                          tooltip="The smaller the number, the higher the priority."
                        >
                          <InputNumber min={0} max={1000} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item
                          name="isActive"
                          label="Enabled"
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
                label: 'Scope of application',
                children: (
                  <>
                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="departmentId" label="Department">
                          <Select
                            allowClear
                            placeholder="Global (all departments)"
                            showSearch
                            optionFilterProp="children"
                            options={departments.map(dept => ({ value: dept.id, label: `${dept.name} (${dept.code})` }))}
                          />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="teamId" label="Team">
                          <Select
                            allowClear
                            placeholder="All teams"
                            showSearch
                            optionFilterProp="children"
                            options={teams.map(team => ({ value: team.id, label: `${team.name} (${team.code})` }))}
                          />
                        </Form.Item>
                      </Col>
                    </Row>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="scenario" label="Scene">
                          <Select allowClear placeholder="All scenes" options={scenarioOptions} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="category" label="Department category">
                          <Select allowClear placeholder="All categories" options={categoryOptions} />
                        </Form.Item>
                      </Col>
                    </Row>
                  </>
                ),
              },
              {
                key: 'advanced',
                label: 'Advanced Settings',
                children: (
                  <>
                    <Form.Item name="approvalChainId" label="Approval chain ID (optional)">
                      <Input placeholder="Override the default approval chain, leave it blank to use the built-in process" />
                    </Form.Item>
                    <Form.Item name="slaPolicyId" label="SLA policy ID (optional)">
                      <Input placeholder="Override the default SLA policy, leave it blank to use the built-in process" />
                    </Form.Item>
                    <Form.Item
                      name="conditions"
                      label="Conditional configuration (JSON format)"
                      validateTrigger="onChange"
                      rules={[{ validator: jsonValidator }]}
                    >
                      <Input.TextArea
                        rows={5}
                        placeholder="For example: {&quot;severity&quot;:&quot;p0&quot;,&quot;min_amount&quot;:100000}"
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
