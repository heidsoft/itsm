import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Spinner } from '../components/spinner.js';
import type { Notification } from '../lib/types.js';

export const NotificationListCommand: React.FC<{ unread?: boolean; page?: number }> = ({ unread, page = 1 }) => {
  const [items, setItems] = useState<Notification[]>([]);
  const [total, setTotal] = useState(0);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  useEffect(() => {
    apiClient.listNotifications({ unread, page, pageSize: 20 })
      .then(r => { setItems(r.items || []); setTotal(r.total); setUnreadCount(r.unread); setLoading(false); })
      .catch(e => { setErr(e.message); setLoading(false); });
  }, []);
  if (loading) return <Spinner text="Loading notifications..." />;
  if (err) return <Text color="red">✗ {err}</Text>;
  return (
    <Box flexDirection="column">
      <Text bold color="cyan">🔔 Notifications ({total} total, {unreadCount} unread)</Text>
      <Newline />
      {items.length === 0 && <Text dimColor>No notifications.</Text>}
      {items.map(n => (
        <Box key={n.id} flexDirection="column" marginBottom={1}>
          <Box>
            <Text color={n.read ? 'gray' : 'yellow'}>
              {n.read ? '○' : '●'} {n.level?.toUpperCase() || 'INFO'} · {n.created_at}
            </Text>
          </Box>
          <Text bold color={n.read ? 'gray' : 'white'}>{n.title}</Text>
          {n.content && <Text dimColor>{n.content}</Text>}
        </Box>
      ))}
    </Box>
  );
};

export const NotificationReadCommand: React.FC<{ id: number }> = ({ id }) => {
  const [err, setErr] = useState('');
  const [done, setDone] = useState(false);
  useEffect(() => {
    apiClient.markNotificationRead(id).then(() => setDone(true)).catch(e => setErr(e.message));
  }, []);
  if (err) return <Text color="red">✗ {err}</Text>;
  if (!done) return <Spinner text="Marking read..." />;
  return <Text color="green">✓ Notification {id} marked read.</Text>;
};
