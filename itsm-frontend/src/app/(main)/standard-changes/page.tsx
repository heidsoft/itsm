'use client';

import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Input,
  Space,
  Tag,
  Modal,
  Form,
  message,
  Popconfirm,
  Card,
  Row,
  Col,
  Select,
  Typography,
  Descriptions,
  Empty,
  Tooltip,
  DatePicker,
} from 'antd';
import {
  Plus,
  Search,
  Edit,
  Delete,
  PlayCircle,
  FolderOpen,
  AlertTriangle,
  Clock,
  CheckCircle,
  AlertCircle,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import type { StandardChange } from '@/lib/api/standard-change-api';
import { StandardChangeApi } from '@/lib/api/standard-change-api';
import { useI18n } from '@/lib/i18n';
import { ChangeRiskLabels, ChangeImpactLabels } from '@/constants/change';

const { Title, Text } = Typography;

const RISK_COLORS: Record<string, string> = {
  low: 'green',
  medium: 'orange',
  high: 'red',
};

const CATEGORY_COLORS: Record<string, string> = {
  server: 'blue',
  network: 'cyan',
  database: 'purple',
  application: 'magenta',
  security: 'red',
  general: 'default',
};

// Labels preserve both current API keys and legacy category values when saving/filtering.
const CATEGORY_LABELS: Readonly<Record<string, string>> = {
  server: 'Servers', network: 'Network', database: 'Database',
  application: 'Application', security: 'Security', general: 'General',
  服务器: 'Servers', 安全: 'Security', 数据库: 'Database', 网络安全: 'Network Security', 应用: 'Application',
  终端支持: 'Endpoint Support', 网络: 'Network', 模板: 'Template',
};
const categoryLabel = (value: string) => Object.hasOwn(CATEGORY_LABELS, value) ? CATEGORY_LABELS[value] : value;

export default function StandardChangesPage() {
  const router = useRouter();
  const { t } = useI18n();
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 「暂无标准变更模板」的空态，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  const [templates, setTemplates] = useState<StandardChange[]>([]);
  const [categories, setCategories] = useState<string[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<StandardChange | null>(null);
  const [form] = Form.useForm();
  const [instantiateModalVisible, setInstantiateModalVisible] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<StandardChange | null>(null);
  const [instantiateForm] = Form.useForm();

  useEffect(() => {
    fetchTemplates();
    fetchCategories();
  }, [page, pageSize, category]);

  const fetchTemplates = async () => {
    setListLoading(true);
    try {
      const data = await StandardChangeApi.getTemplates({
        page,
        pageSize: pageSize,
        category: category || undefined,
        search: search || undefined,
        activeOnly: true,
      });
      setTemplates(data.items);
      setTotal(data.total);
    } catch (error) {
      console.error('Failed to fetch templates:', error);
      message.error(t('standardChanges.loadFailed') || 'Failed to load templates');
    } finally {
      setListLoading(false);
    }
  };

  const fetchCategories = async () => {
    try {
      const data = await StandardChangeApi.getCategories();
      setCategories(data.categories);
    } catch (error) {
      console.error('Failed to fetch categories:', error);
    }
  };

  const handleSearch = () => {
    setPage(1);
    fetchTemplates();
  };

  const handleCreate = () => {
    setEditingTemplate(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (template: StandardChange) => {
    setEditingTemplate(template);
    form.setFieldsValue({
      title: template.title,
      description: template.description,
      implementationPlan: template.implementationPlan,
      rollbackPlan: template.rollbackPlan,
      justification: template.justification,
      category: template.category,
      riskLevel: template.riskLevel,
      impactScope: template.impactScope,
      expectedDuration: template.expectedDuration,
      approvalRequired: template.approvalRequired,
      affectedCis: template.affectedCIs,
      prerequisites: template.prerequisites,
      remarks: template.remarks,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await StandardChangeApi.deleteTemplate(id);
      message.success(t('common.deleteSuccess') || 'Deleted');
      fetchTemplates();
      fetchCategories();
    } catch (error) {
      console.error('Failed to delete template:', error);
      message.error(t('common.deleteFailed') || 'Delete failed');
    }
  };

  const handleInstantiate = (template: StandardChange) => {
    setSelectedTemplate(template);
    instantiateForm.resetFields();
    setInstantiateModalVisible(true);
  };

  const handleInstantiateSubmit = async () => {
    if (!selectedTemplate) return;
    try {
      const values = instantiateForm.getFieldsValue();
      const result = await StandardChangeApi.instantiate(selectedTemplate.id, values);
      message.success(t('standardChanges.instantiateSuccess') || 'Successfully created change from template');
      setInstantiateModalVisible(false);
      router.push(`/changes/${result.changeId}`);
    } catch (error) {
      console.error('Failed to instantiate:', error);
      message.error(t('standardChanges.instantiateFailed') || 'Failed to create change from template');
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingTemplate) {
        await StandardChangeApi.updateTemplate(editingTemplate.id, values);
        message.success(t('common.saveSuccess') || 'Updated');
      } else {
        await StandardChangeApi.createTemplate(values);
        message.success(t('common.submitSuccess') || 'Created');
      }
      setModalVisible(false);
      fetchTemplates();
      fetchCategories();
    } catch (error) {
      console.error('Failed to submit:', error);
      message.error(
        editingTemplate
          ? t('common.saveFailed') || 'Update failed'
          : t('common.submitFailed') || 'Creation failed'
      );
    }
  };

  const columns = [
    {
      title: 'Template Name',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: StandardChange) => (
        <div>
          <Text strong>{text}</Text>
          {record.description && (
            <Text type='secondary' style={{ display: 'block', fontSize: 12 }}>
              {record.description.slice(0, 50)}...
            </Text>
          )}
        </div>
      ),
    },
    {
      title: 'Category',
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (cat: string) => (
        <Tag color={CATEGORY_COLORS[cat] || 'default'}>{categoryLabel(cat || 'general')}</Tag>
      ),
    },
    {
      title: 'Risk Level',
      dataIndex: 'riskLevel',
      key: 'riskLevel',
      width: 100,
      render: (level: string) => (
        <Tag color={RISK_COLORS[level] || 'default'}>
          {ChangeRiskLabels[level as keyof typeof ChangeRiskLabels] || level}
        </Tag>
      ),
    },
    {
      title: 'Impact Range',
      dataIndex: 'impactScope',
      key: 'impactScope',
      width: 100,
      render: (scope: string) => (
        <Tag>{ChangeImpactLabels[scope as keyof typeof ChangeImpactLabels] || scope}</Tag>
      ),
    },
    {
      title: 'Estimated construction period',
      dataIndex: 'expectedDuration',
      key: 'expectedDuration',
      width: 100,
      render: (mins: number) => (mins ? `${mins} minutes` : '-'),
    },
    {
      title: 'No approval required',
      dataIndex: 'approvalRequired',
      key: 'approvalRequired',
      width: 100,
      render: (required: boolean) =>
        required ? (
          <Tag icon={<AlertCircle size={12} />} color='orange'>

            Approval required
          </Tag>
        ) : (
          <Tag icon={<CheckCircle size={12} />} color='green'>

            No approval required
          </Tag>
        ),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 180,
      render: (_: unknown, record: StandardChange) => (
        <Space>
          <Tooltip title="Create changes from template">
            <Button
              aria-label="Create changes from template"
              type='link'
              icon={<PlayCircle size={14} />}
              onClick={() => handleInstantiate(record)}
            />
          </Tooltip>
          <Tooltip title="Edit">
            <Button aria-label="Edit" type='link' icon={<Edit size={14} />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this template?"
            onConfirm={() => handleDelete(record.id)}
            okText="OK"
            cancelText="Cancel"
          >
            <Tooltip title="Delete">
              <Button aria-label="Delete" type='link' danger icon={<Delete size={14} />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className='p-6 bg-gray-50 min-h-full'>
      <div className='mb-6'>
        <div className='flex items-center justify-between mb-4'>
          <div>
            <h1 className='text-2xl font-bold text-gray-800'>Standard Changes Library</h1>
            <p className='text-gray-500 mt-1'>Manage pre-approved standard change templates</p>
          </div>
          <Button type='primary' icon={<Plus size={16} />} onClick={handleCreate}>

            New Template
          </Button>
        </div>

        <Card>
          <Row gutter={16} className='mb-4'>
            <Col span={8}>
              <Input
                placeholder="Search template name or description..."
                prefix={<Search size={14} />}
                value={search}
                onChange={e => setSearch(e.target.value)}
                onPressEnter={handleSearch}
              />
            </Col>
            <Col span={6}>
              <Select
                placeholder="Select category"
                style={{ width: '100%' }}
                allowClear
                value={category || undefined}
                onChange={val => {
                  setCategory(val || '');
                  setPage(1);
                }}
              >
                {categories.map(cat => (
                  <Select.Option key={cat} value={cat}>
                    {categoryLabel(cat)}
                  </Select.Option>
                ))}
              </Select>
            </Col>
            <Col span={4}>
              <Button type='primary' onClick={handleSearch}>

                Search
              </Button>
            </Col>
          </Row>

          <Table
            columns={columns}
            dataSource={templates}
            rowKey='id'
            loading={listLoading}
            pagination={{
              current: page,
              pageSize: pageSize,
              total: total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `Total ${total} items`,
              onChange: (p, ps) => {
                setPage(p);
                setPageSize(ps);
              },
            }}
            locale={{
              emptyText: (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={
                    templates.length === 0 ? (
                      <span>

                        There is no standard change template yet
                        <br />
                        <Button type='link' onClick={handleCreate}>

                          Create template
                        </Button>
                      </span>
                    ) : (
                      'No matching results yet'
                    )
                  }
                />
              ),
            }}
          />
        </Card>
      </div>

      {/* 创建/编辑模板 Modal */}
      <Modal
        title={editingTemplate ? 'Edit Standard Change Template' : 'Create a new standard change template'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        width={700}
        okText={editingTemplate ? 'Update' : 'Create'}
        cancelText="Cancel"
      >
        <Form form={form} layout='vertical' className='mt-4'>
          <Form.Item
            name='title'
            label="Template Name"
            rules={[{ required: true, message: 'Please enter template name' }]}
          >
            <Input placeholder="For example: server restart process" />
          </Form.Item>

          <Form.Item name='description' label="Template description">
            <Input.TextArea rows={2} placeholder="Briefly describe the purpose of this template..." />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name='category' label="Category" initialValue='general'>
                <Select options={Array.from(new Set([
                  ...Object.keys(CATEGORY_COLORS),
                  ...categories,
                  ...(editingTemplate?.category ? [editingTemplate.category] : []),
                ])).map(value => ({ value, label: categoryLabel(value) }))} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='expectedDuration' label="Estimated construction period (minutes)" initialValue={30}>
                <Input type='number' min={1} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name='riskLevel' label="Risk Level" initialValue='low'>
                <Select>
                  <Select.Option value='low'>Low</Select.Option>
                  <Select.Option value='medium'>Medium</Select.Option>
                  <Select.Option value='high'>High</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='impactScope' label="Impact Range" initialValue='low'>
                <Select>
                  <Select.Option value='low'>Low</Select.Option>
                  <Select.Option value='medium'>Medium</Select.Option>
                  <Select.Option value='high'>High</Select.Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name='implementationPlan'
            label="Implementation plan steps"
            rules={[{ required: true, message: 'Please enter the implementation plan steps' }]}
          >
            <Input.TextArea rows={4} placeholder="Detailed description of implementation steps..." />
          </Form.Item>

          <Form.Item
            name='rollbackPlan'
            label="Rollback Plan"
            rules={[{ required: true, message: 'Please enter rollback plan' }]}
          >
            <Input.TextArea rows={3} placeholder="Detailed description of rollback steps..." />
          </Form.Item>

          <Form.Item
            name='approvalRequired'
            label="Is approval required?"
            valuePropName='checked'
            initialValue={false}
          >
            <Input type='checkbox' />
          </Form.Item>

          <Form.Item name='prerequisites' label="Precondition">
            <Select mode='tags' placeholder="Enter the precondition and press Enter to add"></Select>
          </Form.Item>

          <Form.Item name='remarks' label="Notes">
            <Input.TextArea rows={2} placeholder="Other remarks..." />
          </Form.Item>
        </Form>
      </Modal>

      {/* 从模板创建变更 Modal */}
      <Modal
        title={`From template "${selectedTemplate?.title}" Create changes`}
        open={instantiateModalVisible}
        onCancel={() => setInstantiateModalVisible(false)}
        onOk={handleInstantiateSubmit}
        okText="Create Change"
        cancelText="Cancel"
      >
        {selectedTemplate && (
          <div className='mt-4'>
            <Descriptions column={2} size='small' className='mb-4'>
              <Descriptions.Item label="Category">
                <Tag>{categoryLabel(selectedTemplate.category)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Risk Level">
                <Tag color={RISK_COLORS[selectedTemplate.riskLevel]}>
                  {ChangeRiskLabels[selectedTemplate.riskLevel as keyof typeof ChangeRiskLabels]}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Estimated construction period">
                {selectedTemplate.expectedDuration}minutes
              </Descriptions.Item>
              <Descriptions.Item label="No approval required">
                {selectedTemplate.approvalRequired ? 'No' : 'Yes'}
              </Descriptions.Item>
            </Descriptions>

            <Form form={instantiateForm} layout='vertical'>
              <Form.Item name='title' label="Change title" initialValue={selectedTemplate.title}>
                <Input />
              </Form.Item>
              <Form.Item name='plannedStartDate' label="Planned start time">
                <DatePicker showTime style={{ width: '100%' }} placeholder="Select planned start time" />
              </Form.Item>
              <Form.Item name='plannedEndDate' label="Planned end time">
                <DatePicker showTime style={{ width: '100%' }} placeholder="Select planned end time" />
              </Form.Item>
            </Form>
          </div>
        )}
      </Modal>
    </div>
  );
}
