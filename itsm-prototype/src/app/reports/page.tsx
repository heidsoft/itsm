"use client";

import React from "react";
import Link from "next/link";
import { Card, Row, Col, Typography, Space, theme } from "antd";
import {
  FileText,
  AlertTriangle,
  GitMerge,
  Target,
  Search,
  BookOpen,
  BarChart3,
} from "lucide-react";

const { Title, Text } = Typography;

interface ReportCardProps {
  title: string;
  description: string;
  icon: React.ComponentType<{ size?: number; style?: React.CSSProperties }>;
  link: string;
}

const ReportCard: React.FC<ReportCardProps> = ({
  title,
  description,
  icon: Icon,
  link,
}) => {
  const { token } = theme.useToken();

  return (
    <Link href={link} style={{ textDecoration: "none" }}>
      <Card
        hoverable
        style={{ height: "100%" }}
        styles={{
          body: {
            padding: token.paddingLG,
          },
        }}
      >
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <div
            style={{
              display: "inline-flex",
              alignItems: "center",
              justifyContent: "center",
              width: 48,
              height: 48,
              borderRadius: token.borderRadiusLG,
              backgroundColor: "#e6f4ff",
              marginBottom: token.marginSM,
            }}
          >
            <Icon size={24} style={{ color: token.colorPrimary }} />
          </div>

          <div>
            <Title
              level={4}
              style={{
                margin: 0,
                marginBottom: token.marginXS,
                color: token.colorText,
              }}
            >
              {title}
            </Title>
            <Text
              type="secondary"
              style={{
                fontSize: token.fontSizeSM,
                lineHeight: 1.5,
              }}
            >
              {description}
            </Text>
          </div>
        </Space>
      </Card>
    </Link>
  );
};

const ReportsPage = () => {
  const { token } = theme.useToken();

  return (
    <div style={{ padding: token.paddingLG }}>
      {/* 页面标题 */}
      <div style={{ marginBottom: token.marginXL }}>
        <Title level={2} style={{ margin: 0, marginBottom: token.marginXS }}>
          <Space>
            <BarChart3 style={{ color: token.colorPrimary }} />
            报告与分析
          </Space>
        </Title>
        <Text type="secondary" style={{ fontSize: token.fontSize }}>
          获取IT服务绩效的深度洞察
        </Text>
      </div>

      {/* 报告卡片网格 */}
      <Row gutter={[24, 24]}>
        <Col xs={24} sm={12} lg={8}>
          <ReportCard
            title="事件趋势分析"
            description="分析事件发生频率、类型、解决时间等趋势，识别潜在问题。"
            icon={AlertTriangle}
            link="/reports/incident-trends"
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <ReportCard
            title="变更成功率报告"
            description="评估变更的成功率、失败原因和对服务的影响。"
            icon={GitMerge}
            link="/reports/change-success"
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <ReportCard
            title="SLA绩效报告"
            description="监控服务级别协议（SLA）的达成情况，识别违约风险。"
            icon={Target}
            link="/reports/sla-performance"
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <ReportCard
            title="问题管理效率报告"
            description="分析问题解决周期、根本原因分布和已知错误管理情况。"
            icon={Search}
            link="/reports/problem-efficiency"
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <ReportCard
            title="服务目录使用报告"
            description="洞察服务请求的热点、用户偏好和交付效率。"
            icon={BookOpen}
            link="/reports/service-catalog-usage"
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <ReportCard
            title="CMDB数据质量报告"
            description="评估配置项数据的完整性、准确性和关联性。"
            icon={FileText}
            link="/reports/cmdb-quality"
          />
        </Col>
      </Row>
    </div>
  );
};

export default ReportsPage;
