"use client";

import { CheckCircle, Search, XCircle, Cpu, BookOpen, MessageSquare, PlusCircle, FileText, Zap, ArrowLeft, PlayCircle, PauseCircle, LinkIcon, Sparkles } from "lucide-react";

import React, { useState, useEffect } from "react";
import { aiSearchKB, aiSimilarIncidents, aiSummarize } from "../../lib/ai-api";
import { IncidentAPI } from "../../lib/incident-api";
import { AIFeedback } from "../../components/AIFeedback";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";

// 配置项信息接口
interface ConfigurationItem {
  id: number;
  name: string;
  type: string;
  status: string;
  description?: string;
}

// 事件详情接口
interface IncidentDetail {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  source: string;
  type: string;
  incident_number: string;
  is_major_incident: boolean;
  reporter?: {
    id: number;
    name: string;
  };
  assignee?: {
    id: number;
    name: string;
  };
  configuration_item_id?: number;
  configuration_item?: ConfigurationItem;
  created_at: string;
  updated_at: string;
  detected_at?: string;
  confirmed_at?: string;
  resolved_at?: string;
  closed_at?: string;
  // 处理日志和评论
  logs?: string[];
  comments?: Array<{
    author: string;
    timestamp: string;
    text: string;
  }>;
}

// 模拟数据
const mockIncidentDetail = {
  "INC-00125": {
    title: "杭州可用区J的Web服务器CPU使用率超过95%",
    priority: "高",
    status: "处理中",
    source: "阿里云监控",
    reporter: "system",
    assignee: "张三",
    createdAt: "2025-06-28 10:15:23",
    confirmedAt: "2025-06-28 10:18:00",
    resolvedAt: null,
    isMajorIncident: false,
    description:
      "监控系统检测到实例 i-bp1abcdefg 的CPU使用率在过去15分钟内持续高于95%。可能原因：应用程序内存泄漏或流量突增。需要立即排查。",
    affectedCI: {
      name: "i-bp1abcdefg",
      type: "云服务器",
      link: "/cmdb/ci/i-bp1abcdefg",
      id: "CI-ECS-001",
    },
    logs: [
      "[10:15] 事件创建并自动分配给云资源团队。",
      "[10:18] 张三已确认接收事件。",
      "[10:25] 张三正在登录服务器排查进程... ",
      "[10:35] 发现某应用进程占用大量CPU，已尝试重启服务。",
      "[10:40] CPU使用率回落至正常水平，事件状态更新为已解决。",
    ],
    comments: [
      {
        author: "张三",
        timestamp: "2025-06-28 10:20",
        text: "已收到事件，正在排查中。",
      },
    ],
  },
  "INC-00124": {
    title: "用户报告无法访问CRM系统",
    priority: "高",
    status: "已分配",
    source: "服务台",
    reporter: "李四",
    assignee: "王五",
    createdAt: "2025-06-28 09:45:10",
    confirmedAt: "2025-06-28 09:50:00",
    resolvedAt: null,
    isMajorIncident: false,
    description:
      "用户李四报告无法通过浏览器访问CRM系统，尝试清除缓存和更换浏览器无效。怀疑是网络或应用服务问题。",
    affectedCI: {
      name: "CRM-APP-SERVER",
      type: "应用服务器",
      link: "/cmdb/ci/CRM-APP-SERVER",
      id: "CI-APP-CRM",
    },
    logs: [
      "[09:45] 事件创建并分配给应用支持团队。",
      "[09:50] 王五已确认接收事件，开始排查。",
      "[10:05] 初步判断为CRM应用服务异常，正在尝试重启。",
    ],
    comments: [],
  },
  "INC-00123": {
    title: "检测到可疑的SSH登录尝试 (47.98.x.x)",
    priority: "中",
    status: "处理中",
    source: "安全中心",
    reporter: "system",
    assignee: "赵六",
    createdAt: "2025-06-28 08:30:00",
    confirmedAt: "2025-06-28 08:35:00",
    resolvedAt: null,
    isMajorIncident: true, // 模拟一个重大事件
    description:
      "安全系统检测到来自未知IP地址 (47.98.x.x) 对生产环境服务器的多次SSH登录失败尝试。可能存在暴力破解风险。",
    affectedCI: {
      name: "PROD-BASTION-HOST",
      type: "跳板机",
      link: "/cmdb/ci/PROD-BASTION-HOST",
      id: "CI-PROD-BASTION-HOST",
    },
    logs: [
      "[08:30] 事件创建并自动分配给安全团队。",
      "[08:35] 赵六已确认接收事件，正在分析日志。",
      "[08:45] 已将可疑IP加入防火墙黑名单。",
    ],
    comments: [],
  },
  "INC-00122": {
    title: "生产数据库主备同步延迟",
    priority: "中",
    status: "已解决",
    source: "阿里云监控",
    reporter: "system",
    assignee: "钱七",
    createdAt: "2025-06-27 18:00:00",
    confirmedAt: "2025-06-27 18:05:00",
    resolvedAt: "2025-06-27 18:45:00",
    isMajorIncident: false,
    description:
      "监控系统告警，生产数据库（RDS）主备同步延迟超过阈值（30秒）。可能影响数据一致性。",
    affectedCI: {
      name: "RDS-PROD-DB",
      type: "云数据库",
      link: "/cmdb/ci/RDS-PROD-DB",
      id: "CI-RDS-001",
    },
    logs: [
      "[18:00] 事件创建并自动分配给数据库团队。",
      "[18:05] 钱七已确认接收事件，开始排查。",
      "[18:30] 发现是由于某个大事务导致同步阻塞，已优化SQL并强制同步。",
      "[18:45] 同步延迟恢复正常，事件状态更新为已解决。",
    ],
    comments: [],
  },
  "INC-00121": {
    title: "用户请求重置密码失败",
    priority: "低",
    status: "已关闭",
    source: "服务台",
    reporter: "孙八",
    assignee: "周九",
    createdAt: "2025-06-27 10:00:00",
    confirmedAt: "2025-06-27 10:05:00",
    resolvedAt: "2025-06-27 10:20:00",
    isMajorIncident: false,
    description: "用户孙八通过自助服务门户尝试重置密码失败，请求服务台协助。",
    affectedCI: {
      name: "AD-SERVER-01",
      type: "域控制器",
      link: "/cmdb/ci/AD-SERVER-01",
      id: "CI-AD-SERVER-01",
    },
    logs: [
      "[10:00] 事件创建并分配给IT支持团队。",
      "[10:05] 周九已确认接收事件。",
      "[10:15] 经排查，用户输入旧密码错误次数过多导致账户锁定，已解锁并指导用户正确重置。",
      "[10:20] 事件状态更新为已关闭。",
    ],
    comments: [],
  },
};

