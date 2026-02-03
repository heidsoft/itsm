"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Space,
  Typography,
  Statistic,
  Row,
  Col,
  Progress,
  Alert,
  Button,
  Select,
  DatePicker,
  Tooltip,
  Badge,
  Table,
  Tag,
} from "antd";
import {
  TrendingUp,
  TrendingDown,
  BarChart3,
  Activity,
  AlertTriangle,
  Brain,
  Calendar,
  Users,
  Clock,
  Zap,
  Target,
  PieChart,
} from "lucide-react";
// 暂时移除图表依赖，使用简单的进度条和统计展示

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

interface PredictiveMetrics {
  ticketVolume: {
    current: number;
    predicted: number;
    trend: "up" | "down" | "stable";
    confidence: number;
  };
  resourceDemand: {
    current: number;
    predicted: number;
    shortage: number;
    recommendation: string;
  };
  slaBreachRisk: {
    riskLevel: "low" | "medium" | "high";
    probability: number;
    affectedTickets: number;
    mitigation: string;
  };
  seasonalPatterns: {
    peakPeriods: string[];
    lowPeriods: string[];
    recommendations: string[];
  };
}

interface TrendData {
  date: string;
  actual: number;
  predicted: number;
  confidence: number;
}

interface ResourceData {
  role: string;
  current: number;
  required: number;
  gap: number;
  utilization: number;
}

