'use client';

import {
  Plus,
  Edit,
  Trash2,
  Menu as MenuIcon,
  Eye,
  EyeOff,
  Power,
  PowerOff,
  RefreshCw,
  Search,
  Link as LinkIcon,
  Hash,
} from 'lucide-react';

import React, { useEffect, useMemo, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Space,
  Typography,
  Modal,
  Form,
  Switch,
  InputNumber,
  Row,
  Col,
  Statistic,
  Tooltip,
  Popconfirm,
  Alert,
  Tag,
  App,
  AutoComplete,
  Popover,
} from 'antd';
import AppSelect from '@/components/ui/AppSelect';
import type { ColumnsType } from 'antd/es/table';
import { MenuAdminAPI, notifyMenusUpdated, type MenuItem } from '@/lib/api/menu-api';
import { RoleAPI } from '@/lib/api/role-api';
import { iconMap, getIconByName } from '@/components/layout/sidebar/icons';
import { useI18n } from '@/lib/i18n';
import { buildMenuTree, collectMenuDescendantIds } from './menuTreeUtils';

const { Title, Text } = Typography;

/**
 * 菜单管理页面
 * - 树形列表展示（按 parentId 组装、sortOrder 升序）
 * - 新增 / 编辑 / 删除
 * - 启用/可见性 切换
 * - 图标选择器 + 权限码自动补全
 */
