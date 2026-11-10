import React, { useEffect, useState } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import { DetailPanel } from '../components/detail.js';
import { asList, asRecord, pickString, pickValue } from '../lib/records.js';
/** Problem list columns, normalised so a missing field shows a placeholder instead of shifting cells. */
function toRow(item) {
    return {
        problemNumber: pickString(item, 'problemNumber', 'number') ?? `#${pickString(item, 'id') ?? '?'}`,
        title: pickString(item, 'title') ?? '',
        status: pickString(item, 'status') ?? '',
        priority: pickString(item, 'priority') ?? '',
        assigneeName: pickString(item, 'assigneeName') ?? '—',
    };
}
export const ProblemListCommand = ({ status, priority, page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        let cancelled = false;
        setLoading(true);
        apiClient
            .listProblems({ status, priority, page, pageSize: 20 })
            .then((data) => {
            if (cancelled)
                return;
            const list = asList(data);
            setItems(list.items.map(toRow));
            setTotal(list.total);
            setLoading(false);
        })
            .catch((error) => {
            if (cancelled)
                return;
            setErr(error.message);
            setLoading(false);
        });
        return () => {
            cancelled = true;
        };
    }, [status, priority, page]);
    if (loading)
        return React.createElement(Spinner, { text: "Loading problems..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "Problems (",
            total,
            " total)"),
        React.createElement(Text, { dimColor: true },
            "filters: ",
            status ? `status=${status}` : 'all-status',
            " ",
            priority ? `priority=${priority}` : ''),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'problemNumber', header: 'Number', width: 14 },
                { key: 'title', header: 'Title', width: 35 },
                { key: 'status', header: 'Status', width: 12 },
                { key: 'priority', header: 'Prio', width: 8 },
                { key: 'assigneeName', header: 'Assignee', width: 16 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true },
            "Page ",
            page,
            ". Use --page N for more, or `itsm problem <id>` for details.")));
};
export const ProblemViewCommand = ({ id }) => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        let cancelled = false;
        apiClient
            .getProblem(id)
            .then((payload) => {
            if (!cancelled)
                setData(asRecord(payload));
        })
            .catch((error) => {
            if (!cancelled)
                setErr(error.message);
        });
        return () => {
            cancelled = true;
        };
    }, [id]);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!data)
        return React.createElement(Spinner, { text: `Loading problem ${id}...` });
    const number = pickString(data, 'problemNumber', 'number') ?? `#${id}`;
    return (React.createElement(DetailPanel, { title: `🧩 Problem ${number}`, rows: [
            ['Number', pickValue(data, 'problemNumber', 'number')],
            ['Title', pickValue(data, 'title')],
            ['Status', pickValue(data, 'status')],
            ['Priority', pickValue(data, 'priority')],
            ['Category', pickValue(data, 'category')],
            ['Impact', pickValue(data, 'impact')],
            ['Assignee', pickValue(data, 'assigneeName')],
            ['Reporter', pickValue(data, 'createdByName')],
            ['Created', pickValue(data, 'createdAt')],
            ['Updated', pickValue(data, 'updatedAt')],
        ], sections: [
            { label: 'Description', value: pickValue(data, 'description') },
            { label: 'Root cause', value: pickValue(data, 'rootCause') },
            { label: 'Workaround', value: pickValue(data, 'workaround') },
            { label: 'Resolution', value: pickValue(data, 'resolution') },
            { label: 'Associated tickets', value: pickValue(data, 'associatedTickets') },
            { label: 'Associated incidents', value: pickValue(data, 'associatedIncidents') },
        ] }));
};
