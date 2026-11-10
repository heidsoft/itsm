'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Switch,
  Table,
  Button,
  Input,
  Select,
  Modal,
  Form,
  Tag,
  Space,
  Tooltip,
  Row,
  Col,
  Typography,
  Tabs,
  InputNumber,
  Statistic,
  App,
  Alert,
  Spin,
  Divider,
  Popconfirm,
} from 'antd';
import {
  Plus,
  Edit,
  Trash2,
  Copy,
  PlayCircle,
  Settings,
  RefreshCw,
  Users,
  GitBranch,
  Clock,
  CheckCircle,
} from 'lucide-react';
import type {
  AutomationRule,
  CreateAutomationRuleRequest,
  UpdateAutomationRuleRequest,
} from '@/lib/api/ticket-automation-rule-api';
import { TicketAutomationRuleApi } from '@/lib/api/ticket-automation-rule-api';
import { isPermissionDeniedError } from '@/lib/api/http-client';
import { useI18n } from '@/lib/i18n';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;
const WorkflowAutomationPage = () => {
  const { message } = App.useApp();
  const { t } = useI18n();
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<{ denied: boolean; detail: string } | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<AutomationRule | null>(null);
  const [activeTab, setActiveTab] = useState('assignment');
  // 测试规则弹窗（2026-10-14 接线：后端 POST /tickets/automation-rules/:id/test 早已存在）
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [testRuleId, setTestRuleId] = useState<number | null>(null);
  const [testTicketId, setTestTicketId] = useState<string>('');
  const [testResult, setTestResult] = useState<{
    matched: boolean;
    actions?: string[];
    reason?: string;
    error?: string;
  } | null>(null);
  const [testing, setTesting] = useState(false);

  const openTestModal = (ruleId?: number) => {
    setTestRuleId(ruleId ?? null);
    setTestTicketId('');
    setTestResult(null);
    setTestModalOpen(true);
  };

  const runRuleTest = async () => {
    if (!testRuleId) {
      message.warning('Please select a rule to test');
      return;
    }
    if (!testTicketId) {
      message.warning('Please enter the ticket ID used for testing (backend ticketId is required)');
      return;
    }
    setTesting(true);
    setTestResult(null);
    try {
      const result = await TicketAutomationRuleApi.testRule({
        ruleId: testRuleId,
        ticketId: Number(testTicketId),
      });
      setTestResult(result);
    } catch (e: any) {
      message.error('Test failed:' + (e?.message || 'Unknown error'));
    } finally {
      setTesting(false);
    }
  };

  // 复制规则：以「副本」名重建一条（后端无 clone 端点，创建是最小可靠语义）
  const handleCopyRule = async (record: AutomationRule) => {
    try {
      await TicketAutomationRuleApi.createRule({
        name: `${record.name} Copy`,
        description: record.description,
        type: record.type,
        priority: record.priority,
        conditions: record.conditions,
        actions: record.actions,
        isActive: false,
      });
      message.success('Copied as "' + record.name + ' Copy" (disabled by default, please check and enable it)');
      loadRules();
    } catch (e: any) {
      message.error('Copy failed:' + (e?.message || 'Unknown error'));
    }
  };

  useEffect(() => {
    loadRules();
  }, []);

  const loadRules = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const response = await TicketAutomationRuleApi.listRules();
      setRules(response.items);
    } catch (error) {
      // 失败不能画成「0 条规则」：统计卡加空表格会让用户以为系统里确实没有规则。
      setRules([]);
      setLoadError({
        denied: isPermissionDeniedError(error),
        detail: error instanceof Error ? error.message : '',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRule = () => {
    setEditingRule(null);
    setModalVisible(true);
  };

  const handleEditRule = (rule: AutomationRule) => {
    setEditingRule(rule);
    setModalVisible(true);
  };

  const handleDeleteRule = async (id: number) => {
    try {
      await TicketAutomationRuleApi.deleteRule(id);
      message.success(t('common.ruleDeleted'));
      loadRules();
    } catch (error) {
      message.error(t('common.deleteFailed'));
    }
  };

  const handleToggleRule = async (rule: AutomationRule) => {
    try {
      await TicketAutomationRuleApi.updateRule(rule.id, {
        name: rule.name,
        description: rule.description || '',
        type: rule.type,
        conditions: rule.conditions,
        actions: rule.actions,
        priority: rule.priority,
        isActive: !rule.isActive,
      });
      message.success(rule.isActive ? t('common.ruleDisabled') : t('common.ruleEnabled'));
      loadRules();
    } catch (error) {
      message.error(t('common.operationFailed'));
    }
  };

  const getRuleTypeColor = (type: string) => {
    const colors = {
      assignment: 'blue',
      routing: 'green',
      escalation: 'orange',
    };
    return colors[type as keyof typeof colors] || 'default';
  };

  const getRuleTypeText = (type: string) => {
    const texts = {
      assignment: 'Auto Assign',
      routing: 'Intelligent routing',
      escalation: 'Automatic upgrade',
    };
    return texts[type as keyof typeof texts] || type;
  };

  const getRuleTypeIcon = (type: string) => {
    const icons = {
      assignment: <Users className='w-4 h-4' />,
      routing: <GitBranch className='w-4 h-4' />,
      escalation: <Clock className='w-4 h-4' />,
    };
    return icons[type as keyof typeof icons] || <Settings className='w-4 h-4' />;
  };

  const columns = [
    {
      title: 'Rule Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: AutomationRule) => (
        <div>
          <div className='font-medium'>{name}</div>
          <div className='text-sm text-gray-500'>{record.description}</div>
        </div>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => (
        <Tag color={getRuleTypeColor(type)} icon={getRuleTypeIcon(type)}>
          {getRuleTypeText(type)}
        </Tag>
      ),
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: number) => (
        <Tag color={priority === 1 ? 'red' : priority === 2 ? 'orange' : 'blue'}>P{priority}</Tag>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 100,
      render: (isActive: boolean, record: AutomationRule) => (
        <Switch aria-label={`Switching rules ${record.name} enabled status`} checked={isActive} onChange={() => handleToggleRule(record)} size='small' />
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date: string) => (
        <div className='text-sm'>{formatDateTime(date)}</div>
      ),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 150,
      render: (record: AutomationRule) => (
        <Space>
          <Tooltip title="Edit">
            <Button
              aria-label="Edit"
              type='text'
              icon={<Edit className='w-4 h-4' />}
              onClick={() => handleEditRule(record)}
            />
          </Tooltip>
          <Tooltip title="Copy">
            <Button
              aria-label="Copy"
              type='text'
              icon={<Copy className='w-4 h-4' />}
              onClick={() => handleCopyRule(record)}
            />
          </Tooltip>
          <Popconfirm
            title="Confirm to delete this automation rule?"
            description="The rule will stop executing immediately after deletion and cannot be restored."
            okText="Delete"
            okButtonProps={{ danger: true }}
            cancelText="Cancel"
            onConfirm={() => handleDeleteRule(record.id)}
          >
            <Tooltip title="Delete">
              <Button aria-label="Delete" type='text' danger icon={<Trash2 className='w-4 h-4' />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const filteredRules = rules.filter(rule => {
    switch (activeTab) {
      case 'assignment':
        return rule.type === 'assignment';
      case 'routing':
        return rule.type === 'routing';
      case 'escalation':
        return rule.type === 'escalation';
      default:
        return true;
    }
  });

  return (
    <div className='p-6'>
      {/* 页面头部 */}
      <div className='mb-6'>
        <h1 className='text-2xl font-bold text-gray-900'>Workflow automation</h1>
        <p className='text-gray-600 mt-1'>Configure and manage workflow automation rules to improve process efficiency</p>
      </div>
      {/* 全局设置 */}
      <Card className='rounded-lg shadow-sm border border-gray-200 mb-6'>
        <Row gutter={[16, 16]} align='middle'>
          <Col xs={24} sm={12}>
            <div>
              <Title level={5} className='!mb-1'>

                Workflow automation
              </Title>
              <Text type='secondary'>Please operate the start and stop of rules in each line. Batch start and stop are not provided yet.</Text>
            </div>
          </Col>
          <Col xs={24} sm={12}>
            <Space>
              <Button icon={<RefreshCw className='w-4 h-4' />} onClick={loadRules}>

                Refresh
              </Button>
              <Button icon={<PlayCircle className='w-4 h-4' />} type='primary' onClick={() => openTestModal()}>

                Test rules
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 加载失败/无权限时不渲染统计卡与表格：0 条规则和「看不到规则」是两回事 */}
      {loadError ? (
        <Alert
          type={loadError.denied ? 'warning' : 'error'}
          showIcon
          title={loadError.denied ? 'No permission to view automation rules' : t('common.loadRulesFailed')}
          description={
            loadError.denied
              ? 'The current role is not granted read permission for automation rules. This is not "there are no rules in the system". Please contact the administrator to configure permissions for your role\'s opening rules.'
              : loadError.detail || 'The backend failed to return the rule list, please try again later.'
          }
          action={
            <Button
              size='small'
              icon={<RefreshCw className='w-4 h-4' />}
              onClick={loadRules}
              loading={loading}
            >

              Retry
            </Button>
          }
        />
      ) : (
        <>
          {/* 统计信息 */}
          <Row gutter={[16, 16]} className='mb-6'>
            <Col xs={24} sm={12} lg={6}>
              <Card className='rounded-lg shadow-sm border border-gray-200'>
                <Statistic
                  title="Auto-assign rules"
                  value={rules.filter(r => r.type === 'assignment').length}
                  prefix={<Users className='w-5 h-5' />}
                  styles={{ content: { color: '#1890ff' } }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card className='rounded-lg shadow-sm border border-gray-200'>
                <Statistic
                  title="Intelligent routing rules"
                  value={rules.filter(r => r.type === 'routing').length}
                  prefix={<GitBranch className='w-5 h-5' />}
                  styles={{ content: { color: '#52c41a' } }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card className='rounded-lg shadow-sm border border-gray-200'>
                <Statistic
                  title="Automatic upgrade rules"
                  value={rules.filter(r => r.type === 'escalation').length}
                  prefix={<Clock className='w-5 h-5' />}
                  styles={{ content: { color: '#faad14' } }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card className='rounded-lg shadow-sm border border-gray-200'>
                <Statistic
                  title="Active rule"
                  value={rules.filter(r => r.isActive).length}
                  prefix={<CheckCircle className='w-5 h-5' />}
                  styles={{ content: { color: '#52c41a' } }}
                />
              </Card>
            </Col>
          </Row>

          {/* 规则管理 */}
          <Card className='rounded-lg shadow-sm border border-gray-200'>
            <div className='flex items-center justify-between mb-4'>
              <Title level={5}>Automation Rules</Title>
              <Button type='primary' icon={<Plus className='w-4 h-4' />} onClick={handleCreateRule}>

                Create Rule
              </Button>
            </div>

            <Tabs
              activeKey={activeTab}
              onChange={setActiveTab}
              items={[
                {
                  key: 'assignment',
                  label: 'Auto Assign',
                  children: (
                    <Table
                      columns={columns}
                      dataSource={filteredRules}
                      rowKey='id'
                      loading={loading}
                      pagination={false}
                    />
                  ),
                },
                {
                  key: 'routing',
                  label: 'Intelligent routing',
                  children: (
                    <Table
                      columns={columns}
                      dataSource={filteredRules}
                      rowKey='id'
                      loading={loading}
                      pagination={false}
                    />
                  ),
                },
                {
                  key: 'escalation',
                  label: 'Automatic upgrade',
                  children: (
                    <Table
                      columns={columns}
                      dataSource={filteredRules}
                      rowKey='id'
                      loading={loading}
                      pagination={false}
                    />
                  ),
                },
              ]}
            />
          </Card>
        </>
      )}

      {/* 创建/编辑规则模态框 */}
      <Modal
        title={editingRule ? 'EditRule' : 'Create Rule'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
        destroyOnHidden
      >
        <Form
          layout='vertical'
          initialValues={
            editingRule || {
              type: activeTab,
              priority: 1,
              isActive: true,
              conditions: {},
              actions: {},
            }
          }
          onFinish={async values => {
            try {
              if (editingRule) {
                await TicketAutomationRuleApi.updateRule(editingRule.id, {
                  ...values,
                } as unknown as UpdateAutomationRuleRequest);
              } else {
                await TicketAutomationRuleApi.createRule({
                  ...values,
                } as unknown as CreateAutomationRuleRequest);
              }
              message.success(editingRule ? t('common.ruleUpdated') : t('common.ruleCreated'));
              setModalVisible(false);
              loadRules();
            } catch (error) {
              const detail = error instanceof Error ? error.message : '';
              message.error(detail ? `${t('common.operationFailed')}: ${detail}` : t('common.operationFailed'));
            }
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='name'
                label="Rule Name"
                rules={[{ required: true, message: 'Please enter a rule name' }]}
              >
                <Input placeholder="Please enter a rule name" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='type'
                label="Rule type"
                rules={[{ required: true, message: 'Please select a rule type' }]}
              >
                <Select placeholder="Select rule type" options={[{ value: "assignment", label: "Auto Assign" }, { value: "routing", label: "Intelligent routing" }, { value: "escalation", label: "Automatic upgrade" }]} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name='description' label="Rule Description">
            <Input.TextArea rows={3} placeholder="Please enter a rule description" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='priority'
                label="Priority"
                rules={[{ required: true, message: 'Please set priority' }]}
              >
                <Select placeholder="Select priority" options={[{ value: 1, label: "High (P1)" }, { value: 2, label: "Medium (P2)" }, { value: 3, label: "Low (P3)" }]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='isActive' label="Status" valuePropName='checked'>
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Divider>Trigger Condition</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name={['conditions', 'priority']} label="Priority">
                <Select placeholder="Select priority" allowClear options={[{ value: "low", label: "Low" }, { value: "normal", label: "Normal" }, { value: "high", label: "High" }, { value: "critical", label: "Urgent" }]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name={['conditions', 'category']} label="Category">
                <Select placeholder="Select category" allowClear options={[{ value: "technical", label: "Technology" }, { value: "finance", label: "Finance" }, { value: "hr", label: "Personnel" }, { value: "general", label: "General" }]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name={['conditions', 'status']} label="Status">
                <Select placeholder="Select status" allowClear options={[{ value: "pending", label: "Pending" }, { value: "in_progress", label: "In Progress" }, { value: "completed", label: "Completed" }]} />
              </Form.Item>
            </Col>
          </Row>

          <Divider>Action</Divider>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) => prevValues.type !== currentValues.type}
          >
            {({ getFieldValue }) => {
              const ruleType = getFieldValue('type');

              if (ruleType === 'assignment') {
                return (
                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        name={['actions', 'assign_to']}
                        label="Assigned To"
                        rules={[{ required: true, message: 'Please select distribution target' }]}
                      >
                        <Select placeholder="Select distribution target" options={[{ value: "expert", label: "Expert" }, { value: "manager", label: "Manager" }, { value: "supervisor", label: "Supervisor" }, { value: "round_robin", label: "Poll allocation" }, { value: "least_busy", label: "least busy" }]} />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item name={['actions', 'method']} label="Allocation method">
                        <Select placeholder="Select allocation method" options={[{ value: "round_robin", label: "Polling" }, { value: "least_busy", label: "least busy" }, { value: "skill_based", label: "Skill-based" }]} />
                      </Form.Item>
                    </Col>
                  </Row>
                );
              }

              if (ruleType === 'routing') {
                return (
                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        name={['actions', 'route_to']}
                        label="route to"
                        rules={[{ required: true, message: 'Please select a routing target' }]}
                      >
                        <Select placeholder="Select routing target" options={[{ value: "tech_support", label: "Technical Support Group" }, { value: "finance_team", label: "Finance Group" }, { value: "hr_team", label: "Personnel Team" }, { value: "management", label: "Management" }]} />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item name={['actions', 'notify']} label="Notification" valuePropName='checked'>
                        <Switch />
                      </Form.Item>
                    </Col>
                  </Row>
                );
              }

              if (ruleType === 'escalation') {
                return (
                  <Row gutter={16}>
                    <Col span={8}>
                      <Form.Item
                        name={['actions', 'escalate_to']}
                        label="Upgrade to"
                        rules={[{ required: true, message: 'Please select the upgrade target' }]}
                      >
                        <Select placeholder="Select upgrade target" options={[{ value: "manager", label: "Manager" }, { value: "supervisor", label: "Supervisor" }, { value: "director", label: "Director" }]} />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item
                        name={['actions', 'time_limit']}
                        label="Time limit (hours)"
                        rules={[{ required: true, message: 'Please set a time limit' }]}
                      >
                        <InputNumber min={1} max={168} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item name={['actions', 'notify']} label="Notification" valuePropName='checked'>
                        <Switch />
                      </Form.Item>
                    </Col>
                  </Row>
                );
              }

              return null;
            }}
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit'>
                {editingRule ? 'Update' : 'Create'}
              </Button>
              <Button onClick={() => setModalVisible(false)}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 测试规则弹窗：选规则 + 填工单 ID，走后端 /tickets/automation-rules/:id/test */}
      <Modal
        title="Test automation rules"
        open={testModalOpen}
        onOk={runRuleTest}
        onCancel={() => setTestModalOpen(false)}
        confirmLoading={testing}
        okText="Execute test"
        cancelText="Close"
      >
        <Form layout="vertical">
          <Form.Item label="Rule" required>
            <Select
              placeholder="Select rules to test"
              value={testRuleId ?? undefined}
              onChange={(v) => setTestRuleId(v)}
              options={rules.map(r => ({ value: r.id, label: r.name }))}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item label="Test ticket ID" required tooltip="Use a real work order to verify whether the rule conditions are met.">
            <Input
              value={testTicketId}
              onChange={(e) => setTestTicketId(e.target.value.replace(/\D/g, ''))}
              placeholder="e.g. 123"
              inputMode="numeric"
            />
          </Form.Item>
        </Form>
        {testResult && (
          <div className="mt-2">
            <Tag color={testResult.matched ? 'green' : 'orange'}>
              {testResult.matched ? 'Condition hit' : 'Miss'}
            </Tag>
            {testResult.reason && <Text type="secondary" className="ml-2">{testResult.reason}</Text>}
            {testResult.error && <Text type="danger" className="ml-2">{testResult.error}</Text>}
            {testResult.actions && testResult.actions.length > 0 && (
              <div className="mt-2">
                <Text strong>Action to be performed:</Text>
                <div className="mt-1">
                  {testResult.actions.map((a, i) => (
                    <Tag key={i} className="mb-1">{a}</Tag>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default WorkflowAutomationPage;
