import React from 'react';
import { Box, Text } from 'ink';
const STATUS_COLORS = {
    open: 'red',
    in_progress: 'yellow',
    pending: 'blue',
    resolved: 'green',
    closed: 'gray',
};
const PRIORITY_COLORS = {
    low: 'gray',
    medium: 'yellow',
    high: 'red',
    critical: 'magenta',
};
export function StatusBadge({ status }) {
    const color = STATUS_COLORS[status.toLowerCase()] ?? 'white';
    return React.createElement(Text, { color: color }, status.toUpperCase().padEnd(12));
}
export function PriorityBadge({ priority }) {
    const color = PRIORITY_COLORS[priority.toLowerCase()] ?? 'white';
    return React.createElement(Text, { color: color }, priority.toUpperCase().padEnd(8));
}
export function TicketRow({ ticket }) {
    return (React.createElement(Box, null,
        React.createElement(Box, { width: 15 },
            React.createElement(Text, { color: "cyan" }, ticket.ticket_number)),
        React.createElement(Box, { width: 35 },
            React.createElement(Text, null, ticket.title.substring(0, 33))),
        React.createElement(Box, { width: 14 },
            React.createElement(StatusBadge, { status: ticket.status })),
        React.createElement(Box, { width: 10 },
            React.createElement(PriorityBadge, { priority: ticket.priority })),
        React.createElement(Box, { width: 15 },
            React.createElement(Text, { dimColor: true }, ticket.assignee_name ?? '—'))));
}
