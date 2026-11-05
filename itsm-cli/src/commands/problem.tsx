import React, { useEffect, useState } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import { DetailPanel } from '../components/detail.js';
import { asList, asRecord, pickString, pickValue, type JsonRecord } from '../lib/records.js';

interface ListProps {
  status?: string;
  priority?: string;
  page?: number;
}

interface ViewProps {
  id: number;
}

/** Problem list columns, normalised so a missing field shows a placeholder instead of shifting cells. */
function toRow(item: JsonRecord): JsonRecord {
  return {
    problemNumber: pickString(item, 'problemNumber', 'number') ?? `#${pickString(item, 'id') ?? '?'}`,
    title: pickString(item, 'title') ?? '',
    status: pickString(item, 'status') ?? '',
    priority: pickString(item, 'priority') ?? '',
    assigneeName: pickString(item, 'assigneeName') ?? '—',
  };
}

export const ProblemListCommand: React.FC<ListProps> = ({ status, priority, page = 1 }) => {
  const [items, setItems] = useState<JsonRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    apiClient
      .listProblems({ status, priority, page, pageSize: 20 })
      .then((data) => {
        if (cancelled) return;
        const list = asList(data);
        setItems(list.items.map(toRow));
        setTotal(list.total);
        setLoading(false);
      })
      .catch((error: Error) => {
        if (cancelled) return;
        setErr(error.message);
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [status, priority, page]);

  if (loading) return <Spinner text="Loading problems..." />;
  if (err) return <Text color="red">✗ {err}</Text>;

  return (
    <Box flexDirection="column">
      <Text bold color="cyan">Problems ({total} total)</Text>
      <Text dimColor>
        filters: {status ? `status=${status}` : 'all-status'} {priority ? `priority=${priority}` : ''}
      </Text>
      <Newline />
      <Table
        data={items}
        columns={[
          { key: 'problemNumber', header: 'Number', width: 14 },
          { key: 'title', header: 'Title', width: 35 },
          { key: 'status', header: 'Status', width: 12 },
          { key: 'priority', header: 'Prio', width: 8 },
          { key: 'assigneeName', header: 'Assignee', width: 16 },
        ]}
      />
      <Newline />
      <Text dimColor>Page {page}. Use --page N for more, or `itsm problem &lt;id&gt;` for details.</Text>
    </Box>
  );
};

export const ProblemViewCommand: React.FC<ViewProps> = ({ id }) => {
  const [data, setData] = useState<JsonRecord | null>(null);
  const [err, setErr] = useState('');

  useEffect(() => {
    let cancelled = false;
    apiClient
      .getProblem(id)
      .then((payload) => {
        if (!cancelled) setData(asRecord(payload));
      })
      .catch((error: Error) => {
        if (!cancelled) setErr(error.message);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (err) return <Text color="red">✗ {err}</Text>;
  if (!data) return <Spinner text={`Loading problem ${id}...`} />;

  const number = pickString(data, 'problemNumber', 'number') ?? `#${id}`;
  return (
    <DetailPanel
      title={`🧩 Problem ${number}`}
      rows={[
        ['Number', pickValue(data, 'problemNumber', 'number')],
        ['Title', pickValue(data, 'title')],
        ['Status', pickValue(data, 'status')],
        ['Priority', pickValue(data, 'priority')],
        ['Category', pickValue(data, 'category')],
        ['Impact', pickValue(data, 'impact')],
        ['Assignee', pickValue(data, 'assigneeName')],
        ['Reporter', pickValue(data, 'createdByName')],
        ['Created', pickValue(data, 'createdAt')],
        ['Updated', pickValue(data, 'updatedAt')],
      ]}
      sections={[
        { label: 'Description', value: pickValue(data, 'description') },
        { label: 'Root cause', value: pickValue(data, 'rootCause') },
        { label: 'Workaround', value: pickValue(data, 'workaround') },
        { label: 'Resolution', value: pickValue(data, 'resolution') },
        { label: 'Associated tickets', value: pickValue(data, 'associatedTickets') },
        { label: 'Associated incidents', value: pickValue(data, 'associatedIncidents') },
      ]}
    />
  );
};
