import React, { useEffect, useState } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import { DetailPanel } from '../components/detail.js';
import {
  asList,
  asRecord,
  pickNested,
  pickNumber,
  pickString,
  pickValue,
  type JsonRecord,
} from '../lib/records.js';

interface ListProps {
  page?: number;
}

interface ViewProps {
  id: number;
}

function toRow(item: JsonRecord): JsonRecord {
  const current = pickNumber(item, 'currentLevel');
  const total = pickNumber(item, 'totalLevels');
  return {
    requestNumber: pickString(item, 'requestNumber', 'number') ?? `#${pickString(item, 'id') ?? '?'}`,
    title: pickString(item, 'title', 'reason') ?? '—',
    status: pickString(item, 'status') ?? '',
    catalog: pickNested(item, 'catalog', 'name') ?? pickString(item, 'catalogId') ?? '—',
    level: current !== undefined && total !== undefined ? `${current}/${total}` : '—',
    createdAt: pickString(item, 'createdAt') ?? '—',
  };
}

export const RequestListCommand: React.FC<ListProps> = ({ page = 1 }) => {
  const [items, setItems] = useState<JsonRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    apiClient
      .listServiceRequests({ page, pageSize: 20 })
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
  }, [page]);

  if (loading) return <Spinner text="Loading service requests..." />;
  if (err) return <Text color="red">✗ {err}</Text>;

  return (
    <Box flexDirection="column">
      <Text bold color="cyan">Service requests ({total} total)</Text>
      <Newline />
      <Table
        data={items}
        columns={[
          { key: 'requestNumber', header: 'Number', width: 14 },
          { key: 'title', header: 'Title', width: 32 },
          { key: 'status', header: 'Status', width: 16 },
          { key: 'catalog', header: 'Catalog', width: 18 },
          { key: 'level', header: 'Level', width: 8 },
          { key: 'createdAt', header: 'Created', width: 20 },
        ]}
      />
      <Newline />
      <Text dimColor>Page {page}. Use --page N for more, or `itsm request &lt;id&gt;` for details.</Text>
    </Box>
  );
};

export const RequestViewCommand: React.FC<ViewProps> = ({ id }) => {
  const [data, setData] = useState<JsonRecord | null>(null);
  const [err, setErr] = useState('');

  useEffect(() => {
    let cancelled = false;
    apiClient
      .getServiceRequest(id)
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
  if (!data) return <Spinner text={`Loading service request ${id}...`} />;

  const number = pickString(data, 'requestNumber', 'number') ?? `#${id}`;
  const current = pickNumber(data, 'currentLevel');
  const total = pickNumber(data, 'totalLevels');
  return (
    <DetailPanel
      title={`📦 Service request ${number}`}
      rows={[
        ['Number', pickValue(data, 'requestNumber', 'number')],
        ['Ticket', pickValue(data, 'ticketNumber')],
        ['Title', pickValue(data, 'title')],
        ['Status', pickValue(data, 'status')],
        ['Catalog', pickNested(data, 'catalog', 'name') ?? pickValue(data, 'catalogId')],
        ['Requester', pickNested(data, 'requester', 'name') ?? pickValue(data, 'requesterId')],
        ['Approval level', current !== undefined && total !== undefined ? `${current}/${total}` : undefined],
        ['Can approve', pickValue(data, 'canApprove')],
        ['Created', pickValue(data, 'createdAt')],
        ['Updated', pickValue(data, 'updatedAt')],
      ]}
      sections={[
        { label: 'Reason', value: pickValue(data, 'reason') },
        { label: 'Form data', value: pickValue(data, 'formData', 'formFields') },
        { label: 'Approvals', value: pickValue(data, 'approvals') },
        { label: 'Completion note', value: pickValue(data, 'completionNote') },
        { label: 'Last error', value: pickValue(data, 'lastError') },
      ]}
    />
  );
};
