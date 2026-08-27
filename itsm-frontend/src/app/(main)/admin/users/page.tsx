'use client';

import {
  Plus,
  Edit,
  Trash2,
  MoreHorizontal,
  Users,
  UserCheck,
  Download,
  Key,
  UserX,
} from 'lucide-react';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Modal,
  Form,
  Space,
  Dropdown,
  Row,
  Col,
  Statistic,
  Typography,
  theme,
  Switch,
  App,
  Tag,
  Empty,
} from 'antd';
import { UserApi, type User } from '@/lib/api/user-api';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Text } = Typography;
const { Search: AntSearch } = Input;

const UserManagement: React.FC = () => {
  const { token } = theme.useToken();
  const { message } = App.useApp();
  const { t } = useI18n();
  const { currentTenant } = useAuthStore();
  useAuthStoreHydration();

  // 状态管理
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState({ total: 0, active: 0, inactive: 0 });
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  // 筛选和搜索
  const [filters, setFilters] = useState({
    status: '',
    department: '',
    search: '',
  });
  const [departments, setDepartments] = useState<string[]>([]);

  // 模态框状态
  const [isCreateModalVisible, setIsCreateModalVisible] = useState(false);
  const [isEditModalVisible, setIsEditModalVisible] = useState(false);
  const [isPasswordModalVisible, setIsPasswordModalVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);

  // 表单
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [passwordForm] = Form.useForm();

  // 加载用户列表
  const loadUsers = async () => {
    setLoading(true);
    try {
      const params = {
        page: pagination.current,
        pageSize: pagination.pageSize,
        status: filters.status || undefined,
        department: filters.department || undefined,
        search: filters.search || undefined,
      };
      const response = await UserApi.getUsers(params);
      setUsers(response.users);
      setPagination(prev => ({ ...prev, total: response.pagination.total }));

      // 从用户数据派生部门列表（去重+排序）
      const depts = Array.from(new Set(response.users.map((u: User) => u.department).filter(Boolean) as string[])).sort();
      setDepartments(depts);

      const userStats = await UserApi.getUserStats(currentTenant?.id);
      setStats(userStats);
    } catch (error) {
      message.error(t('users.messages.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadUsers();
  }, [pagination.current, pagination.pageSize, filters]);

  // 创建用户
  const handleCreateUser = async (values: any) => {
    setLoading(true);
    try {
      const tenantId = currentTenant?.id;
      if (!tenantId) {
        message.error(t('users.messages.noTenant'));
        return;
      }
      await UserApi.createUser({
        username: values.username,
        email: values.email,
        name: values.name,
        department: values.department,
        phone: values.phone,
        password: values.password,
        tenantId: tenantId,
      });
      message.success(t('users.messages.createSuccess'));
      setIsCreateModalVisible(false);
      createForm.resetFields();
      loadUsers();
    } catch (error) {
      message.error(t('users.messages.createFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 更新用户
  const handleUpdateUser = async (values: any) => {
    if (!selectedUser) return;
    setLoading(true);
    try {
      await UserApi.updateUser(selectedUser.id, {
        username: values.username,
        email: values.email,
        name: values.name,
        department: values.department,
        phone: values.phone,
      });
      message.success(t('users.messages.updateSuccess'));
      setIsEditModalVisible(false);
      editForm.resetFields();
      setSelectedUser(null);
      loadUsers();
    } catch (error) {
      message.error(t('users.messages.updateFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 删除用户
  const handleDeleteUser = async (userId: number) => {
    setLoading(true);
    try {
      await UserApi.deleteUser(userId);
      message.success(t('users.messages.deleteSuccess'));
      loadUsers();
    } catch (error) {
      message.error(t('users.messages.deleteFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 切换用户状态
  const handleToggleUserStatus = async (userId: number, currentStatus: boolean) => {
    setLoading(true);
    try {
      const newStatus = !currentStatus;
      await UserApi.changeUserStatus(userId, newStatus);
      setUsers(prev =>
        prev.map(user => (user.id === userId ? { ...user, active: newStatus } : user))
      );
      message.success(newStatus ? t('users.messages.activated') : t('users.messages.deactivated'));
      loadUsers();
    } catch (error) {
      console.error('Failed to toggle user status:', error);
      message.error(t('users.messages.statusUpdateFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 重置密码
  const handleResetPassword = async (values: { newPassword: string }) => {
    if (!selectedUser) return;
    setLoading(true);
    try {
      await UserApi.resetPassword(selectedUser.id, values.newPassword);
      message.success(t('users.messages.passwordResetSuccess'));
      setIsPasswordModalVisible(false);
      passwordForm.resetFields();
      setSelectedUser(null);
    } catch (error) {
      message.error(t('users.messages.passwordResetFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 搜索和筛选逻辑
  const handleSearch = (value: string) => {
    setFilters(prev => ({ ...prev, search: value }));
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  const handleFilterChange = (key: string, value: string) => {
    setFilters(prev => ({ ...prev, [key]: value }));
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  // 表格列定义
  const columns = [
    {
      title: t('users.columns.username'),
      dataIndex: 'username',
      key: 'username',
      render: (text: string, record: User) => (
        <Space>
          <Text strong>{text}</Text>
          {!record.active && <Tag color="red">{t('users.statusTag.deactivated')}</Tag>}
        </Space>
      ),
    },
    {
      title: t('users.columns.name'),
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: t('users.columns.email'),
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: t('users.columns.department'),
      dataIndex: 'department',
      key: 'department',
    },
    {
      title: t('users.columns.phone'),
      dataIndex: 'phone',
      key: 'phone',
    },
    {
      title: t('users.columns.status'),
      dataIndex: 'active',
      key: 'active',
      render: (active: boolean, record: User) => (
        <Switch
          checked={active}
          loading={loading}
          onChange={() => handleToggleUserStatus(record.id, active)}
          checkedChildren={t('users.statusTag.active')}
          unCheckedChildren={t('users.statusTag.deactivated')}
        />
      ),
    },
    {
      title: t('users.columns.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (text: string) => (text ? new Date(text).toLocaleString('zh-CN') : '-'),
    },
    {
      title: t('common.action'),
      key: 'actions',
      render: (_: unknown, record: User) => (
        <Dropdown
          menu={{
            items: [
              {
                key: 'edit',
                label: t('common.edit'),
                icon: <Edit size={16} />,
                onClick: () => {
                  setSelectedUser(record);
                  editForm.setFieldsValue(record);
                  setIsEditModalVisible(true);
                },
              },
              {
                key: 'password',
                label: t('users.actions.resetPassword'),
                icon: <Key size={16} />,
                onClick: () => {
                  setSelectedUser(record);
                  setIsPasswordModalVisible(true);
                },
              },
              {
                type: 'divider',
              },
              {
                key: 'delete',
                label: t('common.delete'),
                icon: <Trash2 size={16} />,
                danger: true,
                onClick: () => {
                  Modal.confirm({
                    title: t('common.confirmDelete'),
                    content: t('users.confirmDelete', { name: record.name }),
                    onOk: () => handleDeleteUser(record.id),
                  });
                },
              },
            ],
          }}
        >
          <Button type="text" icon={<MoreHorizontal size={16} />} aria-label={t('common.actions')} />
        </Dropdown>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题和统计 */}
      <div>
        <Title level={2} style={{ margin: 0, marginBottom: token.marginXS }}>
          <Space>
            <Users style={{ color: token.colorPrimary }} />
{t('users.title')}
          </Space>
        </Title>
        <Text type="secondary">{t('users.description')}</Text>

        {/* 统计卡片 */}
        <Row gutter={16} style={{ marginTop: token.marginLG }}>
          <Col span={8}>
            <Card>
              <Statistic
                title={t('users.stats.total')}
                value={stats.total}
                prefix={<Users style={{ color: token.colorPrimary }} />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title={t('users.stats.active')}
                value={stats.active}
                prefix={<UserCheck style={{ color: '#52c41a' }} />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title={t('users.stats.inactive')}
                value={stats.inactive}
                prefix={<UserX style={{ color: '#ff4d4f' }} />}
              />
            </Card>
          </Col>
        </Row>
      </div>

      {/* 操作栏 */}
      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col flex="auto">
            <Space wrap>
              <AntSearch
                placeholder={t('users.searchPlaceholder')}
                style={{ width: 280 }}
                onSearch={handleSearch}
                allowClear
              />
              <Select
                placeholder={t('users.filter.status')}
                style={{ width: 120 }}
                allowClear
                onChange={value => handleFilterChange('status', value || '')}
                options={[
                  { value: 'active', label: t('users.statusTag.active') },
                  { value: 'inactive', label: t('users.statusTag.deactivated') },
                ]}
              />
              <Select
                placeholder={t('users.filter.department')}
                style={{ width: 160 }}
                allowClear
                showSearch
                onChange={value => handleFilterChange('department', value || '')}
                options={departments.map(d => ({ value: d, label: d }))}
              />
            </Space>
          </Col>
          <Col>
            <Space>
              <Button
                type="primary"
                icon={<Plus size={16} />}
                onClick={() => setIsCreateModalVisible(true)}
              >
{t('users.createUser')}
              </Button>
              <Button
                icon={<Download size={16} />}
                onClick={() => {
                  // 导出用户数据
                  const exportData = users.map(user => ({
                    用户名: user.username,
                    姓名: user.name,
                    邮箱: user.email,
                    部门: user.department || '',
                    电话: user.phone || '',
                    状态: user.active ? '激活' : '禁用',
                    创建时间: user.createdAt,
                  }));
                  const headers = ['用户名', '姓名', '邮箱', '部门', '电话', '状态', '创建时间'];
                  const csvContent = [
                    headers.join(','),
                    ...exportData.map(row => headers.map(header => row[header as keyof typeof row]).join(',')),
                  ].join('\n');
                  const blob = new Blob(['\ufeff' + csvContent], {
                    type: 'text/csv;charset=utf-8;',
                  });
                  const url = URL.createObjectURL(blob);
                  const link = document.createElement('a');
                  link.href = url;
                  link.download = `{t('users.exportFilename')}_${new Date().toISOString().split('T')[0]}.csv`;
                  link.click();
                  URL.revokeObjectURL(url);
                  message.success(t('users.messages.exportSuccess'));
                }}
              >
{t('common.export')}
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 用户表格 */}
      <Card>
        {users.length === 0 && !loading ? (
          <Empty description={t('users.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Button type="primary" onClick={() => setIsCreateModalVisible(true)}>
{t('users.createFirst')}
            </Button>
          </Empty>
        ) : (
          <Table
            columns={columns}
            dataSource={users}
            rowKey="id"
            loading={loading}
            scroll={{ x: 980 }}
            pagination={{
              current: pagination.current,
              pageSize: pagination.pageSize,
              total: pagination.total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => t('common.totalLabel', { total: total }),
              pageSizeOptions: ['10', '20', '50', '100'],
              onChange: (page, pageSize) => {
                setPagination(prev => ({ ...prev, current: page, pageSize }));
              },
            }}
          />
        )}
      </Card>

      {/* 创建用户模态框 */}
      <Modal
        title={t('users.createUser')}
        open={isCreateModalVisible}
        onCancel={() => {
          setIsCreateModalVisible(false);
          createForm.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreateUser}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="username"
                label={t('users.columns.username')}
                rules={[
                  { required: true, message: t('users.form.requiredUsername') },
                  { min: 3, message: t('users.form.minUsername') },
                ]}
              >
                <Input placeholder={t('users.form.usernamePlaceholder')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="name"
                label={t('users.columns.name')}
                rules={[{ required: true, message: t('users.form.requiredName') }]}
              >
                <Input placeholder={t('users.form.namePlaceholder')} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="email"
                label={t('users.columns.email')}
                rules={[
                  { required: true, message: t('users.form.requiredEmail') },
                  { type: 'email', message: t('users.form.invalidEmail') },
                ]}
              >
                <Input placeholder={t('users.form.emailPlaceholder')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="phone" label={t('users.columns.phone')}>
                <Input placeholder={t('users.form.phonePlaceholder')} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="department" label={t('users.columns.department')}>
                <Select placeholder={t('users.form.departmentPlaceholder')} options={[{ value: 'IT部门', label: 'IT部门' }, { value: '财务部门', label: '财务部门' }, { value: '人事部门', label: '人事部门' }, { value: '市场部门', label: '市场部门' }]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="password"
                label={t('users.form.password')}
                rules={[
                  { required: true, message: t('users.form.requiredPassword') },
                  { min: 6, message: t('users.form.minPassword') },
                ]}
              >
                <Input.Password placeholder={t('users.form.passwordPlaceholder')} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
  {t('users.createUser')}
              </Button>
              <Button
                onClick={() => {
                  setIsCreateModalVisible(false);
                  createForm.resetFields();
                }}
              >
{t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑用户模态框 */}
      <Modal
        title={t('users.editUser')}
        open={isEditModalVisible}
        onCancel={() => {
          setIsEditModalVisible(false);
          editForm.resetFields();
          setSelectedUser(null);
        }}
        footer={null}
        width={600}
      >
        <Form form={editForm} layout="vertical" onFinish={handleUpdateUser}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="username"
                label="用户名"
                rules={[{ min: 3, message: '用户名至少3个字符' }]}
              >
                <Input placeholder="请输入用户名" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="name" label="姓名">
                <Input placeholder="请输入姓名" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="email"
                label="邮箱"
                rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
              >
                <Input placeholder="请输入邮箱" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="phone" label="电话">
                <Input placeholder="请输入电话号码" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="department" label="部门">
            <Select placeholder="请选择部门" options={[
              { value: 'IT部门', label: 'IT部门' },
              { value: '财务部门', label: '财务部门' },
              { value: '人事部门', label: '人事部门' },
              { value: '市场部门', label: '市场部门' },
            ]} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
{t('users.form.save')}
              </Button>
              <Button
                onClick={() => {
                  setIsEditModalVisible(false);
                  editForm.resetFields();
                  setSelectedUser(null);
                }}
              >
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 重置密码模态框 */}
      <Modal
        title={t('users.actions.resetPassword')}
        open={isPasswordModalVisible}
        onCancel={() => {
          setIsPasswordModalVisible(false);
          passwordForm.resetFields();
          setSelectedUser(null);
        }}
        footer={null}
      >
        <Form form={passwordForm} layout="vertical" onFinish={handleResetPassword}>
          <Form.Item
            name="newPassword"
            label={t('users.form.newPassword')}
            rules={[
              { required: true, message: t('users.form.requiredNewPassword') },
              { min: 6, message: '密码至少6个字符' },
            ]}
          >
            <Input.Password placeholder={t('users.form.newPasswordPlaceholder')} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
{t('users.actions.resetPassword')}
              </Button>
              <Button
                onClick={() => {
                  setIsPasswordModalVisible(false);
                  passwordForm.resetFields();
                  setSelectedUser(null);
                }}
              >
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default UserManagement;
