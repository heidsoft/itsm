/**
 * SLA 过滤面板组件
 */

import React from 'react';
import { Card, Row, Col, Select, Input, DatePicker, Button, Space } from 'antd';
import { Search, RefreshCw } from 'lucide-react';
import type { SLAFilters } from '../types';
import { STATUS_OPTIONS, SEVERITY_OPTIONS, TYPE_OPTIONS } from '../utils/sla-chart-config';

interface SLAFilterPanelProps {
  filters: SLAFilters;
  onFiltersChange: (filters: SLAFilters) => void;
  onRefresh: () => void;
  loading?: boolean;
}

export const SLAFilterPanel: React.FC<SLAFilterPanelProps> = ({
  filters,
  onFiltersChange,
  onRefresh,
  loading = false,
}) => {
  const handleChange = (key: keyof SLAFilters, value: any) => {
    onFiltersChange({ ...filters, [key]: value });
  };

  const handleReset = () => {
    onFiltersChange({
      status: '',
      severity: '',
      type: '',
      dateRange: null,
      search: '',
    });
  };

  return (
    <Card title="过滤条件" size="small" className="mb-4">
      <Row gutter={16}>
        <Col span={5}>
          <Select
            placeholder="状态"
            style={{ width: '100%' }}
            value={filters.status}
            onChange={value => handleChange('status', value)}
            allowClear
          >
            {STATUS_OPTIONS.map(opt => (
              <Select.Option key={opt.value} value={opt.value}>
                {opt.label}
              </Select.Option>
            ))}
          </Select>
        </Col>
        <Col span={5}>
          <Select
            placeholder="严重程度"
            style={{ width: '100%' }}
            value={filters.severity}
            onChange={value => handleChange('severity', value)}
            allowClear
          >
            {SEVERITY_OPTIONS.map(opt => (
              <Select.Option key={opt.value} value={opt.value}>
                {opt.label}
              </Select.Option>
            ))}
          </Select>
        </Col>
        <Col span={5}>
          <Select
            placeholder="类型"
            style={{ width: '100%' }}
            value={filters.type}
            onChange={value => handleChange('type', value)}
            allowClear
          >
            {TYPE_OPTIONS.map(opt => (
              <Select.Option key={opt.value} value={opt.value}>
                {opt.label}
              </Select.Option>
            ))}
          </Select>
        </Col>
        <Col span={5}>
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            value={filters.dateRange}
            onChange={dates => handleChange('dateRange', dates)}
          />
        </Col>
        <Col span={4}>
          <Space>
            <Input
              placeholder="搜索"
              prefix={<Search size={14} />}
              value={filters.search}
              onChange={e => handleChange('search', e.target.value)}
              allowClear
            />
            <Button
              icon={<RefreshCw size={14} />}
              onClick={onRefresh}
              loading={loading}
            />
          </Space>
        </Col>
      </Row>
      <div style={{ marginTop: 12 }}>
        <Button type="link" onClick={handleReset}>
          重置过滤
        </Button>
      </div>
    </Card>
  );
};

SLAFilterPanel.displayName = 'SLAFilterPanel';
