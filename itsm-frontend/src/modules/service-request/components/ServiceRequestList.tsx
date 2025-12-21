'use client';

/**
 * 服务请求列表组件
 * 包含"我的请求"和"待办审批"两个视图
 */

import React, { useState, useEffect } from 'react';
import {
    Table, Tag, Button, Tabs, Card, Space, Tooltip,
    Pagination, Input, Select, DatePicker, message
} from 'antd';
import {
    SearchOutlined, EyeOutlined, CheckCircleOutlined,
    SyncOutlined, ClockCircleOutlined
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ServiceRequestApi } from '../api';
import { ServiceRequestStatus, ApprovalStatus } from '../constants';
import type { ServiceRequest, ServiceRequestQuery } from '../types';

const { TabPane } = Tabs;
const { Option } = Select;

// 状态标签颜色映射
const statusColors: Record<string, string> = {
    [ServiceRequestStatus.SUBMITTED]: 'blue',
    [ServiceRequestStatus.MANAGER_APPROVED]: 'cyan',
    [ServiceRequestStatus.IT_APPROVED]: 'geekblue',
    [ServiceRequestStatus.SECURITY_APPROVED]: 'purple',
    [ServiceRequestStatus.COMPLETED]: 'green',
    [ServiceRequestStatus.REJECTED]: 'red',
    [ServiceRequestStatus.CANCELLED]: 'default',
};

const ServiceRequestList: React.FC = () => {
    const router = useRouter();
    const [activeTab, setActiveTab] = useState('my-requests');
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<ServiceRequest[]>([]);
    const [total, setTotal] = useState(0);

    // 查询状态
    const [query, setQuery] = useState<ServiceRequestQuery>({
        page: 1,
        size: 10,
        scope: 'me',
    });

    // 加载数据
    const loadData = async () => {
        setLoading(true);
        try {
            let resp;
            if (activeTab === 'approvals') {
                // 待办审批
                resp = await ServiceRequestApi.getPendingApprovals({
                    ...query,
                    // status: ServiceRequestStatus.SUBMITTED // 或者根据逻辑传参
                });
            } else {
                // 我的请求
                resp = await ServiceRequestApi.getServiceRequests({
                    ...query,
                    scope: 'me'
                });
            }
            setData(resp.requests);
            setTotal(resp.total);
        } catch (error) {
            console.error(error);
            message.error('加载服务请求失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
    }, [query, activeTab]);

    // 处理Tab切换
    const handleTabChange = (key: string) => {
        setActiveTab(key);
        setQuery(prev => ({ ...prev, page: 1, scope: key === 'approvals' ? 'all' : 'me' }));
    };

    // 表格列定义
    const columns = [
        {
            title: 'ID',
            dataIndex: 'id',
            width: 80,
        },
        {
            title: '标题',
            dataIndex: 'title',
            render: (text: string, record: ServiceRequest) => (
                <Space direction="vertical" size={0}>
                    <span style={{ fontWeight: 500 }}>{text || `请求 #${record.id}`}</span>
                    <span style={{ fontSize: '12px', color: '#888' }}>
                        {record.catalog?.name || '未知服务'}
                    </span>
                </Space>
            ),
        },
        {
            title: '状态',
            dataIndex: 'status',
            width: 150,
            render: (status: string) => (
                <Tag color={statusColors[status] || 'default'}>
                    {status}
                </Tag>
            ),
        },
        {
            title: '当前步骤',
            dataIndex: 'current_level',
            width: 120,
            render: (level: number, record: ServiceRequest) => (
                <span>Level {level} / {record.total_levels}</span>
            ),
            responsive: ['md'],
        },
        {
            title: '提交时间',
            dataIndex: 'created_at',
            width: 180,
            render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
            responsive: ['sm'],
        },
        {
            title: '操作',
            key: 'action',
            width: 120,
            render: (_: any, record: ServiceRequest) => (
                <Space size="small">
                    <Tooltip title="查看详情">
                        <Button
                            type="text"
                            icon={<EyeOutlined />}
                            onClick={() => router.push(`/service-requests/${record.id}`)}
                        />
                    </Tooltip>
                    {activeTab === 'approvals' && (
                        <Tooltip title="审批">
                            <Button
                                type="text"
                                icon={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
                                onClick={() => router.push(`/service-requests/${record.id}`)}
                            />
                        </Tooltip>
                    )}
                </Space>
            ),
        },
    ];

    return (
        <Card bodyStyle={{ padding: '0 24px 24px' }} bordered={false}>
            <Tabs activeKey={activeTab} onChange={handleTabChange} size="large">
                <TabPane tab="我的请求" key="my-requests" />
                <TabPane tab="待办审批" key="approvals" />
            </Tabs>

            {/* 工具栏: 搜索和筛选 (后续实现) */}
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
                <Button icon={<SyncOutlined />} onClick={loadData}>刷新</Button>
            </div>

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

export default ServiceRequestList;
