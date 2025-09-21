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
  Rate,
  Modal,
  Form,
  message,
} from "antd";
import {
  HardDrive,
  UserCog,
  ShieldCheck,
  Search,
  Clock,
  ArrowRight,
  PlusCircle,
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

  return (
    <Card
      style={{ height: "100%" }}
      styles={{
        body: {
          padding: 24,
        },
      }}
    >
      <div
        style={{ display: "flex", alignItems: "flex-start", marginBottom: 16 }}
      >
        <div
          style={{
            width: 48,
            height: 48,
            borderRadius: 8,
            backgroundColor: "#e6f7ff",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            marginRight: 16,
          }}
        >
          <IconComponent size={24} style={{ color: "#1890ff" }} />
        </div>
        <div style={{ flex: 1 }}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "flex-start",
            }}
          >
            <Title level={4} style={{ margin: 0, fontSize: 16 }}>
              {catalog.name}
            </Title>
            <Tag
              color={getPriorityColor(catalog.priority)}
              style={{ margin: 0 }}
            >
              {catalog.priority}
            </Tag>
          </div>
          <Text
            type="secondary"
            style={{ fontSize: 12, marginTop: 4, display: "block" }}
          >
            {catalog.category}
          </Text>
        </div>
      </div>

      <Text style={{ marginBottom: 16, display: "block", minHeight: 40 }}>
        {catalog.description}
      </Text>

      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginTop: 16,
        }}
      >
        <div style={{ display: "flex", alignItems: "center" }}>
          <Clock size={14} style={{ marginRight: 4, color: "#666" }} />
          <Text type="secondary" style={{ fontSize: 12 }}>
            {catalog.sla_time || catalog.estimated_time}
          </Text>
        </div>
        <div>
          <Rate
            disabled
            defaultValue={catalog.rating || 4.5}
            count={5}
            style={{ fontSize: 12 }}
          />
        </div>
      </div>

      <div
        style={{
          marginTop: 16,
          paddingTop: 16,
          borderTop: "1px solid #f0f0f0",
        }}
      >
        <Link
          href={`/service-catalog/request/${catalog.id}`}
          style={{ textDecoration: "none" }}
        >
          <Button type="primary" block icon={<ArrowRight size={16} />}>
            申请服务
          </Button>
        </Link>
      </div>
    </Card>
  );
};

