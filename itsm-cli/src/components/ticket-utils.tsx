import React from 'react';
import { Box, Text } from 'ink';

const STATUS_COLORS: Record<string, string> = {
  open: 'red',
  in_progress: 'yellow',
  pending: 'blue',
  resolved: 'green',
  closed: 'gray',
};

const PRIORITY_COLORS: Record<string, string> = {
  low: 'gray',
  medium: 'yellow',
  high: 'red',
  critical: 'magenta',
};

export function StatusBadge({ status }: { status: string }) {
  const color = STATUS_COLORS[status.toLowerCase()] ?? 'white';
  return <Text color={color}>{status.toUpperCase().padEnd(12)}</Text>;
}

export function PriorityBadge({ priority }: { priority: string }) {
  const color = PRIORITY_COLORS[priority.toLowerCase()] ?? 'white';
  return <Text color={color}>{priority.toUpperCase().padEnd(8)}</Text>;
}

export function TicketRow({ ticket }: { ticket: { ticket_number: string; title: string; status: string; priority: string; assignee_name?: string } }) {
  return (
    <Box>
      <Box width={15}>
        <Text color="cyan">{ticket.ticket_number}</Text>
      </Box>
      <Box width={35}>
        <Text>{ticket.title.substring(0, 33)}</Text>
      </Box>
      <Box width={14}>
        <StatusBadge status={ticket.status} />
      </Box>
      <Box width={10}>
        <PriorityBadge priority={ticket.priority} />
      </Box>
      <Box width={15}>
        <Text dimColor>{ticket.assignee_name ?? '—'}</Text>
      </Box>
    </Box>
  );
}
