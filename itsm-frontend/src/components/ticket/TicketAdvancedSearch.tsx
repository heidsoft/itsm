'use client';

import React, { useState, useCallback, useMemo } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  DatePicker,
  Button,
  Space,
  Row,
  Col,
  Divider,
  Tag,
  Tooltip,
  Collapse,
} from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  ClearOutlined,
  SaveOutlined,
  FilterOutlined,
} from '@ant-design/icons';
import type { FormInstance } from 'antd';
import dayjs from 'dayjs';
import type { RangePickerProps } from 'antd/es/date-picker';

const { RangePicker } = DatePicker;
const { Option } = Select;
const { Panel } = Collapse;
const { TextArea } = Input;

interface AdvancedSearchFilters {
  // 基础信息
  keyword?: string;
  ticket_number?: string;
  title?: string;
  description?: string;
  
  // 状态和分类
  status?: string[];
  priority?: string[];
  type?: string[];
  category?: string[];
  
  // 人员
  reporter_id?: number;
  assignee_id?: number;
  created_by?: number;
  
  // 时间范围
  created_after?: string;
  created_before?: string;
  updated_after?: string;
  updated_before?: string;
  due_after?: string;
  due_before?: string;
  resolved_after?: string;
  resolved_before?: string;
  
  // 配置项
  configuration_item_id?: number;
  
  // 来源和渠道
  source?: string[];
  channel?: string[];
  
  // SLA相关
  sla_breach?: boolean;
  sla_warning?: boolean;
  
  // 自定义字段
  tags?: string[];
  metadata?: Record<string, any>;
}

interface TicketAdvancedSearchProps {
  onSearch: (filters: AdvancedSearchFilters) => void;
  onReset: () => void;
  loading?: boolean;
  initialValues?: Partial<AdvancedSearchFilters>;
}

// 预定义的筛选选项
const TICKET_STATUS_OPTIONS = [
  { label: '新建', value: 'new', color: 'blue' },
  { label: '待处理', value: 'open', color: 'blue' },
  { label: '处理中', value: 'in_progress', color: 'orange' },
  { label: '等待中', value: 'pending', color: 'yellow' },
  { label: '已解决', value: 'resolved', color: 'green' },
  { label: '已关闭', value: 'closed', color: 'default' },
  { label: '已取消', value: 'cancelled', color: 'red' },
];

const TICKET_PRIORITY_OPTIONS = [
  { label: '低', value: 'low', color: 'green' },
  { label: '中', value: 'medium', color: 'orange' },
  { label: '高', value: 'high', color: 'red' },
  { label: '紧急', value: 'urgent', color: 'purple' },
  { label: '严重', value: 'critical', color: 'red' },
];

const TICKET_TYPE_OPTIONS = [
  { label: '事件', value: 'incident' },
  { label: '请求', value: 'request' },
  { label: '问题', value: 'problem' },
  { label: '变更', value: 'change' },
  { label: '任务', value: 'task' },
];

const TICKET_SOURCE_OPTIONS = [
  { label: '邮件', value: 'email' },
  { label: '电话', value: 'phone' },
  { label: '门户网站', value: 'portal' },
  { label: 'API', value: 'api' },
  { label: '监控', value: 'monitoring' },
  { label: '手动', value: 'manual' },
];

// 预设搜索模板
const SEARCH_TEMPLATES = [
  {
    name: '我的待办工单',
    description: '分配给我且未完成的工单',
    filters: {
      status: ['new', 'open', 'in_progress'],
      assignee_id: 1, // 当前用户ID
    },
  },
  {
    name: '高优先级工单',
    description: '所有高优先级和紧急工单',
    filters: {
      priority: ['high', 'urgent', 'critical'],
    },
  },
  {
    name: 'SLA即将超时',
    description: 'SLA即将超时的工单',
    filters: {
      sla_warning: true,
    },
  },
  {
    name: '本周创建工单',
    description: '本周内创建的所有工单',
    filters: {
      created_after: dayjs().startOf('week').format('YYYY-MM-DD'),
      created_before: dayjs().endOf('week').format('YYYY-MM-DD'),
    },
  },
  {
    name: '已解决未关闭',
    description: '已解决但未关闭的工单',
    filters: {
      status: ['resolved'],
    },
  },
];