export const PredictiveAnalytics: React.FC = () => {
  const [metrics, setMetrics] = useState<PredictiveMetrics | null>(null);
  const [trendData, setTrendData] = useState<TrendData[]>([]);
  const [resourceData, setResourceData] = useState<ResourceData[]>([]);
  const [timeRange, setTimeRange] = useState<[string, string]>(["7d", "30d"]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadPredictiveData();
  }, [timeRange]);

  const loadPredictiveData = async () => {
    setLoading(true);
    try {
      const { TicketPredictionApi } = await import('@/lib/api/ticket-prediction-api');
      const { SLAApi } = await import('@/lib/api/sla-api');

      // 计算日期范围
      const endDate = new Date();
      const startDate = new Date();
      const rangeDays = parseInt(timeRange[0]);
      startDate.setDate(endDate.getDate() - rangeDays);

      const dateRange: [string, string] = [
        startDate.toISOString().split('T')[0],
        endDate.toISOString().split('T')[0]
      ];

      // 并行请求数据
      const [predictionReport, slaMonitoring] = await Promise.all([
        TicketPredictionApi.getTrendPrediction({
          time_range: dateRange,
          prediction_period: rangeDays <= 30 ? 'week' : 'month',
          model_type: 'arima'
        }),
        SLAApi.getSLAMonitoring({
          start_time: dateRange[0],
          end_time: dateRange[1]
        })
      ]);

      // 转换预测数据
      const mappedMetrics: PredictiveMetrics = {
        ticketVolume: {
          current: predictionReport.data[predictionReport.data.length - 1]?.actual || 0,
          predicted: predictionReport.metrics.next_week_prediction || 0,
          trend: predictionReport.metrics.trend || 'stable',
          confidence: (predictionReport.metrics.accuracy || 0) * 100,
        },
        resourceDemand: {
          // 暂时没有资源需求API，使用默认值或从预测中推断
          current: 0,
          predicted: 0,
          shortage: 0,
          recommendation: "暂无资源建议",
        },
        slaBreachRisk: {
          riskLevel: slaMonitoring.violation_rate > 20 ? 'high' : slaMonitoring.violation_rate > 10 ? 'medium' : 'low',
          probability: slaMonitoring.violation_rate || 0,
          affectedTickets: slaMonitoring.at_risk_tickets || 0,
          mitigation: slaMonitoring.at_risk_tickets > 0 ? "建议优先处理即将违约的工单" : "当前SLA风险较低",
        },
        seasonalPatterns: {
          peakPeriods: [], // API暂不支持
          lowPeriods: [], // API暂不支持
          recommendations: predictionReport.recommendations || [],
        },
      };

      const mappedTrendData: TrendData[] = predictionReport.data.map(d => ({
        date: d.date,
        actual: d.actual || 0,
        predicted: d.predicted,
        confidence: d.confidence || 0.85
      }));

      // 暂时清空资源数据，直到有真实API
      setResourceData([]); 

      setMetrics(mappedMetrics);
      setTrendData(mappedTrendData);
    } catch (error) {
      console.error("加载预测数据失败:", error);
      // 出错时不显示Mock数据，保持空状态或显示错误
      setMetrics(null);
      setTrendData([]);
    } finally {
      setLoading(false);
    }
  };

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case "up":
        return <TrendingUp className="text-red-500" />;
      case "down":
        return <TrendingDown className="text-green-500" />;
      default:
        return <Activity className="text-blue-500" />;
    }
  };

  const getRiskColor = (level: string): string => {
    switch (level) {
      case "high":
        return "red";
      case "medium":
        return "orange";
      case "low":
        return "green";
      default:
        return "default";
    }
  };

  const getUtilizationColor = (utilization: number): string => {
    if (utilization >= 90) return "red";
    if (utilization >= 80) return "orange";
    if (utilization >= 70) return "blue";
    return "green";
  };

  // 图表配置已移除，使用占位符UI

  if (loading) {
    return (
      <Card>
        <div className="animate-pulse space-y-4">
          <div className="h-4 bg-gray-200 rounded w-1/3"></div>
          <div className="space-y-3">
            <div className="h-3 bg-gray-200 rounded"></div>
            <div className="h-3 bg-gray-200 rounded w-5/6"></div>
            <div className="h-3 bg-gray-200 rounded w-4/6"></div>
          </div>
        </div>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* 预测概览 */}
      <Card
        title={
          <Space>
            <Brain className="text-purple-600" />
            <span>智能预测分析</span>
          </Space>
        }
        extra={
          <Space>
            <Select
              value={timeRange[0]}
              onChange={(value) => setTimeRange([value, timeRange[1]])}
              style={{ width: 100 }}
            >
              <Option value="7d">7天</Option>
              <Option value="30d">30天</Option>
              <Option value="90d">90天</Option>
            </Select>
            <Button icon={<Zap />} size="small">
              刷新预测
            </Button>
          </Space>
        }
      >
        <Row gutter={16}>
          <Col span={6}>
            <Statistic
              title="工单量预测"
              value={metrics?.ticketVolume.predicted}
              prefix={getTrendIcon(metrics?.ticketVolume.trend || "stable")}
              suffix={
                <div className="text-xs">
                  <div>置信度: {metrics?.ticketVolume.confidence}%</div>
                  <div className="text-gray-500">
                    当前: {metrics?.ticketVolume.current}
                  </div>
                </div>
              }
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="资源需求预测"
              value={metrics?.resourceDemand.predicted}
              prefix={<Users />}
              suffix={
                <div className="text-xs">
                  <div>缺口: {metrics?.resourceDemand.shortage}</div>
                  <div className="text-orange-500">需要增加</div>
                </div>
              }
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="SLA违约风险"
              value={`${metrics?.slaBreachRisk.probability}%`}
              prefix={<AlertTriangle />}
              styles={{ content: {
                color: getRiskColor(metrics?.slaBreachRisk.riskLevel || "low"),
              }}
              suffix={
                <div className="text-xs">
                  <div>影响工单: {metrics?.slaBreachRisk.affectedTickets}</div>
                </div>
              }
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="预测准确率"
              value="87.5%"
              prefix={<Target />}
              suffix={
                <div className="text-xs text-green-500">
                  <TrendingUp className="inline mr-1" />
                  持续提升
                </div>
              }
            />
          </Col>
        </Row>
      </Card>

      {/* 工单趋势预测 */}
      <Card
        title={
          <Space>
            <BarChart3 className="text-blue-600" />
            <span>工单趋势预测</span>
            <Tag color="blue">AI预测</Tag>
          </Space>
        }
      >
        <div className="mb-4">
          <Text type="secondary">
            基于历史数据和AI算法预测未来工单量趋势，置信度:{" "}
            {Math.min(...trendData.map((d) => d.confidence)) * 100}%
          </Text>
        </div>
        <div className="h-64 flex items-center justify-center bg-gray-50 rounded border">
          <div className="text-center text-gray-500">
            <BarChart3 className="mx-auto mb-2 text-4xl" />
            <div>工单趋势预测图表</div>
            <div className="text-sm">实际值 vs 预测值对比</div>
          </div>
        </div>
      </Card>

      {/* 资源需求预测 */}
      <Card
        title={
          <Space>
            <Users className="text-green-600" />
            <span>资源需求预测</span>
            <Tag color="green">智能推荐</Tag>
          </Space>
        }
      >
        <div className="mb-4">
          <Alert
            message={metrics?.resourceDemand.recommendation}
            type="info"
            showIcon
            icon={<Brain />}
          />
        </div>
        <div className="h-64 flex items-center justify-center bg-gray-50 rounded border">
          <div className="text-center text-gray-500">
            <BarChart3 className="mx-auto mb-2 text-4xl" />
            <div>资源需求预测图表</div>
            <div className="text-sm">当前需求 vs 预测需求对比</div>
          </div>
        </div>

        <div className="mt-4">
          <Text strong>详细资源分析</Text>
          <div className="mt-2 space-y-2">
            {resourceData.map((item, index) => (
              <div
                key={index}
                className="flex items-center justify-between p-2 border rounded"
              >
                <div className="flex-1">
                  <Text strong>{item.role}</Text>
                  <div className="text-sm text-gray-500">
                    当前: {item.current} | 需要: {item.required}
                  </div>
                </div>
                <div className="text-right">
                  {item.gap > 0 ? (
                    <Tag color="red">缺口: {item.gap}</Tag>
                  ) : (
                    <Tag color="green">充足</Tag>
                  )}
                  <div className="mt-1">
                    <Progress
                      percent={item.utilization}
                      size="small"
                      status={item.utilization >= 90 ? "exception" : "normal"}
                      strokeColor={getUtilizationColor(item.utilization)}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </Card>

      {/* 季节性模式分析 */}
      <Card
        title={
          <Space>
            <Calendar className="text-orange-600" />
            <span>季节性模式分析</span>
            <Tag color="orange">AI识别</Tag>
          </Space>
        }
      >
        <Row gutter={16}>
          <Col span={12}>
            <div className="p-4 bg-red-50 border border-red-200 rounded">
              <Title level={5} className="text-red-600 mb-3">
                <TrendingUp className="mr-2" />
                高峰期
              </Title>
              <ul className="space-y-1">
                {metrics?.seasonalPatterns.peakPeriods.map((period, index) => (
                  <li key={index} className="text-red-700">
                    • {period}
                  </li>
                ))}
              </ul>
            </div>
          </Col>
          <Col span={12}>
            <div className="p-4 bg-green-50 border border-green-200 rounded">
              <Title level={5} className="text-green-600 mb-3">
                <TrendingDown className="mr-2" />
                低峰期
              </Title>
              <ul className="space-y-1">
                {metrics?.seasonalPatterns.lowPeriods.map((period, index) => (
                  <li key={index} className="text-green-700">
                    • {period}
                  </li>
                ))}
              </ul>
            </div>
          </Col>
        </Row>

        <div className="mt-4">
          <Text strong>AI优化建议</Text>
          <div className="mt-2 space-y-2">
            {metrics?.seasonalPatterns.recommendations.map((rec, index) => (
              <Alert
                key={index}
                message={rec}
                type="info"
                showIcon
                icon={<Brain />}
                className="mb-2"
              />
            ))}
          </div>
        </div>
      </Card>

      {/* 风险预警 */}
      <Card
        title={
          <Space>
            <AlertTriangle className="text-red-600" />
            <span>SLA违约风险预警</span>
            <Badge count={metrics?.slaBreachRisk.affectedTickets} />
          </Space>
        }
      >
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <Text strong>风险等级: </Text>
              <Tag
                color={getRiskColor(metrics?.slaBreachRisk.riskLevel || "low")}
              >
                {metrics?.slaBreachRisk.riskLevel?.toUpperCase()}
              </Tag>
            </div>
            <div className="text-right">
              <Text type="secondary">违约概率</Text>
              <div className="text-2xl font-bold text-red-600">
                {metrics?.slaBreachRisk.probability}%
              </div>
            </div>
          </div>

          <Progress
            percent={metrics?.slaBreachRisk.probability || 0}
            status="exception"
            strokeColor={{
              "0%": "#ff4d4f",
              "100%": "#ff7875",
            }}
          />

          <Alert
            message={metrics?.slaBreachRisk.mitigation}
            type="warning"
            showIcon
            icon={<Brain />}
            action={
              <Button size="small" type="link">
                查看详情
              </Button>
            }
          />
        </div>
      </Card>
    </div>
  );
};
