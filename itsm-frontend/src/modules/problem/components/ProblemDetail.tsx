'use client';

/**
 * 问题详情组件
 */

import React, { useState, useEffect } from 'react';
import {
    Card, Descriptions, Tag, Button, Space,
    Skeleton, message, Divider, Typography
} from 'antd';
import {
    EditOutlined, ArrowLeftOutlined,
    CheckCircleOutlined, SyncOutlined
} from '@ant-design/icons';
import { useRouter, useParams } from 'next/navigation';
import dayjs from 'dayjs';

import { ProblemApi } from '../api';
import {
    ProblemStatus, ProblemStatusLabels,
    ProblemPriorityLabels
} from '../constants';
import type { Problem } from '../types';

const { Title, Paragraph } = Typography;

const ProblemDetail: React.FC = () => {
    const { id } = useParams() as { id: string };
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<Problem | null>(null);

    const loadData = async () => {
        if (!id) return;
        setLoading(true);
        try {
            const problem = await ProblemApi.getProblem(id);
            setData(problem);
        } catch (error) {
            // console.error(error);
            message.error('加载问题详情失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id]);

    const handleUpdateStatus = async (status: ProblemStatus) => {
        if (!id) return;
        try {
            await ProblemApi.updateProblem(id, { status });
            message.success('状态更新成功');
            loadData();
        } catch (error) {
            message.error('状态更新失败');
        }
    };

    if (loading) {
        return <Card variant="borderless"><Skeleton active /></Card>;
    }

    if (!data) {
        return <Card variant="borderless">未找到该问题</Card>;
    }

    return (
        <Space orientation="vertical" style={{ width: '100%' }} size="middle">
            {/* 操作栏 */}
            <Card variant="borderless" bodyStyle={{ padding: '16px 24px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Space>
                        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/problems')}>
                            返回列表
                        </Button>
                        <Title level={4} style={{ margin: 0 }}>{data.title}</Title>
                        <Tag color={data.status === ProblemStatus.RESOLVED ? 'success' : 'blue'}>
                            {ProblemStatusLabels[data.status]}
                        </Tag>
                    </Space>
                    <Space>
                        <Button
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/problems/${data.id}/edit`)}
                        >
                            编辑
                        </Button>
                        {data.status === ProblemStatus.OPEN && (
                            <Button type="primary" onClick={() => handleUpdateStatus(ProblemStatus.IN_PROGRESS)}>
                                开始处理
                            </Button>
                        )}
                        {data.status === ProblemStatus.IN_PROGRESS && (
                            <Button type="primary" onClick={() => handleUpdateStatus(ProblemStatus.RESOLVED)}>
                                标记解决
                            </Button>
                        )}
                        {data.status === ProblemStatus.RESOLVED && (
                            <Button onClick={() => handleUpdateStatus(ProblemStatus.CLOSED)}>
                                关闭问题
                            </Button>
                        )}
                    </Space>
                </div>
            </Card>

            <Card variant="borderless" title="基本信息">
                <Descriptions column={2}>
                    <Descriptions.Item label="创建人ID">{data.created_by}</Descriptions.Item>
                    <Descriptions.Item label="负责人ID">{data.assignee_id || '-'}</Descriptions.Item>
                    <Descriptions.Item label="优先级">{ProblemPriorityLabels[data.priority]}</Descriptions.Item>
                    <Descriptions.Item label="分类">{data.category}</Descriptions.Item>
                    <Descriptions.Item label="创建时间">
                        {dayjs(data.created_at).format('YYYY-MM-DD HH:mm:ss')}
                    </Descriptions.Item>
                    <Descriptions.Item label="更新时间">
                        {dayjs(data.updated_at).format('YYYY-MM-DD HH:mm:ss')}
                    </Descriptions.Item>
                </Descriptions>

                <Divider />

                <Title level={5}>描述</Title>
                <Paragraph>{data.description}</Paragraph>

                <Divider />

                <Title level={5}>根本原因</Title>
                <Paragraph>{data.root_cause || '暂无分析'}</Paragraph>

                <Divider />

                <Title level={5}>影响范围</Title>
                <Paragraph>{data.impact || '暂无描述'}</Paragraph>
            </Card>
        </Space>
    );
};

export default ProblemDetail;
