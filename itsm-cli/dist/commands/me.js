import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { loadCredentials } from '../lib/credentials.js';
export const MeCommand = () => {
    const [user, setUser] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    useEffect(() => {
        const cred = loadCredentials();
        if (cred?.user)
            setUser(cred.user);
        apiClient.getUserInfo()
            .then((u) => { setUser(u); setLoading(false); })
            .catch((err) => {
            if (!user)
                setError(err.message);
            setLoading(false);
        });
    }, []);
    if (loading)
        return React.createElement(Text, { color: "cyan" }, "Loading...");
    if (error && !user)
        return React.createElement(Text, { color: "red" }, error);
    if (!user)
        return React.createElement(Text, { color: "yellow" }, "Not logged in. Run: itsm login -u <username> -p <password>");
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" }, "Current User"),
        React.createElement(Newline, null),
        React.createElement(Box, null,
            React.createElement(Box, { width: 12 },
                React.createElement(Text, { color: "cyan" }, "ID:")),
            React.createElement(Text, null, user.id)),
        React.createElement(Box, null,
            React.createElement(Box, { width: 12 },
                React.createElement(Text, { color: "cyan" }, "Username:")),
            React.createElement(Text, null, user.username)),
        React.createElement(Box, null,
            React.createElement(Box, { width: 12 },
                React.createElement(Text, { color: "cyan" }, "Name:")),
            React.createElement(Text, null, user.name)),
        React.createElement(Box, null,
            React.createElement(Box, { width: 12 },
                React.createElement(Text, { color: "cyan" }, "Email:")),
            React.createElement(Text, null, user.email)),
        React.createElement(Box, null,
            React.createElement(Box, { width: 12 },
                React.createElement(Text, { color: "cyan" }, "Role:")),
            React.createElement(Text, null, user.role))));
};
