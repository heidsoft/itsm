'use client';

import React, { useState } from 'react';
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
} from 'lucide-react';
import { AIWorkflowAssistant } from '@/components/business/AIWorkflowAssistant';
import { ticketService, TicketPriority, TicketType } from '@/lib/services/ticket-service';

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
  subcategory?: string;
  tags: string[];
  attachments: File[];
}

export default function CreateTicketPage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);
  const [aiSuggestion, setAiSuggestion] = useState<any>(null);
  const [formData, setFormData] = useState<Partial<CreateTicketForm>>({});

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
      const response = await ticketService.createTicket({
        title: values.title,
        description: values.description,
        priority: values.priority,
        type: values.type,
        category: values.category,
        subcategory: values.subcategory,
        tags: values.tags,
      });

      message.success(`工单创建成功！工单 ID: ${response.ticket_id}`);
      router.push(`/tickets/${response.ticket_id}`);
    } catch (error) {
      console.error('Failed to create ticket:', error);
      message.error('工单创建失败，请稍后重试');
    } finally {
      setLoading(false);
    }
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
                    <Input
                      placeholder='请简要描述问题...'
                      size='large'
                      prefix={<FileText />}
                    />
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
                  <Form.Item
                    name='category'
                    label='分类'
                    rules={[{ required: true, message: '请选择分类' }]}
                  >
                    <Select size='large' placeholder='选择问题分类'>
                      <Option value='system'>系统问题</Option>
                      <Option value='network'>网络问题</Option>
                      <Option value='database'>数据库问题</Option>
                      <Option value='hardware'>硬件问题</Option>
                      <Option value='software'>软件问题</Option>
                      <Option value='access'>访问权限</Option>
                      <Option value='other'>其他</Option>
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name='subcategory' label='子分类'>
                    <Select size='large' placeholder='选择子分类 (可选)'>
                      <Option value='login'>登录问题</Option>
                      <Option value='performance'>性能问题</Option>
                      <Option value='error'>错误消息</Option>
                      <Option value='configuration'>配置问题</Option>
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item name='tags' label='标签'>
                    <Select
                      mode='tags'
                      size='large'
                      placeholder='添加相关标签'
                      prefix={<TagIcon />}
                    />
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
        <Text type='secondary'>
          智能工单创建系统，AI 助手将帮助您优化工单分类和工作流程
        </Text>
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
                onClick={() => form.submit()}
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
