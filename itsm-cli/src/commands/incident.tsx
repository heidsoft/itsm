import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import type { Incident } from '../lib/types.js';

interface ListProps { status?: string; priority?: string; page?: number }
interface ViewProps { id: number }

export const IncidentListCommand: React.FC<ListProps> = ({ status, priority, page = 1 }) => {
  const [items, setItems] = useState<Incident[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listIncidents({ status, priority, page, pageSize: 20 })
      .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
      .catch(e => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading incidents..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">Incidents ({total} total)</Text>
      <Text dimColor>filters: {status ? `status=${status}` : 'all-status'} {priority ? `priority=${priority}` : ''}</Text>
      <Newline />
      <Table
        data={items as unknown as Record<string, unknown>[]}
        columns={[
          { key: 'number', header: 'Number', width: 12 },
          { key: 'title', header: 'Title', width: 35 },
          { key: 'status', header: 'Status', width: 12 },
          { key: 'priority', header: 'Prio', width: 8 },
          { key: 'assigneeName', header: 'Assignee', width: 16 },
          { key: 'slaStatus', header: 'SLA', width: 10 },
        ]}
      />
      <Newline />
      <Text dimColor>Page {page}. Use --page N for more, or `itsm incident &lt;id&gt;` for details.</Text>
    </Box>
  );
};

export const IncidentViewCommand: React.FC<ViewProps> = ({ id }) => {
  const [data, setData] = useState<Incident | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getIncident(id)
      .then(d => setData(d))
      .catch(e => setErr(e.message));
  }, [id]);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!data) return <Spinner text={`Loading incident ${id}...`} />;
  const rows: [string, string | number | undefined][] = [
    ['Number', data.incidentNumber],
    ['Title', data.title],
    ['Status', data.status],
    ['Priority', data.priority],
    ['Category', data.category],
    ['Severity', data.severity],
    ['Assignee', data.assigneeName],
    ['Reporter', data.reporterName],
    ['CMDB CI', data.relatedCiName],
    ['SLA', data.slaStatus],
    ['Due', data.dueDate],
    ['Created', data.createdAt],
    ['Updated', data.updatedAt],
    ['Resolved', data.resolvedAt],
  ];
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">📋 Incident {data.incidentNumber}</Text>
      <Newline />
      {rows.filter(([, v]) => v !== undefined && v !== '').map(([k, v]) => (
        <Box key={k}>
          <Text color="gray">{k.padEnd(12)} </Text>
          <Text>{String(v)}</Text>
        </Box>
      ))}
      {data.description && (
        <Box flexDirection="column" marginTop={1}>
          <Text color="gray">Description</Text>
          <Text>{data.description}</Text>
        </Box>
      )}
    </Box>
  );
};

export const IncidentTriageCommand: React.FC<{ title: string; description: string }> = ({ title, description }) => {
  const [result, setResult] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.aiTriage({ title, description })
      .then((r) => setResult(r as Record<string, unknown>))
      .catch((e: Error) => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!result) return <Spinner text="AI is triaging..." />;
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">🤖 AI Triage Result</Text>
      <Newline />
      <Text>{JSON.stringify(result, null, 2)}</Text>
    </Box>
  );
};
