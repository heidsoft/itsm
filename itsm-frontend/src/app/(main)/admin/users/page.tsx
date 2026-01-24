'use client';

import {
  Plus,
  Search,
  Filter,
  Edit,
  Trash2,
  MoreHorizontal,
  Users,
  UserCheck,
  Download,
  Upload,
  Eye,
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
} from 'antd';
import { UserApi, type User } from '@/lib/api/user-api';

const { Title, Text } = Typography;
const { Search: AntSearch } = Input;
const { Option } = Select;

const UserManagement: React.FC = () => {
  const { token } = theme.useToken();
  const { message } = App.useApp();

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

  // 模态框状态
  const [isCreateModalVisible, setIsCreateModalVisible] = useState(false);
  const [isEditModalVisible, setIsEditModalVisible] = useState(false);
  const [isPasswordModalVisible, setIsPasswordModalVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);

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
        page_size: pagination.pageSize,
        status: filters.status || undefined,
        department: filters.department || undefined,
        search: filters.search || undefined,
      };
      const response = await UserApi.getUsers(params);
      setUsers(response.users);
      setPagination(prev => ({ ...prev, total: response.pagination.total }));
      setStats({
        total: response.pagination.total,
        active: response.users.filter(u => u.active).length,
        inactive: response.users.filter(u => !u.active).length,
      });
    } catch (error) {
      message.error('加载用户列表失败');
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
      await UserApi.createUser({
        username: values.username,
        email: values.email,
        name: values.name,
        department: values.department,
        phone: values.phone,
        password: values.password,
        tenant_id: 1,
      });
      message.success('用户创建成功');
      setIsCreateModalVisible(false);
      createForm.resetFields();
      loadUsers();
    } catch (error) {
      message.error('创建用户失败');
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
      message.success('用户更新成功');
      setIsEditModalVisible(false);
      editForm.resetFields();
      setSelectedUser(null);
      loadUsers();
    } catch (error) {
      message.error('更新用户失败');
    } finally {
      setLoading(false);
    }
  };

  // 删除用户
  const handleDeleteUser = async (userId: number) => {
    setLoading(true);
    try {
      await UserApi.deleteUser(userId);
      message.success('用户删除成功');
      loadUsers();
    } catch (error) {
      message.error('删除用户失败');
    } finally {
      setLoading(false);
    }
  };

  // 切换用户状态
  const handleToggleUserStatus = async (userId: number, currentStatus: boolean) => {
    setLoading(true);
    try {
      await UserApi.updateUser(userId, { ...{} });
      setUsers(prev => prev.map(user =>
        user.id === userId
          ? { ...user, active: !currentStatus }
          : user
      ));
      message.success('状态更新成功');
      loadUsers();
    } catch (error) {
      message.error('状态更新失败');
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
      message.success('密码重置成功');
      setIsPasswordModalVisible(false);
      passwordForm.resetFields();
      setSelectedUser(null);
    } catch (error) {
      message.error('密码重置失败');
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
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (text: string, record: User) => (
        <Space>
          <Text strong>{text}</Text>
          {!record.active && <Tag color='red'>已禁用</Tag>}
        </Space>
      ),
    },
    {
      title: '姓名',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '部门',
      dataIndex: 'department',
      key: 'department',
    },
    {
      title: '电话',
      dataIndex: 'phone',
      key: 'phone',
    },
    {
      title: '状态',
      dataIndex: 'active',
      key: 'active',
      render: (active: boolean, record: User) => (
        <Switch
          checked={active}
          loading={loading}
          onChange={() => handleToggleUserStatus(record.id, active)}
          checkedChildren='激活'
          unCheckedChildren='禁用'
        />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text: string) => new Date(text).toLocaleString(),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: unknown, record: User) => (
        <Dropdown
          menu={{
            items: [
              {
                key: 'edit',
                label: '编辑',
                icon: <Edit size={16} />,
                onClick: () => {
                  setSelectedUser(record);
                  editForm.setFieldsValue(record);
                  setIsEditModalVisible(true);
                },
              },
              {
                key: 'password',
                label: '重置密码',
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
                label: '删除',
                icon: <Trash2 size={16} />,
                danger: true,
                onClick: () => {
                  Modal.confirm({
                    title: '确认删除',
                    content: `确定要删除用户 ${record.name} 吗？`,
                    onOk: () => handleDeleteUser(record.id),
                  });
                },
              },
            ],
          }}
        >
          <Button type='text' icon={<MoreHorizontal size={16} />} />
        </Dropdown>
      ),
    },
  ];

  return (
    <div style={{ padding: token.paddingLG }}>
      {/* 页面标题和统计 */}
      <div style={{ marginBottom: token.marginLG }}>
        <Title level={2} style={{ margin: 0, marginBottom: token.marginXS }}>
          <Space>
            <Users style={{ color: token.colorPrimary }} />
            用户管理
          </Space>
        </Title>
        <Text type='secondary'>管理系统用户账户、权限和状态</Text>

        {/* 统计卡片 */}
        <Row gutter={16} style={{ marginTop: token.marginLG }}>
          <Col span={8}>
            <Card>
              <Statistic
                title='总用户数'
                value={stats.total}
                prefix={<Users style={{ color: token.colorPrimary }} />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title='活跃用户'
                value={stats.active}
                prefix={<UserCheck style={{ color: '#52c41a' }} />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title='禁用用户'
                value={stats.inactive}
                prefix={<UserX style={{ color: '#ff4d4f' }} />}
              />
            </Card>
          </Col>
        </Row>
      </div>

      {/* 操作栏 */}
      <Card style={{ marginBottom: token.marginLG }}>
        <Row gutter={[16, 16]} align='middle'>
          <Col flex='auto'>
            <Space wrap>
              <AntSearch
                placeholder='搜索用户名、姓名、邮箱'
                style={{ width: 280 }}
                onSearch={value => setFilters(prev => ({ ...prev, search: value }))}
                allowClear
              />
              <Select
                placeholder='状态筛选'
                style={{ width: 120 }}
                allowClear
                onChange={value => setFilters(prev => ({ ...prev, status: value || '' }))}
              >
                <Option value='active'>激活</Option>
                <Option value='inactive'>禁用</Option>
              </Select>
              <Select
                placeholder='部门筛选'
                style={{ width: 160 }}
                allowClear
                onChange={value => setFilters(prev => ({ ...prev, department: value || '' }))}
              >
                <Option value='IT部门'>IT部门</Option>
                <Option value='财务部门'>财务部门</Option>
                <Option value='人事部门'>人事部门</Option>
                <Option value='市场部门'>市场部门</Option>
              </Select>
            </Space>
          </Col>
          <Col>
            <Space>
              <Button
                type='primary'
                icon={<Plus size={16} />}
                onClick={() => setIsCreateModalVisible(true)}
              >
                新建用户
              </Button>
              <Button icon={<Download size={16} />} onClick={() => message.info('导出功能开发中')}>
                导出
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 用户表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={users}
          rowKey='id'
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条/共 ${total} 条`,
            onChange: (page, pageSize) => {
              setPagination(prev => ({ ...prev, current: page, pageSize }));
            },
          }}
        />
      </Card>

      {/* 创建用户模态框 */}
      <Modal
        title='新建用户'
        open={isCreateModalVisible}
        onCancel={() => {
          setIsCreateModalVisible(false);
          createForm.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form form={createForm} layout='vertical' onFinish={handleCreateUser}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='username'
                label='用户名'
                rules={[
                  { required: true, message: '请输入用户名' },
                  { min: 3, message: '用户名至少3个字符' },
                ]}
              >
                <Input placeholder='请输入用户名' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='name'
                label='姓名'
                rules={[{ required: true, message: '请输入姓名' }]}
              >
                <Input placeholder='请输入姓名' />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='email'
                label='邮箱'
                rules={[
                  { required: true, message: '请输入邮箱' },
                  { type: 'email', message: '请输入有效的邮箱地址' },
                ]}
              >
                <Input placeholder='请输入邮箱' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='phone' label='电话'>
                <Input placeholder='请输入电话号码' />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name='department' label='部门'>
                <Select placeholder='请选择部门'>
                  <Option value='IT部门'>IT部门</Option>
                  <Option value='财务部门'>财务部门</Option>
                  <Option value='人事部门'>人事部门</Option>
                  <Option value='市场部门'>市场部门</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='password'
                label='密码'
                rules={[
                  { required: true, message: '请输入密码' },
                  { min: 6, message: '密码至少6个字符' },
                ]}
              >
                <Input.Password placeholder='请输入密码' />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit' loading={loading}>
                创建用户
              </Button>
              <Button
                onClick={() => {
                  setIsCreateModalVisible(false);
                  createForm.resetFields();
                }}
              >
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑用户模态框 */}
      <Modal
        title='编辑用户'
        open={isEditModalVisible}
        onCancel={() => {
          setIsEditModalVisible(false);
          editForm.resetFields();
          setSelectedUser(null);
        }}
        footer={null}
        width={600}
      >
        <Form form={editForm} layout='vertical' onFinish={handleUpdateUser}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='username'
                label='用户名'
                rules={[{ min: 3, message: '用户名至少3个字符' }]}
              >
                <Input placeholder='请输入用户名' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='name' label='姓名'>
                <Input placeholder='请输入姓名' />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='email'
                label='邮箱'
                rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
              >
                <Input placeholder='请输入邮箱' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='phone' label='电话'>
                <Input placeholder='请输入电话号码' />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name='department' label='部门'>
            <Select placeholder='请选择部门'>
              <Option value='IT部门'>IT部门</Option>
              <Option value='财务部门'>财务部门</Option>
              <Option value='人事部门'>人事部门</Option>
              <Option value='市场部门'>市场部门</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit' loading={loading}>
                保存更改
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
        title='重置密码'
        open={isPasswordModalVisible}
        onCancel={() => {
          setIsPasswordModalVisible(false);
          passwordForm.resetFields();
          setSelectedUser(null);
        }}
        footer={null}
      >
        <Form form={passwordForm} layout='vertical' onFinish={handleResetPassword}>
          <Form.Item
            name='newPassword'
            label='新密码'
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码至少6个字符' },
            ]}
          >
            <Input.Password placeholder='请输入新密码' />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit' loading={loading}>
                重置密码
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
