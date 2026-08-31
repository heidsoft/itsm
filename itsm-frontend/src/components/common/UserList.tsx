/**
 * 用户列表组件
 */

import React, { useState, useEffect } from 'react';
import { Table, Card, Tag, Space, Button, App } from 'antd';
import { User as UserIcon, Users } from 'lucide-react';
import { UserRoleLabels } from '@/constants/common';
import { UserApi, type User } from '@/lib/api/user-api';

const UserList: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const { message } = App.useApp();

  const fetchUsers = async () => {
    setLoading(true);
    try {
      // 后端 /api/v1/users 返回 {users, pagination} 包装响应，
      // 必须解包 .users 后再赋给表格 dataSource，否则 Ant Design Table
      // 的内部 useMemo 调用 dataSource.some(...) 会抛
      // "ed.some is not a function" TypeError，由 error.tsx 渲染 500 页面。
      const response = await UserApi.getUsers();
      setUsers(response.users || []);
    } catch (error) {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const columns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (text: string) => (
        <Space>
          <UserIcon /> {text}
        </Space>
      ),
    },
    {
      title: '姓名',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => (
        <Tag color="blue">{UserRoleLabels[role as keyof typeof UserRoleLabels] || role}</Tag>
      ),
    },
    {
      title: '部门',
      dataIndex: 'department',
      key: 'department',
    },
    {
      title: '状态',
      dataIndex: 'active',
      key: 'active',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'red'}>{active ? '激活' : '禁用'}</Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (text: string) => new Date(text).toLocaleString(),
    },
  ];

  return (
    <Card
      title={
        <span>
          <Users /> 用户管理
        </span>
      }
    >
      <Table
        dataSource={users}
        columns={columns}
        rowKey="id"
        loading={loading}
        scroll={{ x: 'max-content' }}
        pagination={{ pageSize: 10 }}
      />
    </Card>
  );
};

export default UserList;
