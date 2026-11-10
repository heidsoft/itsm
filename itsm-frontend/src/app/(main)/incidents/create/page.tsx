'use client';

import React, { useState, useEffect } from 'react';
import { Button, Card, Form, Input, Select, Upload, Space, Row, Col, App, Tabs, Typography, Divider, Tag, Spin } from 'antd';
import { ArrowLeft, Search, X, Sparkles } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { IncidentAPI } from '@/lib/api/incident-api';
import { IncidentCategoryOptions } from '@/constants/taxonomy';
import type { ConfigurationItem } from '@/types/biz/cmdb';
import { CMDBApi } from '@/lib/api/cmdb-api';
import type { User } from '@/lib/api/user-api';
import { UserApi } from '@/lib/api/user-api';
import { useErrorHandler } from '@/lib/hooks/useErrorHandler';
import { AIApi, type TriageResult, type RagAnswer } from '@/lib/api/ai-api';
import { notify } from '@/lib/notify';

const { Title, Text } = Typography;
const { TextArea } = Input;

// AI 建议回填白名单：只接受与表单选项一致的取值，避免写入无效枚举
const PRIORITY_VALUES = ['critical', 'high', 'medium', 'low'];
const CATEGORY_VALUES = IncidentCategoryOptions.map(option => option.value);

// AI 建议标签的展示文案。此前标签直接打印枚举值，界面出现
// 「优先级：high」「分类：hardware」这类英文标识，与下方中文下拉自相矛盾。
const PRIORITY_LABELS: Record<string, string> = {
  critical: 'Urgent',
  high: 'High',
  medium: 'Medium',
  low: 'Low',
};
const CATEGORY_LABELS: Record<string, string> = IncidentCategoryOptions.reduce<
  Record<string, string>
>((acc, option) => {
  acc[option.value] = option.label;
  return acc;
}, {});

const labelOf = (map: Record<string, string>, value: string): string => map[value] ?? value;

// CI状态中文映射
const ciStatusNameMap: Record<string, string> = {
  active: 'Active',
  inactive: 'Inactive',
  maintenance: 'Maintenance',
  retired: 'Offline',
};

// 表单值类型定义
interface IncidentFormValues {
  title: string;
  description: string;
  priority: 'critical' | 'high' | 'medium' | 'low';
  source: 'manual' | 'monitoring' | 'system' | 'user';
  type: 'incident' | 'service_request' | 'security_event' | 'alert';
  category?: string;
  impact?: 'critical' | 'high' | 'medium' | 'low';
  urgency?: 'critical' | 'high' | 'medium' | 'low';
  assignedTo?: number;
  affectedSystems?: string[];
  rootCause?: string;
}