// 模拟CMDB数据，用于更新CI状态
const mockCIData = {
  "CI-ECS-001": { status: "运行中" },
  "CI-RDS-001": { status: "运行中" },
  "CI-APP-CRM": { status: "运行中" },
  "CI-PROD-BASTION-HOST": { status: "运行中" },
  "CI-AD-SERVER-01": { status: "运行中" },
};

// 计算时间差（分钟）
const calculateTimeDiffInMinutes = (start: string, end: string): string => {
  if (!start || !end) return "N/A";
  const startTime = new Date(start);
  const endTime = new Date(end);
  const diffInMinutes = Math.round(
    (endTime.getTime() - startTime.getTime()) / (1000 * 60)
  );
  return `${diffInMinutes} 分钟`;
};

// 知识库建议组件
const KnowledgeBaseSuggestion = ({
  title,
  similarity,
}: {
  title: string;
  similarity: number;
}) => (
  <div className="p-3 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors">
    <div className="flex items-start justify-between">
      <div className="flex-1">
        <p className="font-semibold text-gray-800">{title}</p>
        <span className="text-sm text-green-600">
          相似度: {Math.round(similarity * 100)}%
        </span>
      </div>
    </div>
  </div>
);

const IncidentDetailPage = () => {
  const params = useParams();
  const router = useRouter();
  const incidentId = params.incidentId as string;
  const [incident, setIncident] = useState<IncidentDetail | null>(null); // 使用useState来管理事件状态
  const [newComment, setNewComment] = useState("");
  const [aiLoading, setAiLoading] = useState(false);
  const [kbAnswers, setKbAnswers] = useState<
    Array<{ title?: string; score?: number }>
  >([]);
  const [summary, setSummary] = useState<string>("");

  useEffect(() => {
    const fetchIncident = async () => {
      try {
        const incidentData = await IncidentAPI.getIncident(
          parseInt(incidentId)
        );
        setIncident(incidentData);
      } catch (error) {
        console.error("Failed to fetch incident:", error);
        setIncident(null);
      }
    };
    fetchIncident();
  }, [incidentId]);

  // 如果没有事件数据，显示加载状态
  if (!incident) {
    return (
      <div className="p-10 bg-gray-50 min-h-full">
        <div className="text-center">
          <div className="text-lg text-gray-600">加载中...</div>
        </div>
      </div>
    );
  }

  const mtta = calculateTimeDiffInMinutes(
    incident.created_at,
    incident.confirmed_at || incident.created_at
  );
  const mttr = calculateTimeDiffInMinutes(
    incident.confirmed_at || incident.created_at,
    incident.resolved_at || incident.updated_at
  );

  const updateIncidentStatus = async (
    newStatus: string,
    logMessage: string,
    updateTimes: Record<string, unknown> = {}
  ) => {
    if (!incident) return;
    try {
      const updatedIncident = await IncidentAPI.updateIncident(incident.id, {
        status: newStatus,
        ...updateTimes,
      });
      setIncident(updatedIncident);
      alert(`事件状态已更新为: ${newStatus}`);
      // 在真实应用中，这里会调用API更新后端数据

      // 模拟CMDB CI状态联动
      if (newStatus === "已解决" && updatedIncident.configuration_item_id) {
        // 这里应该调用CMDB API更新配置项状态
        console.log(
          `CMDB CI ${updatedIncident.configuration_item_id} 状态更新为: 运行中`
        );
      }
    } catch (error) {
      console.error("Failed to update incident status:", error);
      alert("更新事件状态失败。");
    }
  };

  const handleResolveIncident = async () => {
    if (incident?.status === "已解决" || incident?.status === "已关闭") {
      alert("事件已解决或已关闭，无需重复操作。");
      return;
    }
    const resolutionNotes = prompt("请输入解决方案描述：");
    if (resolutionNotes) {
      try {
        const updatedIncident = await IncidentAPI.updateIncident(
          incident?.id || 0,
          {
            status: "已解决",
            resolution_notes: resolutionNotes,
            resolved_at: new Date().toISOString(),
          }
        );
        setIncident(updatedIncident);
        alert(`事件已解决。解决方案：${resolutionNotes}`);
      } catch (error) {
        console.error("Failed to resolve incident:", error);
        alert("解决事件失败。");
      }
    }
  };

  const handleCloseIncident = async () => {
    if (incident?.status === "已关闭") {
      alert("事件已关闭，无需重复操作。");
      return;
    }
    if (incident?.status !== "已解决") {
      const confirmClose = confirm("事件尚未解决，确定要直接关闭吗？");
      if (!confirmClose) return;
    }
    try {
      const updatedIncident = await IncidentAPI.updateIncident(
        incident?.id || 0,
        {
          status: "已关闭",
          closed_at: new Date().toISOString(),
        }
      );
      setIncident(updatedIncident);
      alert("事件已关闭。");
    } catch (error) {
      console.error("Failed to close incident:", error);
      alert("关闭事件失败。");
    }
  };

  const handleSuspendIncident = async () => {
    if (incident?.status === "挂起") {
      alert("事件已处于挂起状态。");
      return;
    }
    const suspendReason = prompt("请输入挂起原因：");
    if (suspendReason) {
      try {
        const updatedIncident = await IncidentAPI.updateIncident(
          incident?.id || 0,
          {
            status: "挂起",
            suspend_reason: suspendReason,
          }
        );
        setIncident(updatedIncident);
        alert(`事件已挂起。原因：${suspendReason}`);
      } catch (error) {
        console.error("Failed to suspend incident:", error);
        alert("挂起事件失败。");
      }
    }
  };

  const handleResumeIncident = async () => {
    if (incident?.status !== "挂起") {
      alert("事件未处于挂起状态。");
      return;
    }
    try {
      const updatedIncident = await IncidentAPI.updateIncident(
        incident?.id || 0,
        {
          status: "处理中",
          suspend_reason: undefined, // 恢复处理时清除挂起原因
        }
      );
      setIncident(updatedIncident);
      alert("事件已恢复处理。");
    } catch (error) {
      console.error("Failed to resume incident:", error);
      alert("恢复事件失败。");
    }
  };

  const handleMarkAsMajorIncident = async () => {
    if (incident?.is_major_incident) {
      alert("此事件已是重大事件。");
      return;
    }
    const confirmMajor = confirm(
      "标记为重大事件将触发MIM流程，包括：\n1. 通知高层管理人员\n2. 创建重大事件会议\n3. 启动升级流程\n\n确定要标记为重大事件吗？"
    );
    if (confirmMajor) {
      try {
        const updatedIncident = await IncidentAPI.updateIncident(
          incident?.id || 0,
          {
            is_major_incident: true,
          }
        );
        setIncident(updatedIncident);
        alert("事件已标记为重大事件！重大事件响应团队已被通知。");
        // 实际应用中，这里会触发更复杂的MIM流程，如创建会议、通知高层等
      } catch (error) {
        console.error("Failed to mark as major incident:", error);
        alert("标记为重大事件失败。");
      }
    }
  };

  const handleAddComment = async () => {
    if (newComment.trim()) {
      try {
        const updatedIncident = await IncidentAPI.addComment(
          incident?.id || 0,
          {
            text: newComment.trim(),
          }
        );
        setIncident(updatedIncident);
        setNewComment("");
        alert("评论已添加。");
      } catch (error) {
        console.error("Failed to add comment:", error);
        alert("添加评论失败。");
      }
    }
  };

  const handleAISidebar = async () => {
    setAiLoading(true);
    try {
      const [kb, sum] = await Promise.all([
        aiSearchKB(`${incident.title}\n${incident.description}`),
        aiSummarize(`${incident.title}\n${incident.description}`),
      ]);
      setKbAnswers(kb.answers || []);
      setSummary(sum.summary);
    } finally {
      setAiLoading(false);
    }
  };

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <button
          onClick={() => router.back()}
          className="flex items-center text-blue-600 hover:underline mb-4"
        >
          <ArrowLeft className="w-5 h-5 mr-2" />
          返回事件列表
        </button>
        <div className="flex justify-between items-start">
          <div>
            <h2 className="text-4xl font-bold text-gray-800 flex items-center">
              {incident.title}
              {incident.is_major_incident && (
                <span className="ml-4 px-3 py-1 text-sm font-bold rounded-full bg-red-600 text-white flex items-center animate-pulse">
                  <Zap className="w-4 h-4 mr-2" /> 重大事件
                </span>
              )}
            </h2>
            <p className="text-gray-500 mt-1">事件ID: {incidentId}</p>
          </div>
          <div className="flex space-x-4">
            {!incident.is_major_incident && (
              <button
                onClick={handleMarkAsMajorIncident}
                className="flex items-center bg-red-500 text-white font-semibold py-2 px-4 rounded-lg hover:bg-red-600 transition-colors"
              >
                <Zap className="w-5 h-5 mr-2" />
                标记为重大事件
              </button>
            )}
            <Link
              href={`/problems/new?fromIncidentId=${incidentId}&incidentTitle=${encodeURIComponent(
                incident.title
              )}&incidentDescription=${encodeURIComponent(
                incident.description
              )}`}
              passHref
            >
              <button className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
                <PlusCircle className="w-5 h-5 mr-2" />
                创建问题
              </button>
            </Link>
            {incident.status === "挂起" ? (
              <button
                onClick={handleResumeIncident}
                className="flex items-center bg-purple-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-purple-700 transition-colors"
              >
                <PlayCircle className="w-5 h-5 mr-2" />
                恢复处理
              </button>
            ) : (
              <button
                onClick={handleSuspendIncident}
                className="flex items-center bg-yellow-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-yellow-700 transition-colors"
              >
                <PauseCircle className="w-5 h-5 mr-2" />
                挂起事件
              </button>
            )}
            {incident.status !== "已解决" && incident.status !== "已关闭" && (
              <button
                onClick={handleResolveIncident}
                className="flex items-center bg-green-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-green-700 transition-colors"
              >
                <CheckCircle className="w-5 h-5 mr-2" />
                解决事件
              </button>
            )}
            {incident.status === "已解决" && incident.status !== "已关闭" && (
              <button
                onClick={handleCloseIncident}
                className="flex items-center bg-gray-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-gray-700 transition-colors"
              >
                <XCircle className="w-5 h-5 mr-2" />
                关闭事件
              </button>
            )}
          </div>
        </div>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* 左侧：事件详情和处理日志 */}
        <div className="lg:col-span-2 bg-white p-8 rounded-lg shadow-md">
          <h3 className="text-xl font-semibold text-gray-700 mb-4">事件描述</h3>
          <p className="text-gray-600 mb-8">{incident.description}</p>

          <h3 className="text-xl font-semibold text-gray-700 mb-4">处理日志</h3>
          <div className="space-y-4">
            {incident.logs?.map((log, i) => (
              <p key={i} className="text-sm text-gray-500 font-mono">
                {log}
              </p>
            ))}
          </div>

          {/* 评论/备注区域 */}
          <div className="mt-8 pt-8 border-t border-gray-200">
            <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
              <MessageSquare className="w-5 h-5 mr-2" /> 内部评论/备注
            </h3>
            <div className="space-y-4 mb-4">
              {incident.comments?.length > 0 ? (
                incident.comments.map((comment, i) => (
                  <div key={i} className="bg-gray-50 p-3 rounded-lg">
                    <p className="text-sm font-semibold text-gray-800">
                      {comment.author}{" "}
                      <span className="text-gray-500 font-normal">
                        于 {comment.timestamp}
                      </span>
                    </p>
                    <p className="text-gray-700 mt-1">{comment.text}</p>
                  </div>
                ))
              ) : (
                <p className="text-sm text-gray-500">暂无评论。</p>
              )}
            </div>
            <textarea
              className="w-full p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
              placeholder="添加内部评论或备注..."
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
            ></textarea>
            <button
              onClick={handleAddComment}
              className="mt-2 bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors"
            >
              添加评论
            </button>
          </div>
        </div>

        {/* 右侧：元数据和联动功能 */}
        <div className="space-y-8">
          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-xl font-semibold text-gray-700 mb-4">
              事件信息
            </h3>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between">
                <span>状态:</span>
                <span className="font-semibold">{incident.status}</span>
              </div>
              <div className="flex justify-between">
                <span>优先级:</span>
                <span className="font-semibold text-red-600">
                  {incident.priority}
                </span>
              </div>
              <div className="flex justify-between">
                <span>负责人:</span>
                <span className="font-semibold">
                  {incident.assignee?.name || "未分配"}
                </span>
              </div>
              <div className="flex justify-between">
                <span>来源:</span>
                <span className="font-semibold">{incident.source}</span>
              </div>
              <div className="flex justify-between">
                <span>创建时间:</span>
                <span className="font-semibold">{incident.created_at}</span>
              </div>
              <div className="flex justify-between">
                <span>平均确认时间 (MTTA):</span>
                <span className="font-semibold">{mtta}</span>
              </div>
              <div className="flex justify-between">
                <span>平均恢复时间 (MTTR):</span>
                <span className="font-semibold">{mttr}</span>
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
              <LinkIcon className="w-5 h-5 mr-2" /> 关联的配置项 (CI)
            </h3>
            {incident.configuration_item ? (
              <div className="flex items-center p-3 bg-blue-50 rounded-lg">
                <Cpu className="w-6 h-6 text-blue-600 mr-3" />
                <div>
                  <Link
                    href={`/cmdb/ci/${incident.configuration_item.id}`}
                    className="font-semibold text-blue-700 hover:underline"
                  >
                    {incident.configuration_item.name}
                  </Link>
                  <p className="text-sm text-gray-600">
                    {incident.configuration_item.type}
                  </p>
                  <p className="text-sm text-gray-600">
                    状态: {incident.configuration_item.status}
                  </p>
                  {incident.configuration_item.description && (
                    <p className="text-sm text-gray-600">
                      描述: {incident.configuration_item.description}
                    </p>
                  )}
                </div>
              </div>
            ) : (
              <div className="text-sm text-gray-500">暂无关联的配置项。</div>
            )}
          </div>

          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
              <BookOpen className="w-5 h-5 mr-2" /> 知识库助手
            </h3>
            <div className="flex justify-between mb-3">
              <button
                onClick={handleAISidebar}
                disabled={aiLoading}
                className="text-sm flex items-center px-2 py-1 border rounded hover:bg-gray-50 disabled:opacity-50"
              >
                <Sparkles className="w-4 h-4 mr-1" /> 一键推荐
              </button>
              <div className="text-sm text-gray-500 flex items-center">
                <FileText className="w-4 h-4 mr-1" />
                摘要
              </div>
            </div>
            {summary && (
              <div className="text-sm text-gray-700 p-2 bg-gray-50 rounded mb-3">
                <div className="flex items-center justify-between">
                  <span>{summary}</span>
                  <AIFeedback
                    kind="summarize"
                    query={`${incident.title}\n${incident.description}`}
                    itemType="incident"
                    itemId={incident.id}
                    className="ml-2"
                  />
                </div>
              </div>
            )}
            <div className="space-y-3">
              {kbAnswers.length === 0 ? (
                <div className="text-sm text-gray-400">
                  暂无推荐，点击上方“一键推荐”试试。
                </div>
              ) : (
                kbAnswers.map((a: any, i: number) => (
                  <div
                    key={i}
                    className="p-3 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <p className="font-semibold text-gray-800">
                          {a.title || "相关内容"}
                        </p>
                        <span className="text-sm text-green-600">
                          相似度:{" "}
                          {(a.score ? Math.round(a.score * 100) : 85) + "%"}
                        </span>
                      </div>
                      <div className="flex items-center space-x-2 ml-2">
                        <button className="text-sm text-blue-600 hover:underline">
                          查看
                        </button>
                        <AIFeedback
                          kind="search"
                          query={`${incident.title}\n${incident.description}`}
                          itemType="knowledge"
                          itemId={a.id}
                        />
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
              <Search className="w-5 h-5 mr-2" /> 相似事件
            </h3>
            <div className="space-y-3">
              {/* similar.length === 0 ? (
                <div className="text-sm text-gray-400">
                  暂无相似事件，点击上方“知识库助手”中的一键推荐会同步拉取。
                </div>
              ) : (
                similar.map((s: any, i: number) => (
                  <div key={i} className="p-3 bg-gray-100 rounded">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="font-medium">{s.title}</div>
                        <div className="text-gray-600 text-sm">{s.snippet}</div>
                      </div>
                      <AIFeedback
                        kind="similar-incidents"
                        query={`${incident.title}\n${incident.description}`}
                        itemType="incident"
                        itemId={incident.id}
                        className="ml-2"
                      />
                    </div>
                  </div>
                ))
              )} */}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default IncidentDetailPage;
