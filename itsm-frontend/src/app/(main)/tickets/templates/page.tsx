'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import {
  App,
  Alert,
  Card,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Space,
  Tag,
  Typography,
  Row,
  Col,
  Divider,
  Tooltip,
  Popconfirm,
  Statistic,
  Radio,
} from 'antd';
import type { RadioChangeEvent } from 'antd';
import { formatDateTime } from '@/lib/formatters';
import {
  AlertTriangle,
  BookOpen,
  CheckCircle,
  Plus,
  Edit,
  Delete,
  Copy,
  Eye,
  Workflow,
  RefreshCw,
  Search,
  FileText,
  Tag as TagIcon,
} from 'lucide-react';
import { TicketApi } from '@/lib/api/ticket-api';
import { isPermissionDeniedError } from '@/lib/api/http-client';
import { usePermissions } from '@/lib/hooks/use-permissions';
// AppLayout is handled by layout.tsx

const { Title, Text } = Typography;
const { TextArea } = Input;

// Ticket template type definition
interface TicketTemplate {
  id: number;
  name: string;
  description: string;
  type: string;
  category: string;
  subcategory?: string;
  priority: string;
  estimatedTime: string;
  sla: string;
  slaType: 'hours' | 'days' | 'business_hours';
  impact: string;
  urgency: string;
  businessValue: string;
  source: string;
  assigneeGroup?: string;
  autoAssign: boolean;
  requiresApproval: boolean;
  approvalLevel: string;
  customFields: CustomField[];
  tags: string[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  icon: React.ReactNode;
  color: string;
}

interface CustomField {
  id: string;
  name: string;
  type: 'text' | 'number' | 'select' | 'multiselect' | 'date' | 'boolean' | 'file' | 'textarea';
  label: string;
  placeholder?: string;
  required: boolean;
  defaultValue?: unknown;
  options?: string[];
  validation?: string;
  helpText?: string;
  order: number;
}

// Template category metadata (used for the filter dropdown)
const templateCategories = [
  {
    key: 'incident',
    label: 'Incidents',
    icon: <AlertTriangle size={16} />,
    color: 'red',
  },
  {
    key: 'serviceRequest',
    label: 'Service Request',
    icon: <FileText size={16} />,
    color: 'blue',
  },
  {
    key: 'problem',
    label: 'Problems',
    icon: <BookOpen size={16} />,
    color: 'orange',
  },
  {
    key: 'change',
    label: 'Changes',
    icon: <Workflow size={16} />,
    color: 'purple',
  },
];

// 服务端的业务错误原因已经在 ApiRequestError.message 里（例如「没有权限执行该操作」），
// 全部折叠成「请稍后重试」会把权限问题和网络问题说成同一件事。
function errorReason(error: unknown): string {
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message;
  }
  return 'Please try again later';
}