const TicketAdvancedSearch: React.FC<TicketAdvancedSearchProps> = ({
  onSearch,
  onReset,
  loading = false,
  initialValues = {},
}) => {
  const [form] = Form.useForm();
  const [activeTemplate, setActiveTemplate] = useState<string>('');
  const [savedSearches, setSavedSearches] = useState<any[]>([]);

  // 应用搜索模板
  const applyTemplate = useCallback((template: typeof SEARCH_TEMPLATES[0]) => {
    form.setFieldsValue(template.filters);
    setActiveTemplate(template.name);
  }, [form]);

  // 保存当前搜索条件
  const saveSearch = useCallback(() => {
    const values = form.getFieldsValue();
    const searchName = `搜索_${dayjs().format('YYYY-MM-DD_HH-mm-ss')}`;
    
    setSavedSearches(prev => [
      ...prev,
      {
        id: Date.now(),
        name: searchName,
        filters: values,
        createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
      },
    ]);
  }, [form]);

  // 执行搜索
  const handleSearch = useCallback(() => {
    const values = form.getFieldsValue();
    
    // 处理日期范围
    const processedValues: AdvancedSearchFilters = {};
    
    Object.keys(values).forEach(key => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '' && 
          (Array.isArray(value) ? value.length > 0 : true)) {
        processedValues[key as keyof AdvancedSearchFilters] = value;
      }
    });

    onSearch(processedValues);
  }, [form, onSearch]);

  // 重置搜索条件
  const handleReset = useCallback(() => {
    form.resetFields();
    setActiveTemplate('');
    onReset();
  }, [form, onReset]);

  // 快速日期范围
  const quickDateRanges: RangePickerProps['ranges'] = {
    '今天': [dayjs().startOf('day'), dayjs().endOf('day')],
    '昨天': [dayjs().subtract(1, 'day').startOf('day'), dayjs().subtract(1, 'day').endOf('day')],
    '本周': [dayjs().startOf('week'), dayjs().endOf('week')],
    '上周': [dayjs().subtract(1, 'week').startOf('week'), dayjs().subtract(1, 'week').endOf('week')],
    '本月': [dayjs().startOf('month'), dayjs().endOf('month')],
    '上月': [dayjs().subtract(1, 'month').startOf('month'), dayjs().subtract(1, 'month').endOf('month')],
    '最近7天': [dayjs().subtract(7, 'day'), dayjs()],
    '最近30天': [dayjs().subtract(30, 'day'), dayjs()],
  };

  return (
    <Card 
      title={
        <div className="flex items-center justify-between">
          <div className="flex items-center">
            <FilterOutlined className="mr-2" />
            <span>高级搜索</span>
          </div>
          <Space>
            <Button size="small" icon={<SaveOutlined />} onClick={saveSearch}>
              保存搜索
            </Button>
          </Space>
        </div>
      }
      size="small"
    >
      {/* 预设模板 */}
      <div className="mb-4">
        <Text strong className="mb-2 block">快速搜索模板：</Text>
        <Space wrap>
          {SEARCH_TEMPLATES.map(template => (
            <Button
              key={template.name}
              size="small"
              type={activeTemplate === template.name ? 'primary' : 'default'}
              onClick={() => applyTemplate(template)}
            >
              <Tooltip title={template.description}>
                {template.name}
              </Tooltip>
            </Button>
          ))}
        </Space>
      </div>

      <Divider />

      <Form
        form={form}
        layout="vertical"
        initialValues={initialValues}
        onFinish={handleSearch}
      >
        <Collapse defaultActiveKey={['basic']} ghost>
          {/* 基础信息 */}
          <Panel header="基础信息" key="basic">
            <Row gutter={[16, 0]}>
              <Col span={6}>
                <Form.Item label="关键字搜索" name="keyword">
                  <Input
                    placeholder="标题、描述、工单号..."
                    prefix={<SearchOutlined />}
                  />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="工单号" name="ticket_number">
                  <Input placeholder="精确工单号" />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="工单标题" name="title">
                  <Input placeholder="工单标题" />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="工单描述" name="description">
                  <TextArea rows={1} placeholder="工单描述" />
                </Form.Item>
              </Col>
            </Row>
          </Panel>

          {/* 状态分类 */}
          <Panel header="状态和分类" key="status">
            <Row gutter={[16, 0]}>
              <Col span={6}>
                <Form.Item label="状态" name="status">
                  <Select mode="multiple" placeholder="选择状态" allowClear>
                    {TICKET_STATUS_OPTIONS.map(option => (
                      <Option key={option.value} value={option.value}>
                        <Tag color={option.color}>{option.label}</Tag>
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="优先级" name="priority">
                  <Select mode="multiple" placeholder="选择优先级" allowClear>
                    {TICKET_PRIORITY_OPTIONS.map(option => (
                      <Option key={option.value} value={option.value}>
                        <Tag color={option.color}>{option.label}</Tag>
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="工单类型" name="type">
                  <Select mode="multiple" placeholder="选择类型" allowClear>
                    {TICKET_TYPE_OPTIONS.map(option => (
                      <Option key={option.value} value={option.value}>
                        {option.label}
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="来源" name="source">
                  <Select mode="multiple" placeholder="选择来源" allowClear>
                    {TICKET_SOURCE_OPTIONS.map(option => (
                      <Option key={option.value} value={option.value}>
                        {option.label}
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
            </Row>
          </Panel>

          {/* 时间范围 */}
          <Panel header="时间范围" key="time">
            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item label="创建时间" name="created_range">
                  <RangePicker
                    style={{ width: '100%' }}
                    ranges={quickDateRanges}
                    format="YYYY-MM-DD"
                    onChange={(dates) => {
                      if (dates && dates[0] && dates[1]) {
                        form.setFieldsValue({
                          created_after: dates[0].format('YYYY-MM-DD'),
                          created_before: dates[1].format('YYYY-MM-DD'),
                        });
                      } else {
                        form.setFieldsValue({
                          created_after: undefined,
                          created_before: undefined,
                        });
                      }
                    }}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="更新时间" name="updated_range">
                  <RangePicker
                    style={{ width: '100%' }}
                    ranges={quickDateRanges}
                    format="YYYY-MM-DD"
                    onChange={(dates) => {
                      if (dates && dates[0] && dates[1]) {
                        form.setFieldsValue({
                          updated_after: dates[0].format('YYYY-MM-DD'),
                          updated_before: dates[1].format('YYYY-MM-DD'),
                        });
                      } else {
                        form.setFieldsValue({
                          updated_after: undefined,
                          updated_before: undefined,
                        });
                      }
                    }}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="截止时间" name="due_range">
                  <RangePicker
                    style={{ width: '100%' }}
                    ranges={quickDateRanges}
                    format="YYYY-MM-DD"
                    onChange={(dates) => {
                      if (dates && dates[0] && dates[1]) {
                        form.setFieldsValue({
                          due_after: dates[0].format('YYYY-MM-DD'),
                          due_before: dates[1].format('YYYY-MM-DD'),
                        });
                      } else {
                        form.setFieldsValue({
                          due_after: undefined,
                          due_before: undefined,
                        });
                      }
                    }}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="解决时间" name="resolved_range">
                  <RangePicker
                    style={{ width: '100%' }}
                    ranges={quickDateRanges}
                    format="YYYY-MM-DD"
                    onChange={(dates) => {
                      if (dates && dates[0] && dates[1]) {
                        form.setFieldsValue({
                          resolved_after: dates[0].format('YYYY-MM-DD'),
                          resolved_before: dates[1].format('YYYY-MM-DD'),
                        });
                      } else {
                        form.setFieldsValue({
                          resolved_after: undefined,
                          resolved_before: undefined,
                        });
                      }
                    }}
                  />
                </Form.Item>
              </Col>
            </Row>
          </Panel>

          {/* SLA相关 */}
          <Panel header="SLA监控" key="sla">
            <Row gutter={[16, 0]}>
              <Col span={6}>
                <Form.Item label="SLA状态" name="sla_status">
                  <Select placeholder="选择SLA状态" allowClear>
                    <Option value="breach">已超时</Option>
                    <Option value="warning">即将超时</Option>
                    <Option value="normal">正常</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>
          </Panel>
        </Collapse>

        {/* 操作按钮 */}
        <div className="flex justify-between items-center mt-6">
          <Space>
            <Button 
              type="primary" 
              icon={<SearchOutlined />} 
              htmlType="submit"
              loading={loading}
            >
              搜索
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
              重置
            </Button>
          </Space>
          
          {savedSearches.length > 0 && (
            <Select
              placeholder="已保存的搜索"
              style={{ width: 200 }}
              onChange={(value) => {
                const search = savedSearches.find(s => s.id === value);
                if (search) {
                  form.setFieldsValue(search.filters);
                }
              }}
              allowClear
            >
              {savedSearches.map(search => (
                <Option key={search.id} value={search.id}>
                  {search.name} ({dayjs(search.createdAt).fromNow()})
                </Option>
              ))}
            </Select>
          )}
        </div>
      </Form>
    </Card>
  );
};

export default TicketAdvancedSearch;