import React from 'react';
import { Box, Text, Newline } from 'ink';
import { LoginCommand } from './commands/login.js';
import { TicketsCommand } from './commands/tickets.js';
import { TicketViewCommand } from './commands/ticket.js';
import { SearchCommand } from './commands/search.js';
import { MeCommand } from './commands/me.js';
import { IncidentListCommand, IncidentViewCommand, IncidentTriageCommand, } from './commands/incident.js';
import { ChangeListCommand, ChangeViewCommand } from './commands/change.js';
import { CmdbListCommand, CmdbViewCommand } from './commands/cmdb.js';
import { KnowledgeListCommand, KnowledgeViewCommand, KnowledgeSearchCommand, } from './commands/knowledge.js';
import { SlaStatsCommand, SlaOverdueCommand } from './commands/sla.js';
import { WorkflowInstancesCommand, WorkflowTasksCommand, WorkflowCompleteCommand, ApprovalListCommand, ApprovalActionCommand, } from './commands/workflow.js';
import { NotificationListCommand, NotificationReadCommand, } from './commands/notification.js';
import { ConnectorListCommand, ConnectorTestCommand, ConnectorHealthCommand, ConnectorSendCommand, } from './commands/connector.js';
import { DashboardOverviewCommand } from './commands/admin.js';
import { isLoggedIn, clearCredentials } from './lib/credentials.js';
const HelpScreen = () => (React.createElement(Box, { flexDirection: "column", padding: 1 },
    React.createElement(Text, { bold: true, color: "cyan" }, "ITSM CLI v1.0.0"),
    React.createElement(Text, null, "AI-Native ITSM \u00B7 React for CLIs \u00B7 built with Ink"),
    React.createElement(Newline, null),
    React.createElement(Text, { bold: true }, "Auth"),
    React.createElement(Text, { dimColor: true }, "  itsm login -u <user> -p <pass>       Login"),
    React.createElement(Text, { dimColor: true }, "  itsm me                              Who am I"),
    React.createElement(Text, { dimColor: true }, "  itsm logout                          Logout"),
    React.createElement(Newline, null),
    React.createElement(Text, { bold: true }, "ITILv3 \u2014 Ticket / Incident / Change"),
    React.createElement(Text, { dimColor: true }, "  itsm tickets [--status] [--priority]      List tickets"),
    React.createElement(Text, { dimColor: true }, "  itsm ticket <id>                       View ticket"),
    React.createElement(Text, { dimColor: true }, "  itsm incident list [--status] [--priority]List incidents"),
    React.createElement(Text, { dimColor: true }, "  itsm incident <id>                     View incident"),
    React.createElement(Text, { dimColor: true }, "  itsm incident triage --title X --desc Y   AI triage"),
    React.createElement(Text, { dimColor: true }, "  itsm change list [--status]               List changes"),
    React.createElement(Text, { dimColor: true }, "  itsm change <id>                       View change"),
    React.createElement(Newline, null),
    React.createElement(Text, { bold: true }, "CMDB / Knowledge / SLA"),
    React.createElement(Text, { dimColor: true }, "  itsm cmdb list [--type] [--status]         List CIs"),
    React.createElement(Text, { dimColor: true }, "  itsm cmdb <id>                          View CI"),
    React.createElement(Text, { dimColor: true }, "  itsm knowledge list                       List articles"),
    React.createElement(Text, { dimColor: true }, "  itsm knowledge search <q>               Search KB"),
    React.createElement(Text, { dimColor: true }, "  itsm knowledge <id>                     Read article"),
    React.createElement(Text, { dimColor: true }, "  itsm sla stats                            SLA dashboard"),
    React.createElement(Text, { dimColor: true }, "  itsm sla overdue                          Overdue tickets"),
    React.createElement(Newline, null),
    React.createElement(Text, { bold: true }, "Workflow / Approvals / Notifications"),
    React.createElement(Text, { dimColor: true }, "  itsm workflow instances                   Process instances"),
    React.createElement(Text, { dimColor: true }, "  itsm workflow tasks                       My tasks"),
    React.createElement(Text, { dimColor: true }, "  itsm workflow complete <id> --outcome OK Complete task"),
    React.createElement(Text, { dimColor: true }, "  itsm workflow approve <id> [--comment]  Approve"),
    React.createElement(Text, { dimColor: true }, "  itsm workflow reject <id> [--comment]   Reject"),
    React.createElement(Text, { dimColor: true }, "  itsm notification list [--unread]         List notifications"),
    React.createElement(Text, { dimColor: true }, "  itsm notification read <id>             Mark read"),
    React.createElement(Newline, null),
    React.createElement(Text, { bold: true }, "Connectors / Plugin / Skill / IM Market"),
    React.createElement(Text, { dimColor: true }, "  itsm connector list                       List market"),
    React.createElement(Text, { dimColor: true }, "  itsm connector test <name>              Test connector"),
    React.createElement(Text, { dimColor: true }, "  itsm connector send <name> --channel X --message Y"),
    React.createElement(Text, { dimColor: true }, "  itsm connector health                     Health check"),
    React.createElement(Newline, null),
    React.createElement(Text, { bold: true }, "Admin / Misc"),
    React.createElement(Text, { dimColor: true }, "  itsm dashboard                            Overview"),
    React.createElement(Text, { dimColor: true }, "  itsm search <query>                     Global search"),
    React.createElement(Newline, null),
    React.createElement(Text, { dimColor: true }, "Backend: http://localhost:8090")));
