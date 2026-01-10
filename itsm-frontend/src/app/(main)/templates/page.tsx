'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Modal,
  Form,
  message,
  Tag,
  Space,
  Row,
  Col,
  Statistic,
  Select,
  Switch,
  Divider,
  Typography,
  Popconfirm,
  Tooltip,
  Badge,
  Collapse,
  InputNumber,
  Checkbox,
  Radio,
  Tabs,
  List,
} from 'antd';
import {
  Plus,
  Edit,
  Delete,
  Copy,
  Eye,
  Search,
  Filter,
  FileText,
  Settings,
  Users,
  Calendar,
  AlertTriangle,
  CheckCircle,
  Clock,
  RefreshCw,
  Download,
  Upload,
  GripVertical,
  Trash2,
} from 'lucide-react';
import { DragDropContext, Droppable, Draggable } from '@hello-pangea/dnd';
import LoadingEmptyError from '@/components/ui/LoadingEmptyError';
import {
  ticketTemplateService,
  type CreateTemplateRequest,
  type UpdateTemplateRequest,
  type TemplateField,
  type TicketTemplate as ServiceTicketTemplate,
} from '@/lib/services/ticket-template-service';

const { TextArea } = Input;
const { Option } = Select;
const { Title, Text } = Typography;
const { Panel } = Collapse;
const { TabPane } = Tabs;

interface WorkflowStep {
  id: string;
  name: string;
  type: 'approval' | 'assignment' | 'notification' | 'automation';
  assignee_group?: string;
  assignee_user?: string;
  due_time?: number;
  conditions?: string;
  order: number;
}

type TicketTemplate = ServiceTicketTemplate & {
  priority?: string;
  form_fields?: Record<string, any> | TemplateField[];
  workflow_steps?: WorkflowStep[];
  tenant_id?: string;
};

type FormField = TemplateField & {
  id: string;
  placeholder?: string;
  order?: number;
};

interface TemplateFilters {
  category?: string;
  is_active?: boolean;
  keyword?: string;
}

