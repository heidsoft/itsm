import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
export const ConnectorListCommand = () => {
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listConnectors()
            .then(r => { setItems(r.items || []); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading connector market..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83E\uDDE9 Connector / Plugin / Skill Market (",
            items.length,
            ")"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'name', header: 'Name', width: 18 },
                { key: 'title', header: 'Title', width: 24 },
                { key: 'provider', header: 'Provider', width: 12 },
                { key: 'type', header: 'Type', width: 14 },
                { key: 'local', header: 'Local', width: 6 },
                { key: 'installed', header: 'Installed', width: 10 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true }, "Use `itsm connector test <name>` to send a test message; `itsm connector health` for status.")));
};
export const ConnectorTestCommand = ({ name }) => {
    const [result, setResult] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.testConnector(name).then(r => setResult(r)).catch(e => setErr(e.message));
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!result)
        return React.createElement(Spinner, { text: `Testing connector ${name}...` });
    return React.createElement(Text, { color: "green" },
        "\u2713 ",
        name,
        ": ",
        JSON.stringify(result));
};
export const ConnectorHealthCommand = () => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.connectorHealth().then(d => setData(d)).catch(e => setErr(e.message));
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!data)
        return React.createElement(Spinner, { text: "Checking health..." });
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" }, "\u2764\uFE0F  Connector health"),
        React.createElement(Newline, null),
        React.createElement(Text, null, JSON.stringify(data, null, 2))));
};
export const ConnectorSendCommand = ({ name, channel, message }) => {
    const [result, setResult] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.sendViaConnector(name, { channel, type: 'text', content: message })
            .then(r => setResult(r))
            .catch(e => setErr(e.message));
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!result)
        return React.createElement(Spinner, { text: "Sending..." });
    return React.createElement(Text, { color: "green" },
        "\u2713 Sent via ",
        name,
        " to ",
        channel);
};
