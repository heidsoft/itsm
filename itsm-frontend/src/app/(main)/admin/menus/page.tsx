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
  Select,
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
import type { ColumnsType } from 'antd/es/table';
import { MenuAdminAPI, notifyMenusUpdated, type MenuItem } from '@/lib/api/menu-api';
import { RoleAPI } from '@/lib/api/role-api';
import { iconMap, getIconByName } from '@/components/layout/sidebar/icons';
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
      antMessage.error('加载菜单列表失败');
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
        antMessage.success('菜单更新成功');
      } else {
        await MenuAdminAPI.create(payload);
        antMessage.success('菜单创建成功');
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
      antMessage.error(err?.message || '保存菜单失败');
    } finally {
      setLoading(false);
    }
  };

  // 删除
  const handleDelete = async (id: number) => {
    try {
      await MenuAdminAPI.remove(id);
      antMessage.success('菜单已删除');
      notifyMenusUpdated();
      loadMenus();
    } catch (err: any) {
      console.error('Delete menu failed', err);
      antMessage.error(err?.message || '删除失败');
    }
  };

  // 切换启用
  const toggleEnabled = async (record: MenuItem) => {
    try {
      await MenuAdminAPI.update(record.id, { isEnabled: !record.isEnabled });
      antMessage.success(record.isEnabled ? '已禁用' : '已启用');
      notifyMenusUpdated();
      loadMenus();
    } catch (err: any) {
      antMessage.error(err?.message || '操作失败');
    }
  };

  // 切换可见性
  const toggleVisible = async (record: MenuItem) => {
    try {
      await MenuAdminAPI.update(record.id, { isVisible: !record.isVisible });
      antMessage.success(record.isVisible ? '已隐藏' : '已显示');
      notifyMenusUpdated();
      loadMenus();
    } catch (err: any) {
      antMessage.error(err?.message || '操作失败');
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
      title: '排序',
      dataIndex: 'sortOrder',
      width: 80,
      sorter: (a, b) => a.sortOrder - b.sortOrder,
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '名称',
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
      title: '路径',
      dataIndex: 'path',
      render: (v: string) => (
        <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{v || '-'}</code>
      ),
    },
    {
      title: '图标',
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
      title: '权限码',
      dataIndex: 'permissionCode',
      width: 160,
      render: (v?: string) =>
        v ? <Tag color="purple">{v}</Tag> : <span className="text-gray-400">无</span>,
    },
    {
      title: '状态',
      key: 'status',
      width: 170,
      render: (_: unknown, r) => (
        <Space size={4}>
          <Tooltip title={r.isEnabled ? '点击禁用' : '点击启用'}>
            <Tag
              color={r.isEnabled ? 'green' : 'default'}
              onClick={() => toggleEnabled(r)}
              style={{ cursor: 'pointer' }}
            >
              {r.isEnabled ? '已启用' : '已禁用'}
            </Tag>
          </Tooltip>
          <Tooltip title={r.isVisible ? '点击隐藏' : '点击显示'}>
            <Tag
              color={r.isVisible ? 'cyan' : 'default'}
              onClick={() => toggleVisible(r)}
              style={{ cursor: 'pointer' }}
            >
              {r.isVisible ? '可见' : '隐藏'}
            </Tag>
          </Tooltip>
        </Space>
      ),
    },
    {
      title: '父菜单',
      dataIndex: 'parentId',
      width: 140,
      render: (v?: number | null) => {
        if (!v) return <span className="text-gray-400">-</span>;
        const parent = menus.find(m => m.id === v);
        return parent ? parent.name : <span className="text-gray-400">#{v}</span>;
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 160,
      fixed: 'right',
      render: (_: unknown, record) => (
        <Space size="small">
          <Tooltip title="编辑">
            <Button type="text" icon={<Edit className="w-4 h-4" />} onClick={() => openEdit(record)} />
          </Tooltip>
          <Popconfirm
            title="确认删除"
            description="删除后不可恢复，关联的子菜单会变成根菜单。"
            onConfirm={() => handleDelete(record.id)}
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
          >
            <Tooltip title="删除">
              <Button type="text" danger icon={<Trash2 className="w-4 h-4" />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <MenuIcon className="inline-block w-6 h-6 mr-2" />
          菜单管理
        </Title>
        <Text type="secondary">
          管理侧边栏菜单：添加、修改、删除、启用/禁用、显示/隐藏。tenant_id 自动从登录态注入。
        </Text>
      </div>

      <Alert
        className="mb-4"
        type="info"
        showIcon
        message="提示"
        description={
          <div>
            <div>• 权限码必须与 <code>permissions</code> 表中已存在的权限代码一致，菜单才会按角色过滤。</div>
            <div>• sortOrder 越小越靠前；建议按 10/20/30… 或 100/110/120… 留出插入空间。</div>
            <div>• 隐藏(isVisible=false)仍占位；禁用(isEnabled=false)会被完全过滤。</div>
            <div>• 默认菜单请通过后端种子数据（seed）初始化，本页面不提供一键初始化。</div>
          </div>
        }
      />

      <Row gutter={[16, 16]} className="mb-4">
        <Col xs={12} md={6}>
          <Card>
            <Statistic
              title="总菜单数"
              value={stats.total}
              prefix={<Hash className="w-5 h-5" />}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic
              title="已启用"
              value={stats.enabled}
              prefix={<Power className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic
              title="已禁用"
              value={stats.disabled}
              prefix={<PowerOff className="w-5 h-5" />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic
              title="已隐藏"
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
              placeholder="搜索 名称 / 路径 / 权限码 / 图标"
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchText}
              onChange={e => setSearchText(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={8} lg={6}>
            <Select
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: '全部状态' },
                { value: 'enabled', label: '已启用' },
                { value: 'disabled', label: '已禁用' },
                { value: 'hidden', label: '已隐藏' },
              ]}
            />
          </Col>
          <Col xs={24} md={6} lg={10} className="text-right">
            <Space>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={loadMenus} loading={loading}>
                刷新
              </Button>
              <Button type="primary" icon={<Plus className="w-4 h-4" />} onClick={openCreate}>
                新建菜单
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
            showTotal: t => `共 ${t} 条`,
            pageSize: 20,
          }}
          scroll={{ x: 1100 }}
        />
      </Card>

      <Modal
        title={
          <span>
            {editing ? <Edit className="w-4 h-4 mr-2 inline-block" /> : <Plus className="w-4 h-4 mr-2 inline-block" />}
            {editing ? '编辑菜单' : '新建菜单'}
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
        okText="保存"
        cancelText="取消"
        width={680}
      >
        <Form form={form} layout="vertical" className="mt-4" preserve={false}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="菜单名称"
                name="name"
                rules={[{ required: true, message: '请输入菜单名称' }]}
              >
                <Input placeholder="如：SLA 模板" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="路径"
                name="path"
                rules={[
                  { required: true, message: '请输入路由路径' },
                  {
                    pattern: /^\/[\w\-./]*$/,
                    message: '路径必须以 / 开头，仅支持字母、数字、- _ . /',
                  },
                ]}
                tooltip="前端路由地址，例如 /admin/sla-templates"
              >
                <Input
                  prefix={<LinkIcon className="w-4 h-4 text-gray-400" />}
                  placeholder="/admin/sla-templates"
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="图标"
                name="icon"
                tooltip="Lucide React 图标名，可手输或从列表选择"
              >
                <Input
                  placeholder="如：Layers / BarChart3"
                  addonBefore={getIconByName(iconValue) ?? <MenuIcon className="w-4 h-4 text-gray-300" />}
                  addonAfter={
                    <Popover
                      title="选择图标"
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
                      <span style={{ cursor: 'pointer' }}>选择</span>
                    </Popover>
                  }
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="排序"
                name="sortOrder"
                rules={[{ required: true, message: '请输入排序号' }]}
              >
                <InputNumber min={0} max={9999} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="权限码"
                name="permissionCode"
                tooltip="菜单关联的权限码，留空则对所有登录用户可见"
              >
                <AutoComplete
                  options={permissionOptions}
                  allowClear
                  placeholder="如：sla:write（支持从已有权限码中选择）"
                  filterOption={(input, option) =>
                    String(option?.value ?? '')
                      .toLowerCase()
                      .includes(input.toLowerCase())
                  }
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="父菜单" name="parentId" tooltip="二级菜单需指定父菜单">
                <Select allowClear placeholder="无（顶级菜单）" showSearch optionFilterProp="label"
                options={parentOptions.map(p => ({ value: p.id, label: p.name }))}
              />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} maxLength={200} showCount placeholder="可选：菜单用途说明" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="启用" name="isEnabled" valuePropName="checked">
                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="可见" name="isVisible" valuePropName="checked">
                <Switch
                  checkedChildren={<><Eye className="w-3 h-3 mr-1 inline-block" />显示</>}
                  unCheckedChildren={<><EyeOff className="w-3 h-3 mr-1 inline-block" />隐藏</>}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
}
