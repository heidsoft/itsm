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
const { Title, Text } = Typography;
const { Search: AntSearch } = Input;
const { Option } = Select;

// 临时使用Mock数据，等后端API调试完成后切换
const mockUsers = [
  {
    id: 1,
    username: 'admin',
    email: 'admin@company.com',
    name: '系统管理员',
    department: 'IT部门',
    phone: '13800138000',
    active: true,
    tenant_id: 1,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    username: 'john.doe',
    email: 'john.doe@company.com',
    name: '约翰·多伊',
    department: 'IT部门',
    phone: '13800138001',
    active: true,
    tenant_id: 1,
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
  },
  {
    id: 3,
    username: 'jane.smith',
    email: 'jane.smith@company.com',
    name: '简·史密斯',
    department: '财务部门',
    phone: '13800138002',
    active: false,
    tenant_id: 1,
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-03T00:00:00Z',
  },
];

interface User {
  id: number;
  username: string;
  email: string;
  name: string;
  department: string;
  phone: string;
  active: boolean;
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

const UserManagement: React.FC = () => {
  const { token } = theme.useToken();
  const { message } = App.useApp();

  // 状态管理
  const [users, setUsers] = useState<User[]>(mockUsers);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState({ total: 3, active: 2, inactive: 1 });
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 3,
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

  // 模拟API调用 - 后续替换为真实API
  const handleCreateUser = async (values: Partial<User>) => {
    setLoading(true);
    try {
      // 模拟API延迟
      await new Promise(resolve => setTimeout(resolve, 1000));

      const newUser: User = {
        id: Math.max(...users.map(u => u.id)) + 1,
        username: values.username || '',
        email: values.email || '',
        name: values.name || '',
        department: values.department || '',
        phone: values.phone || '',
        active: true,
        tenant_id: 1,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      setUsers([...users, newUser]);
      setStats(prev => ({
        ...prev,
        total: prev.total + 1,
        active: prev.active + 1,
      }));
      message.success('用户创建成功');
      setIsCreateModalVisible(false);
      createForm.resetFields();
    } catch (error) {
      message.error('创建用户失败');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateUser = async (values: Partial<User>) => {
    if (!selectedUser) return;

    setLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));

      const updatedUsers = users.map(user =>
        user.id === selectedUser.id
          ? { ...user, ...values, updated_at: new Date().toISOString() }
          : user
      );

      setUsers(updatedUsers);
      message.success('用户更新成功');
      setIsEditModalVisible(false);
      editForm.resetFields();
      setSelectedUser(null);
    } catch (error) {
      message.error('更新用户失败');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteUser = async (userId: number) => {
    setLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));

      const userToDelete = users.find(u => u.id === userId);
      const updatedUsers = users.filter(user => user.id !== userId);

      setUsers(updatedUsers);
      setStats(prev => ({
        ...prev,
        total: prev.total - 1,
        active: userToDelete?.active ? prev.active - 1 : prev.active,
        inactive: !userToDelete?.active ? prev.inactive - 1 : prev.inactive,
      }));
      message.success('用户删除成功');
    } catch (error) {
      message.error('删除用户失败');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleUserStatus = async (userId: number, currentStatus: boolean) => {
    setLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 500));

      const updatedUsers = users.map(user =>
        user.id === userId
          ? {
              ...user,
              active: !currentStatus,
              updated_at: new Date().toISOString(),
            }
          : user
      );

      setUsers(updatedUsers);
      setStats(prev => ({
        ...prev,
        active: currentStatus ? prev.active - 1 : prev.active + 1,
        inactive: currentStatus ? prev.inactive + 1 : prev.inactive - 1,
      }));
      message.success(`用户${!currentStatus ? '激活' : '禁用'}成功`);
    } catch (error) {
      message.error('更改用户状态失败');
    } finally {
      setLoading(false);
    }
  };

  const handleResetPassword = async (values: { newPassword: string }) => {
    setLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      message.success('密码重置成功');
      setIsPasswordModalVisible(false);
      passwordForm.resetFields();
      setSelectedUser(null);
    } catch (error) {
      message.error('重置密码失败');
    } finally {
      setLoading(false);
    }
  };

  // 搜索和筛选逻辑
  const filteredUsers = users.filter(user => {
    const matchesSearch =
      !filters.search ||
      user.username.toLowerCase().includes(filters.search.toLowerCase()) ||
      user.name.toLowerCase().includes(filters.search.toLowerCase()) ||
      user.email.toLowerCase().includes(filters.search.toLowerCase());

    const matchesStatus =
      !filters.status ||
      (filters.status === 'active' && user.active) ||
      (filters.status === 'inactive' && !user.active);

    const matchesDepartment = !filters.department || user.department === filters.department;

    return matchesSearch && matchesStatus && matchesDepartment;
  });

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
          dataSource={filteredUsers}
          rowKey='id'
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: filteredUsers.length,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条/共 ${total} 条`,
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
