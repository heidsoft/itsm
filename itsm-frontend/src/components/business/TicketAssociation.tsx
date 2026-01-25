"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Button,
  Input,
  Select,
  Table,
  Tag,
  Space,
  Typography,
  Modal,
  message,
  Tooltip,
  Badge,
  Divider,
  Alert,
  Row,
  Col,
  Form,
  Checkbox,
} from "antd";
import {
  Link,
  Unlink,
  Merge,
  Search,
  Filter,
  Eye,
  Plus,
  Minus,
  AlertTriangle,
  CheckCircle,
  Clock,
  Users,
  FileText,
} from "lucide-react";

const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;

interface Ticket {
  id: number;
  ticketNumber: string;
  title: string;
  status: string;
  priority: string;
  category: string;
  assignee: string;
  created_at: string;
  updated_at: string;
}

interface TicketRelation {
  id: number;
  sourceTicket: Ticket;
  targetTicket: Ticket;
  relationType:
    | "parent"
    | "child"
    | "duplicate"
    | "related"
    | "blocked_by"
    | "blocks";
  description: string;
  created_at: string;
}

interface MergeCandidate {
  ticket: Ticket;
  similarity: number;
  reason: string;
}

export const TicketAssociation: React.FC = () => {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [relations, setRelations] = useState<TicketRelation[]>([]);
  const [mergeCandidates, setMergeCandidates] = useState<MergeCandidate[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState("");
  const [selectedTickets, setSelectedTickets] = useState<number[]>([]);
  const [showMergeModal, setShowMergeModal] = useState(false);
  const [showRelationModal, setShowRelationModal] = useState(false);
  const [relationForm] = Form.useForm();

  useEffect(() => {
    // 这里不再加载 Mock 数据，而是等待父组件传入或通过 API 获取
    // 暂时保持空数组，等待 API 集成
    setTickets([]);
    setRelations([]);
    setMergeCandidates([]);
  }, []);

  const getStatusColor = (status: string): string => {
    switch (status) {
      case "open":
        return "blue";
      case "in_progress":
        return "orange";
      case "resolved":
        return "green";
      case "closed":
        return "default";
      default:
        return "default";
    }
  };

  const getPriorityColor = (priority: string): string => {
    switch (priority) {
      case "urgent":
        return "red";
      case "high":
        return "orange";
      case "medium":
        return "blue";
      case "low":
        return "green";
      default:
        return "default";
    }
  };

  const getRelationTypeLabel = (type: string): string => {
    switch (type) {
      case "parent":
        return "父工单";
      case "child":
        return "子工单";
      case "duplicate":
        return "重复工单";
      case "related":
        return "相关工单";
      case "blocked_by":
        return "被阻塞";
      case "blocks":
        return "阻塞其他";
      default:
        return type;
    }
  };

  const getRelationTypeColor = (type: string): string => {
    switch (type) {
      case "parent":
        return "blue";
      case "child":
        return "green";
      case "duplicate":
        return "red";
      case "related":
        return "purple";
      case "blocked_by":
        return "orange";
      case "blocks":
        return "red";
      default:
        return "default";
    }
  };

  const handleCreateRelation = async (values: any) => {
    try {
      // 模拟创建关联关系
      const newRelation: TicketRelation = {
        id: Date.now(),
        sourceTicket: tickets.find((t) => t.id === values.sourceTicketId)!,
        targetTicket: tickets.find((t) => t.id === values.targetTicketId)!,
        relationType: values.relationType,
        description: values.description,
        created_at: new Date().toISOString(),
      };

      setRelations([...relations, newRelation]);
      message.success("关联关系创建成功");
      setShowRelationModal(false);
      relationForm.resetFields();
    } catch (error) {
      message.error("创建关联关系失败");
    }
  };

  const handleMergeTickets = async (
    primaryTicketId: number,
    secondaryTicketIds: number[]
  ) => {
    try {
      // 模拟合并工单
      message.success(`工单合并成功，主工单: ${primaryTicketId}`);
      setShowMergeModal(false);
      setSelectedTickets([]);
    } catch (error) {
      message.error("工单合并失败");
    }
  };

  const removeRelation = (relationId: number) => {
    setRelations(relations.filter((r) => r.id !== relationId));
    message.success("关联关系已移除");
  };

  const columns = [
    {
      title: "工单号",
      dataIndex: "ticketNumber",
      key: "ticketNumber",
      render: (text: string, record: Ticket) => (
        <Space>
          <Text strong>{text}</Text>
          <Badge
            count={
              relations.filter(
                (r) =>
                  r.sourceTicket.id === record.id ||
                  r.targetTicket.id === record.id
              ).length
            }
          />
        </Space>
      ),
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      render: (text: string) => <Text>{text}</Text>,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>
          {status === "open"
            ? "待处理"
            : status === "in_progress"
            ? "处理中"
            : status === "resolved"
            ? "已解决"
            : "已关闭"}
        </Tag>
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      render: (priority: string) => (
        <Tag color={getPriorityColor(priority)}>{priority.toUpperCase()}</Tag>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      render: (category: string) => <Text>{category}</Text>,
    },
    {
      title: "处理人",
      dataIndex: "assignee",
      key: "assignee",
      render: (assignee: string) => <Text>{assignee}</Text>,
    },
    {
      title: "操作",
      key: "action",
      render: (_: any, record: Ticket) => (
        <Space>
          <Button
            size="small"
            icon={<Eye />}
            onClick={() => window.open(`/tickets/${record.id}`, "_blank")}
          >
            查看
          </Button>
          <Button
            size="small"
            icon={<Link />}
            onClick={() => {
              setSelectedTickets([record.id]);
              setShowRelationModal(true);
            }}
          >
            关联
          </Button>
        </Space>
      ),
    },
  ];

  const relationColumns = [
    {
      title: "源工单",
      key: "source",
      render: (record: TicketRelation) => (
        <div>
          <div className="font-medium">{record.sourceTicket.ticketNumber}</div>
          <div className="text-sm text-gray-500">
            {record.sourceTicket.title}
          </div>
        </div>
      ),
    },
    {
      title: "关系类型",
      dataIndex: "relationType",
      key: "relationType",
      render: (type: string) => (
        <Tag color={getRelationTypeColor(type)}>
          {getRelationTypeLabel(type)}
        </Tag>
      ),
    },
    {
      title: "目标工单",
      key: "target",
      render: (record: TicketRelation) => (
        <div>
          <div className="font-medium">{record.targetTicket.ticketNumber}</div>
          <div className="text-sm text-gray-500">
            {record.targetTicket.title}
          </div>
        </div>
      ),
    },
    {
      title: "描述",
      dataIndex: "description",
      key: "description",
      render: (text: string) => <Text>{text}</Text>,
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      render: (text: string) => <Text>{text}</Text>,
    },
    {
      title: "操作",
      key: "action",
      render: (_: any, record: TicketRelation) => (
        <Button
          size="small"
          danger
          icon={<Unlink />}
          onClick={() => removeRelation(record.id)}
        >
          移除
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 工单关联管理 */}
      <Card
        title={
          <Space>
            <Link className="text-blue-600" />
            <span>工单关联管理</span>
            <Badge count={relations.length} />
          </Space>
        }
        extra={
          <Space>
            <Button
              icon={<Plus />}
              onClick={() => setShowRelationModal(true)}
              disabled={selectedTickets.length !== 2}
            >
              创建关联
            </Button>
            <Button
              icon={<Merge />}
              onClick={() => setShowMergeModal(true)}
              disabled={selectedTickets.length < 2}
            >
              合并工单
            </Button>
          </Space>
        }
      >
        <div className="mb-4">
          <Input.Search
            placeholder="搜索工单..."
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
            style={{ width: 300 }}
            onSearch={(value) => console.log("搜索:", value)}
          />
        </div>

        <Table
          columns={columns}
          dataSource={tickets}
          rowKey="id"
          rowSelection={{
            type: "checkbox",
            selectedRowKeys: selectedTickets,
            onChange: (selectedRowKeys) =>
              setSelectedTickets(selectedRowKeys as number[]),
          }}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      {/* 关联关系列表 */}
      <Card
        title={
          <Space>
            <Link className="text-green-600" />
            <span>关联关系</span>
            <Badge count={relations.length} />
          </Space>
        }
      >
        {relations.length === 0 ? (
          <div className="text-center py-8 text-gray-500">
            <Link className="w-12 h-12 mx-auto mb-3 text-gray-300" />
            <p>暂无关联关系</p>
          </div>
        ) : (
          <Table
            columns={relationColumns}
            dataSource={relations}
            rowKey="id"
            pagination={{ pageSize: 5 }}
          />
        )}
      </Card>

      {/* 合并候选工单 */}
      <Card
        title={
          <Space>
            <Merge className="text-orange-600" />
            <span>合并候选工单</span>
            <Badge count={mergeCandidates.length} />
          </Space>
        }
      >
        {mergeCandidates.length === 0 ? (
          <div className="text-center py-8 text-gray-500">
            <Merge className="w-12 h-12 mx-auto mb-3 text-gray-300" />
            <p>暂无合并候选工单</p>
          </div>
        ) : (
          <div className="space-y-3">
            {mergeCandidates.map((candidate, index) => (
              <div key={index} className="p-3 border rounded hover:bg-gray-50">
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="flex items-center mb-2">
                      <Text strong className="mr-3">
                        {candidate.ticket.ticketNumber}
                      </Text>
                      <Tag color="orange">相似度: {candidate.similarity}%</Tag>
                    </div>
                    <Text className="block mb-1">{candidate.ticket.title}</Text>
                    <Text type="secondary" className="text-sm">
                      {candidate.reason}
                    </Text>
                  </div>
                  <Space>
                    <Button size="small" icon={<Eye />}>
                      查看
                    </Button>
                    <Button size="small" type="primary" icon={<Merge />}>
                      合并
                    </Button>
                  </Space>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* 创建关联关系模态框 */}
      <Modal
        title="创建工单关联"
        open={showRelationModal}
        onCancel={() => setShowRelationModal(false)}
        footer={null}
        width={600}
      >
        <Form
          form={relationForm}
          layout="vertical"
          onFinish={handleCreateRelation}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="sourceTicketId"
                label="源工单"
                rules={[{ required: true, message: "请选择源工单" }]}
              >
                <Select placeholder="选择源工单">
                  {tickets.map((ticket) => (
                    <Option key={ticket.id} value={ticket.id}>
                      {ticket.ticketNumber} - {ticket.title}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="targetTicketId"
                label="目标工单"
                rules={[{ required: true, message: "请选择目标工单" }]}
              >
                <Select placeholder="选择目标工单">
                  {tickets.map((ticket) => (
                    <Option key={ticket.id} value={ticket.id}>
                      {ticket.ticketNumber} - {ticket.title}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="relationType"
            label="关联类型"
            rules={[{ required: true, message: "请选择关联类型" }]}
          >
            <Select placeholder="选择关联类型">
              <Option value="parent">父工单</Option>
              <Option value="child">子工单</Option>
              <Option value="duplicate">重复工单</Option>
              <Option value="related">相关工单</Option>
              <Option value="blocked_by">被阻塞</Option>
              <Option value="blocks">阻塞其他</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="description"
            label="关联说明"
            rules={[{ required: true, message: "请填写关联说明" }]}
          >
            <TextArea rows={3} placeholder="请描述工单间的关联关系..." />
          </Form.Item>

          <div className="flex justify-end space-x-2">
            <Button onClick={() => setShowRelationModal(false)}>取消</Button>
            <Button type="primary" htmlType="submit">
              创建关联
            </Button>
          </div>
        </Form>
      </Modal>

      {/* 合并工单模态框 */}
      <Modal
        title="合并工单"
        open={showMergeModal}
        onCancel={() => setShowMergeModal(false)}
        footer={null}
        width={800}
      >
        <Alert
          message="工单合并说明"
          description="合并工单将保留主工单，将其他工单的信息合并到主工单中。合并后其他工单将被关闭。"
          type="info"
          showIcon
          className="mb-4"
        />

        <div className="space-y-4">
          <div>
            <Text strong>选择主工单（保留）:</Text>
            <Select
              placeholder="选择要保留的主工单"
              style={{ width: "100%", marginTop: 8 }}
            >
              {tickets
                .filter((t) => selectedTickets.includes(t.id))
                .map((ticket) => (
                  <Option key={ticket.id} value={ticket.id}>
                    {ticket.ticketNumber} - {ticket.title}
                  </Option>
                ))}
            </Select>
          </div>

          <div>
            <Text strong>选择要合并的工单:</Text>
            <div className="mt-2 space-y-2">
              {tickets
                .filter((t) => selectedTickets.includes(t.id))
                .map((ticket) => (
                  <div key={ticket.id} className="flex items-center space-x-2">
                    <Checkbox defaultChecked />
                    <Text>
                      {ticket.ticketNumber} - {ticket.title}
                    </Text>
                  </div>
                ))}
            </div>
          </div>

          <div>
            <Text strong>合并说明:</Text>
            <TextArea
              rows={3}
              placeholder="请描述合并的原因和注意事项..."
              className="mt-2"
            />
          </div>
        </div>

        <div className="flex justify-end space-x-2 mt-6">
          <Button onClick={() => setShowMergeModal(false)}>取消</Button>
          <Button type="primary" danger>
            确认合并
          </Button>
        </div>
      </Modal>
    </div>
  );
};
