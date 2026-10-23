import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';

export const WorkflowInstancesCommand: React.FC<{ page?: number }> = ({ page = 1 }) => {
  const [items, setItems] = useState<Record<string, unknown>[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listProcessInstances({ page, pageSize: 20 })
      .then((r) => {
        const payload = r as unknown as { items?: Record<string, unknown>[]; total?: number };
        setItems(payload.items || []);
        setTotal(payload.total ?? 0);
        setLoading(false);
      })
      .catch((e: Error) => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading process instances..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">⚙ Process instances ({total})</Text>
      <Newline />
      <Table
        data={items}
        columns={[
          { key: 'id', header: 'Instance ID', width: 36 },
          { key: 'processDefinitionKey', header: 'Process', width: 24 },
          { key: 'state', header: 'State', width: 10 },
          { key: 'startTime', header: 'Started', width: 20 },
        ]}
      />
    </Box>
  );
};

export const WorkflowTasksCommand: React.FC = () => {
  const [items, setItems] = useState<Record<string, unknown>[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listUserTasks()
      .then((r) => {
        const payload = (r ?? {}) as { items?: Record<string, unknown>[]; total?: number };
        const list = Array.isArray(payload.items) ? payload.items : (Array.isArray(r) ? (r as Record<string, unknown>[]) : []);
        setItems(list);
        setTotal(payload.total ?? list.length);
        setLoading(false);
      })
      .catch((e: Error) => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading tasks..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">📝 My tasks ({total})</Text>
      <Newline />
      <Table
        data={items}
        columns={[
          { key: 'id', header: 'Task ID', width: 32 },
          { key: 'name', header: 'Name', width: 30 },
          { key: 'priority', header: 'Prio', width: 8 },
          { key: 'createdAt', header: 'Created', width: 20 },
          { key: 'dueDate', header: 'Due', width: 20 },
        ]}
      />
      <Newline />
      <Text dimColor>Use `itsm --json workflow complete &lt;task-id&gt;` or `itsm --json workflow decision &lt;task-id&gt; --action approve`.</Text>
    </Box>
  );
};

export const WorkflowCompleteCommand: React.FC<{ id: string; outcome: string; comment?: string }> = ({ id, outcome, comment }) => {
  const [result, setResult] = useState<unknown>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.completeTask(id, { outcome, ...(comment ? { comment } : {}) })
      .then((r: unknown) => setResult(r))
      .catch((e: Error) => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!result) return <Spinner text="Completing task..." />;
  return <Text color="green">✓ Task {id} completed: {JSON.stringify(result)}</Text>;
};

export const ApprovalListCommand: React.FC<{ status?: string }> = ({ status }) => {
  const [items, setItems] = useState<Record<string, unknown>[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listApprovalRecords(status ? { status } : undefined)
      .then((r) => {
        const payload = (r ?? {}) as { items?: Record<string, unknown>[]; total?: number };
        const list = Array.isArray(payload.items) ? payload.items : (Array.isArray(r) ? (r as Record<string, unknown>[]) : []);
        setItems(list);
        setTotal(payload.total ?? list.length);
        setLoading(false);
      })
      .catch((e: Error) => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading approvals..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">✋ Approval records ({total})</Text>
      <Newline />
      <Table
        data={items}
        columns={[
          { key: 'id', header: 'ID', width: 30 },
          { key: 'subject', header: 'Subject', width: 32 },
          { key: 'status', header: 'Status', width: 12 },
          { key: 'createdAt', header: 'Created', width: 20 },
        ]}
      />
      <Newline />
      <Text dimColor>Approve via `itsm --json ticket workflow-approve &lt;ticketId&gt; --approvalId &lt;n&gt;`.</Text>
    </Box>
  );
};

export const ApprovalActionCommand: React.FC<{ id: string; action: 'approve' | 'reject'; comment?: string }> = ({ id, action, comment }) => {
  const [err, setErr] = useState('');
  const [done, setDone] = useState(false);
  useEffect(() => {
    apiClient.decideTask(id, comment ? { action, comment } : { action })
      .then(() => setDone(true))
      .catch((e: Error) => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!done) return <Spinner text={`${action} ${id}...`} />;
  return <Text color="green">✓ {action} ok: {id}</Text>;
};
