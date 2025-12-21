'use client';

/**
 * 变更列表组件
 */

import React, { useState, useEffect } from 'react';
import {
    Table, Tag, Button, Card, Space, Tooltip,
    Input, Select, Form, message, Modal
} from 'antd';
import {
    SearchOutlined, EyeOutlined, EditOutlined,
    PlusOutlined, SyncOutlined
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ChangeApi } from '../api';
import {
    ChangeStatus, ChangeType, ChangePriority,
    ChangeStatusLabels, ChangeTypeLabels, ChangePriorityLabels
} from '../constants';
import type { Change, ChangeQuery } from '../types';

const { Option } = Select;

// 颜色映射
const statusColors: Record<string, string> = {
    [ChangeStatus.DRAFT]: 'default',
    [ChangeStatus.PENDING]: 'orange',
    [ChangeStatus.APPROVED]: 'cyan',
    [ChangeStatus.IN_PROGRESS]: 'blue',
    [ChangeStatus.COMPLETED]: 'green',
    [ChangeStatus.REJECTED]: 'red',
    [ChangeStatus.ROLLED_BACK]: 'magenta',
    [ChangeStatus.CANCELLED]: 'default',
};

const ChangeList: React.FC = () => {
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<Change[]>([]);
    const [total, setTotal] = useState(0);
    const [form] = Form.useForm();

    const [query, setQuery] = useState<ChangeQuery>({
        page: 1,
        page_size: 10,
    });

    const loadData = async () => {
        setLoading(true);
        try {
            const values = await form.validateFields();
            const resp = await ChangeApi.getChanges({
                ...query,
                ...values,
            });
            setData(resp.changes || []);
            setTotal(resp.total || 0);
        } catch (error) {
            console.error(error);
            message.error('加载变更列表失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
    }, [query]);

    const handleSearch = () => {
        setQuery(prev => ({ ...prev, page: 1 }));
    };

    const handleReset = () => {
        form.resetFields();
        setQuery({ page: 1, page_size: 10 });
    };

    const columns = [
        {
            title: 'ID',
            dataIndex: 'id',
            width: 70,
        },
        {
            title: '标题',
            dataIndex: 'title',
            ellipsis: true,
        },
        {
            title: '模型',
            dataIndex: 'type',
            width: 100,
            render: (type: ChangeType) => ChangeTypeLabels[type] || type,
        },
        {
            title: '状态',
            dataIndex: 'status',
            width: 100,
            render: (status: ChangeStatus) => (
                <Tag color={statusColors[status]}>
                    {ChangeStatusLabels[status]}
                </Tag>
            ),
        },
        {
            title: '优先级',
            dataIndex: 'priority',
            width: 80,
            render: (priority: ChangePriority) => (
                <Tag>{ChangePriorityLabels[priority]}</Tag>
            ),
        },
        {
            title: '创建人',
            dataIndex: 'created_by_name',
            width: 100,
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            width: 160,
            render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
        },
        {
            title: '操作',
            key: 'action',
            width: 120,
            render: (_: any, record: Change) => (
                <Space size="small">
                    <Tooltip title="查看详情">
                        <Button
                            type="text"
                            icon={<EyeOutlined />}
                            onClick={() => router.push(`/changes/${record.id}`)}
                        />
                    </Tooltip>
                    <Tooltip title="编辑">
                        <Button
                            type="text"
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/changes/${record.id}/edit`)}
                        />
                    </Tooltip>
                </Space>
            ),
        },
    ];

    return (
        <Card bordered={false}>
            <Form form={form} layout="inline" style={{ marginBottom: 24 }}>
                <Form.Item name="search">
                    <Input placeholder="搜索标题" allowClear prefix={<SearchOutlined />} />
                </Form.Item>
                <Form.Item name="status">
                    <Select placeholder="状态" style={{ width: 120 }} allowClear>
                        {Object.entries(ChangeStatusLabels).map(([value, label]) => (
                            <Option key={value} value={value}>{label}</Option>
                        ))}
                    </Select>
                </Form.Item>
                <Form.Item name="type">
                    <Select placeholder="类型" style={{ width: 120 }} allowClear>
                        {Object.entries(ChangeTypeLabels).map(([value, label]) => (
                            <Option key={value} value={value}>{label}</Option>
                        ))}
                    </Select>
                </Form.Item>
                <Form.Item>
                    <Space>
                        <Button type="primary" onClick={handleSearch}>查询</Button>
                        <Button icon={<SyncOutlined />} onClick={loadData} />
                    </Space>
                </Form.Item>
                <div style={{ flex: 1, textAlign: 'right' }}>
                    <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => router.push('/changes/create')}
                    >
                        新建变更
                    </Button>
                </div>
            </Form>

            <Table
                rowKey="id"
                columns={columns as any}
                dataSource={data}
                loading={loading}
                pagination={{
                    current: query.page,
                    pageSize: query.page_size,
                    total: total,
                    onChange: (page, page_size) => setQuery(prev => ({ ...prev, page, page_size })),
                }}
            />
        </Card>
    );
};

export default ChangeList;
