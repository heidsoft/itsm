'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  Alert,
  Card,
  Tree,
  Table,
  Button,
  Select,
  App,
  Descriptions,
  Tag,
  Space,
  Modal,
  Form,
  Input,
  InputNumber,
  Switch,
  Row,
  Col,
  Statistic,
  Tooltip,
  Popconfirm,
} from 'antd';
import {
  Building2,
  Plus,
  RefreshCw,
  CheckCircle,
  AlertTriangle,
  Edit,
  Trash2,
} from 'lucide-react';
import type { ProcessBinding, ProcessBindingPayload } from '@/lib/api/process-binding-api';
import { ProcessBindingApi } from '@/lib/api/process-binding-api';
import { WorkflowApi } from '@/lib/api/workflow-api';
import type { Department } from '@/lib/services/department-service';
import { departmentService } from '@/lib/services/department-service';

interface DepartmentTreeNode {
  title: string;
  key: number;
  icon: React.ReactNode;
  children?: DepartmentTreeNode[];
}

// 与流程路由页共用同一套枚举口径，避免两页各说各话
const businessTypeOptions = [
  { value: 'ticket', label: '工单' },
  { value: 'incident', label: '事件' },
  { value: 'change', label: '变更' },
  { value: 'service_request', label: '服务请求' },
  { value: 'problem', label: '问题' },
  { value: 'release', label: '发布' },
];

const scenarioOptions = [
  { value: 'alert_handling', label: '告警处理' },
  { value: 'change_release', label: '变更发布' },
  { value: 'emergency_change', label: '紧急变更' },
  { value: 'code_release', label: '代码发布' },
  { value: 'expense_approval', label: '费用审批' },
  { value: 'leave_approval', label: '请假审批' },
  { value: 'recruitment_approval', label: '招聘审批' },
];

const categoryOptions = [
  { value: 'operations', label: '运维' },
  { value: 'rd', label: '研发' },
  { value: 'finance', label: '财务' },
  { value: 'hr', label: '人力' },
  { value: 'general', label: '综合' },
];

const getScenarioColor = (scenario?: string) => {
  if (!scenario) return 'default';
  if (scenario.includes('alert')) return 'red';
  if (scenario.includes('change')) return 'blue';
  if (scenario.includes('release')) return 'green';
  if (scenario.includes('expense') || scenario.includes('budget')) return 'gold';
  if (scenario.includes('leave') || scenario.includes('recruitment')) return 'purple';
  return 'default';
};

