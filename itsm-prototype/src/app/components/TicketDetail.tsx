"use client";

import {
  CheckCircle,
  Clock,
  XCircle,
  Edit,
  FileText,
  Save,
  ChevronDown,
  X,
  ArrowRight,
  ChevronUp,
} from "lucide-react";

import React, { useState } from "react";
// 工单详情组件的 Props 接口
interface TicketDetailProps {
  ticket: {
    id: string;
    title: string;
    type: "Incident" | "Problem" | "Change" | "Service Request";
    status: string;
    priority: string;
    assignee: string;
    reporter: string;
    createdAt: string;
    lastUpdate: string;
    description: string;
    category?: string;
    subcategory?: string;
    impact?: string;
    urgency?: string;
    resolution?: string;
    workNotes?: string;
    attachments?: Array<{ name: string; url: string; size: string }>;
  };
  workflow?: Array<{
    step: string;
    status: "completed" | "current" | "pending";
    assignee?: string;
    completedAt?: string;
    comments?: string;
  }>;
  logs?: Array<{
    id: string;
    timestamp: string;
    user: string;
    action: string;
    details: string;
    type: "system" | "user" | "approval";
  }>;
  onApprove?: () => void;
  onReject?: () => void;
  onAssign?: (assignee: string) => void;
  onUpdate?: (updates: unknown) => void;
  canApprove?: boolean;
  canEdit?: boolean;
}

// 获取优先级配置
const getPriorityConfig = (priority: string) => {
  const configs = {
    紧急: {
      color: "red",
      bgColor: "bg-red-100",
      textColor: "text-red-800",
      borderColor: "border-red-300",
    },
    高: {
      color: "orange",
      bgColor: "bg-orange-100",
      textColor: "text-orange-800",
      borderColor: "border-orange-300",
    },
    中: {
      color: "yellow",
      bgColor: "bg-yellow-100",
      textColor: "text-yellow-800",
      borderColor: "border-yellow-300",
    },
    低: {
      color: "blue",
      bgColor: "bg-blue-100",
      textColor: "text-blue-800",
      borderColor: "border-blue-300",
    },
  };
  return configs[priority] || configs["中"];
};

// 获取状态配置
const getStatusConfig = (status: string) => {
  const configs = {
    处理中: { bgColor: "bg-blue-100", textColor: "text-blue-800" },
    已分配: { bgColor: "bg-purple-100", textColor: "text-purple-800" },
    已解决: { bgColor: "bg-green-100", textColor: "text-green-800" },
    已关闭: { bgColor: "bg-gray-200", textColor: "text-gray-800" },
    待审批: { bgColor: "bg-yellow-100", textColor: "text-yellow-800" },
    已批准: { bgColor: "bg-green-100", textColor: "text-green-800" },
    实施中: { bgColor: "bg-blue-100", textColor: "text-blue-800" },
    已完成: { bgColor: "bg-gray-200", textColor: "text-gray-800" },
    已拒绝: { bgColor: "bg-red-100", textColor: "text-red-800" },
    调查中: { bgColor: "bg-blue-100", textColor: "text-blue-800" },
    已知错误: { bgColor: "bg-purple-100", textColor: "text-purple-800" },
  };
  return (
    configs[status] || { bgColor: "bg-gray-100", textColor: "text-gray-800" }
  );
};

// 获取工单类型图标
const getTypeIcon = (type: string) => {
  const icons = {
    Incident: AlertTriangle,
    Problem: Cpu,
    Change: GitMerge,
    "Service Request": User,
  };
  return icons[type] || FileText;
};

