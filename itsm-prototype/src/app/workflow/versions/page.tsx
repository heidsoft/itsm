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
  Timeline,
  Row,
  Col,
  Typography,
  Alert,
  Descriptions,
  Divider,
  Badge,
  Progress,
  Statistic,
  App,
} from "antd";
import {
  Plus,
  Edit,
  Trash2,
  Copy,
  PlayCircle,
  RotateCcw,
  GitBranch,
  GitCommit,
  GitMerge,
  Eye,
  Download,
  Upload,
  History,
  CheckCircle,
  AlertCircle,
  Clock,
  User,
  Code,
  Diff,
} from "lucide-react";
// AppLayout is handled by parent layout

const { Title, Text } = Typography;
const { Option } = Select;

interface WorkflowVersion {
  id: number;
  workflow_id: number;
  version: string;
  bpmn_xml: string;
  process_variables: Record<string, any>;
  status: "draft" | "active" | "archived";
  created_by: string;
  created_at: string;
  change_log: string;
  is_current: boolean;
  metadata: Record<string, any>;
}

interface VersionComparison {
  version1: WorkflowVersion;
  version2: WorkflowVersion;
  comparison: {
    elements_added: string[];
    elements_removed: string[];
    elements_modified: string[];
    connections_added: string[];
    connections_removed: string[];
    variables_changed: string[];
    is_identical: boolean;
  };
}

