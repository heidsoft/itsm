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
        <h3 className="text-lg font-semibold text-gray-900 mb-1">筛选器</h3>
        <p className="text-sm text-gray-500">使用筛选条件快速查找问题</p>
      </div>
      <Row gutter={20} align="middle">
        <Col xs={24} sm={12} md={8}>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">搜索</label>
            <SearchInput
              placeholder="搜索问题标题、ID或描述..."
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
            <label className="block text-sm font-medium text-gray-700">状态</label>
            <Select
              placeholder="状态筛选"
              size="large"
              allowClear
              onChange={onFilterChange}
              style={{ width: '100%' }}
              className="rounded-md"
              options={[
                { value: ProblemStatus.OPEN, label: '待处理' },
                { value: ProblemStatus.INVESTIGATING, label: '处理中' },
                { value: ProblemStatus.RESOLVED, label: '已解决' },
                { value: ProblemStatus.CLOSED, label: '已关闭' },
              ]}
            />
          </div>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">操作</label>
            <Button
              icon={<RefreshCw size={20} />}
              onClick={onRefresh}
              loading={loading}
              size="large"
              style={{ width: '100%' }}
              className="flex items-center justify-center gap-2 bg-blue-50 border-blue-200 text-blue-700 hover:bg-blue-100 hover:border-blue-300 rounded-md transition-colors duration-200"
            >
              刷新
            </Button>
          </div>
        </Col>
      </Row>
    </Card>
  );
};