export default function DepartmentProcessPage() {
  const { message } = App.useApp();
  const [departments, setDepartments] = useState<Department[]>([]);
  const [selectedDeptId, setSelectedDeptId] = useState<number | null>(null);
  const [selectedDept, setSelectedDept] = useState<Department | null>(null);
  const [deptProcesses, setDeptProcesses] = useState<ProcessBinding[]>([]);
  // 列表初值必须为 true：选中部门后会发起拉取，初值 false 会让首帧先渲染
  // 「流程总数 0」的假统计卡与空表格，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  const [showInitModal, setShowInitModal] = useState(false);
  const [departmentType, setDepartmentType] = useState('operations');
  const [initSubmitting, setInitSubmitting] = useState(false);

  // 绑定编辑
  const [showBindingModal, setShowBindingModal] = useState(false);
  const [editingBinding, setEditingBinding] = useState<ProcessBinding | null>(null);
  const [bindingSubmitting, setBindingSubmitting] = useState(false);
  const [processDefinitions, setProcessDefinitions] = useState<Array<{ key: string; name: string }>>([]);
  const [form] = Form.useForm();
  // 换部门时 per-dept 状态必须先重置，慢响应后到不得覆盖新选部门（深链竞态套路）
  const loadSeqRef = useRef(0);

  // Load departments
  useEffect(() => {
    loadDepartments();
    // 表格「流程定义」列要按 key 查中文名渲染，不能等打开弹窗才拉定义
    loadProcessDefinitions();
  }, []);

  // Load department processes when selected
  useEffect(() => {
    if (selectedDeptId) {
      const dept = departments.find(d => d.id === selectedDeptId);
      setSelectedDept(dept || null);
      loadDeptProcesses(selectedDeptId);
    }
  }, [selectedDeptId, departments]);

  const loadDepartments = async () => {
    try {
      const data = await departmentService.getDepartmentTree();
      setDepartments(flattenDepartments(data));
    } catch (error) {
      console.error('Failed to load departments:', error);
      message.error('加载部门失败');
    }
  };

  const loadDeptProcesses = async (deptId: number) => {
    const seq = ++loadSeqRef.current;
    setListLoading(true);
    try {
      const data = await ProcessBindingApi.listDepartmentProcesses(deptId);
      if (seq !== loadSeqRef.current) return;
      setDeptProcesses(data);
    } catch (error) {
      if (seq !== loadSeqRef.current) return;
      console.error('Failed to load department processes:', error);
      message.error('加载部门流程失败');
    } finally {
      if (seq === loadSeqRef.current) {
        setListLoading(false);
      }
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

  const handleInitDefaults = async () => {
    if (!selectedDeptId || initSubmitting) return;
    // 防重入必须同步置位：Modal onOk 双击同一帧内穿透 state 异步渲染
    setInitSubmitting(true);
    try {
      await ProcessBindingApi.initDepartmentProcesses(selectedDeptId, departmentType);
      message.success('部门流程初始化成功');
      setShowInitModal(false);
      loadDeptProcesses(selectedDeptId);
    } catch (error) {
      console.error('Failed to initialize department processes:', error);
      message.error(error instanceof Error ? error.message : '初始化部门流程失败');
    } finally {
      setInitSubmitting(false);
    }
  };

  const openCreateBinding = () => {
    setEditingBinding(null);
    form.resetFields();
    setShowBindingModal(true);
  };

  const openEditBinding = (record: ProcessBinding) => {
    setEditingBinding(record);
    form.setFieldsValue({
      businessType: record.businessType,
      businessSubType: record.businessSubType,
      processDefinitionKey: record.processDefinitionKey,
      priority: record.priority,
      isActive: record.isActive,
      scenario: record.scenario,
      category: record.category,
    });
    setShowBindingModal(true);
  };

  const handleSaveBinding = async () => {
    if (bindingSubmitting) return;
    let values: Record<string, unknown>;
    try {
      values = await form.validateFields();
    } catch {
      // 校验拒绝时错误已在表单项内联展示，不再重复弹提示
      return;
    }
    setBindingSubmitting(true);
    try {
      // 后端 UpdateBinding 对 isDefault/priority/isActive/departmentId/teamId/scenario/category
      // 是无条件 Set（全量覆盖语义），编辑时必须回传全部字段，否则未回传项会被清零
      const payload: ProcessBindingPayload = {
        businessType: values.businessType as string,
        businessSubType: (values.businessSubType as string) || undefined,
        processDefinitionKey: values.processDefinitionKey as string,
        priority: (values.priority as number) ?? 0,
        isActive: (values.isActive as boolean) ?? true,
        departmentId: selectedDeptId ?? undefined,
        scenario: (values.scenario as string) || undefined,
        category: (values.category as string) || undefined,
      };
      if (editingBinding) {
        await ProcessBindingApi.update(editingBinding.id, payload);
        message.success('流程绑定已更新');
      } else {
        await ProcessBindingApi.create(payload);
        message.success('流程绑定已创建');
      }
      setShowBindingModal(false);
      form.resetFields();
      setEditingBinding(null);
      if (selectedDeptId) loadDeptProcesses(selectedDeptId);
    } catch (error) {
      console.error('Failed to save department process binding:', error);
      message.error(error instanceof Error ? error.message : '保存流程绑定失败');
    } finally {
      setBindingSubmitting(false);
    }
  };

  const handleDeleteBinding = async (id: number) => {
    try {
      await ProcessBindingApi.delete(id);
      message.success('流程绑定已删除');
      if (selectedDeptId) loadDeptProcesses(selectedDeptId);
    } catch (error) {
      console.error('Failed to delete department process binding:', error);
      message.error(error instanceof Error ? error.message : '删除流程绑定失败');
    }
  };

  const handleToggleActive = async (record: ProcessBinding) => {
    try {
      await ProcessBindingApi.update(record.id, {
        businessType: record.businessType,
        processDefinitionKey: record.processDefinitionKey,
        businessSubType: record.businessSubType,
        priority: record.priority,
        isActive: !record.isActive,
        departmentId: record.departmentId,
        teamId: record.teamId,
        scenario: record.scenario,
        category: record.category,
      });
      message.success(!record.isActive ? '已启用' : '已停用');
      if (selectedDeptId) loadDeptProcesses(selectedDeptId);
    } catch (error) {
      console.error('Failed to toggle department process binding:', error);
      message.error(error instanceof Error ? error.message : '更新流程绑定失败');
    }
  };

  const flattenDepartments = (items: Department[]): Department[] => {
    return items.flatMap(item => [item, ...flattenDepartments(item.children || [])]);
  };

  // Build department tree
  const buildDeptTree = (depts: Department[], parentId: number = 0): DepartmentTreeNode[] => {
    return depts
      .filter(d => (d as any).parentId === parentId)
      .map(d => ({
        title: `${d.name} (${d.code})`,
        key: d.id,
        icon: <Building2 />,
        children: buildDeptTree(depts, d.id),
      }));
  };

  // Table columns
  const columns = [
    {
      title: '场景',
      dataIndex: 'scenario',
      key: 'scenario',
      render: (scenario: string) => (
        <Tag color={getScenarioColor(scenario)}>
          {scenarioOptions.find(s => s.value === scenario)?.label || scenario || '默认'}
        </Tag>
      ),
    },
    {
      title: '业务类型',
      dataIndex: 'businessType',
      key: 'businessType',
      render: (type: string) => (
        <Tag>{businessTypeOptions.find(t => t.value === type)?.label || type}</Tag>
      ),
    },
    {
      title: '流程定义',
      dataIndex: 'processDefinitionKey',
      key: 'processDefinitionKey',
      // 中文名优先、Key 兜底：直接展示英文工程标识用户看不懂
      render: (key: string) => {
        const def = processDefinitions.find(pd => pd.key === key);
        return (
          <Tag color="cyan" title={key}>
            {def?.name || key}
          </Tag>
        );
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (active: boolean, record: ProcessBinding) => (
        <Tooltip title={active ? '点击停用' : '点击启用'}>
          <Button
            type="text"
            size="small"
            aria-label={active ? '停用绑定' : '启用绑定'}
            onClick={() => handleToggleActive(record)}
          >
            <Tag icon={active ? <CheckCircle /> : <AlertTriangle />} color={active ? 'success' : 'error'}>
              {active ? '启用' : '停用'}
            </Tag>
          </Button>
        </Tooltip>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: unknown, record: ProcessBinding) => (
        <Space>
          <Tooltip title="编辑">
            <Button aria-label="编辑绑定" size="small" icon={<Edit />} onClick={() => openEditBinding(record)} />
          </Tooltip>
          <Popconfirm
            title="确定删除这条流程绑定吗？"
            okText="删除"
            cancelText="取消"
            okType="danger"
            onConfirm={() => handleDeleteBinding(record.id)}
          >
            <Tooltip title="删除">
              <Button aria-label="删除绑定" size="small" danger icon={<Trash2 />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // Statistics
  const stats = {
    total: deptProcesses.length,
    active: deptProcesses.filter(p => p.isActive).length,
    scenarios: new Set(deptProcesses.map(p => p.scenario).filter(Boolean)).size,
  };

  return (
    <div className="space-y-6">
      <Card title="部门流程配置">
        <Row gutter={24}>
          {/* Department Tree */}
          <Col span={8}>
            <Card title="部门列表" size="small">
              <Tree
                showIcon
                treeData={buildDeptTree(departments)}
                onSelect={(keys) => {
                  if (keys.length > 0) {
                    setSelectedDeptId(keys[0] as number);
                  }
                }}
                style={{ maxHeight: 600, overflow: 'auto' }}
              />
            </Card>
          </Col>

          {/* Process Configuration */}
          <Col span={16}>
            {selectedDeptId ? (
              <>
                {/* Department Info */}
                <Card size="small" style={{ marginBottom: 16 }}>
                  <Descriptions size="small">
                    <Descriptions.Item label="部门名称">
                      {selectedDept?.name}
                    </Descriptions.Item>
                    <Descriptions.Item label="编码">
                      <Tag>{selectedDept?.code}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="描述">
                      {selectedDept?.description || '-'}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>

                {/* Statistics */}
                {/* 统计卡必须门控：还没拉到数据时渲染成「流程总数 0」会被误读成「该部门没有绑定流程」 */}
                {!listLoading && (
                <Row gutter={16} style={{ marginBottom: 16 }}>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="流程总数" value={stats.total} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="已启用" value={stats.active} styles={{ content: { color: '#3f8600' } }} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="场景数" value={stats.scenarios} />
                    </Card>
                  </Col>
                </Row>
                )}

                {/* Actions */}
                <Card size="small" style={{ marginBottom: 16 }}>
                  <Space>
                    <Button
                      type="primary"
                      icon={<Plus />}
                      onClick={openCreateBinding}
                    >
                      新增绑定
                    </Button>
                    <Button
                      icon={<Plus />}
                      onClick={() => {
                        // 打开前重置为默认类型，避免上次选择残留
                        setDepartmentType('operations');
                        setShowInitModal(true);
                      }}
                    >
                      初始化默认流程模板
                    </Button>
                    <Button
                      icon={<RefreshCw />}
                      onClick={() => selectedDeptId && loadDeptProcesses(selectedDeptId)}
                    >
                      刷新
                    </Button>
                  </Space>
                </Card>

                {/* Process Table */}
                <Card size="small">
                  <Table
                    columns={columns}
                    dataSource={deptProcesses}
                    loading={listLoading}
                    rowKey="id"
                    pagination={false}
                    locale={{ emptyText: '该部门暂无流程绑定' }}
                  />
                </Card>
              </>
            ) : (
              <Card>
                <div style={{ textAlign: 'center', padding: 48, color: '#999' }}>
                  <Building2 style={{ width: 48, height: 48, marginBottom: 16 }} />
                  <p>请从左侧选择一个部门查看流程配置</p>
                </div>
              </Card>
            )}
          </Col>
        </Row>
      </Card>

      {/* Binding Create/Edit Modal */}
      <Modal
        title={editingBinding ? '编辑流程绑定' : '新增流程绑定'}
        open={showBindingModal}
        onCancel={() => {
          setShowBindingModal(false);
          setEditingBinding(null);
          form.resetFields();
        }}
        onOk={handleSaveBinding}
        confirmLoading={bindingSubmitting}
        okText="保存"
        cancelText="取消"
        afterOpenChange={(open) => {
          // 首次打开时拉流程定义选项；deferred 到动画后避免阻塞首帧
          if (open && processDefinitions.length === 0) {
            loadProcessDefinitions();
          }
        }}
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item label="所属部门">
            <Input disabled value={selectedDept ? `${selectedDept.name} (${selectedDept.code})` : ''} />
          </Form.Item>
          <Form.Item
            name="businessType"
            label="业务类型"
            rules={[{ required: true, message: '请选择业务类型' }]}
          >
            <Select placeholder="选择业务类型" options={businessTypeOptions} />
          </Form.Item>
          <Form.Item name="processDefinitionKey" label="流程定义" rules={[{ required: true, message: '请选择流程定义' }]}>
            <Select
              showSearch
              optionFilterProp="label"
              placeholder="选择流程定义"
              loading={processDefinitions.length === 0}
              options={processDefinitions.map(pd => ({ value: pd.key, label: `${pd.name} (${pd.key})` }))}
            />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="priority" label="优先级" initialValue={100} tooltip="数字越小优先级越高">
                <InputNumber min={0} max={1000} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="isActive" label="启用" valuePropName="checked" initialValue={true}>
                <Switch />
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
        </Form>
      </Modal>

      {/* Initialize Modal */}
      <Modal
        title="初始化部门流程"
        open={showInitModal}
        onCancel={() => setShowInitModal(false)}
        onOk={handleInitDefaults}
        confirmLoading={initSubmitting}
        okText="初始化"
        cancelText="取消"
      >
        <Alert
          title="将为所选部门创建默认流程模板（不会影响已有配置）"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Form layout="vertical">
          <Form.Item label="部门类型">
            <Select
              value={departmentType}
              onChange={setDepartmentType}
              options={[{ value: 'operations', label: '运维（告警处理、变更发布）' }, { value: 'rd', label: '研发（代码发布、需求变更）' }, { value: 'finance', label: '财务（费用审批、预算管理）' }, { value: 'hr', label: '人力（请假、招聘）' }]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