const TicketTemplatesPage = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const { hasPermission } = usePermissions();
  const canCreate = hasPermission('ticket_template', 'create');
  const canUpdate = hasPermission('ticket_template', 'update');
  const canDelete = hasPermission('ticket_template', 'delete');
  const [templates, setTemplates] = useState<TicketTemplate[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 空列表与全 0 的统计卡，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  // 后端 403/2003 与真正的空列表必须分开渲染：把「你没有权限」显示成「系统里没有模板」
  // 会让使用者去改筛选条件或找管理员建数据，而不是申请权限。
  const [loadError, setLoadError] = useState<{ denied: boolean; detail: string } | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TicketTemplate | null>(null);
  const [selectedCategory, setSelectedCategory] = useState('incident');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterStatus, setFilterStatus] = useState('all');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  const loadTemplates = useCallback(async () => {
    setListLoading(true);
    try {
      const response = await TicketApi.getTemplates();

      // 后端契约是 formFields（模板表单的原生载荷），不是 content。
      const apiTemplates: TicketTemplate[] = (response.items || []).map(item => ({
        id: item.id,
        name: item.name,
        description: item.description,
        type: (item.formFields?.type as string) || item.category?.toLowerCase() || 'incident',
        category: item.category,
        subcategory: (item.formFields?.subcategory as string) || undefined,
        priority: item.priority || 'medium',
        estimatedTime: (item.formFields?.estimatedTime as string) || '1 hour',
        sla: (item.formFields?.sla as string) || '4 hours',
        slaType: (item.formFields?.slaType as 'hours' | 'days' | 'business_hours') || 'hours',
        impact: (item.formFields?.impact as string) || 'individual',
        urgency: (item.formFields?.urgency as string) || 'medium',
        businessValue: (item.formFields?.businessValue as string) || 'medium',
        source: (item.formFields?.source as string) || 'web',
        assigneeGroup: (item.formFields?.assigneeGroup as string) || undefined,
        autoAssign: (item.formFields?.autoAssign as boolean) || false,
        requiresApproval: (item.formFields?.requiresApproval as boolean) || false,
        approvalLevel: (item.formFields?.approvalLevel as string) || 'none',
        customFields: (item.formFields?.customFields as CustomField[]) || [],
        tags: (item.formFields?.tags as string[]) || [],
        isActive: item.isActive ?? true,
        createdAt: item.createdAt || new Date().toISOString(),
        updatedAt: item.updatedAt || new Date().toISOString(),
        icon: <FileText size={20} />,
        color:
          item.category === 'System Access'
            ? 'red'
            : item.category === 'Hardware Equipment'
              ? 'orange'
              : 'blue',
      }));

      setTemplates(apiTemplates);
      setLoadError(null);
    } catch (error) {
      console.error('Failed to load templates:', error);
      setTemplates([]);
      setLoadError({
        denied: isPermissionDeniedError(error),
        detail: error instanceof Error ? error.message : '',
      });
    } finally {
      setListLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTemplates();
  }, [loadTemplates]);

  const handleCreateTemplate = () => {
    setEditingTemplate(null);
    setModalVisible(true);
  };

  const handleEditTemplate = (template: TicketTemplate) => {
    setEditingTemplate(template);
    setModalVisible(true);
  };

  const handleDeleteTemplate = async (id: number) => {
    try {
      await TicketApi.deleteTemplate(id);
      setTemplates(prev => prev.filter(t => t.id !== id));
      message.success('Template deleted successfully');
    } catch (error) {
      console.error('Delete template failed:', error);
      message.error(`Delete failed:${errorReason(error)}`);
    }
  };

  // 复制走后端 POST /tickets/templates/:id/copy：前端手抄一份表单字段会丢掉
  // 未在当前卡片上映射的字段，复制出来的模板与原件并不等价。
  const handleCopyTemplate = async (template: TicketTemplate) => {
    try {
      await TicketApi.copyTemplate(template.id);
      message.success('Template copied');
      loadTemplates();
    } catch (error) {
      console.error('Copy template failed:', error);
      message.error(`Copy failed:${errorReason(error)}`);
    }
  };

  const filteredTemplates = templates.filter(template => {
    const matchesKeyword =
      template.name.toLowerCase().includes(searchKeyword.toLowerCase()) ||
      template.description.toLowerCase().includes(searchKeyword.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || template.type === selectedCategory;
    const matchesStatus =
      filterStatus === 'all' ||
      (filterStatus === 'active' && template.isActive) ||
      (filterStatus === 'inactive' && !template.isActive);

    return matchesKeyword && matchesCategory && matchesStatus;
  });

  // 写操作按后端同名权限码判定：ticket_template:create/update/delete，
  // 卡片视图与列表视图共用同一份，避免两处口径漂移。
  const templateActions = (template: TicketTemplate): React.ReactNode[] => {
    const actions: React.ReactNode[] = [
      <Tooltip title="View template" key='view'>
        <Button
          type='text'
          icon={<Eye size={16} />}
          onClick={() => router.push(`/tickets/templates/${template.id}`)}
          aria-label="View template details"
        />
      </Tooltip>,
    ];

    if (canUpdate) {
      actions.push(
        <Tooltip title="Edit Template" key='edit'>
          <Button
            type='text'
            icon={<Edit size={16} />}
            onClick={() => handleEditTemplate(template)}
            aria-label="Edit Template"
          />
        </Tooltip>
      );
    }

    if (canCreate) {
      actions.push(
        <Tooltip title="Copy template" key='copy'>
          <Button
            type='text'
            icon={<Copy size={16} />}
            onClick={() => handleCopyTemplate(template)}
            aria-label="Copy template"
          />
        </Tooltip>
      );
    }

    if (canDelete) {
      actions.push(
        <Tooltip title="Delete template" key='delete'>
          <Popconfirm
            title="Are you sure you want to delete this template?"
            onConfirm={() => handleDeleteTemplate(template.id)}
            okText="OK"
            cancelText="Cancel"
          >
            <Button type='text' danger icon={<Delete size={16} />} aria-label="Delete template" />
          </Popconfirm>
        </Tooltip>
      );
    }

    return actions;
  };

  const renderTemplateCard = (template: TicketTemplate) => (
    <Card key={template.id} hoverable className='h-full' actions={templateActions(template)}>
      <div className='flex items-start mb-3'>
        <div
          className={`inline-flex items-center justify-center w-12 h-12 bg-${template.color}-50 rounded-lg mr-3`}
        >
          <span className={`text-${template.color}-500`}>{template.icon}</span>
        </div>
        <div className='flex-1 min-w-0'>
          <Title level={5} className='mb-1 truncate'>
            {template.name}
          </Title>
          <Text type='secondary' className='text-sm line-clamp-2'>
            {template.description}
          </Text>
        </div>
      </div>

      <div className='space-y-2 mb-4'>
        <div className='flex items-center justify-between'>
          <Text type='secondary' className='text-xs'>
            Type
          </Text>
          <Tag color={template.color}>{template.category}</Tag>
        </div>
        <div className='flex items-center justify-between'>
          <Text type='secondary' className='text-xs'>
            Priority
          </Text>
          <Tag
            color={
              template.priority === 'high'
                ? 'red'
                : template.priority === 'medium'
                  ? 'orange'
                  : 'green'
            }
          >
            {template.priority}
          </Tag>
        </div>
        <div className='flex items-center justify-between'>
          <Text type='secondary' className='text-xs'>
            SLA
          </Text>
          <Text className='text-xs'>{template.sla}</Text>
        </div>
      </div>

      <div className='flex items-center justify-between text-xs text-gray-500 mb-3'>
        <span>Creation time: {formatDateTime(template.createdAt)}</span>
        <span>Update time: {formatDateTime(template.updatedAt)}</span>
      </div>

      <div className='flex items-center justify-between'>
        <div className='flex items-center space-x-2'>
          <Tag color={template.isActive ? 'green' : 'default'}>
            {template.isActive ? 'Enabled' : 'Disable'}
          </Tag>
        </div>
      </div>
    </Card>
  );

  const renderTemplateList = (template: TicketTemplate) => (
    <Card key={template.id} className='mb-3'>
      <div className='flex items-center justify-between'>
        <div className='flex items-center space-x-4'>
          <div
            className={`inline-flex items-center justify-center w-10 h-10 bg-${template.color}-50 rounded-lg`}
          >
            <span className={`text-${template.color}-500`}>{template.icon}</span>
          </div>
          <div>
            <Title level={5} className='mb-1'>
              {template.name}
            </Title>
            <Text type='secondary' className='text-sm'>
              {template.description}
            </Text>
          </div>
        </div>

        <div className='flex items-center space-x-4'>
          <div className='text-center'>
            <Text className='text-xs text-gray-500'>Updated At</Text>
            <div className='font-semibold'>
              {formatDateTime(template.updatedAt)}
            </div>
          </div>
          <div className='flex items-center space-x-2'>
            <Tag color={template.color}>{template.category}</Tag>
            <Tag
              color={
                template.priority === 'high'
                  ? 'red'
                  : template.priority === 'medium'
                    ? 'orange'
                    : 'green'
              }
            >
              {template.priority}
            </Tag>
          </div>
          <Space>{templateActions(template)}</Space>
        </div>
      </div>
    </Card>
  );

  return (
    <>
      {/* Page header actions */}
      <div className='mb-6 flex justify-between items-center'>
        <div>
          <h1 className='text-2xl font-bold text-gray-900'>Work order template management</h1>
          <p className='text-gray-600 mt-1'>Manage and configure work order templates to improve work order creation efficiency</p>
        </div>
        <Space>
          <Button icon={<RefreshCw size={16} />} onClick={loadTemplates} aria-label="Refresh template list">

            Refresh
          </Button>
          {canCreate && (
            <Button
              type='primary'
              icon={<Plus size={16} />}
              onClick={handleCreateTemplate}
              aria-label="New Template"
            >

              New Template
            </Button>
          )}
        </Space>
      </div>
      {/* 加载失败或无权限时，统计卡、筛选器和列表都不渲染：0 个模板和"看不到模板"是两回事 */}
      {loadError ? (
        <Alert
          type={loadError.denied ? 'warning' : 'error'}
          showIcon
          title={loadError.denied ? 'No permission to view the work order template' : 'Template list loading failed'}
          description={
            loadError.denied
              ? 'The current role is not granted read permission for the ticket template. This is not "there is no template in the system", please contact the administrator to open ticket_template:read for your role.'
              : loadError.detail || 'The backend failed to return the template list, please try again later.'
          }
          action={
            <Button
              size='small'
              icon={<RefreshCw size={16} />}
              onClick={loadTemplates}
              loading={listLoading}
            >

              Retry
            </Button>
          }
        />
      ) : (
        <>
          {/* Statistics：加载期不渲染，避免首帧先闪一遍全 0 的假指标 */}
          {!listLoading && (
            <Row gutter={16} className='mb-6'>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="Total number of templates"
                    value={templates.length}
                    prefix={<FileText size={16} style={{ color: '#3b82f6' }} />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="Enable template"
                    value={templates.filter(t => t.isActive).length}
                    styles={{ content: { color: '#52c41a' } }}
                    prefix={<CheckCircle size={16} />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="Number of categories"
                    value={new Set(templates.map(t => t.category).filter(Boolean)).size}
                    prefix={<TagIcon size={16} style={{ color: '#faad14' }} />}
                  />
                </Card>
              </Col>
            </Row>
          )}

          {/* Filter and search */}
          <Card title="Template management" className='mb-6'>
            <div className='flex items-center justify-between mb-4'>
              <Title level={5} className='mb-0'>

                Filter criteria
              </Title>
              <Space>
                <Radio.Group
                  value={viewMode}
                  onChange={(e: RadioChangeEvent) => setViewMode(e.target.value)}
                >
                  <Radio.Button value='grid'>Card view</Radio.Button>
                  <Radio.Button value='list'>List View</Radio.Button>
                </Radio.Group>
              </Space>
            </div>

            <Row gutter={16} align='middle'>
              <Col span={8}>
                <Input.Search
                  placeholder="Search templates..."
                  allowClear
                  value={searchKeyword}
                  onChange={e => setSearchKeyword(e.target.value)}
                  prefix={<Search size={16} />}
                />
              </Col>
              <Col span={6}>
                <Select
                  value={selectedCategory}
                  onChange={setSelectedCategory}
                  style={{ width: '100%' }}
                  placeholder="Select category"
                  options={[
                    { value: 'all', label: 'All categories' },
                    ...templateCategories.map(cat => ({
                      value: cat.key,
                      label: (
                        <div className='flex items-center'>
                          <span className={`text-${cat.color}-500 mr-2`}>{cat.icon}</span>
                          {cat.label}
                        </div>
                      ),
                    })),
                  ]}
                />
              </Col>
              <Col span={6}>
                <Select
                  value={filterStatus}
                  onChange={setFilterStatus}
                  style={{ width: '100%' }}
                  placeholder="Status"
                  options={[
                    { value: 'all', label: 'All Status' },
                    { value: 'active', label: 'Enabled' },
                    { value: 'inactive', label: 'Disable' },
                  ]}
                />
              </Col>
              <Col span={4}>
                <Button type='primary' onClick={loadTemplates} block>
                  Apply Filter
                </Button>
              </Col>
            </Row>
          </Card>

          {/* Template list */}
          {listLoading ? (
            <Card>
              <div className='text-center py-16'>
                <div className='inline-flex items-center justify-center w-16 h-16 bg-blue-50 rounded-full mb-4'>
                  <RefreshCw size={32} className='text-blue-500 animate-spin' />
                </div>
                <Text className='text-gray-500'>Loading template...</Text>
              </div>
            </Card>
          ) : filteredTemplates.length === 0 ? (
            <Card>
              <div className='text-center py-16'>
                <div className='inline-flex items-center justify-center w-24 h-24 bg-gray-50 rounded-full mb-4'>
                  <FileText size={48} className='text-gray-400' />
                </div>
                <Title level={4} className='text-gray-600 mb-2'>

                  No template yet
                </Title>
                <p className='text-gray-500 mb-4'>No matching ticket template found</p>
                {canCreate && (
                  <Button type='primary' onClick={() => setModalVisible(true)}>

                    Create template
                  </Button>
                )}
              </div>
            </Card>
          ) : (
            <div>
              {viewMode === 'grid' ? (
                <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4'>
                  {filteredTemplates.map(renderTemplateCard)}
                </div>
              ) : (
                <div>{filteredTemplates.map(renderTemplateList)}</div>
              )}
            </div>
          )}
        </>
      )}

      {/* Create/Edit template modal */}
      <Modal
        title={editingTemplate ? 'Edit ticket template' : 'Create a new work order template'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={1000}
      >
        <Form
          layout='vertical'
          initialValues={editingTemplate || {}}
          onFinish={async values => {
            if (saving) return;
            setSaving(true);
            try {
              // formFields 是后端原样持久化的表单载荷；表单里收集到的字段必须全部写入，
              // 否则下次打开只会读回默认值（SLA 恒显示 4 hours 这类假象）。
              const formFields: Record<string, unknown> = {
                type: values.type,
                subcategory: values.subcategory,
                estimatedTime: values.estimatedTime,
                sla: values.sla,
                slaType: values.slaType,
                impact: values.impact,
                urgency: values.urgency,
                businessValue: values.businessValue,
                priority: values.priority,
                source: values.source,
                autoAssign: values.autoAssign,
                requiresApproval: values.requiresApproval,
                approvalLevel: values.approvalLevel,
                tags: values.tags || [],
              };
              const payload = {
                name: values.name,
                description: values.description,
                category: values.category,
                priority: values.priority,
                formFields,
                isActive: values.isActive ?? true,
              };
              if (editingTemplate) {
                await TicketApi.updateTemplate(editingTemplate.id, payload);
                message.success('Template updated');
              } else {
                await TicketApi.createTemplate(payload);
                message.success('Template created');
              }
              setModalVisible(false);
              setEditingTemplate(null);
              loadTemplates();
            } catch (err) {
              console.error('Save template failed:', err);
              message.error(`Save failed: ${errorReason(err)}`);
            } finally {
              setSaving(false);
            }
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Template Name"
                name='name'
                rules={[{ required: true, message: 'Please enter template name' }]}
              >
                <Input placeholder="Please enter template name" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="TemplateType"
                name='type'
                rules={[{ required: true, message: 'Please select template type' }]}
              >
                <Select
                  placeholder="Please select template type"
                  options={[
                    { value: 'incident', label: 'Incident' },
                    { value: 'service_request', label: 'Service Request' },
                    { value: 'problem', label: 'Problem' },
                    { value: 'change', label: 'Change' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Category"
                name='category'
                rules={[{ required: true, message: 'Please select a category' }]}
              >
                <Input placeholder="Please enter the category" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="Subcategory" name='subcategory'>
                <Input placeholder="Please enter a subcategory (optional)" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="Description"
            name='description'
            rules={[{ required: true, message: 'Enter template description' }]}
          >
            <TextArea rows={3} placeholder="Please describe the template usage and applicable scenarios in detail" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="Priority"
                name='priority'
                rules={[{ required: true, message: 'Please select priority' }]}
              >
                <Select
                  placeholder="Please select priority"
                  options={[
                    { value: 'low', label: 'Low' },
                    { value: 'medium', label: 'Medium' },
                    { value: 'high', label: 'High' },
                    { value: 'urgent', label: 'Urgent' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Estimated processing time"
                name='estimatedTime'
                rules={[{ required: true, message: 'Please enter the estimated processing time' }]}
              >
                <Input placeholder="For example: 2 hours" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label='SLA' name='sla' rules={[{ required: true, message: 'Please enter SLA' }]}>
                <Input placeholder="For example: 4 hours" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="Impact Range"
                name='impact'
                rules={[{ required: true, message: 'Please select impact scope' }]}
              >
                <Select
                  placeholder="Please select impact scope"
                  options={[
                    { value: 'individual', label: 'Personal' },
                    { value: 'department', label: 'Department' },
                    { value: 'organization', label: 'Organization' },
                    { value: 'customer', label: 'Customer' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Urgency"
                name='urgency'
                rules={[{ required: true, message: 'Please select the level of urgency' }]}
              >
                <Select
                  placeholder="Please select the level of urgency"
                  options={[
                    { value: 'low', label: 'Low' },
                    { value: 'medium', label: 'Medium' },
                    { value: 'high', label: 'High' },
                    { value: 'critical', label: 'Critical' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Business value"
                name='businessValue'
                rules={[{ required: true, message: 'Please select business value' }]}
              >
                <Select
                  placeholder="Please select business value"
                  options={[
                    { value: 'low', label: 'Low' },
                    { value: 'medium', label: 'Medium' },
                    { value: 'high', label: 'High' },
                    { value: 'critical', label: 'Critical' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Source"
                name='source'
                rules={[{ required: true, message: 'Please select source' }]}
              >
                <Select
                  placeholder="Please select source"
                  options={[
                    { value: 'web', label: 'Portal' },
                    { value: 'email', label: 'Email' },
                    { value: 'phone', label: 'Phone' },
                    { value: 'chat', label: 'Online Chat' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="Tags" name='tags'>
                <Select mode='tags' placeholder="Add tag..." style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Divider>Advanced Settings</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item label="Automatically assigned" name='autoAssign' valuePropName='checked'>
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="Approval required" name='requiresApproval' valuePropName='checked'>
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="Template status" name='isActive' valuePropName='checked'>
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="SLA type"
                name='slaType'
                rules={[{ required: true, message: 'Please select SLA type' }]}
              >
                <Select
                  placeholder="Please select SLA type"
                  options={[
                    { value: 'hours', label: 'hours' },
                    { value: 'days', label: 'days' },
                    { value: 'business_hours', label: 'Working hours' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="Approval level" name='approvalLevel'>
                <Select
                  placeholder="Please select the approval level"
                  options={[
                    { value: 'none', label: 'No approval required' },
                    { value: 'manager', label: 'Manager Approval' },
                    { value: 'director', label: 'Director Approval' },
                    { value: 'executive', label: 'Executive Approval' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit' loading={saving}>
                {editingTemplate ? 'Update template' : 'Create template'}
              </Button>
              <Button onClick={() => setModalVisible(false)}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default TicketTemplatesPage;
