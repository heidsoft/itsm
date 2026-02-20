'use client';

/**
 * 配置项 (CI) 详情组件
 */

import React, { useState, useEffect } from 'react';
import {
    Card, Descriptions, Tag, Button,
    Skeleton, Result, Divider, List, Typography,
    Tabs, Space, Breadcrumb, Empty, message, Table, Timeline, Alert
} from 'antd';
import {
    ArrowLeftOutlined, ClusterOutlined, HistoryOutlined,
    NodeIndexOutlined, LinkOutlined, WarningOutlined, InfoCircleOutlined
} from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { CMDBApi } from '@/lib/api/';
import { CIStatus, CIStatusLabels } from '@/constants/cmdb';
import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';
import TopologyGraph from './TopologyGraph';
import CIRelationshipManager from './CIRelationshipManager';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

// 影响分析数据类型
interface ImpactAnalysisData {
    target_ci: any;
    upstream_impact: ImpactAnalysisItem[];
    downstream_impact: ImpactAnalysisItem[];
    critical_dependencies: ImpactAnalysisItem[];
    affected_tickets: any[];
    affected_incidents: any[];
    risk_level: string;
    summary: string;
}

interface ImpactAnalysisItem {
    ci_id: number;
    ci_name: string;
    ci_type: string;
    relationship: string;
    impact_level: string;
    distance: number;
    direction: string;
}