const Router = ({ input, flags }) => {
    const [cmd, ...args] = input;
    const sub = args[0];
    switch (cmd) {
        case 'login':
            return React.createElement(LoginCommand, { username: flags.username ?? '', password: flags.password ?? '' });
        case 'me':
            return React.createElement(MeCommand, null);
        case 'logout':
            clearCredentials();
            return React.createElement(Text, { color: "yellow" }, "\u2713 Logged out");
        case 'tickets':
            return React.createElement(TicketsCommand, { status: flags.status, priority: flags.priority, page: flags.page });
        case 'ticket': {
            if (sub === 'list')
                return React.createElement(TicketsCommand, { status: flags.status, priority: flags.priority, page: flags.page });
            const id = parseInt(sub ?? '0', 10);
            if (id > 0)
                return React.createElement(TicketViewCommand, { id: id });
            return React.createElement(Text, { color: "red" }, "Usage: itsm ticket <id> | list");
        }
        case 'incident': {
            if (sub === 'list' || sub === undefined) {
                return React.createElement(IncidentListCommand, { status: flags.status, priority: flags.priority, page: flags.page });
            }
            if (sub === 'triage') {
                if (!flags.title || !flags.description)
                    return React.createElement(Text, { color: "red" }, "Usage: itsm incident triage --title X --description Y");
                return React.createElement(IncidentTriageCommand, { title: flags.title, description: flags.description });
            }
            const id = parseInt(sub, 10);
            if (id > 0)
                return React.createElement(IncidentViewCommand, { id: id });
            return React.createElement(Text, { color: "red" }, "Usage: itsm incident <list|id|triage>");
        }
        case 'change': {
            if (sub === 'list' || sub === undefined) {
                return React.createElement(ChangeListCommand, { status: flags.status, page: flags.page });
            }
            const id = parseInt(sub, 10);
            if (id > 0)
                return React.createElement(ChangeViewCommand, { id: id });
            return React.createElement(Text, { color: "red" }, "Usage: itsm change <list|id>");
        }
        case 'cmdb': {
            if (sub === 'list' || sub === undefined) {
                return React.createElement(CmdbListCommand, { type: flags.type, status: flags.status, page: flags.page });
            }
            const id = parseInt(sub, 10);
            if (id > 0)
                return React.createElement(CmdbViewCommand, { id: id });
            return React.createElement(Text, { color: "red" }, "Usage: itsm cmdb <list|id>");
        }
        case 'knowledge': {
            if (sub === 'search') {
                return React.createElement(KnowledgeSearchCommand, { q: args.slice(1).join(' ') || flags.q || '' });
            }
            if (sub === 'list' || sub === undefined) {
                return React.createElement(KnowledgeListCommand, { category: flags.category, page: flags.page });
            }
            const id = parseInt(sub, 10);
            if (id > 0)
                return React.createElement(KnowledgeViewCommand, { id: id });
            return React.createElement(Text, { color: "red" }, "Usage: itsm knowledge <list|search|id>");
        }
        case 'sla': {
            if (sub === 'overdue')
                return React.createElement(SlaOverdueCommand, null);
            if (sub === 'stats' || sub === undefined)
                return React.createElement(SlaStatsCommand, null);
            return React.createElement(Text, { color: "red" }, "Usage: itsm sla <stats|overdue>");
        }
        case 'workflow': {
            if (sub === 'instances')
                return React.createElement(WorkflowInstancesCommand, { page: flags.page });
            if (sub === 'tasks')
                return React.createElement(WorkflowTasksCommand, null);
            if (sub === 'complete' && args[1]) {
                return React.createElement(WorkflowCompleteCommand, { id: args[1], outcome: flags.outcome ?? 'OK', comment: flags.comment });
            }
            if (sub === 'approve' && args[1]) {
                return React.createElement(ApprovalActionCommand, { id: args[1], action: "approve", comment: flags.comment });
            }
            if (sub === 'reject' && args[1]) {
                return React.createElement(ApprovalActionCommand, { id: args[1], action: "reject", comment: flags.comment });
            }
            if (sub === 'approvals' || sub === 'pending') {
                return React.createElement(ApprovalListCommand, { status: flags.status });
            }
            return React.createElement(Text, { color: "red" }, "Usage: itsm workflow <instances|tasks|complete|approve|reject|approvals>");
        }
        case 'approvals': {
            if (sub === 'pending' || sub === undefined)
                return React.createElement(ApprovalListCommand, { status: flags.status });
            if (sub === 'approve' && args[1])
                return React.createElement(ApprovalActionCommand, { id: args[1], action: "approve", comment: flags.comment });
            if (sub === 'reject' && args[1])
                return React.createElement(ApprovalActionCommand, { id: args[1], action: "reject", comment: flags.comment });
            return React.createElement(Text, { color: "red" }, "Usage: itsm approvals <pending|approve|reject>");
        }
        case 'notification': {
            if (sub === 'read' && args[1]) {
                const id = parseInt(args[1], 10);
                if (id > 0)
                    return React.createElement(NotificationReadCommand, { id: id });
            }
            if (sub === 'list' || sub === undefined) {
                return React.createElement(NotificationListCommand, { unread: flags.unread, page: flags.page });
            }
            return React.createElement(Text, { color: "red" }, "Usage: itsm notification <list|read id>");
        }
        case 'connector': {
            if (sub === 'list' || sub === undefined)
                return React.createElement(ConnectorListCommand, null);
            if (sub === 'health')
                return React.createElement(ConnectorHealthCommand, null);
            if (sub === 'test' && args[1])
                return React.createElement(ConnectorTestCommand, { name: args[1] });
            if (sub === 'send' && args[1] && flags.channel && flags.message) {
                return React.createElement(ConnectorSendCommand, { name: args[1], channel: flags.channel, message: flags.message });
            }
            return React.createElement(Text, { color: "red" }, "Usage: itsm connector <list|health|test name|send name --channel X --message Y>");
        }
        case 'feishu': {
            // alias of connector for the IM flow
            if (sub === 'list' || sub === undefined)
                return React.createElement(ConnectorListCommand, null);
            if (sub === 'test' && args[1])
                return React.createElement(ConnectorTestCommand, { name: args[1] });
            if (sub === 'send' && args[1] && flags.channel && flags.message) {
                return React.createElement(ConnectorSendCommand, { name: args[1], channel: flags.channel, message: flags.message });
            }
            return React.createElement(Text, { color: "red" }, "Usage: itsm feishu <list|test name|send name --channel X --message Y>");
        }
        case 'dashboard':
            return React.createElement(DashboardOverviewCommand, null);
        case 'search':
            return React.createElement(SearchCommand, { query: args.join(' ') || flags.q || '', page: flags.page });
        case undefined:
            return React.createElement(TicketsCommand, null);
        default:
            return React.createElement(Text, { color: "red" },
                "Unknown command: ",
                cmd,
                ". Run `itsm --help`.");
    }
};
export default function App({ cli }) {
    const { input, flags } = cli;
    if (flags.help || flags.version || input.length === 0 || input[0] === 'help') {
        return React.createElement(HelpScreen, null);
    }
    const [cmd] = input;
    if (cmd === 'login')
        return React.createElement(Router, { input: input, flags: flags });
    if (!isLoggedIn() && cmd !== 'logout') {
        return (React.createElement(Box, { flexDirection: "column", padding: 1 },
            React.createElement(Text, { color: "red" }, "Not logged in"),
            React.createElement(Text, null, "Run: itsm login -u <username> -p <password>")));
    }
    return React.createElement(Router, { input: input, flags: flags });
}
