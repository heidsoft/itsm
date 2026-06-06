/**
 * 菜单配置
 * ITIL v4 标准菜单结构
 */

import { iconStyle, getIconByName } from './icons';

// 菜单项接口定义
export interface MenuItem {
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
export interface MenuConfig {
  main: MenuItem[];
  admin: MenuItem[];
}

/**
 * 获取静态菜单配置 - ITIL v4 标准结构
 */
export function getMenuConfig(): MenuConfig {
  return {
    main: [
      // ===== 服务运营 =====
      {
        key: '/dashboard',
        icon: getIconByName('LayoutDashboard')!,
        label: '服务台',
        path: '/dashboard',
        permission: 'dashboard:view',
        description: '服务台概览',
      },
      {
        key: '/service-requests',
        icon: getIconByName('FileText')!,
        label: '服务请求',
        path: '/service-requests',
        permission: 'ticket:read',
        description: '服务请求管理',
        children: [
          {
            key: '/service-requests/list',
            icon: getIconByName('FileText')!,
            label: '服务请求列表',
            path: '/service-requests',
            permission: 'ticket:read',
          },
          {
            key: '/tickets/templates',
            icon: getIconByName('FileText')!,
            label: '服务请求模板',
            path: '/tickets/templates',
            permission: 'ticket:write',
          },
          {
            key: '/tickets/analytics',
            icon: getIconByName('BarChart3')!,
            label: '工单统计',
            path: '/tickets/analytics',
            permission: 'ticket:read',
          },
        ],
      },
      {
        key: '/my-requests',
        icon: getIconByName('User')!,
        label: '我的请求',
        path: '/my-requests',
        permission: 'ticket:read',
        description: '我的服务请求',
      },
      {
        key: '/incidents',
        icon: getIconByName('AlertCircle')!,
        label: '事件管理',
        path: '/incidents',
        permission: 'incident:read',
        description: '事件管理',
        children: [
          {
            key: '/incidents/list',
            icon: getIconByName('AlertCircle')!,
            label: '事件列表',
            path: '/incidents',
            permission: 'incident:read',
          },
          {
            key: '/incidents/create',
            icon: getIconByName('Plus')!,
            label: '新建事件',
            path: '/incidents/create',
            permission: 'incident:write',
          },
        ],
      },
      {
        key: '/problems',
        icon: getIconByName('HelpCircle')!,
        label: '问题管理',
        path: '/problems',
        permission: 'problem:read',
        description: '问题管理',
        children: [
          {
            key: '/problems/list',
            icon: getIconByName('HelpCircle')!,
            label: '问题列表',
            path: '/problems',
            permission: 'problem:read',
          },
          {
            key: '/problems/known-errors',
            icon: getIconByName('AlertCircle')!,
            label: '已知错误',
            path: '/problems/known-errors',
            permission: 'problem:read',
          },
        ],
      },
      {
        key: '/changes',
        icon: getIconByName('BarChart3')!,
        label: '变更管理',
        path: '/changes',
        permission: 'change:read',
        description: '变更管理',
        children: [
          {
            key: '/changes/list',
            icon: getIconByName('BarChart3')!,
            label: '变更列表',
            path: '/changes',
            permission: 'change:read',
          },
          {
            key: '/changes/new',
            icon: getIconByName('Plus')!,
            label: '新建变更',
            path: '/changes/new',
            permission: 'change:write',
          },
        ],
      },
      // ===== 服务保障 =====
      {
        key: '/knowledge',
        icon: getIconByName('Book')!,
        label: '知识库',
        path: '/knowledge',
        permission: 'knowledge:read',
        description: '知识库管理',
        children: [
          {
            key: '/knowledge/list',
            icon: getIconByName('Book')!,
            label: '知识库',
            path: '/knowledge',
            permission: 'knowledge:read',
          },
          {
            key: '/knowledge/articles',
            icon: getIconByName('FileText')!,
            label: '文章管理',
            path: '/knowledge/articles',
            permission: 'knowledge:write',
          },
          {
            key: '/knowledge/articles/create',
            icon: getIconByName('Plus')!,
            label: '新建文章',
            path: '/knowledge/articles/create',
            permission: 'knowledge:write',
          },
        ],
      },
      {
        key: '/service-catalog',
        icon: getIconByName('BookOpen')!,
        label: '服务目录',
        path: '/service-catalog',
        permission: 'service:read',
        description: '服务目录',
        children: [
          {
            key: '/service-catalog/list',
            icon: getIconByName('BookOpen')!,
            label: '服务目录',
            path: '/service-catalog',
            permission: 'service:read',
          },
          {
            key: '/service-catalog/request',
            icon: getIconByName('FileText')!,
            label: '服务请求',
            path: '/service-catalog/request',
            permission: 'service:read',
          },
          {
            key: '/service-catalog/approvals',
            icon: getIconByName('CheckCircle')!,
            label: '待我审批',
            path: '/service-catalog/approvals',
            permission: 'service:read',
          },
        ],
      },
      {
        key: '/cmdb',
        icon: getIconByName('Database')!,
        label: 'CMDB',
        path: '/cmdb',
        permission: 'cmdb:read',
        description: '配置管理数据库',
        children: [
          {
            key: '/cmdb/cis/list',
            icon: getIconByName('Server')!,
            label: '配置项列表',
            path: '/cmdb',
            permission: 'cmdb:read',
          },
          {
            key: '/cmdb/cis/create',
            icon: getIconByName('Plus')!,
            label: '新建CI',
            path: '/cmdb/cis/create',
            permission: 'cmdb:write',
          },
          {
            key: '/cmdb/cloud-resources',
            icon: getIconByName('Cloud')!,
            label: '云资源',
            path: '/cmdb/cloud-resources',
            permission: 'cmdb:read',
          },
          {
            key: '/cmdb/cloud-accounts',
            icon: getIconByName('Key')!,
            label: '云账号',
            path: '/cmdb/cloud-accounts',
            permission: 'cmdb:manage',
          },
          {
            key: '/cmdb/cloud-services',
            icon: getIconByName('Boxes')!,
            label: '云服务',
            path: '/cmdb/cloud-services',
            permission: 'cmdb:read',
          },
          {
            key: '/cmdb/reconciliation',
            icon: getIconByName('RefreshCw')!,
            label: '同步对账',
            path: '/cmdb/reconciliation',
          },
          {
            key: '/cmdb/topology',
            icon: getIconByName('Share2')!,
            label: '拓扑图',
            path: '/cmdb/topology',
            permission: 'cmdb:read',
          },
        ],
      },
      {
        key: '/assets',
        icon: getIconByName('Monitor')!,
        label: '资产管理',
        path: '/assets',
        permission: 'asset:read',
        description: 'IT资产管理',
        children: [
          {
            key: '/assets/list',
            icon: getIconByName('Server')!,
            label: '资产列表',
            path: '/assets',
            permission: 'asset:read',
          },
          {
            key: '/assets/new',
            icon: getIconByName('Plus')!,
            label: '新建资产',
            path: '/assets/new',
            permission: 'asset:write',
          },
          {
            key: '/licenses',
            icon: getIconByName('Key')!,
            label: '软件许可证',
            path: '/licenses',
            permission: 'license:manage',
          },
        ],
      },
      // ===== 报告分析 =====
      {
        key: '/sla',
        icon: getIconByName('Calendar')!,
        label: 'SLA管理',
        path: '/sla',
        permission: 'sla:read',
        description: 'SLA监控与配置',
        children: [
          {
            key: '/sla/overview',
            icon: getIconByName('Activity')!,
            label: 'SLA概览',
            path: '/sla',
            permission: 'sla:read',
          },
          {
            key: '/sla-dashboard',
            icon: getIconByName('BarChart3')!,
            label: 'SLA仪表盘',
            path: '/sla-dashboard',
            permission: 'sla:read',
          },
          {
            key: '/sla-monitor',
            icon: getIconByName('Activity')!,
            label: 'SLA监控',
            path: '/sla-monitor',
            permission: 'sla:read',
          },
        ],
      },
      {
        key: '/reports',
        icon: getIconByName('TrendingUp')!,
        label: '报表中心',
        path: '/reports',
        permission: 'report:read',
        description: '报表与分析',
        children: [
          {
            key: '/reports/overview',
            icon: getIconByName('TrendingUp')!,
            label: '报表中心',
            path: '/reports',
            permission: 'report:read',
          },
          {
            key: '/reports/sla-performance',
            icon: getIconByName('Calendar')!,
            label: 'SLA性能',
            path: '/reports/sla-performance',
            permission: 'report:read',
          },
          {
            key: '/reports/cmdb-quality',
            icon: getIconByName('Database')!,
            label: 'CMDB质量',
            path: '/reports/cmdb-quality',
            permission: 'report:read',
          },
          {
            key: '/reports/incident-trends',
            icon: getIconByName('AlertCircle')!,
            label: '事件趋势',
            path: '/reports/incident-trends',
            permission: 'report:read',
          },
          {
            key: '/reports/change-success',
            icon: getIconByName('CheckCircle')!,
            label: '变更成功率',
            path: '/reports/change-success',
            permission: 'report:read',
          },
          {
            key: '/reports/problem-efficiency',
            icon: getIconByName('HelpCircle')!,
            label: '问题效率',
            path: '/reports/problem-efficiency',
            permission: 'report:read',
          },
        ],
      },
      // ===== 自动化 =====
      {
        key: '/workflow',
        icon: getIconByName('GitMerge')!,
        label: '工作流',
        path: '/workflow',
        permission: 'workflow:read',
        description: '工作流自动化',
        children: [
          {
            key: '/workflow/list',
            icon: getIconByName('List')!,
            label: '工作流列表',
            path: '/workflow',
            permission: 'workflow:read',
          },
          {
            key: '/workflow/designer',
            icon: getIconByName('Edit')!,
            label: '流程设计器',
            path: '/workflow/designer',
            permission: 'workflow:write',
          },
          {
            key: '/workflow/instances',
            icon: getIconByName('Play')!,
            label: '流程实例',
            path: '/workflow/instances',
            permission: 'workflow:read',
          },
          {
            key: '/workflow/versions',
            icon: getIconByName('History')!,
            label: '版本管理',
            path: '/workflow/versions',
            permission: 'workflow:write',
          },
          {
            key: '/workflow/dashboard',
            icon: getIconByName('Activity')!,
            label: '监控仪表盘',
            path: '/workflow/dashboard',
            permission: 'workflow:read',
          },
          {
            key: '/workflow/automation',
            icon: getIconByName('Zap')!,
            label: '自动化规则',
            path: '/workflow/automation',
            permission: 'workflow:write',
          },
        ],
      },
      // ===== AI =====
      {
        key: '/ai/chat',
        icon: getIconByName('Bot')!,
        label: 'AI助手',
        path: '/ai/chat',
        permission: 'ai:use',
        description: 'AI 智能助手',
        children: [
          {
            key: '/ai/chat/list',
            icon: getIconByName('MessageSquare')!,
            label: 'AI对话',
            path: '/ai/chat',
            permission: 'ai:use',
          },
          {
            key: '/tickets/ai-create',
            icon: getIconByName('Sparkles')!,
            label: 'AI创建工单',
            path: '/tickets/ai-create',
            permission: 'ai:use',
          },
        ],
      },
      // ===== 扩展模块 =====
      {
        key: '/approvals/pending',
        icon: getIconByName('CheckCircle')!,
        label: '待我审批',
        path: '/approvals/pending',
        permission: 'approval:read',
        description: '待我审批',
      },
      // ===== MSP与发布 =====
      {
        key: '/msp',
        icon: getIconByName('Building')!,
        label: '客户管理',
        path: '/msp',
        permission: 'msp:read',
        description: '客户管理(MSP)',
        children: [
          {
            key: '/msp/list',
            icon: getIconByName('Building')!,
            label: '客户列表',
            path: '/msp',
            permission: 'msp:read',
          },
          {
            key: '/msp/management',
            icon: getIconByName('Settings')!,
            label: '客户管理',
            path: '/msp/management',
            permission: 'msp:manage',
          },
        ],
      },
      {
        key: '/releases',
        icon: getIconByName('Rocket')!,
        label: '发布管理',
        path: '/releases',
        permission: 'release:read',
        description: '发布管理',
        children: [
          {
            key: '/releases/list',
            icon: getIconByName('Rocket')!,
            label: '发布列表',
            path: '/releases',
            permission: 'release:read',
          },
          {
            key: '/releases/new',
            icon: getIconByName('Plus')!,
            label: '新建发布',
            path: '/releases/new',
            permission: 'release:write',
          },
        ],
      },
    ],
    admin: [
      // ===== 系统管理 =====
      {
        key: '/admin',
        icon: getIconByName('Settings')!,
        label: '系统管理',
        path: '/admin',
        permission: 'admin:write',
        description: '系统管理',
        children: [
          {
            key: '/admin/overview',
            icon: getIconByName('LayoutDashboard')!,
            label: '系统概览',
            path: '/admin/overview',
            permission: 'admin:write',
          },
          {
            key: '/admin/users',
            icon: getIconByName('Users')!,
            label: '用户管理',
            path: '/admin/users',
            permission: 'user:read',
          },
          {
            key: '/admin/roles',
            icon: getIconByName('Shield')!,
            label: '角色管理',
            path: '/admin/roles',
            permission: 'role:read',
          },
          {
            key: '/admin/groups',
            icon: getIconByName('Users')!,
            label: '组管理',
            path: '/admin/groups',
            permission: 'group:read',
          },
          {
            key: '/admin/tenants',
            icon: getIconByName('Building')!,
            label: '租户管理',
            path: '/admin/tenants',
            permission: 'tenant:manage',
          },
          {
            key: '/admin/departments',
            icon: getIconByName('Building')!,
            label: '部门管理',
            path: '/admin/departments',
            permission: 'department:manage',
          },
          {
            key: '/admin/teams',
            icon: getIconByName('Users')!,
            label: '团队管理',
            path: '/admin/teams',
            permission: 'team:manage',
          },
          {
            key: '/admin/ticket-categories',
            icon: getIconByName('Tag')!,
            label: '工单分类',
            path: '/admin/ticket-categories',
            permission: 'ticket:category:manage',
          },
          {
            key: '/admin/tickets/assignment-rules',
            icon: getIconByName('GitBranch')!,
            label: '工单分配规则',
            path: '/admin/tickets/assignment-rules',
            permission: 'ticket:manage',
          },
          {
            key: '/admin/tickets/automation-rules',
            icon: getIconByName('Zap')!,
            label: '自动化规则',
            path: '/admin/tickets/automation-rules',
            permission: 'ticket:manage',
          },
          {
            key: '/admin/approvals',
            icon: getIconByName('GitMerge')!,
            label: '审批管理',
            path: '/admin/approvals',
            permission: 'approval:manage',
          },
          {
            key: '/admin/approval-chains',
            icon: getIconByName('Link')!,
            label: '审批链',
            path: '/admin/approval-chains',
            permission: 'approval:manage',
          },
          {
            key: '/admin/permissions',
            icon: getIconByName('Lock')!,
            label: '权限管理',
            path: '/admin/permissions',
            permission: 'permission:manage',
          },
          {
            key: '/admin/system-config',
            icon: getIconByName('Settings')!,
            label: '系统配置',
            path: '/admin/system-config',
            permission: 'system:config',
          },
          {
            key: '/admin/notifications',
            icon: getIconByName('Bell')!,
            label: '通知配置',
            path: '/notifications',
            permission: 'system:config',
          },
          {
            key: '/admin/audit-logs',
            icon: getIconByName('ClipboardList')!,
            label: '操作日志',
            path: '/admin/overview',
            permission: 'system:audit',
          },
          {
            key: '/admin/cmdb-types',
            icon: getIconByName('Database')!,
            label: 'CMDB 类型',
            path: '/admin/overview',
            permission: 'cmdb:manage',
          },
          {
            key: '/admin/escalation-rules',
            icon: getIconByName('AlertTriangle')!,
            label: '升级规则',
            path: '/admin/escalation-rules',
            permission: 'escalation:manage',
          },
          {
            key: '/admin/service-catalogs',
            icon: getIconByName('Boxes')!,
            label: '服务目录',
            path: '/admin/service-catalogs',
            permission: 'catalog:manage',
          },
          {
            key: '/admin/sla-definitions',
            icon: getIconByName('Clock')!,
            label: 'SLA 定义',
            path: '/admin/sla-definitions',
            permission: 'sla:manage',
          },
          {
            key: '/admin/workflows',
            icon: getIconByName('GitBranch')!,
            label: '工作流',
            path: '/admin/workflows',
            permission: 'workflow:manage',
          },
        ],
      },
    ],
  };
}
