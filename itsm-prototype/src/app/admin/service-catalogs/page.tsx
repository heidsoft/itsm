"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Modal,
  Form,
  Select,
  Tag,
  Space,
  Row,
  Col,
  Statistic,
  Typography,
  Popconfirm,
  Dropdown,
  Tooltip,
  App,
} from "antd";
import {
  Plus,
  Search,
  Trash2,
  Edit,
  Eye,
  MoreHorizontal,
  Filter,
  RefreshCw,
  Download,
  Upload,
  Settings,
  BookOpen,
  Clock,
  CheckCircle,
  AlertCircle,
} from "lucide-react";
import {
  ServiceCatalogApi,
  ServiceCatalog,
  CreateServiceCatalogRequest,
  UpdateServiceCatalogRequest,
} from "../../lib/service-catalog-api";

const { Title, Text } = Typography;
const { Option } = Select;

const ServiceCatalogManagement = () => {
  const { message } = App.useApp();
  const [catalogs, setCatalogs] = useState<ServiceCatalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingCatalog, setEditingCatalog] = useState<ServiceCatalog | null>(
    null
  );
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [categoryFilter, setCategoryFilter] = useState<string>("");
  const [form] = Form.useForm();

  // 统计数据
  const [stats, setStats] = useState({
    total: 0,
    enabled: 0,
    disabled: 0,
    categories: 0,
  });

  // 获取服务目录列表
  const fetchCatalogs = async () => {
    try {
      setLoading(true);
      const response = await ServiceCatalogApi.getServiceCatalogs({
        page: 1,
        size: 100,
      });
      setCatalogs(response.catalogs);

      // 计算统计数据
      const enabledCount = response.catalogs.filter(
        (c) => c.status === "enabled"
      ).length;
      const categories = new Set(response.catalogs.map((c) => c.category)).size;

      setStats({
        total: response.catalogs.length,
        enabled: enabledCount,
        disabled: response.catalogs.length - enabledCount,
        categories: categories,
      });
    } catch (error) {
      console.error("Failed to fetch catalogs:", error);
      message.error("获取服务目录失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCatalogs();
  }, []);

  // 处理表单提交
  const handleSubmit = async (values: CreateServiceCatalogRequest) => {
    try {
      if (editingCatalog) {
        await ServiceCatalogApi.updateServiceCatalog(editingCatalog.id, values);
        message.success("服务目录更新成功");
      } else {
        await ServiceCatalogApi.createServiceCatalog(values);
        message.success("服务目录创建成功");
      }
      setShowModal(false);
      setEditingCatalog(null);
      form.resetFields();
      fetchCatalogs();
    } catch (error) {
      console.error("Failed to save catalog:", error);
      message.error(editingCatalog ? "更新失败" : "创建失败");
    }
  };

  // 编辑服务目录
  const handleEdit = (catalog: ServiceCatalog) => {
    setEditingCatalog(catalog);
    form.setFieldsValue({
      name: catalog.name,
      category: catalog.category,
      description: catalog.description,
      delivery_time: catalog.delivery_time,
      status: catalog.status,
    });
    setShowModal(true);
  };

  // 删除服务目录
  const handleDelete = async (id: number) => {
    try {
      await ServiceCatalogApi.deleteServiceCatalog(id);
      message.success("删除成功");
      fetchCatalogs();
    } catch (error) {
      console.error("Failed to delete catalog:", error);
      message.error("删除失败");
    }
  };

  // 批量删除
  const handleBatchDelete = async () => {
    try {
      await Promise.all(
        selectedRowKeys.map((id) => ServiceCatalogApi.deleteServiceCatalog(id))
      );
      message.success(`成功删除 ${selectedRowKeys.length} 个服务目录`);
      setSelectedRowKeys([]);
      fetchCatalogs();
    } catch (error) {
      console.error("Failed to batch delete:", error);
      message.error("批量删除失败");
    }
  };

  // 批量启用/禁用
  const handleBatchStatusChange = async (status: "enabled" | "disabled") => {
    try {
      await Promise.all(
        selectedRowKeys.map((id) => {
          const catalog = catalogs.find((c) => c.id === id);
          if (catalog) {
            return ServiceCatalogApi.updateServiceCatalog(id, {
              ...catalog,
              status,
            });
          }
          return Promise.resolve();
        })
      );
      message.success(
        `成功${status === "enabled" ? "启用" : "禁用"} ${
          selectedRowKeys.length
        } 个服务目录`
      );
      setSelectedRowKeys([]);
      fetchCatalogs();
    } catch (error) {
      console.error("Failed to batch update status:", error);
      message.error("批量操作失败");
    }
  };

  // 过滤服务目录
  const filteredCatalogs = catalogs.filter((catalog) => {
    const matchesSearch =
      catalog.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      catalog.category.toLowerCase().includes(searchTerm.toLowerCase()) ||
      catalog.description.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesStatus = !statusFilter || catalog.status === statusFilter;
    const matchesCategory =
      !categoryFilter || catalog.category === categoryFilter;

    return matchesSearch && matchesStatus && matchesCategory;
  });

  // 表格列定义
  const columns = [
    {
      title: "服务名称",
      dataIndex: "name",
      key: "name",
      render: (name: string, record: ServiceCatalog) => (
        <div>
          <div className="font-medium text-gray-900">{name}</div>
          <div className="text-sm text-gray-500 mt-1">{record.description}</div>
        </div>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      width: 120,
      render: (category: string) => <Tag color="blue">{category}</Tag>,
    },
    {
      title: "交付时间",
      dataIndex: "delivery_time",
      key: "delivery_time",
      width: 120,
      render: (time: string) => (
        <span className="text-sm flex items-center">
          <Clock className="w-4 h-4 mr-1" />
          {time}
        </span>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag
          color={status === "enabled" ? "green" : "red"}
          icon={
            status === "enabled" ? (
              <CheckCircle className="w-3 h-3" />
            ) : (
              <AlertCircle className="w-3 h-3" />
            )
          }
        >
          {status === "enabled" ? "启用" : "禁用"}
        </Tag>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => (
        <span className="text-sm text-gray-600">
          {new Date(date).toLocaleDateString("zh-CN")}
        </span>
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 180,
      render: (record: ServiceCatalog) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              size="small"
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => handleEdit(record)}
              size="small"
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个服务目录吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button
                type="text"
                icon={<Trash2 className="w-4 h-4" />}
                danger
                size="small"
              />
            </Tooltip>
          </Popconfirm>
          <Dropdown
            menu={{
              items: [
                {
                  key: "enable",
                  label: "启用",
                  icon: <CheckCircle className="w-4 h-4" />,
                  disabled: record.status === "enabled",
                  onClick: () => handleBatchStatusChange("enabled"),
                },
                {
                  key: "disable",
                  label: "禁用",
                  icon: <AlertCircle className="w-4 h-4" />,
                  disabled: record.status === "disabled",
                  onClick: () => handleBatchStatusChange("disabled"),
                },
                {
                  type: "divider" as const,
                },
                {
                  key: "duplicate",
                  label: "复制",
                  icon: <Plus className="w-4 h-4" />,
                },
              ],
            }}
          >
            <Button
              type="text"
              icon={<MoreHorizontal className="w-4 h-4" />}
              size="small"
            />
          </Dropdown>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      {/* 页面标题 */}
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <BookOpen className="inline-block w-6 h-6 mr-2" />
          服务目录管理
        </Title>
        <Text type="secondary">管理IT服务目录和服务分类</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总服务数"
              value={stats.total}
              prefix={<BookOpen className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="启用服务"
              value={stats.enabled}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="禁用服务"
              value={stats.disabled}
              prefix={<AlertCircle className="w-5 h-5" />}
              valueStyle={{ color: "#ff4d4f" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="服务分类"
              value={stats.categories}
              prefix={<Filter className="w-5 h-5" />}
              valueStyle={{ color: "#722ed1" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className="enterprise-card mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={6}>
            <Input
              placeholder="搜索服务目录..."
              prefix={<Search className="w-4 h-4" />}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="状态筛选"
              value={statusFilter}
              onChange={setStatusFilter}
              allowClear
              style={{ width: "100%" }}
            >
              <Option value="enabled">启用</Option>
              <Option value="disabled">禁用</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="分类筛选"
              value={categoryFilter}
              onChange={setCategoryFilter}
              allowClear
              style={{ width: "100%" }}
            >
              <Option value="云服务">云服务</Option>
              <Option value="基础设施">基础设施</Option>
              <Option value="应用服务">应用服务</Option>
              <Option value="数据服务">数据服务</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={10}>
            <Space>
              <Button
                type="primary"
                icon={<Plus className="w-4 h-4" />}
                onClick={() => {
                  setEditingCatalog(null);
                  form.resetFields();
                  setShowModal(true);
                }}
              >
                新建服务目录
              </Button>
              <Button icon={<Upload className="w-4 h-4" />}>导入</Button>
              <Button icon={<Download className="w-4 h-4" />}>导出</Button>
              <Button
                icon={<RefreshCw className="w-4 h-4" />}
                onClick={fetchCatalogs}
              >
                刷新
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 批量操作工具栏 */}
      {selectedRowKeys.length > 0 && (
        <Card className="enterprise-card mb-4">
          <Row justify="space-between" align="middle">
            <Col>
              <Text>已选择 {selectedRowKeys.length} 个服务目录</Text>
            </Col>
            <Col>
              <Space>
                <Button
                  size="small"
                  onClick={() => handleBatchStatusChange("enabled")}
                >
                  批量启用
                </Button>
                <Button
                  size="small"
                  onClick={() => handleBatchStatusChange("disabled")}
                >
                  批量禁用
                </Button>
                <Popconfirm
                  title="确定要删除选中的服务目录吗？"
                  onConfirm={handleBatchDelete}
                  okText="确定"
                  cancelText="取消"
                >
                  <Button size="small" danger>
                    批量删除
                  </Button>
                </Popconfirm>
                <Button size="small" onClick={() => setSelectedRowKeys([])}>
                  取消选择
                </Button>
              </Space>
            </Col>
          </Row>
        </Card>
      )}

      {/* 服务目录表格 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={filteredCatalogs}
          rowKey="id"
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
            selections: [
              Table.SELECTION_ALL,
              Table.SELECTION_INVERT,
              Table.SELECTION_NONE,
            ],
          }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          scroll={{ x: 1000 }}
        />
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={
          <span>
            <BookOpen className="w-4 h-4 mr-2" />
            {editingCatalog ? "编辑服务目录" : "新建服务目录"}
          </span>
        }
        open={showModal}
        onCancel={() => {
          setShowModal(false);
          setEditingCatalog(null);
          form.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{ status: "enabled" }}
        >
          <Form.Item
            name="name"
            label="服务名称"
            rules={[{ required: true, message: "请输入服务名称" }]}
          >
            <Input placeholder="请输入服务名称" />
          </Form.Item>

          <Form.Item
            name="category"
            label="服务分类"
            rules={[{ required: true, message: "请选择服务分类" }]}
          >
            <Select placeholder="请选择服务分类">
              <Option value="云服务">云服务</Option>
              <Option value="基础设施">基础设施</Option>
              <Option value="应用服务">应用服务</Option>
              <Option value="数据服务">数据服务</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="description"
            label="服务描述"
            rules={[{ required: true, message: "请输入服务描述" }]}
          >
            <Input.TextArea
              rows={3}
              placeholder="请输入服务描述"
              showCount
              maxLength={500}
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="delivery_time"
                label="交付时间"
                rules={[{ required: true, message: "请输入交付时间" }]}
              >
                <Input placeholder="例如：1-3个工作日" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="status"
                label="状态"
                rules={[{ required: true, message: "请选择状态" }]}
              >
                <Select>
                  <Option value="enabled">启用</Option>
                  <Option value="disabled">禁用</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item className="mb-0 mt-6">
            <Space className="w-full justify-end">
              <Button
                onClick={() => {
                  setShowModal(false);
                  setEditingCatalog(null);
                  form.resetFields();
                }}
              >
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                {editingCatalog ? "更新" : "创建"}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ServiceCatalogManagement;
