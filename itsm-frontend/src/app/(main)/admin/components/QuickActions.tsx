'use client';

import React from 'react';
import Link from 'next/link';
import { Card, Row, Col, Typography, Space, Avatar, Tooltip, Badge } from 'antd';
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
  ChevronRight,
  UserCheck,
  UserCog,
  UserPlus,
  Lock,
  ClipboardList,
  CheckSquare,
  Clock,
  AlertTriangle,
  Megaphone,
  Ticket,
  Folder,
  Cog,
} from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import { LucideIcon } from 'lucide-react';

const { Title, Text } = Typography;

interface QuickActionItem {
  title: string;
  desc: string;
  href: string;
  count: number;
  color: string;
  icon: LucideIcon;
}

// 独特的设计系统 - 精品企业风格
const DESIGN = {
  colors: {
    primary: '#0f172a',
    accent: '#3b82f6',
    success: '#10b981',
    warning: '#f59e0b',
    surface: '#ffffff',
    surfaceSubtle: '#f8fafc',
    border: '#e2e8f0',
    text: '#1e293b',
    textMuted: '#64748b',
  },
  shadows: {
    card: '0 1px 3px rgb(0 0 0 / 0.05), 0 1px 2px -1px rgb(0 0 0 / 0.05)',
    cardHover: '0 10px 40px -10px rgb(0 0 0 / 0.15)',
    glow: (color: string) => `0 0 30px ${color}20`,
  },
  radius: {
    sm: '8px',
    md: '12px',
    lg: '16px',
    xl: '20px',
  },
};

