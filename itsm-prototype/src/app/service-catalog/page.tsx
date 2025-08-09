"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import {
  Card,
  Row,
  Col,
  Input,
  Select,
  Tag,
  Button,
  Statistic,
  Empty,
  Typography,
  Space,
  Rate,
} from "antd";
import {
  HardDrive,
  UserCog,
  ShieldCheck,
  Search,
  Clock,
  ArrowRight,
} from "lucide-react";
import { ServiceCatalogApi, ServiceCatalog } from "../lib/service-catalog-api";

const { Search: SearchInput } = Input;
const { Option } = Select;
const { Title, Text } = Typography;

// 图标映射
const categoryIcons = {
  云资源服务: HardDrive,
  账号与权限: UserCog,
  安全服务: ShieldCheck,
};

const ServiceItemCard = ({ catalog }: { catalog: ServiceCatalog }) => {
  const IconComponent = categoryIcons[catalog.category] || HardDrive;

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case "高":
        return "red";
      case "中":
        return "orange";
      case "低":
        return "green";
      default:
        return "default";
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "可用":
        return "success";
      case "维护中":
        return "warning";
      case "不可用":
        return "error";
      default:
        return "default";
    }
  };

  return (
    <Card
      hoverable
      style={{ height: "100%" }}
      actions={[
        <Link key="request" href={`/service-catalog/request/${catalog.id}`}>
          <Button type="primary" size="small" icon={<ArrowRight size={14} />}>
            申请服务
          </Button>
        </Link>,
      ]}
    >
      <div style={{ marginBottom: 16 }}>
        <Space align="center">
          <div style={{ color: "#1890ff" }}>
            <IconComponent size={20} />
          </div>
          <Title level={4} style={{ margin: 0 }}>
            {catalog.name}
          </Title>
        </Space>
      </div>

      <div style={{ marginBottom: 16 }}>
        <Text type="secondary">{catalog.description}</Text>
      </div>

      <div style={{ marginBottom: 16 }}>
        <Space wrap>
          <Tag color="blue">{catalog.category}</Tag>
          <Tag color={getStatusColor(catalog.status)}>{catalog.status}</Tag>
          <Tag color={getPriorityColor(catalog.priority)}>
            {catalog.priority}
          </Tag>
        </Space>
      </div>

      <Row gutter={16}>
        <Col span={12}>
          <div style={{ textAlign: "center" }}>
            <div style={{ color: "#faad14", marginBottom: 4 }}>
              <Clock size={16} />
            </div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {catalog.sla_time}
            </Text>
          </div>
        </Col>
        <Col span={12}>
          <div style={{ textAlign: "center" }}>
            <Rate
              disabled
              defaultValue={4.5}
              allowHalf
              style={{ fontSize: 12 }}
            />
            <div>
              <Text type="secondary" style={{ fontSize: 12 }}>
                满意度 4.5
              </Text>
            </div>
          </div>
        </Col>
      </Row>
    </Card>
  );
};

const ServiceCatalogPage = () => {
  const [catalogs, setCatalogs] = useState<ServiceCatalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedCategory, setSelectedCategory] = useState("全部");
  const [selectedStatus, setSelectedStatus] = useState("全部");

  useEffect(() => {
    const fetchCatalogs = async () => {
      try {
        setLoading(true);
        const data = await ServiceCatalogApi.getCatalogs();
        setCatalogs(data);
      } catch (error) {
        console.error("Failed to fetch service catalogs:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchCatalogs();
  }, []);

  const filteredCatalogs = catalogs.filter((catalog) => {
    const matchesSearch =
      catalog.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      catalog.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory =
      selectedCategory === "全部" || catalog.category === selectedCategory;
    const matchesStatus =
      selectedStatus === "全部" || catalog.status === selectedStatus;

    return matchesSearch && matchesCategory && matchesStatus;
  });

  const categories = [
    "全部",
    ...Array.from(new Set(catalogs.map((c) => c.category))),
  ];
  const statuses = [
    "全部",
    ...Array.from(new Set(catalogs.map((c) => c.status))),
  ];

  const stats = {
    total: catalogs.length,
    available: catalogs.filter((c) => c.status === "可用").length,
    maintenance: catalogs.filter((c) => c.status === "维护中").length,
    high_priority: catalogs.filter((c) => c.priority === "高").length,
  };

  return (
    <div style={{ padding: 24 }}>
      {/* 页面头部 */}
      <div style={{ marginBottom: 24 }}>
        <Title level={2} style={{ margin: 0 }}>
          服务目录
        </Title>
        <Text type="secondary">
          浏览和申请IT服务，简化业务流程，提高工作效率
        </Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总服务数"
              value={stats.total}
              prefix={<HardDrive size={16} style={{ color: "#1890ff" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="可用服务"
              value={stats.available}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="维护中"
              value={stats.maintenance}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="高优先级"
              value={stats.high_priority}
              valueStyle={{ color: "#f5222d" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索和筛选 */}
      <Card style={{ marginBottom: 24 }}>
        <Row gutter={16} align="middle">
          <Col span={8}>
            <SearchInput
              placeholder="搜索服务名称或描述"
              allowClear
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              prefix={<Search size={16} />}
            />
          </Col>
          <Col span={4}>
            <Select
              value={selectedCategory}
              onChange={setSelectedCategory}
              style={{ width: "100%" }}
              placeholder="选择分类"
            >
              {categories.map((category) => (
                <Option key={category} value={category}>
                  {category}
                </Option>
              ))}
            </Select>
          </Col>
          <Col span={4}>
            <Select
              value={selectedStatus}
              onChange={setSelectedStatus}
              style={{ width: "100%" }}
              placeholder="选择状态"
            >
              {statuses.map((status) => (
                <Option key={status} value={status}>
                  {status}
                </Option>
              ))}
            </Select>
          </Col>
        </Row>
      </Card>

      {/* 服务卡片网格 */}
      {loading ? (
        <div style={{ textAlign: "center", padding: 48 }}>加载中...</div>
      ) : filteredCatalogs.length > 0 ? (
        <Row gutter={[16, 16]}>
          {filteredCatalogs.map((catalog) => (
            <Col key={catalog.id} xs={24} sm={12} md={8} lg={6}>
              <ServiceItemCard catalog={catalog} />
            </Col>
          ))}
        </Row>
      ) : (
        <Card>
          <Empty
            description="未找到匹配的服务项"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </Card>
      )}
    </div>
  );
};

export default ServiceCatalogPage;
