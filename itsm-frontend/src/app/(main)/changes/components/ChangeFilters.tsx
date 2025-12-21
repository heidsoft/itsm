'use client';

import React from 'react';
import { Card, Row, Col, Input, Select, Button } from 'antd';
import { RefreshCw } from 'lucide-react';

const { Search: SearchInput } = Input;
const { Option } = Select;

interface ChangeFiltersProps {
  loading: boolean;
  searchText: string;
  onSearchTextChange: (value: string) => void;
  filter: string;
  onFilterChange: (value: string) => void;
  onFetchChanges: () => void;
}

export const ChangeFilters: React.FC<ChangeFiltersProps> = ({
  loading,
  searchText,
  onSearchTextChange,
  filter,
  onFilterChange,
  onFetchChanges,
}) => {
  return (
    <Card className='mb-6 shadow-sm border-0'>
      <div className='mb-4'>
        <h3 className='text-lg font-semibold text-gray-800 mb-2'>筛选条件</h3>
      </div>
      <Row gutter={[16, 16]} align='middle'>
        <Col xs={24} sm={12} md={8}>
          <SearchInput
            placeholder='搜索变更标题或描述'
            value={searchText}
            onChange={e => onSearchTextChange(e.target.value)}
            onSearch={onFetchChanges}
            className='rounded-lg'
            size='large'
          />
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Select
            value={filter}
            onChange={onFilterChange}
            className='w-full'
            size='large'
            placeholder='选择状态'
          >
            <Option value='全部'>
              <div className='flex items-center'>
                <div className='w-2 h-2 rounded-full bg-gray-400 mr-2'></div>
                全部状态
              </div>
            </Option>
            <Option value='pending'>
              <div className='flex items-center'>
                <div className='w-2 h-2 rounded-full bg-orange-500 mr-2'></div>
                待审批
              </div>
            </Option>
            <Option value='approved'>
              <div className='flex items-center'>
                <div className='w-2 h-2 rounded-full bg-blue-500 mr-2'></div>
                已批准
              </div>
            </Option>
            <Option value='completed'>
              <div className='flex items-center'>
                <div className='w-2 h-2 rounded-full bg-green-500 mr-2'></div>
                已完成
              </div>
            </Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button
            icon={<RefreshCw size={20} />}
            onClick={onFetchChanges}
            loading={loading}
            size='large'
            className='w-full bg-blue-50 border-blue-200 text-blue-600 hover:bg-blue-100 hover:border-blue-300'
          >
            刷新
          </Button>
        </Col>
      </Row>
    </Card>
  );
};