export const QuickActions: React.FC = () => {
  const { t } = useI18n();

  const actionGroups = [
    {
      id: 'users',
      title: '用户与权限',
      subtitle: '管理用户账户、角色和权限',
      icon: Users,
      accent: '#3b82f6',
      items: [
        { title: '用户管理', desc: '用户账户与组织', href: '/admin/users', count: 1234, color: '#3b82f6', icon: UserCheck },
        { title: '角色管理', desc: '角色与权限配置', href: '/admin/roles', count: 15, color: '#6366f1', icon: UserCog },
        { title: '用户组', desc: '组织架构管理', href: '/admin/groups', count: 28, color: '#06b6d4', icon: UserPlus },
        { title: '权限矩阵', desc: '细粒度权限控制', href: '/admin/permissions', count: 156, color: '#8b5cf6', icon: Lock },
      ],
    },
    {
      id: 'process',
      title: '流程与自动化',
      subtitle: '配置工作流和审批规则',
      icon: Workflow,
      accent: '#10b981',
      items: [
        { title: '工作流设计', desc: 'BPMN流程编排', href: '/admin/workflows', count: 45, color: '#10b981', icon: Workflow },
        { title: '审批链', desc: '多级审批规则', href: '/admin/approval-chains', count: 12, color: '#14b8a6', icon: CheckSquare },
        { title: 'SLA定义', desc: '服务级别协议', href: '/admin/sla-definitions', count: 8, color: '#f59e0b', icon: Clock },
        { title: '升级规则', desc: '自动升级策略', href: '/admin/escalation-rules', count: 6, color: '#f97316', icon: AlertTriangle },
      ],
    },
    {
      id: 'system',
      title: '系统配置',
      subtitle: '服务目录和通知设置',
      icon: Settings,
      accent: '#8b5cf6',
      items: [
        { title: '服务目录', desc: '服务项管理', href: '/admin/service-catalogs', count: 89, color: '#ec4899', icon: BookOpen },
        { title: '通知配置', desc: '消息推送规则', href: '/admin/notifications', count: 24, color: '#ef4444', icon: Megaphone },
        { title: '工单分类', desc: '分类与模板', href: '/admin/ticket-categories', count: 32, color: '#64748b', icon: Folder },
        { title: '系统设置', desc: '全局参数配置', href: '/admin/system-config', count: 67, color: '#0f172a', icon: Cog },
      ],
    },
  ];

  const ActionCard = ({
    item,
    index,
    groupIndex,
  }: {
    item: QuickActionItem;
    index: number;
    groupIndex: number;
  }) => {
    const delay = (groupIndex * 100) + (index * 50);

    return (
      <Link href={item.href} style={{ textDecoration: 'none' }}>
        <Card
          hoverable
          style={{
            height: '100%',
            borderRadius: DESIGN.radius.lg,
            border: `1px solid ${DESIGN.colors.border}`,
            background: 'white',
            transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
            overflow: 'hidden',
          }}
          styles={{ body: { padding: '20px' } }}
          className="action-card"
        >
          {/* 悬停效果背景 */}
          <div
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              background: `linear-gradient(135deg, ${item.color}08 0%, transparent 100%)`,
              opacity: 0,
              transition: 'opacity 0.3s',
              pointerEvents: 'none',
            }}
            className="card-bg"
          />

          <div style={{ position: 'relative', zIndex: 1 }}>
            {/* 图标行 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 14 }}>
              <div
                style={{
                  width: 42,
                  height: 42,
                  borderRadius: DESIGN.radius.md,
                  background: `linear-gradient(135deg, ${item.color}15 0%, ${item.color}08 100%)`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  border: `1px solid ${item.color}20`,
                }}
              >
                <item.icon size={20} style={{ color: item.color }} />
              </div>
              <Badge
                count={item.count}
                style={{
                  backgroundColor: `${item.color}15`,
                  color: item.color,
                  border: 'none',
                  fontWeight: 600,
                  fontSize: 11,
                }}
              />
            </div>

            {/* 标题 */}
            <div style={{ marginBottom: 6 }}>
              <Text strong style={{ fontSize: 15, color: DESIGN.colors.text, fontWeight: 600 }}>
                {item.title}
              </Text>
            </div>

            {/* 描述 */}
            <Text style={{ fontSize: 13, color: DESIGN.colors.textMuted, display: 'block', lineHeight: 1.5 }}>
              {item.desc}
            </Text>

            {/* 箭头指示器 */}
            <div
              style={{
                position: 'absolute',
                right: 16,
                bottom: 16,
                opacity: 0,
                transform: 'translateX(-8px)',
                transition: 'all 0.3s',
                color: item.color,
              }}
              className="arrow-indicator"
            >
              <ChevronRight size={18} />
            </div>
          </div>

          <style>{`
            .action-card:hover {
              box-shadow: ${DESIGN.shadows.cardHover}, ${DESIGN.shadows.glow(item.color)};
              transform: translateY(-2px);
            }
            .action-card:hover .card-bg {
              opacity: 1;
            }
            .action-card:hover .arrow-indicator {
              opacity: 1;
              transform: translateX(0);
            }
          `}</style>
        </Card>
      </Link>
    );
  };

  const GroupSection = ({
    group,
    index,
  }: {
    group: typeof actionGroups[0];
    index: number;
  }) => {
    const Icon = group.icon;

    return (
      <div
        style={{
          marginBottom: 32,
        }}
        className="group-section"
      >
        {/* 分组标题 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 14,
            marginBottom: 20,
            paddingBottom: 16,
            borderBottom: `1px solid ${DESIGN.colors.border}`,
          }}
        >
          <div
            style={{
              width: 44,
              height: 44,
              borderRadius: DESIGN.radius.md,
              background: `linear-gradient(135deg, ${group.accent} 0%, ${group.accent}cc 100%)`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: `0 4px 12px ${group.accent}30`,
              color: 'white',
            }}
          >
            <Icon size={22} />
          </div>
          <div>
            <Title
              level={4}
              style={{
                margin: 0,
                fontSize: 18,
                fontWeight: 700,
                color: DESIGN.colors.text,
                letterSpacing: '-0.01em',
              }}
            >
              {group.title}
            </Title>
            <Text style={{ fontSize: 13, color: DESIGN.colors.textMuted }}>
              {group.subtitle}
            </Text>
          </div>
        </div>

        {/* 卡片网格 */}
        <Row gutter={[16, 16]}>
          {group.items.map((item, itemIndex) => (
            <Col xs={24} sm={12} lg={6} key={itemIndex}>
              <ActionCard item={item} index={itemIndex} groupIndex={index} />
            </Col>
          ))}
        </Row>
      </div>
    );
  };

  return (
    <div style={{ marginBottom: 8 }}>
      {/* 主标题 */}
      <div
        style={{
          marginBottom: 28,
          paddingBottom: 20,
          borderBottom: `1px solid ${DESIGN.colors.border}`,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
          <div
            style={{
              width: 40,
              height: 40,
              borderRadius: DESIGN.radius.md,
              background: 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: '0 4px 12px #f59e0b30',
              color: 'white',
            }}
          >
            <Zap size={20} />
          </div>
          <Title
            level={3}
            style={{
              margin: 0,
              fontSize: 22,
              fontWeight: 700,
              color: DESIGN.colors.text,
              letterSpacing: '-0.02em',
            }}
          >
            快捷操作
          </Title>
        </div>
        <Text style={{ fontSize: 14, color: DESIGN.colors.textMuted }}>
          快速访问系统管理和配置功能
        </Text>
      </div>

      {/* 分组区域 */}
      {actionGroups.map((group, index) => (
        <GroupSection key={group.id} group={group} index={index} />
      ))}

      {/* 动画 */}
      <style>{`
        @keyframes fadeInUp {
          from { opacity: 0; transform: translateY(20px); }
          to { opacity: 1; transform: translateY(0); }
        }
        .group-section {
          animation: fadeInUp 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
          opacity: 0;
          animation-delay: 0.1s;
        }
        .group-section:nth-child(1) { animation-delay: 0.1s; }
        .group-section:nth-child(2) { animation-delay: 0.2s; }
        .group-section:nth-child(3) { animation-delay: 0.3s; }
      `}</style>
    </div>
  );
};
