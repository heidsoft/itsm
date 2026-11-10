'use client';

import React from 'react';
import { Card, Col, Row, Typography } from 'antd';
import { Users, Workflow, BookOpen, AlertCircle, BarChart3 } from 'lucide-react';
import type { AdminStats } from '../hooks/useAdminData';

const { Text } = Typography;

// 增强的设计系统 - 独特的企业仪表盘美学
const DESIGN_SYSTEM = {
  colors: {
    primary: '#0f172a', // 深海军蓝
    accent: '#3b82f6', // 明亮蓝
    success: '#10b981', // 翠绿
    warning: '#f59e0b', // 琥珀
    danger: '#ef4444', // 珊瑚红
    surface: '#ffffff',
    surfaceAlt: '#f8fafc',
    border: '#e2e8f0',
    textPrimary: '#1e293b',
    textSecondary: '#64748b',
    gradient: {
      card: 'linear-gradient(135deg, #ffffff 0%, #f8fafc 100%)',
      accent: 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)',
      success: 'linear-gradient(135deg, #10b981 0%, #059669 100%)',
      warning: 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
      danger: 'linear-gradient(135deg, #ef4444 0%, #dc2626 100%)',
    }
  },
  shadows: {
    card: '0 1px 3px 0 rgb(0 0 0 / 0.05), 0 1px 2px -1px rgb(0 0 0 / 0.05)',
    cardHover: '0 10px 15px -3px rgb(0 0 0 / 0.08), 0 4px 6px -4px rgb(0 0 0 / 0.05)',
    glow: '0 0 20px rgba(59, 130, 246, 0.15)',
  },
  borderRadius: {
    sm: '8px',
    md: '12px',
    lg: '16px',
    xl: '24px',
  },
  fonts: {
    display: 'system-ui, -apple-system, sans-serif',
    body: 'system-ui, -apple-system, sans-serif',
  }
};

interface SystemOverviewProps {
  stats?: AdminStats;
  loading?: boolean;
}

