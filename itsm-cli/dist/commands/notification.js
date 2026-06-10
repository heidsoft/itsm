import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Spinner } from '../components/spinner.js';
export const NotificationListCommand = ({ unread, page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [unreadCount, setUnreadCount] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listNotifications({ unread, page, pageSize: 20 })
            .then(r => { setItems(r.items || []); setTotal(r.total); setUnreadCount(r.unread); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading notifications..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83D\uDD14 Notifications (",
            total,
            " total, ",
            unreadCount,
            " unread)"),
        React.createElement(Newline, null),
        items.length === 0 && React.createElement(Text, { dimColor: true }, "No notifications."),
        items.map(n => (React.createElement(Box, { key: n.id, flexDirection: "column", marginBottom: 1 },
            React.createElement(Box, null,
                React.createElement(Text, { color: n.read ? 'gray' : 'yellow' },
                    n.read ? '○' : '●',
                    " ",
                    n.level?.toUpperCase() || 'INFO',
                    " \u00B7 ",
                    n.created_at)),
            React.createElement(Text, { bold: true, color: n.read ? 'gray' : 'white' }, n.title),
            n.content && React.createElement(Text, { dimColor: true }, n.content))))));
};
export const NotificationReadCommand = ({ id }) => {
    const [err, setErr] = useState('');
    const [done, setDone] = useState(false);
    useEffect(() => {
        apiClient.markNotificationRead(id).then(() => setDone(true)).catch(e => setErr(e.message));
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!done)
        return React.createElement(Spinner, { text: "Marking read..." });
    return React.createElement(Text, { color: "green" },
        "\u2713 Notification ",
        id,
        " marked read.");
};
