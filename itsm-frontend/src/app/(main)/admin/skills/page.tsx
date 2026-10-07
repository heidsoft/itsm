'use client';

import React, { useEffect, useState } from 'react';
import {
  Card, Table, Tag, Button, Space, Modal, Form, Input, Select, App, Drawer, Typography, Empty, Alert,
} from 'antd';
import { Wand2, Eye, Edit, ArrowUp, PowerOff } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import skillApi, {
  type SkillEntry, type SkillCategory, type SkillStatus, type SkillUpsertRequest,
} from '@/lib/api/skill-api';
import { useI18n } from '@/lib/i18n/useI18n';

const { Text, Paragraph } = Typography;

const CATEGORY_COLOR: Record<SkillCategory, string> = {
  ga: 'green',
  pilot: 'gold',
  experimental: 'default',
};

const STATUS_COLOR: Record<SkillStatus, string> = {
  active: 'green',
  disabled: 'red',
};

export default function SkillsAdminPage() {
  const { t } = useI18n();
  const { message } = App.useApp();
  const [items, setItems] = useState<SkillEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [filters, setFilters] = useState<{ q?: string; category?: SkillCategory; status?: SkillStatus }>({});

  const [detail, setDetail] = useState<SkillEntry | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editorTarget, setEditorTarget] = useState<SkillEntry | null>(null);
  const [form] = Form.useForm<SkillUpsertRequest>();

  const load = async () => {
    setLoading(true);
    try {
      const res = await skillApi.list({ ...filters, pageSize: 100 });
      setItems(res.items ?? []);
    } catch (e) {
      message.error((e as Error).message ?? t('admin.skills.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [filters]);

  const openCreate = () => {
    setEditorTarget(null);
    form.resetFields();
    form.setFieldsValue({
      category: 'experimental',
      tags: [],
      capabilities: [],
      requiredPermissions: [],
      version: '0.1.0',
    });
    setEditorOpen(true);
  };

  const openEdit = (entry: SkillEntry) => {
    setEditorTarget(entry);
    form.resetFields();
    form.setFieldsValue({
      code: entry.code,
      version: entry.version,
      title: entry.title,
      description: entry.description,
      longDescription: entry.longDescription,
      category: entry.category,
      tags: entry.tags,
      capabilities: entry.capabilities,
      requiredPermissions: entry.requiredPermissions,
      provider: entry.provider,
      author: entry.author,
      inputSchema: entry.manifest?.inputSchema as Record<string, never> | undefined,
      outputSchema: entry.manifest?.outputSchema as Record<string, never> | undefined,
    });
    setEditorOpen(true);
  };

  const submitEditor = async () => {
    try {
      const values = await form.validateFields();
      if (editorTarget) {
        await skillApi.update(editorTarget.code, values);
        message.success(t('admin.skills.updateSuccess'));
      } else {
        await skillApi.create(values);
        message.success(t('admin.skills.createSuccess'));
      }
      setEditorOpen(false);
      void load();
    } catch (e) {
      message.error((e as Error).message ?? t('admin.skills.saveFailed'));
    }
  };

  const promote = async (entry: SkillEntry) => {
    try {
      await skillApi.promote(entry.code);
      message.success(t('admin.skills.promoted', { code: entry.code }));
      void load();
    } catch (e) {
      message.error((e as Error).message ?? t('admin.skills.promoteFailed'));
    }
  };

  const disable = async (entry: SkillEntry) => {
    try {
      await skillApi.disable(entry.code);
      message.success(t('admin.skills.disabled', { code: entry.code }));
      void load();
    } catch (e) {
      message.error((e as Error).message ?? t('admin.skills.disableFailed'));
    }
  };

  return (
    <PageContainer
      header={{ title: t('admin.skills.title') }}
    >
      <Card>
        <Space wrap style={{ marginBottom: 16 }}>
          <Input.Search
            placeholder={t('admin.skills.search')}
            allowClear
            onSearch={(q) => setFilters((f) => ({ ...f, q }))}
            style={{ width: 240 }}
          />
          <Select
            placeholder={t('admin.skills.category')}
            allowClear
            style={{ width: 160 }}
            value={filters.category}
            onChange={(category) => setFilters((f) => ({ ...f, category }))}
            options={[
              { label: t('admin.skills.categoryGa'), value: 'ga' },
              { label: t('admin.skills.categoryPilot'), value: 'pilot' },
              { label: t('admin.skills.categoryExperimental'), value: 'experimental' },
            ]}
          />
          <Select
            placeholder={t('admin.skills.status')}
            allowClear
            style={{ width: 140 }}
            value={filters.status}
            onChange={(status) => setFilters((f) => ({ ...f, status }))}
            options={[
              { label: t('admin.skills.statusActive'), value: 'active' },
              { label: t('admin.skills.statusDisabled'), value: 'disabled' },
            ]}
          />
          <Button type="primary" icon={<Wand2 size={14} />} onClick={openCreate}>
            {t('admin.skills.register')}
          </Button>
        </Space>

        <Table<SkillEntry>
          rowKey="code"
          loading={loading}
          dataSource={items}
          locale={{ emptyText: <Empty description={t('admin.skills.empty')} /> }}
          columns={[
            {
              title: t('admin.skills.columnCode'),
              dataIndex: 'code',
              key: 'code',
              render: (value: string, row) => (
                <a onClick={() => setDetail(row)}>{value}</a>
              ),
            },
            {
              title: t('admin.skills.columnTitle'),
              dataIndex: 'title',
              key: 'title',
            },
            {
              title: t('admin.skills.columnVersion'),
              dataIndex: 'version',
              key: 'version',
              width: 110,
            },
            {
              title: t('admin.skills.columnCategory'),
              dataIndex: 'category',
              key: 'category',
              width: 120,
              render: (value: SkillCategory) => (
                <Tag color={CATEGORY_COLOR[value]}>{value.toUpperCase()}</Tag>
              ),
            },
            {
              title: t('admin.skills.columnStatus'),
              dataIndex: 'status',
              key: 'status',
              width: 110,
              render: (value: SkillStatus) => (
                <Tag color={STATUS_COLOR[value]}>{value}</Tag>
              ),
            },
            {
              title: t('admin.skills.columnOfficial'),
              dataIndex: 'isOfficial',
              key: 'isOfficial',
              width: 100,
              render: (value: boolean) => (value ? '✓' : '—'),
            },
            {
              title: t('admin.skills.columnCalls'),
              key: 'calls',
              width: 110,
              render: (_, row) => row.metrics?.totalCalls ?? 0,
            },
            {
              title: t('admin.skills.columnActions'),
              key: 'actions',
              width: 220,
              render: (_, row) => (
                <Space size="small">
                  <Button
                    size="small"
                    icon={<Eye size={14} />}
                    onClick={() => setDetail(row)}
                  >
                    {t('admin.skills.actionDetail')}
                  </Button>
                  <Button
                    size="small"
                    icon={<Edit size={14} />}
                    onClick={() => openEdit(row)}
                  >
                    {t('admin.skills.actionEdit')}
                  </Button>
                  {row.category !== 'ga' && (
                    <Button
                      size="small"
                      icon={<ArrowUp size={14} />}
                      onClick={() => promote(row)}
                    >
                      {t('admin.skills.actionPromote')}
                    </Button>
                  )}
                  {row.status === 'active' && (
                    <Button
                      size="small"
                      danger
                      icon={<PowerOff size={14} />}
                      onClick={() => disable(row)}
                    >
                      {t('admin.skills.actionDisable')}
                    </Button>
                  )}
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Drawer
        title={detail ? `${detail.code} · ${detail.title}` : ''}
        open={!!detail}
        onClose={() => setDetail(null)}
        width={640}
      >
        {detail && (
          <>
            <Paragraph>
              <Text strong>{t('admin.skills.detailVersion')}</Text>{' '}
              <Tag>{detail.version}</Tag>
              <Tag color={CATEGORY_COLOR[detail.category]}>{detail.category.toUpperCase()}</Tag>
              <Tag color={STATUS_COLOR[detail.status]}>{detail.status}</Tag>
              {detail.isOfficial && <Tag color="blue">Official</Tag>}
            </Paragraph>
            <Paragraph>
              <Text strong>{t('admin.skills.detailProvider')}</Text> {detail.provider}
              {detail.author && <> · {detail.author}</>}
            </Paragraph>
            {detail.description && (
              <Paragraph>
                <Text strong>{t('admin.skills.detailDescription')}</Text>
                <br />
                {detail.description}
              </Paragraph>
            )}
            <Paragraph>
              <Text strong>{t('admin.skills.detailRequiredPermissions')}</Text>
              <br />
              {detail.requiredPermissions.length === 0 ? (
                <Text type="secondary">—</Text>
              ) : (
                detail.requiredPermissions.map((p) => <Tag key={p}>{p}</Tag>)
              )}
            </Paragraph>
            <Paragraph>
              <Text strong>{t('admin.skills.detailCapabilities')}</Text>
              <br />
              {detail.capabilities.length === 0 ? (
                <Text type="secondary">—</Text>
              ) : (
                detail.capabilities.map((c) => <Tag key={c}>{c}</Tag>)
              )}
            </Paragraph>
            <Paragraph>
              <Text strong>{t('admin.skills.detailManifest')}</Text>
              <pre
                style={{
                  background: '#f5f5f5',
                  padding: 12,
                  borderRadius: 4,
                  maxHeight: 320,
                  overflow: 'auto',
                  fontSize: 12,
                }}
              >
                {JSON.stringify(detail.manifest ?? {}, null, 2)}
              </pre>
            </Paragraph>
            <Paragraph>
              <Text strong>{t('admin.skills.detailMetrics')}</Text>
              <br />
              {t('admin.skills.metricsCalls')}: {detail.metrics?.totalCalls ?? 0}
              {' · '}
              {t('admin.skills.metricsSuccess')}:{' '}
              {((detail.metrics?.successRate ?? 0) * 100).toFixed(1)}%
              {' · '}
              {t('admin.skills.metricsLatency')}: {detail.metrics?.avgLatencyMs ?? 0} ms
            </Paragraph>
          </>
        )}
      </Drawer>

      <Modal
        open={editorOpen}
        title={editorTarget ? t('admin.skills.editorUpdate') : t('admin.skills.editorCreate')}
        onCancel={() => setEditorOpen(false)}
        onOk={submitEditor}
        okText={t('admin.skills.editorSubmit')}
        cancelText={t('admin.skills.editorCancel')}
        width={720}
        destroyOnHidden
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title={t('admin.skills.editorHint')}
        />
        <Form<SkillUpsertRequest> form={form} layout="vertical">
          <Form.Item name="code" label={t('admin.skills.columnCode')} rules={[{ required: true }]}>
            <Input disabled={!!editorTarget} placeholder="triage_classifier" />
          </Form.Item>
          <Form.Item name="version" label={t('admin.skills.columnVersion')} rules={[{ required: true }]}>
            <Input placeholder="1.0.0" />
          </Form.Item>
          <Form.Item name="title" label={t('admin.skills.columnTitle')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="provider" label={t('admin.skills.detailProvider')} rules={[{ required: true }]}>
            <Input placeholder="itsm" />
          </Form.Item>
          <Form.Item name="category" label={t('admin.skills.columnCategory')} rules={[{ required: true }]}>
            <Select
              options={[
                { label: t('admin.skills.categoryGa'), value: 'ga' },
                { label: t('admin.skills.categoryPilot'), value: 'pilot' },
                { label: t('admin.skills.categoryExperimental'), value: 'experimental' },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label={t('admin.skills.detailDescription')}>
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="tags" label={t('admin.skills.editorTags')}>
            <Select mode="tags" tokenSeparators={[',']} placeholder="ai, triage" />
          </Form.Item>
          <Form.Item name="capabilities" label={t('admin.skills.detailCapabilities')}>
            <Select mode="tags" tokenSeparators={[',']} placeholder="classify, route" />
          </Form.Item>
          <Form.Item
            name="requiredPermissions"
            label={t('admin.skills.detailRequiredPermissions')}
            rules={[{ required: true, message: t('admin.skills.requiredPermissionsRequired') }]}
          >
            <Select mode="tags" tokenSeparators={[',']} placeholder="ai:read, ticket:read" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
