'use client';

import React, { useEffect, useState } from 'react';
import {
  Card, Table, Tag, Button, Space, Modal, Form, Input, Switch, Tabs, message, Drawer,
  Typography, Empty, Alert, Spin, Tooltip,
} from 'antd';
import {
  ApiOutlined, CheckCircleOutlined, CloseCircleOutlined, PlusOutlined,
  PoweroffOutlined, SendOutlined, ReloadOutlined, SettingOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@/app/components/PageContainer';
import connectorService, {
  ConnectorManifest, ConnectorConfig, SendConnectorMessageRequest,
} from '@/lib/services/connector-service';

const { Text, Paragraph } = Typography;

type Tab = 'market' | 'instances';

export default function ConnectorsAdminPage() {
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
      message.error(`加载失败: ${(e as Error).message}`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openProvision = (m: ConnectorManifest) => {
    setProvisionTarget(m);
    form.resetFields();
    form.setFieldsValue({ enabled: true, provider: m.provider });
    setProvisionOpen(true);
  };

  const submitProvision = async () => {
    if (!provisionTarget) return;
    try {
      const values = await form.validateFields();
      // 解析 credentials / settings：行格式 "key=value"
      const credentials: Record<string, string> = {};
      const settings: Record<string, unknown> = {};
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
      message.success(`✓ 已启用 ${provisionTarget.title}`);
      setProvisionOpen(false);
      load();
    } catch (e) {
      if ((e as { errorFields?: unknown }).errorFields) return;
      message.error(`启用失败: ${(e as Error).message}`);
    }
  };

  const handleRevoke = async (c: ConnectorConfig) => {
    Modal.confirm({
      title: `停用 ${c.name}?`,
      content: '该连接器实例将被关闭并从当前租户移除。',
      okType: 'danger',
      onOk: async () => {
        await connectorService.revoke(c.name);
        message.success(`已停用 ${c.name}`);
        load();
      },
    });
  };

  const handleTest = async (c: ConnectorConfig) => {
    try {
      const r = await connectorService.test(c.name);
      message.success(`✓ 测试消息已发送至 ${r.channel}`);
    } catch (e) {
      message.error(`测试失败: ${(e as Error).message}`);
    }
  };

  const openSend = (c: ConnectorConfig) => {
    setSendTarget(c);
    sendForm.resetFields();
    sendForm.setFieldsValue({ type: 'text', content: '来自 ITSM 的消息' });
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
      message.success(`✓ 已通过 ${sendTarget.name} 发送给 ${v.channel}`);
      setSendOpen(false);
    } catch (e) {
      if ((e as { errorFields?: unknown }).errorFields) return;
      message.error(`发送失败: ${(e as Error).message}`);
    }
  };

  const instanceOf = (m: ConnectorManifest): ConnectorConfig | undefined =>
    instances.find(c => c.name === m.name);

  const marketColumns = [
    {
      title: '名称', dataIndex: 'title', key: 'title', width: 200,
      render: (t: string, r: ConnectorManifest) => (
        <Space direction="vertical" size={0}>
          <Text strong>{t}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{r.name} · v{r.version}</Text>
        </Space>
      ),
    },
    {
      title: '类型', dataIndex: 'type', key: 'type', width: 100,
      render: (t: string) => <Tag color={t === 'im' ? 'blue' : t === 'webhook' ? 'cyan' : 'default'}>{t}</Tag>,
    },
    {
      title: '提供商', dataIndex: 'provider', key: 'provider', width: 110,
      render: (p: string) => <Tag>{p}</Tag>,
    },
    {
      title: '能力', dataIndex: 'capabilities', key: 'capabilities',
      render: (caps: string[]) => (
        <Space wrap size={[4, 4]}>
          {caps.map(c => <Tag key={c} color="geekblue">{c}</Tag>)}
        </Space>
      ),
    },
    {
      title: '状态', key: 'status', width: 120,
      render: (_: unknown, r: ConnectorManifest) => {
        const inst = instanceOf(r);
        if (!inst) return <Tag color="default">未启用</Tag>;
        if (!inst.enabled) return <Tag color="orange">已停用</Tag>;
        const h = health[`${0}/${r.name}/${r.provider}`];
        return h ? (
          <Tag color={h.ok ? 'green' : 'red'} icon={h.ok ? <CheckCircleOutlined /> : <CloseCircleOutlined />}>
            {h.ok ? '运行中' : '异常'}
          </Tag>
        ) : <Tag color="green">已启用</Tag>;
      },
    },
    {
      title: '操作', key: 'actions', width: 220, fixed: 'right' as const,
      render: (_: unknown, r: ConnectorManifest) => {
        const inst = instanceOf(r);
        return (
          <Space>
            <Button size="small" icon={<ApiOutlined />} onClick={() => { setDetailTarget(r); setDetailOpen(true); }}>详情</Button>
            {inst ? (
              <>
                <Button size="small" icon={<SendOutlined />} onClick={() => openSend(inst)} type="primary" ghost>发消息</Button>
                <Button size="small" icon={<PoweroffOutlined />} danger onClick={() => handleRevoke(inst)}>停用</Button>
              </>
            ) : (
              <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => openProvision(r)}>启用</Button>
            )}
          </Space>
        );
      },
    },
  ];

  const instanceColumns = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 140, render: (n: string) => <Text strong>{n}</Text> },
    { title: '提供商', dataIndex: 'provider', key: 'provider', width: 110, render: (p: string) => <Tag>{p}</Tag> },
    { title: '类型', dataIndex: 'type', key: 'type', width: 100, render: (t: string) => <Tag>{t}</Tag> },
    {
      title: '状态', dataIndex: 'enabled', key: 'enabled', width: 100,
      render: (e: boolean) => <Tag color={e ? 'green' : 'orange'}>{e ? '已启用' : '已停用'}</Tag>,
    },
    {
      title: '凭据', dataIndex: 'credentials', key: 'credentials',
      render: (c?: Record<string, string>) => (
        <Space wrap size={[4, 4]}>
          {c ? Object.keys(c).map(k => <Tag key={k}>{k}</Tag>) : '-'}
        </Space>
      ),
    },
    {
      title: '操作', key: 'actions', width: 220, fixed: 'right' as const,
      render: (_: unknown, r: ConnectorConfig) => (
        <Space>
          <Button size="small" icon={<SendOutlined />} onClick={() => openSend(r)} type="primary" ghost>发消息</Button>
          <Button size="small" icon={<SendOutlined />} onClick={() => handleTest(r)}>测试</Button>
          <Button size="small" icon={<PoweroffOutlined />} danger onClick={() => handleRevoke(r)}>停用</Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      header={{
        title: '🧩 连接器 / 插件 / 技能 / IM 市场',
        breadcrumb: { items: [{ title: '管理' }, { title: '连接器' }] },
      }}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
        </Space>
      }
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="AI-Native 集成层"
        description="飞书 / 钉钉 / 企微 / Webhook / 邮件 / 数据库 / 监控告警 全部走统一的 Connector 框架。新连接器即插即用，无需改核心代码。"
      />

      <Tabs
        activeKey={tab}
        onChange={(k) => setTab(k as Tab)}
        items={[
          {
            key: 'market', label: `市场 (${market.length})`,
            children: market.length === 0 ? (
              <Empty description="暂无可用连接器" />
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
            key: 'instances', label: `已配置 (${instances.length})`,
            children: instances.length === 0 ? (
              <Empty description="还没有启用任何连接器，去市场页启用一个吧" />
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

      {/* 启用连接器 */}
      <Modal
        title={provisionTarget ? `启用 ${provisionTarget.title}` : '启用'}
        open={provisionOpen}
        onCancel={() => setProvisionOpen(false)}
        onOk={submitProvision}
        width={640}
        okText="启用"
      >
        {provisionTarget && (
          <>
            <Paragraph type="secondary">{provisionTarget.description}</Paragraph>
            <Form form={form} layout="vertical">
              <Form.Item name="enabled" label="启用" valuePropName="checked">
                <Switch />
              </Form.Item>
              <Form.Item name="provider" label="提供商">
                <Input placeholder="provider（可留空）" />
              </Form.Item>
              <Form.Item
                name="credText"
                label="凭据（每行 key=value）"
                tooltip="app_id / app_secret / corp_id 等"
              >
                <Input.TextArea rows={4} placeholder={'app_id=cli_xxx\napp_secret=xxx'} />
              </Form.Item>
              <Form.Item
                name="settingText"
                label="设置（每行 key=value）"
                tooltip="base_url / debug_channel 等"
              >
                <Input.TextArea rows={3} placeholder={'debug_channel=ou_xxx'} />
              </Form.Item>
            </Form>
          </>
        )}
      </Modal>

      {/* 发送消息 */}
      <Modal
        title={sendTarget ? `通过 ${sendTarget.name} 发送` : '发送'}
        open={sendOpen}
        onCancel={() => setSendOpen(false)}
        onOk={submitSend}
        width={600}
        okText="发送"
      >
        <Form form={sendForm} layout="vertical">
          <Form.Item name="channel" label="目标" rules={[{ required: true }]}>
            <Input placeholder="ou_xxx（飞书 open_id）/ 钉钉 userid / WeCom UserID / webhook URL" />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true }]}>
            <Input placeholder="text / markdown / card" />
          </Form.Item>
          <Form.Item name="title" label="标题">
            <Input placeholder="可选" />
          </Form.Item>
          <Form.Item name="content" label="内容" rules={[{ required: true }]}>
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情 */}
      <Drawer
        title={detailTarget ? detailTarget.title : ''}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        width={520}
      >
        {detailTarget && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <div>
              <Text type="secondary">标识</Text>
              <div><Text code>{detailTarget.name}</Text> <Tag>v{detailTarget.version}</Tag></div>
            </div>
            <div>
              <Text type="secondary">提供商 / 类型</Text>
              <div><Tag>{detailTarget.provider}</Tag><Tag color="blue">{detailTarget.type}</Tag></div>
            </div>
            <div>
              <Text type="secondary">描述</Text>
              <Paragraph>{detailTarget.description}</Paragraph>
            </div>
            <div>
              <Text type="secondary">能力</Text>
              <div>{detailTarget.capabilities.map(c => <Tag key={c} color="geekblue">{c}</Tag>)}</div>
            </div>
            {detailTarget.tags && detailTarget.tags.length > 0 && (
              <div>
                <Text type="secondary">标签</Text>
                <div>{detailTarget.tags.map(t => <Tag key={t}>{t}</Tag>)}</div>
              </div>
            )}
            {detailTarget.homepage && (
              <div>
                <Text type="secondary">主页</Text>
                <div><a href={detailTarget.homepage} target="_blank" rel="noreferrer">{detailTarget.homepage}</a></div>
              </div>
            )}
          </Space>
        )}
      </Drawer>
    </PageContainer>
  );
}
