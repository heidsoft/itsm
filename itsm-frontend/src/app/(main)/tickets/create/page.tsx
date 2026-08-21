'use client';

import React, { useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import {
  Card,
  Form,
  Input,
  Button,
  Space,
  Typography,
  App,
  Empty,
  Tag,
  Row,
  Col,
  Spin,
  Alert,
  DatePicker,
  Select,
	Checkbox,
	InputNumber,
} from 'antd';
import AppSelect from '@/components/ui/AppSelect';
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
  Sparkles,
} from 'lucide-react';
import { TicketApi } from '@/lib/api/ticket-api';
import { TicketCategoryApi } from '@/lib/api/ticket-category-api';
import { useI18n } from '@/lib/i18n';
import { httpClient } from '@/lib/api/http-client';
import { TicketTypeApi } from '@/lib/api/ticketTypeApi';
import type { CustomFieldDefinition } from '@/types/ticket-type';

const { Title, Text } = Typography;
const { TextArea } = Input;

type Priority = 'low' | 'medium' | 'high' | 'urgent' | 'critical';
type TicketCreateType = 'incident' | 'service_request' | 'change' | 'problem';
type RuntimeTicketType = { id: number; code: string; name: string; description: string; icon: string; color: string; priority: Priority; workflowDefinitionKey?: string; fields: CustomFieldDefinition[] };

// 图标映射
const iconMap: Record<string, React.ReactNode> = {
  Container: <Container className="w-5 h-5" />,
  Database: <Database className="w-5 h-5" />,
  Download: <Download className="w-5 h-5" />,
  Desktop: <Monitor className="w-5 h-5" />,
  User: <User className="w-5 h-5" />,
  Code: <Code className="w-5 h-5" />,
  Global: <Globe className="w-5 h-5" />,
  Safety: <Shield className="w-5 h-5" />,
  Appstore: <Boxes className="w-5 h-5" />,
  Project: <Folder className="w-5 h-5" />,
  Key: <Key className="w-5 h-5" />,
  FileText: <FileText className="w-5 h-5" />,
};

const inferTicketType = (selectedType: RuntimeTicketType | null): TicketCreateType => {
  if (!selectedType) {
    return 'incident';
  }

  const value = `${selectedType.id} ${selectedType.code} ${selectedType.name} ${selectedType.workflowDefinitionKey || ''}`;
  if (/change|变更|ddl|firewall|domain/i.test(value)) {
    return 'change';
  }
  if (/problem|问题/i.test(value)) {
    return 'problem';
  }
  return 'service_request';
};