const TicketTemplatePage: React.FC = () => {
  const [templates, setTemplates] = useState<TicketTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TicketTemplate | null>(null);
  const [filters, setFilters] = useState<TemplateFilters>({});
  const [form] = Form.useForm();
  const [activeTab, setActiveTab] = useState('basic');

  // 表单字段和工作流步骤状态
  const [formFields, setFormFields] = useState<FormField[]>([]);
  const [workflowSteps, setWorkflowSteps] = useState<WorkflowStep[]>([]);

  // 模拟数据（作为备用）
  const mockTemplates: TicketTemplate[] = [];

  // 加载模板数据
  const loadTemplates = async () => {
    try {
      setLoading(true);
      setError(null);

      // 调用真实API
      const response = await ticketTemplateService.getTemplates({
        page: 1,
        page_size: 100,
      });

      setTemplates(response.templates || []);
    } catch (error) {
      console.error('加载模板失败:', error);
      // 如果API调用失败，使用模拟数据作为备用
      setTemplates(mockTemplates);
      setError('加载模板数据失败，使用本地数据');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTemplates();
  }, []);

  // 处理创建模板
  const handleCreateTemplate = () => {
    setEditingTemplate(null);
    setFormFields([]);
    setWorkflowSteps([]);
    form.resetFields();
    setModalVisible(true);
    setActiveTab('basic');
  };

  // 处理编辑模板
  const handleEditTemplate = (template: TicketTemplate) => {
    setEditingTemplate(template);

    // 解析表单字段和工作流步骤
    try {
      if (template.fields && Array.isArray(template.fields)) {
        const normalized = template.fields.map((field, index) => ({
          ...field,
          id: `field-${template.id}-${index}`,
          order: index,
        }));
        setFormFields(normalized as FormField[]);
      } else if (template.form_fields) {
        const fields = Array.isArray(template.form_fields)
          ? template.form_fields
          : Object.values(template.form_fields);
        const normalized = fields.map((field, index) => ({
          ...(field as TemplateField),
          id: `field-${template.id}-${index}`,
          order: index,
        }));
        setFormFields(normalized as FormField[]);
      }

      if (template.workflow_steps) {
        setWorkflowSteps(template.workflow_steps);
      }
    } catch (error) {
      console.error('解析模板数据失败:', error);
    }

    form.setFieldsValue({
      name: template.name,
      description: template.description,
      category: template.category,
      priority: template.priority,
      assignee_group:
        template.form_fields && !Array.isArray(template.form_fields)
          ? template.form_fields.assignee_group
          : undefined,
      is_active: template.is_active,
    });
    setModalVisible(true);
    setActiveTab('basic');
  };

  // 处理删除模板
  const handleDeleteTemplate = async (id: number) => {
    try {
      await ticketTemplateService.deleteTemplate(id);

      setTemplates(templates.filter(t => t.id !== id));
      message.success('模板删除成功');
    } catch (error) {
      message.error('删除失败');
      console.error('删除失败:', error);
    }
  };

  // 处理复制模板
  const handleCopyTemplate = async (template: TicketTemplate) => {
    try {
      const newName = `${template.name} - 副本`;
      const copiedTemplate = await ticketTemplateService.copyTemplate(template.id, newName);

      setTemplates([copiedTemplate, ...templates]);
      message.success('模板复制成功');
    } catch (error) {
      message.error('复制失败');
      console.error('复制失败:', error);
    }
  };

  // 处理表单提交
  const handleFormSubmit = async (values: any) => {
    try {
      const fields: TemplateField[] = formFields.map(field => ({
        name: field.name,
        label: field.label,
        type: field.type,
        required: field.required,
        options: field.options,
        default_value: field.default_value,
        validation_rules: field.validation_rules,
      }));

      if (editingTemplate) {
        // 更新模板
        const updatePayload: UpdateTemplateRequest = {
          name: values.name,
          description: values.description,
          category: values.category,
          fields,
          is_active: values.is_active,
        };
        const updatedTemplate = await ticketTemplateService.updateTemplate(editingTemplate.id, updatePayload);

        setTemplates(templates.map(t => (t.id === editingTemplate.id ? updatedTemplate : t)));
        message.success('模板更新成功');
      } else {
        // 创建新模板
        const createPayload: CreateTemplateRequest = {
          name: values.name,
          description: values.description,
          category: values.category,
          fields,
        };
        const newTemplate = await ticketTemplateService.createTemplate(createPayload);

        setTemplates([newTemplate, ...templates]);
        message.success('模板创建成功');
      }

      setModalVisible(false);
      setEditingTemplate(null);
      setFormFields([]);
      setWorkflowSteps([]);
      form.resetFields();
    } catch (error) {
      message.error('操作失败');
      console.error('操作失败:', error);
    }
  };

  // 表单字段管理
  const addFormField = () => {
    const newField: FormField = {
      id: `field_${Date.now()}`,
      name: '',
      label: '',
      type: 'text',
      required: false,
      order: formFields.length,
    };
    setFormFields([...formFields, newField]);
  };

  const updateFormField = (index: number, field: Partial<FormField>) => {
    const updatedFields = [...formFields];
    updatedFields[index] = { ...updatedFields[index], ...field };
    setFormFields(updatedFields);
  };

  const removeFormField = (index: number) => {
    setFormFields(formFields.filter((_, i) => i !== index));
  };

  const moveFormField = (fromIndex: number, toIndex: number) => {
    const updatedFields = [...formFields];
    const [movedField] = updatedFields.splice(fromIndex, 1);
    updatedFields.splice(toIndex, 0, movedField);
    // 更新顺序
    updatedFields.forEach((field, index) => {
      field.order = index;
    });
    setFormFields(updatedFields);
  };

  // 工作流步骤管理
  const addWorkflowStep = () => {
    const newStep: WorkflowStep = {
      id: `step_${Date.now()}`,
      name: '',
      type: 'assignment',
      order: workflowSteps.length,
    };
    setWorkflowSteps([...workflowSteps, newStep]);
  };

  const updateWorkflowStep = (index: number, step: Partial<WorkflowStep>) => {
    const updatedSteps = [...workflowSteps];
    updatedSteps[index] = { ...updatedSteps[index], ...step };
    setWorkflowSteps(updatedSteps);
  };

  const removeWorkflowStep = (index: number) => {
    setWorkflowSteps(workflowSteps.filter((_, i) => i !== index));
  };

  const moveWorkflowStep = (fromIndex: number, toIndex: number) => {
    const updatedSteps = [...workflowSteps];
    const [movedStep] = updatedSteps.splice(fromIndex, 1);
    updatedSteps.splice(toIndex, 0, movedStep);
    // 更新顺序
    updatedSteps.forEach((step, index) => {
      step.order = index;
    });
    setWorkflowSteps(updatedSteps);
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: 'green',
      inactive: 'red',
    };
    return colors[status] || 'default';
  };

  // 获取优先级颜色
  const getPriorityColor = (priority: string) => {
    const colors: Record<string, string> = {
      low: 'blue',
      medium: 'orange',
      high: 'red',
      critical: 'red',
    };
    return colors[priority] || 'default';
  };

  // 表格列定义
  const columns = [
    {
      title: '模板名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: TicketTemplate) => (
        <div>
          <div className='font-medium text-gray-900'>{name}</div>
          <div className='text-sm text-gray-500'>{record.description}</div>
        </div>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 120,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority?: string) =>
        priority ? (
          <Tag color={getPriorityColor(priority)}>
            {priority === 'low' && '低'}
            {priority === 'medium' && '中'}
            {priority === 'high' && '高'}
            {priority === 'critical' && '紧急'}
          </Tag>
        ) : (
          <Tag>-</Tag>
        ),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 100,
      render: (isActive: boolean) => (
        <Badge status={isActive ? 'success' : 'error'} text={isActive ? '启用' : '禁用'} />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 120,
      render: (date: string) => new Date(date).toLocaleDateString('zh-CN'),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 120,
      render: (date: string) => new Date(date).toLocaleDateString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (record: TicketTemplate) => (
        <Space size='small'>
          <Tooltip title='查看详情'>
            <Button type='text' icon={<Eye className='w-4 h-4' />} size='small' />
          </Tooltip>
          <Tooltip title='编辑模板'>
            <Button
              type='text'
              icon={<Edit className='w-4 h-4' />}
              size='small'
              onClick={() => handleEditTemplate(record)}
            />
          </Tooltip>
          <Tooltip title='复制模板'>
            <Button
              type='text'
              icon={<Copy className='w-4 h-4' />}
              size='small'
              onClick={() => handleCopyTemplate(record)}
            />
          </Tooltip>
          <Tooltip title='删除模板'>
            <Popconfirm
              title='确定要删除这个模板吗？'
              onConfirm={() => handleDeleteTemplate(record.id)}
              okText='确定'
              cancelText='取消'
            >
              <Button type='text' icon={<Delete className='w-4 h-4' />} size='small' danger />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  // 过滤模板
  const filteredTemplates = templates.filter(template => {
    if (filters.category && template.category !== filters.category) return false;
    if (filters.is_active !== undefined && template.is_active !== filters.is_active) return false;
    if (filters.keyword && !template.name.toLowerCase().includes(filters.keyword.toLowerCase()))
      return false;
    return true;
  });

  // 统计数据
  const stats = {
    total: templates.length,
    active: templates.filter(t => t.is_active).length,
    categories: new Set(templates.map(t => t.category)).size,
  };

  // 确定当前状态
  const getCurrentState = () => {
    if (loading) return 'loading';
    if (error) return 'error';
    if (filteredTemplates.length === 0) return 'empty';
    return 'success';
  };

  return (
    <div className='p-6 space-y-6'>
      {/* 页面标题 */}
      <div className='flex items-center justify-between'>
        <div>
          <Title level={2} className='mb-2'>
            工单模板管理
          </Title>
          <Text type='secondary'>管理和配置工单模板，提高工单创建效率</Text>
        </div>
        <Button type='primary' icon={<Plus className='w-4 h-4' />} onClick={handleCreateTemplate}>
          创建模板
        </Button>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className='shadow-sm'>
            <Statistic
              title='总模板数'
              value={stats.total}
              prefix={<FileText className='w-5 h-5' />}
              valueStyle={{ color: '#3b82f6' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='shadow-sm'>
            <Statistic
              title='启用模板'
              value={stats.active}
              prefix={<CheckCircle className='w-5 h-5' />}
              valueStyle={{ color: '#10b981' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='shadow-sm'>
            <Statistic
              title='模板分类'
              value={stats.categories}
              prefix={<Settings className='w-5 h-5' />}
              valueStyle={{ color: '#f59e0b' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选和操作栏 */}
      <Card className='shadow-sm'>
        <Row gutter={[16, 16]} align='middle'>
          <Col flex='auto'>
            <Space size='middle'>
              <Input
                placeholder='搜索模板...'
                prefix={<Search className='w-4 h-4' />}
                style={{ width: 300 }}
                value={filters.keyword}
                onChange={e => setFilters({ ...filters, keyword: e.target.value })}
              />
              <Select
                placeholder='分类筛选'
                style={{ width: 150 }}
                allowClear
                value={filters.category}
                onChange={value => setFilters({ ...filters, category: value })}
              >
                <Option value='系统问题'>系统问题</Option>
                <Option value='用户管理'>用户管理</Option>
                <Option value='硬件问题'>硬件问题</Option>
                <Option value='网络问题'>网络问题</Option>
              </Select>
              <Select
                placeholder='状态筛选'
                style={{ width: 120 }}
                allowClear
                value={filters.is_active}
                onChange={value => setFilters({ ...filters, is_active: value })}
              >
                <Option value={true}>启用</Option>
                <Option value={false}>禁用</Option>
              </Select>
            </Space>
          </Col>
          <Col>
            <Space>
              <Button icon={<RefreshCw className='w-4 h-4' />} onClick={loadTemplates}>
                刷新
              </Button>
              <Button icon={<Download className='w-4 h-4' />}>导出</Button>
              <Button icon={<Upload className='w-4 h-4' />}>导入</Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 模板列表 */}
      <LoadingEmptyError
        state={getCurrentState()}
        loadingText='正在加载模板数据...'
        empty={{
          title: '暂无模板',
          description: '当前没有找到匹配的工单模板',
          actionText: '创建模板',
          onAction: handleCreateTemplate,
          icon: <FileText className='w-10 h-10' />,
        }}
        error={{
          title: '加载失败',
          description: error || '无法获取模板数据，请稍后重试',
          actionText: '重试',
          onAction: loadTemplates,
        }}
      >
        <Card className='shadow-sm'>
          <Table
            columns={columns}
            dataSource={filteredTemplates}
            rowKey='id'
            pagination={{
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            }}
            scroll={{ x: 1200 }}
          />
        </Card>
      </LoadingEmptyError>

      {/* 创建/编辑模板模态框 */}
      <Modal
        title={editingTemplate ? '编辑模板' : '创建模板'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={1000}
        destroyOnHidden
      >
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane tab='基本信息' key='basic'>
            <Form
              form={form}
              layout='vertical'
              onFinish={handleFormSubmit}
              initialValues={{
                priority: 'medium',
                impact: 'medium',
                urgency: 'medium',
                is_active: true,
                sla_response_time: 60,
                sla_resolution_time: 240,
              }}
            >
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name='name'
                    label='模板名称'
                    rules={[{ required: true, message: '请输入模板名称' }]}
                  >
                    <Input placeholder='请输入模板名称' />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name='category'
                    label='分类'
                    rules={[{ required: true, message: '请选择分类' }]}
                  >
                    <Select placeholder='选择分类'>
                      <Option value='系统问题'>系统问题</Option>
                      <Option value='用户管理'>用户管理</Option>
                      <Option value='硬件问题'>硬件问题</Option>
                      <Option value='网络问题'>网络问题</Option>
                      <Option value='软件问题'>软件问题</Option>
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item
                name='description'
                label='描述'
                rules={[{ required: true, message: '请输入模板描述' }]}
              >
                <TextArea rows={3} placeholder='请输入模板描述' />
              </Form.Item>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name='priority'
                    label='默认优先级'
                    rules={[{ required: true, message: '请选择优先级' }]}
                  >
                    <Select placeholder='选择优先级'>
                      <Option value='low'>低</Option>
                      <Option value='medium'>中</Option>
                      <Option value='high'>高</Option>
                      <Option value='critical'>紧急</Option>
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name='assignee_group' label='默认处理组'>
                    <Select placeholder='选择处理组' allowClear>
                      <Option value='一线支持'>一线支持</Option>
                      <Option value='系统维护组'>系统维护组</Option>
                      <Option value='用户管理组'>用户管理组</Option>
                      <Option value='网络维护组'>网络维护组</Option>
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item name='is_active' label='启用状态' valuePropName='checked'>
                <Switch />
              </Form.Item>

              <Form.Item>
                <Space>
                  <Button type='primary' htmlType='submit'>
                    {editingTemplate ? '更新' : '创建'}
                  </Button>
                  <Button onClick={() => setModalVisible(false)}>取消</Button>
                </Space>
              </Form.Item>
            </Form>
          </TabPane>

          <TabPane tab='表单字段' key='fields'>
            <div className='space-y-4'>
              <div className='flex items-center justify-between'>
                <h3 className='text-lg font-medium'>表单字段配置</h3>
                <Button type='primary' icon={<Plus className='w-4 h-4' />} onClick={addFormField}>
                  添加字段
                </Button>
              </div>

              {formFields.length === 0 ? (
                <div className='text-center py-8 text-gray-500'>
                  暂无表单字段，点击"添加字段"开始配置
                </div>
              ) : (
                <DragDropContext
                  onDragEnd={result => {
                    if (result.destination) {
                      moveFormField(result.source.index, result.destination.index);
                    }
                  }}
                >
                  <Droppable droppableId='form-fields'>
                    {provided => (
                      <div {...provided.droppableProps} ref={provided.innerRef}>
                        {formFields.map((field, index) => (
                          <Draggable key={field.id} draggableId={field.id} index={index}>
                            {provided => (
                              <div
                                ref={provided.innerRef}
                                {...provided.draggableProps}
                                className='border rounded-lg p-4 mb-3 bg-gray-50'
                              >
                                <div className='flex items-center space-x-3'>
                                  <div {...provided.dragHandleProps} className='cursor-move'>
                                    <GripVertical className='w-4 h-4 text-gray-400' />
                                  </div>

                                  <div className='flex-1 grid grid-cols-2 gap-3'>
                                    <Input
                                      placeholder='字段名称'
                                      value={field.name}
                                      onChange={e =>
                                        updateFormField(index, {
                                          name: e.target.value,
                                        })
                                      }
                                    />
                                    <Input
                                      placeholder='显示标签'
                                      value={field.label}
                                      onChange={e =>
                                        updateFormField(index, {
                                          label: e.target.value,
                                        })
                                      }
                                    />
                                    <Select
                                      placeholder='字段类型'
                                      value={field.type}
                                      onChange={value => updateFormField(index, { type: value })}
                                    >
                                      <Option value='text'>文本</Option>
                                      <Option value='textarea'>多行文本</Option>
                                      <Option value='select'>下拉选择</Option>
                                      <Option value='number'>数字</Option>
                                      <Option value='date'>日期</Option>
                                      <Option value='checkbox'>复选框</Option>
                                      <Option value='radio'>单选框</Option>
                                    </Select>
                                    <Input
                                      placeholder='默认值'
                                      value={field.default_value || ''}
                                      onChange={e =>
                                        updateFormField(index, {
                                          default_value: e.target.value,
                                        })
                                      }
                                    />
                                  </div>

                                  <div className='flex items-center space-x-2'>
                                    <Checkbox
                                      checked={field.required}
                                      onChange={e =>
                                        updateFormField(index, {
                                          required: e.target.checked,
                                        })
                                      }
                                    >
                                      必填
                                    </Checkbox>
                                    <Button
                                      type='text'
                                      icon={<Trash2 className='w-4 h-4' />}
                                      danger
                                      onClick={() => removeFormField(index)}
                                    />
                                  </div>
                                </div>

                                {field.type === 'select' && (
                                  <div className='mt-3'>
                                    <Input
                                      placeholder='选项值，用逗号分隔'
                                      value={field.options?.join(', ') || ''}
                                      onChange={e =>
                                        updateFormField(index, {
                                          options: e.target.value
                                            .split(',')
                                            .map(s => s.trim())
                                            .filter(s => s),
                                        })
                                      }
                                    />
                                  </div>
                                )}
                              </div>
                            )}
                          </Draggable>
                        ))}
                        {provided.placeholder}
                      </div>
                    )}
                  </Droppable>
                </DragDropContext>
              )}
            </div>
          </TabPane>

          <TabPane tab='工作流' key='workflow'>
            <div className='space-y-4'>
              <div className='flex items-center justify-between'>
                <h3 className='text-lg font-medium'>工作流步骤配置</h3>
                <Button
                  type='primary'
                  icon={<Plus className='w-4 h-4' />}
                  onClick={addWorkflowStep}
                >
                  添加步骤
                </Button>
              </div>

              {workflowSteps.length === 0 ? (
                <div className='text-center py-8 text-gray-500'>
                  暂无工作流步骤，点击"添加步骤"开始配置
                </div>
              ) : (
                <DragDropContext
                  onDragEnd={result => {
                    if (result.destination) {
                      moveWorkflowStep(result.source.index, result.destination.index);
                    }
                  }}
                >
                  <Droppable droppableId='workflow-steps'>
                    {provided => (
                      <div {...provided.droppableProps} ref={provided.innerRef}>
                        {workflowSteps.map((step, index) => (
                          <Draggable key={step.id} draggableId={step.id} index={index}>
                            {provided => (
                              <div
                                ref={provided.innerRef}
                                {...provided.draggableProps}
                                className='border rounded-lg p-4 mb-3 bg-gray-50'
                              >
                                <div className='flex items-center space-x-3'>
                                  <div {...provided.dragHandleProps} className='cursor-move'>
                                    <GripVertical className='w-4 h-4 text-gray-400' />
                                  </div>

                                  <div className='flex-1 grid grid-cols-2 gap-3'>
                                    <Input
                                      placeholder='步骤名称'
                                      value={step.name}
                                      onChange={e =>
                                        updateWorkflowStep(index, {
                                          name: e.target.value,
                                        })
                                      }
                                    />
                                    <Select
                                      placeholder='步骤类型'
                                      value={step.type}
                                      onChange={value =>
                                        updateWorkflowStep(index, {
                                          type: value,
                                        })
                                      }
                                    >
                                      <Option value='approval'>审批</Option>
                                      <Option value='assignment'>分配</Option>
                                      <Option value='notification'>通知</Option>
                                      <Option value='automation'>自动化</Option>
                                    </Select>
                                    <Input
                                      placeholder='处理组'
                                      value={step.assignee_group || ''}
                                      onChange={e =>
                                        updateWorkflowStep(index, {
                                          assignee_group: e.target.value,
                                        })
                                      }
                                    />
                                    <InputNumber
                                      placeholder='预计耗时(小时)'
                                      value={step.due_time || undefined}
                                      onChange={value =>
                                        updateWorkflowStep(index, {
                                          due_time: value || undefined,
                                        })
                                      }
                                      min={0}
                                    />
                                  </div>

                                  <div className='flex items-center space-x-2'>
                                    <Button
                                      type='text'
                                      icon={<Trash2 className='w-4 h-4' />}
                                      danger
                                      onClick={() => removeWorkflowStep(index)}
                                    />
                                  </div>
                                </div>

                                <div className='mt-3'>
                                  <Input
                                    placeholder='执行条件（可选）'
                                    value={step.conditions || ''}
                                    onChange={e =>
                                      updateWorkflowStep(index, {
                                        conditions: e.target.value,
                                      })
                                    }
                                  />
                                </div>
                              </div>
                            )}
                          </Draggable>
                        ))}
                        {provided.placeholder}
                      </div>
                    )}
                  </Droppable>
                </DragDropContext>
              )}
            </div>
          </TabPane>
        </Tabs>
      </Modal>
    </div>
  );
};

export default TicketTemplatePage;
