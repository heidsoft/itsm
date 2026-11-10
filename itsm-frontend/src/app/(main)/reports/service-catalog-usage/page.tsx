'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { Alert, Card, Col, Empty, Row, Select, Space, Spin, Typography } from 'antd';
import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { serviceCategoryLabel } from '@/constants/service-catalog';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import {
  ServiceCategory,
  ServiceStatus,
  type ServiceItem,
} from '@/types/service-catalog';

const { Title, Text } = Typography;

const COLORS = ['#1890ff', '#52c41a', '#faad14', '#ff4d4f', '#722ed1', '#13c2c2'];

const CATEGORY_LABELS: Record<ServiceCategory, string> = {
  [ServiceCategory.IT_SERVICE]: 'IT Services',
  [ServiceCategory.BUSINESS_SERVICE]: 'Business Services',
  [ServiceCategory.TECHNICAL_SERVICE]: 'Technical Services',
  [ServiceCategory.SUPPORT_SERVICE]: 'Support Services',
};

const STATUS_LABELS: Record<ServiceStatus, string> = {
  [ServiceStatus.PUBLISHED]: 'Published',
  [ServiceStatus.DRAFT]: 'Draft',
  [ServiceStatus.RETIRED]: 'Offline',
};

interface ChartDataItem {
  name: string;
  value: number;
  color: string;
}

const ServiceCatalogUsagePage = () => {
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [statusFilter, setStatusFilter] = useState<'all' | ServiceStatus>('all');

  const loadData = async () => {
    setLoading(true);
    try {
      const { services: servicesData } = await ServiceCatalogApi.getServices({
        page: 1,
        pageSize: 100,
      });
      setServices(servicesData ?? []);
    } catch (error) {
      console.error('Failed to load service catalog data:', error);
      setServices([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  // 状态筛选：作用于本页全部图表与卡片（客户端过滤目录存量，口径透明）
  const filteredServices = useMemo(
    () =>
      statusFilter === 'all'
        ? services
        : services.filter((s) => s.status === statusFilter),
    [services, statusFilter]
  );

  const byCategory: ChartDataItem[] = useMemo(() => {
    const counts = new Map<ServiceCategory, number>();
    filteredServices.forEach((s) => {
      counts.set(s.category, (counts.get(s.category) ?? 0) + 1);
    });
    return Array.from(counts.entries()).map(([category, value], index) => ({
      name: CATEGORY_LABELS[category] ?? serviceCategoryLabel(String(category)),
      value,
      color: COLORS[index % COLORS.length],
    }));
  }, [filteredServices]);

  const byStatus: ChartDataItem[] = useMemo(() => {
    const counts = new Map<ServiceStatus, number>();
    filteredServices.forEach((s) => {
      counts.set(s.status, (counts.get(s.status) ?? 0) + 1);
    });
    return Array.from(counts.entries()).map(([status, value]) => ({
      name: STATUS_LABELS[status] ?? String(status),
      value,
      color:
        status === ServiceStatus.PUBLISHED
          ? '#52c41a'
          : status === ServiceStatus.DRAFT
            ? '#1890ff'
            : '#d9d9d9',
    }));
  }, [filteredServices]);

  const CustomTooltip = ({
    active,
    payload,
  }: {
    active?: boolean;
    payload?: Array<{ name: string; value: number; color: string }>;
  }) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 rounded-lg shadow-lg border border-gray-200">
          <p className="font-semibold text-gray-800">{`${payload[0].name}`}</p>
          <p
            className="text-sm"
            style={{ color: payload[0].color }}
          >{`Quantity: ${payload[0].value}`}</p>
        </div>
      );
    }
    return null;
  };

  const hasData = filteredServices.length > 0;

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>Service Catalog Usage Report</Title>
        <p className="text-gray-500 mt-1">Service catalog inventory structure and classification distribution</p>
      </header>

      {/* 控制栏 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Space>
              <Select
                value={statusFilter}
                onChange={(v) => setStatusFilter(v)}
                style={{ width: 120 }}
                options={[
                  { value: 'all', label: 'All Services' },
                  { value: ServiceStatus.PUBLISHED, label: 'Published' },
                  { value: ServiceStatus.DRAFT, label: 'Draft' },
                  { value: ServiceStatus.RETIRED, label: 'Offline' },
                ]}
              />
              <Text type="secondary">Filter applies to all cards and charts on this page</Text>
            </Space>
          </Col>
          <Col>
            <button
              onClick={loadData}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >

              Refresh Data
            </button>
          </Col>
        </Row>
      </Card>

      <Alert
        className="mb-6"
        type="info"
        showIcon
        title="Caliber description"
        description="Request volume statistics (the number of times each service is applied for) rely on back-end collection and are not currently provided; this page shows the inventory structure of the service directory, excluding request data."
      />

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <Spin size="large" description="Loading report data..." />
        </div>
      ) : (
        <>
          {/* 统计卡片 */}
          <Row gutter={[16, 16]} className="mb-6">
            <Col xs={24} sm={8}>
              <Card>
                <div className="text-center">
                  <div className="text-3xl font-bold text-blue-600">
                    {filteredServices.length}
                  </div>
                  <div className="text-gray-500">Total Services</div>
                </div>
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <div className="text-center">
                  <div className="text-3xl font-bold text-green-600">
                    {filteredServices.filter((s) => s.status === ServiceStatus.PUBLISHED).length}
                  </div>
                  <div className="text-gray-500">Published service</div>
                </div>
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <div className="text-center">
                  <div className="text-3xl font-bold text-orange-600">
                    {filteredServices.filter((s) => s.status === ServiceStatus.DRAFT).length}
                  </div>
                  <div className="text-gray-500">Draft service</div>
                </div>
              </Card>
            </Col>
          </Row>

          {/* 图表区域 */}
          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card title="Distribution by service category">
                {hasData && byCategory.length > 0 ? (
                  <ResponsiveContainer width="100%" height={300}>
                    <PieChart>
                      <Pie
                        data={byCategory}
                        cx="50%"
                        cy="50%"
                        outerRadius={100}
                        dataKey="value"
                        nameKey="name"
                        label={({ name, percent }) => `${name} ${((percent || 0) * 100).toFixed(0)}%`}
                      >
                        {byCategory.map((entry, index) => (
                          <Cell
                            key={`cell-${index}`}
                            fill={entry.color || COLORS[index % COLORS.length]}
                          />
                        ))}
                      </Pie>
                      <Tooltip content={<CustomTooltip />} />
                      <Legend />
                    </PieChart>
                  </ResponsiveContainer>
                ) : (
                  <Empty description="No service data yet" />
                )}
              </Card>
            </Col>

            <Col xs={24} lg={12}>
              <Card title="Distribution by state">
                {hasData && byStatus.length > 0 ? (
                  <ResponsiveContainer width="100%" height={300}>
                    <BarChart data={byStatus}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="name" />
                      <YAxis allowDecimals={false} />
                      <Tooltip content={<CustomTooltip />} />
                      <Legend />
                      <Bar dataKey="value" name="Service quantity" fill="#1890ff">
                        {byStatus.map((entry, index) => (
                          <Cell
                            key={`cell-${index}`}
                            fill={entry.color || COLORS[index % COLORS.length]}
                          />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                ) : (
                  <Empty description="No service data yet" />
                )}
              </Card>
            </Col>
          </Row>
        </>
      )}
    </div>
  );
};

export default ServiceCatalogUsagePage;