export default function CreateTicketPage() {
  const router = useRouter();
  const { message } = App.useApp();
  const { t } = useI18n();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
	const [ticketTypes, setTicketTypes] = useState<RuntimeTicketType[]>([]);
	const [ticketTypesLoading, setTicketTypesLoading] = useState(true);
	const [ticketTypesError, setTicketTypesError] = useState<string | null>(null);

  // 选中的工单类型
  const [selectedType, setSelectedType] = useState<RuntimeTicketType | null>(null);
	const [referenceOptions, setReferenceOptions] = useState<Record<'user' | 'department' | 'ci', { label: string; value: number }[]>>({ user: [], department: [], ci: [] });
  // 分类筛选
  // AI 分类建议
  const [aiSuggestions, setAiSuggestions] = useState<{
    category?: string;
    priority?: string;
    urgency?: string;
    confidence?: number;
    reasoning?: string;
  } | null>(null);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);
  // 分类下拉选项：从后端 TicketCategory 主数据动态拉取，避免前后端分类词表零重合
  const [categoryOptions, setCategoryOptions] = useState<{ label: string; value: string }[]>([]);
  const [categoryLoading, setCategoryLoading] = useState(false);

	useEffect(() => {
		let cancelled = false;
		TicketTypeApi.list({ status: 'active', page: 1, pageSize: 100 })
			.then(result => {
				if (cancelled) return;
				setTicketTypes(result.types.map(type => ({
					id: type.id, code: type.code, name: type.name,
					description: type.description ?? '', icon: type.icon ?? 'FileText', color: type.color ?? '#1677ff',
					priority: type.defaultPriority ?? 'medium', workflowDefinitionKey: type.workflowDefinitionKey,
					fields: type.customFields.filter(field => field.visible !== false),
				})));
			})
			.catch(error => { if (!cancelled) setTicketTypesError(error instanceof Error ? error.message : '工单类型加载失败'); })
			.finally(() => { if (!cancelled) setTicketTypesLoading(false); });
		return () => { cancelled = true; };
	}, []);

	useEffect(() => {
		Promise.allSettled([
			httpClient.get<any>('/api/v1/users', { page: 1, pageSize: 200, status: 'active' }),
			httpClient.get<any>('/api/v1/departments', { page: 1, pageSize: 200 }),
			httpClient.get<any>('/api/v1/configuration-items', { page: 1, size: 200 }),
		]).then(([users, departments, cis]) => setReferenceOptions({
			user: users.status === 'fulfilled' ? (users.value.users ?? users.value.items ?? []).map((item: any) => ({ label: item.name ?? item.username, value: item.id })) : [],
			department: departments.status === 'fulfilled' ? (departments.value.departments ?? departments.value.items ?? departments.value ?? []).map((item: any) => ({ label: item.name, value: item.id })) : [],
			ci: cis.status === 'fulfilled' ? (cis.value.items ?? cis.value.cis ?? []).map((item: any) => ({ label: item.name, value: item.id })) : [],
		}));
	}, []);

  useEffect(() => {
    let cancelled = false;
    const loadCategories = async () => {
      setCategoryLoading(true);
      try {
        const res = await TicketCategoryApi.getCategories({ isActive: true, pageSize: 200 });
        if (cancelled) return;
        const list = res.categories ?? res.items ?? [];
        setCategoryOptions(
          list
            .filter(c => c.isActive !== false)
            .map(c => ({ label: c.name, value: c.name })),
        );
      } catch (e) {
        console.error('加载工单分类失败', e);
      } finally {
        if (!cancelled) setCategoryLoading(false);
      }
    };
    loadCategories();
    return () => {
      cancelled = true;
    };
  }, []);

  // 过滤后的类型列表（后端已按 sortOrder 排序，前端不再做分类筛选）
  const filteredTypes = ticketTypes;

  // 跟踪上一次选中类型的动态字段名，切换类型时清理残留值
  const prevFieldNames = useRef<string[]>([]);

  // 选中类型时自动设置优先级并重置动态字段
  useEffect(() => {
    // 清理上一个类型遗留在 form store 中的动态字段值
    if (prevFieldNames.current.length > 0) {
      form.resetFields(prevFieldNames.current);
    }
    prevFieldNames.current = selectedType?.fields.map(f => f.name) ?? [];
    if (selectedType) {
      form.setFieldValue('priority', selectedType.priority);
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
              const optionLabel = field.options?.find(opt => opt.value === value)?.label || value;
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

      // 构建 title：如果没有选择类型，使用表单中的 title；否则使用类型名称
      const title = values.title || (selectedType ? `${selectedType.name}请求` : '新建工单');
      // 构建 priority：如果没有选择类型，使用表单中的 priority；否则使用预设优先级
      const priority = values.priority || (selectedType ? selectedType.priority : 'medium');

      const created = await TicketApi.createTicket({
        title: title,
        description: description,
        priority: priority,
        type: inferTicketType(selectedType),
		ticketTypeId: selectedType?.id,
        category: values.category || undefined,
		formFields: selectedType ? selectedType.fields?.reduce<Record<string, unknown>>((fields, field) => {
			const value = values[field.name];
			fields[field.name] = field.type === 'date' && value?.format ? value.format('YYYY-MM-DD') : value?.toISOString?.() ?? value;
			return fields;
		}, {}) : undefined,
      });

      message.success('工单创建成功');
      router.push(`/tickets/${created.id}`);
    } catch (e: unknown) {
      console.error('Create ticket error:', e);
      const errorObj = e as { message?: string; error?: { message?: string } };
      const errorMsg =
        errorObj?.message || errorObj?.error?.message || '创建工单失败，请检查输入或重新登录';
      message.error(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  // AI 智能分类
  const handleAITriage = async () => {
    try {
      const values = await form.validateFields();
      const title = values.title || (selectedType ? `${selectedType.name}请求` : '');
      const description = values.description || '';

      if (!title) {
        message.warning('请先填写标题');
        return;
      }

      setAiLoading(true);
      setAiError(null);

      const response = await httpClient.post<any>('/api/v1/ai/triage', {
        title,
        description,
        category: values.category,
        priority: values.priority,
      });

      if (response?.suggestions) {
        setAiSuggestions(response.suggestions);
        // 自动应用建议：分类仅在与后端主数据匹配时回填，
        // 否则 AI 返回的英文 slug（database/network/…）与 TicketCategory 名称不匹配会导致提交失败
        if (response.suggestions.category && categoryOptions.some(o => o.value === response.suggestions.category)) {
          form.setFieldValue('category', response.suggestions.category);
        }
        if (response.suggestions.priority) {
          form.setFieldValue('priority', response.suggestions.priority);
        }
        message.success('AI 分类建议已应用');
      }
    } catch (e: unknown) {
      console.error('AI triage error:', e);
      setAiError('AI 分类服务暂时不可用');
    } finally {
      setAiLoading(false);
    }
  };

  // 渲染类型选择卡片
  const renderTypeCard = (type: RuntimeTicketType) => {
    const isSelected = selectedType?.id === type.id;

    return (
      <Card
        key={type.id}
        size="small"
        hoverable
        onClick={() => setSelectedType(type)}
        style={{
          borderColor: isSelected ? type.color : '#d9d9d9',
          backgroundColor: isSelected ? `${type.color}10` : '#fff',
          cursor: 'pointer',
        }}
        className="transition-all"
      >
        <div className="flex items-center gap-3">
          <div
            style={{
              color: type.color,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            {iconMap[type.icon] || <FileText className="w-5 h-5" />}
          </div>
          <div style={{ flex: 1 }}>
            <div className="font-medium">{type.name}</div>
            <Text type="secondary" className="text-xs">
              {type.description}
            </Text>
          </div>
          {isSelected && <ChevronRight className="w-4 h-4" style={{ color: type.color }} />}
        </div>
      </Card>
    );
  };

  return (
    <div className="max-w-6xl mx-auto p-4 md:p-6" role="main" aria-label="创建工单页面">
      <Space orientation="vertical" size={16} style={{ width: '100%' }}>
        {/* 页面头部 */}
        <Card>
          <Space align="center" style={{ width: '100%' }}>
            <Button
              icon={<ArrowLeft className="w-4 h-4" />}
              onClick={() => router.back()}
              aria-label="返回上一页"
            >
              返回
            </Button>
            <div style={{ flex: 1 }}>
              <Title level={4} style={{ marginBottom: 4 }}>
                新建工单
              </Title>
              <Text type="secondary">选择工单类型，填写详细信息后提交</Text>
            </div>
          </Space>
        </Card>

        <Row gutter={[16, 16]}>
          {/* 左侧：类型选择 */}
          <Col xs={24} md={10}>
            <Card
              title={
                <Space>
                  <span>选择工单类型</span>
                  <Tag color="blue">{filteredTypes.length} 种</Tag>
                </Space>
              }
              styles={{ body: { padding: '12px' } }}
              aria-label="工单类型选择区域"
            >
              {/* 类型列表 */}
              <div
                style={{
                  maxHeight: 500,
                  overflowY: 'auto',
                }}
                role="list"
                aria-label="可用工单类型列表"
              >
                <Space
                  orientation="vertical"
                  style={{ width: '100%' }}
                  size={8}
                  role="presentation"
                >
                  {ticketTypesLoading ? <Spin /> : ticketTypesError ? <Alert type="error" showIcon title={ticketTypesError} /> : filteredTypes.length === 0 ? <Empty description="暂无已启用的工单类型" /> : filteredTypes.map(type => renderTypeCard(type))}
                </Space>
              </div>
            </Card>
          </Col>

          {/* 右侧：表单 */}
          <Col xs={24} md={14}>
            <Form form={form} layout="vertical" aria-label="工单表单" requiredMark="optional">
              {/* 已选类型提示 */}
              {selectedType && (
                <Card
                  style={{
                    marginBottom: 16,
                    borderColor: selectedType.color,
                    background: `${selectedType.color}08`,
                  }}
                  role="region"
                  aria-label={`已选工单类型：${selectedType.name}`}
                >
                  <Space orientation="vertical" style={{ width: '100%' }}>
                    <Space>
                      <div style={{ color: selectedType.color }} aria-hidden="true">
                        {iconMap[selectedType.icon] || <FileText className="w-5 h-5" />}
                      </div>
                      <div style={{ flex: 1 }}>
                        <div className="font-medium">{selectedType.name}</div>
                        <Text type="secondary">{selectedType.description}</Text>
                      </div>
                      <Button
                        type="link"
                        size="small"
                        onClick={() => setSelectedType(null)}
                        aria-label="更换工单类型"
                      >
                        更换
                      </Button>
                    </Space>
                    {/* 关联的工作流信息 */}
                    {selectedType.workflowDefinitionKey && (
                      <div
                        style={{
                          marginTop: 8,
                          padding: '8px 12px',
                          background: '#fff',
                          borderRadius: 4,
                          border: `1px solid ${selectedType.color}30`,
                        }}
                      >
                        <Space>
                          <Tag
                            color={
                              selectedType.priority === 'urgent'
                                ? 'red'
                                : selectedType.priority === 'high'
                                  ? 'orange'
                                  : 'blue'
                            }
                          >
                            {selectedType.priority === 'urgent'
                              ? '紧急'
                              : selectedType.priority === 'high'
                                ? '高'
                                : selectedType.priority === 'medium'
                                  ? '中'
                                  : '低'}
                          </Tag>
                          <Text type="secondary">审批流程: </Text>
                          <Text strong>{selectedType.workflowDefinitionKey}</Text>
                        </Space>
                      </div>
                    )}
                  </Space>
                </Card>
              )}

              {/* 智能显示表单：已选类型有字段→自定义表单 | 已选类型无字段或未选择→基础表单 */}
              {selectedType?.fields && selectedType.fields.length > 0 ? (
                /* 有自定义字段：只显示自定义表单（已包含所有必要信息） */
                <Card
                  key={selectedType.id}
                  title={`${selectedType.name} - 详细信息`}
                  style={{ marginBottom: 16 }}
                  aria-label={`${selectedType.name} 自定义表单`}
                >
                  <Row gutter={[16, 0]}>
                    {selectedType.fields.map(field => (
                      <Col span={24} key={field.name}>
                        <Form.Item
                          name={field.name}
                          label={field.label}
						  initialValue={field.defaultValue}
						  valuePropName={['boolean', 'checkbox'].includes(field.type) ? 'checked' : 'value'}
                          rules={
                            field.required
                              ? [{ required: true, message: `请填写${field.label}` }]
                              : []
                          }
                        >
                          {field.type === 'textarea' ? (
                            <TextArea
                              rows={3}
							  disabled={field.readonly}
                              placeholder={field.placeholder}
                              aria-label={field.label}
                            />
                          ) : field.type === 'select' ? (
                            <AppSelect
							  disabled={field.readonly}
                              placeholder={field.placeholder || `请选择${field.label}`}
                              options={field.options}
                              aria-label={field.label}
                            />
						  ) : field.type === 'multi_select' ? (
							<Select disabled={field.readonly} mode="multiple" options={field.options} placeholder={field.placeholder} />
						  ) : field.type === 'radio' ? (
							<Select disabled={field.readonly} options={field.options} placeholder={field.placeholder} />
						  ) : field.type === 'number' ? (
							<InputNumber disabled={field.readonly} style={{ width: '100%' }} placeholder={field.placeholder} />
						  ) : field.type === 'date' ? (
                            <DatePicker
							  disabled={field.readonly}
                              style={{ width: '100%' }}
                              aria-label={field.label}
                              placeholder={field.placeholder || `请选择${field.label}`}
                            />
						  ) : field.type === 'datetime' ? (
							<DatePicker disabled={field.readonly} showTime style={{ width: '100%' }} />
						  ) : ['boolean', 'checkbox'].includes(field.type) ? (
							<Checkbox disabled={field.readonly}>{field.placeholder}</Checkbox>
						  ) : ['user', 'user_picker', 'department', 'department_picker', 'ci'].includes(field.type) ? (
							<Select disabled={field.readonly} showSearch optionFilterProp="label" options={referenceOptions[field.type === 'ci' ? 'ci' : field.type.startsWith('department') ? 'department' : 'user']} placeholder={field.placeholder} />
						  ) : (
							<Input disabled={field.readonly} placeholder={field.placeholder} aria-label={field.label} />
                          )}
                        </Form.Item>
                      </Col>
                    ))}
                  </Row>
                </Card>
              ) : (
                /* 无自定义字段：显示基础表单 */
                <Card
                  title="工单信息"
                  style={{ marginBottom: 16 }}
                  aria-label="基础工单信息表单"
                  data-testid="ticket-form"
                >
                  <Alert
                    type="info"
                    showIcon
                    className="mb-4"
                    message="填写最少信息即可提交，补充说明建议写在详细描述中。"
                  />
                  <Form.Item
                    name="title"
                    label="标题"
                    rules={[
                      { required: true, message: '请输入标题' },
                      { min: 2, message: '标题至少需要2个字符' },
                    ]}
                  >
                    <Input
                      placeholder="例如：VPN 无法连接"
                      aria-required="true"
                      aria-describedby="title-help"
                      data-testid="ticket-title-input"
                    />
                  </Form.Item>

                  <Form.Item
                    name="description"
                    label="详细描述"
                    rules={[
                      { required: true, message: '请输入描述（至少10个字符）' },
                      { min: 10, message: '描述至少需要10个字符' },
                    ]}
                    extra="建议写清现象、影响范围和期望结果。"
                  >
                    <TextArea
                      rows={6}
                      placeholder="请详细描述问题/需求与影响范围..."
                      aria-required="true"
                      data-testid="ticket-description-input"
                    />
                  </Form.Item>

                  <Row gutter={[16, 16]}>
                    <Col xs={24} sm={12}>
                      <Form.Item
                        name="priority"
                        label="优先级"
                        initialValue="medium"
                        rules={[{ required: true }]}
                      >
                        <AppSelect
                          allowClear
                          options={[
                            { label: '低', value: 'low' },
                            { label: '中', value: 'medium' },
                            { label: '高', value: 'high' },
                            { label: '紧急', value: 'urgent' },
                          ]}
                          placeholder="选择优先级"
                          aria-label="选择工单优先级"
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={12}>
                      <Form.Item name="category" label="分类">
                        <AppSelect
                          allowClear
                          loading={categoryLoading}
                          options={categoryOptions}
                          placeholder={categoryLoading ? '加载分类中…' : '选择分类'}
                          aria-label="选择工单分类"
                        />
                      </Form.Item>
                    </Col>
                  </Row>
                </Card>
              )}

              <Space
                orientation="vertical"
                size="middle"
                style={{ width: '100%' }}
                role="group"
                aria-label="表单操作按钮"
              >
                <Button
                  type="primary"
                  onClick={handleSubmit}
                  loading={loading}
                  size="large"
                  block
                  aria-busy={loading}
                  data-testid="ticket-submit-button"
                >
                  创建工单
                </Button>
                <Button
                  onClick={() => router.push('/tickets')}
                  size="large"
                  block
                  aria-label="取消创建，返回工单列表"
                >
                  {t('common.cancel')}
                </Button>
              </Space>

              {/* AI 智能分类区域 */}
              <Card
                size="small"
                className="mt-4"
                title={
                  <span className="flex items-center gap-2">
                    <Sparkles className="w-4 h-4 text-yellow-500" />
                    AI 辅助分类
                  </span>
                }
              >
                <Spin spinning={aiLoading}>
                  {aiError && <Alert title={aiError} type="warning" showIcon className="mb-2" />}
                  {aiSuggestions ? (
                    <div className="space-y-2">
                      <div className="flex flex-wrap gap-2">
                        {aiSuggestions.category && (
                          <Tag color="blue">分类: {aiSuggestions.category}</Tag>
                        )}
                        {aiSuggestions.priority && (
                          <Tag color={aiSuggestions.priority === 'urgent' ? 'red' : 'orange'}>
                            优先级: {aiSuggestions.priority}
                          </Tag>
                        )}
                        {aiSuggestions.urgency && (
                          <Tag color="purple">紧急度: {aiSuggestions.urgency}</Tag>
                        )}
                      </div>
                      {aiSuggestions.reasoning && (
                        <Text type="secondary" className="text-sm">
                          {aiSuggestions.reasoning}
                        </Text>
                      )}
                      {aiSuggestions.confidence && (
                        <Text type="secondary" className="text-xs">
                          置信度: {Math.round(aiSuggestions.confidence * 100)}%
                        </Text>
                      )}
                    </div>
                  ) : (
                    <Text type="secondary">点击下方按钮获取AI智能分类建议</Text>
                  )}
                </Spin>
                <Button
                  type="default"
                  icon={<Sparkles className="w-4 h-4" />}
                  onClick={handleAITriage}
                  loading={aiLoading}
                  className="mt-2"
                  block
                >
                  获取 AI 建议
                </Button>
              </Card>
            </Form>
          </Col>
        </Row>
      </Space>
    </div>
  );
}
