"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Space,
  Timeline,
  Upload,
  Input,
  Avatar,
  List,
  Divider,
  Modal,
  Form,
  Select,
  DatePicker,
  message,
  Tooltip,
  Badge,
  Tabs,
  Progress,
} from "antd";
import {
  User,
  Clock,
  MessageSquare,
  Paperclip,
  Edit,
  Eye,
  History,
  Link,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Plus,
  Send,
  Download,
  Delete,
  Share2,
  Copy,
  BookOpen,
  Workflow,
  Calendar,
  MapPin,
  Phone,
  Mail,
} from "lucide-react";
import { LoadingEmptyError } from "../ui/LoadingEmptyError";
import { LoadingSkeleton } from "../ui/LoadingSkeleton";

const { TextArea } = Input;
const { TabPane } = Tabs;

interface TicketComment {
  id: number;
  content: string;
  author: {
    id: number;
    name: string;
    avatar?: string;
    role: string;
  };
  createdAt: string;
  isInternal: boolean;
  attachments?: Array<{
    id: number;
    name: string;
    size: number;
    url: string;
  }>;
}

interface TicketAttachment {
  id: number;
  name: string;
  size: number;
  type: string;
  url: string;
  uploadedBy: string;
  uploadedAt: string;
}

interface TicketActivity {
  id: number;
  type: 'status_change' | 'assignment' | 'comment' | 'attachment' | 'priority_change' | 'category_change';
  description: string;
  operator: string;
  timestamp: string;
  details?: any;
}

interface TicketDetailProps {
  ticketId: number;
  onRefresh?: () => void;
}

