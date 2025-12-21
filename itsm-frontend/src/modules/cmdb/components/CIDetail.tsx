'use client';

/**
 * 配置项 (CI) 详情组件
 */

import React, { useState, useEffect } from 'react';
import {
    Card, Descriptions, Tag, Button,
    Skeleton, Result, Divider, List, Typography,
    Tabs, Space, Breadcrumb, Empty, message
} from 'antd';
import { ArrowLeftOutlined, ClusterOutlined, HistoryOutlined } from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { CMDBApi } from '../api';
import { CIStatus, CIStatusLabels } from '../constants';
import type { ConfigurationItem, CIType } from '../types';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

const statusColors: Record<string, string> = {
    [CIStatus.ACTIVE]: 'green',
    [CIStatus.INACTIVE]: 'default',
    [CIStatus.MAINTENANCE]: 'orange',
    [CIStatus.DECOMMISSIONED]: 'red',
};

const CIDetail: React.FC = () => {
    const { id } = useParams() as { id: string };
    const router = useRouter();
    const [loading, setLoading] = useState(true);
    const [ci, setCi] = useState<ConfigurationItem | null>(null);
    const [types, setTypes] = useState<CIType[]>([]);

    useEffect(() => {
        if (id) {
            loadDetail();
        }
    }, [id]);

    const loadDetail = async () => {
        setLoading(true);
        try {
            const [ciData, typeData] = await Promise.all([
                CMDBApi.getCI(id!),
                CMDBApi.getTypes()
            ]);
            setCi(ciData);
            setTypes(typeData || []);
        } catch (error) {
            console.error(error);
            message.error('加载资产详情失败');
        } finally {
            setLoading(false);
        }
    };

    if (loading) return <Card><Skeleton active /></Card>;

    if (!ci) {
        return (
            <Card>
                <Result
                    status="404"
                    title="404"
                    subTitle="抱歉，您访问的配置项不存在"
                    extra={<Button type="primary" onClick={() => router.push('/cmdb')}>返回列表</Button>}
                />
            </Card>
        );
    }

    const typeInfo = types.find(t => t.id === ci.ci_type_id);

    return (
        <div style={{ padding: '0 0 24px' }}>
            <Breadcrumb style={{ marginBottom: 16 }}>
                <Breadcrumb.Item>首页</Breadcrumb.Item>
                <Breadcrumb.Item>配置管理</Breadcrumb.Item>
                <Breadcrumb.Item onClick={() => router.push('/cmdb')}>配置项列表</Breadcrumb.Item>
                <Breadcrumb.Item>资产详情</Breadcrumb.Item>
            </Breadcrumb>

            <Card>
                <div style={{ marginBottom: 24 }}>
                    <Button
                        icon={<ArrowLeftOutlined />}
                        onClick={() => router.push('/cmdb')}
                        style={{ marginBottom: 16 }}
                    >
                        返回列表
                    </Button>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                        <Space direction="vertical" size={0}>
                            <Title level={3} style={{ margin: 0 }}>{ci.name}</Title>
                            <Text type="secondary">配置项 ID: {ci.id}</Text>
                        </Space>
                        <Tag color={statusColors[ci.status]} style={{ padding: '4px 12px', fontSize: 14 }}>
                            {CIStatusLabels[ci.status] || ci.status}
                        </Tag>
                    </div>
                </div>

                <Tabs defaultActiveKey="1">
                    <TabPane tab="基础信息" key="1">
                        <Descriptions bordered column={2}>
                            <Descriptions.Item label="资产分类">{typeInfo?.name || `类型 ${ci.ci_type_id}`}</Descriptions.Item>
                            <Descriptions.Item label="序列号">{ci.serial_number || '-'}</Descriptions.Item>
                            <Descriptions.Item label="型号">{ci.model || '-'}</Descriptions.Item>
                            <Descriptions.Item label="厂商">{ci.vendor || '-'}</Descriptions.Item>
                            <Descriptions.Item label="所在位置">{ci.location || '-'}</Descriptions.Item>
                            <Descriptions.Item label="所属租户">{ci.tenant_id}</Descriptions.Item>
                            <Descriptions.Item label="创建时间">{dayjs(ci.created_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
                            <Descriptions.Item label="最后更新">{dayjs(ci.updated_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
                            <Descriptions.Item label="描述" span={2}>{ci.description || '无'}</Descriptions.Item>
                        </Descriptions>
                    </TabPane>

                    <TabPane tab={<span><ClusterOutlined /> 拓扑关系</span>} key="2">
                        <Empty description="暂无关联资产信息" style={{ margin: '40px 0' }} />
                    </TabPane>

                    <TabPane tab={<span><HistoryOutlined /> 变更历史</span>} key="3">
                        <Empty description="暂无历史审计记录" style={{ margin: '40px 0' }} />
                    </TabPane>
                </Tabs>
            </Card>
        </div>
    );
};

export default CIDetail;
