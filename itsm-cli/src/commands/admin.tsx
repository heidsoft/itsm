import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Spinner } from '../components/spinner.js';

export const DashboardOverviewCommand: React.FC = () => {
  const [data, setData] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getDashboardOverview().then(d => setData(d)).catch(e => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!data) return <Spinner text="Loading dashboard..." />;
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">🏠 Dashboard Overview</Text>
      <Newline />
      <Text>{JSON.stringify(data, null, 2)}</Text>
    </Box>
  );
};
