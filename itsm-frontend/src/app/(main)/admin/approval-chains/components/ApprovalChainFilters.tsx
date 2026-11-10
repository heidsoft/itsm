'use client';

/**
 * 审批链筛选器组件
 */

import React, { useState, useCallback } from 'react';
import { Card, Row, Col, Input, Select, Button, Space } from 'antd';
import { RefreshCw, Filter, Plus } from 'lucide-react';
import type { ApprovalChainFilters as ApprovalChainFiltersType } from '@/types/approval-chain';
import { ENTITY_TYPE_LABELS } from '@/types/approval-chain';
import { useDebouncedCallback } from '@/lib/component-utils';

const { Search } = Input;

interface ApprovalChainFiltersProps {
  filters: ApprovalChainFiltersType;
  onFilterChange: (filters: ApprovalChainFiltersType) => void;
  onRefresh: () => void;
  onCreate?: () => void;
  loading?: boolean;
}

export function ApprovalChainFilters({
  filters,
  onFilterChange,
  onRefresh,
  onCreate,
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

  const handleEntityTypeChange = useCallback(
    (entityType?: string) => {
      const newFilters = { ...localFilters, entityType: entityType || undefined };
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
            <span className="text-sm font-medium text-gray-700">Search name</span>
          </div>
          <Search
            placeholder="Search approval chain name..."
            value={localFilters.name || ''}
            onChange={e => handleSearch(e.target.value)}
            onSearch={handleSearch}
            allowClear
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <div className="mb-2">
            <span className="text-sm font-medium text-gray-700">Applicable objects</span>
          </div>
          <Select
            placeholder="Select applicable objects"
            value={localFilters.entityType}
            onChange={handleEntityTypeChange}
            style={{ width: '100%' }}
            allowClear
            options={Object.entries(ENTITY_TYPE_LABELS).map(([value, label]) => ({
              value,
              label,
            }))}
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <div className="mb-2">
            <span className="text-sm font-medium text-gray-700">Status</span>
          </div>
          <Select
            placeholder="Select status"
            value={localFilters.status}
            onChange={handleStatusChange}
            style={{ width: '100%' }}
            allowClear
            options={[
              { value: 'active', label: 'Active' },
              { value: 'inactive', label: 'Inactive' },
            ]}
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <div className="mb-2">
            <span className="text-sm font-medium text-gray-700">Actions</span>
          </div>
          <Space>
            <Button icon={<RefreshCw className="w-4 h-4" />} onClick={onRefresh} loading={loading}>

              Refresh
            </Button>
            <Button icon={<Filter className="w-4 h-4" />} onClick={handleReset}>

              Reset
            </Button>
            {onCreate && (
              <Button type="primary" icon={<Plus className="w-4 h-4" />} onClick={onCreate}>

                Create approval chain
              </Button>
            )}
          </Space>
        </Col>
      </Row>
    </Card>
  );
}