export default function ServiceCatalogPage() {
  const [catalogs, setCatalogs] = useState<ServiceCatalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchText, setSearchText] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("");
  const [priorityFilter, setPriorityFilter] = useState("");

  // 新增状态
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [createForm] = Form.useForm();

  // 统计数据
  const [stats, setStats] = useState({
    total: 0,
    cloudServices: 0,
    accountServices: 0,
    securityServices: 0,
  });

  useEffect(() => {
    loadServiceCatalogs();
    loadStats();
  }, []);

  const loadServiceCatalogs = async () => {
    try {
      const data = await ServiceCatalogApi.getCatalogs();
      setCatalogs(data);
    } catch (error) {
      console.error("加载服务目录失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      // 模拟统计数据
      setStats({
        total: catalogs.length,
        cloudServices: catalogs.filter((c) => c.category === "云资源服务")
          .length,
        accountServices: catalogs.filter((c) => c.category === "账号与权限")
          .length,
        securityServices: catalogs.filter((c) => c.category === "安全服务")
          .length,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  };

  useEffect(() => {
    // 更新统计数据
    setStats({
      total: catalogs.length,
      cloudServices: catalogs.filter((c) => c.category === "云资源服务").length,
      accountServices: catalogs.filter((c) => c.category === "账号与权限")
        .length,
      securityServices: catalogs.filter((c) => c.category === "安全服务")
        .length,
    });
  }, [catalogs]);

  // 渲染统计卡片
  const renderStatsCards = () => (
    <div style={{ marginBottom: 24 }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title="服务总数"
              value={stats.total}
              prefix={<HardDrive size={16} style={{ color: "#1890ff" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="云资源服务"
              value={stats.cloudServices}
              valueStyle={{ color: "#1890ff" }}
              prefix={<HardDrive size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="账号与权限"
              value={stats.accountServices}
              valueStyle={{ color: "#722ed1" }}
              prefix={<UserCog size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="安全服务"
              value={stats.securityServices}
              valueStyle={{ color: "#52c41a" }}
              prefix={<ShieldCheck size={16} />}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );

  const handleCreateService = () => {
    setCreateModalVisible(true);
  };

  const handleCreateServiceConfirm = async () => {
    try {
      const values = await createForm.validateFields();
      // 调用API创建服务目录
      await ServiceCatalogApi.createServiceCatalog({
        name: values.name,
        category: values.category,
        description: values.description,
        delivery_time: values.deliveryTime,
        status: values.status,
      });

      message.success("服务创建成功");
      setCreateModalVisible(false);
      createForm.resetFields();
      // 重新加载数据
      loadServiceCatalogs();
    } catch (error) {
      console.error("创建服务失败:", error);
      message.error("创建服务失败");
    }
  };

  // 渲染筛选器
  const renderFilters = () => (
    <Card style={{ marginBottom: 24 }}>
      <Row gutter={20} align="middle">
        <Col xs={24} sm={12} md={8}>
          <SearchInput
            placeholder="搜索服务名称或描述..."
            allowClear
            onSearch={(value) => setSearchText(value)}
            size="large"
            enterButton
          />
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder="分类筛选"
            size="large"
            allowClear
            value={categoryFilter}
            onChange={(value) => setCategoryFilter(value)}
            style={{ width: "100%" }}
          >
            <Option value="云资源服务">云资源服务</Option>
            <Option value="账号与权限">账号与权限</Option>
            <Option value="安全服务">安全服务</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder="优先级筛选"
            size="large"
            allowClear
            value={priorityFilter}
            onChange={(value) => setPriorityFilter(value)}
            style={{ width: "100%" }}
          >
            <Option value="高">高</Option>
            <Option value="中">中</Option>
            <Option value="低">低</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button
            icon={<Search size={20} />}
            onClick={() => {}}
            size="large"
            style={{ width: "100%" }}
          >
            刷新
          </Button>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button
            type="primary"
            icon={<PlusCircle size={20} />}
            size="large"
            style={{ width: "100%" }}
            onClick={handleCreateService}
          >
            新建服务
          </Button>
        </Col>
      </Row>
    </Card>
  );

  // 过滤服务目录
  const filteredCatalogs = catalogs.filter((catalog) => {
    const matchesSearch = searchText
      ? catalog.name.toLowerCase().includes(searchText.toLowerCase()) ||
        catalog.description.toLowerCase().includes(searchText.toLowerCase())
      : true;
    const matchesCategory = categoryFilter
      ? catalog.category === categoryFilter
      : true;
    const matchesPriority = priorityFilter
      ? catalog.priority === priorityFilter
      : true;
    return matchesSearch && matchesCategory && matchesPriority;
  });

  return (
    <div>
      {renderStatsCards()}
      {renderFilters()}

      {filteredCatalogs.length === 0 && !loading ? (
        <Card>
          <Empty description="暂无匹配的服务目录" />
        </Card>
      ) : (
        <Row gutter={[24, 24]}>
          {filteredCatalogs.map((catalog) => (
            <Col key={catalog.id} xs={24} sm={12} md={8} lg={6}>
              <ServiceItemCard catalog={catalog} />
            </Col>
          ))}
        </Row>
      )}

      {/* 创建服务模态框 */}
      <Modal
        title={
          <div style={{ display: "flex", alignItems: "center" }}>
            <div style={{ width: 32, height: 32, backgroundColor: "#1890ff", borderRadius: 6, display: "flex", alignItems: "center", justifyContent: "center", marginRight: 12 }}>
              <PlusCircle size={18} style={{ color: "#fff" }} />
            </div>
            <span style={{ fontSize: "large", fontWeight: "medium" }}>新建服务</span>
          </div>
        }
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          createForm.resetFields();
        }}
        onOk={handleCreateServiceConfirm}
        width={600}
      >
        <Form form={createForm} layout="vertical" style={{ marginTop: 20 }}>
          <Form.Item
            label="服务名称"
            name="name"
            rules={[{ required: true, message: "请输入服务名称" }]}
          >
            <Input placeholder="请输入服务名称" />
          </Form.Item>

          <Form.Item
            label="服务分类"
            name="category"
            rules={[{ required: true, message: "请选择服务分类" }]}
          >
            <Select placeholder="请选择服务分类">
              <Option value="云资源服务">云资源服务</Option>
              <Option value="账号与权限">账号与权限</Option>
              <Option value="安全服务">安全服务</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="交付时间"
            name="deliveryTime"
            rules={[{ required: true, message: "请输入交付时间" }]}
          >
            <Input placeholder="例如：1-3个工作日" />
          </Form.Item>

          <Form.Item
            label="状态"
            name="status"
            rules={[{ required: true, message: "请选择状态" }]}
          >
            <Select placeholder="请选择状态">
              <Option value="enabled">启用</Option>
              <Option value="disabled">禁用</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="服务描述"
            name="description"
          >
            <Input.TextArea
              placeholder="请输入服务描述"
              rows={4}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
