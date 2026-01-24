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
  Space,
  Button,
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
  LineChart,
  Line,
} from 'recharts';
import { ReloadOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

const COLORS = {
  completeness: '#1890ff',
  accuracy: '#52c41a',
  consistency: '#faad14',
};

interface QualityData {
  name: string;
  completeness: number;
  accuracy: number;
  consistency: number;
}

const CMDBQualityReport = () => {
  const [loading, setLoading] = useState(false);
  const [qualityData, setQualityData] = useState<QualityData[]>([]);

  const loadData = async () => {
    setLoading(true);
    try {
      // 模拟CMDB质量数据 - 实际项目中应调用真实API
      const data: QualityData[] = [
        { name: '服务器', completeness: 95, accuracy: 90, consistency: 88 },
        { name: '应用系统', completeness: 80, accuracy: 85, consistency: 82 },
        { name: '数据库', completeness: 90, accuracy: 92, consistency: 95 },
        { name: '网络设备', completeness: 85, accuracy: 88, consistency: 90 },
        { name: '存储设备', completeness: 75, accuracy: 82, consistency: 78 },
        { name: '安全设备', completeness: 88, accuracy: 91, consistency: 85 },
      ];

      setQualityData(data);
    } catch (error) {
      console.error('加载CMDB质量数据失败:', error);
      message.error('加载数据失败');

      // 演示数据
      setQualityData([
        { name: '服务器', completeness: 95, accuracy: 90, consistency: 88 },
        { name: '应用系统', completeness: 80, accuracy: 85, consistency: 82 },
        { name: '数据库', completeness: 90, accuracy: 92, consistency: 95 },
        { name: '网络设备', completeness: 85, accuracy: 88, consistency: 90 },
        { name: '存储设备', completeness: 75, accuracy: 82, consistency: 78 },
        { name: '安全设备', completeness: 88, accuracy: 91, consistency: 85 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const avgCompleteness = qualityData.length > 0
    ? qualityData.reduce((sum, d) => sum + d.completeness, 0) / qualityData.length
    : 0;
  const avgAccuracy = qualityData.length > 0
    ? qualityData.reduce((sum, d) => sum + d.accuracy, 0) / qualityData.length
    : 0;
  const avgConsistency = qualityData.length > 0
    ? qualityData.reduce((sum, d) => sum + d.consistency, 0) / qualityData.length
    : 0;

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 rounded-lg shadow-lg border border-gray-200">
          <p className="font-semibold text-gray-800 mb-2">{`分类: ${label}`}</p>
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

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>CMDB数据质量报表</Title>
        <p className="text-gray-500 mt-1">展示配置管理数据库的数据质量指标</p>
      </header>

      {/* 控制栏 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Text className="text-gray-600">配置项数据质量监控</Text>
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
                  title="平均完整度"
                  value={avgCompleteness}
                  suffix="%"
                  valueStyle={{ color: COLORS.completeness }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title="平均准确度"
                  value={avgAccuracy}
                  suffix="%"
                  valueStyle={{ color: COLORS.accuracy }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title="平均一致度"
                  value={avgConsistency}
                  suffix="%"
                  valueStyle={{ color: COLORS.consistency }}
                />
              </Card>
            </Col>
          </Row>

          {/* 质量指标对比图 */}
          <Card title="各类别数据质量对比" className="mb-6">
            <ResponsiveContainer width="100%" height={350}>
              <BarChart data={qualityData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis domain={[0, 100]} />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                <Bar dataKey="completeness" name="完整度" fill={COLORS.completeness} />
                <Bar dataKey="accuracy" name="准确度" fill={COLORS.accuracy} />
                <Bar dataKey="consistency" name="一致度" fill={COLORS.consistency} />
              </BarChart>
            </ResponsiveContainer>
          </Card>

          {/* 质量趋势 */}
          <Card title="数据质量趋势">
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={qualityData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis domain={[0, 100]} />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                <Line
                  type="monotone"
                  dataKey="completeness"
                  name="完整度"
                  stroke={COLORS.completeness}
                  strokeWidth={2}
                />
                <Line
                  type="monotone"
                  dataKey="accuracy"
                  name="准确度"
                  stroke={COLORS.accuracy}
                  strokeWidth={2}
                />
                <Line
                  type="monotone"
                  dataKey="consistency"
                  name="一致度"
                  stroke={COLORS.consistency}
                  strokeWidth={2}
                />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </>
      )}
    </div>
  );
};

export default CMDBQualityReport;
