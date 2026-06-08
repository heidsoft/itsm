import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import type { Change } from '../lib/types.js';

interface ListProps { status?: string; page?: number }
interface ViewProps { id: number }

export const ChangeListCommand: React.FC<ListProps> = ({ status, page = 1 }) => {
  const [items, setItems] = useState<Change[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listChanges({ status, page, pageSize: 20 })
      .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
      .catch(e => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading changes..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">Changes ({total} total)</Text>
      {status && <Text dimColor>status filter: {status}</Text>}
      <Newline />
      <Table
        data={items as unknown as Record<string, unknown>[]}
        columns={[
          { key: 'number', header: 'Number', width: 12 },
          { key: 'title', header: 'Title', width: 35 },
          { key: 'type', header: 'Type', width: 12 },
          { key: 'risk_level', header: 'Risk', width: 8 },
          { key: 'status', header: 'Status', width: 12 },
          { key: 'scheduled_at', header: 'Scheduled', width: 20 },
        ]}
      />
      <Newline />
      <Text dimColor>Use `itsm change &lt;id&gt;` for details.</Text>
    </Box>
  );
};

export const ChangeViewCommand: React.FC<ViewProps> = ({ id }) => {
  const [data, setData] = useState<Change | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getChange(id).then(d => setData(d)).catch(e => setErr(e.message));
  }, [id]);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!data) return <Spinner text={`Loading change ${id}...`} />;
  const rows: [string, string | undefined][] = [
    ['Number', data.number], ['Title', data.title], ['Type', data.type],
    ['Risk', data.risk_level], ['Status', data.status], ['Priority', data.priority],
    ['Assignee', data.assignee_name], ['Implementer', data.implementer_name],
    ['Scheduled', data.scheduled_at], ['Created', data.created_at], ['Updated', data.updated_at],
  ];
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">🔧 Change {data.number}</Text>
      <Newline />
      {rows.filter(([, v]) => v).map(([k, v]) => (
        <Box key={k}><Text color="gray">{k.padEnd(12)} </Text><Text>{v}</Text></Box>
      ))}
      {data.description && <Box flexDirection="column" marginTop={1}><Text color="gray">Description</Text><Text>{data.description}</Text></Box>}
      {data.rollback_plan && <Box flexDirection="column" marginTop={1}><Text color="gray">Rollback</Text><Text>{data.rollback_plan}</Text></Box>}
    </Box>
  );
};
