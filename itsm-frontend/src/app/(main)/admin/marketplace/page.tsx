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
import { formatDateTime } from '@/lib/formatters';
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
  { label: 'All', value: 'all' },
  { label: 'Already on the shelves', value: 'published' },
  { label: 'Draft', value: 'draft' },
  { label: 'Removed', value: 'deprecated' },
];

const TYPE_OPTIONS: { label: string; value: CatalogItemType }[] = [
  { label: 'Connectors', value: 'connector' },
  { label: 'Skill', value: 'skill' },
  { label: 'Plug-in', value: 'plugin' },
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
      message.error(error instanceof Error ? error.message : 'Failed to load product catalog');
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
        message.success('Saved');
      } else {
        await createCatalogItem({ ...payload, version: values.version });
        message.success('A draft has been created. After confirming the version, click "Publish" to make it visible to tenants.');
      }
      setFormOpen(false);
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'SaveFailed');
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
      message.error(error instanceof Error ? error.message : 'Operation failed');
    }
  };

  const confirmDelete = (item: CatalogItem) => {
    modal.confirm({
      title: `Delete "${item.title}」？`,
      content: 'Only items that are not listed and have never been installed can be deleted. If it is already on the shelves, please remove it first.',
      okButtonProps: { danger: true },
      okText: 'Delete',
      cancelText: 'Cancel',
      onOk: () => runAction(() => deleteCatalogItem(item.id), 'Deleted'),
    });
  };

  const openVersions = async (item: CatalogItem) => {
    try {
      // 详情接口内联返回版本：页面判断「能不能上架」的依据就是有没有 published 版本。
      setVersionTarget(await getCatalogItem(item.id));
      setVersionsOpen(true);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to load version');
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
      message.success('Version created (draft). The online version number will be promoted after release');
      versionForm.resetFields();
      await refreshVersions();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to add new version');
    } finally {
      setSubmitting(false);
    }
  };

  const columns: ColumnsType<CatalogItem> = [
    {
      title: 'Product',
      dataIndex: 'title',
      render: (_: unknown, item) => (
        <Space orientation='vertical' size={0}>
          <Text strong>{item.title}</Text>
          <Text type='secondary' style={{ fontSize: 12 }}>
            {item.name} · {item.provider}
          </Text>
        </Space>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      width: 90,
      render: (type: CatalogItemType) => (
        <Tag>{TYPE_OPTIONS.find(option => option.value === type)?.label ?? type}</Tag>
      ),
    },
    {
      title: 'Status',
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
      title: 'Online version',
      dataIndex: 'latestVersion',
      width: 110,
      render: (version: string) =>
        version ? <Text code>{version}</Text> : <Text type='secondary'>Not published</Text>,
    },
    {
      title: 'Number of installations',
      dataIndex: 'installCount',
      width: 90,
    },
    {
      title: 'Updated At',
      dataIndex: 'updatedAt',
      width: 170,
      render: (value: string) => formatDateTime(value) || '—',
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 260,
      render: (_: unknown, item) => (
        <Space size={4} wrap>
          {item.status === 'published' ? (
            <Button
              type='link'
              size='small'
              icon={<ArrowDownCircle size={14} />}
              onClick={() => runAction(() => unpublishCatalogItem(item.id), 'Delisted: Stop new installations, already installed tenants will not be affected')}
            >

              Removed
            </Button>
          ) : (
            <Button
              type='link'
              size='small'
              icon={<ArrowUpCircle size={14} />}
              onClick={() => runAction(() => publishCatalogItem(item.id), 'Already on the shelves')}
            >

              On the shelves
            </Button>
          )}
          <Button type='link' size='small' icon={<History size={14} />} onClick={() => void openVersions(item)}>

            Version
          </Button>
          <Button type='link' size='small' icon={<Pencil size={14} />} onClick={() => openEdit(item)}>

            Edit
          </Button>
          {item.status !== 'published' && (
            <Button
              type='link'
              size='small'
              danger
              icon={<Trash2 size={14} />}
              onClick={() => confirmDelete(item)}
            >

              Delete
            </Button>
          )}
        </Space>
      ),
    },
  ];

  const versionColumns: ColumnsType<CatalogItemVersion> = [
    {
      title: 'Version',
      dataIndex: 'version',
      width: 100,
      render: (version: string) => <Text code>{version}</Text>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (value: CatalogItemVersion['status']) => (
        <Tag color={value === 'published' ? 'green' : value === 'draft' ? 'default' : 'orange'}>
          {value === 'published' ? 'Published' : value === 'draft' ? 'Draft' : value === 'deprecated' ? 'Deprecated' : 'Withdrawn'}
        </Tag>
      ),
    },
    { title: 'Update Log', dataIndex: 'changelog', render: (text: string) => text || '—' },
    {
      title: 'Release time',
      dataIndex: 'releasedAt',
      width: 170,
      render: (value: string) => formatDateTime(value) || '—',
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 130,
      render: (_: unknown, version) => {
        const isLive = versionTarget?.item.latestVersion === version.version;
        return (
          <Tooltip title={isLive ? 'Current online version' : 'Publishing this version will push/rollback the online version to it'}>
            <Button
              type='link'
              size='small'
              disabled={isLive || version.status === 'withdrawn'}
              onClick={() =>
                versionTarget &&
                runAction(() => publishItemVersion(versionTarget.item.id, version.version), `Published ${version.version}`)
                  .then(refreshVersions)
              }
            >
              {isLive ? 'Online' : 'Release'}
            </Button>
          </Tooltip>
        );
      },
    },
  ];

  return (
    <PageContainer
      header={{ title: 'Product Catalog' }}
      extra={
        <Space>
          <Button icon={<RefreshCw size={16} />} onClick={() => void load()} loading={loading}>

            Refresh
          </Button>
          <Button type='primary' icon={<Plus size={16} />} onClick={openCreate}>

            Create new product
          </Button>
        </Space>
      }
    >
      <Space orientation='vertical' size={16} style={{ width: '100%' }}>
        <Row gutter={16}>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title="Total number of items" value={total} prefix={<Boxes size={16} />} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title="Already on the shelves" value={summary.published} styles={{ content: { color: '#3f8600' } }} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title="Draft" value={summary.draft} styles={{ content: { color: '#d48806' } }} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title="Number of installations on this page" value={summary.installed} />
            </Card>
          </Col>
        </Row>

        <Alert
          type='info'
          showIcon
          title="The directory is global and takes effect for all tenants"
          description="After being put on the shelf, all tenants can browse and install it; taking it off the shelf will only stop new installations, and already installed tenants will not be affected. The name of the connector product must correspond to a registered connector (such as feishu / wecom / dingtalk / webhook), otherwise the running state cannot be started during installation."
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
            placeholder="Search by name/title/provider"
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
                    ? 'Directory is empty. You can create new products or confirm whether the seeder has imported factory products.'
                    : 'No products matching the filter criteria'
                }
              />
            ),
          }}
        />
      </Space>

      <Drawer
        title={editing ? `Edit:${editing.title}` : 'Create new product'}
        width={520}
        open={formOpen}
        onClose={() => setFormOpen(false)}
        extra={
          <Space>
            <Button onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button type='primary' loading={submitting} onClick={() => void submitForm()}>

              Save
            </Button>
          </Space>
        }
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label="identification (name)"
            extra={
              editing
                ? 'The name cannot be modified after creation: it determines the connector provider during installation.'
                : 'Lowercase letters/numbers/hyphens. Please use <provider>-connector for connector class, such as feishu-connector.'
            }
            rules={[{ required: true, message: 'Please enter ID' }]}
          >
            <Input disabled={Boolean(editing)} placeholder='feishu-connector' />
          </Form.Item>
          <Form.Item name='type' label="Type" rules={[{ required: true }]} extra="cannot be modified after creation">
            <Select options={TYPE_OPTIONS} disabled={Boolean(editing)} />
          </Form.Item>
          <Form.Item
            name='title'
            label="Display Name"
            rules={[{ required: true, message: 'Please enter a display name' }]}
            extra={selectedType === 'connector' && !editing ? 'Leave blank to take the title from the connector manifest' : undefined}
          >
            <Input placeholder="Feishu approves enhanced connector" />
          </Form.Item>
          <Form.Item name='provider' label="Provider" extra="For display purposes, has nothing to do with the connector provider">
            <Input placeholder='ITSM Foundation' />
          </Form.Item>
          <Form.Item name='category' label="Category">
            <Input placeholder='im / monitoring / ai' />
          </Form.Item>
          <Form.Item name='description' label="Introduction">
            <Input.TextArea rows={2} maxLength={200} showCount />
          </Form.Item>
          <Form.Item name='tags' label="Tags">
            <Select mode='tags' placeholder="Enter to add" />
          </Form.Item>
          <Form.Item name='iconUrl' label="Icon URL">
            <Input placeholder='https://…' />
          </Form.Item>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name='isOfficial' label="Official" valuePropName='checked'>
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name='isFree' label="Free" valuePropName='checked'>
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name='price' label="Price">
                <InputNumber min={0} precision={2} style={{ width: '100%' }} disabled={form.getFieldValue('isFree')} />
              </Form.Item>
            </Col>
          </Row>
          {!editing && (
            <Form.Item
              name='version'
              label="Initial version number"
              extra={
                selectedType === 'connector'
                  ? 'If the connector class is left blank, the runtime manifest version will be used; it must be consistent with the manifest when filled in.'
                  : 'Skill/plug-in type is required, format x.y.z.'
              }
            >
              <Input placeholder='1.0.0' />
            </Form.Item>
          )}
        </Form>
      </Drawer>

      <Drawer
        title={versionTarget ? `Version:${versionTarget.item.title}` : 'Version'}
        width={720}
        open={versionsOpen}
        onClose={() => setVersionsOpen(false)}
      >
        {versionTarget && (
          <Space orientation='vertical' size={16} style={{ width: '100%' }}>
            <Descriptions size='small' column={2} bordered>
              <Descriptions.Item label="Online version">
                {versionTarget.item.latestVersion || 'Not published'}
              </Descriptions.Item>
              <Descriptions.Item label="Status">
                <Tag color={CATALOG_STATUS_META[versionTarget.item.status].color}>
                  {CATALOG_STATUS_META[versionTarget.item.status].label}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            <Card size='small' title="New version">
              <Form form={versionForm} layout='inline' style={{ rowGap: 12 }}>
                <Form.Item name='version' rules={[{ required: true, message: 'Version' }]}>
                  <Input placeholder="version number, such as 1.1.0" style={{ width: 160 }} />
                </Form.Item>
                <Form.Item name='changelog'>
                  <Input placeholder="Update Log" style={{ width: 300 }} />
                </Form.Item>
                <Form.Item>
                  <Button type='primary' loading={submitting} onClick={() => void submitVersion()}>

                    New
                  </Button>
                </Form.Item>
              </Form>
              <Text type='secondary' style={{ fontSize: 12 }}>

                The new version is a draft by default, and the online version number will be promoted after it is released. The connector class version number must be consistent with the runtime manifest.
              </Text>
            </Card>

            <Table<CatalogItemVersion>
              rowKey='version'
              size='small'
              columns={versionColumns}
              dataSource={versionTarget.versions}
              pagination={false}
              locale={{ emptyText: <Empty description="There is no version yet, please add one first." /> }}
            />
          </Space>
        )}
      </Drawer>
    </PageContainer>
  );
}
