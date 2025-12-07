'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  message,
  Space,
  Typography,
  Row,
  Col,
  Divider,
  Alert,
  Tag,
  Steps,
  Collapse,
  Upload,
  Spin,
} from 'antd';
import {
  FileText,
  Paperclip,
  Send,
  Bot,
  Lightbulb,
  Workflow,
  CheckCircle,
  Tag as TagIcon,
  ArrowLeft,
  Upload as UploadIcon,
} from 'lucide-react';
import { AIWorkflowAssistant } from '@/components/business/AIWorkflowAssistant';
import { ticketService, TicketPriority, TicketType } from '@/lib/services/ticket-service';
import { ticketCategoryService } from '@/lib/services/ticket-category-service';
import { ticketTemplateService } from '@/lib/services/ticket-template-service';
import { ticketTagService } from '@/lib/services/ticket-tag-service';
import type { UploadFile } from 'antd';
import { httpClient } from '@/lib/api/http-client';

const { TextArea } = Input;
const { Option } = Select;
const { Title, Text } = Typography;
const { Step } = Steps;
const { Panel } = Collapse;

interface CreateTicketForm {
  title: string;
  description: string;
  priority: TicketPriority;
  type: TicketType;
  category: string;
  category_id?: number;
  template_id?: number;
  subcategory?: string;
  tags: string[];
  tag_ids?: number[];
  attachments: File[];
}

