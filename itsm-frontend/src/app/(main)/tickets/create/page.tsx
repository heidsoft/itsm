'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Space,
  Typography,
  App,
  Divider,
  Empty,
  Tag,
  Row,
  Col,
} from 'antd';
import {
  ArrowLeft,
  Container,
  Database,
  Download,
  Monitor,
  User,
  Code,
  Globe,
  Shield,
  Boxes,
  Folder,
  Key,
  FileText,
  ChevronRight,
} from 'lucide-react';
import { TicketApi } from '@/lib/api/ticket-api';
import {
  ticketTypePresets,
  ticketTypeCategories,
  getTicketTypesByCategory,
  type TicketTypePreset,
} from '@/lib/ticket-type-presets';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

type Priority = 'low' | 'medium' | 'high' | 'urgent';

// 图标映射
const iconMap: Record<string, React.ReactNode> = {
  Container: <Container className='w-5 h-5' />,
  Database: <Database className='w-5 h-5' />,
  Download: <Download className='w-5 h-5' />,
  Desktop: <Monitor className='w-5 h-5' />,
  User: <User className='w-5 h-5' />,
  Code: <Code className='w-5 h-5' />,
  Global: <Globe className='w-5 h-5' />,
  Safety: <Shield className='w-5 h-5' />,
  Appstore: <Boxes className='w-5 h-5' />,
  Project: <Folder className='w-5 h-5' />,
  Key: <Key className='w-5 h-5' />,
  FileText: <FileText className='w-5 h-5' />,
};

