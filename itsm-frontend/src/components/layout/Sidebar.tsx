'use client';

import React, { useState, useEffect } from 'react';
import { Layout, Menu, theme, Badge } from 'antd';
import {
  LayoutDashboard,
  FileText,
  AlertCircle,
  HelpCircle,
  BarChart3,
  Database,
  Book,
  Calendar,
  GitMerge,
  Shield,
  TrendingUp,
  Rocket,
  Monitor,
  Key,
  Workflow,
  Play,
  Settings,
  History,
  Users,
  User,
  Activity,
  Clock,
  ClipboardList,
  Bot,
  MessageSquare,
  Sparkles,
  Lightbulb,
  Send,
  CheckSquare,
  RotateCcw,
  ArrowRight,
  Folder,
  Target,
  Wrench,
  ArrowLeftRight,
  AlertTriangle,
  CheckCircle,
  Zap,
  Tag,
  Link,
  Lock,
  Bell,
  Cloud,
  Boxes,
  RefreshCw,
  Server,
  GitBranch,
  List,
  BookOpen,
  ArrowUpCircle,
  Plus,
  Building,
  Search,
  Edit,
} from 'lucide-react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import styles from './Sidebar.module.css';
import { useI18n } from '@/lib/i18n';
import { getUserMenus, MenuItem as MenuItemType, MenuTreeResponse } from '@/lib/api/menu-api';

const { Sider } = Layout;

// 图标样式，统一大小
const iconStyle = { width: 16, height: 16 };

// 菜单项接口定义
interface MenuItem {
  key: string;
  icon: React.ReactNode;
  label: string | React.ReactNode;
  path?: string;
  permission?: string;
  description?: string;
  badge?: string;
  children?: MenuItem[];
}

// 菜单配置接口
interface MenuConfig {
  main: MenuItem[];
  admin: MenuItem[];
}

