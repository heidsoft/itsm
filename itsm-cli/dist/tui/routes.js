/**
 * TUI route registry.
 *
 * The interactive surface used to be a hand-written `switch` in `index.tsx`,
 * separate from the agent command registry. That made drift structural: adding
 * a command to the registry never reached the TUI, and the two tables could
 * disagree about which verbs exist. Routes now live here as data, help is
 * generated from the same list, and `GenericCommand` (see `GenericCommand.tsx`)
 * executes anything the curated list does not cover through the *agent*
 * registry — so every one of the CLI's commands is reachable from the TUI,
 * with bespoke screens only where a table/detail view earns its keep.
 *
 * Rendering functions receive the parsed flags but must not mutate server
 * state directly; the components fetch through the shared `apiClient`.
 */
import React from 'react';
import { Text } from 'ink';
import { LoginCommand } from '../commands/login.js';
import { MeCommand } from '../commands/me.js';
import { clearCredentials } from '../lib/credentials.js';
import { TicketsCommand } from '../commands/tickets.js';
import { TicketViewCommand } from '../commands/ticket.js';
import { IncidentListCommand, IncidentTriageCommand, IncidentViewCommand } from '../commands/incident.js';
import { ProblemListCommand, ProblemViewCommand } from '../commands/problem.js';
import { ChangeListCommand, ChangeViewCommand } from '../commands/change.js';
import { ReleaseListCommand, ReleaseViewCommand } from '../commands/release.js';
import { RequestListCommand, RequestViewCommand } from '../commands/request.js';
import { CmdbListCommand, CmdbViewCommand } from '../commands/cmdb.js';
import { KnowledgeListCommand, KnowledgeSearchCommand, KnowledgeViewCommand } from '../commands/knowledge.js';
import { SlaOverdueCommand, SlaStatsCommand } from '../commands/sla.js';
import { ApprovalActionCommand, ApprovalListCommand, WorkflowCompleteCommand, WorkflowInstancesCommand, WorkflowTasksCommand, } from '../commands/workflow.js';
import { NotificationListCommand, NotificationReadCommand } from '../commands/notification.js';
import { ConnectorHealthCommand, ConnectorListCommand, ConnectorSendCommand, ConnectorTestCommand, } from '../commands/connector.js';
import { DashboardOverviewCommand } from '../commands/admin.js';
import { SearchCommand } from '../commands/search.js';
function domain(input, name) {
    return input[0] === name;
}
/** `itsm x` or `itsm x list` — the conventional list entrypoint. */
function listOf(input, name) {
    return domain(input, name) && (input[1] === undefined || input[1] === 'list');
}
function verb(input, name, action) {
    return domain(input, name) && input[1] === action;
}
/** Numeric detail id; `ticket list` must not be mistaken for `ticket <id>`. */
function idOf(input, name) {
    if (!domain(input, name))
        return undefined;
    const raw = input[1];
    if (!raw || !/^\d+$/.test(raw))
        return undefined;
    const id = Number(raw);
    return id > 0 ? id : undefined;
}
export const TUI_ROUTES = [
    {
        id: 'login',
        group: '认证与身份',
        usage: 'itsm login -u <user>',
        summary: '登录（密码走掩码提示或 stdin，永不进 argv）',
        match: (input) => domain(input, 'login'),
        render: ({ flags }) => React.createElement(LoginCommand, { username: flags.username ?? '', password: flags.password }),
    },
    {
        id: 'me',
        group: '认证与身份',
        usage: 'itsm me',
        summary: '查看当前身份',
        match: (input) => domain(input, 'me'),
        render: () => React.createElement(MeCommand, null),
    },
    {
        id: 'logout',
        group: '认证与身份',
        usage: 'itsm logout',
        summary: '清除本地凭据',
        match: (input) => domain(input, 'logout'),
        render: () => {
            clearCredentials();
            return React.createElement(Text, { color: "yellow" }, "\u2713 Logged out");
        },
    },
    {
        id: 'ticket.list',
        group: '工单',
        usage: 'itsm tickets [--status] [--priority] [--page]',
        summary: '工单列表',
        match: (input) => domain(input, 'tickets') || listOf(input, 'ticket'),
        render: ({ flags }) => React.createElement(TicketsCommand, { status: flags.status, priority: flags.priority, page: flags.page }),
    },
    {
        id: 'ticket.get',
        group: '工单',
        usage: 'itsm ticket <id>',
        summary: '工单详情',
        match: (input) => idOf(input, 'ticket') !== undefined,
        render: ({ input }) => React.createElement(TicketViewCommand, { id: idOf(input, 'ticket') ?? 0 }),
    },
    {
        id: 'incident.list',
        group: '事件',
        usage: 'itsm incident list [--status] [--priority] [--page]',
        summary: '事件列表',
        match: (input) => listOf(input, 'incident'),
        render: ({ flags }) => React.createElement(IncidentListCommand, { status: flags.status, priority: flags.priority, page: flags.page }),
    },
    {
        id: 'incident.get',
        group: '事件',
        usage: 'itsm incident <id>',
        summary: '事件详情',
        match: (input) => idOf(input, 'incident') !== undefined,
        render: ({ input }) => React.createElement(IncidentViewCommand, { id: idOf(input, 'incident') ?? 0 }),
    },
    {
        id: 'incident.triage',
        group: '事件',
        usage: 'itsm incident triage --title X --description Y',
        summary: 'AI 分诊（只做决策支持）',
        match: (input) => verb(input, 'incident', 'triage'),
        render: ({ flags }) => flags.title && flags.description ? (React.createElement(IncidentTriageCommand, { title: flags.title, description: flags.description })) : (React.createElement(Text, { color: "red" }, "Usage: itsm incident triage --title X --description Y")),
    },
    {
        id: 'problem.list',
        group: '问题',
        usage: 'itsm problem list [--status] [--priority] [--page]',
        summary: '问题列表（根因 / 已知错误入口）',
        match: (input) => listOf(input, 'problem'),
        render: ({ flags }) => React.createElement(ProblemListCommand, { status: flags.status, priority: flags.priority, page: flags.page }),
    },
    {
        id: 'problem.get',
        group: '问题',
        usage: 'itsm problem <id>',
        summary: '问题详情（根因 / 临时措施 / 关联）',
        match: (input) => idOf(input, 'problem') !== undefined,
        render: ({ input }) => React.createElement(ProblemViewCommand, { id: idOf(input, 'problem') ?? 0 }),
    },
    {
        id: 'change.list',
        group: '变更',
        usage: 'itsm change list [--status] [--page]',
        summary: '变更列表',
        match: (input) => listOf(input, 'change'),
        render: ({ flags }) => React.createElement(ChangeListCommand, { status: flags.status, page: flags.page }),
    },
    {
        id: 'change.get',
        group: '变更',
        usage: 'itsm change <id>',
        summary: '变更详情（回滚方案 / 风险 / 窗口）',
        match: (input) => idOf(input, 'change') !== undefined,
        render: ({ input }) => React.createElement(ChangeViewCommand, { id: idOf(input, 'change') ?? 0 }),
    },
    {
        id: 'release.list',
        group: '发布',
        usage: 'itsm release list [--status] [--type] [--page]',
        summary: '发布列表',
        match: (input) => listOf(input, 'release'),
        render: ({ flags }) => React.createElement(ReleaseListCommand, { status: flags.status, type: flags.type, page: flags.page }),
    },
    {
        id: 'release.get',
        group: '发布',
        usage: 'itsm release <id>',
        summary: '发布详情（范围 / 窗口 / 回滚程序）',
        match: (input) => idOf(input, 'release') !== undefined,
        render: ({ input }) => React.createElement(ReleaseViewCommand, { id: idOf(input, 'release') ?? 0 }),
    },
    {
        id: 'request.list',
        group: '服务请求',
        usage: 'itsm request list [--page]',
        summary: '服务请求列表',
        match: (input) => listOf(input, 'request'),
        render: ({ flags }) => React.createElement(RequestListCommand, { page: flags.page }),
    },
    {
        id: 'request.get',
        group: '服务请求',
        usage: 'itsm request <id>',
        summary: '服务请求详情（审批级别 / 交付进度）',
        match: (input) => idOf(input, 'request') !== undefined,
        render: ({ input }) => React.createElement(RequestViewCommand, { id: idOf(input, 'request') ?? 0 }),
    },
    {
        id: 'cmdb.list',
        group: 'CMDB',
        usage: 'itsm cmdb list [--type] [--status] [--page]',
        summary: '配置项列表',
        match: (input) => listOf(input, 'cmdb'),
        render: ({ flags }) => React.createElement(CmdbListCommand, { type: flags.type, status: flags.status, page: flags.page }),
    },
    {
        id: 'cmdb.get',
        group: 'CMDB',
        usage: 'itsm cmdb <id>',
        summary: '配置项详情',
        match: (input) => idOf(input, 'cmdb') !== undefined,
        render: ({ input }) => React.createElement(CmdbViewCommand, { id: idOf(input, 'cmdb') ?? 0 }),
    },
    {
        id: 'knowledge.list',
        group: '知识库',
        usage: 'itsm knowledge list [--category] [--page]',
        summary: '知识文章列表',
        match: (input) => listOf(input, 'knowledge'),
        render: ({ flags }) => React.createElement(KnowledgeListCommand, { category: flags.category, page: flags.page }),
    },
    {
        id: 'knowledge.search',
        group: '知识库',
        usage: 'itsm knowledge search <q>',
        summary: '知识检索（RAG 遵循权限与时效）',
        match: (input) => verb(input, 'knowledge', 'search'),
        render: ({ input, flags }) => (React.createElement(KnowledgeSearchCommand, { q: input.slice(2).join(' ') || flags.q || '' })),
    },
    {
        id: 'knowledge.get',
        group: '知识库',
        usage: 'itsm knowledge <id>',
        summary: '知识文章正文',
        match: (input) => idOf(input, 'knowledge') !== undefined,
        render: ({ input }) => React.createElement(KnowledgeViewCommand, { id: idOf(input, 'knowledge') ?? 0 }),
    },
    {
        id: 'sla.stats',
        group: 'SLA',
        usage: 'itsm sla stats',
        summary: 'SLA 概览',
        match: (input) => domain(input, 'sla') && (input[1] === undefined || input[1] === 'stats'),
        render: () => React.createElement(SlaStatsCommand, null),
    },
    {
        id: 'sla.overdue',
        group: 'SLA',
        usage: 'itsm sla overdue',
        summary: '超时工单',
        match: (input) => verb(input, 'sla', 'overdue'),
        render: () => React.createElement(SlaOverdueCommand, null),
    },
    {
        id: 'workflow.instances',
        group: '工作流与审批',
        usage: 'itsm workflow instances [--page]',
        summary: '流程实例',
        match: (input) => verb(input, 'workflow', 'instances'),
        render: ({ flags }) => React.createElement(WorkflowInstancesCommand, { page: flags.page }),
    },
    {
        id: 'workflow.tasks',
        group: '工作流与审批',
        usage: 'itsm workflow tasks',
        summary: '我的待办',
        match: (input) => verb(input, 'workflow', 'tasks'),
        render: () => React.createElement(WorkflowTasksCommand, null),
    },
    {
        id: 'workflow.complete',
        group: '工作流与审批',
        usage: 'itsm workflow complete <id> --outcome OK',
        summary: '完成人工任务',
        match: (input) => verb(input, 'workflow', 'complete') && input[2] !== undefined,
        render: ({ input, flags }) => (React.createElement(WorkflowCompleteCommand, { id: input[2] ?? '', outcome: flags.outcome ?? 'OK', comment: flags.comment })),
    },
    {
        id: 'approvals.pending',
        group: '工作流与审批',
        usage: 'itsm approvals pending [--status]',
        summary: '待审批列表',
        match: (input) => verb(input, 'approvals', 'pending') ||
            (domain(input, 'approvals') && input[1] === undefined) ||
            (domain(input, 'workflow') && (input[1] === 'approvals' || input[1] === 'pending')),
        render: ({ flags }) => React.createElement(ApprovalListCommand, { status: flags.status }),
    },
    {
        id: 'approvals.action',
        group: '工作流与审批',
        usage: 'itsm approvals <approve|reject> <id> [--comment]',
        summary: '审批通过 / 驳回',
        match: (input) => {
            const action = input[1];
            return ((action === 'approve' || action === 'reject') &&
                ((input[0] === 'approvals' && input[2] !== undefined) ||
                    (input[0] === 'workflow' && input[2] !== undefined)));
        },
        render: ({ input, flags }) => (React.createElement(ApprovalActionCommand, { id: input[2] ?? '', action: input[1] === 'approve' ? 'approve' : 'reject', comment: flags.comment })),
    },
    {
        id: 'notification.list',
        group: '通知',
        usage: 'itsm notification list [--unread] [--page]',
        summary: '通知列表',
        match: (input) => listOf(input, 'notification'),
        render: ({ flags }) => React.createElement(NotificationListCommand, { unread: flags.unread, page: flags.page }),
    },
    {
        id: 'notification.read',
        group: '通知',
        usage: 'itsm notification read <id>',
        summary: '标记通知已读',
        match: (input) => verb(input, 'notification', 'read') && /^\d+$/.test(input[2] ?? ''),
        render: ({ input }) => React.createElement(NotificationReadCommand, { id: Number(input[2]) }),
    },
    {
        id: 'connector.list',
        group: '连接器',
        usage: 'itsm connector list',
        summary: '连接器列表（feishu 为别名）',
        match: (input) => (domain(input, 'connector') && (input[1] === undefined || input[1] === 'list')) ||
            (domain(input, 'feishu') && (input[1] === undefined || input[1] === 'list')),
        render: () => React.createElement(ConnectorListCommand, null),
    },
    {
        id: 'connector.health',
        group: '连接器',
        usage: 'itsm connector health',
        summary: '连接器健康检查',
        match: (input) => verb(input, 'connector', 'health'),
        render: () => React.createElement(ConnectorHealthCommand, null),
    },
    {
        id: 'connector.test',
        group: '连接器',
        usage: 'itsm connector test <name>',
        summary: '测试单个连接器',
        match: (input) => (verb(input, 'connector', 'test') || verb(input, 'feishu', 'test')) && input[2] !== undefined,
        render: ({ input }) => React.createElement(ConnectorTestCommand, { name: input[2] ?? '' }),
    },
    {
        id: 'connector.send',
        group: '连接器',
        usage: 'itsm connector send <name> --channel X --message Y',
        summary: '通过连接器发送消息',
        match: (input) => (verb(input, 'connector', 'send') || verb(input, 'feishu', 'send')) && input[2] !== undefined,
        render: ({ input, flags }) => flags.channel && flags.message ? (React.createElement(ConnectorSendCommand, { name: input[2] ?? '', channel: flags.channel, message: flags.message })) : (React.createElement(Text, { color: "red" }, "Usage: itsm connector send <name> --channel X --message Y")),
    },
    {
        id: 'dashboard',
        group: '管理',
        usage: 'itsm dashboard',
        summary: '运营概览',
        match: (input) => domain(input, 'dashboard'),
        render: () => React.createElement(DashboardOverviewCommand, null),
    },
    {
        id: 'search',
        group: '搜索',
        usage: 'itsm search <query>',
        summary: '全局搜索',
        match: (input) => domain(input, 'search'),
        render: ({ input, flags }) => React.createElement(SearchCommand, { query: input.slice(1).join(' ') || flags.q || '', page: flags.page }),
    },
];
/** First matching route wins; specific routes must be listed before broader ones. */
export function resolveTuiRoute(input) {
    return TUI_ROUTES.find((route) => route.match(input));
}
