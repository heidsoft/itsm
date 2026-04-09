import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import type { Ticket } from '../lib/types.js';

interface TicketViewProps {
  id: number;
}

export const TicketViewCommand: React.FC<TicketViewProps> = ({ id }) => {
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    apiClient.getTicket(id)
      .then((t: Ticket) => { setTicket(t); setLoading(false); })
      .catch((err: Error) => { setError(err.message); setLoading(false); });
  }, [id]);

  if (loading) return <Text color="cyan">Loading ticket #{id}...</Text>;
  if (error) return <Text color="red">{error}</Text>;
  if (!ticket) return <Text>Ticket not found</Text>;

  const fmt = (k: string, v: unknown) => (
    <Box key={k}>
      <Box width={14}><Text color="cyan">{k}:</Text></Box>
      <Text>{String(v ?? '—')}</Text>
    </Box>
  );

  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">Ticket #{ticket.ticket_number}</Text>
      <Newline />
      {fmt('Title', ticket.title)}
      {fmt('Status', ticket.status)}
      {fmt('Priority', ticket.priority)}
      {fmt('Category', ticket.category)}
      {fmt('Assignee', ticket.assignee_name)}
      {fmt('Requester', ticket.requester_name)}
      {fmt('Created', ticket.created_at)}
      {fmt('Updated', ticket.updated_at)}
      {ticket.sla_status && fmt('SLA', ticket.sla_status)}
      <Newline />
      <Text bold>Description:</Text>
      <Text>{ticket.description}</Text>
    </Box>
  );
};