// 获取菜单配置 - ITIL v4 标准结构
const getMenuConfig = (t: (key: string, params?: Record<string, string | number>) => string): MenuConfig => ({
  main: [
    // ===== 服务运营 =====
    {
      key: '/dashboard',
      icon: <LayoutDashboard style={iconStyle} />,
      label: '服务台',
      path: '/dashboard',
      permission: 'dashboard:view',
      description: '服务台概览',
    },
    {
      key: '/service-requests',
      icon: <FileText style={iconStyle} />,
      label: '服务请求',
      path: '/service-requests',
      permission: 'ticket:read',
      description: '服务请求管理',
      children: [
        { key: '/service-requests', icon: <FileText style={iconStyle} />, label: '服务请求', path: '/service-requests', permission: 'ticket:read' },
        { key: '/tickets/templates', icon: <FileText style={iconStyle} />, label: '服务请求模板', path: '/tickets/templates', permission: 'ticket:write' },
        { key: '/tickets/analytics', icon: <BarChart3 style={iconStyle} />, label: '工单统计', path: '/tickets/analytics', permission: 'ticket:read' },
      ],
    },
    {
      key: '/my-requests',
      icon: <User style={iconStyle} />,
      label: '我的请求',
      path: '/my-requests',
      permission: 'ticket:read',
      description: '我的服务请求',
    },
    {
      key: '/incidents',
      icon: <AlertCircle style={iconStyle} />,
      label: '事件管理',
      path: '/incidents',
      permission: 'incident:read',
      description: '事件管理',
      children: [
        { key: '/incidents', icon: <AlertCircle style={iconStyle} />, label: '事件列表', path: '/incidents', permission: 'incident:read' },
        { key: '/incidents/major', icon: <AlertTriangle style={iconStyle} />, label: '重大事件', path: '/incidents/major', permission: 'incident:read' },
        { key: '/incidents/analytics', icon: <TrendingUp style={iconStyle} />, label: '事件分析', path: '/incidents/analytics', permission: 'incident:read' },
        { key: '/incidents/convert', icon: <ArrowRight style={iconStyle} />, label: '事件转问题', path: '/incidents/convert', permission: 'incident:manage' },
      ],
    },
    {
      key: '/problems',
      icon: <HelpCircle style={iconStyle} />,
      label: '问题管理',
      path: '/problems',
      permission: 'problem:read',
      description: '问题管理',
      children: [
        { key: '/problems', icon: <HelpCircle style={iconStyle} />, label: '问题列表', path: '/problems', permission: 'problem:read' },
        { key: '/problems/known-errors', icon: <AlertCircle style={iconStyle} />, label: '已知错误', path: '/problems/known-errors', permission: 'problem:read' },
        { key: '/problems/root-cause', icon: <Search style={iconStyle} />, label: '根因分析', path: '/problems/root-cause', permission: 'problem:analyze' },
        { key: '/problems/links', icon: <Link style={iconStyle} />, label: '问题链接', path: '/problems/links', permission: 'problem:manage' },
      ],
    },
    {
      key: '/changes',
      icon: <BarChart3 style={iconStyle} />,
      label: '变更管理',
      path: '/changes',
      permission: 'change:read',
      description: '变更管理',
      children: [
        { key: '/changes', icon: <BarChart3 style={iconStyle} />, label: '变更列表', path: '/changes', permission: 'change:read' },
        { key: '/changes/standard', icon: <CheckCircle style={iconStyle} />, label: '标准变更', path: '/changes/standard', permission: 'change:read' },
        { key: '/changes/emergency', icon: <Zap style={iconStyle} />, label: '紧急变更', path: '/changes/emergency', permission: 'change:manage' },
        { key: '/changes/approvals', icon: <GitMerge style={iconStyle} />, label: '变更审批', path: '/changes/approvals', permission: 'change:approve' },
        { key: '/changes/cab', icon: <Users style={iconStyle} />, label: 'CAB会议', path: '/changes/cab', permission: 'change:manage' },
        { key: '/changes/calendar', icon: <Calendar style={iconStyle} />, label: '变更日历', path: '/changes/calendar', permission: 'change:read' },
      ],
    },
    // ===== 服务保障 =====
    {
      key: '/knowledge',
      icon: <Book style={iconStyle} />,
      label: '知识库',
      path: '/knowledge',
      permission: 'knowledge:read',
      description: '知识库管理',
      children: [
        { key: '/knowledge', icon: <Book style={iconStyle} />, label: '知识库', path: '/knowledge', permission: 'knowledge:read' },
        { key: '/knowledge/articles', icon: <FileText style={iconStyle} />, label: '文章管理', path: '/knowledge/articles', permission: 'knowledge:write' },
        { key: '/knowledge/categories', icon: <Tag style={iconStyle} />, label: '文章分类', path: '/knowledge/categories', permission: 'knowledge:manage' },
        { key: '/knowledge/review', icon: <CheckSquare style={iconStyle} />, label: '知识审核', path: '/knowledge/review', permission: 'knowledge:approve' },
        { key: '/knowledge/ai-recommend', icon: <Sparkles style={iconStyle} />, label: 'AI知识推荐', path: '/knowledge/ai-recommend', permission: 'knowledge:read' },
      ],
    },
    {
      key: '/service-catalog',
      icon: <BookOpen style={iconStyle} />,
      label: '服务目录',
      path: '/service-catalog',
      permission: 'service:read',
      description: '服务目录',
      children: [
        { key: '/service-catalog', icon: <BookOpen style={iconStyle} />, label: '服务目录', path: '/service-catalog', permission: 'service:read' },
        { key: '/service-catalog/items', icon: <List style={iconStyle} />, label: '服务项管理', path: '/service-catalog/items', permission: 'service:write' },
        { key: '/service-catalog/categories', icon: <Folder style={iconStyle} />, label: '服务类别', path: '/service-catalog/categories', permission: 'service:write' },
        { key: '/service-catalog/templates', icon: <FileText style={iconStyle} />, label: '服务请求模板', path: '/service-catalog/templates', permission: 'service:write' },
      ],
    },
    {
      key: '/cmdb',
      icon: <Database style={iconStyle} />,
      label: 'CMDB',
      path: '/cmdb',
      permission: 'cmdb:read',
      description: '配置管理数据库',
      children: [
        { key: '/cmdb/cis', icon: <Server style={iconStyle} />, label: '配置项列表', path: '/cmdb/cis', permission: 'cmdb:read' },
        { key: '/cmdb/types', icon: <Database style={iconStyle} />, label: 'CI类型管理', path: '/cmdb/types', permission: 'cmdb:write' },
        { key: '/cmdb/cloud-resources', icon: <Cloud style={iconStyle} />, label: '云资源', path: '/cmdb/cloud-resources', permission: 'cmdb:read' },
        { key: '/cmdb/cloud-accounts', icon: <Key style={iconStyle} />, label: '云账号', path: '/cmdb/cloud-accounts', permission: 'cmdb:manage' },
        { key: '/cmdb/cloud-services', icon: <Boxes style={iconStyle} />, label: '云服务', path: '/cmdb/cloud-services', permission: 'cmdb:read' },
        { key: '/cmdb/reconciliation', icon: <RefreshCw style={iconStyle} />, label: '同步对账', path: '/cmdb/reconciliation', permission: 'cmdb:write' },
        { key: '/cmdb/relationships', icon: <GitBranch style={iconStyle} />, label: '关系图谱', path: '/cmdb/relationships', permission: 'cmdb:read' },
      ],
    },
    {
      key: '/assets',
      icon: <Monitor style={iconStyle} />,
      label: '资产管理',
      path: '/assets',
      permission: 'asset:read',
      description: 'IT资产管理',
      children: [
        { key: '/assets/list', icon: <Server style={iconStyle} />, label: '资产列表', path: '/assets/list', permission: 'asset:read' },
        { key: '/assets/licenses', icon: <Key style={iconStyle} />, label: '软件许可证', path: '/assets/licenses', permission: 'license:manage' },
        { key: '/assets/categories', icon: <Tag style={iconStyle} />, label: '资产分类', path: '/assets/categories', permission: 'asset:write' },
        { key: '/assets/maintenance', icon: <Wrench style={iconStyle} />, label: '维保管理', path: '/assets/maintenance', permission: 'asset:manage' },
      ],
    },
    // ===== 报告分析 =====
    {
      key: '/sla',
      icon: <Calendar style={iconStyle} />,
      label: 'SLA管理',
      path: '/sla',
      permission: 'sla:read',
      description: 'SLA监控与配置',
      children: [
        { key: '/sla/monitor', icon: <Activity style={iconStyle} />, label: 'SLA监控', path: '/sla/monitor', permission: 'sla:read' },
        { key: '/sla/definitions', icon: <FileText style={iconStyle} />, label: 'SLA定义', path: '/sla/definitions', permission: 'sla:write' },
        { key: '/sla/escalations', icon: <ArrowUpCircle style={iconStyle} />, label: '升级规则', path: '/sla/escalations', permission: 'sla:write' },
        { key: '/sla/reports', icon: <BarChart3 style={iconStyle} />, label: 'SLA报表', path: '/sla/reports', permission: 'sla:read' },
        { key: '/sla/targets', icon: <Target style={iconStyle} />, label: '目标管理', path: '/sla/targets', permission: 'sla:manage' },
      ],
    },
    {
      key: '/reports',
      icon: <TrendingUp style={iconStyle} />,
      label: '报表中心',
      path: '/reports',
      permission: 'report:read',
      description: '报表与分析',
      children: [
        { key: '/reports/tickets', icon: <FileText style={iconStyle} />, label: '工单报表', path: '/reports/tickets', permission: 'report:read' },
        { key: '/reports/incidents', icon: <AlertCircle style={iconStyle} />, label: '事件报表', path: '/reports/incidents', permission: 'report:read' },
        { key: '/reports/problems', icon: <HelpCircle style={iconStyle} />, label: '问题报表', path: '/reports/problems', permission: 'report:read' },
        { key: '/reports/changes', icon: <BarChart3 style={iconStyle} />, label: '变更报表', path: '/reports/changes', permission: 'report:read' },
        { key: '/reports/sla', icon: <Calendar style={iconStyle} />, label: 'SLA报表', path: '/reports/sla', permission: 'report:read' },
        { key: '/reports/cmdb-quality', icon: <Database style={iconStyle} />, label: 'CMDB质量', path: '/reports/cmdb-quality', permission: 'report:read' },
        { key: '/reports/catalog-usage', icon: <BookOpen style={iconStyle} />, label: '服务目录使用', path: '/reports/catalog-usage', permission: 'report:read' },
        { key: '/reports/operations', icon: <TrendingUp style={iconStyle} />, label: '运维报表', path: '/reports/operations', permission: 'report:read' },
      ],
    },
    // ===== 自动化 =====
    {
      key: '/workflow',
      icon: <GitMerge style={iconStyle} />,
      label: '工作流',
      path: '/workflow',
      permission: 'workflow:read',
      description: '工作流自动化',
      children: [
        { key: '/workflow/list', icon: <List style={iconStyle} />, label: '工作流列表', path: '/workflow/list', permission: 'workflow:read' },
        { key: '/workflow/designer', icon: <Edit style={iconStyle} />, label: '流程设计器', path: '/workflow/designer', permission: 'workflow:write' },
        { key: '/workflow/instances', icon: <Play style={iconStyle} />, label: '流程实例', path: '/workflow/instances', permission: 'workflow:read' },
        { key: '/workflow/versions', icon: <History style={iconStyle} />, label: '版本管理', path: '/workflow/versions', permission: 'workflow:write' },
        { key: '/workflow/dashboard', icon: <Activity style={iconStyle} />, label: '监控仪表盘', path: '/workflow/dashboard', permission: 'workflow:read' },
        { key: '/workflow/audit', icon: <ClipboardList style={iconStyle} />, label: '审计日志', path: '/workflow/audit', permission: 'workflow:read' },
        { key: '/workflow/automation', icon: <Zap style={iconStyle} />, label: '自动化规则', path: '/workflow/automation', permission: 'workflow:write' },
      ],
    },
    // ===== AI =====
    {
      key: '/ai/chat',
      icon: <Bot style={iconStyle} />,
      label: 'AI助手',
      path: '/ai/chat',
      permission: 'ai:use',
      description: 'AI 智能助手',
      children: [
        { key: '/ai/chat', icon: <MessageSquare style={iconStyle} />, label: 'AI对话', path: '/ai/chat', permission: 'ai:use' },
        { key: '/tickets/ai-create', icon: <Sparkles style={iconStyle} />, label: 'AI创建工单', path: '/tickets/ai-create', permission: 'ai:use' },
        { key: '/ai/analyze', icon: <Search style={iconStyle} />, label: '故障分析', path: '/ai/analyze', permission: 'ai:use' },
        { key: '/ai/recommend', icon: <Lightbulb style={iconStyle} />, label: '智能推荐', path: '/ai/recommend', permission: 'ai:use' },
      ],
    },
    // ===== 扩展模块 =====
    {
      key: '/access',
      icon: <Key style={iconStyle} />,
      label: '访问管理',
      path: '/access',
      permission: 'access:read',
      description: '访问权限管理',
      children: [
        { key: '/access/requests', icon: <Send style={iconStyle} />, label: '权限申请', path: '/access/requests', permission: 'access:request' },
        { key: '/access/approvals', icon: <CheckCircle style={iconStyle} />, label: '权限审批', path: '/access/approvals', permission: 'access:approve' },
        { key: '/access/audit', icon: <ClipboardList style={iconStyle} />, label: '权限审计', path: '/access/audit', permission: 'access:audit' },
        { key: '/access/role-requests', icon: <Shield style={iconStyle} />, label: '角色申请', path: '/access/role-requests', permission: 'access:request' },
      ],
    },
    {
      key: '/shifts',
      icon: <Calendar style={iconStyle} />,
      label: '服务台调度',
      path: '/shifts',
      permission: 'helpdesk:manage',
      description: '服务台调度管理',
      children: [
        { key: '/shifts/schedules', icon: <Calendar style={iconStyle} />, label: '班次管理', path: '/shifts/schedules', permission: 'helpdesk:manage' },
        { key: '/shifts/handoffs', icon: <ArrowLeftRight style={iconStyle} />, label: '交接班', path: '/shifts/handoffs', permission: 'helpdesk:manage' },
        { key: '/shifts/logs', icon: <FileText style={iconStyle} />, label: '值班记录', path: '/shifts/logs', permission: 'helpdesk:read' },
      ],
    },
    // ===== MSP与发布 =====
    {
      key: '/msp',
      icon: <Building style={iconStyle} />,
      label: '客户管理',
      path: '/msp',
      permission: 'msp:read',
      description: '客户管理(MSP)',
      children: [
        { key: '/msp/dashboard', icon: <LayoutDashboard style={iconStyle} />, label: '客户仪表盘', path: '/msp/dashboard', permission: 'msp:read' },
        { key: '/msp/allocations', icon: <GitBranch style={iconStyle} />, label: '分配管理', path: '/msp/allocations', permission: 'msp:manage' },
        { key: '/msp/contracts', icon: <FileText style={iconStyle} />, label: '客户合同', path: '/msp/contracts', permission: 'msp:read' },
        { key: '/msp/sla-reports', icon: <BarChart3 style={iconStyle} />, label: 'SLA报告', path: '/msp/sla-reports', permission: 'msp:read' },
      ],
    },
    {
      key: '/releases',
      icon: <Rocket style={iconStyle} />,
      label: '发布管理',
      path: '/releases',
      permission: 'release:read',
      description: '发布管理',
      children: [
        { key: '/releases/plans', icon: <Calendar style={iconStyle} />, label: '发布计划', path: '/releases/plans', permission: 'release:read' },
        { key: '/releases/phases', icon: <GitBranch style={iconStyle} />, label: '发布阶段', path: '/releases/phases', permission: 'release:manage' },
        { key: '/releases/rollbacks', icon: <RotateCcw style={iconStyle} />, label: '回滚计划', path: '/releases/rollbacks', permission: 'release:manage' },
        { key: '/releases/reviews', icon: <CheckSquare style={iconStyle} />, label: '发布评审', path: '/releases/reviews', permission: 'release:approve' },
        { key: '/releases/history', icon: <History style={iconStyle} />, label: '发布历史', path: '/releases/history', permission: 'release:read' },
      ],
    },
  ],
  admin: [
    // ===== 系统管理 =====
    {
      key: '/admin',
      icon: <Settings style={iconStyle} />,
      label: '系统管理',
      path: '/admin',
      permission: 'admin:write',
      description: '系统管理',
      children: [
        { key: '/admin/overview', icon: <LayoutDashboard style={iconStyle} />, label: '系统概览', path: '/admin/overview', permission: 'admin:write' },
        { key: '/admin/users', icon: <Users style={iconStyle} />, label: '用户管理', path: '/admin/users', permission: 'user:read' },
        { key: '/admin/roles', icon: <Shield style={iconStyle} />, label: '角色管理', path: '/admin/roles', permission: 'role:read' },
        { key: '/admin/groups', icon: <Users style={iconStyle} />, label: '组管理', path: '/admin/groups', permission: 'group:read' },
        { key: '/admin/tenants', icon: <Building style={iconStyle} />, label: '租户管理', path: '/admin/tenants', permission: 'tenant:manage' },
        { key: '/admin/departments', icon: <Building style={iconStyle} />, label: '部门管理', path: '/admin/departments', permission: 'department:manage' },
        { key: '/admin/teams', icon: <Users style={iconStyle} />, label: '团队管理', path: '/admin/teams', permission: 'team:manage' },
        { key: '/admin/ticket-categories', icon: <Tag style={iconStyle} />, label: '工单分类', path: '/admin/ticket-categories', permission: 'ticket:category:manage' },
        { key: '/admin/tickets/assignment', icon: <GitBranch style={iconStyle} />, label: '工单分配规则', path: '/admin/tickets/assignment', permission: 'ticket:manage' },
        { key: '/admin/tickets/automation', icon: <Zap style={iconStyle} />, label: '自动化规则', path: '/admin/tickets/automation', permission: 'ticket:manage' },
        { key: '/admin/approvals', icon: <GitMerge style={iconStyle} />, label: '审批管理', path: '/admin/approvals', permission: 'approval:manage' },
        { key: '/admin/approval-chains', icon: <Link style={iconStyle} />, label: '审批链', path: '/admin/approval-chains', permission: 'approval:manage' },
        { key: '/admin/permissions', icon: <Lock style={iconStyle} />, label: '权限管理', path: '/admin/permissions', permission: 'permission:manage' },
        { key: '/admin/system-config', icon: <Settings style={iconStyle} />, label: '系统配置', path: '/admin/system-config', permission: 'system:config' },
        { key: '/admin/notifications', icon: <Bell style={iconStyle} />, label: '通知配置', path: '/admin/notifications', permission: 'system:config' },
        { key: '/admin/audit-logs', icon: <ClipboardList style={iconStyle} />, label: '操作日志', path: '/admin/audit-logs', permission: 'system:audit' },
      ],
    },
  ],
});

