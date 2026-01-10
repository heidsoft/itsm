'use client';

/**
 * 审批链筛选器组件
 */

import React, { useState, useCallback } from 'react';
import { Card, Row, Col, Input, Select, DatePicker, Button, Space } from 'antd';
import { SearchOutlined, ReloadOutlined, FilterOutlined } from '@ant-design/icons';
import { ApprovalChainFilters as ApprovalChainFiltersType } from '@/types/approval-chain';
import { useDebouncedCallback } from '@/lib/component-utils';
import dayjs from 'dayjs';

const { Search } = Input;
const { Option } = Select;
const { RangePicker } = DatePicker;

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

  // 防抖搜索
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
    (status: ('active' | 'inactive')[]) => {
      const newFilters = { ...localFilters, status };
      setLocalFilters(newFilters);
      onFilterChange(newFilters);
    },
    [localFilters, onFilterChange]
  );

  const handleDateRangeChange = useCallback(
    (dates: any, dateStrings: [string, string]) => {
      const newFilters = {
        ...localFilters,
        dateRange: dates
          ? {
              field: 'created' as const,
              start: dateStrings[0],
              end: dateStrings[1],
            }
          : undefined,
      };
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
    <Card className='mb-6'>
      <Row gutter={[16, 16]} align='middle'>
        <Col xs={24} sm={12} md={8} lg={6}>
          <div className='mb-2'>
            <span className='text-sm font-medium text-gray-700'>搜索名称</span>
          </div>
          <Search
            placeholder='搜索审批链名称...'
            value={localFilters.name || ''}
            onChange={e => handleSearch(e.target.value)}
            onSearch={handleSearch}
            allowClear
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <div className='mb-2'>
            <span className='text-sm font-medium text-gray-700'>状态</span>
          </div>
          <Select
            mode='multiple'
            placeholder='选择状态'
            value={localFilters.status}
            onChange={handleStatusChange}
            style={{ width: '100%' }}
            allowClear
          >
            <Option value='active'>活跃</Option>
            <Option value='inactive'>非活跃</Option>
          </Select>
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <div className='mb-2'>
            <span className='text-sm font-medium text-gray-700'>创建时间</span>
          </div>
          <RangePicker
            placeholder={['开始日期', '结束日期']}
            value={
              localFilters.dateRange
                ? [dayjs(localFilters.dateRange.start), dayjs(localFilters.dateRange.end)]
                : null
            }
            onChange={handleDateRangeChange}
            style={{ width: '100%' }}
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <div className='mb-2'>
            <span className='text-sm font-medium text-gray-700'>操作</span>
          </div>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
              刷新
            </Button>
            <Button icon={<FilterOutlined />} onClick={handleReset}>
              重置
            </Button>
          </Space>
        </Col>
      </Row>
    </Card>
  );
}
