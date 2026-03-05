'use client';

import React from 'react';
import { Card, Col, Row, Statistic, Typography, theme, Avatar, Progress } from 'antd';
import { Users, Workflow, BookOpen, AlertCircle, TrendingUp, BarChart3, Activity, Shield, Zap } from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import { AdminStats } from '../hooks/useAdminData';

const { Paragraph, Title, Text } = Typography;

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

export const SystemOverview: React.FC<SystemOverviewProps> = ({ stats, loading }) => {
  const { t } = useI18n();
  const { token } = theme.useToken();

  const systemStats = [
    {
      title: '活跃用户',
      value: stats?.activeUsers ?? '1,234',
      change: '+12%',
      changeValue: '+132',
      icon: Users,
      color: DESIGN_SYSTEM.colors.accent,
      gradient: DESIGN_SYSTEM.colors.gradient.accent,
      trend: 'up',
      description: '较上月新增132位活跃用户',
      progress: 78,
    },
    {
      title: '运行中的流程',
      value: stats?.runningWorkflows ?? '45',
      change: '+6.7%',
      changeValue: '+3',
      icon: Workflow,
      color: DESIGN_SYSTEM.colors.success,
      gradient: DESIGN_SYSTEM.colors.gradient.success,
      trend: 'up',
      description: '3个工作流新启动',
      progress: 65,
    },
    {
      title: '服务目录项',
      value: stats?.serviceCatalogItems ?? '89',
      change: '+5.9%',
      changeValue: '+5',
      icon: BookOpen,
      color: '#8b5cf6',
      gradient: 'linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%)',
      trend: 'up',
      description: '新增5项服务目录',
      progress: 82,
    },
    {
      title: '系统告警',
      value: stats?.systemAlerts ?? '2',
      change: '-33%',
      changeValue: '-1',
      icon: AlertCircle,
      color: DESIGN_SYSTEM.colors.warning,
      gradient: DESIGN_SYSTEM.colors.gradient.warning,
      trend: 'down',
      description: '较昨日减少1个告警',
      progress: 15,
    },
  ];

  const EnhancedStatCard = ({ stat, index }: { stat: typeof systemStats[0]; index: number }) => {
    const Icon = stat.icon;
    const isPositive = stat.trend === 'up';
    const delay = index * 100;

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

          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '4px 10px',
              borderRadius: 20,
              background: isPositive ? `${DESIGN_SYSTEM.colors.success}15` : `${DESIGN_SYSTEM.colors.danger}15`,
              color: isPositive ? DESIGN_SYSTEM.colors.success : DESIGN_SYSTEM.colors.danger,
              fontSize: 13,
              fontWeight: 600,
            }}
          >
            <TrendingUp size={14} style={{ transform: isPositive ? 'none' : 'rotate(180deg)' }} />
            <span>{stat.change}</span>
          </div>
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
            color: DESIGN_SYSTEM.colors.textPrimary,
            lineHeight: 1.2,
            marginBottom: 12,
            fontFamily: DESIGN_SYSTEM.fonts.display,
            letterSpacing: '-0.02em',
          }}
        >
          {stat.value}
        </div>

        {/* 进度条 */}
        <div style={{ marginBottom: 12 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
            <Text style={{ fontSize: 12, color: DESIGN_SYSTEM.colors.textSecondary }}>
              {stat.description}
            </Text>
            <Text style={{ fontSize: 12, fontWeight: 600, color: stat.color }}>
              {stat.progress}%
            </Text>
          </div>
          <div
            style={{
              height: 6,
              borderRadius: 3,
              background: '#f1f5f9',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                height: '100%',
                width: `${stat.progress}%`,
                borderRadius: 3,
                background: stat.gradient,
                transition: 'width 1s cubic-bezier(0.4, 0, 0.2, 1)',
              }}
            />
          </div>
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
          <Title
            level={3}
            style={{
              margin: 0,
              fontSize: 22,
              fontWeight: 700,
              color: DESIGN_SYSTEM.colors.textPrimary,
              letterSpacing: '-0.02em',
            }}
          >
            系统概览
          </Title>
        </div>
        <Text style={{ color: DESIGN_SYSTEM.colors.textSecondary, fontSize: 14 }}>
          实时监控系统关键指标和业务健康状态
        </Text>
      </div>

      {/* 统计卡片网格 */}
      <Row gutter={[20, 20]}>
        {systemStats.map((stat, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <EnhancedStatCard stat={stat} index={index} />
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
