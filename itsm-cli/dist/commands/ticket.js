import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
export const TicketViewCommand = ({ id }) => {
    const [ticket, setTicket] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    useEffect(() => {
        apiClient.getTicket(id)
            .then((t) => { setTicket(t); setLoading(false); })
            .catch((err) => { setError(err.message); setLoading(false); });
    }, [id]);
    if (loading)
        return React.createElement(Text, { color: "cyan" },
            "Loading ticket #",
            id,
            "...");
    if (error)
        return React.createElement(Text, { color: "red" }, error);
    if (!ticket)
        return React.createElement(Text, null, "Ticket not found");
    const fmt = (k, v) => (React.createElement(Box, { key: k },
        React.createElement(Box, { width: 14 },
            React.createElement(Text, { color: "cyan" },
                k,
                ":")),
        React.createElement(Text, null, String(v ?? '—'))));
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" },
            "Ticket #",
            ticket.ticket_number),
        React.createElement(Newline, null),
        fmt('Title', ticket.title),
        fmt('Status', ticket.status),
        fmt('Priority', ticket.priority),
        fmt('Category', ticket.category),
        fmt('Assignee', ticket.assignee_name),
        fmt('Requester', ticket.requester_name),
        fmt('Created', ticket.created_at),
        fmt('Updated', ticket.updated_at),
        ticket.sla_status && fmt('SLA', ticket.sla_status),
        React.createElement(Newline, null),
        React.createElement(Text, { bold: true }, "Description:"),
        React.createElement(Text, null, ticket.description)));
};
