'use client';

/**
 * SLA 详情组件
 */

import React, { useState, useEffect } from 'react';
import {
    Card, Tag, Button, Skeleton, Result,
    Descriptions, Space, Breadcrumb, message, Divider
} from 'antd';
import { ArrowLeftOutlined, EditOutlined } from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';

import { SLAApi } from '../api';
import { SLAPriorityLabels, SLAPriorityColors } from '../constants';
import type { SLADefinition } from '../types';

const SLADetail: React.FC = () => {
    const { id } = useParams() as { id: string };
    const router = useRouter();
    const [loading, setLoading] = useState(true);
    const [data, setData] = useState<SLADefinition | null>(null);

    useEffect(() => {
        if (id) {
            loadDetail();
        }
    }, [id]);

    const loadDetail = async () => {
        setLoading(true);
        try {
            const res = await SLAApi.getDefinition(id!);
            setData(res);
        } catch (error) {
            console.error(error);
            message.error('加载 SLA 详情失败');
        } finally {
            setLoading(false);
        }
    };

    if (loading) return <Card><Skeleton active /></Card>;

    if (!data) {
        return (
            <Card>
                <Result
                    status="404"
                    title="404"
                    subTitle="抱歉，您访问的 SLA 定义不存在"
                    extra={<Button type="primary" onClick={() => router.push('/sla')}>返回列表</Button>}
                />
            </Card>
        );
    }

    return (
        <div style={{ padding: '0 0 24px' }}>
            <Breadcrumb style={{ marginBottom: 16 }}>
                <Breadcrumb.Item>首页</Breadcrumb.Item>
                <Breadcrumb.Item>服务级别管理</Breadcrumb.Item>
                <Breadcrumb.Item onClick={() => router.push('/sla')}>SLA 定义</Breadcrumb.Item>
                <Breadcrumb.Item>详情</Breadcrumb.Item>
            </Breadcrumb>

            <Card
                title={data.name}
                extra={
                    <Button
                        type="primary"
                        icon={<EditOutlined />}
                        onClick={() => router.push(`/sla/definitions/${data.id}/edit`)}
                    >
                        编辑
                    </Button>
                }
            >
                <Descriptions bordered column={2}>
                    <Descriptions.Item label="名称">{data.name}</Descriptions.Item>
                    <Descriptions.Item label="服务类型">{data.service_type || '-'}</Descriptions.Item>
                    <Descriptions.Item label="优先级">
                        <Tag color={SLAPriorityColors[data.priority] || 'blue'}>
                            {SLAPriorityLabels[data.priority] || data.priority}
                        </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="状态">
                        {data.is_active ? <Tag color="green">激活</Tag> : <Tag color="red">禁用</Tag>}
                    </Descriptions.Item>
                    <Descriptions.Item label="响应时间设定">{data.response_time} 分钟</Descriptions.Item>
                    <Descriptions.Item label="解决时间设定">{data.resolution_time} 分钟</Descriptions.Item>
                    <Descriptions.Item label="描述" span={2}>{data.description || '无'}</Descriptions.Item>
                </Descriptions>

                <Divider orientation="left">适用条件</Divider>
                <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
                    {JSON.stringify(data.conditions, null, 2)}
                </pre>

                <Divider orientation="left">升级规则</Divider>
                <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
                    {JSON.stringify(data.escalation_rules, null, 2)}
                </pre>

                <div style={{ marginTop: 24 }}>
                    <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/sla')}>
                        返回列表
                    </Button>
                </div>
            </Card>
        </div>
    );
};

export default SLADetail;
