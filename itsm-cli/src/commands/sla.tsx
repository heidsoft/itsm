import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import type { Ticket } from '../lib/types.js';

export const SlaStatsCommand: React.FC = () => {
  const [data, setData] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getSLAStats().then(d => setData(d)).catch(e => setErr(e.message));
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
  const [items, setItems] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getSLAOverdue()
      .then(d => { setItems(d || []); setLoading(false); })
      .catch(e => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading overdue tickets..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="red">⚠ Overdue tickets ({items.length})</Text>
      <Newline />
      <Table
        data={items as unknown as Record<string, unknown>[]}
        columns={[
          { key: 'ticket_number', header: 'Number', width: 12 },
          { key: 'title', header: 'Title', width: 40 },
          { key: 'priority', header: 'Prio', width: 8 },
          { key: 'sla_status', header: 'SLA', width: 12 },
          { key: 'assignee_name', header: 'Assignee', width: 16 },
        ]}
      />
    </Box>
  );
};
