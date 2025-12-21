'use client';

/**
 * 事件列表组件
 */

import React, { useState, useEffect } from 'react';
import {
    Table, Tag, Button, Card, Space, Tooltip,
    Input, Select, Form, Row, Col, message
} from 'antd';
import {
    SearchOutlined, EyeOutlined, EditOutlined,
    SyncOutlined, PlusOutlined
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { IncidentApi } from '../api';
import {
    IncidentStatus, IncidentPriority, IncidentSeverity,
    IncidentStatusLabels, IncidentPriorityLabels, IncidentSeverityLabels
} from '../constants';
import type { Incident, IncidentQuery } from '../types';

const { Option } = Select;

// 状态颜色映射
const statusColors: Record<string, string> = {
    [IncidentStatus.NEW]: 'blue',
    [IncidentStatus.IN_PROGRESS]: 'processing',
    [IncidentStatus.RESOLVED]: 'success',
    [IncidentStatus.CLOSED]: 'default',
};

// 优先级颜色
const priorityColors: Record<string, string> = {
    [IncidentPriority.URGENT]: 'red',
    [IncidentPriority.HIGH]: 'orange',
    [IncidentPriority.MEDIUM]: 'gold',
    [IncidentPriority.LOW]: 'cyan',
};

const IncidentList: React.FC = () => {
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<Incident[]>([]);
    const [total, setTotal] = useState(0);
    const [form] = Form.useForm();

    const [query, setQuery] = useState<IncidentQuery>({
        page: 1,
        size: 10,
        scope: 'all', // 默认显示所有
    });

    const loadData = async () => {
        setLoading(true);
        try {
            const values = await form.validateFields();
            const apiQuery = {
                ...query,
                ...values,
            };
            const resp = await IncidentApi.getIncidents(apiQuery);
            setData(resp.items);
            setTotal(resp.total);
        } catch (error) {
            console.error(error);
            message.error('加载事件列表失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
    }, [query.page, query.size, query.scope]);

    const handleSearch = () => {
        setQuery(prev => ({ ...prev, page: 1 }));
    };

    const handleReset = () => {
        form.resetFields();
        setQuery(prev => ({ ...prev, page: 1 }));
    };

    const columns = [
        {
            title: '编号',
            dataIndex: 'incident_number',
            width: 120,
            render: (text: string) => <a>{text}</a>,
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
            render: (status: IncidentStatus) => (
                <Tag color={statusColors[status]}>
                    {IncidentStatusLabels[status]}
                </Tag>
            ),
        },
        {
            title: '优先级',
            dataIndex: 'priority',
            width: 100,
            render: (p: IncidentPriority) => (
                <Tag color={priorityColors[p]}>
                    {IncidentPriorityLabels[p]}
                </Tag>
            ),
        },
        {
            title: '严重程度',
            dataIndex: 'severity',
            width: 100,
            render: (s: IncidentSeverity) => IncidentSeverityLabels[s],
            responsive: ['md'],
        },
        {
            title: '报告人',
            dataIndex: 'reporter_id', // 暂时显示ID，后续关联User
            width: 100,
            responsive: ['lg'],
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            width: 180,
            render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
            responsive: ['sm'],
        },
        {
            title: '操作',
            key: 'action',
            width: 120,
            render: (_: any, record: Incident) => (
                <Space size="small">
                    <Tooltip title="查看详情">
                        <Button
                            type="text"
                            icon={<EyeOutlined />}
                            onClick={() => router.push(`/incidents/${record.id}`)}
                        />
                    </Tooltip>
                    <Tooltip title="编辑">
                        <Button
                            type="text"
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/incidents/${record.id}/edit`)}
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
                    <Input placeholder="搜索编号或标题" allowClear prefix={<SearchOutlined />} />
                </Form.Item>
                <Form.Item name="status">
                    <Select placeholder="状态" style={{ width: 120 }} allowClear>
                        {Object.entries(IncidentStatusLabels).map(([key, label]) => (
                            <Option key={key} value={key}>{label}</Option>
                        ))}
                    </Select>
                </Form.Item>
                <Form.Item name="priority">
                    <Select placeholder="优先级" style={{ width: 120 }} allowClear>
                        {Object.entries(IncidentPriorityLabels).map(([key, label]) => (
                            <Option key={key} value={key}>{label}</Option>
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
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => router.push('/incidents/create')}>
                        新建事件
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
                    pageSize: query.size,
                    total: total,
                    onChange: (page, size) => setQuery(prev => ({ ...prev, page, size })),
                    showSizeChanger: true,
                    showTotal: (t) => `共 ${t} 条记录`
                }}
            />
        </Card>
    );
};

export default IncidentList;