interface SidebarProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ collapsed, onCollapse }) => {
  const { token } = theme.useToken();
  const router = useRouter();
  const pathname = usePathname();
  const { user } = useAuthStore();
  const { t } = useI18n();
  // 手动触发 auth store 的 hydration
  useAuthStoreHydration();
  const MENU_CONFIG = getMenuConfig(t);

  // 动态菜单状态
  const [dynamicMenus, setDynamicMenus] = useState<MenuTreeResponse | null>(null);
  const [menuLoading, setMenuLoading] = useState(false);

  // 加载用户菜单
  useEffect(() => {
    const loadMenus = async () => {
      if (!user) return;

      try {
        setMenuLoading(true);
        const menus = await getUserMenus();
        if (menus && (menus.main?.length > 0 || menus.admin?.length > 0)) {
          setDynamicMenus(menus);
        }
      } catch (error) {
        console.warn('Failed to load dynamic menus, using fallback:', error);
      } finally {
        setMenuLoading(false);
      }
    };

    loadMenus();
  }, [user]);

  // 将API菜单转换为Sidebar格式
  const convertApiMenuToSidebar = (menus: MenuItemType[]): MenuItem[] => {
    return menus.map(menu => {
      const icon = getIconByName(menu.icon);
      const item: MenuItem = {
        key: menu.path,
        icon,
        label: menu.name,
        path: menu.path,
        permission: menu.permission_code,
        description: menu.description,
        children: menu.children ? convertApiMenuToSidebar(menu.children) : undefined,
      };
      return item;
    });
  };

