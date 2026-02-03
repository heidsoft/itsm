'use client';

import React from 'react';
import { Card, Col, Row, Statistic, Typography, theme, Avatar } from 'antd';
import { Users, Workflow, BookOpen, AlertCircle, TrendingUp, BarChart3 } from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import { AdminStats } from '../hooks/useAdminData';

const { Paragraph, Title } = Typography;

interface SystemOverviewProps {
    stats?: AdminStats;
    loading?: boolean;
}

export const SystemOverview: React.FC<SystemOverviewProps> = ({ stats, loading }) => {
    const { t } = useI18n();
    const systemStats = [
        {
            title: t('admin.activeUsers'),
            value: stats?.activeUsers ?? '1,234',
            change: '+12%',
            changeValue: '+132',
            icon: Users,
            color: 'blue',
            trend: 'up',
            description: t('admin.newUsersDescription', { count: 132 }),
        },
        {
            title: t('admin.runningWorkflows'),
            value: stats?.runningWorkflows ?? '45',
            change: '+6.7%',
            changeValue: '+3',
            icon: Workflow,
            color: 'green',
            trend: 'up',
            description: t('admin.newWorkflowsDescription', { count: 3 }),
        },
        {
            title: t('admin.serviceCatalogItems'),
            value: stats?.serviceCatalogItems ?? '89',
            change: '+5.9%',
            changeValue: '+5',
            icon: BookOpen,
            color: 'purple',
            trend: 'up',
            description: t('admin.newServiceItemsDescription', { count: 5 }),
        },
        {
            title: t('admin.systemAlerts'),
            value: stats?.systemAlerts ?? '2',
            change: '-33%',
            changeValue: '-1',
            icon: AlertCircle,
            color: 'red',
            trend: 'down',
            description: t('admin.alertsReducedDescription', { count: 1 }),
        },
    ];

    const EnhancedStatCard = ({ stat }: { stat: (typeof systemStats)[0] }) => {
        const { token } = theme.useToken();
        const Icon = stat.icon;
        const isPositive = stat.trend === 'up';

        const getColorByName = (colorName: string) => {
            switch (colorName) {
                case 'blue':
                    return token.colorPrimary;
                case 'green':
                    return token.colorSuccess;
                case 'purple':
                    return '#722ed1';
                case 'red':
                    return token.colorError;
                default:
                    return token.colorPrimary;
            }
        };

        const trendColor = isPositive ? token.colorSuccess : token.colorError;

        return (
            <Card hoverable style={{ height: '100%' }}>
                <div
                    style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'flex-start',
                        marginBottom: token.marginLG,
                    }}
                >
                    <Avatar
                        size={48}
                        style={{
                            backgroundColor: getColorByName(stat.color),
                            border: 'none',
                        }}
                        icon={<Icon className='w-6 h-6' />}
                    />
                    <div
                        style={{
                            color: trendColor,
                            fontSize: token.fontSizeSM,
                            fontWeight: 600,
                        }}
                    >
                        <TrendingUp className={`w-4 h-4 ${isPositive ? '' : 'rotate-180'}`} />
                        <span>{stat.change}</span>
                    </div>
                </div>

                <Statistic
                    title={stat.title}
                    value={stat.value}
                    loading={loading}
                    styles={{ content: { color: token.colorText, fontSize: 28, fontWeight: 700 } }}
                />
                <Paragraph
                    type='secondary'
                    style={{ marginTop: token.marginSM, marginBottom: 0 }}
                >
                    {stat.description}
                </Paragraph>
            </Card>
        );
    };

    return (
        <div style={{ marginBottom: 24 }}>
          <Title level={3} style={{ marginBottom: 24 }}>
            <BarChart3
              className="w-5 h-5"
              style={{ marginRight: 8, color: '#1890ff' }}
            />
            {t('admin.systemOverview')}
          </Title>
            <Row gutter={[16, 16]}>
                {systemStats.map((stat, index) => (
                    <Col xs={24} sm={12} lg={6} key={index}>
                        <EnhancedStatCard stat={stat} />
                    </Col>
                ))}
            </Row>
        </div>
    );
};
