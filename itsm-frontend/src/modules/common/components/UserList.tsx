/**
 * 用户列表组件
 */

import React, { useState, useEffect } from 'react';
import { Table, Card, Tag, Space, Button, message } from 'antd';
import { UserOutlined, TeamOutlined } from '@ant-design/icons';
import { CommonApi } from '../api';
import { UserRoleLabels } from '../constants';
import type { User } from '../types';

const UserList: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [users, setUsers] = useState<User[]>([]);

    const fetchUsers = async () => {
        setLoading(true);
        try {
            const data = await CommonApi.listUsers();
            setUsers(data);
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
            render: (text: string) => <Space><UserOutlined /> {text}</Space>,
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
            dataIndex: 'created_at',
            key: 'created_at',
            render: (text: string) => new Date(text).toLocaleString(),
        },
    ];

    return (
        <Card title={<span><TeamOutlined /> 用户管理</span>}>
            <Table
                dataSource={users}
                columns={columns}
                rowKey="id"
                loading={loading}
                pagination={{ pageSize: 10 }}
            />
        </Card>
    );
};

export default UserList;
