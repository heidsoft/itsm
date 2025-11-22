'use client';

import React from 'react';
import { Card, Select, Input, DatePicker, Button, Space, Row, Col } from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  FilterOutlined,
  ClearOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;

export type TicketFilterState = {
  status: 'all' | 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'all' | 'p1' | 'p2' | 'p3' | 'p4';
  type?: 'all' | string;
  keyword: string;
  dateStart: string; // YYYY-MM-DD
  dateEnd: string; // YYYY-MM-DD
  sortBy: 'createdAt_desc' | 'createdAt_asc' | 'priority_desc' | 'priority_asc';
};

interface Props {
  filters: TicketFilterState;
  onFilterChange: (filters: Partial<TicketFilterState>) => void;
  onSearch?: (keyword: string) => void;
  onRefresh?: () => void;
  loading?: boolean;
}

const DEFAULT_VALUE: TicketFilterState = {
  status: 'all',
  priority: 'all',
  type: 'all',
  keyword: '',
  dateStart: '',
  dateEnd: '',
  sortBy: 'createdAt_desc',
};

// 组件策略：使用Ant Design组件提升UI质量
function TicketFilters({ 
  filters = DEFAULT_VALUE, 
  onFilterChange, 
  onSearch, 
  onRefresh,
  loading = false 
}: Props) {
  const [localKeyword, setLocalKeyword] = React.useState(filters?.keyword || '');

  // 同步外部filters到本地keyword
  React.useEffect(() => {
    if (filters?.keyword !== undefined) {
      setLocalKeyword(filters.keyword);
    }
  }, [filters?.keyword]);

  const handleStatusChange = (value: string) => {
    onFilterChange({ status: value as TicketFilterState['status'] });
  };

  const handlePriorityChange = (value: string) => {
    onFilterChange({ priority: value as TicketFilterState['priority'] });
  };

  const handleSortChange = (value: string) => {
    onFilterChange({ sortBy: value as TicketFilterState['sortBy'] });
  };

  const handleSearch = () => {
    if (onSearch) {
      onSearch(localKeyword);
    } else {
      onFilterChange({ keyword: localKeyword });
    }
  };

  const handleDateRangeChange = (dates: any) => {
    if (dates && dates[0] && dates[1]) {
      onFilterChange({
        dateStart: dates[0].format('YYYY-MM-DD'),
        dateEnd: dates[1].format('YYYY-MM-DD'),
      });
    } else {
      onFilterChange({
        dateStart: '',
        dateEnd: '',
      });
    }
  };

  const handleReset = () => {
    setLocalKeyword('');
    onFilterChange(DEFAULT_VALUE);
  };

  const dateRange =
    filters?.dateStart && filters?.dateEnd
      ? [dayjs(filters.dateStart), dayjs(filters.dateEnd)]
      : null;

  return (
    <Card
      className="mb-4"
      styles={{ body: { padding: '16px' } }}
      data-testid="ticket-filters"
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Row gutter={[12, 12]} align="middle">
          {/* 搜索框 */}
          <Col xs={24} sm={24} md={8} lg={6}>
            <Input
              placeholder="搜索工单标题或描述..."
              prefix={<SearchOutlined />}
              value={localKeyword}
              onChange={(e) => setLocalKeyword(e.target.value)}
              onPressEnter={handleSearch}
              allowClear
              data-testid="filter-keyword-input"
            />
          </Col>

          {/* 状态筛选 */}
          <Col xs={12} sm={8} md={4} lg={3}>
            <Select
              style={{ width: '100%' }}
              placeholder="状态"
              value={filters?.status || 'all'}
              onChange={handleStatusChange}
              data-testid="filter-status-select"
              options={[
                { label: '全部状态', value: 'all' },
                { label: '待处理', value: 'open' },
                { label: '处理中', value: 'in_progress' },
                { label: '已解决', value: 'resolved' },
                { label: '已关闭', value: 'closed' },
              ]}
            />
          </Col>

          {/* 优先级筛选 */}
          <Col xs={12} sm={8} md={4} lg={3}>
            <Select
              style={{ width: '100%' }}
              placeholder="优先级"
              value={filters?.priority || 'all'}
              onChange={handlePriorityChange}
              data-testid="filter-priority-select"
              options={[
                { label: '全部优先级', value: 'all' },
                { label: 'P1-紧急', value: 'p1' },
                { label: 'P2-高', value: 'p2' },
                { label: 'P3-中', value: 'p3' },
                { label: 'P4-低', value: 'p4' },
              ]}
            />
          </Col>

          {/* 日期范围 */}
          <Col xs={24} sm={12} md={6} lg={5}>
            <RangePicker
              style={{ width: '100%' }}
              value={dateRange as any}
              onChange={handleDateRangeChange}
              format="YYYY-MM-DD"
              placeholder={['开始日期', '结束日期']}
              data-testid="filter-date-range"
            />
          </Col>

          {/* 排序 */}
          <Col xs={12} sm={8} md={4} lg={3}>
            <Select
              style={{ width: '100%' }}
              placeholder="排序"
              value={filters?.sortBy || 'createdAt_desc'}
              onChange={handleSortChange}
              data-testid="filter-sort-select"
              options={[
                { label: '最新创建', value: 'createdAt_desc' },
                { label: '最早创建', value: 'createdAt_asc' },
                { label: '优先级↓', value: 'priority_desc' },
                { label: '优先级↑', value: 'priority_asc' },
              ]}
            />
          </Col>

          {/* 操作按钮 */}
          <Col xs={12} sm={8} md={6} lg={4}>
            <Space.Compact style={{ width: '100%' }}>
              <Button
                icon={<SearchOutlined />}
                type="primary"
                onClick={handleSearch}
                loading={loading}
                data-testid="filter-apply-btn"
              >
                搜索
              </Button>
              <Button
                icon={<ClearOutlined />}
                onClick={handleReset}
                data-testid="filter-reset-btn"
              >
                重置
              </Button>
              {onRefresh && (
                <Button
                  icon={<ReloadOutlined />}
                  onClick={onRefresh}
                  loading={loading}
                  data-testid="filter-refresh-btn"
                >
                  刷新
                </Button>
              )}
            </Space.Compact>
          </Col>
        </Row>
      </Space>
    </Card>
  );
}

export default TicketFilters;
export { TicketFilters };