  // 根据图标名称获取图标组件
  const getIconByName = (iconName?: string) => {
    if (!iconName) return undefined;
    const iconMap: Record<string, React.ReactNode> = {
      LayoutDashboard: <LayoutDashboard style={iconStyle} />,
      FileText: <FileText style={iconStyle} />,
      AlertCircle: <AlertCircle style={iconStyle} />,
      HelpCircle: <HelpCircle style={iconStyle} />,
      BarChart3: <BarChart3 style={iconStyle} />,
      Database: <Database style={iconStyle} />,
      Book: <Book style={iconStyle} />,
      Calendar: <Calendar style={iconStyle} />,
      GitMerge: <GitMerge style={iconStyle} />,
      Shield: <Shield style={iconStyle} />,
      TrendingUp: <TrendingUp style={iconStyle} />,
      Rocket: <Rocket style={iconStyle} />,
      Monitor: <Monitor style={iconStyle} />,
      Key: <Key style={iconStyle} />,
      Workflow: <Workflow style={iconStyle} />,
      Play: <Play style={iconStyle} />,
      Settings: <Settings style={iconStyle} />,
      History: <History style={iconStyle} />,
      Users: <Users style={iconStyle} />,
      Activity: <Activity style={iconStyle} />,
      Clock: <Clock style={iconStyle} />,
      ClipboardList: <ClipboardList style={iconStyle} />,
      Bot: <Bot style={iconStyle} />,
    };
    return iconMap[iconName];
  };

