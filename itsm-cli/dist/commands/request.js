import React, { useEffect, useState } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import { DetailPanel } from '../components/detail.js';
import { asList, asRecord, pickNested, pickNumber, pickString, pickValue, } from '../lib/records.js';
function toRow(item) {
    const current = pickNumber(item, 'currentLevel');
    const total = pickNumber(item, 'totalLevels');
    return {
        requestNumber: pickString(item, 'requestNumber', 'number') ?? `#${pickString(item, 'id') ?? '?'}`,
        title: pickString(item, 'title', 'reason') ?? '—',
        status: pickString(item, 'status') ?? '',
        catalog: pickNested(item, 'catalog', 'name') ?? pickString(item, 'catalogId') ?? '—',
        level: current !== undefined && total !== undefined ? `${current}/${total}` : '—',
        createdAt: pickString(item, 'createdAt') ?? '—',
    };
}
export const RequestListCommand = ({ page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        let cancelled = false;
        setLoading(true);
        apiClient
            .listServiceRequests({ page, pageSize: 20 })
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
    }, [page]);
    if (loading)
        return React.createElement(Spinner, { text: "Loading service requests..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "Service requests (",
            total,
            " total)"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'requestNumber', header: 'Number', width: 14 },
                { key: 'title', header: 'Title', width: 32 },
                { key: 'status', header: 'Status', width: 16 },
                { key: 'catalog', header: 'Catalog', width: 18 },
                { key: 'level', header: 'Level', width: 8 },
                { key: 'createdAt', header: 'Created', width: 20 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true },
            "Page ",
            page,
            ". Use --page N for more, or `itsm request <id>` for details.")));
};
export const RequestViewCommand = ({ id }) => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        let cancelled = false;
        apiClient
            .getServiceRequest(id)
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
        return React.createElement(Spinner, { text: `Loading service request ${id}...` });
    const number = pickString(data, 'requestNumber', 'number') ?? `#${id}`;
    const current = pickNumber(data, 'currentLevel');
    const total = pickNumber(data, 'totalLevels');
    return (React.createElement(DetailPanel, { title: `📦 Service request ${number}`, rows: [
            ['Number', pickValue(data, 'requestNumber', 'number')],
            ['Ticket', pickValue(data, 'ticketNumber')],
            ['Title', pickValue(data, 'title')],
            ['Status', pickValue(data, 'status')],
            ['Catalog', pickNested(data, 'catalog', 'name') ?? pickValue(data, 'catalogId')],
            ['Requester', pickNested(data, 'requester', 'name') ?? pickValue(data, 'requesterId')],
            ['Approval level', current !== undefined && total !== undefined ? `${current}/${total}` : undefined],
            ['Can approve', pickValue(data, 'canApprove')],
            ['Created', pickValue(data, 'createdAt')],
            ['Updated', pickValue(data, 'updatedAt')],
        ], sections: [
            { label: 'Reason', value: pickValue(data, 'reason') },
            { label: 'Form data', value: pickValue(data, 'formData', 'formFields') },
            { label: 'Approvals', value: pickValue(data, 'approvals') },
            { label: 'Completion note', value: pickValue(data, 'completionNote') },
            { label: 'Last error', value: pickValue(data, 'lastError') },
        ] }));
};
