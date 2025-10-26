import React from 'react';
import AppLayout from '@/components/layout/AppLayout';

export default function TicketsLayout({ children }: { children: React.ReactNode }) {
  return (
    <AppLayout title='工单管理' description='管理和跟踪IT工单' showPageHeader={true}>
      {children}
    </AppLayout>
  );
}
