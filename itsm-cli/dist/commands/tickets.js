import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
export const TicketsCommand = ({ status, priority, page = 1 }) => {
    const [tickets, setTickets] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    useEffect(() => {
        apiClient.listTickets({ status, priority, page, pageSize: 20 })
            .then(res => {
            setTickets(res.items);
            setTotal(res.total);
            setLoading(false);
        })
            .catch((err) => {
            setError(err.message);
            setLoading(false);
        });
    }, []);
    if (loading)
        return React.createElement(Text, { color: "cyan" }, "Loading tickets...");
    if (error)
        return React.createElement(Text, { color: "red" }, error);
    const columns = [
        { key: 'ticket_number', header: 'ID', width: 14 },
        { key: 'title', header: 'Title', width: 35 },
        { key: 'status', header: 'Status', width: 14 },
        { key: 'priority', header: 'Prio', width: 10 },
        { key: 'assignee_name', header: 'Assignee', width: 15 },
    ];
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "Tickets (",
            total,
            " total)"),
        React.createElement(Newline, null),
        React.createElement(Table, { columns: columns, data: tickets }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true },
            "Page ",
            page,
            " \u00B7 Use --page <n> for more")));
};
