'use client';

import React from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { useI18n } from '@/lib/i18n';
import './dashboard.css';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();

  return (
    <AppLayout
      title={t('dashboard.title')}
      description={t('dashboard.description')}
      showPageHeader={true}
    >
      {children}
    </AppLayout>
  );
}
