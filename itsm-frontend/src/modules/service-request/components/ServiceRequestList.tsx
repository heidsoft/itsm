'use client';

/**
 * 服务请求列表组件
 * 包含"我的请求"和"待办审批"两个视图
 */

import React, { useState, useEffect } from 'react';
import {
    Table, Tag, Button, Tabs, Card, Space, Tooltip,
    message
} from 'antd';
import {
    EyeOutlined, CheckCircleOutlined,
    SyncOutlined
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ServiceRequestApi } from '../api';
import { ServiceRequestStatus } from '../constants';
import type { ServiceRequest, ServiceRequestQuery } from '../types';

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
            // console.error(error);
            message.error('加载服务请求失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
        // eslint-disable-next-line react-hooks/exhaustive-deps
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
                <div className="flex flex-col">
                    <span className="font-medium text-gray-900">{text || `请求 #${record.id}`}</span>
                    <span className="text-xs text-gray-500">
                        {record.catalog?.name || '未知服务'}
                    </span>
                </div>
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
                            className="text-blue-600 hover:text-blue-800 hover:bg-blue-50"
                            onClick={() => router.push(`/service-requests/${record.id}`)}
                        />
                    </Tooltip>
                    {activeTab === 'approvals' && (
                        <Tooltip title="审批">
                            <Button
                                type="text"
                                icon={<CheckCircleOutlined />}
                                className="text-green-600 hover:text-green-800 hover:bg-green-50"
                                onClick={() => router.push(`/service-requests/${record.id}`)}
                            />
                        </Tooltip>
                    )}
                </Space>
            ),
        },
    ];

    return (
        <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
            <div className="flex justify-between items-center mb-4">
                <Tabs 
                    activeKey={activeTab} 
                    onChange={handleTabChange} 
                    size="large"
                    className="flex-1"
                    items={[
                        { label: '我的请求', key: 'my-requests' },
                        { label: '待办审批', key: 'approvals' },
                    ]}
                />
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
