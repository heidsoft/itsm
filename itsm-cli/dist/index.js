import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { LoginCommand } from './commands/login.js';
import { TicketsCommand } from './commands/tickets.js';
import { TicketViewCommand } from './commands/ticket.js';
import { SearchCommand } from './commands/search.js';
import { MeCommand } from './commands/me.js';
import { isLoggedIn, clearCredentials } from './lib/credentials.js';
import { apiClient } from './lib/api-client.js';
const HelpScreen = () => (React.createElement(Box, { flexDirection: "column", padding: 1 },
    React.createElement(Text, { bold: true, color: "cyan" }, "ITSM CLI v1.0.0"),
    React.createElement(Text, null, "React for CLIs \u00B7 built with Ink"),
    React.createElement(Newline, null),
    React.createElement(Text, { bold: true }, "Usage:"),
    React.createElement(Text, { dimColor: true }, "  itsm login -u <user> -p <pass>   Login"),
    React.createElement(Text, { dimColor: true }, "  itsm tickets [--status] [--priority]  List tickets"),
    React.createElement(Text, { dimColor: true }, "  itsm ticket <id>                    View ticket"),
    React.createElement(Text, { dimColor: true }, "  itsm search <query>                Search"),
    React.createElement(Text, { dimColor: true }, "  itsm me                           Who am I"),
    React.createElement(Text, { dimColor: true }, "  itsm logout                        Logout"),
    React.createElement(Newline, null),
    React.createElement(Text, { dimColor: true }, "Backend: http://localhost:8090")));
const Router = ({ input, flags }) => {
    const [cmd, ...args] = input;
    switch (cmd) {
        case 'login':
            return React.createElement(LoginCommand, { username: flags.username ?? '', password: flags.password ?? '' });
        case 'ticket': {
            if (args[0] === 'list')
                return React.createElement(TicketsCommand, { status: flags.status, priority: flags.priority, page: flags.page });
            const id = parseInt(args[0] ?? '0', 10);
            if (id > 0)
                return React.createElement(TicketViewCommand, { id: id });
            return React.createElement(Text, { color: "red" }, "Usage: itsm ticket <id>");
        }
        case 'tickets':
            return React.createElement(TicketsCommand, { status: flags.status, priority: flags.priority, page: flags.page });
        case 'search':
            return React.createElement(SearchCommand, { query: args.join(' '), page: flags.page });
        case 'me':
            return React.createElement(MeCommand, null);
        case 'logout': {
            clearCredentials();
            return React.createElement(Text, { color: "yellow" }, "\u2713 Logged out");
        }
        case undefined:
            return React.createElement(TicketsCommand, null);
        default:
            return React.createElement(Text, { color: "red" },
                "Unknown command: ",
                cmd);
    }
};
export default function App({ cli }) {
    const { input, flags } = cli;
    if (flags.help || flags.version || input.length === 0 || (input[0] === 'help')) {
        return React.createElement(HelpScreen, null);
    }
    const [cmd] = input;
    if (cmd === 'login') {
        return React.createElement(Router, { input: input, flags: flags });
    }
    if (!isLoggedIn() && cmd !== 'logout') {
        return (React.createElement(Box, { flexDirection: "column", padding: 1 },
            React.createElement(Text, { color: "red" }, "Not logged in"),
            React.createElement(Text, null, "Run: itsm login -u <username> -p <password>")));
    }
    if (cmd === 'sla') {
        return React.createElement(SlaStatusCommand, null);
    }
    return React.createElement(Router, { input: input, flags: flags });
}
// Inline SLA status command
const SlaStatusCommand = () => {
    const [stats, setStats] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    useEffect(() => {
        apiClient.getSLAStats()
            .then((s) => { setStats(s); setLoading(false); })
            .catch((err) => { setError(err.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Text, { color: "cyan" }, "Loading SLA stats...");
    if (error)
        return React.createElement(Text, { color: "red" }, error);
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" }, "SLA Status"),
        React.createElement(Newline, null),
        React.createElement(Text, null, JSON.stringify(stats, null, 2))));
};
