"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Tabs,
  Button,
  Input,
  Form,
  message,
  Avatar,
  Tag,
  Timeline,
  Upload,
  Space,
  Divider,
  Typography,
  Row,
  Col,
  Statistic,
  Badge,
  Tooltip,
  Modal,
  Select,
  DatePicker,
} from "antd";
import KnowledgeIntegration from "./KnowledgeIntegration";
import {
  User,
  Clock,
  MessageSquare,
  Paperclip,
  Edit,
  Save,
  X,
  CheckCircle,
  AlertTriangle,
  TrendingUp,
  FileText,
  Download,
  Eye,
  History,
  Workflow,
  Settings,
  Plus,
  Send,
  MoreHorizontal,
  Star,
  Flag,
  Link,
  Calendar,
  Phone,
  Mail,
  MapPin,
  BookOpen,
} from "lucide-react";
import type { UploadFile, UploadProps } from "antd/es/upload/interface";

const { TextArea } = Input;
const { Text, Title } = Typography;
const { Option } = Select;

// 工单详情接口
interface EnhancedTicket {
  id: number;
  ticket_number: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  category: string;
  subcategory?: string;
  assignee?: {
    id: number;
    name: string;
    avatar?: string;
    email: string;
    phone?: string;
    department: string;
  };
  requester: {
    id: number;
    name: string;
    avatar?: string;
    email: string;
    phone?: string;
    department: string;
  };
  created_at: string;
  updated_at: string;
  due_date?: string;
  sla_deadline?: string;
  tags: string[];
  impact: "low" | "medium" | "high" | "critical";
  urgency: "low" | "medium" | "high" | "critical";
  resolution?: string;
  work_notes?: string;
  attachments: Attachment[];
  comments: Comment[];
  activities: Activity[];
  workflow_steps: WorkflowStep[];
  related_tickets: RelatedTicket[];
  sla_metrics: SLAMetrics;
}

interface Attachment {
  id: number;
  name: string;
  size: number;
  type: string;
  url: string;
  uploaded_by: string;
  uploaded_at: string;
}

interface Comment {
  id: number;
  content: string;
  author: {
    id: number;
    name: string;
    avatar?: string;
    role: string;
  };
  created_at: string;
  is_internal: boolean;
  attachments?: Attachment[];
}

interface Activity {
  id: number;
  action: string;
  details: string;
  user: {
    id: number;
    name: string;
    avatar?: string;
  };
  timestamp: string;
  type:
    | "status_change"
    | "assignment"
    | "comment"
    | "attachment"
    | "sla_update";
}

interface WorkflowStep {
  id: number;
  name: string;
  status: "pending" | "in_progress" | "completed" | "skipped";
  assignee?: {
    id: number;
    name: string;
    avatar?: string;
  };
  due_date?: string;
  completed_at?: string;
  comments?: string;
}

interface RelatedTicket {
  id: number;
  ticket_number: string;
  title: string;
  status: string;
  priority: string;
  relationship_type: "parent" | "child" | "related" | "duplicate";
}

interface SLAMetrics {
  response_time: number;
  resolution_time: number;
  response_deadline: string;
  resolution_deadline: string;
  is_breached: boolean;
  breach_reason?: string;
}

interface EnhancedTicketDetailProps {
  ticket: EnhancedTicket;
  onUpdate: (updates: Partial<EnhancedTicket>) => Promise<void>;
  onAssign: (assigneeId: number) => Promise<void>;
  onStatusChange: (status: string) => Promise<void>;
  onPriorityChange: (priority: string) => Promise<void>;
  onAddComment: (comment: Omit<Comment, "id" | "created_at">) => Promise<void>;
  onAddAttachment: (file: File) => Promise<void>;
  canEdit: boolean;
  canAssign: boolean;
  canChangeStatus: boolean;
}

