'use client';

/**
 * 变更详情组件
 */

import React, { useState, useEffect } from 'react';
import {
    Card, Descriptions, Tag, Button,
    Skeleton, Result, Divider, List, Typography,
    Steps, Spin, Empty, Tabs, Space
} from 'antd';
import { ArrowLeftOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ChangeApi } from '../api';
import {
    ChangeStatus, ChangeStatusLabels,
    ChangeTypeLabels, ChangePriorityLabels,
    ChangeImpactLabels, ChangeRiskLabels
} from '../constants';
import type { Change, ApprovalRecord } from '../types';

const { Title, Text, Paragraph } = Typography;

const statusColors: Record<string, string> = {
    [ChangeStatus.DRAFT]: 'default',
    [ChangeStatus.PENDING]: 'orange',
    [ChangeStatus.APPROVED]: 'cyan',
    [ChangeStatus.IN_PROGRESS]: 'blue',
    [ChangeStatus.COMPLETED]: 'green',
    [ChangeStatus.REJECTED]: 'red',
    [ChangeStatus.ROLLED_BACK]: 'magenta',
};

const ChangeDetail: React.FC = () => {
    const { id } = useParams() as { id: string };
    const router = useRouter();
    const [loading, setLoading] = useState(true);
    const [change, setChange] = useState<Change | null>(null);
    const [approvals, setApprovals] = useState<ApprovalRecord[]>([]);

    useEffect(() => {
        if (id) {
            loadDetail();
        }
    }, [id]);

    const loadDetail = async () => {
        setLoading(true);
        try {
            const data = await ChangeApi.getChange(id!);
            setChange(data);

            // Try to load approval summary
            try {
                const summary = await ChangeApi.getApprovalSummary(Number(id));
                setApprovals(summary.history || []);
            } catch (e) {
                console.warn('Failed to load approval summary', e);
            }
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    };

    if (loading) return <Card><Skeleton active /></Card>;

    if (!change) {
        return (
            <Card>
                <Result
                    status="404"
                    title="404"
                    subTitle="抱歉，您访问的变更不存在"
                    extra={<Button type="primary" onClick={() => router.push('/changes')}>返回列表</Button>}
                />
            </Card>
        );
    }

    return (
        <Space direction="vertical" style={{ width: '100%' }} size="large">
            <Card>
                <div style={{ marginBottom: 24 }}>
                    <Button
                        icon={<ArrowLeftOutlined />}
                        onClick={() => router.push('/changes')}
                        style={{ marginBottom: 16 }}
                    >
                        返回列表
                    </Button>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                        <Title level={3}>{change.title}</Title>
                        <Tag color={statusColors[change.status]} style={{ padding: '4px 12px', fontSize: 14 }}>
                            {ChangeStatusLabels[change.status]}
                        </Tag>
                    </div>
                </div>

                <Descriptions bordered column={2}>
                    <Descriptions.Item label="变更编号">{change.id}</Descriptions.Item>
                    <Descriptions.Item label="变更类型">{ChangeTypeLabels[change.type]}</Descriptions.Item>
                    <Descriptions.Item label="优先级">{ChangePriorityLabels[change.priority]}</Descriptions.Item>
                    <Descriptions.Item label="风险等级">{ChangeRiskLabels[change.risk_level]}</Descriptions.Item>
                    <Descriptions.Item label="影响范围">{ChangeImpactLabels[change.impact_scope]}</Descriptions.Item>
                    <Descriptions.Item label="负责人">{change.assignee_name || '未分配'}</Descriptions.Item>
                    <Descriptions.Item label="计划起始">{change.planned_start_date ? dayjs(change.planned_start_date).format('YYYY-MM-DD HH:mm') : '-'}</Descriptions.Item>
                    <Descriptions.Item label="计划截止">{change.planned_end_date ? dayjs(change.planned_end_date).format('YYYY-MM-DD HH:mm') : '-'}</Descriptions.Item>
                </Descriptions>

                <Tabs defaultActiveKey="1" style={{ marginTop: 24 }}>
                    <Tabs.TabPane tab="基础信息" key="1">
                        <Title level={5}>变更原因 / 理由</Title>
                        <Paragraph>{change.justification || '无'}</Paragraph>

                        <Title level={5}>变更描述</Title>
                        <Paragraph>{change.description || '无'}</Paragraph>

                        <Divider />

                        <Title level={5}>实施计划</Title>
                        <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{change.implementation_plan || '未提供实施计划'}</Paragraph>

                        <Title level={5}>回滚计划</Title>
                        <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{change.rollback_plan || '未提供回滚计划'}</Paragraph>
                    </Tabs.TabPane>

                    <Tabs.TabPane tab="审批记录" key="2">
                        {approvals.length > 0 ? (
                            <List
                                itemLayout="horizontal"
                                dataSource={approvals}
                                renderItem={record => (
                                    <List.Item>
                                        <List.Item.Meta
                                            avatar={record.status === 'approved' ?
                                                <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 24 }} /> :
                                                <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 24 }} />
                                            }
                                            title={
                                                <Space>
                                                    <Text strong>{record.approver_name}</Text>
                                                    <Tag color={statusColors[record.status]}>{ChangeStatusLabels[record.status]}</Tag>
                                                    <Text type="secondary">{dayjs(record.created_at).format('YYYY-MM-DD HH:mm')}</Text>
                                                </Space>
                                            }
                                            description={record.comment || '无意见'}
                                        />
                                    </List.Item>
                                )}
                            />
                        ) : (
                            <Empty description="暂无审批记录" />
                        )}
                    </Tabs.TabPane>
                </Tabs>
            </Card>
        </Space>
    );
};

export default ChangeDetail;
