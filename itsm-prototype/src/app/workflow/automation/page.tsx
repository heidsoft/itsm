"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Modal,
  Form,
  Tag,
  Space,
  Tooltip,
  Switch,
  Row,
  Col,
  Typography,
  Tabs,
  Divider,
  InputNumber,
  Statistic,
  App,
} from "antd";
import {
  Plus,
  Edit,
  Trash2,
  Copy,
  PlayCircle,
  Settings,
  Users,
  GitBranch,
  Clock,
  CheckCircle,
  RefreshCw,
} from "lucide-react";
import AppLayout from "../../components/AppLayout";

const { Title, Text } = Typography;
const { Option } = Select;

interface AutomationRule {
  id: number;
  name: string;
  description: string;
  type: "assignment" | "routing" | "escalation";
  conditions: Record<string, any>;
  actions: Record<string, any>;
  priority: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

const WorkflowAutomationPage = () => {
  const { message } = App.useApp();
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<AutomationRule | null>(null);
  const [activeTab, setActiveTab] = useState("assignment");
  const [automationEnabled, setAutomationEnabled] = useState(true);

  // 模拟数据
  const mockRules: AutomationRule[] = [
    {
      id: 1,
      name: "高优先级任务自动分配给专家",
      description: "将高优先级的技术任务自动分配给专家用户",
      type: "assignment",
      conditions: {
        priority: "high",
        category: "technical",
      },
      actions: {
        assign_to: "expert",
        method: "round_robin",
      },
      priority: 1,
      is_active: true,
      created_at: "2024-01-15T10:30:00Z",
      updated_at: "2024-01-15T14:20:00Z",
    },
    {
      id: 2,
      name: "技术问题智能路由",
      description: "技术相关问题自动路由到技术支持组",
      type: "routing",
      conditions: {
        category: "technical",
        priority: "high",
      },
      actions: {
        route_to: "tech_support",
        notify: true,
      },
      priority: 1,
      is_active: true,
      created_at: "2024-01-15T09:15:00Z",
      updated_at: "2024-01-15T11:45:00Z",
    },
    {
      id: 3,
      name: "高优先级任务4小时自动升级",
      description: "高优先级任务4小时内未处理自动升级到经理",
      type: "escalation",
      conditions: {
        priority: "high",
        status: "pending",
      },
      actions: {
        escalate_to: "manager",
        time_limit: 4,
        notify: true,
      },
      priority: 1,
      is_active: true,
      created_at: "2024-01-14T16:20:00Z",
      updated_at: "2024-01-15T10:30:00Z",
    },
  ];

  useEffect(() => {
    loadRules();
  }, []);

  const loadRules = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise((resolve) => setTimeout(resolve, 500));
      setRules(mockRules);
    } catch (error) {
      message.error("加载自动化规则失败");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRule = () => {
    setEditingRule(null);
    setModalVisible(true);
  };

  const handleEditRule = (rule: AutomationRule) => {
    setEditingRule(rule);
    setModalVisible(true);
  };

  const handleDeleteRule = async (id: number) => {
    try {
      // 模拟删除
      message.success("规则删除成功");
      loadRules();
    } catch (error) {
      message.error("删除失败");
    }
  };

  const handleToggleRule = async (id: number, isActive: boolean) => {
    try {
      // 模拟切换状态
      message.success(`规则${isActive ? "启用" : "禁用"}成功`);
      loadRules();
    } catch (error) {
      message.error("操作失败");
    }
  };

  const getRuleTypeColor = (type: string) => {
    const colors = {
      assignment: "blue",
      routing: "green",
      escalation: "orange",
    };
    return colors[type as keyof typeof colors] || "default";
  };

  const getRuleTypeText = (type: string) => {
    const texts = {
      assignment: "自动分配",
      routing: "智能路由",
      escalation: "自动升级",
    };
    return texts[type as keyof typeof texts] || type;
  };

  const getRuleTypeIcon = (type: string) => {
    const icons = {
      assignment: <Users className="w-4 h-4" />,
      routing: <GitBranch className="w-4 h-4" />,
      escalation: <Clock className="w-4 h-4" />,
    };
    return (
      icons[type as keyof typeof icons] || <Settings className="w-4 h-4" />
    );
  };

  const columns = [
    {
      title: "规则名称",
      dataIndex: "name",
      key: "name",
      render: (name: string, record: AutomationRule) => (
        <div>
          <div className="font-medium">{name}</div>
          <div className="text-sm text-gray-500">{record.description}</div>
        </div>
      ),
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 120,
      render: (type: string) => (
        <Tag color={getRuleTypeColor(type)} icon={getRuleTypeIcon(type)}>
          {getRuleTypeText(type)}
        </Tag>
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      render: (priority: number) => (
        <Tag
          color={priority === 1 ? "red" : priority === 2 ? "orange" : "blue"}
        >
          P{priority}
        </Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "is_active",
      key: "is_active",
      width: 100,
      render: (isActive: boolean, record: AutomationRule) => (
        <Switch
          checked={isActive}
          onChange={(checked) => handleToggleRule(record.id, checked)}
          size="small"
        />
      ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => (
        <div className="text-sm">
          {new Date(date).toLocaleDateString("zh-CN")}
        </div>
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 150,
      render: (record: AutomationRule) => (
        <Space>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => handleEditRule(record)}
            />
          </Tooltip>
          <Tooltip title="复制">
            <Button type="text" icon={<Copy className="w-4 h-4" />} />
          </Tooltip>
          <Tooltip title="删除">
            <Button
              type="text"
              danger
              icon={<Trash2 className="w-4 h-4" />}
              onClick={() => handleDeleteRule(record.id)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  const filteredRules = rules.filter((rule) => {
    switch (activeTab) {
      case "assignment":
        return rule.type === "assignment";
      case "routing":
        return rule.type === "routing";
      case "escalation":
        return rule.type === "escalation";
      default:
        return true;
    }
  });

  return (
    <AppLayout
      title="工作流自动化"
      breadcrumb={[
        { title: "工作流管理", href: "/workflow" },
        { title: "工作流自动化", href: "/workflow/automation" },
      ]}
    >
      {/* 全局设置 */}
      <Card className="enterprise-card mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12}>
            <div className="flex items-center space-x-4">
              <Switch
                checked={automationEnabled}
                onChange={setAutomationEnabled}
                size="large"
              />
              <div>
                <Title level={5} className="mb-1">
                  工作流自动化
                </Title>
                <Text type="secondary">启用或禁用所有自动化规则</Text>
              </div>
            </div>
          </Col>
          <Col xs={24} sm={12}>
            <Space>
              <Button
                icon={<RefreshCw className="w-4 h-4" />}
                onClick={loadRules}
              >
                刷新
              </Button>
              <Button icon={<PlayCircle className="w-4 h-4" />} type="primary">
                测试规则
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 统计信息 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="自动分配规则"
              value={rules.filter((r) => r.type === "assignment").length}
              prefix={<Users className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="智能路由规则"
              value={rules.filter((r) => r.type === "routing").length}
              prefix={<GitBranch className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="自动升级规则"
              value={rules.filter((r) => r.type === "escalation").length}
              prefix={<Clock className="w-5 h-5" />}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="活跃规则"
              value={rules.filter((r) => r.is_active).length}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 规则管理 */}
      <Card className="enterprise-card">
        <div className="flex items-center justify-between mb-4">
          <Title level={5}>自动化规则</Title>
          <Button
            type="primary"
            icon={<Plus className="w-4 h-4" />}
            onClick={handleCreateRule}
          >
            新建规则
          </Button>
        </div>

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: "assignment",
              label: "自动分配",
              children: (
                <Table
                  columns={columns}
                  dataSource={filteredRules}
                  rowKey="id"
                  loading={loading}
                  pagination={false}
                />
              ),
            },
            {
              key: "routing",
              label: "智能路由",
              children: (
                <Table
                  columns={columns}
                  dataSource={filteredRules}
                  rowKey="id"
                  loading={loading}
                  pagination={false}
                />
              ),
            },
            {
              key: "escalation",
              label: "自动升级",
              children: (
                <Table
                  columns={columns}
                  dataSource={filteredRules}
                  rowKey="id"
                  loading={loading}
                  pagination={false}
                />
              ),
            },
          ]}
        />
      </Card>

      {/* 创建/编辑规则模态框 */}
      <Modal
        title={editingRule ? "编辑规则" : "新建规则"}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
        destroyOnHidden
      >
        <Form
          layout="vertical"
          initialValues={
            editingRule || {
              type: activeTab,
              priority: 1,
              is_active: true,
              conditions: {},
              actions: {},
            }
          }
          onFinish={async (values) => {
            try {
              message.success(editingRule ? "规则更新成功" : "规则创建成功");
              setModalVisible(false);
              loadRules();
            } catch (error) {
              message.error("操作失败");
            }
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="规则名称"
                rules={[{ required: true, message: "请输入规则名称" }]}
              >
                <Input placeholder="请输入规则名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="type"
                label="规则类型"
                rules={[{ required: true, message: "请选择规则类型" }]}
              >
                <Select placeholder="选择规则类型">
                  <Option value="assignment">自动分配</Option>
                  <Option value="routing">智能路由</Option>
                  <Option value="escalation">自动升级</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="description" label="规则描述">
            <Input.TextArea rows={3} placeholder="请输入规则描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="优先级"
                rules={[{ required: true, message: "请设置优先级" }]}
              >
                <Select placeholder="选择优先级">
                  <Option value={1}>高 (P1)</Option>
                  <Option value={2}>中 (P2)</Option>
                  <Option value={3}>低 (P3)</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="is_active" label="状态" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Divider>触发条件</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name={["conditions", "priority"]} label="优先级">
                <Select placeholder="选择优先级" allowClear>
                  <Option value="low">低</Option>
                  <Option value="normal">普通</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">紧急</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name={["conditions", "category"]} label="分类">
                <Select placeholder="选择分类" allowClear>
                  <Option value="technical">技术</Option>
                  <Option value="finance">财务</Option>
                  <Option value="hr">人事</Option>
                  <Option value="general">通用</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name={["conditions", "status"]} label="状态">
                <Select placeholder="选择状态" allowClear>
                  <Option value="pending">待处理</Option>
                  <Option value="in_progress">处理中</Option>
                  <Option value="completed">已完成</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Divider>执行动作</Divider>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) =>
              prevValues.type !== currentValues.type
            }
          >
            {({ getFieldValue }) => {
              const ruleType = getFieldValue("type");

              if (ruleType === "assignment") {
                return (
                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        name={["actions", "assign_to"]}
                        label="分配给"
                        rules={[{ required: true, message: "请选择分配目标" }]}
                      >
                        <Select placeholder="选择分配目标">
                          <Option value="expert">专家</Option>
                          <Option value="manager">经理</Option>
                          <Option value="supervisor">主管</Option>
                          <Option value="round_robin">轮询分配</Option>
                          <Option value="least_busy">最少忙碌</Option>
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item name={["actions", "method"]} label="分配方法">
                        <Select placeholder="选择分配方法">
                          <Option value="round_robin">轮询</Option>
                          <Option value="least_busy">最少忙碌</Option>
                          <Option value="skill_based">基于技能</Option>
                        </Select>
                      </Form.Item>
                    </Col>
                  </Row>
                );
              }

              if (ruleType === "routing") {
                return (
                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        name={["actions", "route_to"]}
                        label="路由到"
                        rules={[{ required: true, message: "请选择路由目标" }]}
                      >
                        <Select placeholder="选择路由目标">
                          <Option value="tech_support">技术支持组</Option>
                          <Option value="finance_team">财务组</Option>
                          <Option value="hr_team">人事组</Option>
                          <Option value="management">管理层</Option>
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name={["actions", "notify"]}
                        label="通知"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                  </Row>
                );
              }

              if (ruleType === "escalation") {
                return (
                  <Row gutter={16}>
                    <Col span={8}>
                      <Form.Item
                        name={["actions", "escalate_to"]}
                        label="升级到"
                        rules={[{ required: true, message: "请选择升级目标" }]}
                      >
                        <Select placeholder="选择升级目标">
                          <Option value="manager">经理</Option>
                          <Option value="supervisor">主管</Option>
                          <Option value="director">总监</Option>
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item
                        name={["actions", "time_limit"]}
                        label="时间限制(小时)"
                        rules={[{ required: true, message: "请设置时间限制" }]}
                      >
                        <InputNumber
                          min={1}
                          max={168}
                          style={{ width: "100%" }}
                        />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item
                        name={["actions", "notify"]}
                        label="通知"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                  </Row>
                );
              }

              return null;
            }}
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingRule ? "更新" : "创建"}
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </AppLayout>
  );
};

export default WorkflowAutomationPage;
