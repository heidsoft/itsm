import React, { useState, useEffect } from 'react';
import { Box, Text, Spacer } from 'ink';
import { saveCredentials } from '../lib/credentials.js';
import { apiClient } from '../lib/api-client.js';

interface LoginProps {
  username: string;
  password: string;
}

export const LoginCommand: React.FC<LoginProps> = ({ username, password }) => {
  const [done, setDone] = useState<'ok' | 'error' | null>(null);
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!username || !password) {
      setMessage('Usage: itsm login -u <username> -p <password>');
      setDone('error');
      return;
    }
    apiClient.login({ username, password })
      .then(result => {
        saveCredentials({
          token: result.token,
          refreshToken: result.refresh_token,
          user: result.user,
          tenantId: result.tenant_id,
          tenantName: result.tenant_name,
        });
        setMessage(`Logged in as ${result.user.name} (${result.tenant_name})`);
        setDone('ok');
      })
      .catch(err => {
        setMessage(err instanceof Error ? err.message : 'Login failed');
        setDone('error');
      });
  }, []);

  if (done === null) {
    return <Text color="cyan">Logging in...</Text>;
  }

  return (
    <Box flexDirection="column">
      <Text color={done === 'ok' ? 'green' : 'red'}>{done === 'ok' ? '✓' : '✗'} {message}</Text>
      {done === 'ok' && <Text dimColor>Token saved to ~/.itsm/credentials</Text>}
    </Box>
  );
};
