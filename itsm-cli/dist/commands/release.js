import React, { useEffect, useState } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import { DetailPanel } from '../components/detail.js';
import { asList, asRecord, pickString, pickValue } from '../lib/records.js';
function toRow(item) {
    return {
        releaseNumber: pickString(item, 'releaseNumber', 'number') ?? `#${pickString(item, 'id') ?? '?'}`,
        title: pickString(item, 'title') ?? '',
        type: pickString(item, 'type') ?? '',
        environment: pickString(item, 'environment') ?? '',
        status: pickString(item, 'status') ?? '',
        plannedStartDate: pickString(item, 'plannedStartDate') ?? '—',
    };
}
export const ReleaseListCommand = ({ status, type, page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        let cancelled = false;
        setLoading(true);
        apiClient
            .listReleases({ status, type, page, pageSize: 20 })
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
    }, [status, type, page]);
    if (loading)
        return React.createElement(Spinner, { text: "Loading releases..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "Releases (",
            total,
            " total)"),
        React.createElement(Text, { dimColor: true },
            "filters: ",
            status ? `status=${status}` : 'all-status',
            " ",
            type ? `type=${type}` : ''),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'releaseNumber', header: 'Number', width: 14 },
                { key: 'title', header: 'Title', width: 32 },
                { key: 'type', header: 'Type', width: 10 },
                { key: 'environment', header: 'Env', width: 12 },
                { key: 'status', header: 'Status', width: 12 },
                { key: 'plannedStartDate', header: 'Planned', width: 20 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true },
            "Page ",
            page,
            ". Use --page N for more, or `itsm release <id>` for details.")));
};
export const ReleaseViewCommand = ({ id }) => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        let cancelled = false;
        apiClient
            .getRelease(id)
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
        return React.createElement(Spinner, { text: `Loading release ${id}...` });
    const number = pickString(data, 'releaseNumber', 'number') ?? `#${id}`;
    return (React.createElement(DetailPanel, { title: `🚀 Release ${number}`, rows: [
            ['Number', pickValue(data, 'releaseNumber', 'number')],
            ['Title', pickValue(data, 'title')],
            ['Type', pickValue(data, 'type')],
            ['Status', pickValue(data, 'status')],
            ['Severity', pickValue(data, 'severity')],
            ['Environment', pickValue(data, 'environment')],
            ['Owner', pickValue(data, 'ownerName')],
            ['Emergency', pickValue(data, 'isEmergency')],
            ['Planned start', pickValue(data, 'plannedStartDate')],
            ['Planned end', pickValue(data, 'plannedEndDate')],
            ['Actual', pickValue(data, 'actualReleaseDate')],
            ['Created', pickValue(data, 'createdAt')],
        ], sections: [
            { label: 'Description', value: pickValue(data, 'description') },
            { label: 'Release notes', value: pickValue(data, 'releaseNotes') },
            { label: 'Rollback procedure', value: pickValue(data, 'rollbackProcedure') },
            { label: 'Validation criteria', value: pickValue(data, 'validationCriteria') },
            { label: 'Affected systems', value: pickValue(data, 'affectedSystems') },
            { label: 'Deployment steps', value: pickValue(data, 'deploymentSteps') },
        ] }));
};
