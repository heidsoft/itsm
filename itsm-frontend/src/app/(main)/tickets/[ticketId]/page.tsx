'use client';

import React from 'react';
import { App, Button } from 'antd';
import { ArrowLeft } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import TicketDetail from '@/components/ticket/TicketDetail';
import { useI18n } from '@/lib/i18n/useI18n';

export default function TicketDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.ticketId as string;
  const { t } = useI18n();

  return (
    <App>
      <div style={{ padding: 24 }}>
        <div style={{ marginBottom: 16 }}>
          <Button
            type="link"
            icon={<ArrowLeft />}
            onClick={() => router.back()}
            style={{ paddingLeft: 0, color: '#666' }}
          >
            {t('common.back')}
          </Button>
        </div>
        <TicketDetail id={id} />
      </div>
    </App>
  );
}
