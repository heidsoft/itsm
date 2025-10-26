import React from 'react';
import AppLayout from '@/components/layout/AppLayout';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <AppLayout
      title='Dashboard'
      description='System overview and key metrics'
      showPageHeader={true}
    >
      {children}
    </AppLayout>
  );
}