export const SystemOverview: React.FC<SystemOverviewProps> = ({ stats }) => {
  const formatValue = (value: string | number | null | undefined) => {
    if (value === null || value === undefined || value === '') {
      return '—';
    }
    return value;
  };

  // 数值均来自后端真实统计接口；接口失败时显示 '—'，不展示虚构的增长百分比或进度。
  const systemStats = [
    {
      title: 'Active Users',
      value: formatValue(stats?.activeUsers),
      icon: Users,
      color: DESIGN_SYSTEM.colors.accent,
      gradient: DESIGN_SYSTEM.colors.gradient.accent,
      description: stats?.activeUsers == null ? 'User statistics interface is unavailable' : 'Real-time statistics from user center',
      placeholder: stats?.activeUsers == null,
    },
    {
      title: 'Running process',
      value: formatValue(stats?.runningWorkflows),
      icon: Workflow,
      color: DESIGN_SYSTEM.colors.success,
      gradient: DESIGN_SYSTEM.colors.gradient.success,
      description: stats?.runningWorkflows == null ? 'Process statistics interface is not available' : 'Process instance currently executing',
      placeholder: stats?.runningWorkflows == null,
    },
    {
      title: 'Service Catalog Items',
      value: formatValue(stats?.serviceCatalogItems),
      icon: BookOpen,
      color: '#8b5cf6',
      gradient: 'linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%)',
      description: stats?.serviceCatalogItems == null ? 'Service catalog interface is not available' : 'Total number of registered services in the service directory',
      placeholder: stats?.serviceCatalogItems == null,
    },
    {
      title: 'Pending work order',
      value: formatValue(stats?.pendingTickets),
      icon: AlertCircle,
      color: DESIGN_SYSTEM.colors.warning,
      gradient: DESIGN_SYSTEM.colors.gradient.warning,
      description: stats?.pendingTickets == null ? 'The work order statistics interface is unavailable' : 'Work orders with new and processing status',
      placeholder: stats?.pendingTickets == null,
    },
  ];

  const EnhancedStatCard = ({ stat }: { stat: (typeof systemStats)[0] }) => {
    const Icon = stat.icon;

    return (
      <Card
        hoverable
        style={{
          height: '100%',
          borderRadius: DESIGN_SYSTEM.borderRadius.lg,
          border: `1px solid ${DESIGN_SYSTEM.colors.border}`,
          background: DESIGN_SYSTEM.colors.gradient.card,
          boxShadow: DESIGN_SYSTEM.shadows.card,
          transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
          overflow: 'hidden',
          position: 'relative',
        }}
        styles={{
          body: { padding: '24px' }
        }}
        className="animate-fade-in"
      >
        {/* 装饰性背景元素 */}
        <div
          style={{
            position: 'absolute',
            top: -50,
            right: -50,
            width: 150,
            height: 150,
            borderRadius: '50%',
            background: stat.gradient,
            opacity: 0.08,
            pointerEvents: 'none',
          }}
        />

        {/* 头部区域 */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            marginBottom: 20,
          }}
        >
          <div
            style={{
              width: 52,
              height: 52,
              borderRadius: DESIGN_SYSTEM.borderRadius.md,
              background: stat.gradient,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: `0 4px 12px ${stat.color}30`,
              color: '#fff',
            }}
          >
            <Icon size={24} />
          </div>
          {stat.placeholder && (
            <span
              style={{
                padding: '4px 10px',
                borderRadius: 20,
                background: `${DESIGN_SYSTEM.colors.textSecondary}15`,
                color: DESIGN_SYSTEM.colors.textSecondary,
                fontSize: 13,
                fontWeight: 600,
              }}
            >

              No data
            </span>
          )}
        </div>

        {/* 统计数字 */}
        <div style={{ marginBottom: 8 }}>
          <Text style={{ color: DESIGN_SYSTEM.colors.textSecondary, fontSize: 14, fontWeight: 500 }}>
            {stat.title}
          </Text>
        </div>

        <div
          style={{
            fontSize: 32,
            fontWeight: 700,
            color: stat.placeholder ? DESIGN_SYSTEM.colors.textSecondary : DESIGN_SYSTEM.colors.textPrimary,
            lineHeight: 1.2,
            marginBottom: 12,
            fontFamily: DESIGN_SYSTEM.fonts.display,
            letterSpacing: '-0.02em',
          }}
        >
          {stat.value}
        </div>

        {/* 数据来源说明 */}
        <div>
          <Text style={{ fontSize: 12, color: DESIGN_SYSTEM.colors.textSecondary }}>
            {stat.description}
          </Text>
        </div>
      </Card>
    );
  };

  return (
    <div style={{ marginBottom: 32 }}>
      {/* 页面标题区 */}
      <div
        style={{
          marginBottom: 28,
          paddingBottom: 20,
          borderBottom: `1px solid ${DESIGN_SYSTEM.colors.border}`,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
          <div
            style={{
              width: 40,
              height: 40,
              borderRadius: DESIGN_SYSTEM.borderRadius.md,
              background: DESIGN_SYSTEM.colors.gradient.accent,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#fff',
              boxShadow: DESIGN_SYSTEM.shadows.glow,
            }}
          >
            <BarChart3 size={20} />
          </div>
          <Text
            style={{
              fontSize: 22,
              fontWeight: 700,
              color: DESIGN_SYSTEM.colors.textPrimary,
              letterSpacing: '-0.02em',
            }}
          >

            System Overview
          </Text>
        </div>
        <Text style={{ color: DESIGN_SYSTEM.colors.textSecondary, fontSize: 14 }}>

          Real-time monitoring of system key indicators and business health status
        </Text>
      </div>

      {/* 统计卡片网格 */}
      <Row gutter={[20, 20]}>
        {systemStats.map((stat) => (
          <Col xs={24} sm={12} lg={6} key={stat.title}>
            <EnhancedStatCard stat={stat} />
          </Col>
        ))}
      </Row>

      {/* 动画样式 */}
      <style>{`
        @keyframes fadeInUp {
          from {
            opacity: 0;
            transform: translateY(20px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
        .animate-fade-in {
          animation: fadeInUp 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
          opacity: 0;
        }
      `}</style>
    </div>
  );
};