  // 确定使用哪个菜单配置 - 优先使用数据库动态菜单
  const useDynamicMenu = !!dynamicMenus;
  const mainMenus = useDynamicMenu ? convertApiMenuToSidebar(dynamicMenus.main) : MENU_CONFIG.main;
  const adminMenus = useDynamicMenu ? convertApiMenuToSidebar(dynamicMenus.admin) : MENU_CONFIG.admin;

  const handleMenuClick = ({ key }: { key: string }) => {
    router.push(key);
  };

  const isAdmin = user?.role === 'admin' || user?.role === 'super_admin';

  // 渲染菜单项，支持徽章和描述
  const renderMenuItems = (items: MenuItem[]) => {
    return items.map(item => {
      // 如果有子菜单
      if (item.children) {
        return {
          key: item.key,
          icon: item.icon,
          label: (
            <div className={styles.menuItemLabel} title={typeof item.description === 'string' ? item.description : typeof item.label === 'string' ? item.label : undefined}>
              <span className="truncate">{item.label}</span>
            </div>
          ),
          children: item.children.map((child: MenuItem) => ({
            key: child.key,
            icon: child.icon,
            label: (
              <div className={styles.menuItemLabel}>
                <span className="truncate">{child.label}</span>
              </div>
            ),
            onClick: () => handleMenuClick({ key: child.key }),
          })),
        };
      }

      return {
        key: item.key,
        icon: item.icon,
        label: (
          <div className={styles.menuItemLabel} title={typeof item.description === 'string' ? item.description : typeof item.label === 'string' ? item.label : undefined}>
            <span className="truncate">{item.label}</span>
            {item.badge && (
              <Badge count={item.badge} size="small" className={styles.menuItemBadge} />
            )}
          </div>
        ),
        onClick: () => handleMenuClick({ key: item.key }),
        className: styles.menuItem,
      };
    });
  };

