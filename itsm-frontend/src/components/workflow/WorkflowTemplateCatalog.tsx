'use client';

import { useEffect, useState } from 'react';
import { App, Button, Card, Collapse, Descriptions, Form, Input, Modal, Select, Space, Switch, Table, Tag, Typography } from 'antd';
import { Edit, Eye, History, Plus, Upload, Archive, Copy } from 'lucide-react';
import BPMNDiagramPreview from '@/components/workflow/BPMNDiagramPreview';
import {
  BPMNWorkflowTemplateApi,
  type WorkflowTemplate,
  type CreateWorkflowTemplateRequest,
} from '@/lib/api/bpmn-workflow-template-api';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';
import { formatDateTime } from '@/lib/formatters';

const { Text } = Typography;

const DOMAIN_OPTIONS = [
  { value: 'incident', label: 'Incident' },
  { value: 'service_request', label: 'Service Request' },
  { value: 'change', label: 'Change' },
  { value: 'problem', label: 'Problem' },
  { value: 'leave', label: 'Asking for leave' },
  { value: 'expense', label: 'Fee' },
  { value: 'hr', label: 'Personnel' },
  { value: 'procurement', label: 'Procurement' },
  { value: 'it', label: 'IT' },
  { value: 'custom', label: 'Custom' },
];

const STATUS_LABELS: Record<string, { label: string; color: string }> = {
  draft: { label: 'Draft', color: 'processing' },
  published: { label: 'Published', color: 'success' },
  archived: { label: 'Disabled', color: 'default' },
};

