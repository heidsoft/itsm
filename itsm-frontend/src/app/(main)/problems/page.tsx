'use client';

import React from 'react';
import ProblemList from '@/modules/problem/components/ProblemList';

export default function ProblemListPage() {
  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>问题管理</h2>
      </div>
      <ProblemList />
    </div>
  );
}
