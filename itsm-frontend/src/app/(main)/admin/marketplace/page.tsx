'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Segmented,
  Select,
  Space,
  Statistic,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ArrowDownCircle, ArrowUpCircle, Boxes, History, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import {
  CATALOG_STATUS_META,
  createCatalogItem,
  createItemVersion,
  deleteCatalogItem,
  getCatalogItem,
  listCatalogItems,
  publishCatalogItem,
  publishItemVersion,
  unpublishCatalogItem,
  updateCatalogItem,
} from '@/lib/api/marketplace-catalog-api';
import type {
  CatalogItem,
  CatalogItemDetail,
  CatalogItemInput,
  CatalogItemStatus,
  CatalogItemType,
  CatalogItemVersion,
} from '@/lib/api/marketplace-catalog-api';

const { Text } = Typography;

/**
 * 商品目录管理（平台运营）。
 *
 * 引入原因：`marketplace_items` 只有 seeder 能写入，管理员无法上架/下架/改版本，
 * 于是「应用市场」在客户环境里永远只有出厂那几条商品——这是该能力停在 Pilot 的
 * 真实阻断项，不是接口缺失。
 *
 * 三个刻意的取舍：
 *  1. 页面只给 sysadmin（菜单权限码 marketplace_catalog:read）。目录是全局表，
 *     租户管理员能写就会跨租户污染，所以不做「按租户过滤」这种假隔离。
 *  2. 上架前必须已有发布版本，且版本号不可手填 latest_version——否则安装记录
 *     会写下查不到的版本号。
 *  3. 下架用「停止新安装」而不是删除：已安装租户的卸载路径仍要读该条目。
 */

type StatusFilter = 'all' | CatalogItemStatus;

const STATUS_OPTIONS: { label: string; value: StatusFilter }[] = [
  { label: '全部', value: 'all' },
  { label: '已上架', value: 'published' },
  { label: '草稿', value: 'draft' },
  { label: '已下架', value: 'deprecated' },
];

const TYPE_OPTIONS: { label: string; value: CatalogItemType }[] = [
  { label: '连接器', value: 'connector' },
  { label: '技能', value: 'skill' },
  { label: '插件', value: 'plugin' },
];

interface ItemFormValues {
  name: string;
  type: CatalogItemType;
  title: string;
  provider: string;
  category?: string;
  description?: string;
  iconUrl?: string;
  tags?: string[];
  homepage?: string;
  repository?: string;
  license?: string;
  isOfficial?: boolean;
  isFree?: boolean;
  price?: number;
  version?: string;
}

