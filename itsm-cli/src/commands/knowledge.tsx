import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import type { KnowledgeArticle } from '../lib/types.js';

export const KnowledgeListCommand: React.FC<{ category?: string; page?: number }> = ({ category, page = 1 }) => {
  const [items, setItems] = useState<KnowledgeArticle[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listKnowledge({ category, page, pageSize: 20 })
      .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
      .catch(e => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading knowledge base..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">📚 Knowledge Base ({total} total)</Text>
      <Newline />
      <Table
        data={items as unknown as Record<string, unknown>[]}
        columns={[
          { key: 'id', header: 'ID', width: 6 },
          { key: 'title', header: 'Title', width: 40 },
          { key: 'category', header: 'Category', width: 14 },
          { key: 'author_name', header: 'Author', width: 16 },
          { key: 'views', header: 'Views', width: 6 },
          { key: 'status', header: 'Status', width: 10 },
        ]}
      />
      <Newline />
      <Text dimColor>Use `itsm knowledge &lt;id&gt;` to read; `itsm knowledge search &lt;q&gt;` to search.</Text>
    </Box>
  );
};

export const KnowledgeViewCommand: React.FC<{ id: number }> = ({ id }) => {
  const [data, setData] = useState<KnowledgeArticle | null>(null);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.getKnowledgeArticle(id).then(d => setData(d)).catch(e => setErr(e.message));
  }, [id]);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!data) return <Spinner text="Loading article..." />;
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">📖 {data.title}</Text>
      <Text dimColor>by {data.author_name} · {data.created_at}</Text>
      <Newline />
      {data.summary && <Text>{data.summary}</Text>}
      <Newline />
      <Text>{data.content ?? '(no content)'}</Text>
    </Box>
  );
};

export const KnowledgeSearchCommand: React.FC<{ q: string }> = ({ q }) => {
  const [items, setItems] = useState<KnowledgeArticle[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.searchKnowledge(q)
      .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
      .catch(e => { setErr(e.message); setLoading(false); });
  }, [q]);
  if (loading) return <Spinner text={`Searching "${q}"...`} />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">🔍 Knowledge search "{q}" ({total} hits)</Text>
      <Newline />
      <Table
        data={items as unknown as Record<string, unknown>[]}
        columns={[
          { key: 'id', header: 'ID', width: 6 },
          { key: 'title', header: 'Title', width: 45 },
          { key: 'category', header: 'Category', width: 14 },
          { key: 'views', header: 'Views', width: 6 },
        ]}
      />
    </Box>
  );
};