export default function WorkflowTemplateCatalog() {
  const { message, modal } = App.useApp();
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
      const result = await BPMNWorkflowTemplateApi.list({ ...filters, page: 1, pageSize: BACKEND_MAX_PAGE_SIZE });
      setItems(result.items);
    } catch {
      message.error('Failed to load template directory');
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
        message.success(editing.status === 'published' ? 'A new version draft has been created' : 'Template draft saved');
      } else {
        await BPMNWorkflowTemplateApi.create(values);
        message.success('Template draft created');
      }
      setModalOpen(false);
      await load();
    } catch {
      message.error('Template save failed, please check key, BPMN XML and permissions');
    } finally {
      setLoading(false);
    }
  };

  const publish = (item: WorkflowTemplate) => {
    modal.confirm({
      title: `Release ${item.name} ${item.version}？`,
      content: 'BPMN Lint will be executed before publishing, and the version after publishing cannot be edited directly.',
      okText: 'Release',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await BPMNWorkflowTemplateApi.publish(item.key);
          message.success('Template has been published');
          await load();
        } catch {
          message.error('The template failed the BPMN verification and the publishing failed.');
        }
      },
    });
  };

  const archive = (item: WorkflowTemplate) => {
    modal.confirm({
      title: `Disable ${item.name}？`,
      content: 'Deactivation only affects the template directory status and does not delete historical versions.',
      okText: 'Disable',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await BPMNWorkflowTemplateApi.archive(item.key);
          message.success('Template deactivated');
          await load();
        } catch {
          message.error('Template deactivation failed, please check permissions or template status');
        }
      },
    });
  };

  const showVersions = async (item: WorkflowTemplate) => {
    try {
      setVersions(await BPMNWorkflowTemplateApi.versions(item.key));
      setVersionsOpen(true);
    } catch {
      message.error('Failed to load template version');
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
      title="AI workflow template directory"
      extra={<Button type="primary" icon={<Plus size={16} />} onClick={openCreate}>Create template draft</Button>}
    >
      <Space wrap className="mb-4">
        <Input.Search placeholder="Search key, name or description" allowClear value={filters.keyword} onChange={e => setFilters({ ...filters, keyword: e.target.value })} onSearch={value => setFilters({ ...filters, keyword: value })} />
        <Select allowClear placeholder="Business domain" style={{ width: 150 }} options={DOMAIN_OPTIONS} value={filters.domain || undefined} onChange={domain => setFilters({ ...filters, domain: domain || '' })} />
        <Select allowClear placeholder="Status" style={{ width: 130 }} options={Object.entries(STATUS_LABELS).map(([value, data]) => ({ value, label: data.label }))} value={filters.status || undefined} onChange={status => setFilters({ ...filters, status: status || '' })} />
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={items}
        pagination={{ pageSize: 10, showSizeChanger: true }}
        columns={[
          { title: 'Templates', dataIndex: 'name', render: (name: string, item: WorkflowTemplate) => <div><Text strong>{name}</Text><div><Text type="secondary">{item.key}</Text></div></div> },
          { title: 'Business domain', dataIndex: 'domain', render: (value: string) => DOMAIN_OPTIONS.find(option => option.value === value)?.label || value },
          { title: 'Version', dataIndex: 'version' },
          { title: 'Status', dataIndex: 'status', render: (value: string) => <Tag color={STATUS_LABELS[value]?.color}>{STATUS_LABELS[value]?.label || value}</Tag> },
          { title: 'Updated At', dataIndex: 'updatedAt', render: (value: string) => formatDateTime(value) },
          { title: 'Actions', key: 'actions', render: (_: unknown, item: WorkflowTemplate) => <Space>
            <Button type="text" icon={<Eye size={16} />} aria-label="View Details" onClick={() => showDetail(item)} />
            {item.status === 'draft' && <Button type="text" icon={<Edit size={16} />} aria-label="Edit draft" onClick={() => openEdit(item)} />}
            {item.status === 'draft' && <Button type="text" icon={<Upload size={16} />} aria-label="Publish template" onClick={() => publish(item)} />}
            {item.status === 'published' && <Button type="text" icon={<Copy size={16} />} aria-label="Create new version" onClick={() => createNewVersion(item)} />}
            {item.status === 'published' && <Button type="text" danger icon={<Archive size={16} />} aria-label="Deactivate template" onClick={() => archive(item)} />}
            <Button type="text" icon={<History size={16} />} aria-label="View historical versions" onClick={() => void showVersions(item)} />
          </Space> },
        ]}
      />
      <Modal title={editing ? (editing.status === 'published' ? `Based on ${editing.name} ${editing.version} Create new version` : `Edit template draft ${editing.version}`) : 'Create template draft'} open={modalOpen} onOk={() => void save()} confirmLoading={loading} onCancel={() => setModalOpen(false)} width={820}>
        <Form form={form} layout="vertical">
          <Form.Item name="key" label="Template Key" rules={[{ required: true, min: 2, max: 120 }]}><Input disabled={!!editing} placeholder="e.g. expense_approval" /></Form.Item>
          <Form.Item name="name" label="Template Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="domain" label="Business domain" rules={[{ required: true }]}><Select options={DOMAIN_OPTIONS} /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="bpmnXml" label="BPMN XML" rules={[{ required: true, message: 'Please enter BPMN XML' }]}><Input.TextArea rows={12} placeholder="Pasting executable BPMN 2.0 XML will execute Lint when publishing" /></Form.Item>
          <Form.Item name="isPublic" label="Visible to tenant users" valuePropName="checked"><Switch /></Form.Item>
        </Form>
      </Modal>
      <Modal title="Template version history" open={versionsOpen} onCancel={() => setVersionsOpen(false)} footer={null} width={900}>
        <Table rowKey="id" dataSource={versions} pagination={false} columns={[
          { title: 'Version', dataIndex: 'version' },
          { title: 'Status', dataIndex: 'status', render: (value: string) => <Tag color={STATUS_LABELS[value]?.color}>{STATUS_LABELS[value]?.label || value}</Tag> },
          { title: 'Updated At', dataIndex: 'updatedAt', render: (value: string) => formatDateTime(value) },
          { title: 'BPMN', dataIndex: 'bpmnXml', render: (value: string) => value ? 'Configured' : 'missing' },
        ]} />
      </Modal>
      <Modal title={detailItem ? `Template details — ${detailItem.name}` : 'Template details'} open={detailOpen} onCancel={() => setDetailOpen(false)} footer={null} width={860} destroyOnHidden>
        {detailItem && (
          <div className="space-y-4">
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="Template Key" span={2}>{detailItem.key}</Descriptions.Item>
              <Descriptions.Item label="Template Name">{detailItem.name}</Descriptions.Item>
              <Descriptions.Item label="Business domain">{DOMAIN_OPTIONS.find(option => option.value === detailItem.domain)?.label || detailItem.domain}</Descriptions.Item>
              <Descriptions.Item label="Version">{detailItem.version}</Descriptions.Item>
              <Descriptions.Item label="Status"><Tag color={STATUS_LABELS[detailItem.status]?.color}>{STATUS_LABELS[detailItem.status]?.label || detailItem.status}</Tag></Descriptions.Item>
              <Descriptions.Item label="Visible to tenant users">{detailItem.isPublic ? 'Yes' : 'No'}</Descriptions.Item>
              <Descriptions.Item label="Created">{formatDateTime(detailItem.createdAt)}</Descriptions.Item>
              <Descriptions.Item label="Description" span={2}>{detailItem.description || '—'}</Descriptions.Item>
            </Descriptions>

            <div>
              <div className="mb-2 text-sm font-medium">Process preview</div>
              <BPMNDiagramPreview xml={detailItem.bpmnXml || ''} height={340} />
            </div>

            {detailItem.bpmnXml && (
              <Collapse
                size="small"
                items={[
                  {
                    key: 'xml',
                    label: 'BPMN XML',
                    children: (
                      <pre className="max-h-[280px] overflow-auto text-xs leading-relaxed">
                        {detailItem.bpmnXml}
                      </pre>
                    ),
                  },
                ]}
              />
            )}
          </div>
        )}
      </Modal>
    </Card>
  );
}
