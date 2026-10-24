import React, { useEffect, useState } from 'react';
import { Box, Text, useInput } from 'ink';
import * as fs from 'fs';
import { saveCredentials } from '../lib/credentials.js';
import { apiClient } from '../lib/api-client.js';
import { redactSecrets } from '../lib/secrets.js';

interface LoginProps {
  username: string;
  /** Password captured from argv. Never used: argv leaks through `ps` and shell history. */
  password?: string;
}

/**
 * Interactive login.
 *
 * The password is NEVER taken from argv. It is read either from a masked
 * prompt (TTY) or from stdin (`printf %s "$PASS" | itsm login -u admin`), so it
 * never appears in the process table or shell history. Headless callers should
 * use `itsm auth login --password-stdin`, which additionally stores a refresh
 * token and emits a JSON envelope.
 */
export const LoginCommand: React.FC<LoginProps> = ({ username, password }) => {
  const [secret, setSecret] = useState('');
  const [done, setDone] = useState<'ok' | 'error' | null>(null);
  const [message, setMessage] = useState('');
  const [piped, setPiped] = useState(false);

  const argvRefused = typeof password === 'string' && password.length > 0;

  useEffect(() => {
    if (argvRefused) {
      setMessage(
        'Refusing a password on the command line: argv is visible in shell history and the process table. ' +
          'Run `itsm login -u <username>` for a masked prompt, pipe it (`printf %s "$PASS" | itsm login -u <username>`), ' +
          'or use `itsm auth login --password-stdin` for agent mode.'
      );
      setDone('error');
      return;
    }
    if (!username) {
      setMessage('Usage: itsm login -u <username>   (the password is prompted, never passed as a flag)');
      setDone('error');
      return;
    }
    // Non-interactive stdin: `printf %s "$PASS" | itsm login -u admin`.
    if (!process.stdin.isTTY) {
      try {
        const value = fs.readFileSync(0, 'utf-8').replace(/\r?\n$/, '');
        if (!value) {
          setMessage('No password received on stdin.');
          setDone('error');
          return;
        }
        setPiped(true);
        setSecret(value);
      } catch (err) {
        setMessage(`Failed to read the password from stdin: ${err instanceof Error ? err.message : String(err)}`);
        setDone('error');
      }
    }
    // Mount-only probe of argv/stdin.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Piped mode has no Enter key, so submit as soon as the value lands.
  useEffect(() => {
    if (!piped || !secret) return;
    void submit(secret);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [piped, secret]);

  useInput(
    (input, key) => {
      if (done !== null) return;
      if (key.return) {
        if (!secret) {
          setMessage('Password cannot be empty.');
          setDone('error');
          return;
        }
        void submit(secret);
        return;
      }
      if (key.backspace || key.delete) {
        setSecret((current) => current.slice(0, -1));
        return;
      }
      if (key.ctrl || key.escape || key.tab || key.upArrow || key.downArrow || key.leftArrow || key.rightArrow) return;
      if (input) setSecret((current) => current + input);
    },
    { isActive: !argvRefused && Boolean(username) && Boolean(process.stdin.isTTY) && done === null }
  );

  async function submit(value: string): Promise<void> {
    try {
      const result = await apiClient.login({ username, password: value });
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
    } catch (err) {
      setMessage(redactSecrets(err instanceof Error ? err.message : 'Login failed'));
      setDone('error');
    }
  }

  if (done === 'ok') {
    return (
      <Box flexDirection="column">
        <Text color="green">✓ {message}</Text>
        <Text dimColor>Token saved to ~/.itsm/credentials (mode 0600)</Text>
      </Box>
    );
  }

  if (done === 'error') {
    return <Text color="red">✗ {message}</Text>;
  }

  if (!process.stdin.isTTY) {
    return <Text color="cyan">Logging in...</Text>;
  }

  return (
    <Box flexDirection="column">
      <Text color="cyan">Logging in as {username}</Text>
      <Box>
        <Text>Password: </Text>
        <Text>{'*'.repeat(secret.length)}</Text>
      </Box>
      <Text dimColor>Input is masked. Press Enter to submit.</Text>
    </Box>
  );
};