  return (
    <Sider
      trigger={null}
      collapsible
      collapsed={collapsed}
      onCollapse={onCollapse}
      breakpoint={LAYOUT_CONFIG.sider.breakpoint}
      collapsedWidth={LAYOUT_CONFIG.sider.collapsedWidth}
      width={LAYOUT_CONFIG.sider.width}
      className={styles.sider}
      style={{
        borderRight: `1px solid ${token.colorBorder}`,
        zIndex: LAYOUT_CONFIG.zIndex.sider,
      }}
    >
      {/* Logo 区域 */}
      <div className={`${styles.logoArea} ${collapsed ? styles.logoAreaCollapsed : ''}`}>
        <div className={styles.logoIcon}>IT</div>
        {!collapsed && (
          <div className={styles.logoTextContainer}>
            <div className={styles.logoText}>ITSM</div>
            <div className={styles.logoSubtext}>系统</div>
          </div>
        )}
      </div>

      {/* 主菜单 */}
      <div className={styles.mainMenu} style={{ flex: 1, overflowY: 'auto' }}>
        <Menu
          mode="inline"
          inlineIndent={24}
          selectedKeys={[pathname]}
          className={styles.customMenu}
          items={renderMenuItems(mainMenus)}
          theme="light"
        />
      </div>

      {/* 管理员菜单 */}
      {isAdmin && (
        <div className={styles.adminMenuContainer}>
          {!collapsed && <div className={styles.adminMenuHeader}>管理功能</div>}
          <div className={styles.adminMenu}>
            <Menu
              mode="inline"
              inlineIndent={24}
              selectedKeys={[pathname]}
              className={styles.customMenu}
              items={renderMenuItems(adminMenus)}
              theme="light"
            />
          </div>
        </div>
      )}

      {/* 底部用户信息 */}
      {!collapsed && (
        <div className={styles.userInfoContainer}>
          <div className={styles.userInfo}>
            <div className={styles.userAvatar}>{user?.name?.[0] || user?.username?.[0] || 'U'}</div>
            <div className={styles.userDetails}>
              <div className={styles.userName}>{user?.name || user?.username}</div>
              <div className={styles.userRole}>{user?.role || 'user'}</div>
            </div>
          </div>
        </div>
      )}
    </Sider>
  );
};