export default function CreateIncidentPage() {
  const router = useRouter();
  const [form] = Form.useForm();
  // 提交动作态（表单页首帧是空表单，无列表假空态问题）
  const [actionLoading, setActionLoading] = useState(false);
  const { message } = App.useApp();
  const [activeTab, setActiveTab] = useState('basic');
  const [selectedCIs, setSelectedCIs] = useState<ConfigurationItem[]>([]);
  const [ciSearchTerm, setCISearchTerm] = useState('');
  const [ciSearchResults, setCISearchResults] = useState<ConfigurationItem[]>([]);
  const [ciSearching, setCISearching] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const { handleError } = useErrorHandler();

  // AI 智能辅助：分类建议 + 相似历史事件
  // 说明：AI 建议一律「显式采纳」，不自动回填表单，避免用户不知情被改写输入
  const [aiLoading, setAiLoading] = useState(false);
  const [aiSuggestion, setAiSuggestion] = useState<TriageResult | null>(null);
  const [similarIncidents, setSimilarIncidents] = useState<RagAnswer[]>([]);

  // 加载用户列表
  useEffect(() => {
    const fetchUsers = async () => {
      setUsersLoading(true);
      try {
        const response = await UserApi.getUsers({ page: 1, pageSize: 100 });
        setUsers(response.items || []);
      } catch (error) {
        // 用户列表加载失败不阻塞页面，使用空列表
        setUsers([]);
      } finally {
        setUsersLoading(false);
      }
    };
    fetchUsers();
  }, []);

  // CI配置项搜索
  // effect 只依赖搜索词：handleError 每次 render 都是新引用，放进依赖会形成
  // setCISearchResults → 重渲染 → 新 handleError → 再触发 effect 的死循环
  // （Maximum update depth exceeded）。
  useEffect(() => {
    if (ciSearchTerm.trim()) {
      const fetchCIs = async () => {
        setCISearching(true);
        try {
          // 后端 CI 列表只绑定 search；传 keyword 不会报错，但搜索词被静默忽略，
          // 下拉里出来的是未过滤的第一页。
          const result = await CMDBApi.getCIs({ search: ciSearchTerm });
          setCISearchResults(result.items);
        } catch (error) {
          handleError(error, 'getCIs', 'Failed to search for configuration items');
          setCISearchResults([]);
        } finally {
          setCISearching(false);
        }
      };

      const timeoutId = setTimeout(fetchCIs, 300);
      return () => clearTimeout(timeoutId);
    } else {
      setCISearchResults([]);
    }
  }, [ciSearchTerm]);

  const handleAddCI = (ci: ConfigurationItem) => {
    if (!selectedCIs.find(item => item.id === ci.id)) {
      setSelectedCIs([...selectedCIs, ci]);
    }
    setCISearchTerm('');
    setCISearchResults([]);
  };

  const handleRemoveCI = (ciId: number) => {
    setSelectedCIs(selectedCIs.filter(ci => ci.id !== ciId));
  };

  /**
   * AI 智能分析：并行请求分类建议与相似历史事件。
   * 两项独立降级——任一失败不影响另一项，避免用户产生「AI 整体不可用」的错觉。
   */
  const handleAIAnalyze = async () => {
    const title: string = form.getFieldValue('title') || '';
    const description: string = form.getFieldValue('description') || '';

    if (!title.trim() && !description.trim()) {
      notify.warning('Please fill in the event title or description first so that AI can give suggestions based on it.');
      return;
    }

    setAiLoading(true);
    setAiSuggestion(null);
    setSimilarIncidents([]);

    const [triageRes, similarRes] = await Promise.allSettled([
      AIApi.triage(title, description),
      AIApi.similarIncidents(`${title} ${description}`.trim(), 3),
    ]);

    let hasResult = false;
    if (triageRes.status === 'fulfilled' && triageRes.value) {
      setAiSuggestion(triageRes.value);
      hasResult = true;
    }
    if (similarRes.status === 'fulfilled' && similarRes.value?.incidents?.length) {
      setSimilarIncidents(similarRes.value.incidents);
      hasResult = true;
    }

    setAiLoading(false);

    if (!hasResult) {
      notify.aiUnavailable('AI intelligent analysis');
    }
  };

  /** 采纳 AI 建议：仅回填白名单内且有值的字段，并明确告知采纳了哪些 */
  const handleApplySuggestion = () => {
    if (!aiSuggestion) return;

    const patch: Record<string, unknown> = {};
    const applied: string[] = [];

    if (aiSuggestion.priority && PRIORITY_VALUES.includes(aiSuggestion.priority)) {
      patch.priority = aiSuggestion.priority;
      applied.push('Priority');
    }
    if (aiSuggestion.urgency && PRIORITY_VALUES.includes(aiSuggestion.urgency)) {
      patch.urgency = aiSuggestion.urgency;
      applied.push('Urgency');
    }
    if (aiSuggestion.category && CATEGORY_VALUES.includes(aiSuggestion.category)) {
      patch.category = aiSuggestion.category;
      applied.push('Category');
    }

    if (applied.length === 0) {
      notify.warning('The AI suggested fields do not match the current form options, please select manually');
      return;
    }

    form.setFieldsValue(patch);
    notify.success(`Adopted ${applied.join('、')}, you can still adjust it manually`);
  };

  const handleSubmit = async (values: IncidentFormValues) => {
    setActionLoading(true);
    try {
      await IncidentAPI.createIncident({
        title: values.title,
        description: values.description,
        priority: values.priority,
        source: values.source || 'manual',
        type: values.type || 'incident',
        category: values.category,
        impact: values.impact,
        urgency: values.urgency,
        assigneeId: values.assignedTo,
        configurationItemIds: selectedCIs.map(ci => ci.id),
      });
      message.success('Incident created successfully');
      router.push('/incidents');
    } catch (error) {
      handleError(error, 'createIncident', 'Creation failed, please try again');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 返回按钮 */}
      <div className="mb-6">
        <Button
          type="link"
          icon={<ArrowLeft />}
          onClick={() => router.back()}
          style={{ paddingLeft: 0 }}
        >

          Back to List
        </Button>
      </div>

      {/* 页面标题 */}
      <div className="mb-6">
        <Title level={2} style={{ marginBottom: 4 }}>Create event</Title>
        <Text type="secondary">Fill in event information to create a new event record</Text>
      </div>

      <Row gutter={24}>
        {/* 左侧表单 */}
        <Col xs={24} lg={16}>
          <Card>
            <Form
              form={form}
              data-testid="incident-create-form"
              layout="vertical"
              onFinish={handleSubmit}
              initialValues={{
                priority: 'medium',
                impact: 'medium',
                urgency: 'medium',
                source: 'manual',
                type: 'incident',
              }}
            >
              <Tabs
                activeKey={activeTab}
                onChange={setActiveTab}
                items={[
                  {
                    key: 'basic',
                    label: 'Basic Info',
                    children: (
                      <>
                        <Form.Item
                          name="title"
                          label="Incident Title"
                          rules={[{ required: true, message: 'Please enter event title' }]}
                        >
                          <Input placeholder="Briefly describe the event" maxLength={200} showCount data-testid="incident-title-input" />
                        </Form.Item>

                        <Form.Item
                          name="description"
                          label="Detailed description"
                          rules={[{ required: true, message: 'Please enter event description' }]}
                        >
                          <TextArea
                            rows={6}
                            placeholder="Detailed description of the occurrence, scope of impact, error message, etc."
                            data-testid="incident-description-input"
                          />
                        </Form.Item>

                        {/* AI 智能辅助：分类建议 + 相似历史事件 */}
                        <div className="mb-4">
                          <Button
                            icon={<Sparkles />}
                            onClick={handleAIAnalyze}
                            loading={aiLoading}
                            data-testid="incident-ai-analyze-btn"
                          >
                            {aiLoading ? 'AI analysis in progress…' : 'AI intelligent analysis'}
                          </Button>
                          <Text type="secondary" className="ml-2" style={{ fontSize: 12 }}>

                            Recommend priority/category based on title and description, and retrieve similar historical events for reference
                          </Text>
                        </div>

                        {aiSuggestion && (
                          <Card
                            size="small"
                            className="mb-4"
                            title={
                              <Space>
                                <Sparkles size={14} />
                                <span>AI classification suggestions</span>
                                <Tag color="blue">

                                  Confidence{' '}
                                  {Math.round(
                                    aiSuggestion.confidence > 1
                                      ? aiSuggestion.confidence
                                      : aiSuggestion.confidence * 100
                                  )}
                                  %
                                </Tag>
                              </Space>
                            }
                            extra={
                              <Button
                                type="link"
                                size="small"
                                onClick={handleApplySuggestion}
                                data-testid="incident-ai-apply-btn"
                              >

                                Adopt suggestion
                              </Button>
                            }
                          >
                            <Space orientation="vertical" size={4} style={{ width: '100%' }}>
                              <Space wrap size={4}>
                                {aiSuggestion.priority && (
                                  <Tag color="red">

                                    Priority:{labelOf(PRIORITY_LABELS, aiSuggestion.priority)}
                                  </Tag>
                                )}
                                {aiSuggestion.urgency && (
                                  <Tag color="orange">

                                    Urgency:{labelOf(PRIORITY_LABELS, aiSuggestion.urgency)}
                                  </Tag>
                                )}
                                {aiSuggestion.category && (
                                  <Tag color="purple">

                                    Category:{labelOf(CATEGORY_LABELS, aiSuggestion.category)}
                                  </Tag>
                                )}
                              </Space>
                              {aiSuggestion.explanation && (
                                <Text type="secondary" style={{ fontSize: 12 }}>

                                  Based on:{aiSuggestion.explanation}
                                </Text>
                              )}
                              <Text type="secondary" style={{ fontSize: 12 }}>

                                The suggestions are for reference only. You need to click "Accept Suggestions" before they will be written into the form.
                              </Text>
                            </Space>
                          </Card>
                        )}

                        {similarIncidents.length > 0 && (
                          <Card size="small" className="mb-4" title="Similar historical events">
                            <Space orientation="vertical" size={8} style={{ width: '100%' }}>
                              {similarIncidents.map(item => (
                                <div
                                  key={`${item.objectType}-${item.id}`}
                                  style={{
                                    borderLeft: '3px solid #d9d9d9',
                                    paddingLeft: 8,
                                  }}
                                >
                                  <Space size={6} wrap>
                                    <a
                                      href={`/incidents/${item.id}`}
                                      target="_blank"
                                      rel="noopener noreferrer"
                                      style={{ fontWeight: 500 }}
                                    >
                                      {item.title || `Event #${item.id}`}
                                    </a>
                                    {typeof item.score === 'number' && (
                                      <Tag>Matching degree {Math.round(item.score * 100)}%</Tag>
                                    )}
                                    {item.authorityLevel !== undefined && (
                                      <Tag color={item.authorityLevel >= 20 ? 'green' : 'default'}>
                                        {item.authorityLevel >= 30
                                          ? 'The One Source of Truth'
                                          : item.authorityLevel >= 20
                                            ? 'official standard'
                                            : item.authorityLevel >= 10
                                              ? 'Department recommendation'
                                              : 'Normal'}
                                      </Tag>
                                    )}
                                  </Space>
                                  {item.snippet && (
                                    <div
                                      style={{
                                        color: 'rgba(0,0,0,0.45)',
                                        fontSize: 12,
                                        marginTop: 2,
                                      }}
                                    >
                                      {item.snippet}
                                    </div>
                                  )}
                                </div>
                              ))}
                            </Space>
                          </Card>
                        )}

                        <Row gutter={16}>
                          <Col span={8}>
                            <Form.Item
                              name="priority"
                              label="Priority"
                              rules={[{ required: true }]}
                            >
                              <Select options={[
                                { value: 'critical', label: 'Urgent' },
                                { value: 'high', label: 'High' },
                                { value: 'medium', label: 'Medium' },
                                { value: 'low', label: 'Low' },
                              ]} />
                            </Form.Item>
                          </Col>
                          <Col span={8}>
                            <Form.Item
                              name="source"
                              label="Source"
                              rules={[{ required: true }]}
                            >
                              <Select options={[
                                { value: 'manual', label: 'Manually created' },
                                { value: 'monitoring', label: 'Monitoring alarm' },
                                { value: 'system', label: 'System' },
                                { value: 'user', label: 'Users' },
                              ]} />
                            </Form.Item>
                          </Col>
                          <Col span={8}>
                            <Form.Item
                              name="type"
                              label="Type"
                              rules={[{ required: true }]}
                            >
                              <Select options={[
                                { value: 'incident', label: 'Incident' },
                                { value: 'service_request', label: 'Service Request' },
                                { value: 'security_event', label: 'Security Incident' },
                                { value: 'alert', label: 'Alarm' },
                              ]} />
                            </Form.Item>
                          </Col>
                        </Row>
                        <Row gutter={16}>
                          <Col span={12}>
                            <Form.Item
                              name="category"
                              label="Incident Category"
                            >
                              <Select placeholder="Select category" options={IncidentCategoryOptions} />
                            </Form.Item>
                          </Col>
                          <Col span={12}>
                            <Form.Item
                              name="assignedTo"
                              label="Assign To"
                            >
                              <Select
                                placeholder="Select person in charge"
                                allowClear
                                loading={usersLoading}
                                showSearch
                                optionFilterProp="children"
                                options={users.map(user => ({
                                  value: user.id,
                                  label: user.name || user.username,
                                }))}
                              />
                            </Form.Item>
                          </Col>
                        </Row>

                        {/* CI配置项关联 */}
                        <Form.Item label="Associated configuration items">
                          <div className="space-y-3">
                            {/* 已选择的CI列表 */}
                            {selectedCIs.length > 0 && (
                              <div className="flex flex-wrap gap-2">
                                {selectedCIs.map(ci => (
                                  <Tag
                                    key={ci.id}
                                    closable
                                    onClose={() => handleRemoveCI(ci.id)}
                                    color="blue"
                                  >
									{ci.name} ({ci.type || 'CI'})
                                  </Tag>
                                ))}
                              </div>
                            )}

                            {/* CI搜索框 */}
                            <div className="relative">
                              <Input
                                placeholder="Search configuration item name or number"
                                prefix={<Search />}
                                value={ciSearchTerm}
                                onChange={e => setCISearchTerm(e.target.value)}
                                suffix={ciSearching ? <Spin size="small" /> : null}
                              />

                              {/* 搜索结果下拉 */}
                              {ciSearchResults.length > 0 && (
                                <div className="absolute z-10 w-full mt-1 bg-white border rounded-md shadow-lg max-h-60 overflow-auto">
                                  {ciSearchResults.map(ci => (
                                    <div
                                      key={ci.id}
                                      className="px-3 py-2 hover:bg-gray-50 cursor-pointer flex justify-between items-center"
                                      onClick={() => handleAddCI(ci)}
                                    >
                                      <div>
                                        <div className="font-medium">{ci.name}</div>
										<div className="text-xs text-gray-500">{ci.type || 'CI'} - {ciStatusNameMap[ci.status] || ci.status}</div>
                                      </div>
                                      {selectedCIs.find(item => item.id === ci.id) && (
                                        <Tag color="green">Selected</Tag>
                                      )}
                                    </div>
                                  ))}
                                </div>
                              )}
                            </div>
                          </div>
                        </Form.Item>
                      </>
                    ),
                  },
                  {
                    key: 'impact',
                    label: 'Impact Analysis',
                    children: (
                      <>
                        <Row gutter={16}>
                          <Col span={12}>
                            <Form.Item
                              name="impact"
                              label="Impact Range"
                              rules={[{ required: true }]}
                            >
                              <Select options={[
                                { value: 'critical', label: 'Global' },
                                { value: 'high', label: 'Department level' },
                                { value: 'medium', label: 'Team level' },
                                { value: 'low', label: 'Personal' },
                              ]} />
                            </Form.Item>
                          </Col>
                          <Col span={12}>
                            <Form.Item
                              name="urgency"
                              label="Urgency"
                              rules={[{ required: true }]}
                            >
                              <Select options={[
                                { value: 'critical', label: 'Urgent' },
                                { value: 'high', label: 'High' },
                                { value: 'medium', label: 'Medium' },
                                { value: 'low', label: 'Low' },
                              ]} />
                            </Form.Item>
                          </Col>
                        </Row>

                        <Form.Item
                          name="affectedSystems"
                          label="Affected systems"
                        >
                          <Select
                            mode="multiple"
                            placeholder="Select affected systems"
                            allowClear
                            options={[
                              { value: 'web', label: 'Web site' },
                              { value: 'api', label: 'API Service' },
                              { value: 'database', label: 'Database' },
                              { value: 'network', label: 'Network' },
                              { value: 'storage', label: 'Storage' },
                            ]}
                          />
                        </Form.Item>

                        <Form.Item
                          name="rootCause"
                          label="Preliminary cause analysis"
                        >
                          <TextArea
                            rows={4}
                            placeholder="Preliminary analysis of possible causes"
                          />
                        </Form.Item>
                      </>
                    ),
                  },
                  {
                    key: 'attachment',
                    label: 'Attachments',
                    children: (
                      <>
                        <Form.Item
                          name="attachments"
                          label="Upload Attachment"
                          valuePropName="fileList"
                          getValueFromEvent={(e) => {
                            if (Array.isArray(e)) return e;
                            return e?.fileList;
                          }}
                        >
                          <Upload name="logo" action="/upload.do" listType="text">
                            <Button icon={<Upload />}>Upload Attachment</Button>
                          </Upload>
                        </Form.Item>
                        <Text type="secondary">

                          Supports uploading attachments such as pictures and documents. A single file does not exceed 10MB.
                        </Text>
                      </>
                    ),
                  },
                ]}
              />

              <Divider />

              <Form.Item className="!mb-0">
                <Space>
                  <Button
                    type="primary"
                    htmlType="submit"
                    loading={actionLoading}
                    data-testid="incident-submit-button"
                  >

                    Submit
                  </Button>
                  <Button onClick={() => router.back()}>Cancel</Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        </Col>

        {/* 右侧信息 */}
        <Col xs={24} lg={8}>
          <Card title="Create prompt" className="mb-4">
            <Space orientation="vertical" className="w-full">
              <div>
                <Text strong>Priority description</Text>
                <ul className="mt-2 text-sm text-gray-600">
                  <li>🔴 Emergency: The system is completely unavailable</li>
                  <li>🟠 High: Core functions are affected</li>
                  <li>🔵 Medium: Non-core functions are affected</li>
                  <li>🟢 Low: Minor problem</li>
                </ul>
              </div>
              <Divider className="!my-2" />
              <div>
                <Text strong>Emergency Contact Information</Text>
                <ul className="mt-2 text-sm text-gray-600">
                  <li>Tel: 400-XXX-XXXX</li>
                  <li>Email: support@example.com</li>
                </ul>
              </div>
            </Space>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
