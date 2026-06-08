import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import type { ConnectorManifest } from '../lib/types.js';

export const ConnectorListCommand: React.FC = () => {
  const [items, setItems] = useState<ConnectorManifest[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listConnectors()
      .then(r => { setItems(r.items || []); setLoading(false); })
      .catch(e => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading connector market..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">🧩 Connector / Plugin / Skill Market ({items.length})</Text>
      <Newline />
      <Table
        data={items as unknown as Record<string, unknown>[]}
        columns={[
          { key: 'name', header: 'Name', width: 18 },
          { key: 'title', header: 'Title', width: 24 },
          { key: 'provider', header: 'Provider', width: 12 },
          { key: 'type', header: 'Type', width: 14 },
          { key: 'local', header: 'Local', width: 6 },
          { key: 'installed', header: 'Installed', width: 10 },
        ]}
      />
      <Newline />
      <Text dimColor>Use `itsm connector test &lt;name&gt;` to send a test message; `itsm connector health` for status.</Text>
    </Box>
  );
};

export const ConnectorTestCommand: React.FC<{ name: string }> = ({ name }) => {
  const [result, setResult] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.testConnector(name).then(r => setResult(r)).catch(e => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!result) return <Spinner text={`Testing connector ${name}...`} />;
  return <Text color="green">✓ {name}: {JSON.stringify(result)}</Text>;
};

export const ConnectorHealthCommand: React.FC = () => {
  const [data, setData] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.connectorHealth().then(d => setData(d)).catch(e => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!data) return <Spinner text="Checking health..." />;
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">❤️  Connector health</Text>
      <Newline />
      <Text>{JSON.stringify(data, null, 2)}</Text>
    </Box>
  );
};

export const ConnectorSendCommand: React.FC<{ name: string; channel: string; message: string }> = ({ name, channel, message }) => {
  const [result, setResult] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.sendViaConnector(name, { channel, type: 'text', content: message })
      .then(r => setResult(r))
      .catch(e => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!result) return <Spinner text="Sending..." />;
  return <Text color="green">✓ Sent via {name} to {channel}</Text>;
};
