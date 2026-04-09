import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { loadCredentials } from '../lib/credentials.js';
import type { User } from '../lib/types.js';

export const MeCommand: React.FC = () => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const cred = loadCredentials();
    if (cred?.user) setUser(cred.user);

    apiClient.getUserInfo()
      .then((u: User) => { setUser(u); setLoading(false); })
      .catch((err: Error) => {
        if (!user) setError(err.message);
        setLoading(false);
      });
  }, []);

  if (loading) return <Text color="cyan">Loading...</Text>;
  if (error && !user) return <Text color="red">{error}</Text>;
  if (!user) return <Text color="yellow">Not logged in. Run: itsm login -u &lt;username&gt; -p &lt;password&gt;</Text>;

  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">Current User</Text>
      <Newline />
      <Box><Box width={12}><Text color="cyan">ID:</Text></Box><Text>{user.id}</Text></Box>
      <Box><Box width={12}><Text color="cyan">Username:</Text></Box><Text>{user.username}</Text></Box>
      <Box><Box width={12}><Text color="cyan">Name:</Text></Box><Text>{user.name}</Text></Box>
      <Box><Box width={12}><Text color="cyan">Email:</Text></Box><Text>{user.email}</Text></Box>
      <Box><Box width={12}><Text color="cyan">Role:</Text></Box><Text>{user.role}</Text></Box>
    </Box>
  );
};