const WorkflowVersionsPage = () => {
  const { message } = App.useApp();
  const [versions, setVersions] = useState<WorkflowVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [compareModalVisible, setCompareModalVisible] = useState(false);
  const [selectedVersion, setSelectedVersion] =
    useState<WorkflowVersion | null>(null);
  const [comparison, setComparison] = useState<VersionComparison | null>(null);
  const [workflowId, setWorkflowId] = useState<number>(1);

  // 模拟数据
  const mockVersions: WorkflowVersion[] = [
    {
      id: 1,
      workflow_id: 1,
      version: "1.0.0",
      bpmn_xml:
        '<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions>...</bpmn:definitions>',
      process_variables: { priority: "string", category: "string" },
      status: "active",
      created_by: "张三",
      created_at: "2024-01-15T10:30:00Z",
      change_log: "初始版本，包含基本的工单审批流程",
      is_current: true,
      metadata: { elements_count: 5, connections_count: 4 },
    },
    {
      id: 2,
      workflow_id: 1,
      version: "1.1.0",
      bpmn_xml:
        '<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions>...</bpmn:definitions>',
      process_variables: {
        priority: "string",
        category: "string",
        department: "string",
      },
      status: "draft",
      created_by: "李四",
      created_at: "2024-01-16T14:20:00Z",
      change_log: "添加部门审批环节，优化流程逻辑",
      is_current: false,
      metadata: { elements_count: 7, connections_count: 6 },
    },
    {
      id: 3,
      workflow_id: 1,
      version: "1.0.1",
      bpmn_xml:
        '<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions>...</bpmn:definitions>',
      process_variables: { priority: "string", category: "string" },
      status: "archived",
      created_by: "王五",
      created_at: "2024-01-15T16:45:00Z",
      change_log: "修复审批流程中的bug",
      is_current: false,
      metadata: { elements_count: 5, connections_count: 4 },
    },
  ];

  useEffect(() => {
    loadVersions();
  }, [workflowId]);

  const loadVersions = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise((resolve) => setTimeout(resolve, 500));
      setVersions(mockVersions);
    } catch (error) {
      message.error("加载版本列表失败");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateVersion = () => {
    setSelectedVersion(null);
    setModalVisible(true);
  };

  const handleDeployVersion = async (version: WorkflowVersion) => {
    try {
      // 模拟部署
      message.success(`版本 ${version.version} 部署成功`);
      loadVersions();
    } catch (error) {
      message.error("部署失败");
    }
  };

  const handleRollbackVersion = async (version: WorkflowVersion) => {
    try {
      // 模拟回滚
      message.success(`回滚到版本 ${version.version} 成功`);
      loadVersions();
    } catch (error) {
      message.error("回滚失败");
    }
  };

  const handleCompareVersions = (
    version1: WorkflowVersion,
    version2: WorkflowVersion
  ) => {
    // 模拟版本比较
    const mockComparison: VersionComparison = {
      version1,
      version2,
      comparison: {
        elements_added: ["user_task_3", "exclusive_gateway_1"],
        elements_removed: [],
        elements_modified: ["user_task_1"],
        connections_added: ["flow_5", "flow_6"],
        connections_removed: [],
        variables_changed: ["department"],
        is_identical: false,
      },
    };
    setComparison(mockComparison);
    setCompareModalVisible(true);
  };

  const handleDeleteVersion = async (id: number) => {
    try {
      // 模拟删除
      message.success("版本删除成功");
      loadVersions();
    } catch (error) {
      message.error("删除失败");
    }
  };

  const getStatusColor = (status: string) => {
    const colors = {
      draft: "orange",
      active: "green",
      archived: "gray",
    };
    return colors[status as keyof typeof colors] || "default";
  };

  const getStatusText = (status: string) => {
    const texts = {
      draft: "草稿",
      active: "激活",
      archived: "归档",
    };
    return texts[status as keyof typeof texts] || status;
  };

  const columns = [
    {
      title: "版本号",
      dataIndex: "version",
      key: "version",
      width: 120,
      render: (version: string, record: WorkflowVersion) => (
        <div className="flex items-center space-x-2">
          <span className="font-mono text-sm">{version}</span>
          {record.is_current && (
            <Badge count="当前" style={{ backgroundColor: "#52c41a" }} />
          )}
        </div>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
      ),
    },
    {
      title: "变更日志",
      dataIndex: "change_log",
      key: "change_log",
      render: (changeLog: string) => (
        <div className="max-w-xs truncate">
          <Tooltip title={changeLog}>
            <span>{changeLog}</span>
          </Tooltip>
        </div>
      ),
    },
    {
      title: "创建人",
      dataIndex: "created_by",
      key: "created_by",
      width: 120,
      render: (createdBy: string) => (
        <div className="flex items-center">
          <User className="w-4 h-4 mr-1" />
          <span>{createdBy}</span>
        </div>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => (
        <div className="text-sm">{new Date(date).toLocaleString("zh-CN")}</div>
      ),
    },
    {
      title: "统计",
      key: "stats",
      width: 120,
      render: (record: WorkflowVersion) => (
        <div className="text-sm">
          <div>元素: {record.metadata?.elements_count || 0}</div>
          <div>连接: {record.metadata?.connections_count || 0}</div>
        </div>
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 200,
      render: (record: WorkflowVersion) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => setSelectedVersion(record)}
            />
          </Tooltip>
          {record.status === "draft" && (
            <Tooltip title="部署">
              <Button
                type="text"
                icon={<PlayCircle className="w-4 h-4" />}
                onClick={() => handleDeployVersion(record)}
              />
            </Tooltip>
          )}
          {!record.is_current && record.status === "active" && (
            <Tooltip title="回滚">
              <Button
                type="text"
                icon={<RotateCcw className="w-4 h-4" />}
                onClick={() => handleRollbackVersion(record)}
              />
            </Tooltip>
          )}
          <Tooltip title="比较">
            <Button
              type="text"
              icon={<Diff className="w-4 h-4" />}
              onClick={() => handleCompareVersions(record, versions[0])}
            />
          </Tooltip>
          {record.status === "draft" && (
            <Tooltip title="删除">
              <Button
                type="text"
                danger
                icon={<Trash2 className="w-4 h-4" />}
                onClick={() => handleDeleteVersion(record.id)}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  return (
    <>
      {/* 页面头部 */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">工作流版本管理</h1>
        <p className="text-gray-600 mt-1">管理工作流的不同版本，支持版本控制和回滚</p>
      </div>
      {/* 工具栏 */}
      <Card className="enterprise-card mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={6}>
            <Select
              placeholder="选择工作流"
              value={workflowId}
              onChange={setWorkflowId}
              style={{ width: "100%" }}
            >
              <Option value={1}>工单审批流程</Option>
              <Option value={2}>事件处理流程</Option>
              <Option value={3}>变更管理流程</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={18}>
            <Space>
              <Button
                type="primary"
                icon={<Plus className="w-4 h-4" />}
                onClick={handleCreateVersion}
              >
                新建版本
              </Button>
              <Button icon={<Upload className="w-4 h-4" />}>导入版本</Button>
              <Button icon={<Download className="w-4 h-4" />}>导出版本</Button>
              <Button icon={<History className="w-4 h-4" />}>版本历史</Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 版本统计 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总版本数"
              value={versions.length}
              prefix={<GitBranch className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="激活版本"
              value={versions.filter((v) => v.status === "active").length}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="草稿版本"
              value={versions.filter((v) => v.status === "draft").length}
              prefix={<Clock className="w-5 h-5" />}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="归档版本"
              value={versions.filter((v) => v.status === "archived").length}
              prefix={<AlertCircle className="w-5 h-5" />}
              valueStyle={{ color: "#666" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 版本列表 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={versions}
          rowKey="id"
          loading={loading}
          pagination={false}
          scroll={{ x: 1200 }}
        />
      </Card>

      {/* 版本详情模态框 */}
      <Modal
        title={`版本详情 - ${selectedVersion?.version}`}
        open={!!selectedVersion}
        onCancel={() => setSelectedVersion(null)}
        footer={null}
        width={800}
        destroyOnHidden
      >
        {selectedVersion && (
          <div className="space-y-6">
            <Descriptions column={2}>
              <Descriptions.Item label="版本号">
                {selectedVersion.version}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={getStatusColor(selectedVersion.status)}>
                  {getStatusText(selectedVersion.status)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="创建人">
                {selectedVersion.created_by}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {new Date(selectedVersion.created_at).toLocaleString("zh-CN")}
              </Descriptions.Item>
              <Descriptions.Item label="当前版本">
                {selectedVersion.is_current ? "是" : "否"}
              </Descriptions.Item>
            </Descriptions>

            <Divider>变更日志</Divider>
            <div className="bg-gray-50 p-4 rounded">
              <Text>{selectedVersion.change_log}</Text>
            </div>

            <Divider>流程变量</Divider>
            <pre className="bg-gray-100 p-4 rounded text-sm overflow-auto max-h-40">
              {JSON.stringify(selectedVersion.process_variables, null, 2)}
            </pre>

            <Divider>BPMN XML</Divider>
            <pre className="bg-gray-100 p-4 rounded text-sm overflow-auto max-h-40">
              {selectedVersion.bpmn_xml}
            </pre>
          </div>
        )}
      </Modal>

      {/* 版本比较模态框 */}
      <Modal
        title="版本比较"
        open={compareModalVisible}
        onCancel={() => setCompareModalVisible(false)}
        footer={null}
        width={1000}
        destroyOnHidden
      >
        {comparison && (
          <div className="space-y-6">
            <Row gutter={16}>
              <Col span={12}>
                <Card
                  title={`版本 ${comparison.version1.version}`}
                  size="small"
                >
                  <Descriptions column={1} size="small">
                    <Descriptions.Item label="状态">
                      <Tag color={getStatusColor(comparison.version1.status)}>
                        {getStatusText(comparison.version1.status)}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="创建人">
                      {comparison.version1.created_by}
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间">
                      {new Date(
                        comparison.version1.created_at
                      ).toLocaleDateString("zh-CN")}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>
              </Col>
              <Col span={12}>
                <Card
                  title={`版本 ${comparison.version2.version}`}
                  size="small"
                >
                  <Descriptions column={1} size="small">
                    <Descriptions.Item label="状态">
                      <Tag color={getStatusColor(comparison.version2.status)}>
                        {getStatusText(comparison.version2.status)}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="创建人">
                      {comparison.version2.created_by}
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间">
                      {new Date(
                        comparison.version2.created_at
                      ).toLocaleDateString("zh-CN")}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>
              </Col>
            </Row>

            <Divider>变更详情</Divider>

            <Row gutter={16}>
              <Col span={12}>
                <Card title="新增元素" size="small">
                  {comparison.comparison.elements_added.length > 0 ? (
                    <ul className="list-disc list-inside">
                      {comparison.comparison.elements_added.map(
                        (element, index) => (
                          <li key={index} className="text-green-600">
                            {element}
                          </li>
                        )
                      )}
                    </ul>
                  ) : (
                    <Text type="secondary">无新增元素</Text>
                  )}
                </Card>
              </Col>
              <Col span={12}>
                <Card title="删除元素" size="small">
                  {comparison.comparison.elements_removed.length > 0 ? (
                    <ul className="list-disc list-inside">
                      {comparison.comparison.elements_removed.map(
                        (element, index) => (
                          <li key={index} className="text-red-600">
                            {element}
                          </li>
                        )
                      )}
                    </ul>
                  ) : (
                    <Text type="secondary">无删除元素</Text>
                  )}
                </Card>
              </Col>
            </Row>

            <Row gutter={16}>
              <Col span={12}>
                <Card title="修改元素" size="small">
                  {comparison.comparison.elements_modified.length > 0 ? (
                    <ul className="list-disc list-inside">
                      {comparison.comparison.elements_modified.map(
                        (element, index) => (
                          <li key={index} className="text-orange-600">
                            {element}
                          </li>
                        )
                      )}
                    </ul>
                  ) : (
                    <Text type="secondary">无修改元素</Text>
                  )}
                </Card>
              </Col>
              <Col span={12}>
                <Card title="变量变更" size="small">
                  {comparison.comparison.variables_changed.length > 0 ? (
                    <ul className="list-disc list-inside">
                      {comparison.comparison.variables_changed.map(
                        (variable, index) => (
                          <li key={index} className="text-blue-600">
                            {variable}
                          </li>
                        )
                      )}
                    </ul>
                  ) : (
                    <Text type="secondary">无变量变更</Text>
                  )}
                </Card>
              </Col>
            </Row>

            <Alert
              message={
                comparison.comparison.is_identical
                  ? "两个版本完全相同"
                  : "两个版本存在差异"
              }
              type={comparison.comparison.is_identical ? "info" : "warning"}
              showIcon
            />
          </div>
        )}
      </Modal>
    </>
  );
};

export default WorkflowVersionsPage;