export default function CreateTicketPage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);
  const [aiSuggestion, setAiSuggestion] = useState<any>(null);
  const [formData, setFormData] = useState<Partial<CreateTicketForm>>({});

  // API数据状态
  const [categories, setCategories] = useState<any[]>([]);
  const [templates, setTemplates] = useState<any[]>([]);
  const [tags, setTags] = useState<any[]>([]);
  const [loadingCategories, setLoadingCategories] = useState(false);
  const [loadingTemplates, setLoadingTemplates] = useState(false);
  const [loadingTags, setLoadingTags] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const DEFAULT_CATEGORIES = [
    { id: -1, name: '网络' },
    { id: -2, name: '性能' },
    { id: -3, name: '安全' },
    { id: -4, name: '存储' },
    { id: -5, name: '连接' },
  ];

  const steps = [
    {
      title: '基本信息',
      icon: <FileText />,
      content: '填写基本工单信息和描述',
    },
    {
      title: 'AI 分析',
      icon: <Bot />,
      content: 'AI 分析工单内容并提供建议',
    },
    {
      title: '确认创建',
      icon: <CheckCircle />,
      content: '确认工单信息并创建',
    },
  ];

  // 加载API数据
  useEffect(() => {
    const loadData = async () => {
      // 加载分类
      setLoadingCategories(true);
      try {
        const categoriesData = await ticketCategoryService.listCategories({
          page: 1,
          page_size: 100,
          is_active: true,
        });
        const list = categoriesData.categories || [];
        setCategories(Array.isArray(list) && list.length > 0 ? list : DEFAULT_CATEGORIES);
        if (!list || list.length === 0) {
          message.info('分类列表为空，已使用默认分类');
        }
      } catch (error) {
        console.error('Failed to load categories:', error);
        setCategories(DEFAULT_CATEGORIES);
        message.warning('加载分类失败，使用默认分类');
      } finally {
        setLoadingCategories(false);
      }

      // 加载模板
      setLoadingTemplates(true);
      try {
        const templatesData = await ticketTemplateService.getTemplates({ page: 1, page_size: 100 });
        // 处理不同的响应格式
        if (Array.isArray(templatesData)) {
          setTemplates(templatesData);
        } else if (
          templatesData &&
          typeof templatesData === 'object' &&
          'templates' in templatesData
        ) {
          setTemplates((templatesData as any).templates || []);
        } else if (templatesData && typeof templatesData === 'object' && 'data' in templatesData) {
          const data = (templatesData as any).data;
          setTemplates(Array.isArray(data) ? data : []);
        } else {
          setTemplates([]);
        }
      } catch (error) {
        console.error('Failed to load templates:', error);
        message.warning('加载模板失败');
      } finally {
        setLoadingTemplates(false);
      }

      // 加载标签
      setLoadingTags(true);
      try {
        const tagsData = await ticketTagService.listTags({
          page: 1,
          page_size: 100,
          is_active: true,
        });
        setTags(tagsData.tags || []);
      } catch (error) {
        console.error('Failed to load tags:', error);
        message.warning('加载标签失败');
      } finally {
        setLoadingTags(false);
      }
    };

    loadData();
  }, []);

  useEffect(() => {
    if (!loadingCategories && categories && categories.length > 0) {
      const current = form.getFieldValue('category_id');
      if (!current) {
        const first = categories[0];
        form.setFieldsValue({ category_id: first.id, category: first.name });
      }
    }
  }, [loadingCategories, categories, form]);

  const handleAISuggestion = (suggestion: unknown) => {
    setAiSuggestion(suggestion);

    // Auto-fill form
    form.setFieldsValue({
      priority: suggestion.priority,
      category: suggestion.category,
      tags: [suggestion.category, suggestion.priority],
    });

    setFormData(prev => ({
      ...prev,
      priority: suggestion.priority,
      category: suggestion.category,
      tags: [suggestion.category, suggestion.priority],
    }));

    message.success('AI 建议已应用，请查看并确认');
  };

  const handleFormChange = (changedValues: unknown, allValues: unknown) => {
    setFormData(allValues);
  };

  const handleSubmit = async (values: CreateTicketForm) => {
    setLoading(true);
    try {
      if (!values.category_id && categories && categories.length > 0) {
        values.category_id = categories[0].id;
        values.category = categories[0].name;
      }
      // 转换标签名称到标签ID
      const tagIds: number[] = [];
      if (values.tag_ids && values.tag_ids.length > 0) {
        tagIds.push(...values.tag_ids);
      } else if (values.tags && values.tags.length > 0) {
        // 如果提供的是标签名称，查找对应的ID
        for (const tagName of values.tags) {
          const tag = tags.find(t => t.name === tagName);
          if (tag) {
            tagIds.push(tag.id);
          }
        }
      }

      // 转换前端格式到后端格式
      const response = await ticketService.createTicket({
        title: values.title,
        description: values.description,
        priority: values.priority,
        category: values.category || values.type, // 使用type作为category的fallback
        category_id: values.category_id,
        template_id: values.template_id,
        tag_ids: tagIds.length > 0 ? tagIds : undefined,
        assignee_id: undefined, // 后端会自动设置
      });

      // 如果有附件，上传附件
      if (fileList.length > 0 && response.id) {
        try {
          const uploadPromises = fileList
            .filter(file => file.originFileObj)
            .map(file => {
              const formData = new FormData();
              formData.append('file', file.originFileObj as File);
              return httpClient.post(`/api/v1/tickets/${response.id}/attachments`, formData);
            });

          await Promise.all(uploadPromises);
          message.success('附件上传成功');
        } catch (uploadError) {
          console.error('Failed to upload attachments:', uploadError);
          message.warning('工单创建成功，但附件上传失败');
        }
      }

      message.success(`工单创建成功！工单编号: ${response.ticket_number || response.id}`);
      router.push(`/tickets/${response.id}`);
    } catch (error: any) {
      console.error('Failed to create ticket:', error);
      message.error(error?.message || '工单创建失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  const handleFileChange = (info: any) => {
    setFileList(info.fileList);
  };

  const nextStep = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1);
    }
  };

  const prevStep = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return (
          <div className='space-y-6'>
            <Alert
              message='智能工单创建'
              description='填写工单信息，AI 将帮助优化分类、优先级和工作流程'
              type='info'
              showIcon
              icon={<Lightbulb />}
            />

            <Form
              form={form}
              layout='vertical'
              onValuesChange={handleFormChange}
              initialValues={{
                priority: TicketPriority.MEDIUM,
                type: TicketType.INCIDENT,
                tags: [],
              }}
            >
              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item
                    name='title'
                    label='工单标题'
                    rules={[{ required: true, message: '请输入工单标题' }]}
                  >
                    <Input placeholder='请简要描述问题...' size='large' prefix={<FileText />} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item
                    name='description'
                    label='详细描述'
                    rules={[{ required: true, message: '请输入详细描述' }]}
                  >
                    <TextArea
                      placeholder='请描述问题症状、影响范围、重现步骤等...'
                      rows={6}
                      showCount
                      maxLength={1000}
                    />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name='type'
                    label='工单类型'
                    rules={[{ required: true, message: '请选择工单类型' }]}
                  >
                    <Select size='large'>
                      <Option value={TicketType.INCIDENT}>故障 (Incident)</Option>
                      <Option value={TicketType.SERVICE_REQUEST}>服务请求 (Service Request)</Option>
                      <Option value={TicketType.PROBLEM}>问题 (Problem)</Option>
                      <Option value={TicketType.CHANGE}>变更 (Change)</Option>
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name='priority'
                    label='优先级'
                    rules={[{ required: true, message: '请选择优先级' }]}
                  >
                    <Select size='large'>
                      <Option value={TicketPriority.LOW}>低 (Low)</Option>
                      <Option value={TicketPriority.MEDIUM}>中 (Medium)</Option>
                      <Option value={TicketPriority.HIGH}>高 (High)</Option>
                      <Option value={TicketPriority.URGENT}>紧急 (Urgent)</Option>
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name='category_id' label='分类'>
                    <Select
                      size='large'
                      placeholder={loadingCategories ? '加载中...' : '选择问题分类'}
                      loading={loadingCategories}
                      showSearch
                      optionFilterProp='label'
                      onChange={value => {
                        form.setFieldsValue({ category_id: value });
                        const selectedCategory = categories.find(c => c.id === value);
                        if (selectedCategory) {
                          form.setFieldsValue({ category: selectedCategory.name });
                        }
                      }}
                    >
                      {categories.map(cat => (
                        <Option key={cat.id} value={cat.id} label={cat.name}>
                          {cat.name}
                        </Option>
                      ))}
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name='template_id' label='模板 (可选)'>
                    <Select
                      size='large'
                      placeholder={loadingTemplates ? '加载中...' : '选择工单模板'}
                      loading={loadingTemplates}
                      showSearch
                      optionFilterProp='label'
                      allowClear
                    >
                      {templates.map(tpl => (
                        <Option key={tpl.id} value={tpl.id} label={tpl.name}>
                          {tpl.name} - {tpl.description || tpl.category}
                        </Option>
                      ))}
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item name='tag_ids' label='标签'>
                    <Select
                      mode='multiple'
                      size='large'
                      placeholder={loadingTags ? '加载中...' : '选择相关标签'}
                      loading={loadingTags}
                      showSearch
                      optionFilterProp='label'
                      allowClear
                    >
                      {tags.map(tag => (
                        <Option key={tag.id} value={tag.id} label={tag.name}>
                          <Tag color={tag.color}>{tag.name}</Tag>
                        </Option>
                      ))}
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item name='attachments' label='附件'>
                    <Upload
                      fileList={fileList}
                      onChange={handleFileChange}
                      beforeUpload={() => false} // 阻止自动上传
                      multiple
                    >
                      <Button icon={<UploadIcon />} size='large'>
                        选择文件
                      </Button>
                    </Upload>
                    <div className='text-sm text-gray-500 mt-2'>
                      支持多个文件上传，单个文件不超过10MB
                    </div>
                  </Form.Item>
                </Col>
              </Row>
            </Form>
          </div>
        );

      case 1:
        return (
          <div className='space-y-6'>
            <Alert
              message='AI 智能分析'
              description='AI 将分析您的工单内容并提供分类建议、优先级推荐和最佳工作流程'
              type='success'
              showIcon
              icon={<Bot />}
            />

            <AIWorkflowAssistant
              initialTitle={form.getFieldValue('title')}
              initialDescription={form.getFieldValue('description')}
              onSuggestion={handleAISuggestion}
            />

            {aiSuggestion && (
              <Card
                title={
                  <Space>
                    <CheckCircle className='text-green-600' />
                    <span>AI 建议已应用</span>
                  </Space>
                }
                className='border-green-200 bg-green-50'
              >
                <div className='space-y-3'>
                  <div className='flex items-center justify-between'>
                    <Text>推荐分类:</Text>
                    <Tag color='blue'>{aiSuggestion.category}</Tag>
                  </div>
                  <div className='flex items-center justify-between'>
                    <Text>推荐优先级:</Text>
                    <Tag color={aiSuggestion.priority === 'urgent' ? 'red' : 'orange'}>
                      {aiSuggestion.priority.toUpperCase()}
                    </Tag>
                  </div>
                  <div className='flex items-center justify-between'>
                    <Text>预计处理时间:</Text>
                    <Text strong>{aiSuggestion.estimatedTime}</Text>
                  </div>
                  <div className='flex items-center justify-between'>
                    <Text>置信度:</Text>
                    <Text strong>{Math.round(aiSuggestion.confidence * 100)}%</Text>
                  </div>
                </div>
              </Card>
            )}
          </div>
        );

      case 2:
        return (
          <div className='space-y-6'>
            <Alert
              message='确认工单信息'
              description='请在创建前确认工单信息正确'
              type='warning'
              showIcon
              icon={<CheckCircle />}
            />

            <Card title='工单信息预览'>
              <div className='space-y-4'>
                <div className='grid grid-cols-2 gap-4'>
                  <div>
                    <Text type='secondary'>标题</Text>
                    <div className='mt-1 font-medium'>{formData.title}</div>
                  </div>
                  <div>
                    <Text type='secondary'>类型</Text>
                    <div className='mt-1'>
                      <Tag color='blue'>{formData.type}</Tag>
                    </div>
                  </div>
                  <div>
                    <Text type='secondary'>优先级</Text>
                    <div className='mt-1'>
                      <Tag
                        color={
                          formData.priority === 'urgent'
                            ? 'red'
                            : formData.priority === 'high'
                            ? 'orange'
                            : formData.priority === 'medium'
                            ? 'blue'
                            : 'green'
                        }
                      >
                        {formData.priority?.toUpperCase()}
                      </Tag>
                    </div>
                  </div>
                  <div>
                    <Text type='secondary'>分类</Text>
                    <div className='mt-1 font-medium'>{formData.category}</div>
                  </div>
                </div>

                <div>
                  <Text type='secondary'>描述</Text>
                  <div className='mt-1 p-3 bg-gray-50 rounded'>{formData.description}</div>
                </div>

                {formData.tags && formData.tags.length > 0 && (
                  <div>
                    <Text type='secondary'>标签</Text>
                    <div className='mt-1'>
                      {formData.tags.map((tag, index) => (
                        <Tag key={index} className='mb-1'>
                          {tag}
                        </Tag>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </Card>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className='max-w-6xl mx-auto p-6'>
      <div className='mb-6'>
        <Title level={2}>
          <FileText className='mr-3 text-blue-600' />
          创建工单
        </Title>
        <Text type='secondary'>智能工单创建系统，AI 助手将帮助您优化工单分类和工作流程</Text>
      </div>

      <Card>
        <Steps current={currentStep} className='mb-8'>
          {steps.map((step, index) => (
            <Step key={index} title={step.title} description={step.content} icon={step.icon} />
          ))}
        </Steps>

        <div className='min-h-[400px]'>{renderStepContent()}</div>

        <Divider />

        <div className='flex justify-between'>
          <Button onClick={prevStep} disabled={currentStep === 0} icon={<ArrowLeft />}>
            上一步
          </Button>

          <Space>
            {currentStep < steps.length - 1 ? (
              <Button type='primary' onClick={nextStep} icon={<Workflow />}>
                下一步
              </Button>
            ) : (
              <Button
                type='primary'
                onClick={() => {
                  const values = form.getFieldsValue(true) as any;
                  handleSubmit(values);
                }}
                loading={loading}
                icon={<Send />}
                size='large'
              >
                创建工单
              </Button>
            )}
          </Space>
        </div>
      </Card>
    </div>
  );
}
