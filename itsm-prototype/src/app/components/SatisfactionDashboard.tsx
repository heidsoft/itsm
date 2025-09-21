"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Statistic,
  Progress,
  Space,
  Typography,
  Row,
  Col,
  Table,
  Tag,
  Button,
  Select,
  DatePicker,
  Alert,
  Badge,
  Tooltip,
  Rate,
  Avatar,
  Divider,
} from "antd";
import {
  Smile,
  Meh,
  Frown,
  TrendingUp,
  TrendingDown,
  Star,
  MessageSquare,
  Users,
  Clock,
  Filter,
  Download,
  Eye,
  BarChart3,
  Heart,
  ThumbsUp,
  ThumbsDown,
  AlertTriangle,
  CheckCircle,
} from "lucide-react";

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

interface SatisfactionData {
  overallScore: number;
  totalResponses: number;
  responseRate: number;
  trend: "up" | "down" | "stable";
  categoryScores: {
    category: string;
    score: number;
    count: number;
    trend: "up" | "down" | "stable";
  }[];
  monthlyTrend: {
    month: string;
    score: number;
    responses: number;
  }[];
  agentPerformance: {
    agent: string;
    avatar: string;
    score: number;
    tickets: number;
    responseTime: number;
    satisfaction: number;
  }[];
  recentFeedback: {
    id: number;
    ticketNumber: string;
    title: string;
    score: number;
    comment: string;
    agent: string;
    category: string;
    date: string;
    tags: string[];
  }[];
}

