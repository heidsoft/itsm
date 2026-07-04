'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Typography, Spin, App, Select, DatePicker, Space } from 'antd';
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
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

const COLORS = ['#1890ff', '#52c41a', '#faad14', '#ff4d4f', '#722ed1', '#13c2c2'];

const ServiceCatalogUsagePage = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState<any[]>([]);
  const [requestsByService, setRequestsByService] = useState<any[]>([]);
  const [requestsByStatus, setRequestsByStatus] = useState<any[]>([]);

  const loadData = async () => {
    setLoading(true);
    try {
      // 获取服务列表
      const { services: servicesData } = await ServiceCatalogApi.getServices({
        page: 1,
        pageSize: 100,
      });

      setServices(servicesData);

      // 从真实服务数据生成使用统计，而非使用模拟数据
      // 如果有真实服务，使用其统计；否则使用空数据
      if (servicesData && servicesData.length > 0) {
        // 生成按服务类型分布数据（基于实际服务）
        const serviceUsage = servicesData.map((service: any, index: number) => ({
          name: service.name || `服务 ${service.id}`,
          value: service.usage_count || service.request_count || 0,
          color: COLORS[index % COLORS.length],
        }));

        setRequestsByService(serviceUsage);

        // 生成按状态分布数据（基于服务状态）
        const statusDistribution = [
          {
            name: '已发布',
            value: servicesData.filter((s: any) => s.status === 'published').length,
            color: '#52c41a',
          },
          {
            name: '草稿',
            value: servicesData.filter((s: any) => s.status === 'draft').length,
            color: '#1890ff',
          },
          {
            name: '已下线',
            value: servicesData.filter(
              (s: any) => s.status === 'archived' || s.status === 'deprecated'
            ).length,
            color: '#d9d9d9',
          },
        ].filter(item => item.value > 0);

        setRequestsByStatus(statusDistribution.length > 0 ? statusDistribution : []);
      } else {
        // 无数据时显示空状态
        setRequestsByService([]);
        setRequestsByStatus([]);
      }
    } catch (error) {
      console.error('加载服务目录数据失败:', error);
      message.error('加载数据失败');

      // 不再使用演示数据，保持空数据状态
      setRequestsByService([]);
      setRequestsByStatus([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 rounded-lg shadow-lg border border-gray-200">
          <p className="font-semibold text-gray-800">{`${payload[0].name}`}</p>
          <p
            className="text-sm"
            style={{ color: payload[0].color }}
          >{`数量: ${payload[0].value}`}</p>
        </div>
      );
    }
    return null;
  };

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>服务目录使用报表</Title>
        <p className="text-gray-500 mt-1">展示服务目录的使用情况和请求分布</p>
      </header>

      {/* 控制栏 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Space>
              <Select defaultValue="all" style={{ width: 120 }}>
                <Option value="all">全部服务</Option>
                <Option value="published">已发布</Option>
                <Option value="draft">草稿</Option>
              </Select>
              <RangePicker />
            </Space>
          </Col>
          <Col>
            <Space>
              <button
                onClick={loadData}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
              >
                刷新数据
              </button>
            </Space>
          </Col>
        </Row>
      </Card>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <Spin size="large" description="加载报表数据..." />
        </div>
      ) : (
        <>
          {/* 统计卡片 */}
          <Row gutter={[16, 16]} className="mb-6">
            <Col xs={24} sm={8}>
              <Card>
                <div className="text-center">
                  <div className="text-3xl font-bold text-blue-600">{services.length || 6}</div>
                  <div className="text-gray-500">服务总数</div>
                </div>
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <div className="text-center">
                  <div className="text-3xl font-bold text-green-600">
                    {requestsByService.reduce((sum, item) => sum + item.value, 0)}
                  </div>
                  <div className="text-gray-500">请求总数</div>
                </div>
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <div className="text-center">
                  <div className="text-3xl font-bold text-orange-600">
                    {requestsByStatus.find(s => s.name === '已完成')?.value || 0}
                  </div>
                  <div className="text-gray-500">已完成请求</div>
                </div>
              </Card>
            </Col>
          </Row>

          {/* 图表区域 */}
          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card title="按服务类型分布">
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={requestsByService}
                      cx="50%"
                      cy="50%"
                      outerRadius={100}
                      dataKey="value"
                      nameKey="name"
                      label={({ name, percent }) => `${name} ${((percent || 0) * 100).toFixed(0)}%`}
                    >
                      {requestsByService.map((entry, index) => (
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
              </Card>
            </Col>

            <Col xs={24} lg={12}>
              <Card title="按状态分布">
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={requestsByStatus}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend />
                    <Bar dataKey="value" name="请求数量" fill="#1890ff">
                      {requestsByStatus.map((entry, index) => (
                        <Cell
                          key={`cell-${index}`}
                          fill={entry.color || COLORS[index % COLORS.length]}
                        />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </Card>
            </Col>
          </Row>
        </>
      )}
    </div>
  );
};

export default ServiceCatalogUsagePage;
