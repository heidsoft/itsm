'use client';

import React from 'react';
import { Card, Row, Col, Input, Select, Button } from 'antd';
import { RefreshCw } from 'lucide-react';
import { ProblemStatus } from '@/constants/problem';

const { Search: SearchInput } = Input;

interface ProblemFiltersProps {
  loading: boolean;
  onSearch: (value: string) => void;
  onFilterChange: (value: string) => void;
  onRefresh: () => void;
}

export const ProblemFilters: React.FC<ProblemFiltersProps> = ({
  loading,
  onSearch,
  onFilterChange,
  onRefresh,
}) => {
  return (
    <Card
      className="mb-6 bg-white shadow-sm border border-gray-200 rounded-lg"
      style={{ marginBottom: 24 }}
    >
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-gray-900 mb-1">Filter</h3>
        <p className="text-sm text-gray-500">Use filters to find issues quickly</p>
      </div>
      <Row gutter={20} align="middle">
        <Col xs={24} sm={12} md={8}>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Search</label>
            <SearchInput
              placeholder="Search problem title, ID or description..."
              allowClear
              onSearch={onSearch}
              size="large"
              enterButton
              className="rounded-md"
              style={{
                borderRadius: '6px',
              }}
            />
          </div>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Status</label>
            <Select
              placeholder="Status"
              size="large"
              allowClear
              onChange={onFilterChange}
              style={{ width: '100%' }}
              className="rounded-md"
              options={[
                { value: ProblemStatus.OPEN, label: 'Pending' },
                { value: ProblemStatus.INVESTIGATING, label: 'In Progress' },
                { value: ProblemStatus.RESOLVED, label: 'Resolved' },
                { value: ProblemStatus.CLOSED, label: 'Closed' },
              ]}
            />
          </div>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Actions</label>
            <Button
              icon={<RefreshCw size={20} />}
              onClick={onRefresh}
              loading={loading}
              size="large"
              style={{ width: '100%' }}
              className="flex items-center justify-center gap-2 bg-blue-50 border-blue-200 text-blue-700 hover:bg-blue-100 hover:border-blue-300 rounded-md transition-colors duration-200"
            >

              Refresh
            </Button>
          </div>
        </Col>
      </Row>
    </Card>
  );
};