export default function MenuManagementPage() {
  const { t } = useI18n();
  const { message: antMessage } = App.useApp();
  const [menus, setMenus] = useState<MenuItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'enabled' | 'disabled' | 'hidden'>(
    'all',
  );
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<MenuItem | null>(null);
  const [permissionOptions, setPermissionOptions] = useState<{ value: string }[]>([]);
  const [iconPickerOpen, setIconPickerOpen] = useState(false);
  const [form] = Form.useForm();
  const iconValue = Form.useWatch('icon', form);

  // 加载列表
  const loadMenus = async () => {
    setLoading(true);
    try {
      const res = await MenuAdminAPI.list();
      setMenus(res.menus || []);
    } catch (err) {
      console.error('Failed to load menus', err);
      antMessage.error(t('menus.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadMenus();
  }, []);

  // 加载权限码候选（用于 AutoComplete 提示）
  useEffect(() => {
    RoleAPI.getPermissions()
      .then(perms => setPermissionOptions((perms || []).map(p => ({ value: p }))))
      .catch(() => setPermissionOptions([]));
  }, []);

  // 过滤后的列表
  const filteredMenus = useMemo(() => {
    const q = searchText.trim().toLowerCase();
    return menus
      .filter(m => {
        if (q) {
          const hay = `${m.name} ${m.path} ${m.permissionCode ?? ''} ${m.icon ?? ''}`.toLowerCase();
          if (!hay.includes(q)) return false;
        }
        if (statusFilter === 'enabled') return m.isEnabled;
        if (statusFilter === 'disabled') return !m.isEnabled;
        if (statusFilter === 'hidden') return !m.isVisible;
        return true;
      })
      .sort((a, b) => a.sortOrder - b.sortOrder);
  }, [menus, searchText, statusFilter]);

  // 树形表格数据（被过滤掉父级的节点提升为根，保证搜索结果可见）
  const treeMenus = useMemo(() => buildMenuTree(filteredMenus), [filteredMenus]);

  // 统计
  const stats = useMemo(() => {
    return {
      total: menus.length,
      enabled: menus.filter(m => m.isEnabled).length,
      disabled: menus.filter(m => !m.isEnabled).length,
      hidden: menus.filter(m => !m.isVisible).length,
    };
  }, [menus]);

  // 打开新建
  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      sortOrder: 200,
      isVisible: true,
      isEnabled: true,
    });
    setShowModal(true);
  };

  // 打开编辑
  const openEdit = (record: MenuItem) => {
    setEditing(record);
    form.setFieldsValue({
      name: record.name,
      path: record.path,
      icon: record.icon,
      parentId: record.parentId ?? undefined,
      permissionCode: record.permissionCode ?? undefined,
      sortOrder: record.sortOrder,
      isVisible: record.isVisible,
      isEnabled: record.isEnabled,
      description: record.description,
    });
    setShowModal(true);
  };

  // 保存
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      const payload = {
        name: values.name,
        path: values.path,
        icon: values.icon || undefined,
        parentId: values.parentId ?? null,
        permissionCode: values.permissionCode || null,
        sortOrder: values.sortOrder ?? 0,
        isVisible: values.isVisible ?? true,
        isEnabled: values.isEnabled ?? true,
        description: values.description || undefined,
      };
      if (editing) {
        await MenuAdminAPI.update(editing.id, payload);
        antMessage.success(t('menus.updateSuccess'));
      } else {
        await MenuAdminAPI.create(payload);
        antMessage.success(t('menus.createSuccess'));
      }
      setShowModal(false);
      setEditing(null);
      form.resetFields();
      notifyMenusUpdated();
      loadMenus();
    } catch (err: any) {
      if (err?.errorFields) {
        // 表单校验错误
        return;
      }
      console.error('Save menu failed', err);
      antMessage.error(err?.message || t('common.saveFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 删除
  const handleDelete = async (id: number) => {
    try {
      await MenuAdminAPI.remove(id);
      antMessage.success(t('menus.deleteSuccess'));
      notifyMenusUpdated();
      loadMenus();
    } catch (err: any) {
      console.error('Delete menu failed', err);
      antMessage.error(err?.message || t('common.deleteFailed'));
    }
  };

  // 切换启用
  const toggleEnabled = async (record: MenuItem) => {
    try {
      await MenuAdminAPI.update(record.id, { isEnabled: !record.isEnabled });
      antMessage.success(record.isEnabled ? t('menus.disabled') : t('menus.enabled'));
      notifyMenusUpdated();
      loadMenus();
    } catch (err: any) {
      antMessage.error(err?.message || t('common.operationFailed'));
    }
  };

  // 切换可见性
  const toggleVisible = async (record: MenuItem) => {
    try {
      await MenuAdminAPI.update(record.id, { isVisible: !record.isVisible });
      antMessage.success(record.isVisible ? t('menus.hidden') : t('menus.visible'));
      notifyMenusUpdated();
      loadMenus();
    } catch (err: any) {
      antMessage.error(err?.message || t('common.operationFailed'));
    }
  };

  // 父菜单候选（排除自身及全部后代，避免成环）
  const parentOptions = useMemo(() => {
    if (!editing) return menus;
    const excluded = collectMenuDescendantIds(menus, editing.id);
    return menus.filter(m => !excluded.has(m.id));
  }, [menus, editing]);

  const columns: ColumnsType<MenuItem> = [
    {
      title: t('menus.sortOrder'),
      dataIndex: 'sortOrder',
      width: 80,
      sorter: (a, b) => a.sortOrder - b.sortOrder,
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: t('menus.menuName'),
      dataIndex: 'name',
      width: 160,
      render: (v: string, r) => (
        <div>
          <div className="font-medium">{v}</div>
          {r.description && (
            <div className="text-xs text-gray-500 mt-0.5">{r.description}</div>
          )}
        </div>
      ),
    },
    {
      title: t('menus.menuPath'),
      dataIndex: 'path',
      render: (v: string) => (
        <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{v || '-'}</code>
      ),
    },
    {
      title: t('menus.icon'),
      dataIndex: 'icon',
      width: 130,
      render: (v?: string) =>
        v ? (
          <Space size={4}>
            {getIconByName(v)}
            <Tag>{v}</Tag>
          </Space>
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
    {
      title: t('menus.permissionCode'),
      dataIndex: 'permissionCode',
      width: 160,
      render: (v?: string) =>
        v ? <Tag color="purple">{v}</Tag> : <span className="text-gray-400">{t('menus.none')}</span>,
      },
    {
      title: t('menus.statusColumn'),
      key: 'status',
      width: 170,
      render: (_: unknown, r) => (
        <Space size={4}>
          <Tooltip title={r.isEnabled ? t('menus.toggleDisableTooltip') : t('menus.toggleEnableTooltip')}>
            <Tag
              color={r.isEnabled ? 'green' : 'default'}
              onClick={() => toggleEnabled(r)}
              style={{ cursor: 'pointer' }}
            >
              {r.isEnabled ? t('menus.enabled') : t('menus.disabled')}
            </Tag>
          </Tooltip>
          <Tooltip title={r.isVisible ? t('menus.toggleHideTooltip') : t('menus.toggleShowTooltip')}>
            <Tag
              color={r.isVisible ? 'cyan' : 'default'}
              onClick={() => toggleVisible(r)}
              style={{ cursor: 'pointer' }}
            >
              {r.isVisible ? t('menus.visibleLabel') : t('menus.hiddenLabel')}
            </Tag>
          </Tooltip>
        </Space>
      ),
    },
    {
      title: t('menus.parentMenu'),
      dataIndex: 'parentId',
      width: 140,
      render: (v?: number | null) => {
        if (!v) return <span className="text-gray-400">-</span>;
        const parent = menus.find(m => m.id === v);
        return parent ? parent.name : <span className="text-gray-400">#{v}</span>;
      },
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 160,
      fixed: 'right',
      render: (_: unknown, record) => (
        <Space size="small">
          <Tooltip title={t('common.edit')}>
            <Button type="text" icon={<Edit className="w-4 h-4" />} onClick={() => openEdit(record)} />
          </Tooltip>
          <Popconfirm
            title={t('common.confirmDelete')}
            description={t('menus.deleteWarning', { name: '删除后不可恢复，关联的子菜单会变成根菜单。' })}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.delete')}
            cancelText={t('common.cancel')}
            okButtonProps={{ danger: true }}
          >
            <Tooltip title={t('common.delete')}>
              <Button type="text" danger icon={<Trash2 className="w-4 h-4" />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2">
          <MenuIcon className="inline-block w-6 h-6 mr-2" />
          {t('menus.title')}
        </Title>
        <Text type="secondary">{t('menus.description')}</Text>
      </div>

      <Alert
        type="info"
        showIcon
        message={t('menus.usageTip')}
        description={
          <div>
            <div>• {t('menus.usageTipLine1')}</div>
            <div>• {t('menus.usageTipLine2')}</div>
            <div>• {t('menus.usageTipLine3')}</div>
          </div>
        }
      />

      <Row gutter={[16, 16]}>
        <Col xs={12} md={6}>
          <Card>
            <Statistic
              title={t('menus.totalMenus')}
              value={stats.total}
              prefix={<Hash className="w-5 h-5" />}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic
              title={t('menus.enabledMenus')}
              value={stats.enabled}
              prefix={<Power className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic
              title={t('menus.disabledMenus')}
              value={stats.disabled}
              prefix={<PowerOff className="w-5 h-5" />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic
              title={t('menus.hiddenMenus')}
              value={stats.hidden}
              prefix={<EyeOff className="w-5 h-5" />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
      </Row>

      <Card className="mb-4">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={10} lg={8}>
            <Input
              placeholder={t('menus.searchPlaceholder')}
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchText}
              onChange={e => setSearchText(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={8} lg={6}>
            <AppSelect
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { value: 'all', label: t('menus.allStatus') },
                { value: 'visible', label: t('menus.visibleStatus') },
                { value: 'hidden', label: t('menus.hiddenStatus') },
                { value: 'enabled', label: t('menus.enabledStatus') },
                { value: 'disabled', label: t('menus.disabledStatus') },
              ]}
            />
          </Col>
          <Col xs={24} md={6} lg={10} className="text-right">
            <Space>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={loadMenus} loading={loading}>
                {t('menus.refresh')}
              </Button>
              <Button type="primary" icon={<Plus className="w-4 h-4" />} onClick={openCreate}>
                {t('menus.createMenu')}
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      <Card>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={treeMenus}
          expandable={{ indentSize: 20 }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => t('menus.totalLabel', { total }),
            pageSize: 20,
          }}
          scroll={{ x: 1100 }}
        />
      </Card>

      <Modal
        title={
          <span>
            {editing ? <Edit className="w-4 h-4 mr-2 inline-block" /> : <Plus className="w-4 h-4 mr-2 inline-block" />}
            {editing ? t('menus.editMenu') : t('menus.createMenu')}
          </span>
        }
        open={showModal}
        onOk={handleSave}
        onCancel={() => {
          setShowModal(false);
          setEditing(null);
          form.resetFields();
        }}
        confirmLoading={loading}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        width={680}
      >
        <Form form={form} layout="vertical" className="mt-4" preserve={false}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label={t('menus.menuName')}
                name="name"
                rules={[{ required: true, message: t('common.required') }]}
              >
                <Input placeholder={t('menus.menuNamePlaceholder')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label={t('menus.menuPath')}
                name="path"
                rules={[
                  { required: true, message: t('menus.pathRequired', { name: '请输入路由路径' }) },
                  {
                    pattern: /^\/[\w\-./]*$/,
                    message: t('menus.pathFormatError', { name: '路径必须以 / 开头，仅支持字母、数字、- _ . /' }),
                  },
                ]}
                tooltip={t('menus.pathTooltip', { name: '前端路由地址，例如 /admin/sla-templates' })}
              >
                <Input
                  prefix={<LinkIcon className="w-4 h-4 text-gray-400" />}
                  placeholder={t('menus.pathPlaceholder', { name: '/admin/sla-templates' })}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label={t('menus.icon')}
                name="icon"
                tooltip={t('menus.iconTooltip', { name: 'Lucide React 图标名，可手输或从列表选择' })}
              >
                <Input
                  placeholder={t('menus.iconPlaceholder')}
                  addonBefore={getIconByName(iconValue) ?? <MenuIcon className="w-4 h-4 text-gray-300" />}
                  addonAfter={
                    <Popover
                      title={t('menus.selectIcon')}
                      trigger="click"
                      open={iconPickerOpen}
                      onOpenChange={setIconPickerOpen}
                      content={
                        <div
                          style={{
                            width: 320,
                            maxHeight: 260,
                            overflowY: 'auto',
                            display: 'grid',
                            gridTemplateColumns: 'repeat(8, 1fr)',
                            gap: 4,
                          }}
                        >
                          {Object.keys(iconMap).map(name => (
                            <Tooltip title={name} key={name}>
                              <Button
                                type={iconValue === name ? 'primary' : 'text'}
                                size="small"
                                onClick={() => {
                                  form.setFieldsValue({ icon: name });
                                  setIconPickerOpen(false);
                                }}
                              >
                                {iconMap[name]}
                              </Button>
                            </Tooltip>
                          ))}
                        </div>
                      }
                    >
                      <span style={{ cursor: 'pointer' }}>{t('menus.selectLabel')}</span>
                    </Popover>
                  }
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label={t('menus.sortOrder')}
                name="sortOrder"
                rules={[{ required: true, message: t('menus.sortOrderRequired', { name: '请输入排序号' }) }]}
              >
                <InputNumber min={0} max={9999} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label={t('menus.permissionCode')}
                name="permissionCode"
                tooltip={t('menus.permissionCodeTooltip', { name: '菜单关联的权限码，留空则对所有登录用户可见' })}
              >
                <AutoComplete
                  options={permissionOptions}
                  allowClear
                  placeholder={t('menus.permissionCodePlaceholder')}
                  filterOption={(input, option) =>
                    String(option?.value ?? '')
                      .toLowerCase()
                      .includes(input.toLowerCase())
                  }
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label={t('menus.parentMenu')} name="parentId" tooltip={t('menus.parentMenuTooltip', { name: '二级菜单需指定父菜单' })}>
                <AppSelect allowClear placeholder={t('menus.topLevelMenu')}
                options={parentOptions.map(p => ({ value: p.id, label: p.name }))}
              />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label={t('common.description')} name="description">
            <Input.TextArea rows={2} maxLength={200} showCount placeholder={t('menus.descriptionPlaceholder')} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label={t('menus.isEnabled')} name="isEnabled" valuePropName="checked">
                <Switch checkedChildren={t('menus.enabled')} unCheckedChildren={t('menus.disabled')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label={t('menus.isVisible')} name="isVisible" valuePropName="checked">
                <Switch
                  checkedChildren={<><Eye className="w-3 h-3 mr-1 inline-block" />{t('menus.showLabel')}</>}
                  unCheckedChildren={<><EyeOff className="w-3 h-3 mr-1 inline-block" />{t('menus.hideLabel')}</>}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
}
