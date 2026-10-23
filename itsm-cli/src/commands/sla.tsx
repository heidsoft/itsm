import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';

export const SlaStatsCommand: React.FC = () => {
  const [data, setData] = useState<unknown>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getSLAStats().then((d: unknown) => setData(d)).catch((e: Error) => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!data) return <Spinner text="Loading SLA stats..." />;
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">📊 SLA Stats</Text>
      <Newline />
      <Text>{JSON.stringify(data, null, 2)}</Text>
    </Box>
  );
};

export const SlaOverdueCommand: React.FC = () => {
  const [items, setItems] = useState<Record<string, unknown>[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getSLAViolations()
      .then((d) => {
        const raw = (d as { items?: unknown[] }).items ?? (Array.isArray(d) ? d : []);
        setItems(raw as Record<string, unknown>[]);
        setLoading(false);
      })
      .catch((e: Error) => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading SLA violations..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="red">⚠ SLA violations ({items.length})</Text>
      <Newline />
      <Table
        data={items}
        columns={[
          { key: 'ticketNumber', header: 'Number', width: 12 },
          { key: 'ticketTitle', header: 'Title', width: 40 },
          { key: 'violationType', header: 'Type', width: 14 },
          { key: 'breachedAt', header: 'Breached', width: 20 },
        ]}
      />
      <Newline />
      <Text dimColor>Raw payload: {JSON.stringify(items).slice(0, 400)}</Text>
    </Box>
  );
};
