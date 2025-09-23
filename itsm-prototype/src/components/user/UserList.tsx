'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Card,
  Button,
  Input,
  Select,
  Space,
  Tag,
  Avatar,
  Tooltip,
  Dropdown,
  Modal,
  message,
  Badge,
  Row,
  Col,
  Switch,
} from 'antd';
import {
  FilterOutlined,
  PlusOutlined,
  ReloadOutlined,
  ExportOutlined,
  EyeOutlined,
  EditOutlined,
  DeleteOutlined,
  MoreOutlined,
  UserOutlined,
  MailOutlined,
  PhoneOutlined,
  ExclamationCircleOutlined,
  LockOutlined,
  UnlockOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TableProps } from 'antd/es/table';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import { useRouter } from 'next/navigation';

import type { 
  User, 
  UserRole, 
  UserStatus,
  UserFilters
} from '@/types/user';

const { Search } = Input;
const { Option } = Select;

interface UserListProps {
  embedded?: boolean;
  showHeader?: boolean;
  filters?: Partial<UserFilters>;
  onUserSelect?: (user: User) => void;
}

const UserList: React.FC<UserListProps> = ({
  embedded = false,
  showHeader = true,
  onUserSelect,
}) => {
  const router = useRouter();
  
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const [filters, setFilters] = useState<UserFilters>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [searchText, setSearchText] = useState('');
  const [showFilters, setShowFilters] = useState(false);

  // 获取用户列表
  const fetchUsers = useCallback(async (params?: {
    page?: number;
    pageSize?: number;
    filters?: UserFilters;
  }) => {
    try {
      setLoading(true);
      // TODO: 实现API调用
      const mockUsers: User[] = [
        {
          id: 1,
          username: 'admin',
          email: 'admin@example.com',
          fullName: '系统管理员',
          role: 'admin',
          status: 'active',
          department: 'IT部门',
          jobTitle: '系统管理员',
          permissions: ['*'],
          groups: [],
          createdAt: '2024-01-01T00:00:00Z',
          updatedAt: '2024-01-01T00:00:00Z',
          preferences: {
            theme: 'light',
            language: 'zh-CN',
            timezone: 'Asia/Shanghai',
            dateFormat: 'YYYY-MM-DD',
            timeFormat: '24h',
            notifications: {
              email: true,
              sms: false,
              inApp: true,
              desktop: true,
              ticketAssigned: true,
              ticketUpdated: true,
              ticketEscalated: true,
              ticketResolved: true,
              slaBreached: true,
              systemMaintenance: true,
            },
            ui: {
              sidebarCollapsed: false,
              tablePageSize: 20,
              defaultView: 'table',
              showAvatars: true,
              compactMode: false,
            },
            work: {
              autoAssign: false,
              defaultPriority: 'medium',
              workingHours: {
                start: '09:00',
                end: '18:00',
                timezone: 'Asia/Shanghai',
              },
              workingDays: [1, 2, 3, 4, 5],
            },
          },
          tenantId: 1,
          isActive: true,
          emailVerified: true,
          phoneVerified: false,
        },
      ];
      
      setUsers(mockUsers);
      setPagination({
        current: params?.page || 1,
        pageSize: params?.pageSize || 20,
        total: mockUsers.length,
      });
    } catch (error) {
      message.error('获取用户列表失败');
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  // 搜索处理
  const handleSearch = (value: string) => {
    setSearchText(value);
    setFilters({ ...filters, search: value });
    fetchUsers({ page: 1, filters: { ...filters, search: value } });
  };

  // 筛选处理
  const handleFilterChange = (key: keyof UserFilters, value: string | string[] | undefined) => {
    const newFilters = { ...filters, [key]: value };
    setFilters(newFilters);
    fetchUsers({ page: 1, filters: newFilters });
  };

  // 清除筛选
  const handleClearFilters = () => {
    setFilters({});
    setSearchText('');
    fetchUsers({ page: 1, filters: {} });
  };

  // 表格分页处理
  const handleTableChange: TableProps<User>['onChange'] = (paginationInfo) => {
    fetchUsers({
      page: paginationInfo.current,
      pageSize: paginationInfo.pageSize,
    });
  };

  // 行选择处理
  const rowSelection = {
    selectedRowKeys,
    onChange: (newSelectedRowKeys: React.Key[]) => {
      setSelectedRowKeys(newSelectedRowKeys);
    },
  };

  // 操作菜单
  const getActionMenu = (record: User): MenuProps => ({
    items: [
      {
        key: 'view',
        icon: <EyeOutlined />,
        label: '查看详情',
        onClick: () => handleViewUser(record),
      },
      {
        key: 'edit',
        icon: <EditOutlined />,
        label: '编辑用户',
        onClick: () => handleEditUser(record),
      },
      {
        type: 'divider',
      },
      {
        key: 'status',
        icon: record.status === 'active' ? <LockOutlined /> : <UnlockOutlined />,
        label: record.status === 'active' ? '禁用用户' : '启用用户',
        onClick: () => handleToggleStatus(record),
      },
      {
        key: 'reset-password',
        label: '重置密码',
        onClick: () => handleResetPassword(record),
      },
      {
        type: 'divider',
      },
      {
        key: 'delete',
        icon: <DeleteOutlined />,
        label: '删除用户',
        danger: true,
        onClick: () => handleDeleteUser(record),
      },
    ],
  });

  // 操作处理函数
  const handleViewUser = (user: User) => {
    if (onUserSelect) {
      onUserSelect(user);
    } else {
      router.push(`/users/${user.id}`);
    }
  };

  const handleEditUser = (user: User) => {
    router.push(`/users/${user.id}/edit`);
  };

  const handleToggleStatus = async (user: User) => {
    try {
      const newStatus = user.status === 'active' ? 'inactive' : 'active';
      // TODO: 实现API调用
      message.success(`用户${newStatus === 'active' ? '启用' : '禁用'}成功`);
      fetchUsers();
    } catch {
      message.error('操作失败');
    }
  };

  const handleResetPassword = (user: User) => {
    Modal.confirm({
      title: '重置密码',
      content: `确定要重置用户 ${user.fullName} 的密码吗？`,
      icon: <ExclamationCircleOutlined />,
      okText: '重置',
      cancelText: '取消',
      onOk: async () => {
        try {
          // TODO: 实现API调用
          message.success('密码重置成功，新密码已发送到用户邮箱');
        } catch {
          message.error('密码重置失败');
        }
      },
    });
  };

  const handleDeleteUser = (user: User) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除用户 ${user.fullName} 吗？此操作不可恢复。`,
      icon: <ExclamationCircleOutlined />,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          // TODO: 实现API调用
          message.success('用户删除成功');
          fetchUsers();
        } catch {
          message.error('用户删除失败');
        }
      },
    });
  };

  // 批量操作
  const handleBatchAction = (action: string) => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要操作的用户');
      return;
    }
    
    // TODO: 实现批量操作逻辑
    message.info(`批量${action}功能开发中`);
  };

  // 导出数据
  const handleExport = () => {
    // TODO: 实现导出功能
    message.info('导出功能开发中');
  };

  // 状态标签渲染
  const renderStatusTag = (status: UserStatus) => {
    const statusConfig = {
      active: { color: 'green', text: '正常' },
      inactive: { color: 'default', text: '禁用' },
      suspended: { color: 'red', text: '暂停' },
      pending: { color: 'orange', text: '待激活' },
    };
    
    const config = statusConfig[status];
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  // 角色标签渲染
  const renderRoleTag = (role: UserRole) => {
    const roleConfig = {
      admin: { color: 'red', text: '管理员' },
      manager: { color: 'blue', text: '经理' },
      agent: { color: 'green', text: '客服' },
      technician: { color: 'orange', text: '技术员' },
      end_user: { color: 'default', text: '普通用户' },
    };
    
    const config = roleConfig[role];
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  // 表格列定义
  const columns: ColumnsType<User> = [
    {
      title: '用户',
      key: 'user',
      width: 200,
      fixed: 'left',
      render: (_, record) => (
        <Space>
          <Avatar 
            size="small" 
            icon={<UserOutlined />} 
            src={record.avatar}
          />
          <div>
            <div>
              <Button
                type="link"
                onClick={() => handleViewUser(record)}
                style={{ padding: 0, height: 'auto' }}
              >
                {record.fullName}
              </Button>
            </div>
            <div style={{ fontSize: '12px', color: '#999' }}>
              @{record.username}
            </div>
          </div>
        </Space>
      ),
    },
    {
      title: '联系方式',
      key: 'contact',
      width: 200,
      render: (_, record) => (
        <Space direction="vertical" size="small">
          <Space size="small">
            <MailOutlined style={{ color: '#999' }} />
            <span>{record.email}</span>
          </Space>
          {record.phone && (
            <Space size="small">
              <PhoneOutlined style={{ color: '#999' }} />
              <span>{record.phone}</span>
            </Space>
          )}
        </Space>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 100,
      render: renderRoleTag,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: renderStatusTag,
    },
    {
      title: '部门',
      dataIndex: 'department',
      key: 'department',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: '职位',
      dataIndex: 'jobTitle',
      key: 'jobTitle',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: '最后登录',
      dataIndex: 'lastLoginAt',
      key: 'lastLoginAt',
      width: 150,
      render: (text) => text ? (
        <Tooltip title={dayjs(text).format('YYYY-MM-DD HH:mm:ss')}>
          {dayjs(text).format('MM-DD HH:mm')}
        </Tooltip>
      ) : (
        <span style={{ color: '#999' }}>从未登录</span>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (text) => (
        <Tooltip title={dayjs(text).format('YYYY-MM-DD HH:mm:ss')}>
          {dayjs(text).format('MM-DD HH:mm')}
        </Tooltip>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      fixed: 'right',
      render: (_, record) => (
        <Dropdown menu={getActionMenu(record)} trigger={['click']}>
          <Button type="text" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
  ];

  // 筛选器组件
  const renderFilters = () => (
    <Card size="small" style={{ marginBottom: 16 }}>
      <Row gutter={[16, 16]}>
        <Col span={6}>
          <Select
            placeholder="角色"
            allowClear
            style={{ width: '100%' }}
            value={filters.role}
            onChange={(value) => handleFilterChange('role', value)}
          >
            <Option value="admin">管理员</Option>
            <Option value="manager">经理</Option>
            <Option value="agent">客服</Option>
            <Option value="technician">技术员</Option>
            <Option value="end_user">普通用户</Option>
          </Select>
        </Col>
        <Col span={6}>
          <Select
            placeholder="状态"
            allowClear
            style={{ width: '100%' }}
            value={filters.status}
            onChange={(value) => handleFilterChange('status', value)}
          >
            <Option value="active">正常</Option>
            <Option value="inactive">禁用</Option>
            <Option value="suspended">暂停</Option>
            <Option value="pending">待激活</Option>
          </Select>
        </Col>
        <Col span={6}>
          <Select
            placeholder="部门"
            allowClear
            style={{ width: '100%' }}
            value={filters.department}
            onChange={(value) => handleFilterChange('department', value)}
          >
            <Option value="IT部门">IT部门</Option>
            <Option value="人事部">人事部</Option>
            <Option value="财务部">财务部</Option>
            <Option value="市场部">市场部</Option>
          </Select>
        </Col>
        <Col span={6}>
          <Space>
            <span>仅显示活跃用户:</span>
            <Switch
              checked={filters.isActive}
              onChange={(checked) => handleFilterChange('isActive', checked)}
            />
          </Space>
        </Col>
      </Row>
      <Row style={{ marginTop: 16 }}>
        <Col>
          <Button onClick={handleClearFilters}>清除筛选</Button>
        </Col>
      </Row>
    </Card>
  );

  return (
    <div>
      {showHeader && (
        <Card>
          <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
            <Col>
              <Space>
                <Search
                  placeholder="搜索用户..."
                  allowClear
                  style={{ width: 300 }}
                  value={searchText}
                  onSearch={handleSearch}
                  onChange={(e) => setSearchText(e.target.value)}
                />
                <Button
                  icon={<FilterOutlined />}
                  onClick={() => setShowFilters(!showFilters)}
                >
                  筛选
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => fetchUsers()}
                >
                  刷新
                </Button>
              </Space>
            </Col>
            <Col>
              <Space>
                {selectedRowKeys.length > 0 && (
                  <>
                    <Badge count={selectedRowKeys.length}>
                      <Button onClick={() => handleBatchAction('启用')}>
                        批量启用
                      </Button>
                    </Badge>
                    <Button onClick={() => handleBatchAction('禁用')}>
                      批量禁用
                    </Button>
                  </>
                )}
                <Button
                  icon={<ExportOutlined />}
                  onClick={handleExport}
                >
                  导出
                </Button>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => router.push('/users/new')}
                >
                  新建用户
                </Button>
              </Space>
            </Col>
          </Row>

          {showFilters && renderFilters()}
        </Card>
      )}

      <Card>
        <Table
          rowSelection={embedded ? undefined : rowSelection}
          columns={columns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          onChange={handleTableChange}
          scroll={{ x: 1200 }}
          size={embedded ? 'small' : 'middle'}
        />
      </Card>
    </div>
  );
};

export default UserList;