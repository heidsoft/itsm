import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import type { Ticket } from '../lib/types.js';
import { Table } from '../components/table.js';

interface TicketsProps {
  status?: string;
  priority?: string;
  page?: number;
}

export const TicketsCommand: React.FC<TicketsProps> = ({ status, priority, page = 1 }) => {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    apiClient.listTickets({ status, priority, page, pageSize: 20 })
      .then(res => {
        setTickets(res.items);
        setTotal(res.total);
        setLoading(false);
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  if (loading) return <Text color="cyan">Loading tickets...</Text>;
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
      <Text bold color="cyan">Tickets ({total} total)</Text>
      <Newline />
      <Table columns={columns} data={tickets as unknown as Record<string, unknown>[]} />
      <Newline />
      <Text dimColor>Page {page} · Use --page &lt;n&gt; for more</Text>
    </Box>
  );
};
