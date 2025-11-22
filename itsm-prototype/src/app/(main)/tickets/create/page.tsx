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
      title: 'Basic Information',
      icon: <FileText />,
      content: 'Fill in basic ticket information and description',
    },
    {
      title: 'AI Analysis',
      icon: <Bot />,
      content: 'AI analyzes ticket content and provides suggestions',
    },
    {
      title: 'Confirm Creation',
      icon: <CheckCircle />,
      content: 'Confirm ticket information and create',
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

    message.success('AI suggestions applied, please review and confirm');
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

      message.success(`Ticket created successfully! Ticket ID: ${response.ticket_id}`);
      router.push(`/tickets/${response.ticket_id}`);
    } catch (error) {
      console.error('Failed to create ticket:', error);
      message.error('Failed to create ticket, please try again later');
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
              message='Smart Ticket Creation'
              description='Fill in ticket information, AI will help optimize classification, priority and workflow'
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
                    label='Ticket Title'
                    rules={[{ required: true, message: 'Please enter ticket title' }]}
                  >
                    <Input
                      placeholder='Please briefly describe the issue...'
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
                    label='Detailed Description'
                    rules={[{ required: true, message: 'Please enter detailed description' }]}
                  >
                    <TextArea
                      placeholder='Please describe the issue symptoms, impact scope, reproduction steps, etc...'
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
                    label='Ticket Type'
                    rules={[{ required: true, message: 'Please select ticket type' }]}
                  >
                    <Select size='large'>
                      <Option value={TicketType.INCIDENT}>Incident</Option>
                      <Option value={TicketType.SERVICE_REQUEST}>Service Request</Option>
                      <Option value={TicketType.PROBLEM}>Problem</Option>
                      <Option value={TicketType.CHANGE}>Change</Option>
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name='priority'
                    label='Priority'
                    rules={[{ required: true, message: 'Please select priority' }]}
                  >
                    <Select size='large'>
                      <Option value={TicketPriority.LOW}>Low</Option>
                      <Option value={TicketPriority.MEDIUM}>Medium</Option>
                      <Option value={TicketPriority.HIGH}>High</Option>
                      <Option value={TicketPriority.URGENT}>Urgent</Option>
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name='category'
                    label='Category'
                    rules={[{ required: true, message: 'Please select category' }]}
                  >
                    <Select size='large' placeholder='Select issue category'>
                      <Option value='system'>System Issue</Option>
                      <Option value='network'>Network Issue</Option>
                      <Option value='database'>Database Issue</Option>
                      <Option value='hardware'>Hardware Issue</Option>
                      <Option value='software'>Software Issue</Option>
                      <Option value='access'>Access Permission</Option>
                      <Option value='other'>Other</Option>
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name='subcategory' label='Subcategory'>
                    <Select size='large' placeholder='Select subcategory (optional)'>
                      <Option value='login'>Login Issue</Option>
                      <Option value='performance'>Performance Issue</Option>
                      <Option value='error'>Error Message</Option>
                      <Option value='configuration'>Configuration Issue</Option>
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item name='tags' label='Tags'>
                    <Select
                      mode='tags'
                      size='large'
                      placeholder='Add relevant tags'
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
              message='AI Smart Analysis'
              description='AI will analyze your ticket content and provide classification suggestions, priority recommendations and optimal workflow'
              type='success'
              showIcon
              icon={<Bot />}
            />

            <AIWorkflowAssistant onSuggestion={handleAISuggestion} />

            {aiSuggestion && (
              <Card
                title={
                  <Space>
                    <CheckCircle className='text-green-600' />
                    <span>AI Suggestions Applied</span>
                  </Space>
                }
                className='border-green-200 bg-green-50'
              >
                <div className='space-y-3'>
                  <div className='flex items-center justify-between'>
                    <Text>Recommended Category:</Text>
                    <Tag color='blue'>{aiSuggestion.category}</Tag>
                  </div>
                  <div className='flex items-center justify-between'>
                    <Text>Recommended Priority:</Text>
                    <Tag color={aiSuggestion.priority === 'urgent' ? 'red' : 'orange'}>
                      {aiSuggestion.priority.toUpperCase()}
                    </Tag>
                  </div>
                  <div className='flex items-center justify-between'>
                    <Text>Estimated Processing Time:</Text>
                    <Text strong>{aiSuggestion.estimatedTime}</Text>
                  </div>
                  <div className='flex items-center justify-between'>
                    <Text>Confidence:</Text>
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
              message='Confirm Ticket Information'
              description='Please confirm the ticket information is correct before creating'
              type='warning'
              showIcon
              icon={<CheckCircle />}
            />

            <Card title='Ticket Information Preview'>
              <div className='space-y-4'>
                <div className='grid grid-cols-2 gap-4'>
                  <div>
                    <Text type='secondary'>Title</Text>
                    <div className='mt-1 font-medium'>{formData.title}</div>
                  </div>
                  <div>
                    <Text type='secondary'>Type</Text>
                    <div className='mt-1'>
                      <Tag color='blue'>{formData.type}</Tag>
                    </div>
                  </div>
                  <div>
                    <Text type='secondary'>Priority</Text>
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
                    <Text type='secondary'>Category</Text>
                    <div className='mt-1 font-medium'>{formData.category}</div>
                  </div>
                </div>

                <div>
                  <Text type='secondary'>Description</Text>
                  <div className='mt-1 p-3 bg-gray-50 rounded'>{formData.description}</div>
                </div>

                {formData.tags && formData.tags.length > 0 && (
                  <div>
                    <Text type='secondary'>Tags</Text>
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
          Create Ticket
        </Title>
        <Text type='secondary'>
          Smart ticket creation system, AI assistant will help you optimize ticket classification
          and workflow
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
          <Button onClick={prevStep} disabled={currentStep === 0} icon={<Calendar />}>
            Previous
          </Button>

          <Space>
            {currentStep < steps.length - 1 ? (
              <Button type='primary' onClick={nextStep} icon={<Workflow />}>
                Next
              </Button>
            ) : (
              <Button
                type='primary'
                onClick={() => form.submit()}
                loading={loading}
                icon={<Send />}
                size='large'
              >
                Create Ticket
              </Button>
            )}
          </Space>
        </div>
      </Card>
    </div>
  );
}
