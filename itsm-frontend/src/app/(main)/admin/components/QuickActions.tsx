'use client';

import React from 'react';
import Link from 'next/link';
import { Card, Row, Col, Typography, Space, Avatar, theme, Tooltip } from 'antd';
import {
  Users,
  Shield,
  Workflow,
  Bell,
  BookOpen,
  Database,
  Settings,
  Zap,
  FileText,
  Calendar,
  Mail,
  Globe,
  ArrowUpRight,
} from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;

export const QuickActions: React.FC = () => {
  const { token } = theme.useToken();
  const { t } = useI18n();

  const quickActionGroups = {
    userManagement: {
      title: t('admin.userManagement'),
      description: t('admin.manageUsers'),
      actions: [
        {
          title: t('admin.users'),
          description: t('admin.userAccounts'),
          href: '/admin/users',
          icon: Users,
          color: 'bg-blue-500',
          stats: t('admin.usersCount', { count: 1234 }),
        },
        {
          title: t('admin.roles'),
          description: t('admin.rolePermissions'),
          href: '/admin/roles',
          icon: Shield,
          color: 'bg-indigo-500',
          stats: t('admin.rolesCount', { count: 15 }),
        },
        {
          title: t('admin.userGroups'),
          description: t('admin.userOrganization'),
          href: '/admin/groups',
          icon: Users,
          color: 'bg-cyan-500',
          stats: t('admin.userGroupsCount', { count: 28 }),
        },
        {
          title: t('admin.permissions'),
          description: t('admin.permissionMatrix'),
          href: '/admin/permissions',
          icon: Shield,
          color: 'bg-purple-500',
          stats: t('admin.permissionsCount', { count: 156 }),
        },
      ],
    },
    processConfig: {
      title: t('admin.processConfig'),
      description: t('admin.processAutomation'),
      actions: [
        {
          title: t('admin.workflows'),
          description: t('admin.workflowDesign'),
          href: '/admin/workflows',
          icon: Workflow,
          color: 'bg-green-500',
          stats: t('admin.workflowsCount', { count: 45 }),
        },
        {
          title: t('admin.approvalChains'),
          description: t('admin.approvalProcess'),
          href: '/admin/approval-chains',
          icon: FileText,
          color: 'bg-emerald-500',
          stats: t('admin.approvalChainsCount', { count: 12 }),
        },
        {
          title: t('admin.slaDefinitions'),
          description: t('admin.slaSettings'),
          href: '/admin/sla-definitions',
          icon: Calendar,
          color: 'bg-teal-500',
          stats: t('admin.slaRulesCount', { count: 8 }),
        },
        {
          title: t('admin.escalationRules'),
          description: t('admin.escalationStrategy'),
          href: '/admin/escalation-rules',
          icon: Zap,
          color: 'bg-yellow-500',
          stats: t('admin.escalationRulesCount', { count: 6 }),
        },
      ],
    },
    systemConfig: {
      title: t('admin.systemConfig'),
      description: t('admin.systemSettings'),
      actions: [
        {
          title: t('admin.serviceCatalog'),
          description: t('admin.catalogManagement'),
          href: '/admin/service-catalogs',
          icon: BookOpen,
          color: 'bg-orange-500',
          stats: t('admin.serviceItemsCount', { count: 89 }),
        },
        {
          title: t('admin.notificationConfig'),
          description: t('admin.notificationRules'),
          href: '/admin/notifications',
          icon: Bell,
          color: 'bg-red-500',
          stats: t('admin.notificationRulesCount', { count: 24 }),
        },
        {
          title: t('admin.emailTemplates'),
          description: t('admin.emailTemplateManagement'),
          href: '/admin/email-templates',
          icon: Mail,
          color: 'bg-pink-500',
          stats: t('admin.templatesCount', { count: 18 }),
        },
        {
          title: t('admin.dataSources'),
          description: t('admin.dataSourceConfig'),
          href: '/admin/data-sources',
          icon: Database,
          color: 'bg-slate-500',
          stats: t('admin.dataSourcesCount', { count: 5 }),
        },
        {
          title: t('admin.integrations'),
          description: t('admin.integrationManagement'),
          href: '/admin/integrations',
          icon: Globe,
          color: 'bg-violet-500',
          stats: t('admin.integrationsCount', { count: 12 }),
        },
        {
          title: t('admin.systemProperties'),
          description: t('admin.systemParameters'),
          href: '/admin/system-properties',
          icon: Settings,
          color: 'bg-gray-500',
          stats: t('admin.propertiesCount', { count: 67 }),
        },
      ],
    },
  };

  const QuickActionGroup = ({
    group,
    actions,
  }: {
    group: {
      title: string;
      description: string;
      actions: Array<{
        title: string;
        description: string;
        href: string;
        icon: React.ComponentType<{ className?: string }>;
        color: string;
        stats: string;
      }>;
    };
    actions: Array<{
      title: string;
      description: string;
      href: string;
      icon: React.ComponentType<{ className?: string }>;
      color: string;
      stats: string;
    }>;
  }) => {
    const { token } = theme.useToken();

    const getColorByClass = (colorClass: string) => {
      const colorMap: { [key: string]: string } = {
        'bg-blue-500': token.colorPrimary,
        'bg-indigo-500': '#6366f1',
        'bg-cyan-500': '#06b6d4',
        'bg-purple-500': '#722ed1',
        'bg-green-500': token.colorSuccess,
        'bg-emerald-500': '#10b981',
        'bg-teal-500': '#14b8a6',
        'bg-yellow-500': token.colorWarning,
        'bg-orange-500': '#f97316',
        'bg-red-500': token.colorError,
        'bg-pink-500': '#ec4899',
        'bg-slate-500': '#64748b',
        'bg-violet-500': '#8b5cf6',
        'bg-gray-500': token.colorTextSecondary,
      };
      return colorMap[colorClass] || token.colorPrimary;
    };

    return (
      <Card
        title={
          <div>
            <Title level={4} style={{ margin: 0 }}>
              {group.title}
            </Title>
            <Text type='secondary'>{group.description}</Text>
          </div>
        }
      >
        <Row gutter={[16, 16]}>
          {actions.map((action, index) => {
            const Icon = action.icon;
            return (
              <Col xs={24} md={12} key={index}>
                <Link href={action.href} style={{ textDecoration: 'none' }}>
                  <Card
                    size='small'
                    hoverable
                    style={{ height: '100%' }}
                    styles={{ body: { padding: token.paddingMD } }}
                  >
                    <Space align='start' style={{ width: '100%' }}>
                      <Avatar
                        size={40}
                        style={{
                          backgroundColor: getColorByClass(action.color),
                          border: 'none',
                        }}
                        icon={<Icon className='w-5 h-5' />}
                      />
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            marginBottom: 4,
                          }}
                        >
                          <Text strong style={{ color: token.colorText }}>
                            {action.title}
                          </Text>
                          <ArrowUpRight
                            className='w-4 h-4'
                            style={{ color: token.colorTextSecondary }}
                          />
                        </div>
                        <Tooltip title={action.description}>
                          <p
                            className='truncate'
                            style={{
                              fontSize: token.fontSizeSM,
                              margin: '4px 0',
                              lineHeight: 1.4,
                            }}
                          >
                            {action.description}
                          </p>
                        </Tooltip>
                        <Text
                          type='secondary'
                          style={{ fontSize: token.fontSizeSM, fontWeight: 500 }}
                        >
                          {action.stats}
                        </Text>
                      </div>
                    </Space>
                  </Card>
                </Link>
              </Col>
            );
          })}
        </Row>
      </Card>
    );
  };

  return (
    <div style={{ marginBottom: token.marginLG }}>
      <Title level={3} style={{ marginBottom: token.marginLG }}>
        <Zap
          className='w-5 h-5'
          style={{ marginRight: token.marginXS, color: token.colorPrimary }}
        />
        {t('admin.quickActions')}
      </Title>
      <Space orientation='vertical' size='large' style={{ width: '100%' }}>
        {Object.entries(quickActionGroups).map(([key, group]) => (
          <QuickActionGroup key={key} group={group} actions={group.actions} />
        ))}
      </Space>
    </div>
  );
};