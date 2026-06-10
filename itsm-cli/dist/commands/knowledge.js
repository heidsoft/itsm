import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
export const KnowledgeListCommand = ({ category, page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listKnowledge({ category, page, pageSize: 20 })
            .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading knowledge base..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83D\uDCDA Knowledge Base (",
            total,
            " total)"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'id', header: 'ID', width: 6 },
                { key: 'title', header: 'Title', width: 40 },
                { key: 'category', header: 'Category', width: 14 },
                { key: 'author_name', header: 'Author', width: 16 },
                { key: 'views', header: 'Views', width: 6 },
                { key: 'status', header: 'Status', width: 10 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true }, "Use `itsm knowledge <id>` to read; `itsm knowledge search <q>` to search.")));
};
export const KnowledgeViewCommand = ({ id }) => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.getKnowledgeArticle(id).then(d => setData(d)).catch(e => setErr(e.message));
    }, [id]);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!data)
        return React.createElement(Spinner, { text: "Loading article..." });
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83D\uDCD6 ",
            data.title),
        React.createElement(Text, { dimColor: true },
            "by ",
            data.author_name,
            " \u00B7 ",
            data.created_at),
        React.createElement(Newline, null),
        data.summary && React.createElement(Text, null, data.summary),
        React.createElement(Newline, null),
        React.createElement(Text, null, data.content ?? '(no content)')));
};
export const KnowledgeSearchCommand = ({ q }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.searchKnowledge(q)
            .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, [q]);
    if (loading)
        return React.createElement(Spinner, { text: `Searching "${q}"...` });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83D\uDD0D Knowledge search \"",
            q,
            "\" (",
            total,
            " hits)"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'id', header: 'ID', width: 6 },
                { key: 'title', header: 'Title', width: 45 },
                { key: 'category', header: 'Category', width: 14 },
                { key: 'views', header: 'Views', width: 6 },
            ] })));
};