export const TicketDetail: React.FC<TicketDetailProps> = ({
  ticketId,
  onRefresh
}) => {
  const [loading, setLoading] = useState(true);
  const [ticket, setTicket] = useState<any>(null);
  const [comments, setComments] = useState<TicketComment[]>([]);
  const [attachments, setAttachments] = useState<TicketAttachment[]>([]);
  const [activities, setActivities] = useState<TicketActivity[]>([]);
  const [commentModalVisible, setCommentModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [newComment, setNewComment] = useState("");
  const [commentForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // 模拟加载工单详情
  useEffect(() => {
    loadTicketDetail();
  }, [ticketId]);

  const loadTicketDetail = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // 模拟工单数据
      const mockTicket = {
        id: ticketId,
        title: "网络连接异常 - 无法访问内网资源",
        description: "用户反映无法访问内网文件服务器，ping测试显示网络不通。需要检查网络配置和防火墙设置。",
        status: "in_progress",
        priority: "high",
        category: "网络故障",
        subcategory: "连接问题",
        assignee: {
          id: 1,
          name: "张三",
          email: "zhangsan@company.com",
          phone: "13800138000",
          department: "IT运维部"
        },
        requester: {
          id: 2,
          name: "李四",
          email: "lisi@company.com",
          phone: "13900139000",
          department: "销售部"
        },
        createdAt: "2024-01-15 09:30:00",
        updatedAt: "2024-01-15 14:20:00",
        dueDate: "2024-01-16 18:00:00",
        sla: {
          target: "4小时",
          remaining: "2小时30分钟",
          status: "normal" // normal, warning, breached
        },
        tags: ["网络", "紧急", "影响业务"],
        location: "总部大楼3楼",
        impact: "high",
        urgency: "high"
      };

      // 模拟评论数据
      const mockComments: TicketComment[] = [
        {
          id: 1,
          content: "已收到工单，正在检查网络配置",
          author: { id: 1, name: "张三", role: "处理人" },
          createdAt: "2024-01-15 10:00:00",
          isInternal: false
        },
        {
          id: 2,
          content: "内部备注：检查防火墙规则，可能是新部署的应用导致",
          author: { id: 1, name: "张三", role: "处理人" },
          createdAt: "2024-01-15 10:15:00",
          isInternal: true
        }
      ];

      // 模拟附件数据
      const mockAttachments: TicketAttachment[] = [
        {
          id: 1,
          name: "网络配置截图.png",
          size: 1024000,
          type: "image/png",
          url: "#",
          uploadedBy: "张三",
          uploadedAt: "2024-01-15 10:30:00"
        }
      ];

      // 模拟活动数据
      const mockActivities: TicketActivity[] = [
        {
          id: 1,
          type: "status_change",
          description: "工单状态从'新建'变更为'处理中'",
          operator: "张三",
          timestamp: "2024-01-15 10:00:00"
        },
        {
          id: 2,
          type: "assignment",
          description: "工单分配给张三",
          operator: "系统",
          timestamp: "2024-01-15 09:35:00"
        }
      ];

      setTicket(mockTicket);
      setComments(mockComments);
      setAttachments(mockAttachments);
      setActivities(mockActivities);
    } catch (error) {
      message.error("加载工单详情失败");
    } finally {
      setLoading(false);
    }
  };

  const handleAddComment = async () => {
    try {
      const values = await commentForm.validateFields();
      
      const newCommentData: TicketComment = {
        id: Date.now(),
        content: values.content,
        author: {
          id: 1,
          name: "张三",
          role: "处理人"
        },
        createdAt: new Date().toLocaleString(),
        isInternal: values.isInternal || false
      };

      setComments(prev => [newCommentData, ...prev]);
      setCommentModalVisible(false);
      commentForm.resetFields();
      message.success("评论添加成功");
    } catch (error) {
      console.error("添加评论失败:", error);
    }
  };

  const handleStatusChange = async (newStatus: string) => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      
      setTicket(prev => ({
        ...prev,
        status: newStatus,
        updatedAt: new Date().toLocaleString()
      }));

      // 添加活动记录
      const newActivity: TicketActivity = {
        id: Date.now(),
        type: "status_change",
        description: `工单状态变更为'${newStatus}'`,
        operator: "张三",
        timestamp: new Date().toLocaleString()
      };

      setActivities(prev => [newActivity, ...prev]);
      message.success("状态更新成功");
    } catch (error) {
      message.error("状态更新失败");
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'new': return 'blue';
      case 'in_progress': return 'orange';
      case 'pending': return 'yellow';
      case 'resolved': return 'green';
      case 'closed': return 'gray';
      case 'cancelled': return 'red';
      default: return 'default';
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'low': return 'blue';
      case 'medium': return 'orange';
      case 'high': return 'red';
      case 'critical': return 'purple';
      default: return 'default';
    }
  };

  const getSLAStatusColor = (status: string) => {
    switch (status) {
      case 'normal': return 'green';
      case 'warning': return 'orange';
      case 'breached': return 'red';
      default: return 'default';
    }
  };

  if (loading) {
    return <LoadingSkeleton type="card" rows={8} />;
  }

  if (!ticket) {
    return (
      <LoadingEmptyError
        state="error"
        error={{
          title: "工单不存在",
          description: "无法找到指定的工单信息",
          actionText: "返回列表",
          onAction: () => window.history.back()
        }}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* 工单头部信息 */}
      <Card>
        <div className="flex justify-between items-start mb-4">
          <div className="flex-1">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">
              {ticket.title}
            </h1>
            <div className="flex items-center gap-3 text-sm text-gray-500">
              <span>工单号: #{ticket.id}</span>
              <span>创建时间: {ticket.createdAt}</span>
              <span>更新时间: {ticket.updatedAt}</span>
            </div>
          </div>
          <Space>
            <Button icon={<Edit size={16} />} onClick={() => setEditModalVisible(true)}>
              编辑
            </Button>
            <Button icon={<Share2 size={16} />}>
              分享
            </Button>
            <Button icon={<Copy size={16} />}>
              复制
            </Button>
          </Space>
        </div>

        {/* 状态和优先级 */}
        <div className="flex items-center gap-4 mb-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-600">状态:</span>
            <Tag color={getStatusColor(ticket.status)} size="large">
              {ticket.status === 'new' && '新建'}
              {ticket.status === 'in_progress' && '处理中'}
              {ticket.status === 'pending' && '等待中'}
              {ticket.status === 'resolved' && '已解决'}
              {ticket.status === 'closed' && '已关闭'}
              {ticket.status === 'cancelled' && '已取消'}
            </Tag>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-600">优先级:</span>
            <Tag color={getPriorityColor(ticket.priority)} size="large">
              {ticket.priority === 'low' && '低'}
              {ticket.priority === 'medium' && '中'}
              {ticket.priority === 'high' && '高'}
              {ticket.priority === 'critical' && '紧急'}
            </Tag>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-600">分类:</span>
            <Tag color="blue">{ticket.category} / {ticket.subcategory}</Tag>
          </div>
        </div>

        {/* SLA 信息 */}
        <div className="bg-gray-50 p-4 rounded-lg">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-700">SLA 状态</span>
            <Tag color={getSLAStatusColor(ticket.sla.status)}>
              {ticket.sla.status === 'normal' && '正常'}
              {ticket.sla.status === 'warning' && '预警'}
              {ticket.sla.status === 'breached' && '超时'}
            </Tag>
          </div>
          <div className="flex items-center gap-4 text-sm">
            <span>目标: {ticket.sla.target}</span>
            <span>剩余: {ticket.sla.remaining}</span>
            <Progress 
              percent={60} 
              size="small" 
              status={ticket.sla.status === 'breached' ? 'exception' : 'normal'}
            />
          </div>
        </div>
      </Card>

      {/* 主要内容区域 */}
      <Tabs defaultActiveKey="overview" size="large">
        <TabPane tab="概览" key="overview">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* 基本信息 */}
            <Card title="基本信息" className="lg:col-span-2">
              <Descriptions column={2} bordered>
                <Descriptions.Item label="工单标题" span={2}>
                  {ticket.title}
                </Descriptions.Item>
                <Descriptions.Item label="描述" span={2}>
                  {ticket.description}
                </Descriptions.Item>
                <Descriptions.Item label="影响程度">
                  <Tag color={ticket.impact === 'high' ? 'red' : 'orange'}>
                    {ticket.impact === 'high' ? '高' : '中'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="紧急程度">
                  <Tag color={ticket.urgency === 'high' ? 'red' : 'orange'}>
                    {ticket.urgency === 'high' ? '高' : '中'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="位置">
                  <span className="flex items-center gap-1">
                    <MapPin size={14} />
                    {ticket.location}
                  </span>
                </Descriptions.Item>
                <Descriptions.Item label="到期时间">
                  <span className="flex items-center gap-1">
                    <Calendar size={14} />
                    {ticket.dueDate}
                  </span>
                </Descriptions.Item>
                <Descriptions.Item label="标签" span={2}>
                  <Space>
                    {ticket.tags.map((tag: string) => (
                      <Tag key={tag} color="blue">{tag}</Tag>
                    ))}
                  </Space>
                </Descriptions.Item>
              </Descriptions>
            </Card>

            {/* 相关人员 */}
            <Card title="相关人员">
              <div className="space-y-4">
                {/* 处理人 */}
                <div>
                  <h4 className="font-medium text-gray-700 mb-2">处理人</h4>
                  <div className="flex items-center gap-3 p-3 bg-blue-50 rounded-lg">
                    <Avatar size="large" style={{ backgroundColor: '#1890ff' }}>
                      {ticket.assignee.name[0]}
                    </Avatar>
                    <div>
                      <div className="font-medium">{ticket.assignee.name}</div>
                      <div className="text-sm text-gray-500">{ticket.assignee.department}</div>
                      <div className="text-sm text-gray-500">{ticket.assignee.email}</div>
                    </div>
                  </div>
                </div>

                {/* 申请人 */}
                <div>
                  <h4 className="font-medium text-gray-700 mb-2">申请人</h4>
                  <div className="flex items-center gap-3 p-3 bg-gray-50 rounded-lg">
                    <Avatar size="large" style={{ backgroundColor: '#52c41a' }}>
                      {ticket.requester.name[0]}
                    </Avatar>
                    <div>
                      <div className="font-medium">{ticket.requester.name}</div>
                      <div className="text-sm text-gray-500">{ticket.requester.department}</div>
                      <div className="text-sm text-gray-500">{ticket.requester.email}</div>
                    </div>
                  </div>
                </div>
              </div>
            </Card>
          </div>
        </TabPane>

        {/* 评论和沟通 */}
        <TabPane tab="沟通记录" key="comments">
          <Card>
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-medium">沟通记录</h3>
              <Button 
                type="primary" 
                icon={<Plus size={16} />}
                onClick={() => setCommentModalVisible(true)}
              >
                添加评论
              </Button>
            </div>

            <div className="space-y-4">
              {comments.map((comment) => (
                <div key={comment.id} className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <Avatar size="small">
                      {comment.author.name[0]}
                    </Avatar>
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <span className="font-medium">{comment.author.name}</span>
                        <Tag size="small" color={comment.isInternal ? 'orange' : 'blue'}>
                          {comment.isInternal ? '内部备注' : '公开'}
                        </Tag>
                        <span className="text-sm text-gray-500">{comment.createdAt}</span>
                      </div>
                      <div className="text-gray-700 mb-2">{comment.content}</div>
                      {comment.attachments && comment.attachments.length > 0 && (
                        <div className="flex gap-2">
                          {comment.attachments.map((attachment) => (
                            <Button key={attachment.id} size="small" icon={<Download size={14} />}>
                              {attachment.name}
                            </Button>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </TabPane>

        {/* 附件管理 */}
        <TabPane tab="附件" key="attachments">
          <Card>
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-medium">附件管理</h3>
              <Upload>
                <Button icon={<Plus size={16} />}>上传附件</Button>
              </Upload>
            </div>

            <List
              dataSource={attachments}
              renderItem={(attachment) => (
                <List.Item
                  actions={[
                    <Button key="download" size="small" icon={<Download size={14} />}>
                      下载
                    </Button>,
                    <Button key="delete" size="small" danger icon={<Delete size={14} />}>
                      删除
                    </Button>
                  ]}
                >
                  <List.Item.Meta
                    avatar={<Paperclip size={20} className="text-gray-400" />}
                    title={attachment.name}
                    description={
                      <div className="text-sm text-gray-500">
                        <span>{attachment.uploadedBy}</span>
                        <span className="mx-2">•</span>
                        <span>{attachment.uploadedAt}</span>
                        <span className="mx-2">•</span>
                        <span>{(attachment.size / 1024).toFixed(1)} KB</span>
                      </div>
                    }
                  />
                </List.Item>
              )}
            />
          </Card>
        </TabPane>

        {/* 活动历史 */}
        <TabPane tab="活动历史" key="history">
          <Card>
            <h3 className="text-lg font-medium mb-4">活动历史</h3>
            <Timeline>
              {activities.map((activity) => (
                <Timeline.Item
                  key={activity.id}
                  color={
                    activity.type === 'status_change' ? 'blue' :
                    activity.type === 'assignment' ? 'green' :
                    activity.type === 'comment' ? 'orange' : 'gray'
                  }
                >
                  <div className="mb-2">
                    <div className="font-medium">{activity.description}</div>
                    <div className="text-sm text-gray-500">
                      {activity.operator} • {activity.timestamp}
                    </div>
                  </div>
                </Timeline.Item>
              ))}
            </Timeline>
          </Card>
        </TabPane>

        {/* 相关工单 */}
        <TabPane tab="相关工单" key="related">
          <Card>
            <h3 className="text-lg font-medium mb-4">相关工单</h3>
            <div className="text-center text-gray-500 py-8">
              <Link size={48} className="mx-auto mb-4 text-gray-300" />
              <p>暂无相关工单</p>
              <Button type="link" className="mt-2">
                查找相关工单
              </Button>
            </div>
          </Card>
        </TabPane>
      </Tabs>

      {/* 添加评论模态框 */}
      <Modal
        title="添加评论"
        open={commentModalVisible}
        onOk={handleAddComment}
        onCancel={() => setCommentModalVisible(false)}
        okText="添加"
        cancelText="取消"
      >
        <Form form={commentForm} layout="vertical">
          <Form.Item
            name="content"
            label="评论内容"
            rules={[{ required: true, message: '请输入评论内容' }]}
          >
            <TextArea rows={4} placeholder="请输入评论内容..." />
          </Form.Item>
          <Form.Item name="isInternal" valuePropName="checked">
            <div className="flex items-center gap-2">
              <input type="checkbox" id="isInternal" />
              <label htmlFor="isInternal">设为内部备注</label>
            </div>
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑工单模态框 */}
      <Modal
        title="编辑工单"
        open={editModalVisible}
        onOk={() => {
          // 处理编辑逻辑
          setEditModalVisible(false);
          message.success("工单更新成功");
        }}
        onCancel={() => setEditModalVisible(false)}
        okText="保存"
        cancelText="取消"
        width={800}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item label="工单标题" name="title">
            <Input placeholder="请输入工单标题" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <TextArea rows={4} placeholder="请输入工单描述" />
          </Form.Item>
          <div className="grid grid-cols-2 gap-4">
            <Form.Item label="优先级" name="priority">
              <Select placeholder="请选择优先级">
                <Select.Option value="low">低</Select.Option>
                <Select.Option value="medium">中</Select.Option>
                <Select.Option value="high">高</Select.Option>
                <Select.Option value="critical">紧急</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item label="分类" name="category">
              <Select placeholder="请选择分类">
                <Select.Option value="network">网络故障</Select.Option>
                <Select.Option value="hardware">硬件故障</Select.Option>
                <Select.Option value="software">软件故障</Select.Option>
              </Select>
            </Form.Item>
          </div>
          <Form.Item label="到期时间" name="dueDate">
            <DatePicker showTime placeholder="请选择到期时间" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TicketDetail;
