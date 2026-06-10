import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
export const CmdbListCommand = ({ type, status, page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listCIs({ type, status, page, pageSize: 20 })
            .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading CIs..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "CMDB Configuration Items (",
            total,
            " total)"),
        React.createElement(Text, { dimColor: true },
            "filters: ",
            type ? `type=${type}` : 'all-types',
            " ",
            status ? `status=${status}` : ''),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'id', header: 'ID', width: 6 },
                { key: 'name', header: 'Name', width: 30 },
                { key: 'type', header: 'Type', width: 14 },
                { key: 'status', header: 'Status', width: 10 },
                { key: 'owner', header: 'Owner', width: 16 },
                { key: 'location', header: 'Location', width: 16 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true }, "Use `itsm cmdb <id>` for details.")));
};
export const CmdbViewCommand = ({ id }) => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.getCI(id).then(d => setData(d)).catch(e => setErr(e.message));
    }, [id]);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!data)
        return React.createElement(Spinner, { text: `Loading CI ${id}...` });
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83D\uDDC4\uFE0F  CI #",
            data.id,
            " ",
            data.name),
        React.createElement(Newline, null),
        React.createElement(Text, null,
            "Type:     ",
            data.type),
        React.createElement(Text, null,
            "Status:   ",
            data.status),
        React.createElement(Text, null,
            "Owner:    ",
            data.owner ?? '-'),
        React.createElement(Text, null,
            "Location: ",
            data.location ?? '-'),
        React.createElement(Text, null,
            "Created:  ",
            data.created_at),
        React.createElement(Text, null,
            "Updated:  ",
            data.updated_at)));
};