export const EnhancedTicketDetail: React.FC<EnhancedTicketDetailProps> = ({
  ticket,
  onUpdate,
  onAssign,
  onStatusChange,
  onPriorityChange,
  onAddComment,
  onAddAttachment,
  canEdit,
  canAssign,
  canChangeStatus,
}) => {
  const [activeTab, setActiveTab] = useState("overview");
  const [isEditing, setIsEditing] = useState(false);
  const [editingTicket, setEditingTicket] = useState<EnhancedTicket>(ticket);
  const [commentText, setCommentText] = useState("");
  const [isInternalComment, setIsInternalComment] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      open: "blue",
      in_progress: "processing",
      pending: "warning",
      resolved: "success",
      closed: "default",
      cancelled: "error",
    };
    return colors[status] || "default";
  };

  // 获取优先级颜色
  const getPriorityColor = (priority: string) => {
    const colors: Record<string, string> = {
      low: "blue",
      medium: "orange",
      high: "red",
      critical: "red",
    };
    return colors[priority] || "default";
  };

  // 获取影响程度颜色
  const getImpactColor = (impact: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "orange",
      high: "red",
      critical: "red",
    };
    return colors[impact] || "default";
  };

  // 处理保存编辑
  const handleSave = async () => {
    try {
      await onUpdate(editingTicket);
      setIsEditing(false);
      message.success("工单更新成功");
    } catch (error) {
      message.error("更新失败");
    }
  };

  // 处理取消编辑
  const handleCancel = () => {
    setEditingTicket(ticket);
    setIsEditing(false);
  };

  // 处理添加评论
  const handleAddComment = async () => {
    if (!commentText.trim()) return;

    try {
      await onAddComment({
        content: commentText,
        author: {
          id: 1, // 当前用户ID
          name: "当前用户",
          role: "用户",
        },
        is_internal: isInternalComment,
      });
      setCommentText("");
      message.success("评论添加成功");
    } catch (error) {
      message.error("评论添加失败");
    }
  };

  // 处理文件上传
  const handleUpload: UploadProps["customRequest"] = async (options) => {
    const { file, onSuccess, onError } = options;
    if (!file) return;

    try {
      setUploading(true);
      await onAddAttachment(file as File);
      onSuccess?.(file);
      message.success("附件上传成功");
    } catch (error) {
      onError?.(error as Error);
      message.error("附件上传失败");
    } finally {
      setUploading(false);
    }
  };

  // 计算SLA状态
  const getSLAStatus = () => {
    if (!ticket.sla_metrics) return null;

    const now = new Date();
    const responseDeadline = new Date(ticket.sla_metrics.response_deadline);
    const resolutionDeadline = new Date(ticket.sla_metrics.resolution_deadline);

    if (ticket.sla_metrics.is_breached) {
      return { status: "breached", color: "red", text: "SLA已违反" };
    }

    if (now > resolutionDeadline) {
      return { status: "overdue", color: "orange", text: "解决时间已超期" };
    }

    if (now > responseDeadline) {
      return {
        status: "response_overdue",
        color: "orange",
        text: "响应时间已超期",
      };
    }

    return { status: "normal", color: "green", text: "SLA正常" };
  };

  const slaStatus = getSLAStatus();

  return (
    <div className="space-y-6">
      {/* 工单头部信息 */}
      <Card className="shadow-sm">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center space-x-3 mb-4">
              <Badge
                status={getStatusColor(ticket.status) as any}
                text={
                  <span className="text-lg font-medium">
                    {ticket.ticket_number}
                  </span>
                }
              />
              <Title level={3} className="mb-0">
                {isEditing ? (
                  <Input
                    value={editingTicket.title}
                    onChange={(e) =>
                      setEditingTicket({
                        ...editingTicket,
                        title: e.target.value,
                      })
                    }
                    className="text-lg"
                  />
                ) : (
                  ticket.title
                )}
              </Title>
            </div>

            <div className="flex items-center space-x-4 mb-4">
              <Tag color={getStatusColor(ticket.status)}>
                {ticket.status === "open" && "待处理"}
                {ticket.status === "in_progress" && "处理中"}
                {ticket.status === "pending" && "待审批"}
                {ticket.status === "resolved" && "已解决"}
                {ticket.status === "closed" && "已关闭"}
                {ticket.status === "cancelled" && "已取消"}
              </Tag>
              <Tag color={getPriorityColor(ticket.priority)}>
                {ticket.priority === "low" && "低优先级"}
                {ticket.priority === "medium" && "中优先级"}
                {ticket.priority === "high" && "高优先级"}
                {ticket.priority === "critical" && "紧急"}
              </Tag>
              <Tag color={getImpactColor(ticket.impact)}>
                影响: {ticket.impact === "low" && "低"}
                {ticket.impact === "medium" && "中"}
                {ticket.impact === "high" && "高"}
                {ticket.impact === "critical" && "严重"}
              </Tag>
              {slaStatus && <Tag color={slaStatus.color}>{slaStatus.text}</Tag>}
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <Text type="secondary">分类</Text>
                <div className="font-medium">{ticket.category}</div>
              </div>
              <div>
                <Text type="secondary">子分类</Text>
                <div className="font-medium">{ticket.subcategory || "-"}</div>
              </div>
              <div>
                <Text type="secondary">创建时间</Text>
                <div className="font-medium">
                  {new Date(ticket.created_at).toLocaleString("zh-CN")}
                </div>
              </div>
              <div>
                <Text type="secondary">最后更新</Text>
                <div className="font-medium">
                  {new Date(ticket.updated_at).toLocaleString("zh-CN")}
                </div>
              </div>
            </div>
          </div>

          <div className="flex space-x-2">
            {canEdit && (
              <>
                {isEditing ? (
                  <>
                    <Button type="primary" icon={<Save />} onClick={handleSave}>
                      保存
                    </Button>
                    <Button icon={<X />} onClick={handleCancel}>
                      取消
                    </Button>
                  </>
                ) : (
                  <Button icon={<Edit />} onClick={() => setIsEditing(true)}>
                    编辑
                  </Button>
                )}
              </>
            )}
          </div>
        </div>
      </Card>

      {/* 主要内容区域 */}
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: "overview",
            label: (
              <span className="flex items-center space-x-2">
                <Eye className="w-4 h-4" />
                <span>概览</span>
              </span>
            ),
            children: (
              <div className="space-y-6">
                {/* 基本信息 */}
                <Card title="基本信息" className="shadow-sm">
                  <Row gutter={[24, 16]}>
                    <Col xs={24} md={12}>
                      <div className="space-y-4">
                        <div>
                          <Text type="secondary">描述</Text>
                          {isEditing ? (
                            <TextArea
                              value={editingTicket.description}
                              onChange={(e) =>
                                setEditingTicket({
                                  ...editingTicket,
                                  description: e.target.value,
                                })
                              }
                              rows={4}
                              className="mt-2"
                            />
                          ) : (
                            <div className="mt-2 p-3 bg-gray-50 rounded-md">
                              {ticket.description}
                            </div>
                          )}
                        </div>

                        <div>
                          <Text type="secondary">工作备注</Text>
                          {isEditing ? (
                            <TextArea
                              value={editingTicket.work_notes || ""}
                              onChange={(e) =>
                                setEditingTicket({
                                  ...editingTicket,
                                  work_notes: e.target.value,
                                })
                              }
                              rows={3}
                              className="mt-2"
                              placeholder="添加工作备注..."
                            />
                          ) : (
                            <div className="mt-2 p-3 bg-gray-50 rounded-md">
                              {ticket.work_notes || "暂无工作备注"}
                            </div>
                          )}
                        </div>
                      </div>
                    </Col>

                    <Col xs={24} md={12}>
                      <div className="space-y-4">
                        {/* 分配信息 */}
                        <div>
                          <Text type="secondary">处理人</Text>
                          <div className="mt-2">
                            {ticket.assignee ? (
                              <div className="flex items-center space-x-2 p-2 bg-blue-50 rounded-md">
                                <Avatar src={ticket.assignee.avatar}>
                                  {ticket.assignee.name.charAt(0)}
                                </Avatar>
                                <div>
                                  <div className="font-medium">
                                    {ticket.assignee.name}
                                  </div>
                                  <div className="text-sm text-gray-500">
                                    {ticket.assignee.department}
                                  </div>
                                </div>
                              </div>
                            ) : (
                              <div className="text-gray-400">未分配</div>
                            )}
                          </div>
                        </div>

                        {/* 请求人信息 */}
                        <div>
                          <Text type="secondary">请求人</Text>
                          <div className="mt-2 p-2 bg-gray-50 rounded-md">
                            <div className="flex items-center space-x-2">
                              <Avatar src={ticket.requester.avatar}>
                                {ticket.requester.name.charAt(0)}
                              </Avatar>
                              <div>
                                <div className="font-medium">
                                  {ticket.requester.name}
                                </div>
                                <div className="text-sm text-gray-500">
                                  {ticket.requester.department}
                                </div>
                                <div className="text-sm text-gray-500">
                                  {ticket.requester.email}
                                </div>
                              </div>
                            </div>
                          </div>
                        </div>

                        {/* SLA信息 */}
                        {ticket.sla_metrics && (
                          <div>
                            <Text type="secondary">SLA信息</Text>
                            <div className="mt-2 space-y-2">
                              <div className="flex justify-between">
                                <span>响应时间:</span>
                                <span>
                                  {ticket.sla_metrics.response_time}分钟
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span>解决时间:</span>
                                <span>
                                  {ticket.sla_metrics.resolution_time}分钟
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span>响应截止:</span>
                                <span>
                                  {new Date(
                                    ticket.sla_metrics.response_deadline
                                  ).toLocaleString("zh-CN")}
                                </span>
                              </div>
                            </div>
                          </div>
                        )}
                      </div>
                    </Col>
                  </Row>
                </Card>

                {/* 工作流状态 */}
                {ticket.workflow_steps.length > 0 && (
                  <Card title="工作流状态" className="shadow-sm">
                    <Timeline>
                      {ticket.workflow_steps.map((step, index) => (
                        <Timeline.Item
                          key={step.id}
                          color={
                            step.status === "completed"
                              ? "green"
                              : step.status === "in_progress"
                              ? "blue"
                              : "gray"
                          }
                        >
                          <div className="flex items-center justify-between">
                            <div>
                              <div className="font-medium">{step.name}</div>
                              {step.assignee && (
                                <div className="text-sm text-gray-500">
                                  负责人: {step.assignee.name}
                                </div>
                              )}
                              {step.comments && (
                                <div className="text-sm text-gray-500 mt-1">
                                  {step.comments}
                                </div>
                              )}
                            </div>
                            <div className="text-right">
                              <div className="text-sm text-gray-500">
                                {step.status === "completed" &&
                                step.completed_at
                                  ? new Date(
                                      step.completed_at
                                    ).toLocaleDateString("zh-CN")
                                  : step.due_date
                                  ? `截止: ${new Date(
                                      step.due_date
                                    ).toLocaleDateString("zh-CN")}`
                                  : ""}
                              </div>
                            </div>
                          </div>
                        </Timeline.Item>
                      ))}
                    </Timeline>
                  </Card>
                )}

                {/* 相关工单 */}
                {ticket.related_tickets.length > 0 && (
                  <Card title="相关工单" className="shadow-sm">
                    <div className="space-y-2">
                      {ticket.related_tickets.map((related) => (
                        <div
                          key={related.id}
                          className="flex items-center justify-between p-3 bg-gray-50 rounded-md"
                        >
                          <div className="flex items-center space-x-3">
                            <Link className="w-4 h-4 text-blue-500" />
                            <div>
                              <div className="font-medium">
                                {related.ticket_number}
                              </div>
                              <div className="text-sm text-gray-600">
                                {related.title}
                              </div>
                            </div>
                          </div>
                          <div className="flex items-center space-x-2">
                            <Tag color={getStatusColor(related.status)}>
                              {related.status}
                            </Tag>
                            <Tag color={getPriorityColor(related.priority)}>
                              {related.priority}
                            </Tag>
                            <Tag color="blue">
                              {related.relationship_type === "parent" &&
                                "父工单"}
                              {related.relationship_type === "child" &&
                                "子工单"}
                              {related.relationship_type === "related" &&
                                "相关"}
                              {related.relationship_type === "duplicate" &&
                                "重复"}
                            </Tag>
                          </div>
                        </div>
                      ))}
                    </div>
                  </Card>
                )}
              </div>
            ),
          },
          {
            key: "comments",
            label: (
              <span className="flex items-center space-x-2">
                <MessageSquare className="w-4 h-4" />
                <span>评论 ({ticket.comments.length})</span>
              </span>
            ),
            children: (
              <div className="space-y-6">
                {/* 添加评论 */}
                <Card title="添加评论" className="shadow-sm">
                  <div className="space-y-4">
                    <div className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        id="internal-comment"
                        checked={isInternalComment}
                        onChange={(e) => setIsInternalComment(e.target.checked)}
                      />
                      <label htmlFor="internal-comment" className="text-sm">
                        内部评论（仅处理人可见）
                      </label>
                    </div>
                    <TextArea
                      value={commentText}
                      onChange={(e) => setCommentText(e.target.value)}
                      rows={4}
                      placeholder="输入您的评论..."
                    />
                    <div className="flex justify-between items-center">
                      <Upload
                        fileList={fileList}
                        onChange={({ fileList }) => setFileList(fileList)}
                        customRequest={handleUpload}
                        multiple
                        showUploadList={false}
                      >
                        <Button icon={<Paperclip />} loading={uploading}>
                          添加附件
                        </Button>
                      </Upload>
                      <Button
                        type="primary"
                        icon={<Send />}
                        onClick={handleAddComment}
                        disabled={!commentText.trim()}
                      >
                        发送评论
                      </Button>
                    </div>
                  </div>
                </Card>

                {/* 评论列表 */}
                <div className="space-y-4">
                  {ticket.comments.map((comment) => (
                    <Card key={comment.id} className="shadow-sm">
                      <div className="flex space-x-3">
                        <Avatar src={comment.author.avatar}>
                          {comment.author.name.charAt(0)}
                        </Avatar>
                        <div className="flex-1">
                          <div className="flex items-center space-x-2 mb-2">
                            <span className="font-medium">
                              {comment.author.name}
                            </span>
                            <Tag size="small" color="blue">
                              {comment.author.role}
                            </Tag>
                            {comment.is_internal && (
                              <Tag size="small" color="orange">
                                内部
                              </Tag>
                            )}
                            <span className="text-sm text-gray-500">
                              {new Date(comment.created_at).toLocaleString(
                                "zh-CN"
                              )}
                            </span>
                          </div>
                          <div className="text-gray-700 mb-3">
                            {comment.content}
                          </div>
                          {comment.attachments &&
                            comment.attachments.length > 0 && (
                              <div className="space-y-2">
                                {comment.attachments.map((attachment) => (
                                  <div
                                    key={attachment.id}
                                    className="flex items-center space-x-2 p-2 bg-gray-50 rounded-md"
                                  >
                                    <Paperclip className="w-4 h-4 text-gray-500" />
                                    <span className="text-sm text-gray-600">
                                      {attachment.name}
                                    </span>
                                    <span className="text-sm text-gray-400">
                                      ({attachment.size} bytes)
                                    </span>
                                    <Button
                                      type="link"
                                      size="small"
                                      icon={<Download />}
                                      onClick={() =>
                                        window.open(attachment.url, "_blank")
                                      }
                                    >
                                      下载
                                    </Button>
                                  </div>
                                ))}
                              </div>
                            )}
                        </div>
                      </div>
                    </Card>
                  ))}
                </div>
              </div>
            ),
          },
          {
            key: "attachments",
            label: (
              <span className="flex items-center space-x-2">
                <Paperclip className="w-4 h-4" />
                <span>附件 ({ticket.attachments.length})</span>
              </span>
            ),
            children: (
              <div className="space-y-6">
                {/* 上传附件 */}
                <Card title="上传附件" className="shadow-sm">
                  <Upload
                    fileList={fileList}
                    onChange={({ fileList }) => setFileList(fileList)}
                    customRequest={handleUpload}
                    multiple
                    showUploadList={false}
                  >
                    <Button icon={<Plus />} loading={uploading}>
                      选择文件
                    </Button>
                  </Upload>
                  <div className="text-sm text-gray-500 mt-2">
                    支持的文件格式: PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, JPG,
                    PNG, GIF
                  </div>
                </Card>

                {/* 附件列表 */}
                <Card title="附件列表" className="shadow-sm">
                  <div className="space-y-2">
                    {ticket.attachments.map((attachment) => (
                      <div
                        key={attachment.id}
                        className="flex items-center justify-between p-3 bg-gray-50 rounded-md"
                      >
                        <div className="flex items-center space-x-3">
                          <FileText className="w-5 h-5 text-blue-500" />
                          <div>
                            <div className="font-medium">{attachment.name}</div>
                            <div className="text-sm text-gray-500">
                              {attachment.type} •{" "}
                              {(attachment.size / 1024).toFixed(1)} KB
                            </div>
                            <div className="text-sm text-gray-400">
                              上传者: {attachment.uploaded_by} •{" "}
                              {new Date(attachment.uploaded_at).toLocaleString(
                                "zh-CN"
                              )}
                            </div>
                          </div>
                        </div>
                        <div className="flex space-x-2">
                          <Button
                            type="text"
                            icon={<Eye />}
                            onClick={() =>
                              window.open(attachment.url, "_blank")
                            }
                          >
                            预览
                          </Button>
                          <Button
                            type="text"
                            icon={<Download />}
                            onClick={() =>
                              window.open(attachment.url, "_blank")
                            }
                          >
                            下载
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              </div>
            ),
          },
          {
            key: "activities",
            label: (
              <span className="flex items-center space-x-2">
                <History className="w-4 h-4" />
                <span>活动记录 ({ticket.activities.length})</span>
              </span>
            ),
            children: (
              <Card className="shadow-sm">
                <Timeline>
                  {ticket.activities.map((activity) => (
                    <Timeline.Item
                      key={activity.id}
                      color={
                        activity.type === "status_change"
                          ? "blue"
                          : activity.type === "assignment"
                          ? "green"
                          : activity.type === "comment"
                          ? "purple"
                          : "gray"
                      }
                    >
                      <div className="flex items-start space-x-3">
                        <Avatar src={activity.user.avatar} size="small">
                          {activity.user.name.charAt(0)}
                        </Avatar>
                        <div className="flex-1">
                          <div className="flex items-center space-x-2 mb-1">
                            <span className="font-medium">
                              {activity.user.name}
                            </span>
                            <span className="text-gray-500">
                              {activity.action}
                            </span>
                          </div>
                          <div className="text-gray-600 mb-1">
                            {activity.details}
                          </div>
                          <div className="text-sm text-gray-400">
                            {new Date(activity.timestamp).toLocaleString(
                              "zh-CN"
                            )}
                          </div>
                        </div>
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              </Card>
            ),
          },
          {
            key: "knowledge",
            label: (
              <span className="flex items-center space-x-2">
                <BookOpen className="w-4 h-4" />
                <span>知识库集成</span>
              </span>
            ),
            children: (
              <KnowledgeIntegration
                ticketId={ticket.id}
                ticketTitle={ticket.title}
                ticketDescription={ticket.description}
                ticketCategory={ticket.category}
              />
            ),
          },
        ]}
      />
    </div>
  );
};

export default EnhancedTicketDetail;
