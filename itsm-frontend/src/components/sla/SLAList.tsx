'use client';

/**
 * SLA 定义列表组件
 */

import React, { useState, useEffect } from 'react';
import {
    Table, Tag, Button, Card, Space, Tooltip,
    App, Modal, Breadcrumb, Switch, Empty
} from 'antd';
import {
    PlusOutlined, ReloadOutlined, EditOutlined,
    DeleteOutlined, BellOutlined
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { SLAApi } from '@/lib/api/';
import { SLAPriorityLabels, SLAPriorityColors } from '@/constants/sla';
import type { SLADefinition } from '@/types/biz/sla';

const SLAList: React.FC = () => {
    const router = useRouter();
    const { message } = App.useApp();
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<SLADefinition[]>([]);
    const [total, setTotal] = useState(0);
    const [pagination, setPagination] = useState({ page: 1, size: 10 });

    const loadData = async () => {
        setLoading(true);
        try {
            const resp = await SLAApi.getDefinitions(pagination);
            setData((resp.items || []) as unknown as SLADefinition[]);
            setTotal((resp as any).total || 0);
        } catch (error) {
            // console.error(error);
            message.error('加载 SLA 列表失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [pagination]);

    const handleToggleActive = async (record: SLADefinition, checked: boolean) => {
        try {
            await SLAApi.updateDefinition(record.id, { is_active: checked });
            message.success(`${checked ? '激活' : '禁用'}成功`);
            loadData();
        } catch (e) {
            message.error('操作失败');
        }
    };

    const handleDelete = (id: number) => {
        Modal.confirm({
            title: '确定要删除此 SLA 定义吗？',
            content: '删除后无法恢复，且可能影响相关合规性检查。',
            onOk: async () => {
                try {
                    await SLAApi.deleteDefinition(id);
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
            title: '名称',
            dataIndex: 'name',
            key: 'name',
            render: (text: string, record: SLADefinition) => (
                <a onClick={() => router.push(`/sla/definitions/${record.id}`)}>{text}</a>
            )
        },
        {
            title: '适用优先级',
            dataIndex: 'priority',
            render: (priority: string) => (
                <Tag color={SLAPriorityColors[priority] || 'blue'}>
                    {SLAPriorityLabels[priority] || priority}
                </Tag>
            ),
        },
        {
            title: '响应时间(分)',
            dataIndex: 'response_time',
            width: 120,
        },
        {
            title: '解决时间(分)',
            dataIndex: 'resolution_time',
            width: 120,
        },
        {
            title: '状态',
            dataIndex: 'is_active',
            width: 100,
            render: (active: boolean, record: SLADefinition) => (
                <Switch
                    size="small"
                    checked={active}
                    onChange={(checked) => handleToggleActive(record, checked)}
                />
            ),
        },
        {
            title: '更新时间',
            dataIndex: 'updated_at',
            width: 160,
            render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
        },
        {
            title: '操作',
            key: 'action',
            width: 160,
            render: (_: unknown, record: SLADefinition) => (
                <Space size="small">
                    <Tooltip title="编辑">
                        <Button
                            type="text"
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/sla/definitions/${record.id}/edit`)}
                        />
                    </Tooltip>
                    <Tooltip title="预警规则">
                        <Button
                            type="text"
                            icon={<BellOutlined />}
                            onClick={() => router.push(`/sla/definitions/${record.id}/alerts`)}
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
        <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
            <Breadcrumb className="mb-4">
                <Breadcrumb.Item>首页</Breadcrumb.Item>
                <Breadcrumb.Item>服务级别管理</Breadcrumb.Item>
                <Breadcrumb.Item>SLA 定义</Breadcrumb.Item>
            </Breadcrumb>

            <div className="flex justify-between mb-4">
                <Space>
                    <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => router.push('/sla/definitions/new')}
                    >
                        新建 SLA
                    </Button>
                </Space>
                <Button icon={<ReloadOutlined />} onClick={loadData} />
            </div>

            <Table
                rowKey="id"
                columns={columns as any}
                dataSource={data}
                loading={loading}
                scroll={{ x: 'max-content' }}
                locale={{
                    emptyText: (
                        <Empty
                            image={Empty.PRESENTED_IMAGE_SIMPLE}
                            description="暂无SLA数据"
                        >
                            <Button type="primary" onClick={() => router.push('/sla/definitions/new')}>
                                创建第一个SLA
                            </Button>
                        </Empty>
                    ),
                }}
                pagination={{
                    current: pagination.page,
                    pageSize: pagination.size,
                    total: total,
                    onChange: (page, size) => setPagination({ page, size }),
                }}
                getPopupContainer={(node) => node.parentElement || document.body}
            />
        </Card>
    );
};

export default SLAList;
