import React, { useState, useEffect } from 'react';
import { Box, Text } from 'ink';
import { saveCredentials } from '../lib/credentials.js';
import { apiClient } from '../lib/api-client.js';
export const LoginCommand = ({ username, password }) => {
    const [done, setDone] = useState(null);
    const [message, setMessage] = useState('');
    useEffect(() => {
        if (!username || !password) {
            setMessage('Usage: itsm login -u <username> -p <password>');
            setDone('error');
            return;
        }
        apiClient.login({ username, password })
            .then(result => {
            if (!result.accessToken) {
                setMessage('Login succeeded but no access token cookie was returned by the backend');
                setDone('error');
                return;
            }
            saveCredentials({
                token: result.accessToken,
                refreshToken: result.refreshToken,
                user: result.user,
                tenantId: result.user.tenantId,
            });
            setMessage(`Logged in as ${result.user.name} (token expires in ${result.expiresIn}s)`);
            setDone('ok');
        })
            .catch(err => {
            setMessage(err instanceof Error ? err.message : 'Login failed');
            setDone('error');
        });
    }, []);
    if (done === null) {
        return React.createElement(Text, { color: "cyan" }, "Logging in...");
    }
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { color: done === 'ok' ? 'green' : 'red' },
            done === 'ok' ? '✓' : '✗',
            " ",
            message),
        done === 'ok' && React.createElement(Text, { dimColor: true }, "Token saved to ~/.itsm/credentials")));
};
