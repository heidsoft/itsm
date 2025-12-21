'use client';

import React from 'react';
import IncidentList from '@/modules/incident/components/IncidentList';

export default function IncidentListPage() {
  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>事件管理</h2>
      </div>
      <IncidentList />
    </div>
  );
}
