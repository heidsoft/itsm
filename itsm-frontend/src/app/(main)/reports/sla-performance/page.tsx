'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Typography,
  Spin,
  message,
  Statistic,
  Button,
  Progress,
  Space,
} from 'antd';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts';
import { ReloadOutlined, CheckCircle, CloseCircle, Clock } from '@ant-design/icons';

const { Title, Text } = Typography;

const COLORS = {
  met: '#52c41a',
  breached: '#ff4d4f',
  warning: '#faad14',
};

interface SLAData {
  name: string;
  met: number;
  breached: number;
  compliance: number;
}

const SLAPerformanceReport = () => {
  const [loading, setLoading] = useState(false);
  const [slaData, setSLAData] = useState<SLAData[]>([]);

  const loadData = async () => {
    setLoading(true);
    try {
      // 模拟SLA性能数据 - 实际项目中应调用真实API
      const data: SLAData[] = [
        { name: '事件响应', met: 95, breached: 5, compliance: 95 },
        { name: '请求处理', met: 98, breached: 2, compliance: 98 },
        { name: '问题解决', met: 88, breached: 12, compliance: 88 },
        { name: '变更实施', met: 92, breached: 8, compliance: 92 },
        { name: '服务可用性', met: 99, breached: 1, compliance: 99 },
      ];

      setSLAData(data);
    } catch (error) {
      console.error('加载SLA性能数据失败:', error);
      message.error('加载数据失败');

      setSLAData([
        { name: '事件响应', met: 95, breached: 5, compliance: 95 },
        { name: '请求处理', met: 98, breached: 2, compliance: 98 },
        { name: '问题解决', met: 88, breached: 12, compliance: 88 },
        { name: '变更实施', met: 92, breached: 8, compliance: 92 },
        { name: '服务可用性', met: 99, breached: 1, compliance: 99 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const totalMet = slaData.reduce((sum, d) => sum + d.met, 0);
  const totalBreached = slaData.reduce((sum, d) => sum + d.breached, 0);
  const avgCompliance = slaData.length > 0
    ? slaData.reduce((sum, d) => sum + d.compliance, 0) / slaData.length
    : 0;

  const pieData = [
    { name: '达标', value: totalMet, color: COLORS.met },
    { name: '违规', value: totalBreached, color: COLORS.breached },
  ];

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 rounded-lg shadow-lg border border-gray-200">
          <p className="font-semibold text-gray-800 mb-2">{`SLA类型: ${label}`}</p>
          {payload.map((entry: any, index: number) => (
            <p key={index} className="text-sm" style={{ color: entry.color }}>
              {`${entry.name}: ${entry.value}%`}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  const getComplianceStatus = (compliance: number) => {
    if (compliance >= 95) return 'success';
    if (compliance >= 85) return 'normal';
    return 'exception';
  };

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>SLA性能报表</Title>
        <p className="text-gray-500 mt-1">展示服务级别协议的达成情况和性能指标</p>
      </header>

      {/* 控制栏 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Text className="text-gray-600">SLA合规率监控</Text>
          </Col>
          <Col>
            <Button icon={<ReloadOutlined />} onClick={loadData}>
              刷新数据
            </Button>
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
                <Statistic
                  title="平均合规率"
                  value={avgCompliance}
                  suffix="%"
                  valueStyle={{
                    color: avgCompliance >= 95 ? '#52c41a' : avgCompliance >= 85 ? '#faad14' : '#ff4d4f'
                  }}
                  prefix={<CheckCircle />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title="达标次数"
                  value={totalMet}
                  valueStyle={{ color: COLORS.met }}
                  prefix={<CheckCircle />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title="违规次数"
                  value={totalBreached}
                  valueStyle={{ color: COLORS.breached }}
                  prefix={<CloseCircle />}
                />
              </Card>
            </Col>
          </Row>

          {/* 图表区域 */}
          <Row gutter={[16, 16]} className="mb-6">
            <Col xs={24} lg={16}>
              <Card title="各类型SLA达成情况">
                <ResponsiveContainer width="100%" height={350}>
                  <BarChart data={slaData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend />
                    <Bar dataKey="met" name="达标" stackId="a" fill={COLORS.met} />
                    <Bar dataKey="breached" name="违规" stackId="a" fill={COLORS.breached} />
                  </BarChart>
                </ResponsiveContainer>
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card title="总体达标率">
                <div className="text-center py-8">
                  <div className="text-5xl font-bold mb-4" style={{ color: COLORS.met }}>
                    {((totalMet / (totalMet + totalBreached)) * 100).toFixed(1)}%
                  </div>
                  <Text type="secondary">SLA总体达标率</Text>
                  <Progress
                    percent={((totalMet / (totalMet + totalBreached)) * 100)}
                    strokeColor={COLORS.met}
                    showInfo={false}
                    className="mt-4"
                  />
                </div>
              </Card>
            </Col>
          </Row>

          {/* 合规率详情 */}
          <Card title="各SLA合规率详情">
            <Row gutter={[16, 16]}>
              {slaData.map((sla, index) => (
                <Col xs={24} sm={12} md={8} key={index}>
                  <Card size="small" className="h-full">
                    <div className="flex items-center justify-between mb-4">
                      <Text strong>{sla.name}</Text>
                      <Text
                        type={getComplianceStatus(sla.compliance) as any}
                        strong
                      >
                        {sla.compliance}%
                      </Text>
                    </div>
                    <Progress
                      percent={sla.compliance}
                      strokeColor={
                        sla.compliance >= 95 ? COLORS.met :
                        sla.compliance >= 85 ? COLORS.warning : COLORS.breached
                      }
                      showInfo={false}
                    />
                    <div className="flex justify-between mt-2 text-sm text-gray-500">
                      <span>
                        <CheckCircle className="inline mr-1" style={{ color: COLORS.met }} />
                        {sla.met}%
                      </span>
                      <span>
                        <CloseCircle className="inline mr-1" style={{ color: COLORS.breached }} />
                        {sla.breached}%
                      </span>
                    </div>
                  </Card>
                </Col>
              ))}
            </Row>
          </Card>
        </>
      )}
    </div>
  );
};

export default SLAPerformanceReport;
