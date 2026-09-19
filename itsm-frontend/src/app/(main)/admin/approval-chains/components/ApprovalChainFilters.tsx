'use client';

/**
 * 审批链筛选器组件
 */

import React, { useState, useCallback } from 'react';
import { Card, Row, Col, Input, Select, Button, Space } from 'antd';
import { RefreshCw, Filter } from 'lucide-react';
import type { ApprovalChainFilters as ApprovalChainFiltersType } from '@/types/approval-chain';
import { useDebouncedCallback } from '@/lib/component-utils';

const { Search } = Input;

interface ApprovalChainFiltersProps {
  filters: ApprovalChainFiltersType;
  onFilterChange: (filters: ApprovalChainFiltersType) => void;
  onRefresh: () => void;
  loading?: boolean;
}

export function ApprovalChainFilters({
  filters,
  onFilterChange,
  onRefresh,
  loading = false,
}: ApprovalChainFiltersProps) {
  const [localFilters, setLocalFilters] = useState<ApprovalChainFiltersType>(filters);

  const debouncedSearch = useDebouncedCallback((...args: unknown[]) => {
    const keyword = args[0] as string;
    onFilterChange({ ...localFilters, name: keyword });
  }, 300);

  const handleSearch = useCallback(
    (value: string) => {
      setLocalFilters((prev: ApprovalChainFiltersType) => ({ ...prev, name: value }));
      debouncedSearch(value);
    },
    [debouncedSearch]
  );

  const handleStatusChange = useCallback(
    (status: 'active' | 'inactive') => {
      const newFilters = { ...localFilters, status: status || undefined };
      setLocalFilters(newFilters);
      onFilterChange(newFilters);
    },
    [localFilters, onFilterChange]
  );

  const handleReset = useCallback(() => {
    const resetFilters: ApprovalChainFiltersType = {};
    setLocalFilters(resetFilters);
    onFilterChange(resetFilters);
  }, [onFilterChange]);

  return (
    <Card className="mb-6">
      <Row gutter={[16, 16]} align="middle">
        <Col xs={24} sm={12} md={8} lg={6}>
          <div className="mb-2">
            <span className="text-sm font-medium text-gray-700">搜索名称</span>
          </div>
          <Search
            placeholder="搜索审批链名称..."
            value={localFilters.name || ''}
            onChange={e => handleSearch(e.target.value)}
            onSearch={handleSearch}
            allowClear
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <div className="mb-2">
            <span className="text-sm font-medium text-gray-700">状态</span>
          </div>
          <Select
            placeholder="选择状态"
            value={localFilters.status}
            onChange={handleStatusChange}
            style={{ width: '100%' }}
            allowClear
            options={[
              { value: 'active', label: '活跃' },
              { value: 'inactive', label: '非活跃' },
            ]}
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <div className="mb-2">
            <span className="text-sm font-medium text-gray-700">操作</span>
          </div>
          <Space>
            <Button icon={<RefreshCw className="w-4 h-4" />} onClick={onRefresh} loading={loading}>
              刷新
            </Button>
            <Button icon={<Filter className="w-4 h-4" />} onClick={handleReset}>
              重置
            </Button>
          </Space>
        </Col>
      </Row>
    </Card>
  );
}
