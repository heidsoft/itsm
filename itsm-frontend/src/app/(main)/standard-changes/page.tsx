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
import { StandardChangeApi, StandardChange } from '@/lib/api/standard-change-api';
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

export default function StandardChangesPage() {
  const router = useRouter();
  const { t } = useI18n();
  const [loading, setLoading] = useState(false);
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
    setLoading(true);
    try {
      const data = await StandardChangeApi.getTemplates({
        page,
        page_size: pageSize,
        category: category || undefined,
        search: search || undefined,
        active_only: true,
      });
      setTemplates(data.templates);
      setTotal(data.total);
    } catch (error) {
      console.error('Failed to fetch templates:', error);
      message.error('获取模板列表失败');
    } finally {
      setLoading(false);
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
      implementation_plan: template.implementationPlan,
      rollback_plan: template.rollbackPlan,
      justification: template.justification,
      category: template.category,
      risk_level: template.riskLevel,
      impact_scope: template.impactScope,
      expected_duration: template.expectedDuration,
      approval_required: template.approvalRequired,
      affected_cis: template.affectedCIs,
      prerequisites: template.prerequisites,
      remarks: template.remarks,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await StandardChangeApi.deleteTemplate(id);
      message.success('删除成功');
      fetchTemplates();
      fetchCategories();
    } catch (error) {
      console.error('Failed to delete template:', error);
      message.error('删除失败');
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
      message.success('已成功从模板创建变更');
      setInstantiateModalVisible(false);
      router.push(`/changes/${result.change_id}`);
    } catch (error) {
      console.error('Failed to instantiate:', error);
      message.error('从模板创建变更失败');
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingTemplate) {
        await StandardChangeApi.updateTemplate(editingTemplate.id, values);
        message.success('更新成功');
      } else {
        await StandardChangeApi.createTemplate(values);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchTemplates();
      fetchCategories();
    } catch (error) {
      console.error('Failed to submit:', error);
      message.error(editingTemplate ? '更新失败' : '创建失败');
    }
  };

  const columns = [
    {
      title: '模板名称',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: StandardChange) => (
        <div>
          <Text strong>{text}</Text>
          {record.description && (
            <Text type="secondary" style={{ display: 'block', fontSize: 12 }}>
              {record.description.slice(0, 50)}...
            </Text>
          )}
        </div>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (cat: string) => (
        <Tag color={CATEGORY_COLORS[cat] || 'default'}>{cat || 'general'}</Tag>
      ),
    },
    {
      title: '风险等级',
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
      title: '影响范围',
      dataIndex: 'impactScope',
      key: 'impactScope',
      width: 100,
      render: (scope: string) => (
        <Tag>{ChangeImpactLabels[scope as keyof typeof ChangeImpactLabels] || scope}</Tag>
      ),
    },
    {
      title: '预计工期',
      dataIndex: 'expectedDuration',
      key: 'expectedDuration',
      width: 100,
      render: (mins: number) => (mins ? `${mins}分钟` : '-'),
    },
    {
      title: '免审批',
      dataIndex: 'approvalRequired',
      key: 'approvalRequired',
      width: 100,
      render: (required: boolean) =>
        required ? (
          <Tag icon={<AlertCircle size={12} />} color="orange">
            需审批
          </Tag>
        ) : (
          <Tag icon={<CheckCircle size={12} />} color="green">
            免审批
          </Tag>
        ),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: unknown, record: StandardChange) => (
        <Space>
          <Tooltip title="从模板创建变更">
            <Button
              type="link"
              icon={<PlayCircle size={14} />}
              onClick={() => handleInstantiate(record)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button type="link" icon={<Edit size={14} />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Popconfirm
            title="确定要删除此模板吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button type="link" danger icon={<Delete size={14} />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-800">标准变更库</h1>
            <p className="text-gray-500 mt-1">管理预批准的标准变更模板</p>
          </div>
          <Button type="primary" icon={<Plus size={16} />} onClick={handleCreate}>
            新建模板
          </Button>
        </div>

        <Card>
          <Row gutter={16} className="mb-4">
            <Col span={8}>
              <Input
                placeholder="搜索模板名称或描述..."
                prefix={<Search size={14} />}
                value={search}
                onChange={e => setSearch(e.target.value)}
                onPressEnter={handleSearch}
              />
            </Col>
            <Col span={6}>
              <Select
                placeholder="选择分类"
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
                    {cat}
                  </Select.Option>
                ))}
              </Select>
            </Col>
            <Col span={4}>
              <Button type="primary" onClick={handleSearch}>
                搜索
              </Button>
            </Col>
          </Row>

          <Table
            columns={columns}
            dataSource={templates}
            rowKey="id"
            loading={loading}
            pagination={{
              current: page,
              pageSize: pageSize,
              total: total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `共 ${total} 条`,
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
                        暂无标准变更模板
                        <br />
                        <Button type="link" onClick={handleCreate}>
                          点击创建第一个模板
                        </Button>
                      </span>
                    ) : (
                      '暂无匹配结果'
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
        title={editingTemplate ? '编辑标准变更模板' : '新建标准变更模板'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        width={700}
        okText={editingTemplate ? '更新' : '创建'}
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            name="title"
            label="模板名称"
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder="例如：服务器重启流程" />
          </Form.Item>

          <Form.Item name="description" label="模板描述">
            <Input.TextArea rows={2} placeholder="简要描述此模板的用途..." />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="category" label="分类" initialValue="general">
                <Select>
                  <Select.Option value="server">服务器</Select.Option>
                  <Select.Option value="network">网络</Select.Option>
                  <Select.Option value="database">数据库</Select.Option>
                  <Select.Option value="application">应用</Select.Option>
                  <Select.Option value="security">安全</Select.Option>
                  <Select.Option value="general">通用</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="expected_duration" label="预计工期（分钟）" initialValue={30}>
                <Input type="number" min={1} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="risk_level" label="风险等级" initialValue="low">
                <Select>
                  <Select.Option value="low">低</Select.Option>
                  <Select.Option value="medium">中</Select.Option>
                  <Select.Option value="high">高</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="impact_scope" label="影响范围" initialValue="low">
                <Select>
                  <Select.Option value="low">低</Select.Option>
                  <Select.Option value="medium">中</Select.Option>
                  <Select.Option value="high">高</Select.Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="implementation_plan"
            label="实施计划步骤"
            rules={[{ required: true, message: '请输入实施计划步骤' }]}
          >
            <Input.TextArea rows={4} placeholder="详细描述实施步骤..." />
          </Form.Item>

          <Form.Item
            name="rollback_plan"
            label="回滚计划"
            rules={[{ required: true, message: '请输入回滚计划' }]}
          >
            <Input.TextArea rows={3} placeholder="详细描述回滚步骤..." />
          </Form.Item>

          <Form.Item name="approval_required" label="是否需要审批" valuePropName="checked" initialValue={false}>
            <Input type="checkbox" />
          </Form.Item>

          <Form.Item name="prerequisites" label="前置条件">
            <Select mode="tags" placeholder="输入前置条件，按回车添加">
            </Select>
          </Form.Item>

          <Form.Item name="remarks" label="备注">
            <Input.TextArea rows={2} placeholder="其他备注信息..." />
          </Form.Item>
        </Form>
      </Modal>

      {/* 从模板创建变更 Modal */}
      <Modal
        title={`从模板 "${selectedTemplate?.title}" 创建变更`}
        open={instantiateModalVisible}
        onCancel={() => setInstantiateModalVisible(false)}
        onOk={handleInstantiateSubmit}
        okText="创建变更"
        cancelText="取消"
      >
        {selectedTemplate && (
          <div className="mt-4">
            <Descriptions column={2} size="small" className="mb-4">
              <Descriptions.Item label="分类">
                <Tag>{selectedTemplate.category}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="风险等级">
                <Tag color={RISK_COLORS[selectedTemplate.riskLevel]}>
                  {ChangeRiskLabels[selectedTemplate.riskLevel as keyof typeof ChangeRiskLabels]}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="预计工期">
                {selectedTemplate.expectedDuration}分钟
              </Descriptions.Item>
              <Descriptions.Item label="免审批">
                {selectedTemplate.approvalRequired ? '否' : '是'}
              </Descriptions.Item>
            </Descriptions>

            <Form form={instantiateForm} layout="vertical">
              <Form.Item name="title" label="变更标题" initialValue={selectedTemplate.title}>
                <Input />
              </Form.Item>
              <Form.Item name="planned_start_date" label="计划开始时间">
                <Input type="datetime-local" />
              </Form.Item>
              <Form.Item name="planned_end_date" label="计划结束时间">
                <Input type="datetime-local" />
              </Form.Item>
            </Form>
          </div>
        )}
      </Modal>
    </div>
  );
}
