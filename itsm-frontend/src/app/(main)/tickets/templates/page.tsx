'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import {
  Card,
  Button,
  Table,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  message,
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
    label: '事件管理',
    icon: <AlertTriangle size={16} />,
    color: 'red',
  },
  {
    key: 'serviceRequest',
    label: '服务请求',
    icon: <FileText size={16} />,
    color: 'blue',
  },
  {
    key: 'problem',
    label: '问题管理',
    icon: <BookOpen size={16} />,
    color: 'orange',
  },
  {
    key: 'change',
    label: '变更管理',
    icon: <Workflow size={16} />,
    color: 'purple',
  },
];

const TicketTemplatesPage = () => {
  const router = useRouter();
  const [templates, setTemplates] = useState<TicketTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TicketTemplate | null>(null);
  const [selectedCategory, setSelectedCategory] = useState('incident');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterStatus, setFilterStatus] = useState('all');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  const loadTemplates = useCallback(async () => {
    setLoading(true);
    try {
      // 调用实际API
      const response = await TicketApi.getTemplates({
        page: 1,
        pageSize: 100,
        category: selectedCategory !== 'all' ? selectedCategory : undefined,
      });

      // 将API响应转换为组件期望的格式
      const apiTemplates: TicketTemplate[] = (response.items || []).map(
        (item: {
          id: number;
          name: string;
          description: string;
          category: string;
          content?: Record<string, unknown>;
          isActive?: boolean;
          createdAt?: string;
          updatedAt?: string;
        }) => ({
          id: item.id,
          name: item.name,
          description: item.description,
          type: (item.content?.type as string) || item.category?.toLowerCase() || 'incident',
          category: item.category,
          subcategory: (item.content?.subcategory as string) || undefined,
          priority: (item.content?.priority as string) || 'medium',
          estimatedTime: (item.content?.estimatedTime as string) || '1 hour',
          sla: (item.content?.sla as string) || '4 hours',
          slaType: (item.content?.slaType as 'hours' | 'days' | 'business_hours') || 'hours',
          impact: (item.content?.impact as string) || 'individual',
          urgency: (item.content?.urgency as string) || 'medium',
          businessValue: (item.content?.businessValue as string) || 'medium',
          source: (item.content?.source as string) || 'web',
          assigneeGroup: (item.content?.assigneeGroup as string) || undefined,
          autoAssign: (item.content?.autoAssign as boolean) || false,
          requiresApproval: (item.content?.requiresApproval as boolean) || false,
          approvalLevel: (item.content?.approvalLevel as string) || 'none',
          customFields: (item.content?.customFields as CustomField[]) || [],
          tags: (item.content?.tags as string[]) || [],
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
        })
      );

      setTemplates(apiTemplates);
    } catch (error) {
      console.error('Failed to load templates:', error);
      message.error('加载模板列表失败，请稍后重试');
      setTemplates([]);
    } finally {
      setLoading(false);
    }
  }, [selectedCategory]);

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
      message.success('模板删除成功');
    } catch (error) {
      console.error('Delete template failed:', error);
      message.error('删除失败，请稍后重试');
    }
  };

  const handleCopyTemplate = async (template: TicketTemplate) => {
    try {
      const formFields: Record<string, unknown> = {
        type: template.type,
        priority: template.priority,
        source: template.source,
        autoAssign: template.autoAssign,
        requiresApproval: template.requiresApproval,
        slaType: template.slaType,
        approvalLevel: template.approvalLevel,
        tags: template.tags,
      };
      await TicketApi.createTemplate({
        name: `${template.name} - Copy`,
        description: template.description,
        category: template.category,
        priority: template.priority,
        formFields,
        isActive: template.isActive,
      });
      message.success('模板复制成功');
      loadTemplates();
    } catch (error) {
      console.error('Copy template failed:', error);
      message.error('复制失败，请稍后重试');
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

  const renderTemplateCard = (template: TicketTemplate) => (
    <Card
      key={template.id}
      hoverable
      className="h-full"
      actions={[
        <Tooltip title="查看模板" key="view">
          <Button
            type="text"
            icon={<Eye size={16} />}
            onClick={() => router.push(`/tickets/templates/${template.id}`)}
            aria-label="查看模板详情"
          />
        </Tooltip>,
        <Tooltip title="编辑模板" key="edit">
          <Button
            type="text"
            icon={<Edit size={16} />}
            onClick={() => handleEditTemplate(template)}
            aria-label="编辑模板"
          />
        </Tooltip>,
        <Tooltip title="复制模板" key="copy">
          <Button
            type="text"
            icon={<Copy size={16} />}
            onClick={() => handleCopyTemplate(template)}
            aria-label="复制模板"
          />
        </Tooltip>,
        <Tooltip title="删除模板" key="delete">
          <Popconfirm
            title="确定删除这个模板吗？"
            onConfirm={() => handleDeleteTemplate(template.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="text" danger icon={<Delete size={16} />} aria-label="删除模板" />
          </Popconfirm>
        </Tooltip>,
      ]}
    >
      <div className="flex items-start mb-3">
        <div
          className={`inline-flex items-center justify-center w-12 h-12 bg-${template.color}-50 rounded-lg mr-3`}
        >
          <span className={`text-${template.color}-500`}>{template.icon}</span>
        </div>
        <div className="flex-1 min-w-0">
          <Title level={5} className="mb-1 truncate">
            {template.name}
          </Title>
          <Text type="secondary" className="text-sm line-clamp-2">
            {template.description}
          </Text>
        </div>
      </div>

      <div className="space-y-2 mb-4">
        <div className="flex items-center justify-between">
          <Text type="secondary" className="text-xs">
            Type
          </Text>
          <Tag color={template.color}>{template.category}</Tag>
        </div>
        <div className="flex items-center justify-between">
          <Text type="secondary" className="text-xs">
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
        <div className="flex items-center justify-between">
          <Text type="secondary" className="text-xs">
            SLA
          </Text>
          <Text className="text-xs">{template.sla}</Text>
        </div>
      </div>

      <div className="flex items-center justify-between text-xs text-gray-500 mb-3">
        <span>创建时间: {new Date(template.createdAt).toLocaleDateString('zh-CN')}</span>
        <span>更新时间: {new Date(template.updatedAt).toLocaleDateString('zh-CN')}</span>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Switch checked={template.isActive} size="small" />
          <Text className="text-xs">{template.isActive ? '启用' : '停用'}</Text>
        </div>
      </div>
    </Card>
  );

  const renderTemplateList = (template: TicketTemplate) => (
    <Card key={template.id} className="mb-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <div
            className={`inline-flex items-center justify-center w-10 h-10 bg-${template.color}-50 rounded-lg`}
          >
            <span className={`text-${template.color}-500`}>{template.icon}</span>
          </div>
          <div>
            <Title level={5} className="mb-1">
              {template.name}
            </Title>
            <Text type="secondary" className="text-sm">
              {template.description}
            </Text>
          </div>
        </div>

        <div className="flex items-center space-x-4">
          <div className="text-center">
            <Text className="text-xs text-gray-500">更新时间</Text>
            <div className="font-semibold">
              {new Date(template.updatedAt).toLocaleDateString('zh-CN')}
            </div>
          </div>
          <div className="flex items-center space-x-2">
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
          <Space>
            <Button
              type="text"
              icon={<Eye size={16} />}
              onClick={() => router.push(`/tickets/templates/${template.id}`)}
              aria-label="查看模板详情"
            />
            <Button
              type="text"
              icon={<Edit size={16} />}
              onClick={() => handleEditTemplate(template)}
              aria-label="编辑模板"
            />
            <Button
              type="text"
              icon={<Copy size={16} />}
              onClick={() => handleCopyTemplate(template)}
              aria-label="复制模板"
            />
            <Popconfirm
              title="确定删除这个模板吗？"
              onConfirm={() => handleDeleteTemplate(template.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button type="text" danger icon={<Delete size={16} />} aria-label="删除模板" />
            </Popconfirm>
          </Space>
        </div>
      </div>
    </Card>
  );

  return (
    <>
      {/* Page header actions */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">工单模板管理</h1>
          <p className="text-gray-600 mt-1">
            管理并配置工单模板，提升工单创建效率
          </p>
        </div>
        <Space>
          <Button icon={<RefreshCw size={16} />} onClick={loadTemplates} aria-label="刷新模板列表">
            刷新
          </Button>
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={handleCreateTemplate}
            aria-label="新建模板"
          >
            新建模板
          </Button>
        </Space>
      </div>
      {/* Statistics */}
      <Row gutter={16} className="mb-6">
        <Col span={8}>
          <Card>
            <Statistic
              title="模板总数"
              value={templates.length}
              prefix={<FileText size={16} style={{ color: '#3b82f6' }} />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="启用模板"
              value={templates.filter(t => t.isActive).length}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<CheckCircle size={16} />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="分类数"
              value={new Set(templates.map(t => t.category).filter(Boolean)).size}
              prefix={<TagIcon size={16} style={{ color: '#faad14' }} />}
            />
          </Card>
        </Col>
      </Row>

      {/* Filter and search */}
      <Card title="模板管理" className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <Title level={5} className="mb-0">
            筛选条件
          </Title>
          <Space>
            <Radio.Group
              value={viewMode}
              onChange={(e: RadioChangeEvent) => setViewMode(e.target.value)}
            >
              <Radio.Button value="grid">卡片视图</Radio.Button>
              <Radio.Button value="list">列表视图</Radio.Button>
            </Radio.Group>
          </Space>
        </div>

        <Row gutter={16} align="middle">
          <Col span={8}>
            <Input.Search
              placeholder="搜索模板..."
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
              placeholder="选择分类"
              options={[
                { value: 'all', label: '全部分类' },
                ...templateCategories.map(cat => ({
                  value: cat.key,
                  label: <div className="flex items-center"><span className={`text-${cat.color}-500 mr-2`}>{cat.icon}</span>{cat.label}</div>,
                })),
              ]}
            />
          </Col>
          <Col span={6}>
            <Select
              value={filterStatus}
              onChange={setFilterStatus}
              style={{ width: '100%' }}
              placeholder="状态筛选"
              options={[
                { value: 'all', label: '全部状态' },
                { value: 'active', label: '启用' },
                { value: 'inactive', label: '停用' },
              ]}
            />
          </Col>
          <Col span={4}>
            <Button type="primary" onClick={loadTemplates} block>
              Apply Filter
            </Button>
          </Col>
        </Row>
      </Card>

      {/* Template list */}
      {loading ? (
        <Card>
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-50 rounded-full mb-4">
              <RefreshCw size={32} className="text-blue-500 animate-spin" />
            </div>
            <Text className="text-gray-500">加载模板中...</Text>
          </div>
        </Card>
      ) : filteredTemplates.length === 0 ? (
        <Card>
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-24 h-24 bg-gray-50 rounded-full mb-4">
              <FileText size={48} className="text-gray-400" />
            </div>
            <Title level={4} className="text-gray-600 mb-2">
              暂无模板
            </Title>
            <p className="text-gray-500 mb-4">未找到匹配的工单模板</p>
            <Button type="primary" onClick={() => setModalVisible(true)}>
              创建第一个模板
            </Button>
          </div>
        </Card>
      ) : (
        <div>
          {viewMode === 'grid' ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {filteredTemplates.map(renderTemplateCard)}
            </div>
          ) : (
            <div>{filteredTemplates.map(renderTemplateList)}</div>
          )}
        </div>
      )}

      {/* Create/Edit template modal */}
      <Modal
        title={editingTemplate ? '编辑工单模板' : '新建工单模板'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={1000}
      >
        <Form
          layout="vertical"
          initialValues={editingTemplate || {}}
          onFinish={async values => {
            try {
              // 把表单字段映射到后端 dto.TicketTemplate
              const formFields: Record<string, unknown> = {
                type: values.type,
                priority: values.priority,
                source: values.source,
                autoAssign: values.autoAssign,
                requiresApproval: values.requiresApproval,
                slaType: values.slaType,
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
                message.success('模板更新成功');
              } else {
                await TicketApi.createTemplate(payload);
                message.success('模板创建成功');
              }
              setModalVisible(false);
              setEditingTemplate(null);
              loadTemplates();
            } catch (err) {
              console.error('Save template failed:', err);
              message.error('保存失败，请稍后重试');
            }
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="模板名称"
                name="name"
                rules={[{ required: true, message: '请输入模板名称' }]}
              >
                <Input placeholder="请输入模板名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="模板类型"
                name="type"
                rules={[{ required: true, message: '请选择模板类型' }]}
              >
                <Select placeholder="请选择模板类型" options={[
                  { value: 'incident', label: '事件' },
                  { value: 'service_request', label: '服务请求' },
                  { value: 'problem', label: '问题' },
                  { value: 'change', label: '变更' },
                ]} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="分类"
                name="category"
                rules={[{ required: true, message: '请选择分类' }]}
              >
                <Input placeholder="请输入分类" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="子分类" name="subcategory">
                <Input placeholder="请输入子分类（可选）" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="描述"
            name="description"
            rules={[{ required: true, message: '请输入模板描述' }]}
          >
            <TextArea
              rows={3}
              placeholder="请详细描述模板用途与适用场景"
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="优先级"
                name="priority"
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select placeholder="请选择优先级" options={[
                  { value: 'low', label: '低' },
                  { value: 'medium', label: '中' },
                  { value: 'high', label: '高' },
                  { value: 'urgent', label: '紧急' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="预计处理时长"
                name="estimatedTime"
                rules={[{ required: true, message: '请输入预计处理时长' }]}
              >
                <Input placeholder="如：2 小时" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="SLA"
                name="sla"
                rules={[{ required: true, message: '请输入 SLA' }]}
              >
                <Input placeholder="如：4 小时" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="影响范围"
                name="impact"
                rules={[{ required: true, message: '请选择影响范围' }]}
              >
                <Select placeholder="请选择影响范围" options={[
                  { value: 'individual', label: '个人' },
                  { value: 'department', label: '部门' },
                  { value: 'organization', label: '组织' },
                  { value: 'customer', label: '客户' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="紧急程度"
                name="urgency"
                rules={[{ required: true, message: '请选择紧急程度' }]}
              >
                <Select placeholder="请选择紧急程度" options={[
                  { value: 'low', label: '低' },
                  { value: 'medium', label: '中' },
                  { value: 'high', label: '高' },
                  { value: 'critical', label: '严重' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="业务价值"
                name="businessValue"
                rules={[{ required: true, message: '请选择业务价值' }]}
              >
                <Select placeholder="请选择业务价值" options={[
                  { value: 'low', label: '低' },
                  { value: 'medium', label: '中' },
                  { value: 'high', label: '高' },
                  { value: 'critical', label: '严重' },
                ]} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="来源"
                name="source"
                rules={[{ required: true, message: '请选择来源' }]}
              >
                <Select placeholder="请选择来源" options={[
                  { value: 'web', label: '门户' },
                  { value: 'email', label: '邮件' },
                  { value: 'phone', label: '电话' },
                  { value: 'chat', label: '在线聊天' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="标签" name="tags">
                <Select mode="tags" placeholder="添加标签..." style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Divider>高级设置</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item label="自动指派" name="autoAssign" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="需要审批" name="requiresApproval" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="模板状态" name="isActive" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="SLA 类型"
                name="slaType"
                rules={[{ required: true, message: '请选择 SLA 类型' }]}
              >
                <Select placeholder="请选择 SLA 类型" options={[
                  { value: 'hours', label: '小时' },
                  { value: 'days', label: '天' },
                  { value: 'business_hours', label: '工作时间' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="审批层级" name="approvalLevel">
                <Select placeholder="请选择审批层级" options={[
                  { value: 'none', label: '无需审批' },
                  { value: 'manager', label: '经理审批' },
                  { value: 'director', label: '总监审批' },
                  { value: 'executive', label: '高管审批' },
                ]} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingTemplate ? '更新模板' : '创建模板'}
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default TicketTemplatesPage;
