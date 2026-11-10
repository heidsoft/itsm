import React, { useEffect, useState } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import { DetailPanel } from '../components/detail.js';
import { asList, asRecord, pickString, pickValue, type JsonRecord } from '../lib/records.js';

interface ListProps {
  status?: string;
  type?: string;
  page?: number;
}

interface ViewProps {
  id: number;
}

function toRow(item: JsonRecord): JsonRecord {
  return {
    releaseNumber: pickString(item, 'releaseNumber', 'number') ?? `#${pickString(item, 'id') ?? '?'}`,
    title: pickString(item, 'title') ?? '',
    type: pickString(item, 'type') ?? '',
    environment: pickString(item, 'environment') ?? '',
    status: pickString(item, 'status') ?? '',
    plannedStartDate: pickString(item, 'plannedStartDate') ?? '—',
  };
}

export const ReleaseListCommand: React.FC<ListProps> = ({ status, type, page = 1 }) => {
  const [items, setItems] = useState<JsonRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    apiClient
      .listReleases({ status, type, page, pageSize: 20 })
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
  }, [status, type, page]);

  if (loading) return <Spinner text="Loading releases..." />;
  if (err) return <Text color="red">✗ {err}</Text>;

  return (
    <Box flexDirection="column">
      <Text bold color="cyan">Releases ({total} total)</Text>
      <Text dimColor>
        filters: {status ? `status=${status}` : 'all-status'} {type ? `type=${type}` : ''}
      </Text>
      <Newline />
      <Table
        data={items}
        columns={[
          { key: 'releaseNumber', header: 'Number', width: 14 },
          { key: 'title', header: 'Title', width: 32 },
          { key: 'type', header: 'Type', width: 10 },
          { key: 'environment', header: 'Env', width: 12 },
          { key: 'status', header: 'Status', width: 12 },
          { key: 'plannedStartDate', header: 'Planned', width: 20 },
        ]}
      />
      <Newline />
      <Text dimColor>Page {page}. Use --page N for more, or `itsm release &lt;id&gt;` for details.</Text>
    </Box>
  );
};

export const ReleaseViewCommand: React.FC<ViewProps> = ({ id }) => {
  const [data, setData] = useState<JsonRecord | null>(null);
  const [err, setErr] = useState('');

  useEffect(() => {
    let cancelled = false;
    apiClient
      .getRelease(id)
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
  if (!data) return <Spinner text={`Loading release ${id}...`} />;

  const number = pickString(data, 'releaseNumber', 'number') ?? `#${id}`;
  return (
    <DetailPanel
      title={`🚀 Release ${number}`}
      rows={[
        ['Number', pickValue(data, 'releaseNumber', 'number')],
        ['Title', pickValue(data, 'title')],
        ['Type', pickValue(data, 'type')],
        ['Status', pickValue(data, 'status')],
        ['Severity', pickValue(data, 'severity')],
        ['Environment', pickValue(data, 'environment')],
        ['Owner', pickValue(data, 'ownerName')],
        ['Emergency', pickValue(data, 'isEmergency')],
        ['Planned start', pickValue(data, 'plannedStartDate')],
        ['Planned end', pickValue(data, 'plannedEndDate')],
        ['Actual', pickValue(data, 'actualReleaseDate')],
        ['Created', pickValue(data, 'createdAt')],
      ]}
      sections={[
        { label: 'Description', value: pickValue(data, 'description') },
        { label: 'Release notes', value: pickValue(data, 'releaseNotes') },
        { label: 'Rollback procedure', value: pickValue(data, 'rollbackProcedure') },
        { label: 'Validation criteria', value: pickValue(data, 'validationCriteria') },
        { label: 'Affected systems', value: pickValue(data, 'affectedSystems') },
        { label: 'Deployment steps', value: pickValue(data, 'deploymentSteps') },
      ]}
    />
  );
};
