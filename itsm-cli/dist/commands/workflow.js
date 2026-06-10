import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
export const WorkflowInstancesCommand = ({ page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listProcessInstances({ page, pageSize: 20 })
            .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading process instances..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\u2699 Process instances (",
            total,
            ")"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'id', header: 'Instance ID', width: 36 },
                { key: 'process_key', header: 'Process', width: 24 },
                { key: 'state', header: 'State', width: 10 },
                { key: 'start_time', header: 'Started', width: 20 },
            ] })));
};
export const WorkflowTasksCommand = () => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listUserTasks()
            .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading tasks..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83D\uDCDD My tasks (",
            total,
            ")"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'id', header: 'Task ID', width: 32 },
                { key: 'name', header: 'Name', width: 30 },
                { key: 'priority', header: 'Prio', width: 8 },
                { key: 'created_at', header: 'Created', width: 20 },
                { key: 'due_date', header: 'Due', width: 20 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true }, "Use `itsm workflow complete <task-id>` to complete; `itsm workflow approve/reject` for approvals.")));
};
export const WorkflowCompleteCommand = ({ id, outcome, comment }) => {
    const [result, setResult] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.completeTask(id, outcome, comment)
            .then(r => { setResult(r); })
            .catch(e => { setErr(e.message); });
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!result)
        return React.createElement(Spinner, { text: "Completing task..." });
    return React.createElement(Text, { color: "green" },
        "\u2713 Task ",
        id,
        " completed: ",
        JSON.stringify(result));
};
export const ApprovalListCommand = ({ status }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listApprovals(status)
            .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading approvals..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\u270B Pending approvals (",
            total,
            ")"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'id', header: 'ID', width: 30 },
                { key: 'name', header: 'Subject', width: 32 },
                { key: 'priority', header: 'Prio', width: 8 },
                { key: 'created_at', header: 'Created', width: 20 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true }, "`itsm workflow approve <id>` / `itsm workflow reject <id>`")));
};
export const ApprovalActionCommand = ({ id, action, comment }) => {
    const [err, setErr] = useState('');
    const [done, setDone] = useState(false);
    useEffect(() => {
        const p = action === 'approve' ? apiClient.approveTask(id, comment) : apiClient.rejectTask(id, comment);
        p.then(() => setDone(true)).catch(e => setErr(e.message));
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!done)
        return React.createElement(Spinner, { text: `${action} ${id}...` });
    return React.createElement(Text, { color: "green" },
        "\u2713 ",
        action,
        " ok: ",
        id);
};
