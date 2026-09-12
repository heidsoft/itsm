'use client';

import React from 'react';
import { App } from 'antd';
import { useParams } from 'next/navigation';
import TicketDetail from '@/components/ticket/TicketDetail';

export default function TicketDetailPage() {
  const params = useParams();
  const id = params?.ticketId as string;

  return (
    <App>
      <div style={{ padding: 24 }}>
        <TicketDetail id={id} />
      </div>
    </App>
  );
}
