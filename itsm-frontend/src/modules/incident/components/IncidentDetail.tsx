'use client';

/**
 * 事件详情组件
 */

import React, { useState, useEffect } from 'react';
import {
    Card, Descriptions, Tag, Button, Space,
    Timeline, Skeleton, message, Divider, Modal
} from 'antd';
import {
    EditOutlined, ArrowUpOutlined, CheckCircleOutlined,
    ClockCircleOutlined, ExclamationCircleOutlined
} from '@ant-design/icons';
import { useRouter, useParams } from 'next/navigation';
import dayjs from 'dayjs';

import { IncidentApi } from '../api';
import {
    IncidentStatus, IncidentStatusLabels,
    IncidentPriorityLabels, IncidentSeverityLabels
} from '../constants';
import type { Incident } from '../types';

const IncidentDetail: React.FC = () => {
    const { id } = useParams() as { id: string };
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<Incident | null>(null);

    const loadData = async () => {
        if (!id) return;
        setLoading(true);
        try {
            const resp = await IncidentApi.getIncident(id);
            setData(resp);
        } catch (error) {
            console.error(error);
            message.error('加载事件详情失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
    }, [id]);

    const handleEscalate = () => {
        // TODO: 实现升级弹窗
        message.info('功能开发中');
    };

    if (loading) {
        return <Card bordered={false}><Skeleton active /></Card>;
    }

    if (!data) {
        return <Card bordered={false}>未找到事件</Card>;
    }

    return (
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
            {/* 头部操作栏 */}
            <Card bordered={false} bodyStyle={{ padding: '16px 24px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                        <span style={{ fontSize: 20, fontWeight: 500, marginRight: 16 }}>
                            {data.incident_number} {data.title}
                        </span>
                        <Tag color={data.status === IncidentStatus.RESOLVED ? 'success' : 'blue'}>
                            {IncidentStatusLabels[data.status]}
                        </Tag>
                    </div>
                    <Space>
                        <Button
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/incidents/${data.id}/edit`)}
                        >
                            编辑
                        </Button>
                        <Button
                            icon={<ArrowUpOutlined />}
                            onClick={handleEscalate}
                        >
                            升级
                        </Button>
                        {data.status !== IncidentStatus.RESOLVED && (
                            <Button type="primary" icon={<CheckCircleOutlined />}>
                                解决
                            </Button>
                        )}
                    </Space>
                </div>
            </Card>

            {/* 基本信息 */}
            <Card bordered={false} title="基本信息">
                <Descriptions column={2}>
                    <Descriptions.Item label="报告人ID">{data.reporter_id}</Descriptions.Item>
                    <Descriptions.Item label="负责人ID">{data.assignee_id || '-'}</Descriptions.Item>
                    <Descriptions.Item label="优先级">{IncidentPriorityLabels[data.priority]}</Descriptions.Item>
                    <Descriptions.Item label="严重程度">{IncidentSeverityLabels[data.severity]}</Descriptions.Item>
                    <Descriptions.Item label="分类">{data.category}</Descriptions.Item>
                    <Descriptions.Item label="子分类">{data.subcategory}</Descriptions.Item>
                    <Descriptions.Item label="检测时间">
                        {dayjs(data.detected_at).format('YYYY-MM-DD HH:mm:ss')}
                    </Descriptions.Item>
                    <Descriptions.Item label="来源">{data.source}</Descriptions.Item>
                </Descriptions>
                <Divider />
                <Descriptions title="详细描述" column={1}>
                    <Descriptions.Item label="描述">
                        {data.description}
                    </Descriptions.Item>
                </Descriptions>

                {/* 影响分析 (如果有) */}
                {data.impact_analysis && (
                    <>
                        <Divider />
                        <Descriptions title="影响分析" column={1}>
                            <Descriptions.Item>
                                <pre>{JSON.stringify(data.impact_analysis, null, 2)}</pre>
                            </Descriptions.Item>
                        </Descriptions>
                    </>
                )}
            </Card>

            {/* 解决记录 (如果有) */}
            {(data.resolution_steps && data.resolution_steps.length > 0) && (
                <Card bordered={false} title="处理流程">
                    <Timeline>
                        {data.resolution_steps.map((step, index) => (
                            <Timeline.Item key={index}>
                                <p>{(step as any).description || '处理步骤'}</p>
                                <span style={{ fontSize: '12px', color: '#999' }}>
                                    {(step as any).timestamp}
                                </span>
                            </Timeline.Item>
                        ))}
                    </Timeline>
                </Card>
            )}
        </Space>
    );
};

export default IncidentDetail;
