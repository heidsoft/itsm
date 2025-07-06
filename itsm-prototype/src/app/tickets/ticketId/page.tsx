"use client";

import React, { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { TicketApi } from "../../lib/ticket-api";
import { Ticket } from "../../lib/api-config";
import { TicketDetail } from "../../components/TicketDetail";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";

const TicketDetailPage: React.FC = () => {
  const params = useParams();
  const router = useRouter();
  const ticketId = parseInt(params.ticketId as string);

  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 获取工单详情
  const fetchTicket = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await TicketApi.getTicket(ticketId);

      if (response.code === 0) {
        setTicket(response.data);
      } else {
        setError(response.message || "获取工单详情失败");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "网络错误");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (ticketId) {
      fetchTicket();
    }
  }, [ticketId]);

  // 处理审批
  const handleApprove = async () => {
    try {
      const response = await TicketApi.approveTicket(ticketId, {
        action: "approve",
        comment: "审批通过",
        step_name: "审批",
      });

      if (response.code === 0) {
        alert("审批成功");
        fetchTicket(); // 刷新数据
      } else {
        alert(response.message || "审批失败");
      }
    } catch (error) {
      alert(error instanceof Error ? error.message : "网络错误");
    }
  };

  // 处理拒绝
  const handleReject = async () => {
    try {
      const response = await TicketApi.approveTicket(ticketId, {
        action: "reject",
        comment: "审批拒绝",
        step_name: "审批",
      });

      if (response.code === 0) {
        alert("已拒绝");
        fetchTicket(); // 刷新数据
      } else {
        alert(response.message || "操作失败");
      }
    } catch (error) {
      alert(error instanceof Error ? error.message : "网络错误");
    }
  };

  // 处理分配
  const handleAssign = async (assignee: string) => {
    try {
      const response = await TicketApi.updateTicket(ticketId, {
        assignee_id: parseInt(assignee), // 这里需要根据实际情况处理
      });

      if (response.code === 0) {
        alert("分配成功");
        fetchTicket(); // 刷新数据
      } else {
        alert(response.message || "分配失败");
      }
    } catch (error) {
      alert(error instanceof Error ? error.message : "网络错误");
    }
  };

  // 处理更新
  const handleUpdate = async (updates: any) => {
    try {
      const response = await TicketApi.updateTicket(ticketId, updates);

      if (response.code === 0) {
        alert("更新成功");
        fetchTicket(); // 刷新数据
      } else {
        alert(response.message || "更新失败");
      }
    } catch (error) {
      alert(error instanceof Error ? error.message : "网络错误");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="p-6">
        <div className="text-center text-gray-500">工单不存在</div>
      </div>
    );
  }

  // 转换数据格式以适配TicketDetail组件
  const ticketDetailData = {
    id: ticket.id.toString(),
    title: ticket.title,
    type: "Service Request" as const,
    status: ticket.status,
    priority: ticket.priority,
    assignee: ticket.assignee?.name || "未分配",
    reporter: ticket.requester?.name || "未知",
    createdAt: new Date(ticket.created_at).toLocaleString(),
    lastUpdate: new Date(ticket.updated_at).toLocaleString(),
    description: ticket.description,
  };

  return (
    <div>
      {/* 头部导航 */}
      <div className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center">
          <Link
            href="/tickets"
            className="flex items-center text-gray-600 hover:text-gray-900 mr-4"
          >
            <ArrowLeft className="w-4 h-4 mr-1" />
            返回工单列表
          </Link>
        </div>
      </div>

      {/* 工单详情 */}
      <TicketDetail
        ticket={ticketDetailData}
        workflow={[]} // 可以根据需要添加工作流数据
        logs={[]} // 可以根据需要添加日志数据
        onApprove={handleApprove}
        onReject={handleReject}
        onAssign={handleAssign}
        onUpdate={handleUpdate}
        canApprove={true}
        canEdit={true}
      />
    </div>
  );
};

export default TicketDetailPage;