// 变更历史数据类型
interface ChangeHistoryData {
    logs: any[];
    total: number;
    page: number;
    page_size: number;
}

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
    const [impactAnalysis, setImpactAnalysis] = useState<ImpactAnalysisData | null>(null);
    const [impactLoading, setImpactLoading] = useState(false);
    const [changeHistory, setChangeHistory] = useState<ChangeHistoryData | null>(null);
    const [historyLoading, setHistoryLoading] = useState(false);

    useEffect(() => {
        if (id) {
            loadDetail();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id]);

    const loadDetail = async () => {
        setLoading(true);
        try {
            const [ciData, typeData] = await Promise.all([
                CMDBApi.getCI(id!),
                CMDBApi.getTypes()
            ]);
            setCi(ciData as unknown as ConfigurationItem);
            setTypes(typeData || []);
        } catch (error) {
            // console.error(error);
            message.error('加载资产详情失败');
        } finally {
            setLoading(false);
        }
    };

    const loadImpactAnalysis = async () => {
        setImpactLoading(true);
        try {
            const data = await CMDBApi.getCIImpactAnalysis(Number(id));
            setImpactAnalysis(data);
        } catch (error) {
            console.error('加载影响分析失败:', error);
        } finally {
            setImpactLoading(false);
        }
    };

    const loadChangeHistory = async () => {
        setHistoryLoading(true);
        try {
            const data = await CMDBApi.getCIChangeHistory(Number(id));
            setChangeHistory(data);
        } catch (error) {
            console.error('加载变更历史失败:', error);
        } finally {
            setHistoryLoading(false);
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
            <Breadcrumb
                style={{ marginBottom: 16 }}
                items={[
                    { title: '首页' },
                    { title: '配置管理' },
                    { title: <a onClick={() => router.push('/cmdb')}>配置项列表</a> },
                    { title: '资产详情' },
                ]}
            />

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
                        <Space orientation="vertical" size={0}>
                            <Title level={3} style={{ margin: 0 }}>{ci.name}</Title>
                            <Text type="secondary">配置项 ID: {ci.id}</Text>
                        </Space>
                        <Tag color={statusColors[ci.status]} style={{ padding: '4px 12px', fontSize: 14 }}>
                            {CIStatusLabels[ci.status] || ci.status}
                        </Tag>
                    </div>
                </div>

                <Tabs defaultActiveKey="basic">
                    <TabPane tab="基本信息" key="basic">
                        <Descriptions bordered column={2}>
                            <Descriptions.Item label="资产分类">{typeInfo?.name || `类型 ${ci.ci_type_id}`}</Descriptions.Item>
                            <Descriptions.Item label="资产标签">{ci.asset_tag || '-'}</Descriptions.Item>
                            <Descriptions.Item label="序列号">{ci.serial_number || '-'}</Descriptions.Item>
                            <Descriptions.Item label="型号">{ci.model || '-'}</Descriptions.Item>
                            <Descriptions.Item label="厂商">{ci.vendor || '-'}</Descriptions.Item>
                            <Descriptions.Item label="所在位置">{ci.location || '-'}</Descriptions.Item>
                            <Descriptions.Item label="环境">{ci.environment || '-'}</Descriptions.Item>
                            <Descriptions.Item label="重要性">{ci.criticality || '-'}</Descriptions.Item>
                            <Descriptions.Item label="分配给">{ci.assigned_to || '-'}</Descriptions.Item>
                            <Descriptions.Item label="拥有者">{ci.owned_by || '-'}</Descriptions.Item>
                            <Descriptions.Item label="发现源">{ci.discovery_source || '-'}</Descriptions.Item>
                            <Descriptions.Item label="数据来源">{ci.source || '-'}</Descriptions.Item>
                            <Descriptions.Item label="云厂商">{ci.cloud_provider || '-'}</Descriptions.Item>
                            <Descriptions.Item label="云账号">{ci.cloud_account_id || '-'}</Descriptions.Item>
                            <Descriptions.Item label="Region">{ci.cloud_region || '-'}</Descriptions.Item>
                            <Descriptions.Item label="Zone">{ci.cloud_zone || '-'}</Descriptions.Item>
                            <Descriptions.Item label="云资源ID">{ci.cloud_resource_id || '-'}</Descriptions.Item>
                            <Descriptions.Item label="云资源类型">{ci.cloud_resource_type || '-'}</Descriptions.Item>
                            <Descriptions.Item label="云资源引用ID">{ci.cloud_resource_ref_id || '-'}</Descriptions.Item>
                            <Descriptions.Item label="同步状态">{ci.cloud_sync_status || '-'}</Descriptions.Item>
                            <Descriptions.Item label="同步时间">
                                {ci.cloud_sync_time ? dayjs(ci.cloud_sync_time).format('YYYY-MM-DD HH:mm:ss') : '-'}
                            </Descriptions.Item>
                            <Descriptions.Item label="所属租户">{ci.tenant_id}</Descriptions.Item>
                            <Descriptions.Item label="创建时间">{dayjs(ci.created_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
                            <Descriptions.Item label="最后更新">{dayjs(ci.updated_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
                            <Descriptions.Item label="描述" span={2}>{ci.description || '无'}</Descriptions.Item>
                        </Descriptions>
                    </TabPane>

                    <TabPane tab={<span><NodeIndexOutlined /> 服务拓扑</span>} key="topology">
                        <TopologyGraph ciId={ci.id} initialDepth={2} height={500} />
                    </TabPane>

                    <TabPane tab={<span><LinkOutlined /> 关系管理</span>} key="relationships">
                        <CIRelationshipManager
                            ciId={ci.id}
                            ciName={ci.name}
                            onRefresh={() => {
                                // 刷新时可以触发拓扑图刷新
                            }}
                        />
                    </TabPane>

                    <TabPane tab={<span><ClusterOutlined /> 影响分析</span>} key="impact">
                        <Card title="CI影响分析" size="small">
                            <Button
                                type="primary"
                                onClick={loadImpactAnalysis}
                                loading={impactLoading}
                                style={{ marginBottom: 16 }}
                            >
                                执行影响分析
                            </Button>

                            {impactAnalysis && (
                                <>
                                    <Alert
                                        message="风险等级"
                                        description={
                                            <Space>
                                                <Tag color={
                                                    impactAnalysis.risk_level === 'critical' ? 'red' :
                                                    impactAnalysis.risk_level === 'high' ? 'orange' :
                                                    impactAnalysis.risk_level === 'medium' ? 'gold' : 'green'
                                                }>
                                                    {impactAnalysis.risk_level === 'critical' ? '严重' :
                                                     impactAnalysis.risk_level === 'high' ? '高' :
                                                     impactAnalysis.risk_level === 'medium' ? '中' : '低'}
                                                </Tag>
                                                <Text>{impactAnalysis.summary}</Text>
                                            </Space>
                                        }
                                        type={
                                            impactAnalysis.risk_level === 'critical' ? 'error' :
                                            impactAnalysis.risk_level === 'high' ? 'warning' : 'info'
                                        }
                                        style={{ marginBottom: 16 }}
                                    />

                                    <Tabs
                                        items={[
                                            {
                                                key: 'upstream',
                                                label: `上游影响 (${impactAnalysis.upstream_impact?.length || 0})`,
                                                children: (
                                                    impactAnalysis.upstream_impact && impactAnalysis.upstream_impact.length > 0 ? (
                                                        <List
                                                            size="small"
                                                            dataSource={impactAnalysis.upstream_impact}
                                                            renderItem={(item: ImpactAnalysisItem) => (
                                                                <List.Item>
                                                                    <Space>
                                                                        <Tag color={item.impact_level === 'critical' ? 'red' : item.impact_level === 'high' ? 'orange' : 'blue'}>
                                                                            {item.impact_level}
                                                                        </Tag>
                                                                        <Text>{item.ci_name}</Text>
                                                                        <Text type="secondary">({item.ci_type})</Text>
                                                                        <Text type="secondary">- {item.relationship}</Text>
                                                                    </Space>
                                                                </List.Item>
                                                            )}
                                                        />
                                                    ) : (
                                                        <Empty description="无上游依赖" />
                                                    )
                                                ),
                                            },
                                            {
                                                key: 'downstream',
                                                label: `下游影响 (${impactAnalysis.downstream_impact?.length || 0})`,
                                                children: (
                                                    impactAnalysis.downstream_impact && impactAnalysis.downstream_impact.length > 0 ? (
                                                        <List
                                                            size="small"
                                                            dataSource={impactAnalysis.downstream_impact}
                                                            renderItem={(item: ImpactAnalysisItem) => (
                                                                <List.Item>
                                                                    <Space>
                                                                        <Tag color={item.impact_level === 'critical' ? 'red' : item.impact_level === 'high' ? 'orange' : 'blue'}>
                                                                            {item.impact_level}
                                                                        </Tag>
                                                                        <Text>{item.ci_name}</Text>
                                                                        <Text type="secondary">({item.ci_type})</Text>
                                                                        <Text type="secondary">- {item.relationship}</Text>
                                                                    </Space>
                                                                </List.Item>
                                                            )}
                                                        />
                                                    ) : (
                                                        <Empty description="无下游影响" />
                                                    )
                                                ),
                                            },
                                            {
                                                key: 'tickets',
                                                label: `关联工单 (${impactAnalysis.affected_tickets?.length || 0})`,
                                                children: (
                                                    impactAnalysis.affected_tickets && impactAnalysis.affected_tickets.length > 0 ? (
                                                        <List
                                                            size="small"
                                                            dataSource={impactAnalysis.affected_tickets}
                                                            renderItem={(item: any) => (
                                                                <List.Item>
                                                                    <Space>
                                                                        <Tag>{item.status}</Tag>
                                                                        <Text>{item.title}</Text>
                                                                        <Text type="secondary">优先级: {item.priority}</Text>
                                                                    </Space>
                                                                </List.Item>
                                                            )}
                                                        />
                                                    ) : (
                                                        <Empty description="无关联工单" />
                                                    )
                                                ),
                                            },
                                            {
                                                key: 'incidents',
                                                label: `关联事件 (${impactAnalysis.affected_incidents?.length || 0})`,
                                                children: (
                                                    impactAnalysis.affected_incidents && impactAnalysis.affected_incidents.length > 0 ? (
                                                        <List
                                                            size="small"
                                                            dataSource={impactAnalysis.affected_incidents}
                                                            renderItem={(item: any) => (
                                                                <List.Item>
                                                                    <Space>
                                                                        <Tag color={item.severity === 'critical' ? 'red' : item.severity === 'high' ? 'orange' : 'blue'}>
                                                                            {item.severity}
                                                                        </Tag>
                                                                        <Text>{item.title}</Text>
                                                                        <Tag>{item.status}</Tag>
                                                                    </Space>
                                                                </List.Item>
                                                            )}
                                                        />
                                                    ) : (
                                                        <Empty description="无关联事件" />
                                                    )
                                                ),
                                            },
                                        ]}
                                    />
                                </>
                            )}

                            {!impactAnalysis && !impactLoading && (
                                <Empty description="点击上方按钮执行影响分析" />
                            )}
                        </Card>
                    </TabPane>

                    <TabPane tab={<span><HistoryOutlined /> 变更历史</span>} key="history">
                        <Card title="CI变更历史" size="small">
                            <Button
                                onClick={loadChangeHistory}
                                loading={historyLoading}
                                style={{ marginBottom: 16 }}
                            >
                                加载变更历史
                            </Button>

                            {changeHistory && changeHistory.logs && changeHistory.logs.length > 0 ? (
                                <Timeline
                                    items={changeHistory.logs.map((log: any) => ({
                                        color: log.action === 'create' ? 'green' : log.action === 'update' ? 'blue' : 'gray',
                                        children: (
                                            <div>
                                                <Space>
                                                    <Tag color={
                                                        log.action === 'create' ? 'green' :
                                                        log.action === 'update' ? 'blue' :
                                                        log.action === 'delete' ? 'red' : 'default'
                                                    }>
                                                        {log.action || log.Method}
                                                    </Tag>
                                                    <Text>{log.resource}</Text>
                                                </Space>
                                                <div>
                                                    <Text type="secondary">
                                                        {log.path} - {log.Method} - {log.StatusCode}
                                                    </Text>
                                                </div>
                                                <div>
                                                    <Text type="secondary">
                                                        {dayjs(log.created_at).format('YYYY-MM-DD HH:mm:ss')}
                                                    </Text>
                                                </div>
                                            </div>
                                        ),
                                    }))}
                                />
                            ) : (
                                <Empty description={historyLoading ? "加载中..." : "暂无历史审计记录"} />
                            )}
                        </Card>
                    </TabPane>
                </Tabs>
            </Card>
        </div>
    );
};

export default CIDetail;
