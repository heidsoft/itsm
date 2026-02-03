'use client';

import React from 'react';
import ProblemList from '@/modules/problem/components/ProblemList';

export default function ProblemListPage() {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900 m-0">问题管理</h2>
      </div>
      <ProblemList />
    </div>
  );
}
