import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { LoginCommand } from './commands/login.js';
import { TicketsCommand } from './commands/tickets.js';
import { TicketViewCommand } from './commands/ticket.js';
import { SearchCommand } from './commands/search.js';
import { MeCommand } from './commands/me.js';
import { isLoggedIn, clearCredentials } from './lib/credentials.js';
import { apiClient } from './lib/api-client.js';

interface CliFlags {
  status?: string;
  priority?: string;
  page?: number;
  username?: string;
  password?: string;
  help: boolean;
  version: boolean;
}

interface AppProps {
  cli: {
    input: string[];
    flags: CliFlags;
  };
}

const HelpScreen: React.FC = () => (
  <Box flexDirection="column" padding={1}>
    <Text bold color="cyan">ITSM CLI v1.0.0</Text>
    <Text>React for CLIs · built with Ink</Text>
    <Newline />
    <Text bold>Usage:</Text>
    <Text dimColor>  itsm login -u &lt;user&gt; -p &lt;pass&gt;   Login</Text>
    <Text dimColor>  itsm tickets [--status] [--priority]  List tickets</Text>
    <Text dimColor>  itsm ticket &lt;id&gt;                    View ticket</Text>
    <Text dimColor>  itsm search &lt;query&gt;                Search</Text>
    <Text dimColor>  itsm me                           Who am I</Text>
    <Text dimColor>  itsm logout                        Logout</Text>
    <Newline />
    <Text dimColor>Backend: http://localhost:8090</Text>
  </Box>
);

const Router: React.FC<{ input: string[]; flags: CliFlags }> = ({ input, flags }) => {
  const [cmd, ...args] = input;

  switch (cmd) {
    case 'login':
      return <LoginCommand username={flags.username ?? ''} password={flags.password ?? ''} />;

    case 'ticket': {
      if (args[0] === 'list') return <TicketsCommand status={flags.status} priority={flags.priority} page={flags.page} />;
      const id = parseInt(args[0] ?? '0', 10);
      if (id > 0) return <TicketViewCommand id={id} />;
      return <Text color="red">Usage: itsm ticket &lt;id&gt;</Text>;
    }

    case 'tickets':
      return <TicketsCommand status={flags.status} priority={flags.priority} page={flags.page} />;

    case 'search':
      return <SearchCommand query={args.join(' ')} page={flags.page} />;

    case 'me':
      return <MeCommand />;

    case 'logout': {
      clearCredentials();
      return <Text color="yellow">✓ Logged out</Text>;
    }

    case undefined:
      return <TicketsCommand />;

    default:
      return <Text color="red">Unknown command: {cmd}</Text>;
  }
};

export default function App({ cli }: AppProps) {
  const { input, flags } = cli;

  if (flags.help || flags.version || input.length === 0 || (input[0] === 'help')) {
    return <HelpScreen />;
  }

  const [cmd] = input;

  if (cmd === 'login') {
    return <Router input={input} flags={flags} />;
  }

  if (!isLoggedIn() && cmd !== 'logout') {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="red">Not logged in</Text>
        <Text>Run: itsm login -u &lt;username&gt; -p &lt;password&gt;</Text>
      </Box>
    );
  }

  if (cmd === 'sla') {
    return <SlaStatusCommand />;
  }

  return <Router input={input} flags={flags} />;
}

// Inline SLA status command
const SlaStatusCommand: React.FC = () => {
  const [stats, setStats] = useState<unknown>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    apiClient.getSLAStats()
      .then((s: unknown) => { setStats(s); setLoading(false); })
      .catch((err: Error) => { setError(err.message); setLoading(false); });
  }, []);

  if (loading) return <Text color="cyan">Loading SLA stats...</Text>;
  if (error) return <Text color="red">{error}</Text>;
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">SLA Status</Text>
      <Newline />
      <Text>{JSON.stringify(stats, null, 2)}</Text>
    </Box>
  );
};