export default function CreateTicketPage() {
  const router = useRouter();
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  // 选中的工单类型
  const [selectedType, setSelectedType] = useState<TicketTypePreset | null>(null);
  // 分类筛选
  const [categoryFilter, setCategoryFilter] = useState('all');

  // 过滤后的类型列表
  const filteredTypes = getTicketTypesByCategory(categoryFilter);

  // 选中类型时自动设置优先级
  useEffect(() => {
    if (selectedType) {
      form.setFieldValue('priority', selectedType.priority);
      // 设置分类为预设的分类
      form.setFieldValue('category', selectedType.category);
    }
  }, [selectedType, form]);

  // 处理提交
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      // 构建描述：基础描述 + 预设字段
      let description = values.description || '';

      // 如果有预设字段，添加到描述中
      if (selectedType?.fields && selectedType.fields.length > 0) {
        const fieldDetails = selectedType.fields
          .map(field => {
            const value = values[field.name];
            if (value) {
              const optionLabel =
                field.options?.find(opt => opt.value === value)?.label || value;
              return `${field.label}: ${optionLabel}`;
            }
            return null;
          })
          .filter(Boolean)
          .join('\n');

        if (fieldDetails) {
          description = `[${selectedType.name}]\n${fieldDetails}\n\n---\n${description}`;
        }
      }

      // 检查描述长度
      if (description.length < 10) {
        message.warning('描述至少需要10个字符，请详细描述问题');
        setLoading(false);
        return;
      }

      const created = await TicketApi.createTicket({
        title: values.title,
        description: description,
        priority: values.priority,
        category: values.category,
      });

      message.success('工单创建成功');
      router.push(`/tickets/${created.id}`);
    } catch (e: any) {
      console.error('Create ticket error:', e);
      const errorMsg = e?.message || e?.error?.message || '创建工单失败，请检查输入或重新登录';
      message.error(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  // 渲染类型选择卡片
  const renderTypeCard = (type: TicketTypePreset) => {
    const isSelected = selectedType?.id === type.id;

    return (
      <Card
        key={type.id}
        size='small'
        hoverable
        onClick={() => setSelectedType(type)}
        style={{
          borderColor: isSelected ? type.color : '#d9d9d9',
          backgroundColor: isSelected ? `${type.color}10` : '#fff',
          cursor: 'pointer',
        }}
        className='transition-all'
      >
        <div className='flex items-center gap-3'>
          <div
            style={{
              color: type.color,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            {iconMap[type.icon] || <FileText className='w-5 h-5' />}
          </div>
          <div style={{ flex: 1 }}>
            <div className='font-medium'>{type.name}</div>
            <Text type='secondary' className='text-xs'>
              {type.description}
            </Text>
          </div>
          {isSelected && <ChevronRight className='w-4 h-4' style={{ color: type.color }} />}
        </div>
      </Card>
    );
  };

  // 渲染动态表单字段
  const renderDynamicFields = () => {
    if (!selectedType?.fields || selectedType.fields.length === 0) {
      return null;
    }

    return (
      <Card title={`${selectedType.name} - 详细信息`} style={{ marginBottom: 16 }}>
        <Row gutter={[16, 0]}>
          {selectedType.fields.map(field => (
            <Col span={24} key={field.name}>
              <Form.Item
                name={field.name}
                label={field.label}
                rules={
                  field.required
                    ? [{ required: true, message: `请填写${field.label}` }]
                    : []
                }
              >
                {field.type === 'textarea' ? (
                  <TextArea rows={3} placeholder={field.placeholder} />
                ) : field.type === 'select' ? (
                  <Select
                    placeholder={field.placeholder || `请选择${field.label}`}
                    options={field.options}
                  />
                ) : field.type === 'number' ? (
                  <Input type='number' placeholder={field.placeholder} />
                ) : field.type === 'date' ? (
                  <Input type='date' />
                ) : (
                  <Input placeholder={field.placeholder} />
                )}
              </Form.Item>
            </Col>
          ))}
        </Row>
      </Card>
    );
  };

  return (
    <div className='max-w-6xl mx-auto p-6'>
      <Space orientation='vertical' size={16} style={{ width: '100%' }}>
        {/* 页面头部 */}
        <Card>
          <Space align='center' style={{ width: '100%' }}>
            <Button icon={<ArrowLeft className='w-4 h-4' />} onClick={() => router.back()}>
              返回
            </Button>
            <div style={{ flex: 1 }}>
              <Title level={4} style={{ marginBottom: 4 }}>
                新建工单
              </Title>
              <Text type='secondary'>选择工单类型，填写详细信息后提交</Text>
            </div>
          </Space>
        </Card>

        <Row gutter={16}>
          {/* 左侧：类型选择 */}
          <Col xs={24} md={10}>
            <Card
              title={
                <Space>
                  <span>选择工单类型</span>
                  <Tag color='blue'>{filteredTypes.length} 种</Tag>
                </Space>
              }
              bodyStyle={{ padding: '12px' }}
            >
              {/* 分类筛选 */}
              <div style={{ marginBottom: 12 }}>
                <Space wrap>
                  {ticketTypeCategories.map(cat => (
                    <Tag
                      key={cat.key}
                      color={categoryFilter === cat.key ? 'blue' : 'default'}
                      style={{ cursor: 'pointer' }}
                      onClick={() => setCategoryFilter(cat.key)}
                    >
                      {cat.label}
                    </Tag>
                  ))}
                </Space>
              </div>

              {/* 类型列表 */}
              <div
                style={{
                  maxHeight: 500,
                  overflowY: 'auto',
                }}
              >
                <Space direction='vertical' style={{ width: '100%' }} size={8}>
                  {filteredTypes.map(type => renderTypeCard(type))}
                </Space>
              </div>
            </Card>
          </Col>

          {/* 右侧：表单 */}
          <Col xs={24} md={14}>
            <Form form={form} layout='vertical'>
              {/* 已选类型提示 */}
              {selectedType && (
                <Card
                  style={{
                    marginBottom: 16,
                    borderColor: selectedType.color,
                    background: `${selectedType.color}08`,
                  }}
                >
                  <Space>
                    <div style={{ color: selectedType.color }}>
                      {iconMap[selectedType.icon] || <FileText className='w-5 h-5' />}
                    </div>
                    <div>
                      <div className='font-medium'>{selectedType.name}</div>
                      <Text type='secondary'>{selectedType.description}</Text>
                    </div>
                    <Button
                      type='link'
                      size='small'
                      onClick={() => setSelectedType(null)}
                    >
                      更换
                    </Button>
                  </Space>
                </Card>
              )}

              {/* 动态字段 */}
              {renderDynamicFields()}

              {/* 基础字段 */}
              <Card title='基础信息'>
                <Form.Item
                  name='title'
                  label='标题'
                  rules={[{ required: true, message: '请输入标题', min: 2 }]}
                >
                  <Input
                    placeholder={
                      selectedType
                        ? `例如：${selectedType.name} - xxx`
                        : '例如：VPN 无法连接'
                    }
                  />
                </Form.Item>

                <Form.Item
                  name='description'
                  label='详细描述'
                  rules={[
                    { required: true, message: '请输入描述（至少10个字符）' },
                    { min: 10, message: '描述至少需要10个字符' },
                  ]}
                >
                  <TextArea
                    rows={6}
                    placeholder={
                      selectedType
                        ? `请详细描述${selectedType.name}的具体需求...`
                        : '请详细描述问题/需求与影响范围...'
                    }
                  />
                </Form.Item>

                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item
                      name='priority'
                      label='优先级'
                      initialValue='medium'
                      rules={[{ required: true }]}
                    >
                      <Select<Priority>
                        options={[
                          { label: '低', value: 'low' },
                          { label: '中', value: 'medium' },
                          { label: '高', value: 'high' },
                          { label: '紧急', value: 'urgent' },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item
                      name='category'
                      label='分类'
                      initialValue={selectedType?.category || '技术支持'}
                    >
                      <Select
                        options={[
                          { label: '技术支持', value: '技术支持' },
                          { label: '账户问题', value: '账户问题' },
                          { label: '系统故障', value: '系统故障' },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                </Row>
              </Card>

              <Space>
                <Button type='primary' onClick={handleSubmit} loading={loading}>
                  创建工单
                </Button>
                <Button onClick={() => router.push('/tickets')}>取消</Button>
              </Space>
            </Form>
          </Col>
        </Row>
      </Space>
    </div>
  );
}
