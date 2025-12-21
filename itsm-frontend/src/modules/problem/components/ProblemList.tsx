'use client';

/**
 * 问题列表组件
 */

import React, { useState, useEffect } from 'react';
import {
    Table, Tag, Button, Card, Space, Tooltip,
    Input, Select, Form, message, Modal
} from 'antd';
import {
    SearchOutlined, EyeOutlined, EditOutlined,
    DeleteOutlined, PlusOutlined, SyncOutlined
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ProblemApi } from '../api';
import {
    ProblemStatus, ProblemPriority,
    ProblemStatusLabels, ProblemPriorityLabels
} from '../constants';
import type { Problem, ProblemQuery } from '../types';

const { Option } = Select;

// 状态和优先级颜色映射
const statusColors: Record<string, string> = {
    [ProblemStatus.OPEN]: 'red',
    [ProblemStatus.IN_PROGRESS]: 'blue',
    [ProblemStatus.RESOLVED]: 'green',
    [ProblemStatus.CLOSED]: 'default',
};

const priorityColors: Record<string, string> = {
    [ProblemPriority.CRITICAL]: 'magenta',
    [ProblemPriority.HIGH]: 'red',
    [ProblemPriority.MEDIUM]: 'orange',
    [ProblemPriority.LOW]: 'blue',
};

const ProblemList: React.FC = () => {
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<Problem[]>([]);
    const [total, setTotal] = useState(0);
    const [form] = Form.useForm();

    const [query, setQuery] = useState<ProblemQuery>({
        page: 1,
        page_size: 10,
    });

    const loadData = async () => {
        setLoading(true);
        try {
            const values = await form.validateFields();
            const resp = await ProblemApi.getProblems({
                ...query,
                ...values,
            });
            setData(resp.problems || []);
            setTotal(resp.total || 0);
        } catch (error) {
            console.error(error);
            message.error('加载问题列表失败');
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

    const handleDelete = (id: number) => {
        Modal.confirm({
            title: '确认删除',
            content: '确定要删除该问题吗？此操作不可撤销。',
            onOk: async () => {
                try {
                    await ProblemApi.deleteProblem(id);
                    message.success('删除成功');
                    loadData();
                } catch (error) {
                    message.error('删除失败');
                }
            },
        });
    };

    const columns = [
        {
            title: 'ID',
            dataIndex: 'id',
            width: 80,
        },
        {
            title: '标题',
            dataIndex: 'title',
            ellipsis: true,
        },
        {
            title: '状态',
            dataIndex: 'status',
            width: 100,
            render: (status: ProblemStatus) => (
                <Tag color={statusColors[status]}>
                    {ProblemStatusLabels[status]}
                </Tag>
            ),
        },
        {
            title: '优先级',
            dataIndex: 'priority',
            width: 100,
            render: (priority: ProblemPriority) => (
                <Tag color={priorityColors[priority]}>
                    {ProblemPriorityLabels[priority]}
                </Tag>
            ),
        },
        {
            title: '分类',
            dataIndex: 'category',
            width: 120,
        },
        {
            title: '更新时间',
            dataIndex: 'updated_at',
            width: 180,
            render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
        },
        {
            title: '操作',
            key: 'action',
            width: 150,
            render: (_: any, record: Problem) => (
                <Space size="small">
                    <Tooltip title="查看详情">
                        <Button
                            type="text"
                            icon={<EyeOutlined />}
                            onClick={() => router.push(`/problems/${record.id}`)}
                        />
                    </Tooltip>
                    <Tooltip title="编辑">
                        <Button
                            type="text"
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/problems/${record.id}/edit`)}
                        />
                    </Tooltip>
                    <Tooltip title="删除">
                        <Button
                            type="text"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => handleDelete(record.id)}
                        />
                    </Tooltip>
                </Space>
            ),
        },
    ];

    return (
        <Card bordered={false}>
            <Form form={form} layout="inline" style={{ marginBottom: 24 }}>
                <Form.Item name="keyword">
                    <Input placeholder="搜索标题或内容" allowClear prefix={<SearchOutlined />} />
                </Form.Item>
                <Form.Item name="status">
                    <Select placeholder="状态" style={{ width: 120 }} allowClear>
                        {Object.entries(ProblemStatusLabels).map(([value, label]) => (
                            <Option key={value} value={value}>{label}</Option>
                        ))}
                    </Select>
                </Form.Item>
                <Form.Item name="priority">
                    <Select placeholder="优先级" style={{ width: 120 }} allowClear>
                        {Object.entries(ProblemPriorityLabels).map(([value, label]) => (
                            <Option key={value} value={value}>{label}</Option>
                        ))}
                    </Select>
                </Form.Item>
                <Form.Item>
                    <Space>
                        <Button type="primary" onClick={handleSearch}>查询</Button>
                        <Button onClick={handleReset}>重置</Button>
                    </Space>
                </Form.Item>
                <div style={{ flex: 1, textAlign: 'right' }}>
                    <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => router.push('/problems/create')}
                    >
                        新建问题
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
                    showSizeChanger: true,
                    showTotal: (t) => `共 ${t} 条记录`
                }}
            />
        </Card>
    );
};

export default ProblemList;
