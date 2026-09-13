'use client';

import { useEffect, useState } from 'react';
import { App, Button, Card, Descriptions, Form, Input, Modal, Select, Space, Switch, Table, Tag, Typography } from 'antd';
import { Edit, Eye, History, Plus, Upload, Archive, Copy } from 'lucide-react';
import {
  BPMNWorkflowTemplateApi,
  type WorkflowTemplate,
  type CreateWorkflowTemplateRequest,
} from '@/lib/api/bpmn-workflow-template-api';

const { Text } = Typography;

const DOMAIN_OPTIONS = [
  { value: 'incident', label: '事件' },
  { value: 'service_request', label: '服务请求' },
  { value: 'change', label: '变更' },
  { value: 'problem', label: '问题' },
  { value: 'leave', label: '请假' },
  { value: 'expense', label: '费用' },
  { value: 'hr', label: '人事' },
  { value: 'procurement', label: '采购' },
  { value: 'it', label: 'IT' },
  { value: 'custom', label: '自定义' },
];

const STATUS_LABELS: Record<string, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'processing' },
  published: { label: '已发布', color: 'success' },
  archived: { label: '已停用', color: 'default' },
};

export default function WorkflowTemplateCatalog() {
  const { message } = App.useApp();
  const [items, setItems] = useState<WorkflowTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [filters, setFilters] = useState({ keyword: '', domain: '', status: '' });
  const [editing, setEditing] = useState<WorkflowTemplate | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [versions, setVersions] = useState<WorkflowTemplate[]>([]);
  const [versionsOpen, setVersionsOpen] = useState(false);
  const [detailItem, setDetailItem] = useState<WorkflowTemplate | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [form] = Form.useForm<CreateWorkflowTemplateRequest>();

  const load = async () => {
    setLoading(true);
    try {
      const result = await BPMNWorkflowTemplateApi.list({ ...filters, page: 1, pageSize: 100 });
      setItems(result.items);
    } catch {
      message.error('加载模板目录失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void load(); }, [filters]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ domain: 'custom', isPublic: false });
    setModalOpen(true);
  };

  const openEdit = (item: WorkflowTemplate) => {
    setEditing(item);
    form.setFieldsValue({
      key: item.key,
      name: item.name,
      description: item.description,
      domain: item.domain,
      bpmnXml: item.bpmnXml,
      isPublic: item.isPublic,
    });
    setModalOpen(true);
  };

  const save = async () => {
    const values = await form.validateFields();
    setLoading(true);
    try {
      if (editing) {
        await BPMNWorkflowTemplateApi.update(editing.key, {
          name: values.name,
          description: values.description,
          domain: values.domain,
          bpmnXml: values.bpmnXml,
          isPublic: values.isPublic,
        });
        message.success(editing.status === 'published' ? '新版本草稿已创建' : '模板草稿已保存');
      } else {
        await BPMNWorkflowTemplateApi.create(values);
        message.success('模板草稿已创建');
      }
      setModalOpen(false);
      await load();
    } catch {
      message.error('模板保存失败，请检查 key、BPMN XML 和权限');
    } finally {
      setLoading(false);
    }
  };

  const publish = (item: WorkflowTemplate) => {
    Modal.confirm({
      title: `发布 ${item.name} ${item.version}？`,
      content: '发布前会执行 BPMN Lint，发布后的版本不可直接编辑。',
      okText: '发布',
      cancelText: '取消',
      onOk: async () => {
        try {
          await BPMNWorkflowTemplateApi.publish(item.key);
          message.success('模板已发布');
          await load();
        } catch {
          message.error('模板未通过 BPMN 校验，发布失败');
        }
      },
    });
  };

  const archive = (item: WorkflowTemplate) => {
    Modal.confirm({
      title: `停用 ${item.name}？`,
      content: '停用只影响模板目录状态，不删除历史版本。',
      okText: '停用',
      cancelText: '取消',
      onOk: async () => {
        try {
          await BPMNWorkflowTemplateApi.archive(item.key);
          message.success('模板已停用');
          await load();
        } catch {
          message.error('模板停用失败，请检查权限或模板状态');
        }
      },
    });
  };

  const showVersions = async (item: WorkflowTemplate) => {
    try {
      setVersions(await BPMNWorkflowTemplateApi.versions(item.key));
      setVersionsOpen(true);
    } catch {
      message.error('加载模板版本失败');
    }
  };

  const showDetail = (item: WorkflowTemplate) => {
    setDetailItem(item);
    setDetailOpen(true);
  };

  const createNewVersion = (item: WorkflowTemplate) => {
    setEditing(item);
    form.setFieldsValue({
      key: item.key,
      name: item.name,
      description: item.description,
      domain: item.domain,
      bpmnXml: item.bpmnXml,
      isPublic: item.isPublic,
    });
    setModalOpen(true);
  };

  return (
    <Card
      title="AI 工作流模板目录"
      extra={<Button type="primary" icon={<Plus size={16} />} onClick={openCreate}>创建模板草稿</Button>}
    >
      <Space wrap className="mb-4">
        <Input.Search placeholder="搜索 key、名称或描述" allowClear value={filters.keyword} onChange={e => setFilters({ ...filters, keyword: e.target.value })} onSearch={value => setFilters({ ...filters, keyword: value })} />
        <Select allowClear placeholder="业务域" style={{ width: 150 }} options={DOMAIN_OPTIONS} value={filters.domain || undefined} onChange={domain => setFilters({ ...filters, domain: domain || '' })} />
        <Select allowClear placeholder="状态" style={{ width: 130 }} options={Object.entries(STATUS_LABELS).map(([value, data]) => ({ value, label: data.label }))} value={filters.status || undefined} onChange={status => setFilters({ ...filters, status: status || '' })} />
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={items}
        pagination={{ pageSize: 10, showSizeChanger: true }}
        columns={[
          { title: '模板', dataIndex: 'name', render: (name: string, item: WorkflowTemplate) => <div><Text strong>{name}</Text><div><Text type="secondary">{item.key}</Text></div></div> },
          { title: '业务域', dataIndex: 'domain', render: (value: string) => DOMAIN_OPTIONS.find(option => option.value === value)?.label || value },
          { title: '版本', dataIndex: 'version' },
          { title: '状态', dataIndex: 'status', render: (value: string) => <Tag color={STATUS_LABELS[value]?.color}>{STATUS_LABELS[value]?.label || value}</Tag> },
          { title: '更新时间', dataIndex: 'updatedAt' },
          { title: '操作', key: 'actions', render: (_: unknown, item: WorkflowTemplate) => <Space>
            <Button type="text" icon={<Eye size={16} />} aria-label="查看详情" onClick={() => showDetail(item)} />
            {item.status === 'draft' && <Button type="text" icon={<Edit size={16} />} aria-label="编辑草稿" onClick={() => openEdit(item)} />}
            {item.status === 'draft' && <Button type="text" icon={<Upload size={16} />} aria-label="发布模板" onClick={() => publish(item)} />}
            {item.status === 'published' && <Button type="text" icon={<Copy size={16} />} aria-label="创建新版本" onClick={() => createNewVersion(item)} />}
            {item.status === 'published' && <Button type="text" danger icon={<Archive size={16} />} aria-label="停用模板" onClick={() => archive(item)} />}
            <Button type="text" icon={<History size={16} />} aria-label="查看历史版本" onClick={() => void showVersions(item)} />
          </Space> },
        ]}
      />
      <Modal title={editing ? (editing.status === 'published' ? `基于 ${editing.name} ${editing.version} 创建新版本` : `编辑模板草稿 ${editing.version}`) : '创建模板草稿'} open={modalOpen} onOk={() => void save()} confirmLoading={loading} onCancel={() => setModalOpen(false)} width={820}>
        <Form form={form} layout="vertical">
          <Form.Item name="key" label="模板 Key" rules={[{ required: true, min: 2, max: 120 }]}><Input disabled={!!editing} placeholder="例如 expense_approval" /></Form.Item>
          <Form.Item name="name" label="模板名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="domain" label="业务域" rules={[{ required: true }]}><Select options={DOMAIN_OPTIONS} /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="bpmnXml" label="BPMN XML" rules={[{ required: true, message: '请输入 BPMN XML' }]}><Input.TextArea rows={12} placeholder="粘贴可执行的 BPMN 2.0 XML，发布时会执行 Lint" /></Form.Item>
          <Form.Item name="isPublic" label="对租户用户可见" valuePropName="checked"><Switch /></Form.Item>
        </Form>
      </Modal>
      <Modal title="模板版本历史" open={versionsOpen} onCancel={() => setVersionsOpen(false)} footer={null} width={900}>
        <Table rowKey="id" dataSource={versions} pagination={false} columns={[
          { title: '版本', dataIndex: 'version' },
          { title: '状态', dataIndex: 'status', render: (value: string) => <Tag color={STATUS_LABELS[value]?.color}>{STATUS_LABELS[value]?.label || value}</Tag> },
          { title: '更新时间', dataIndex: 'updatedAt' },
          { title: 'BPMN', dataIndex: 'bpmnXml', render: (value: string) => value ? '已配置' : '缺失' },
        ]} />
      </Modal>
      <Modal title={detailItem ? `模板详情 — ${detailItem.name}` : '模板详情'} open={detailOpen} onCancel={() => setDetailOpen(false)} footer={null} width={800}>
        {detailItem && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="模板 Key" span={2}>{detailItem.key}</Descriptions.Item>
            <Descriptions.Item label="模板名称">{detailItem.name}</Descriptions.Item>
            <Descriptions.Item label="业务域">{DOMAIN_OPTIONS.find(option => option.value === detailItem.domain)?.label || detailItem.domain}</Descriptions.Item>
            <Descriptions.Item label="版本">{detailItem.version}</Descriptions.Item>
            <Descriptions.Item label="状态"><Tag color={STATUS_LABELS[detailItem.status]?.color}>{STATUS_LABELS[detailItem.status]?.label || detailItem.status}</Tag></Descriptions.Item>
            <Descriptions.Item label="对租户用户可见">{detailItem.isPublic ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{detailItem.createdAt}</Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>{detailItem.description || '—'}</Descriptions.Item>
            <Descriptions.Item label="BPMN XML" span={2}>
              {detailItem.bpmnXml ? (
                <Typography.Paragraph ellipsis={{ rows: 6, expandable: true, symbol: '展开' }} style={{ marginBottom: 0, fontSize: 12, fontFamily: 'monospace', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                  {detailItem.bpmnXml}
                </Typography.Paragraph>
              ) : '未配置'}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </Card>
  );
}
