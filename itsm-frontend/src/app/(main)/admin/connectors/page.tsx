'use client';

import React, { useEffect, useState } from 'react';
import {
  Card, Table, Tag, Button, Space, Modal, Form, Input, InputNumber, Switch, Tabs, App, Drawer,
  Typography, Empty, Alert, Tooltip,
} from 'antd';
import { Plus, RotateCcw, CheckCircle, XCircle, Plug, Send, Power } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import type {
  ConnectorManifest, ConnectorConfig, SendConnectorMessageRequest,
} from '@/lib/services/connector-service';
import connectorService from '@/lib/services/connector-service';
import { useI18n } from '@/lib/i18n/useI18n';

const { Text, Paragraph } = Typography;

type Tab = 'market' | 'instances';

export default function ConnectorsAdminPage() {
  const { t } = useI18n();
  const { message } = App.useApp();
  const [tab, setTab] = useState<Tab>('market');
  const [market, setMarket] = useState<ConnectorManifest[]>([]);
  const [instances, setInstances] = useState<ConnectorConfig[]>([]);
  const [health, setHealth] = useState<Record<string, { ok: boolean; message?: string }>>({});
  const [loading, setLoading] = useState(false);

  const [provisionOpen, setProvisionOpen] = useState(false);
  const [provisionTarget, setProvisionTarget] = useState<ConnectorManifest | null>(null);
  const [sendOpen, setSendOpen] = useState(false);
  const [sendTarget, setSendTarget] = useState<ConnectorConfig | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailTarget, setDetailTarget] = useState<ConnectorManifest | null>(null);

  const [form] = Form.useForm();
  const [sendForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [m, c, h] = await Promise.all([
        connectorService.list(),
        connectorService.configs(),
        connectorService.health().catch(() => ({})),
      ]);
      setMarket(m);
      setInstances(c);
      setHealth(h || {});
    } catch (e) {
      message.error(t('connectors.loadFailed', { msg: (e as Error).message }));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openProvision = (m: ConnectorManifest) => {
    setProvisionTarget(m);
    form.resetFields();
    form.setFieldsValue({ enabled: true, provider: m.provider, imapHost: 'imap.qq.com', imapPort: 993, smtpHost: 'smtp.qq.com', smtpPort: 465, mailbox: 'INBOX', pollIntervalSeconds: 30 });
    setProvisionOpen(true);
  };

  const submitProvision = async () => {
    if (!provisionTarget) return;
    try {
      const values = await form.validateFields();
      const credentials: Record<string, string> = {};
      const settings: Record<string, unknown> = {};
      if (provisionTarget.name === 'email') {
        credentials.username = values.emailUsername;
        credentials.password = values.emailPassword;
        Object.assign(settings, {
          imapHost: values.imapHost,
          imapPort: values.imapPort,
          smtpHost: values.smtpHost,
          smtpPort: values.smtpPort,
          mailbox: values.mailbox,
          pollIntervalSeconds: values.pollIntervalSeconds,
          debug_channel: values.emailUsername,
        });
      }
      if (typeof values.credText === 'string') {
        values.credText.split('\n').forEach((line: string) => {
          const [k, ...rest] = line.split('=');
          if (k && rest.length) credentials[k.trim()] = rest.join('=').trim();
        });
      }
      if (typeof values.settingText === 'string') {
        values.settingText.split('\n').forEach((line: string) => {
          const [k, ...rest] = line.split('=');
          if (k && rest.length) settings[k.trim()] = rest.join('=').trim();
        });
      }
      await connectorService.provision({
        name: provisionTarget.name,
        provider: values.provider || provisionTarget.provider,
        enabled: values.enabled,
        credentials,
        settings,
      });
      message.success(t('connectors.provision.enableSuccess', { title: provisionTarget.title }));
      setProvisionOpen(false);
      load();
    } catch (e) {
      if ((e as { errorFields?: unknown }).errorFields) return;
      message.error(t('connectors.provision.enableFailed', { msg: (e as Error).message }));
    }
  };

  const handleRevoke = async (c: ConnectorConfig) => {
    Modal.confirm({
      title: t('connectors.revoke.title', { name: c.name }),
      content: t('connectors.revoke.content'),
      okType: 'danger',
      okText: t('connectors.buttons.disable'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        await connectorService.revoke(c.name);
        message.success(t('connectors.revoke.success', { name: c.name }));
        load();
      },
    });
  };

  const handleTest = async (c: ConnectorConfig) => {
    try {
      const r = await connectorService.test(c.name);
      message.success(t('connectors.test.success', { channel: r.channel }));
    } catch (e) {
      message.error(t('connectors.test.failed', { msg: (e as Error).message }));
    }
  };

  const openSend = (c: ConnectorConfig) => {
    setSendTarget(c);
    sendForm.resetFields();
    sendForm.setFieldsValue({ type: 'text', content: t('connectors.send.defaultContent') });
    setSendOpen(true);
  };

  const submitSend = async () => {
    if (!sendTarget) return;
    try {
      const v = await sendForm.validateFields();
      const payload: SendConnectorMessageRequest = {
        channel: v.channel,
        type: v.type,
        title: v.title,
        content: v.content,
      };
      await connectorService.send(sendTarget.name, payload);
      message.success(t('connectors.send.success', { name: sendTarget.name, channel: v.channel }));
      setSendOpen(false);
    } catch (e) {
      if ((e as { errorFields?: unknown }).errorFields) return;
      message.error(t('connectors.send.failed', { msg: (e as Error).message }));
    }
  };

  const instanceOf = (m: ConnectorManifest): ConnectorConfig | undefined =>
    instances.find(c => c.name === m.name);

  const marketColumns = [
    {
      title: t('connectors.columns.name'), dataIndex: 'title', key: 'title', width: 200,
      render: (col: string, r: ConnectorManifest) => (
        <Space orientation="vertical" size={0}>
          <Text strong>{col}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{r.name} · v{r.version}</Text>
        </Space>
      ),
    },
    {
      title: t('connectors.columns.type'), dataIndex: 'type', key: 'type', width: 100,
      render: (col: string) => <Tag color={col === 'im' ? 'blue' : col === 'webhook' ? 'cyan' : 'default'}>{col}</Tag>,
    },
    {
      title: t('connectors.columns.provider'), dataIndex: 'provider', key: 'provider', width: 110,
      render: (p: string) => <Tag>{p}</Tag>,
    },
    {
      title: t('connectors.columns.capabilities'), dataIndex: 'capabilities', key: 'capabilities',
      render: (caps: string[]) => (
        <Space wrap size={[4, 4]}>
          {caps.map(c => <Tag key={c} color="geekblue">{c}</Tag>)}
        </Space>
      ),
    },
    {
      title: t('connectors.columns.status'), key: 'status', width: 120,
      render: (_: unknown, r: ConnectorManifest) => {
        const inst = instanceOf(r);
        if (!inst) return <Tag color="default">{t('connectors.status.notEnabled')}</Tag>;
        if (!inst.enabled) return <Tag color="orange">{t('connectors.status.disabled')}</Tag>;
        const h = health[`${0}/${r.name}/${r.provider}`];
        return h ? (
          <Tag color={h.ok ? 'green' : 'red'} icon={h.ok ? <CheckCircle /> : <XCircle />}>
            {h.ok ? t('connectors.status.running') : t('connectors.status.error')}
          </Tag>
        ) : <Tag color="green">{t('connectors.status.enabled')}</Tag>;
      },
    },
    {
      title: t('connectors.columns.actions'), key: 'actions', width: 220, fixed: 'right' as const,
      render: (_: unknown, r: ConnectorManifest) => {
        const inst = instanceOf(r);
        return (
          <Space>
            <Button size="small" icon={<Plug />} onClick={() => { setDetailTarget(r); setDetailOpen(true); }}>
              {t('connectors.buttons.detail')}
            </Button>
            {inst ? (
              <>
                <Button size="small" icon={<Send />} onClick={() => openSend(inst)} type="primary" ghost>
                  {t('connectors.buttons.sendMessage')}
                </Button>
                <Button size="small" icon={<Power />} danger onClick={() => handleRevoke(inst)}>
                  {t('connectors.buttons.disable')}
                </Button>
              </>
            ) : (
              <Button size="small" type="primary" icon={<Plus />} onClick={() => openProvision(r)}>
                {t('connectors.buttons.enable')}
              </Button>
            )}
          </Space>
        );
      },
    },
  ];

  const instanceColumns = [
    { title: t('connectors.columns.name'), dataIndex: 'name', key: 'name', width: 140, render: (n: string) => <Text strong>{n}</Text> },
    { title: t('connectors.columns.provider'), dataIndex: 'provider', key: 'provider', width: 110, render: (p: string) => <Tag>{p}</Tag> },
    { title: t('connectors.columns.type'), dataIndex: 'type', key: 'type', width: 100, render: (col: string) => <Tag>{col}</Tag> },
    {
      title: t('connectors.columns.status'), dataIndex: 'enabled', key: 'enabled', width: 100,
      render: (e: boolean) => (
        <Tag color={e ? 'green' : 'orange'}>
          {e ? t('connectors.status.enabled') : t('connectors.status.disabled')}
        </Tag>
      ),
    },
    {
      title: t('connectors.provision.fieldCredentials'), dataIndex: 'credentials', key: 'credentials',
      render: (c?: Record<string, string>) => (
        <Space wrap size={[4, 4]}>
          {c ? Object.keys(c).map(k => <Tag key={k}>{k}</Tag>) : '-'}
        </Space>
      ),
    },
    {
      title: t('connectors.columns.actions'), key: 'actions', width: 220, fixed: 'right' as const,
      render: (_: unknown, r: ConnectorConfig) => (
        <Space>
          <Button size="small" icon={<Send />} onClick={() => openSend(r)} type="primary" ghost>
            {t('connectors.buttons.sendMessage')}
          </Button>
          <Button size="small" icon={<Send />} onClick={() => handleTest(r)}>
            {t('connectors.buttons.test')}
          </Button>
          <Button size="small" icon={<Power />} danger onClick={() => handleRevoke(r)}>
            {t('connectors.buttons.disable')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      header={{
        title: t('connectors.title'),
        breadcrumb: {
          items: [
            { title: t('connectors.breadcrumbHome') },
            { title: t('connectors.breadcrumbCurrent') },
          ],
        },
      }}
      extra={
        <Space>
          <Button icon={<RotateCcw />} onClick={load} loading={loading}>
            {t('connectors.buttons.refresh')}
          </Button>
        </Space>
      }
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message={t('connectors.pageDescription')}
        description={t('connectors.description')}
      />

      <Tabs
        activeKey={tab}
        onChange={(k) => setTab(k as Tab)}
        items={[
          {
            key: 'market',
            label: `${t('connectors.tabs.market')} (${market.length})`,
            children: market.length === 0 ? (
              <Empty description={t('connectors.marketEmpty')} />
            ) : (
              <Table
                rowKey="name"
                loading={loading}
                columns={marketColumns}
                dataSource={market}
                scroll={{ x: 1100 }}
                pagination={false}
              />
            ),
          },
          {
            key: 'instances',
            label: `${t('connectors.tabs.configured')} (${instances.length})`,
            children: instances.length === 0 ? (
              <Empty description={t('connectors.configuredEmpty')} />
            ) : (
              <Table
                rowKey="name"
                loading={loading}
                columns={instanceColumns}
                dataSource={instances}
                scroll={{ x: 1000 }}
                pagination={false}
              />
            ),
          },
        ]}
      />

      <Modal
        title={provisionTarget ? t('connectors.provision.enableTitle', { title: provisionTarget.title }) : t('connectors.actions.enable')}
        open={provisionOpen}
        onCancel={() => setProvisionOpen(false)}
        onOk={submitProvision}
        width={640}
        okText={t('connectors.actions.enable')}
        cancelText={t('common.cancel')}
      >
        {provisionTarget && (
          <>
            <Paragraph type="secondary">{provisionTarget.description}</Paragraph>
            <Form form={form} layout="vertical">
              <Form.Item name="enabled" label={t('connectors.provision.fieldEnabled')} valuePropName="checked">
                <Switch />
              </Form.Item>
              <Form.Item name="provider" label={t('connectors.provision.fieldProvider')}>
                <Input placeholder={t('connectors.provision.providerPlaceholder')} />
              </Form.Item>
              {provisionTarget.name === 'email' ? <>
                <Alert type="warning" showIcon message="请使用 QQ 邮箱授权码，不要填写登录密码。凭据只会加密保存在服务端，后续不会回显。" style={{ marginBottom: 16 }} />
                <Form.Item name="emailUsername" label="邮箱地址" rules={[{ required: true, type: 'email' }]}><Input autoComplete="off" /></Form.Item>
                <Form.Item name="emailPassword" label="邮箱授权码" rules={[{ required: true }]}><Input.Password autoComplete="new-password" /></Form.Item>
                <Space align="start" wrap>
                  <Form.Item name="imapHost" label="IMAP 主机" rules={[{ required: true }]}><Input /></Form.Item>
                  <Form.Item name="imapPort" label="IMAP 端口" rules={[{ required: true }]}><InputNumber min={1} max={65535} /></Form.Item>
                  <Form.Item name="smtpHost" label="SMTP 主机" rules={[{ required: true }]}><Input /></Form.Item>
                  <Form.Item name="smtpPort" label="SMTP 端口" rules={[{ required: true }]}><InputNumber min={1} max={65535} /></Form.Item>
                </Space>
                <Form.Item name="mailbox" label="收件箱" rules={[{ required: true }]}><Input /></Form.Item>
                <Form.Item name="pollIntervalSeconds" label="轮询间隔（秒）" rules={[{ required: true }]}><InputNumber min={15} /></Form.Item>
              </> : <><Form.Item
                name="credText"
                label={t('connectors.provision.fieldCredentials')}
                tooltip={t('connectors.provision.credentialsTooltip')}
              >
                <Input.TextArea rows={4} placeholder={t('connectors.provision.credentialsPlaceholder')} />
              </Form.Item>
              <Form.Item
                name="settingText"
                label={t('connectors.provision.fieldSettings')}
                tooltip={t('connectors.provision.settingsTooltip')}
              >
                <Input.TextArea rows={3} placeholder={t('connectors.provision.settingsPlaceholder')} />
              </Form.Item></>}
            </Form>
          </>
        )}
      </Modal>

      <Modal
        title={sendTarget ? t('connectors.send.title', { name: sendTarget.name }) : t('connectors.actions.send')}
        open={sendOpen}
        onCancel={() => setSendOpen(false)}
        onOk={submitSend}
        width={600}
        okText={t('connectors.actions.send')}
        cancelText={t('common.cancel')}
      >
        <Form form={sendForm} layout="vertical">
          <Form.Item name="channel" label={t('connectors.send.fieldChannel')} rules={[{ required: true }]}>
            <Input placeholder={t('connectors.send.channelPlaceholder')} />
          </Form.Item>
          <Form.Item name="type" label={t('connectors.send.fieldType')} rules={[{ required: true }]}>
            <Input placeholder={t('connectors.send.typePlaceholder')} />
          </Form.Item>
          <Form.Item name="title" label={t('connectors.send.fieldTitle')}>
            <Input placeholder={t('connectors.send.titlePlaceholder')} />
          </Form.Item>
          <Form.Item name="content" label={t('connectors.send.fieldContent')} rules={[{ required: true }]}>
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={detailTarget ? detailTarget.title : ''}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        width={520}
      >
        {detailTarget && (
          <Space orientation="vertical" style={{ width: '100%' }} size="middle">
            <div>
              <Text type="secondary">{t('connectors.detail.identifier')}</Text>
              <div><Text code>{detailTarget.name}</Text> <Tag>v{detailTarget.version}</Tag></div>
            </div>
            <div>
              <Text type="secondary">{t('connectors.detail.providerType')}</Text>
              <div><Tag>{detailTarget.provider}</Tag><Tag color="blue">{detailTarget.type}</Tag></div>
            </div>
            <div>
              <Text type="secondary">{t('connectors.detail.description')}</Text>
              <Paragraph>{detailTarget.description}</Paragraph>
            </div>
            <div>
              <Text type="secondary">{t('connectors.detail.capabilities')}</Text>
              <div>{detailTarget.capabilities.map(c => <Tag key={c} color="geekblue">{c}</Tag>)}</div>
            </div>
            {detailTarget.tags && detailTarget.tags.length > 0 && (
              <div>
                <Text type="secondary">{t('connectors.detail.tags')}</Text>
                <div>{detailTarget.tags.map(t => <Tag key={t}>{t}</Tag>)}</div>
              </div>
            )}
            {detailTarget.homepage && (
              <div>
                <Text type="secondary">{t('connectors.detail.homepage')}</Text>
                <div><a href={detailTarget.homepage} target="_blank" rel="noreferrer">{detailTarget.homepage}</a></div>
              </div>
            )}
          </Space>
        )}
      </Drawer>
    </PageContainer>
  );
}
