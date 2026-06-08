import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { LoginCommand } from './commands/login.js';
import { TicketsCommand } from './commands/tickets.js';
import { TicketViewCommand } from './commands/ticket.js';
import { SearchCommand } from './commands/search.js';
import { MeCommand } from './commands/me.js';
import {
  IncidentListCommand, IncidentViewCommand, IncidentTriageCommand,
} from './commands/incident.js';
import { ChangeListCommand, ChangeViewCommand } from './commands/change.js';
import { CmdbListCommand, CmdbViewCommand } from './commands/cmdb.js';
import {
  KnowledgeListCommand, KnowledgeViewCommand, KnowledgeSearchCommand,
} from './commands/knowledge.js';
import { SlaStatsCommand, SlaOverdueCommand } from './commands/sla.js';
import {
  WorkflowInstancesCommand, WorkflowTasksCommand, WorkflowCompleteCommand,
  ApprovalListCommand, ApprovalActionCommand,
} from './commands/workflow.js';
import {
  NotificationListCommand, NotificationReadCommand,
} from './commands/notification.js';
import {
  ConnectorListCommand, ConnectorTestCommand, ConnectorHealthCommand, ConnectorSendCommand,
} from './commands/connector.js';
import { DashboardOverviewCommand } from './commands/admin.js';
import { isLoggedIn, clearCredentials } from './lib/credentials.js';
import { apiClient } from './lib/api-client.js';

interface CliFlags {
  status?: string;
  priority?: string;
  page?: number;
  category?: string;
  type?: string;
  q?: string;
  unread?: boolean;
  channel?: string;
  comment?: string;
  outcome?: string;
  message?: string;
  title?: string;
  description?: string;
  username?: string;
  password?: string;
  help: boolean;
  version: boolean;
}

const HelpScreen: React.FC = () => (
  <Box flexDirection="column" padding={1}>
    <Text bold color="cyan">ITSM CLI v1.0.0</Text>
    <Text>AI-Native ITSM · React for CLIs · built with Ink</Text>
    <Newline />
    <Text bold>Auth</Text>
    <Text dimColor>  itsm login -u &lt;user&gt; -p &lt;pass&gt;       Login</Text>
    <Text dimColor>  itsm me                              Who am I</Text>
    <Text dimColor>  itsm logout                          Logout</Text>
    <Newline />
    <Text bold>ITILv3 — Ticket / Incident / Change</Text>
    <Text dimColor>  itsm tickets [--status] [--priority]      List tickets</Text>
    <Text dimColor>  itsm ticket &lt;id&gt;                       View ticket</Text>
    <Text dimColor>  itsm incident list [--status] [--priority]List incidents</Text>
    <Text dimColor>  itsm incident &lt;id&gt;                     View incident</Text>
    <Text dimColor>  itsm incident triage --title X --desc Y   AI triage</Text>
    <Text dimColor>  itsm change list [--status]               List changes</Text>
    <Text dimColor>  itsm change &lt;id&gt;                       View change</Text>
    <Newline />
    <Text bold>CMDB / Knowledge / SLA</Text>
    <Text dimColor>  itsm cmdb list [--type] [--status]         List CIs</Text>
    <Text dimColor>  itsm cmdb &lt;id&gt;                          View CI</Text>
    <Text dimColor>  itsm knowledge list                       List articles</Text>
    <Text dimColor>  itsm knowledge search &lt;q&gt;               Search KB</Text>
    <Text dimColor>  itsm knowledge &lt;id&gt;                     Read article</Text>
    <Text dimColor>  itsm sla stats                            SLA dashboard</Text>
    <Text dimColor>  itsm sla overdue                          Overdue tickets</Text>
    <Newline />
    <Text bold>Workflow / Approvals / Notifications</Text>
    <Text dimColor>  itsm workflow instances                   Process instances</Text>
    <Text dimColor>  itsm workflow tasks                       My tasks</Text>
    <Text dimColor>  itsm workflow complete &lt;id&gt; --outcome OK Complete task</Text>
    <Text dimColor>  itsm workflow approve &lt;id&gt; [--comment]  Approve</Text>
    <Text dimColor>  itsm workflow reject &lt;id&gt; [--comment]   Reject</Text>
    <Text dimColor>  itsm notification list [--unread]         List notifications</Text>
    <Text dimColor>  itsm notification read &lt;id&gt;             Mark read</Text>
    <Newline />
    <Text bold>Connectors / Plugin / Skill / IM Market</Text>
    <Text dimColor>  itsm connector list                       List market</Text>
    <Text dimColor>  itsm connector test &lt;name&gt;              Test connector</Text>
    <Text dimColor>  itsm connector send &lt;name&gt; --channel X --message Y</Text>
    <Text dimColor>  itsm connector health                     Health check</Text>
    <Newline />
    <Text bold>Admin / Misc</Text>
    <Text dimColor>  itsm dashboard                            Overview</Text>
    <Text dimColor>  itsm search &lt;query&gt;                     Global search</Text>
    <Newline />
    <Text dimColor>Backend: http://localhost:8090</Text>
  </Box>
);

