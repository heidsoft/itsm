"use client";

import React from "react";
import { Input, Select, Button, Space, Card, Row, Col } from "antd";
import { Search, Filter, RefreshCw, Download } from "lucide-react";

const { Option } = Select;

interface FilterOption {
  key: string;
  label: string;
  options: { value: string; label: string }[];
  placeholder?: string;
  allowClear?: boolean;
}

interface SearchFiltersProps {
  searchPlaceholder?: string;
  filters?: FilterOption[];
  onSearch?: (value: string) => void;
  onFilterChange?: (key: string, value: string) => void;
  onReset?: () => void;
  onExport?: () => void;
  showExport?: boolean;
  loading?: boolean;
  className?: string;
}

export const SearchFilters: React.FC<SearchFiltersProps> = ({
  searchPlaceholder = "搜索...",
  filters = [],
  onSearch,
  onFilterChange,
  onReset,
  onExport,
  showExport = false,
  loading = false,
  className = "",
}) => {
  const handleSearch = (value: string) => {
    onSearch?.(value);
  };

  const handleFilterChange = (key: string, value: string) => {
    onFilterChange?.(key, value);
  };

  const handleReset = () => {
    onReset?.();
  };

  return (
    <Card className={`mb-4 ${className}`}>
      <Row gutter={[16, 16]} align="middle">
        {/* 搜索框 */}
        <Col xs={24} sm={12} md={8} lg={6}>
          <Input.Search
            placeholder={searchPlaceholder}
            onSearch={handleSearch}
            allowClear
            enterButton={<Search size={16} />}
            size="large"
          />
        </Col>

        {/* 过滤器 */}
        {filters.map((filter) => (
          <Col xs={24} sm={12} md={8} lg={6} key={filter.key}>
            <Select
              placeholder={filter.placeholder || filter.label}
              allowClear={filter.allowClear !== false}
              size="large"
              style={{ width: "100%" }}
              onChange={(value) => handleFilterChange(filter.key, value)}
            >
              {filter.options.map((option) => (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              ))}
            </Select>
          </Col>
        ))}

        {/* 操作按钮 */}
        <Col xs={24} sm={24} md={24} lg={24 - filters.length * 6}>
          <Space size="middle" className="w-full justify-end">
            <Button
              icon={<RefreshCw size={16} />}
              onClick={handleReset}
              loading={loading}
            >
              重置
            </Button>
            {showExport && (
              <Button
                icon={<Download size={16} />}
                onClick={onExport}
                type="primary"
              >
                导出
              </Button>
            )}
          </Space>
        </Col>
      </Row>
    </Card>
  );
};
