'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Typography,
  Spin,
  message,
  Select,
  DatePicker,
  Space,
} from 'antd';
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
      } as any);

      setServices(servicesData);

      // 模拟请求数据 - 实际项目中应调用真实API
      const mockRequestsByService = [
        { name: '账号申请', value: 156, color: '#1890ff' },
        { name: '权限变更', value: 98, color: '#52c41a' },
        { name: '设备报修', value: 87, color: '#faad14' },
        { name: '软件安装', value: 65, color: '#ff4d4f' },
        { name: '网络申请', value: 43, color: '#722ed1' },
        { name: '其他服务', value: 32, color: '#13c2c2' },
      ];

      const mockRequestsByStatus = [
        { name: '已完成', value: 312, color: '#52c41a' },
        { name: '处理中', value: 89, color: '#1890ff' },
        { name: '待审批', value: 45, color: '#faad14' },
        { name: '已取消', value: 23, color: '#d9d9d9' },
      ];

      setRequestsByService(mockRequestsByService);
      setRequestsByStatus(mockRequestsByStatus);
    } catch (error) {
      console.error('加载服务目录数据失败:', error);
      message.error('加载数据失败，使用演示数据');

      // 演示数据
      setRequestsByService([
        { name: '账号申请', value: 156, color: '#1890ff' },
        { name: '权限变更', value: 98, color: '#52c41a' },
        { name: '设备报修', value: 87, color: '#faad14' },
        { name: '软件安装', value: 65, color: '#ff4d4f' },
        { name: '网络申请', value: 43, color: '#722ed1' },
        { name: '其他服务', value: 32, color: '#13c2c2' },
      ]);

      setRequestsByStatus([
        { name: '已完成', value: 312, color: '#52c41a' },
        { name: '处理中', value: 89, color: '#1890ff' },
        { name: '待审批', value: 45, color: '#faad14' },
        { name: '已取消', value: 23, color: '#d9d9d9' },
      ]);
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
          <p className="text-sm" style={{ color: payload[0].color }}>{`数量: ${payload[0].value}`}</p>
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
          <Spin size="large" tip="加载报表数据..." />
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
                        <Cell key={`cell-${index}`} fill={entry.color || COLORS[index % COLORS.length]} />
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
                        <Cell key={`cell-${index}`} fill={entry.color || COLORS[index % COLORS.length]} />
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