const Router: React.FC<{ input: string[]; flags: CliFlags }> = ({ input, flags }) => {
  const [cmd, ...args] = input;
  const sub = args[0];

  switch (cmd) {
    case 'login':
      return <LoginCommand username={flags.username ?? ''} password={flags.password ?? ''} />;

    case 'me':
      return <MeCommand />;

    case 'logout':
      clearCredentials();
      return <Text color="yellow">✓ Logged out</Text>;

    case 'tickets':
      return <TicketsCommand status={flags.status} priority={flags.priority} page={flags.page} />;

    case 'ticket': {
      if (sub === 'list') return <TicketsCommand status={flags.status} priority={flags.priority} page={flags.page} />;
      const id = parseInt(sub ?? '0', 10);
      if (id > 0) return <TicketViewCommand id={id} />;
      return <Text color="red">Usage: itsm ticket &lt;id&gt; | list</Text>;
    }

    case 'incident': {
      if (sub === 'list' || sub === undefined) {
        return <IncidentListCommand status={flags.status} priority={flags.priority} page={flags.page} />;
      }
      if (sub === 'triage') {
        if (!flags.title || !flags.description) return <Text color="red">Usage: itsm incident triage --title X --description Y</Text>;
        return <IncidentTriageCommand title={flags.title} description={flags.description} />;
      }
      const id = parseInt(sub, 10);
      if (id > 0) return <IncidentViewCommand id={id} />;
      return <Text color="red">Usage: itsm incident &lt;list|id|triage&gt;</Text>;
    }

    case 'change': {
      if (sub === 'list' || sub === undefined) {
        return <ChangeListCommand status={flags.status} page={flags.page} />;
      }
      const id = parseInt(sub, 10);
      if (id > 0) return <ChangeViewCommand id={id} />;
      return <Text color="red">Usage: itsm change &lt;list|id&gt;</Text>;
    }

    case 'cmdb': {
      if (sub === 'list' || sub === undefined) {
        return <CmdbListCommand type={flags.type} status={flags.status} page={flags.page} />;
      }
      const id = parseInt(sub, 10);
      if (id > 0) return <CmdbViewCommand id={id} />;
      return <Text color="red">Usage: itsm cmdb &lt;list|id&gt;</Text>;
    }

    case 'knowledge': {
      if (sub === 'search') {
        return <KnowledgeSearchCommand q={args.slice(1).join(' ') || flags.q || ''} />;
      }
      if (sub === 'list' || sub === undefined) {
        return <KnowledgeListCommand category={flags.category} page={flags.page} />;
      }
      const id = parseInt(sub, 10);
      if (id > 0) return <KnowledgeViewCommand id={id} />;
      return <Text color="red">Usage: itsm knowledge &lt;list|search|id&gt;</Text>;
    }

    case 'sla': {
      if (sub === 'overdue') return <SlaOverdueCommand />;
      if (sub === 'stats' || sub === undefined) return <SlaStatsCommand />;
      return <Text color="red">Usage: itsm sla &lt;stats|overdue&gt;</Text>;
    }

    case 'workflow': {
      if (sub === 'instances') return <WorkflowInstancesCommand page={flags.page} />;
      if (sub === 'tasks') return <WorkflowTasksCommand />;
      if (sub === 'complete' && args[1]) {
        return <WorkflowCompleteCommand id={args[1]} outcome={flags.outcome ?? 'OK'} comment={flags.comment} />;
      }
      if (sub === 'approve' && args[1]) {
        return <ApprovalActionCommand id={args[1]} action="approve" comment={flags.comment} />;
      }
      if (sub === 'reject' && args[1]) {
        return <ApprovalActionCommand id={args[1]} action="reject" comment={flags.comment} />;
      }
      if (sub === 'approvals' || sub === 'pending') {
        return <ApprovalListCommand status={flags.status} />;
      }
      return <Text color="red">Usage: itsm workflow &lt;instances|tasks|complete|approve|reject|approvals&gt;</Text>;
    }

    case 'approvals': {
      if (sub === 'pending' || sub === undefined) return <ApprovalListCommand status={flags.status} />;
      if (sub === 'approve' && args[1]) return <ApprovalActionCommand id={args[1]} action="approve" comment={flags.comment} />;
      if (sub === 'reject' && args[1]) return <ApprovalActionCommand id={args[1]} action="reject" comment={flags.comment} />;
      return <Text color="red">Usage: itsm approvals &lt;pending|approve|reject&gt;</Text>;
    }

    case 'notification': {
      if (sub === 'read' && args[1]) {
        const id = parseInt(args[1], 10);
        if (id > 0) return <NotificationReadCommand id={id} />;
      }
      if (sub === 'list' || sub === undefined) {
        return <NotificationListCommand unread={flags.unread} page={flags.page} />;
      }
      return <Text color="red">Usage: itsm notification &lt;list|read id&gt;</Text>;
    }

    case 'connector': {
      if (sub === 'list' || sub === undefined) return <ConnectorListCommand />;
      if (sub === 'health') return <ConnectorHealthCommand />;
      if (sub === 'test' && args[1]) return <ConnectorTestCommand name={args[1]} />;
      if (sub === 'send' && args[1] && flags.channel && flags.message) {
        return <ConnectorSendCommand name={args[1]} channel={flags.channel} message={flags.message} />;
      }
      return <Text color="red">Usage: itsm connector &lt;list|health|test name|send name --channel X --message Y&gt;</Text>;
    }

    case 'feishu': {
      // alias of connector for the IM flow
      if (sub === 'list' || sub === undefined) return <ConnectorListCommand />;
      if (sub === 'test' && args[1]) return <ConnectorTestCommand name={args[1]} />;
      if (sub === 'send' && args[1] && flags.channel && flags.message) {
        return <ConnectorSendCommand name={args[1]} channel={flags.channel} message={flags.message} />;
      }
      return <Text color="red">Usage: itsm feishu &lt;list|test name|send name --channel X --message Y&gt;</Text>;
    }

    case 'dashboard':
      return <DashboardOverviewCommand />;

    case 'search':
      return <SearchCommand query={args.join(' ') || flags.q || ''} page={flags.page} />;

    case undefined:
      return <TicketsCommand />;

    default:
      return <Text color="red">Unknown command: {cmd}. Run `itsm --help`.</Text>;
  }
};

export default function App({ cli }: { cli: { input: string[]; flags: CliFlags } }) {
  const { input, flags } = cli;
  if (flags.help || flags.version || input.length === 0 || input[0] === 'help') {
    return <HelpScreen />;
  }
  const [cmd] = input;
  if (cmd === 'login') return <Router input={input} flags={flags} />;
  if (!isLoggedIn() && cmd !== 'logout') {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="red">Not logged in</Text>
        <Text>Run: itsm login -u &lt;username&gt; -p &lt;password&gt;</Text>
      </Box>
    );
  }
  return <Router input={input} flags={flags} />;
}
