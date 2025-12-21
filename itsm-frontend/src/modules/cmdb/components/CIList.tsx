'use client';

/**
 * 配置项 (CI) 列表组件
 */

import React, { useState, useEffect } from 'react';
import {
    Table, Tag, Button, Card, Space, Tooltip,
    Input, Select, Form, message, Modal, Breadcrumb
} from 'antd';
import {
    SearchOutlined, EyeOutlined, EditOutlined,
    PlusOutlined, ReloadOutlined, DeleteOutlined
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { CMDBApi } from '../api';
import { CIStatus, CIStatusLabels } from '../constants';
import type { ConfigurationItem, CIType, CIQuery } from '../types';

const { Option } = Select;

const statusColors: Record<string, string> = {
    [CIStatus.ACTIVE]: 'green',
    [CIStatus.INACTIVE]: 'default',
    [CIStatus.MAINTENANCE]: 'orange',
    [CIStatus.DECOMMISSIONED]: 'red',
};

const CIList: React.FC = () => {
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<ConfigurationItem[]>([]);
    const [total, setTotal] = useState(0);
    const [types, setTypes] = useState<CIType[]>([]);
    const [form] = Form.useForm();

    const [query, setQuery] = useState<CIQuery>({
        page: 1,
        page_size: 10,
    });

    const loadTypes = async () => {
        try {
            const res = await CMDBApi.getTypes();
            setTypes(res || []);
        } catch (e) {
            console.error(e);
        }
    };

    const loadData = async () => {
        setLoading(true);
        try {
            const values = await form.validateFields();
            const resp = await CMDBApi.getCIs({
                ...query,
                ...values,
            });
            setData(resp.items || []);
            setTotal(resp.total || 0);
        } catch (error) {
            console.error(error);
            message.error('加载配置项列表失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadTypes();
        loadData();
    }, [query]);

    const handleSearch = () => {
        setQuery(prev => ({ ...prev, page: 1 }));
    };

    const handleDelete = (id: number) => {
        Modal.confirm({
            title: '重申此操作',
            content: '确定要删除此配置项吗？相关关系也将受到影响。',
            onOk: async () => {
                try {
                    await CMDBApi.deleteCI(id);
                    message.success('删除成功');
                    loadData();
                } catch (e) {
                    message.error('删除失败');
                }
            }
        });
    };

    const columns = [
        {
            title: 'ID',
            dataIndex: 'id',
            width: 70,
        },
        {
            title: '资产名称',
            dataIndex: 'name',
            ellipsis: true,
            render: (text: string, record: ConfigurationItem) => (
                <a onClick={() => router.push(`/cmdb/cis/${record.id}`)}>{text}</a>
            )
        },
        {
            title: '类型',
            dataIndex: 'ci_type_id',
            width: 120,
            render: (id: number) => types.find(t => t.id === id)?.name || `类型 ${id}`,
        },
        {
            title: '状态',
            dataIndex: 'status',
            width: 100,
            render: (status: CIStatus) => (
                <Tag color={statusColors[status]}>
                    {CIStatusLabels[status] || status}
                </Tag>
            ),
        },
        {
            title: '型号/厂商',
            key: 'model_vendor',
            width: 180,
            render: (_: any, record: ConfigurationItem) => (
                <span>{record.model || '-'} / {record.vendor || '-'}</span>
            )
        },
        {
            title: '最后更新',
            dataIndex: 'updated_at',
            width: 160,
            render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
        },
        {
            title: '操作',
            key: 'action',
            width: 120,
            render: (_: any, record: ConfigurationItem) => (
                <Space size="small">
                    <Tooltip title="编辑">
                        <Button
                            type="text"
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/cmdb/cis/${record.id}/edit`)}
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
            <Breadcrumb style={{ marginBottom: 16 }}>
                <Breadcrumb.Item>首页</Breadcrumb.Item>
                <Breadcrumb.Item>配置管理</Breadcrumb.Item>
                <Breadcrumb.Item>配置项列表</Breadcrumb.Item>
            </Breadcrumb>

            <Form form={form} layout="inline" style={{ marginBottom: 24 }}>
                <Form.Item name="search">
                    <Input placeholder="搜索名称/序列号" allowClear prefix={<SearchOutlined />} />
                </Form.Item>
                <Form.Item name="ci_type_id">
                    <Select placeholder="资产类型" style={{ width: 140 }} allowClear>
                        {types.map(t => (
                            <Option key={t.id} value={t.id}>{t.name}</Option>
                        ))}
                    </Select>
                </Form.Item>
                <Form.Item name="status">
                    <Select placeholder="状态" style={{ width: 110 }} allowClear>
                        {Object.entries(CIStatusLabels).map(([value, label]) => (
                            <Option key={value} value={value}>{label}</Option>
                        ))}
                    </Select>
                </Form.Item>
                <Form.Item>
                    <Space>
                        <Button type="primary" onClick={handleSearch}>查询</Button>
                        <Button icon={<ReloadOutlined />} onClick={loadData} />
                    </Space>
                </Form.Item>
                <div style={{ flex: 1, textAlign: 'right' }}>
                    <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => router.push('/cmdb/cis/create')}
                    >
                        录入资产
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

export default CIList;
