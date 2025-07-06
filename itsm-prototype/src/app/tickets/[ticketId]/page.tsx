"use client";

import React from "react";
import { TicketDetail } from "../../components/TicketDetail";

// 模拟工单数据
const mockTicketData = {
  id: "INC-00125",
  title: "杭州可用区J的Web服务器CPU使用率超过95%",
  type: "Incident" as const,
  status: "处理中",
  priority: "高",
  assignee: "张三",
  reporter: "系统监控",
  createdAt: "2025-06-28 10:15:23",
  lastUpdate: "2025-06-28 15:30:45",
  description: `监控系统检测到杭州可用区J的Web服务器CPU使用率持续超过95%，已持续15分钟。\n\n影响范围：\n- 用户访问响应时间明显增加\n- 部分API请求超时\n- 可能导致服务不可用\n\n初步分析：\n- 可能是由于流量突增导致\n- 需要检查是否有异常进程占用CPU\n- 考虑扩容或负载均衡调整`,
  category: "性能问题",
  subcategory: "CPU使用率",
  impact: "高",
  urgency: "高",
  resolution: "",
  workNotes: "已联系运维团队进行紧急处理",
  attachments: [
    {
      name: "cpu_monitoring_report.pdf",
      url: "/files/cpu_report.pdf",
      size: "2.3MB",
    },
    { name: "server_logs.txt", url: "/files/server_logs.txt", size: "856KB" },
  ],
};

// 模拟工作流数据
const mockWorkflow = [
  {
    step: "事件创建",
    status: "completed" as const,
    assignee: "系统监控",
    completedAt: "2025-06-28 10:15:23",
    comments: "自动创建事件",
  },
  {
    step: "初步分析",
    status: "completed" as const,
    assignee: "一线支持",
    completedAt: "2025-06-28 10:30:15",
    comments: "确认为CPU使用率异常",
  },
  {
    step: "分配处理",
    status: "current" as const,
    assignee: "张三",
    comments: "正在进行深入分析和处理",
  },
  {
    step: "解决验证",
    status: "pending" as const,
    assignee: "待分配",
  },
  {
    step: "关闭事件",
    status: "pending" as const,
    assignee: "待分配",
  },
];

// 模拟日志数据
const mockLogs = [
  {
    id: "1",
    timestamp: "2025-06-28 15:30:45",
    user: "张三",
    action: "更新工作备注",
    details: "已开始CPU使用率分析，初步怀疑是某个批处理任务导致",
    type: "user" as const,
  },
  {
    id: "2",
    timestamp: "2025-06-28 14:45:20",
    user: "系统",
    action: "状态变更",
    details: '状态从"已分配"变更为"处理中"',
    type: "system" as const,
  },
  {
    id: "3",
    timestamp: "2025-06-28 10:30:15",
    user: "李四",
    action: "分配工单",
    details: "工单已分配给张三处理",
    type: "user" as const,
  },
  {
    id: "4",
    timestamp: "2025-06-28 10:15:23",
    user: "系统监控",
    action: "创建事件",
    details: "自动检测到CPU使用率异常，创建事件",
    type: "system" as const,
  },
];

export default function TicketDetailPage({
  params,
}: {
  params: { ticketId: string };
}) {
  const handleApprove = () => {
    console.log("批准工单:", params.ticketId);
    // 这里实现批准逻辑
  };

  const handleReject = () => {
    console.log("拒绝工单:", params.ticketId);
    // 这里实现拒绝逻辑
  };

  const handleAssign = (assignee: string) => {
    console.log("分配工单给:", assignee);
    // 这里实现分配逻辑
  };

  const handleUpdate = (updates: any) => {
    console.log("更新工单:", updates);
    // 这里实现更新逻辑
  };

  return (
    <TicketDetail
      ticket={mockTicketData}
      workflow={mockWorkflow}
      logs={mockLogs}
      onApprove={handleApprove}
      onReject={handleReject}
      onAssign={handleAssign}
      onUpdate={handleUpdate}
      canApprove={false} // 根据用户权限设置
      canEdit={true} // 根据用户权限设置
    />
  );
}
