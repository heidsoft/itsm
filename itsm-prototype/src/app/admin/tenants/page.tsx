"use client";

import {Building2, AlertCircle} from 'lucide-react';

import React, { useState, useEffect } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Space,
  Typography,
  Modal,
  Form,
  Row,
  Col,
  Statistic,
  Badge,
  Tooltip,
  Popconfirm,
  message,
  Divider,
  Tag,
} from "antd";
const { Title, Text } = Typography;
const { Option } = Select;

// 租户状态配置
const TENANT_STATUS = {
  active: {
    label: "活跃",
    color: "success",
    icon: CheckCircle,
  },
  suspended: {
    label: "暂停",
    color: "warning",
    icon: AlertCircle,
  },
  expired: { label: "过期", color: "error", icon: AlertCircle },
  trial: { label: "试用", color: "processing", icon: Clock },
};

// 租户类型配置
const TENANT_TYPES = {
  trial: { label: "试用版", color: "default" },
  standard: { label: "标准版", color: "blue" },
  professional: { label: "专业版", color: "purple" },
  enterprise: { label: "企业版", color: "gold" },
};

// 引入租户API
import { TenantAPI } from "../../lib/tenant-api";

export default function TenantManagement() {
  const [tenants, setTenants] = useState<unknown[]>([]);
  const [loading, setLoading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedTenant, setSelectedTenant] = useState<any>(null);
  const [form] = Form.useForm();
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState("all");
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    suspended: 0,
    trial: 0,
  });

  // 加载租户数据
  const loadTenants = async () => {
    setLoading(true);
    try {
      const response = await TenantAPI.getTenants({
        search: searchTerm || undefined,
        status: statusFilter !== "all" ? statusFilter : undefined,
        type: typeFilter !== "all" ? typeFilter : undefined,
      });
      
      setTenants(response.tenants);
      
      // 计算统计数据
      const total = response.tenants.length;
      const active = response.tenants.filter(t => t.status === "active").length;
      const suspended = response.tenants.filter(t => t.status === "suspended").length;
      const trial = response.tenants.filter(t => t.status === "trial").length;
      
      setStats({
        total,
        active,
        suspended,
        trial,
      });
    } catch (error) {
      console.error("加载租户数据失败:", error);
      message.error("加载租户数据失败");
    } finally {
      setLoading(false);
    }
  };

  // 初始化加载数据
  useEffect(() => {
    loadTenants();
  }, [searchTerm, statusFilter, typeFilter]);

  // 处理保存租户
  const handleSaveTenant = async () => {
    try {
      const values = await form.validateFields();
      
      if (selectedTenant) {
        // 更新租户
        await TenantAPI.updateTenant(selectedTenant.id, values);
        message.success("租户更新成功");
      } else {
        // 创建租户
        await TenantAPI.createTenant(values);
        message.success("租户创建成功");
      }
      
      setShowModal(false);
      form.resetFields();
      setSelectedTenant(null);
      loadTenants(); // 重新加载数据
    } catch (error) {
      console.error("保存租户失败:", error);
      message.error("保存租户失败");
    }
  };

  // 处理删除租户
  const handleDeleteTenant = async (id: number) => {
    try {
      await TenantAPI.deleteTenant(id);
      message.success("租户删除成功");
      loadTenants(); // 重新加载数据
    } catch (error) {
      console.error("删除租户失败:", error);
      message.error("删除租户失败");
    }
  };

  // 表格列定义
  const columns = [
    {
      title: "租户信息",
      key: "info",
      render: (_: unknown, record: unknown) => (
        <div className="flex items-center">
          <div className="flex-shrink-0 h-10 w-10 rounded-full bg-blue-100 flex items-center justify-center">
            <Building2 className="h-5 w-5 text-blue-600" />
          </div>
          <div className="ml-4">
            <div className="text-sm font-medium text-gray-900">{record.name}</div>
            <div className="text-sm text-gray-500">{record.code} • {record.domain}</div>
          </div>
        </div>
      ),
    },
    {
      title: "类型/状态",
      key: "type-status",
      render: (_: unknown, record: unknown) => (
        <div className="space-y-1">
          <Tag color={TENANT_TYPES[record.type]?.color || "default"}>
            {TENANT_TYPES[record.type]?.label || record.type}
          </Tag>
          <div className="flex items-center">
            <Tag color={TENANT_STATUS[record.status]?.color || "default"}>
              {TENANT_STATUS[record.status]?.label || record.status}
            </Tag>
          </div>
        </div>
      ),
    },
    {
      title: "资源使用",
      key: "usage",
      render: (_: unknown, record: unknown) => (
        <div className="space-y-1">
          <div className="flex items-center">
            <Users className="w-4 h-4 mr-1 text-gray-400" />
            <span>
              {record.userCount || 0} 用户
            </span>
          </div>
          <div className="text-xs text-gray-500">
            {record.ticketCount || 0} 工单
          </div>
        </div>
      ),
    },
    {
      title: "到期时间",
      key: "expires",
      dataIndex: "expires_at",
      render: (expiresAt: string) => (
        <div className="flex items-center">
          <Calendar className="w-4 h-4 mr-1 text-gray-400" />
          {expiresAt ? new Date(expiresAt).toLocaleDateString() : "无"}
        </div>
      ),
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      render: (_: unknown, record: unknown) => (
        <Space size="small">
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => {
                setSelectedTenant(record);
                form.setFieldsValue(record);
                setShowModal(true);
              }}
            />
          </Tooltip>
          <Tooltip title="查看">
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => {
                setSelectedTenant(record);
                form.setFieldsValue(record);
                setShowModal(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title="确认删除"
            description="确定要删除这个租户吗？此操作不可恢复。"
            onConfirm={() => handleDeleteTenant(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button
                type="text"
                danger
                icon={<Trash2 className="w-4 h-4" />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <Building2 className="inline-block w-6 h-6 mr-2" />
          租户管理
        </Title>
        <Text type="secondary">管理系统中的租户和组织</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总租户数"
              value={stats.total}
              prefix={<Building2 className="w-5 h-5" />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="活跃租户"
              value={stats.active}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="暂停租户"
              value={stats.suspended}
              prefix={<AlertCircle className="w-5 h-5" />}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="试用租户"
              value={stats.trial}
              prefix={<Clock className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索和过滤 */}
      <Card className="mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={12} lg={8}>
            <Input
              placeholder="搜索租户名称、编码或域名..."
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={8} lg={4}>
            <Select
              placeholder="筛选状态"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: "100%" }}
            >
              <Option value="all">全部状态</Option>
              <Option value="active">活跃</Option>
              <Option value="suspended">暂停</Option>
              <Option value="expired">过期</Option>
              <Option value="trial">试用</Option>
            </Select>
          </Col>
          <Col xs={24} md={8} lg={4}>
            <Select
              placeholder="筛选类型"
              value={typeFilter}
              onChange={setTypeFilter}
              style={{ width: "100%" }}
            >
              <Option value="all">全部类型</Option>
              <Option value="trial">试用版</Option>
              <Option value="standard">标准版</Option>
              <Option value="professional">专业版</Option>
              <Option value="enterprise">企业版</Option>
            </Select>
          </Col>
          <Col xs={24} md={4} lg={8} className="text-right">
            <Button
              type="primary"
              icon={<Plus className="w-4 h-4" />}
              onClick={() => {
                setSelectedTenant(null);
                form.resetFields();
                setShowModal(true);
              }}
            >
              新建租户
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 租户列表 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={tenants}
          rowKey="id"
          loading={loading}
          pagination={{
            total: tenants.length,
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
          className="enterprise-table"
        />
      </Card>

      {/* 租户编辑模态框 */}
      <Modal
        title={
          <span>
            {selectedTenant ? (
              <>
                <Edit className="w-4 h-4 mr-2" />
                编辑租户
              </>
            ) : (
              <>
                <Plus className="w-4 h-4 mr-2" />
                新建租户
              </>
            )}
          </span>
        }
        open={showModal}
        onOk={handleSaveTenant}
        onCancel={() => {
          setShowModal(false);
          setSelectedTenant(null);
          form.resetFields();
        }}
        width={600}
        confirmLoading={loading}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label="租户名称"
            name="name"
            rules={[{ required: true, message: "请输入租户名称" }]}
          >
            <Input placeholder="请输入租户名称" />
          </Form.Item>
          
          <Form.Item
            label="租户编码"
            name="code"
            rules={[{ required: true, message: "请输入租户编码" }]}
          >
            <Input placeholder="请输入租户编码" />
          </Form.Item>
          
          <Form.Item
            label="域名"
            name="domain"
          >
            <Input placeholder="请输入域名" />
          </Form.Item>
          
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="租户类型"
                name="type"
                rules={[{ required: true, message: "请选择租户类型" }]}
              >
                <Select placeholder="请选择租户类型">
                  <Option value="trial">试用版</Option>
                  <Option value="standard">标准版</Option>
                  <Option value="professional">专业版</Option>
                  <Option value="enterprise">企业版</Option>
                </Select>
              </Form.Item>
            </Col>
            
            <Col span={12}>
              <Form.Item
                label="状态"
                name="status"
                rules={[{ required: true, message: "请选择状态" }]}
              >
                <Select placeholder="请选择状态">
                  <Option value="active">活跃</Option>
                  <Option value="suspended">暂停</Option>
                  <Option value="expired">过期</Option>
                  <Option value="trial">试用</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          
          <Form.Item
            label="到期时间"
            name="expires_at"
          >
            <Input type="date" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}