export const SatisfactionDashboard: React.FC = () => {
  const [satisfactionData, setSatisfactionData] =
    useState<SatisfactionData | null>(null);
  const [timeRange, setTimeRange] = useState<[string, string]>(["30d", "7d"]);
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadSatisfactionData();
  }, [timeRange, categoryFilter]);

  const loadSatisfactionData = async () => {
    try {
      // 模拟API调用
      const mockData: SatisfactionData = {
        overallScore: 4.2,
        totalResponses: 1247,
        responseRate: 78.5,
        trend: "up",
        categoryScores: [
          { category: "系统问题", score: 4.1, count: 456, trend: "up" },
          { category: "网络问题", score: 4.3, count: 234, trend: "stable" },
          { category: "数据库问题", score: 4.0, count: 189, trend: "down" },
          { category: "硬件问题", score: 4.5, count: 156, trend: "up" },
          { category: "软件问题", score: 4.2, count: 212, trend: "up" },
        ],
        monthlyTrend: [
          { month: "2024-01", score: 4.1, responses: 156 },
          { month: "2024-02", score: 4.2, responses: 189 },
          { month: "2024-03", score: 4.3, responses: 234 },
          { month: "2024-04", score: 4.2, responses: 267 },
          { month: "2024-05", score: 4.4, responses: 298 },
          { month: "2024-06", score: 4.2, responses: 1247 },
        ],
        agentPerformance: [
          {
            agent: "张三",
            avatar: "张",
            score: 4.5,
            tickets: 156,
            responseTime: 2.3,
            satisfaction: 92.3,
          },
          {
            agent: "李四",
            avatar: "李",
            score: 4.2,
            tickets: 134,
            responseTime: 3.1,
            satisfaction: 87.6,
          },
          {
            agent: "王五",
            avatar: "王",
            score: 4.3,
            tickets: 189,
            responseTime: 2.8,
            satisfaction: 89.4,
          },
          {
            agent: "赵六",
            avatar: "赵",
            score: 4.1,
            tickets: 98,
            responseTime: 4.2,
            satisfaction: 84.7,
          },
        ],
        recentFeedback: [
          {
            id: 1,
            ticketNumber: "T-2024-001",
            title: "数据库连接超时问题",
            score: 5,
            comment: "处理速度很快，技术人员专业，问题得到完美解决！",
            agent: "张三",
            category: "数据库问题",
            date: "2024-01-15",
            tags: ["专业", "快速", "满意"],
          },
          {
            id: 2,
            ticketNumber: "T-2024-002",
            title: "系统登录异常",
            score: 4,
            comment: "响应及时，解决方案有效，但处理时间稍长",
            agent: "李四",
            category: "系统问题",
            date: "2024-01-14",
            tags: ["及时", "有效"],
          },
          {
            id: 3,
            ticketNumber: "T-2024-003",
            title: "网络连接问题",
            score: 3,
            comment: "问题解决了，但沟通不够清晰，希望改进",
            agent: "王五",
            category: "网络问题",
            date: "2024-01-13",
            tags: ["已解决"],
          },
        ],
      };

      setSatisfactionData(mockData);
    } catch (error) {
      console.error("加载满意度数据失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const getScoreColor = (score: number): string => {
    if (score >= 4.5) return "#52c41a";
    if (score >= 4.0) return "#1890ff";
    if (score >= 3.5) return "#faad14";
    return "#f5222d";
  };

  const getScoreLevel = (score: number): string => {
    if (score >= 4.5) return "非常满意";
    if (score >= 4.0) return "满意";
    if (score >= 3.5) return "一般";
    if (score >= 3.0) return "不满意";
    return "非常不满意";
  };

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case "up":
        return <TrendingUp className="text-green-500" />;
      case "down":
        return <TrendingDown className="text-red-500" />;
      default:
        return <TrendingUp className="text-blue-500" />;
    }
  };

  const getTrendColor = (trend: string): string => {
    switch (trend) {
      case "up":
        return "green";
      case "down":
        return "red";
      default:
        return "blue";
    }
  };

  const agentColumns = [
    {
      title: "处理人",
      key: "agent",
      render: (record: any) => (
        <Space>
          <Avatar size="small" style={{ backgroundColor: "#1890ff" }}>
            {record.avatar}
          </Avatar>
          <Text strong>{record.agent}</Text>
        </Space>
      ),
    },
    {
      title: "满意度评分",
      dataIndex: "score",
      key: "score",
      render: (score: number) => (
        <Space>
          <Rate disabled defaultValue={score} style={{ fontSize: 14 }} />
          <Text strong style={{ color: getScoreColor(score) }}>
            {score.toFixed(1)}
          </Text>
        </Space>
      ),
    },
    {
      title: "处理工单数",
      dataIndex: "tickets",
      key: "tickets",
      render: (tickets: number) => <Text>{tickets}</Text>,
    },
    {
      title: "平均响应时间",
      dataIndex: "responseTime",
      key: "responseTime",
      render: (time: number) => <Text>{time} 小时</Text>,
    },
    {
      title: "满意度率",
      dataIndex: "satisfaction",
      key: "satisfaction",
      render: (satisfaction: number) => (
        <Progress
          percent={satisfaction}
          size="small"
          status={
            satisfaction >= 90
              ? "success"
              : satisfaction >= 80
              ? "normal"
              : "exception"
          }
        />
      ),
    },
  ];

  const feedbackColumns = [
    {
      title: "工单号",
      dataIndex: "ticketNumber",
      key: "ticketNumber",
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      render: (text: string) => <Text>{text}</Text>,
    },
    {
      title: "评分",
      dataIndex: "score",
      key: "score",
      render: (score: number) => (
        <Space>
          <Rate disabled defaultValue={score} style={{ fontSize: 12 }} />
          <Tag color={getScoreColor(score) >= 4.0 ? "green" : "orange"}>
            {getScoreLevel(score)}
          </Tag>
        </Space>
      ),
    },
    {
      title: "处理人",
      dataIndex: "agent",
      key: "agent",
      render: (text: string) => <Text>{text}</Text>,
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      render: (text: string) => <Text>{text}</Text>,
    },
    {
      title: "评价时间",
      dataIndex: "date",
      key: "date",
      render: (text: string) => <Text>{text}</Text>,
    },
    {
      title: "操作",
      key: "action",
      render: (record: any) => (
        <Button size="small" icon={<Eye />}>
          查看详情
        </Button>
      ),
    },
  ];

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
      {/* 满意度概览 */}
      <Card
        title={
          <Space>
            <Heart className="text-red-600" />
            <span>满意度概览</span>
            <Badge count={satisfactionData?.totalResponses} />
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
            <Button icon={<Download />} size="small">
              导出报告
            </Button>
          </Space>
        }
      >
        <Row gutter={16}>
          <Col span={6}>
            <Statistic
              title="整体满意度"
              value={satisfactionData?.overallScore}
              precision={1}
              prefix={
                <Star
                  style={{
                    color: getScoreColor(satisfactionData?.overallScore || 0),
                  }}
                />
              }
              suffix={
                <div className="text-xs">
                  <div className="text-gray-500">
                    {getScoreLevel(satisfactionData?.overallScore || 0)}
                  </div>
                  <div className="flex items-center">
                    {getTrendIcon(satisfactionData?.trend || "stable")}
                    <span className="ml-1 text-xs">较上月</span>
                  </div>
                </div>
              }
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="评价总数"
              value={satisfactionData?.totalResponses}
              prefix={<MessageSquare />}
              suffix={
                <div className="text-xs text-gray-500">
                  响应率: {satisfactionData?.responseRate}%
                </div>
              }
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="非常满意"
              value={Math.round(
                ((satisfactionData?.recentFeedback.filter((f) => f.score >= 4.5)
                  .length || 0) /
                  (satisfactionData?.recentFeedback.length || 1)) *
                  100
              )}
              prefix={<Smile className="text-green-500" />}
              suffix="%"
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="不满意"
              value={Math.round(
                ((satisfactionData?.recentFeedback.filter((f) => f.score < 3.5)
                  .length || 0) /
                  (satisfactionData?.recentFeedback.length || 1)) *
                  100
              )}
              prefix={<Frown className="text-red-500" />}
              suffix="%"
            />
          </Col>
        </Row>

        <div className="mt-6">
          <div className="flex items-center justify-between mb-2">
            <Text>满意度分布</Text>
            <Text strong>{satisfactionData?.overallScore.toFixed(1)}/5.0</Text>
          </div>
          <Progress
            percent={((satisfactionData?.overallScore || 0) / 5) * 100}
            status="success"
            strokeColor={{
              "0%": "#f5222d",
              "50%": "#faad14",
              "100%": "#52c41a",
            }}
          />
        </div>
      </Card>

      {/* 分类满意度 */}
      <Card
        title={
          <Space>
            <BarChart3 className="text-blue-600" />
            <span>分类满意度分析</span>
          </Space>
        }
      >
        <div className="space-y-4">
          {satisfactionData?.categoryScores.map((category, index) => (
            <div
              key={index}
              className="flex items-center justify-between p-3 border rounded hover:bg-gray-50"
            >
              <div className="flex-1">
                <div className="flex items-center mb-2">
                  <Text strong className="mr-3">
                    {category.category}
                  </Text>
                  <Tag color={getTrendColor(category.trend)}>
                    {getTrendIcon(category.trend)}
                    {category.trend === "up"
                      ? "上升"
                      : category.trend === "down"
                      ? "下降"
                      : "稳定"}
                  </Tag>
                </div>
                <div className="flex items-center text-sm text-gray-500">
                  <span className="mr-4">评价数: {category.count}</span>
                  <span>评分: {category.score.toFixed(1)}/5.0</span>
                </div>
              </div>
              <div className="text-right">
                <Progress
                  percent={(category.score / 5) * 100}
                  size="small"
                  status={category.score >= 4.0 ? "success" : "normal"}
                  strokeColor={getScoreColor(category.score)}
                />
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* 处理人绩效 */}
      <Card
        title={
          <Space>
            <Users className="text-green-600" />
            <span>处理人绩效分析</span>
          </Space>
        }
      >
        <Table
          columns={agentColumns}
          dataSource={satisfactionData?.agentPerformance}
          rowKey="agent"
          pagination={false}
          size="small"
        />
      </Card>

      {/* 最近评价 */}
      <Card
        title={
          <Space>
            <MessageSquare className="text-purple-600" />
            <span>最近评价反馈</span>
            <Badge count={satisfactionData?.recentFeedback.length} />
          </Space>
        }
      >
        <Table
          columns={feedbackColumns}
          dataSource={satisfactionData?.recentFeedback}
          rowKey="id"
          pagination={{ pageSize: 5 }}
          expandable={{
            expandedRowRender: (record) => (
              <div className="p-4 bg-gray-50 rounded">
                <div className="mb-3">
                  <Text strong>评价内容:</Text>
                  <div className="mt-1 p-2 bg-white rounded border">
                    {record.comment}
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  {record.tags.map((tag, index) => (
                    <Tag key={index} color="blue">
                      {tag}
                    </Tag>
                  ))}
                </div>
              </div>
            ),
          }}
        />
      </Card>

      {/* 改进建议 */}
      <Card
        title={
          <Space>
            <ThumbsUp className="text-orange-600" />
            <span>改进建议</span>
          </Space>
        }
      >
        <Alert
          message="基于满意度分析的建议"
          description="根据用户反馈和满意度数据，建议重点关注数据库问题处理效率和沟通清晰度改进"
          type="info"
          showIcon
          icon={<ThumbsUp />}
          className="mb-4"
        />

        <div className="space-y-3">
          <div className="p-3 border border-orange-200 rounded bg-orange-50">
            <div className="flex items-center mb-2">
              <AlertTriangle className="text-orange-500 mr-2" />
              <Text strong>需要改进的方面</Text>
            </div>
            <ul className="space-y-1 text-sm">
              <li>• 数据库问题处理时间较长，建议优化流程</li>
              <li>• 技术沟通不够清晰，建议加强培训</li>
              <li>• 响应时间在高峰期有所延迟</li>
            </ul>
          </div>

          <div className="p-3 border border-green-200 rounded bg-green-50">
            <div className="flex items-center mb-2">
              <CheckCircle className="text-green-500 mr-2" />
              <Text strong>表现优秀的方面</Text>
            </div>
            <ul className="space-y-1 text-sm">
              <li>• 硬件问题处理满意度最高</li>
              <li>• 系统问题解决效率持续提升</li>
              <li>• 整体服务态度获得好评</li>
            </ul>
          </div>
        </div>
      </Card>
    </div>
  );
};
