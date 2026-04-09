import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import type { Ticket } from '../lib/types.js';
import { Table } from '../components/table.js';

interface SearchProps {
  query: string;
  page?: number;
}

export const SearchCommand: React.FC<SearchProps> = ({ query, page = 1 }) => {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!query) {
      setError('Usage: itsm search <query>');
      setLoading(false);
      return;
    }
    apiClient.searchTickets(query, { page, pageSize: 20 })
      .then((res: { items: Ticket[]; total: number }) => {
        setTickets(res.items);
        setTotal(res.total);
        setLoading(false);
      })
      .catch((err: Error) => { setError(err.message); setLoading(false); });
  }, [query]);

  if (loading) return <Text color="cyan">Searching "{query}"...</Text>;
  if (error) return <Text color="red">{error}</Text>;

  const columns = [
    { key: 'ticket_number', header: 'ID', width: 14 },
    { key: 'title', header: 'Title', width: 35 },
    { key: 'status', header: 'Status', width: 14 },
    { key: 'priority', header: 'Prio', width: 10 },
    { key: 'assignee_name', header: 'Assignee', width: 15 },
  ];

  return (
    <Box flexDirection="column">
      <Text bold color="cyan">Results for "{query}" ({total} found)</Text>
      <Newline />
      <Table columns={columns} data={tickets as unknown as Record<string, unknown>[]} />
    </Box>
  );
};