export default function MarketplaceCatalogAdminPage() {
  const { message, modal } = App.useApp();
  const [items, setItems] = useState<CatalogItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<StatusFilter>('all');
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<CatalogItem | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<ItemFormValues>();
  const selectedType = Form.useWatch('type', form);

  const [versionsOpen, setVersionsOpen] = useState(false);
  const [versionTarget, setVersionTarget] = useState<CatalogItemDetail | null>(null);
  const [versionForm] = Form.useForm<{ version: string; changelog?: string }>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listCatalogItems({
        status: status === 'all' ? undefined : status,
        search: keyword.trim() || undefined,
        page,
        pageSize,
      });
      setItems(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载商品目录失败');
      setItems([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [keyword, message, page, status]);

  useEffect(() => {
    void load();
  }, [load]);

  const summary = useMemo(() => {
    const published = items.filter(item => item.status === 'published').length;
    const draft = items.filter(item => item.status === 'draft').length;
    const installed = items.reduce((sum, item) => sum + item.installCount, 0);
    return { published, draft, installed };
  }, [items]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ type: 'connector', isOfficial: true, isFree: true, price: 0 });
    setFormOpen(true);
  };

  const openEdit = (item: CatalogItem) => {
    setEditing(item);
    form.setFieldsValue({
      name: item.name,
      type: item.type,
      title: item.title,
      provider: item.provider,
      category: item.category,
      description: item.description,
      iconUrl: item.iconUrl,
      tags: item.tags,
      homepage: item.homepage,
      repository: item.repository,
      license: item.license,
      isOfficial: item.isOfficial,
      isFree: item.isFree,
      price: item.price,
    });
    setFormOpen(true);
  };

  const submitForm = async () => {
    const values = await form.validateFields();
    const payload: CatalogItemInput = {
      name: values.name,
      type: values.type,
      title: values.title,
      provider: values.provider,
      category: values.category,
      description: values.description,
      iconUrl: values.iconUrl,
      tags: values.tags,
      homepage: values.homepage,
      repository: values.repository,
      license: values.license,
      isOfficial: values.isOfficial,
      isFree: values.isFree,
      price: values.price,
    };
    setSubmitting(true);
    try {
      if (editing) {
        // name/type 不提交：后端会拒绝（身份不可变），这里也不给用户改的机会。
        delete payload.name;
        delete payload.type;
        await updateCatalogItem(editing.id, payload);
        message.success('已保存');
      } else {
        await createCatalogItem({ ...payload, version: values.version });
        message.success('已创建草稿，确认版本后点击「上架」才会对租户可见');
      }
      setFormOpen(false);
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败');
    } finally {
      setSubmitting(false);
    }
  };

  const runAction = async (action: () => Promise<unknown>, successText: string) => {
    try {
      await action();
      message.success(successText);
      await load();
    } catch (error) {
      // 后端的 409 文案本身就是处置指引（先发布版本 / 先下架 / 已有租户安装），
      // 原样透出比「操作失败」有用得多。
      message.error(error instanceof Error ? error.message : '操作失败');
    }
  };

  const confirmDelete = (item: CatalogItem) => {
    modal.confirm({
      title: `删除「${item.title}」？`,
      content: '仅未上架且从未被安装的条目可以删除。已上架的请先下架。',
      okButtonProps: { danger: true },
      okText: '删除',
      cancelText: '取消',
      onOk: () => runAction(() => deleteCatalogItem(item.id), '已删除'),
    });
  };

  const openVersions = async (item: CatalogItem) => {
    try {
      // 详情接口内联返回版本：页面判断「能不能上架」的依据就是有没有 published 版本。
      setVersionTarget(await getCatalogItem(item.id));
      setVersionsOpen(true);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载版本失败');
    }
  };

  const refreshVersions = async () => {
    if (!versionTarget) return;
    setVersionTarget(await getCatalogItem(versionTarget.item.id));
    await load();
  };

  const submitVersion = async () => {
    if (!versionTarget) return;
    const values = await versionForm.validateFields();
    setSubmitting(true);
    try {
      await createItemVersion(versionTarget.item.id, values);
      message.success('版本已创建（草稿）。发布后才会推进线上版本号');
      versionForm.resetFields();
      await refreshVersions();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '新增版本失败');
    } finally {
      setSubmitting(false);
    }
  };

  const columns: ColumnsType<CatalogItem> = [
    {
      title: '商品',
      dataIndex: 'title',
      render: (_: unknown, item) => (
        <Space direction='vertical' size={0}>
          <Text strong>{item.title}</Text>
          <Text type='secondary' style={{ fontSize: 12 }}>
            {item.name} · {item.provider}
          </Text>
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 90,
      render: (type: CatalogItemType) => (
        <Tag>{TYPE_OPTIONS.find(option => option.value === type)?.label ?? type}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (value: CatalogItemStatus) => {
        const meta = CATALOG_STATUS_META[value];
        return (
          <Tooltip title={meta.hint}>
            <Tag color={meta.color}>{meta.label}</Tag>
          </Tooltip>
        );
      },
    },
    {
      title: '线上版本',
      dataIndex: 'latestVersion',
      width: 110,
      render: (version: string) =>
        version ? <Text code>{version}</Text> : <Text type='secondary'>未发布</Text>,
    },
    {
      title: '安装数',
      dataIndex: 'installCount',
      width: 90,
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      width: 170,
      render: (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '—'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 260,
      render: (_: unknown, item) => (
        <Space size={4} wrap>
          {item.status === 'published' ? (
            <Button
              type='link'
              size='small'
              icon={<ArrowDownCircle size={14} />}
              onClick={() => runAction(() => unpublishCatalogItem(item.id), '已下架：停止新安装，已安装租户不受影响')}
            >
              下架
            </Button>
          ) : (
            <Button
              type='link'
              size='small'
              icon={<ArrowUpCircle size={14} />}
              onClick={() => runAction(() => publishCatalogItem(item.id), '已上架')}
            >
              上架
            </Button>
          )}
          <Button type='link' size='small' icon={<History size={14} />} onClick={() => void openVersions(item)}>
            版本
          </Button>
          <Button type='link' size='small' icon={<Pencil size={14} />} onClick={() => openEdit(item)}>
            编辑
          </Button>
          {item.status !== 'published' && (
            <Button
              type='link'
              size='small'
              danger
              icon={<Trash2 size={14} />}
              onClick={() => confirmDelete(item)}
            >
              删除
            </Button>
          )}
        </Space>
      ),
    },
  ];

  const versionColumns: ColumnsType<CatalogItemVersion> = [
    {
      title: '版本',
      dataIndex: 'version',
      width: 100,
      render: (version: string) => <Text code>{version}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (value: CatalogItemVersion['status']) => (
        <Tag color={value === 'published' ? 'green' : value === 'draft' ? 'default' : 'orange'}>
          {value === 'published' ? '已发布' : value === 'draft' ? '草稿' : value === 'deprecated' ? '已弃用' : '已撤回'}
        </Tag>
      ),
    },
    { title: '更新日志', dataIndex: 'changelog', render: (text: string) => text || '—' },
    {
      title: '发布时间',
      dataIndex: 'releasedAt',
      width: 170,
      render: (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '—'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 130,
      render: (_: unknown, version) => {
        const isLive = versionTarget?.item.latestVersion === version.version;
        return (
          <Tooltip title={isLive ? '当前线上版本' : '发布该版本会把线上版本推进/回滚到它'}>
            <Button
              type='link'
              size='small'
              disabled={isLive || version.status === 'withdrawn'}
              onClick={() =>
                versionTarget &&
                runAction(() => publishItemVersion(versionTarget.item.id, version.version), `已发布 ${version.version}`)
                  .then(refreshVersions)
              }
            >
              {isLive ? '线上' : '发布'}
            </Button>
          </Tooltip>
        );
      },
    },
  ];

  return (
    <PageContainer
      header={{ title: '商品目录' }}
      extra={
        <Space>
          <Button icon={<RefreshCw size={16} />} onClick={() => void load()} loading={loading}>
            刷新
          </Button>
          <Button type='primary' icon={<Plus size={16} />} onClick={openCreate}>
            新建商品
          </Button>
        </Space>
      }
    >
      <Space direction='vertical' size={16} style={{ width: '100%' }}>
        <Row gutter={16}>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title='商品总数' value={total} prefix={<Boxes size={16} />} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title='已上架' value={summary.published} valueStyle={{ color: '#3f8600' }} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title='草稿' value={summary.draft} valueStyle={{ color: '#d48806' }} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title='本页安装数' value={summary.installed} />
            </Card>
          </Col>
        </Row>

        <Alert
          type='info'
          showIcon
          message='目录是全局的，对所有租户生效'
          description='上架后所有租户都能浏览与安装；下架只停止新安装，已安装租户不受影响。连接器类商品的名称必须对应一个已注册的连接器（如 feishu / wecom / dingtalk / webhook），否则安装时无法启动运行态。'
        />

        <Space wrap>
          <Segmented<StatusFilter>
            options={STATUS_OPTIONS}
            value={status}
            onChange={value => {
              setPage(1);
              setStatus(value);
            }}
          />
          <Input.Search
            allowClear
            placeholder='按名称/标题/提供商搜索'
            style={{ width: 280 }}
            onSearch={value => {
              setPage(1);
              setKeyword(value);
            }}
          />
        </Space>

        <Table<CatalogItem>
          rowKey='id'
          size='small'
          loading={loading}
          columns={columns}
          dataSource={items}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: false,
            onChange: setPage,
          }}
          locale={{
            emptyText: (
              <Empty
                description={
                  total === 0
                    ? '目录为空。可以新建商品，或确认 seeder 是否已导入出厂商品'
                    : '没有符合筛选条件的商品'
                }
              />
            ),
          }}
        />
      </Space>

      <Drawer
        title={editing ? `编辑：${editing.title}` : '新建商品'}
        width={520}
        open={formOpen}
        onClose={() => setFormOpen(false)}
        extra={
          <Space>
            <Button onClick={() => setFormOpen(false)}>取消</Button>
            <Button type='primary' loading={submitting} onClick={() => void submitForm()}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label='标识（name）'
            extra={
              editing
                ? '名称创建后不可修改：它决定安装时的连接器 provider。'
                : '小写字母/数字/连字符。连接器类请用 <provider>-connector，如 feishu-connector。'
            }
            rules={[{ required: true, message: '请输入标识' }]}
          >
            <Input disabled={Boolean(editing)} placeholder='feishu-connector' />
          </Form.Item>
          <Form.Item name='type' label='类型' rules={[{ required: true }]} extra='创建后不可修改'>
            <Select options={TYPE_OPTIONS} disabled={Boolean(editing)} />
          </Form.Item>
          <Form.Item
            name='title'
            label='显示名称'
            rules={[{ required: true, message: '请输入显示名称' }]}
            extra={selectedType === 'connector' && !editing ? '留空则取连接器 manifest 中的标题' : undefined}
          >
            <Input placeholder='飞书审批增强连接器' />
          </Form.Item>
          <Form.Item name='provider' label='提供商' extra='展示用，与连接器 provider 无关'>
            <Input placeholder='ITSM Foundation' />
          </Form.Item>
          <Form.Item name='category' label='分类'>
            <Input placeholder='im / monitoring / ai' />
          </Form.Item>
          <Form.Item name='description' label='简介'>
            <Input.TextArea rows={2} maxLength={200} showCount />
          </Form.Item>
          <Form.Item name='tags' label='标签'>
            <Select mode='tags' placeholder='回车添加' />
          </Form.Item>
          <Form.Item name='iconUrl' label='图标 URL'>
            <Input placeholder='https://…' />
          </Form.Item>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name='isOfficial' label='官方' valuePropName='checked'>
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name='isFree' label='免费' valuePropName='checked'>
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name='price' label='价格'>
                <InputNumber min={0} precision={2} style={{ width: '100%' }} disabled={form.getFieldValue('isFree')} />
              </Form.Item>
            </Col>
          </Row>
          {!editing && (
            <Form.Item
              name='version'
              label='初始版本号'
              extra={
                selectedType === 'connector'
                  ? '连接器类留空则取运行时 manifest 版本；填写时必须与 manifest 一致。'
                  : '技能/插件类必填，格式 x.y.z。'
              }
            >
              <Input placeholder='1.0.0' />
            </Form.Item>
          )}
        </Form>
      </Drawer>

      <Drawer
        title={versionTarget ? `版本：${versionTarget.item.title}` : '版本'}
        width={720}
        open={versionsOpen}
        onClose={() => setVersionsOpen(false)}
      >
        {versionTarget && (
          <Space direction='vertical' size={16} style={{ width: '100%' }}>
            <Descriptions size='small' column={2} bordered>
              <Descriptions.Item label='线上版本'>
                {versionTarget.item.latestVersion || '未发布'}
              </Descriptions.Item>
              <Descriptions.Item label='状态'>
                <Tag color={CATALOG_STATUS_META[versionTarget.item.status].color}>
                  {CATALOG_STATUS_META[versionTarget.item.status].label}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            <Card size='small' title='新增版本'>
              <Form form={versionForm} layout='inline' style={{ rowGap: 12 }}>
                <Form.Item name='version' rules={[{ required: true, message: '版本号' }]}>
                  <Input placeholder='版本号，如 1.1.0' style={{ width: 160 }} />
                </Form.Item>
                <Form.Item name='changelog'>
                  <Input placeholder='更新日志' style={{ width: 300 }} />
                </Form.Item>
                <Form.Item>
                  <Button type='primary' loading={submitting} onClick={() => void submitVersion()}>
                    新增
                  </Button>
                </Form.Item>
              </Form>
              <Text type='secondary' style={{ fontSize: 12 }}>
                新版本默认草稿，发布后才会推进线上版本号。连接器类版本号必须与运行时 manifest 一致。
              </Text>
            </Card>

            <Table<CatalogItemVersion>
              rowKey='version'
              size='small'
              columns={versionColumns}
              dataSource={versionTarget.versions}
              pagination={false}
              locale={{ emptyText: <Empty description='还没有版本，先新增一个' /> }}
            />
          </Space>
        )}
      </Drawer>
    </PageContainer>
  );
}
