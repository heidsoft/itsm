import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import type { CI } from '../lib/types.js';

interface ListProps { type?: string; status?: string; page?: number }
interface ViewProps { id: number }

export const CmdbListCommand: React.FC<ListProps> = ({ type, status, page = 1 }) => {
  const [items, setItems] = useState<CI[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listCIs({ type, status, page, pageSize: 20 })
      .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
      .catch(e => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading CIs..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">CMDB Configuration Items ({total} total)</Text>
      <Text dimColor>filters: {type ? `type=${type}` : 'all-types'} {status ? `status=${status}` : ''}</Text>
      <Newline />
      <Table
        data={items as unknown as Record<string, unknown>[]}
        columns={[
          { key: 'id', header: 'ID', width: 6 },
          { key: 'name', header: 'Name', width: 30 },
          { key: 'type', header: 'Type', width: 14 },
          { key: 'status', header: 'Status', width: 10 },
          { key: 'owner', header: 'Owner', width: 16 },
          { key: 'location', header: 'Location', width: 16 },
        ]}
      />
      <Newline />
      <Text dimColor>Use `itsm cmdb &lt;id&gt;` for details.</Text>
    </Box>
  );
};

export const CmdbViewCommand: React.FC<ViewProps> = ({ id }) => {
  const [data, setData] = useState<CI | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getCI(id).then(d => setData(d)).catch(e => setErr(e.message));
  }, [id]);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!data) return <Spinner text={`Loading CI ${id}...`} />;
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">🗄️  CI #{data.id} {data.name}</Text>
      <Newline />
      <Text>Type:     {data.type}</Text>
      <Text>Status:   {data.status}</Text>
      <Text>Owner:    {data.owner ?? '-'}</Text>
      <Text>Location: {data.location ?? '-'}</Text>
      <Text>Created:  {data.created_at}</Text>
      <Text>Updated:  {data.updated_at}</Text>
    </Box>
  );
};