export const TicketDetail: React.FC<TicketDetailProps> = ({
  ticket,
  workflow = [],
  logs = [],
  onApprove,
  onReject,
  onAssign,
  onUpdate,
  canApprove = false,
  canEdit = false,
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editedTicket, setEditedTicket] = useState(ticket);
  const [showLogs, setShowLogs] = useState(true);
  const [newComment, setNewComment] = useState("");
  const [assigneeInput, setAssigneeInput] = useState("");

  const priorityConfig = getPriorityConfig(ticket.priority);
  const statusConfig = getStatusConfig(ticket.status);
  const TypeIcon = getTypeIcon(ticket.type);

  const handleSave = () => {
    onUpdate?.(editedTicket);
    setIsEditing(false);
  };

  const handleCancel = () => {
    setEditedTicket(ticket);
    setIsEditing(false);
  };

  const handleAssign = () => {
    if (assigneeInput.trim()) {
      onAssign?.(assigneeInput.trim());
      setAssigneeInput("");
    }
  };

  return (
    <div className="max-w-7xl mx-auto p-6 bg-gray-50 min-h-screen">
      {/* 头部信息 */}
      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center space-x-3">
            <TypeIcon className="w-8 h-8 text-blue-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">
                {ticket.title}
              </h1>
              <p className="text-gray-600">
                {ticket.type} #{ticket.id}
              </p>
            </div>
          </div>
          <div className="flex items-center space-x-3">
            {canEdit && (
              <button
                onClick={() => setIsEditing(!isEditing)}
                className="flex items-center px-3 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
              >
                {isEditing ? (
                  <X className="w-4 h-4 mr-1" />
                ) : (
                  <Edit className="w-4 h-4 mr-1" />
                )}
                {isEditing ? "取消" : "编辑"}
              </button>
            )}
            {isEditing && (
              <button
                onClick={handleSave}
                className="flex items-center px-3 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 transition-colors"
              >
                <Save className="w-4 h-4 mr-1" />
                保存
              </button>
            )}
          </div>
        </div>

        {/* 状态和优先级标签 */}
        <div className="flex items-center space-x-4 mb-6">
          <span
            className={`px-3 py-1 text-sm font-medium rounded-full ${statusConfig.bgColor} ${statusConfig.textColor}`}
          >
            {ticket.status}
          </span>
          <span
            className={`px-3 py-1 text-sm font-semibold rounded-full border ${priorityConfig.bgColor} ${priorityConfig.textColor} ${priorityConfig.borderColor}`}
          >
            优先级: {ticket.priority}
          </span>
        </div>

        {/* 基本信息网格 */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              负责人
            </label>
            {isEditing ? (
              <input
                type="text"
                value={editedTicket.assignee}
                onChange={(e) =>
                  setEditedTicket({ ...editedTicket, assignee: e.target.value })
                }
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            ) : (
              <p className="text-gray-900">{ticket.assignee}</p>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              报告人
            </label>
            <p className="text-gray-900">{ticket.reporter}</p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              创建时间
            </label>
            <p className="text-gray-900">{ticket.createdAt}</p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              最后更新
            </label>
            <p className="text-gray-900">{ticket.lastUpdate}</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 左侧：详细信息 */}
        <div className="lg:col-span-2 space-y-6">
          {/* 描述 */}
          <div className="bg-white rounded-lg shadow-md p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              问题描述
            </h3>
            {isEditing ? (
              <textarea
                value={editedTicket.description}
                onChange={(e) =>
                  setEditedTicket({
                    ...editedTicket,
                    description: e.target.value,
                  })
                }
                rows={4}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            ) : (
              <p className="text-gray-700 whitespace-pre-wrap">
                {ticket.description}
              </p>
            )}
          </div>

          {/* 工作流程图 */}
          {workflow.length > 0 && (
            <div className="bg-white rounded-lg shadow-md p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">
                处理流程
              </h3>
              <div className="space-y-4">
                {workflow.map((step, index) => (
                  <div key={index} className="flex items-center space-x-4">
                    <div
                      className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${
                        step.status === "completed"
                          ? "bg-green-100 text-green-600"
                          : step.status === "current"
                          ? "bg-blue-100 text-blue-600"
                          : "bg-gray-100 text-gray-400"
                      }`}
                    >
                      {step.status === "completed" ? (
                        <CheckCircle className="w-5 h-5" />
                      ) : step.status === "current" ? (
                        <Clock className="w-5 h-5" />
                      ) : (
                        <div className="w-3 h-3 rounded-full bg-current" />
                      )}
                    </div>
                    <div className="flex-1">
                      <p
                        className={`font-medium ${
                          step.status === "completed"
                            ? "text-green-900"
                            : step.status === "current"
                            ? "text-blue-900"
                            : "text-gray-500"
                        }`}
                      >
                        {step.step}
                      </p>
                      {step.assignee && (
                        <p className="text-sm text-gray-600">
                          负责人: {step.assignee}
                        </p>
                      )}
                      {step.completedAt && (
                        <p className="text-sm text-gray-500">
                          完成时间: {step.completedAt}
                        </p>
                      )}
                      {step.comments && (
                        <p className="text-sm text-gray-600 mt-1">
                          {step.comments}
                        </p>
                      )}
                    </div>
                    {index < workflow.length - 1 && (
                      <ArrowRight className="w-4 h-4 text-gray-400" />
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 解决方案 */}
          {ticket.resolution && (
            <div className="bg-white rounded-lg shadow-md p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">
                解决方案
              </h3>
              {isEditing ? (
                <textarea
                  value={editedTicket.resolution}
                  onChange={(e) =>
                    setEditedTicket({
                      ...editedTicket,
                      resolution: e.target.value,
                    })
                  }
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              ) : (
                <p className="text-gray-700 whitespace-pre-wrap">
                  {ticket.resolution}
                </p>
              )}
            </div>
          )}

          {/* 附件 */}
          {ticket.attachments && ticket.attachments.length > 0 && (
            <div className="bg-white rounded-lg shadow-md p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">附件</h3>
              <div className="space-y-2">
                {ticket.attachments.map((file, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between p-3 bg-gray-50 rounded-md"
                  >
                    <div className="flex items-center space-x-3">
                      <FileText className="w-5 h-5 text-gray-500" />
                      <span className="text-sm font-medium text-gray-900">
                        {file.name}
                      </span>
                      <span className="text-sm text-gray-500">
                        ({file.size})
                      </span>
                    </div>
                    <a
                      href={file.url}
                      className="text-blue-600 hover:text-blue-800 text-sm font-medium"
                      download
                    >
                      下载
                    </a>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* 右侧：操作和日志 */}
        <div className="space-y-6">
          {/* 审批操作 */}
          {canApprove && ticket.status === "待审批" && (
            <div className="bg-white rounded-lg shadow-md p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">
                审批操作
              </h3>
              <div className="space-y-3">
                <button
                  onClick={onApprove}
                  className="w-full flex items-center justify-center px-4 py-2 bg-green-600 text-white font-medium rounded-md hover:bg-green-700 transition-colors"
                >
                  <CheckCircle className="w-4 h-4 mr-2" />
                  批准
                </button>
                <button
                  onClick={onReject}
                  className="w-full flex items-center justify-center px-4 py-2 bg-red-600 text-white font-medium rounded-md hover:bg-red-700 transition-colors"
                >
                  <XCircle className="w-4 h-4 mr-2" />
                  拒绝
                </button>
              </div>
            </div>
          )}

          {/* 分配操作 */}
          <div className="bg-white rounded-lg shadow-md p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              分配工单
            </h3>
            <div className="space-y-3">
              <input
                type="text"
                value={assigneeInput}
                onChange={(e) => setAssigneeInput(e.target.value)}
                placeholder="输入负责人姓名"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                onClick={handleAssign}
                disabled={!assigneeInput.trim()}
                className="w-full px-4 py-2 bg-blue-600 text-white font-medium rounded-md hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
              >
                分配
              </button>
            </div>
          </div>

          {/* 活动日志 */}
          <div className="bg-white rounded-lg shadow-md p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">活动日志</h3>
              <button
                onClick={() => setShowLogs(!showLogs)}
                className="text-gray-500 hover:text-gray-700"
              >
                {showLogs ? (
                  <ChevronUp className="w-5 h-5" />
                ) : (
                  <ChevronDown className="w-5 h-5" />
                )}
              </button>
            </div>

            {showLogs && (
              <div className="space-y-4 max-h-96 overflow-y-auto">
                {logs.map((log) => (
                  <div
                    key={log.id}
                    className="border-l-2 border-gray-200 pl-4 pb-4"
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm font-medium text-gray-900">
                        {log.user}
                      </span>
                      <span className="text-xs text-gray-500">
                        {log.timestamp}
                      </span>
                    </div>
                    <p className="text-sm text-gray-700 mb-1">{log.action}</p>
                    <p className="text-xs text-gray-600">{log.details}</p>
                  </div>
                ))}

                {logs.length === 0 && (
                  <p className="text-gray-500 text-center py-4">暂无活动记录</p>
                )}
              </div>
            )}

            {/* 添加评论 */}
            <div className="mt-4 pt-4 border-t border-gray-200">
              <textarea
                value={newComment}
                onChange={(e) => setNewComment(e.target.value)}
                placeholder="添加工作备注..."
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              />
              <button
                onClick={() => {
                  if (newComment.trim()) {
                    // 这里可以调用添加评论的回调
                    console.log("添加评论:", newComment);
                    setNewComment("");
                  }
                }}
                disabled={!newComment.trim()}
                className="mt-2 w-full px-3 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
              >
                添加备注
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
