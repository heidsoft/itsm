"use client";

import React, { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { TicketApi } from "../../lib/ticket-api";
import { CreateTicketRequest } from "../../lib/api-config";
import {
  ArrowLeft,
  Save,
  Brain,
  Sparkles,
  UserCircle,
  BookOpen,
  FileText,
  Eye,
  CheckCircle,
} from "lucide-react";
import Link from "next/link";
import { aiSearchKB, aiSummarize, aiTriage, RagAnswer } from "../../lib/ai-api";
import { AIFeedback } from "../../components/AIFeedback";
import {
  ticketTemplateService,
  type TicketTemplate,
} from "../../lib/services/ticket-template-service";

const CreateTicketPage: React.FC = () => {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState<CreateTicketRequest>({
    title: "",
    description: "",
    priority: "medium",
    form_fields: {},
  });

  // 模板相关状态
  const [templates, setTemplates] = useState<TicketTemplate[]>([]);
  const [selectedTemplate, setSelectedTemplate] =
    useState<TicketTemplate | null>(null);
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [templateLoading, setTemplateLoading] = useState(false);

  const [aiLoading, setAiLoading] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);
  const [kbAnswers, setKbAnswers] = useState<RagAnswer[]>([]);
  const [triage, setTriage] = useState<{
    category: string;
    priority: string;
    assignee_id: number;
    confidence: number;
  } | null>(null);
  const [summary, setSummary] = useState<string>("");

  // 添加优先级映射函数
  const priorityMap: { [key: string]: string } = {
    低: "low",
    中: "medium",
    高: "high",
    紧急: "critical",
  };

  // 加载模板数据
  useEffect(() => {
    loadTemplates();
  }, []);

  const loadTemplates = async () => {
    try {
      setTemplateLoading(true);
      const response = await ticketTemplateService.getTemplates({
        page: 1,
        size: 100,
        is_active: true,
        sort_by: "name",
        sort_order: "asc",
      });
      setTemplates(response.data || []);
    } catch (error) {
      console.error("加载模板失败:", error);
    } finally {
      setTemplateLoading(false);
    }
  };

  // 应用模板
  const applyTemplate = (template: TicketTemplate) => {
    setFormData({
      title: "",
      description: "",
      priority: template.priority || "medium",
      form_fields: {
        ...template.form_fields,
        category: template.category,
        template_id: template.id,
      },
    });
    setSelectedTemplate(template);
    setShowTemplateModal(false);
  };

  // 清除模板
  const clearTemplate = () => {
    setSelectedTemplate(null);
    setFormData({
      title: "",
      description: "",
      priority: "medium",
      form_fields: {},
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.title.trim() || !formData.description.trim()) {
      setError("请填写标题和描述");
      return;
    }

    try {
      setLoading(true);
      setError(null);

      // 创建提交数据，确保priority是英文值
      const submitData = {
        ...formData,
        priority: priorityMap[formData.priority] || formData.priority,
      };

      const response = await TicketApi.createTicket(submitData);

      if (response && response.id) {
        router.push(`/tickets/${response.id}`);
      } else {
        setError("创建工单失败");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "网络错误");
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (
    field: keyof CreateTicketRequest,
    value: unknown
  ) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleAISuggest = async () => {
    if (!formData.title.trim() && !formData.description.trim()) return;
    setAiLoading(true);
    setAiError(null);
    try {
      const [t, kb, sum] = await Promise.all([
        aiTriage(formData.title, formData.description),
        aiSearchKB(`${formData.title}\n${formData.description}`, 5),
        aiSummarize(`${formData.title}\n${formData.description}`, 160),
      ]);
      setTriage(t);
      setKbAnswers(kb.answers || []);
      setSummary(sum.summary);
      // 回填优先级
      if (t?.priority) {
        // triage 返回的可能是 low|medium|high|urgent，简单映射
        const p = t.priority.toLowerCase();
        const normalized =
          p === "urgent"
            ? "critical"
            : ["low", "medium", "high", "critical"].includes(p)
            ? p
            : "medium";
        setFormData((prev) => ({
          ...prev,
          priority: normalized as CreateTicketRequest["priority"],
          assignee_id: t.assignee_id || prev.assignee_id,
          form_fields: {
            ...(prev.form_fields || {}),
            category: t.category || (prev.form_fields as any)?.category || "",
          },
        }));
      }
      // TODO: 回填受理人（assignee_id）与分类（需要前端有对应字段/下拉）
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "智能建议失败";
      setAiError(msg);
    } finally {
      setAiLoading(false);
    }
  };

  return (
    <div className="p-6">
      {/* 头部 */}
      <div className="flex items-center mb-6">
        <Link
          href="/tickets"
          className="flex items-center text-gray-600 hover:text-gray-900 mr-4"
        >
          <ArrowLeft className="w-4 h-4 mr-1" />
          返回
        </Link>
        <h1 className="text-2xl font-bold text-gray-900">创建工单</h1>
      </div>

      {/* 模板选择器 */}
      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center">
            <FileText className="w-5 h-5 mr-2 text-blue-600" />
            <h2 className="text-lg font-semibold text-gray-900">选择模板</h2>
          </div>
          {selectedTemplate && (
            <button
              onClick={clearTemplate}
              className="text-sm text-gray-500 hover:text-gray-700"
            >
              清除模板
            </button>
          )}
        </div>

        {selectedTemplate ? (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <CheckCircle className="w-5 h-5 text-blue-600 mr-2" />
                <div>
                  <div className="font-medium text-blue-900">
                    已选择模板：{selectedTemplate.name}
                  </div>
                  <div className="text-sm text-blue-700">
                    {selectedTemplate.description}
                  </div>
                  <div className="text-xs text-blue-600 mt-1">
                    分类：{selectedTemplate.category} | 默认优先级：
                    {selectedTemplate.priority}
                  </div>
                </div>
              </div>
              <button
                onClick={() => setShowTemplateModal(true)}
                className="text-sm text-blue-600 hover:text-blue-800"
              >
                更换模板
              </button>
            </div>
          </div>
        ) : (
          <div className="text-center py-8">
            <FileText className="w-12 h-12 text-gray-400 mx-auto mb-3" />
            <p className="text-gray-500 mb-4">选择一个模板来快速创建工单</p>
            <button
              onClick={() => setShowTemplateModal(true)}
              className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              <FileText className="w-4 h-4 mr-2" />
              选择模板
            </button>
          </div>
        )}
      </div>

      {/* 错误提示 */}
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {/* 表单 */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* 标题 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              工单标题 *
            </label>
            <input
              type="text"
              value={formData.title}
              onChange={(e) => handleInputChange("title", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="请输入工单标题"
              required
            />
          </div>

          {/* 优先级 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              优先级
            </label>
            <select
              value={formData.priority}
              onChange={(e) => handleInputChange("priority", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="low">低</option>
              <option value="medium">中</option>
              <option value="high">高</option>
              <option value="critical">紧急</option>
            </select>
          </div>

          {/* 受理人ID（可选） */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              受理人ID（可选）
            </label>
            <input
              type="number"
              value={formData.assignee_id ?? ""}
              onChange={(e) =>
                handleInputChange(
                  "assignee_id",
                  e.target.value ? Number(e.target.value) : undefined
                )
              }
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="填写受理人用户ID"
            />
          </div>

          {/* 分类（可选，写入表单字段） */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              分类（可选）
            </label>
            <input
              type="text"
              value={(formData.form_fields as any)?.category || ""}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  form_fields: {
                    ...(prev.form_fields || {}),
                    category: e.target.value,
                  },
                }))
              }
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="如 database / network / general"
            />
          </div>

          {/* 描述 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              问题描述 *
            </label>
            <textarea
              value={formData.description}
              onChange={(e) => handleInputChange("description", e.target.value)}
              rows={6}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="请详细描述遇到的问题..."
              required
            />
          </div>

          {/* 智能建议 */}
          <div className="p-4 border rounded-md bg-gray-50">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center">
                <Brain className="w-4 h-4 mr-2" />
                <span className="font-semibold">智能建议</span>
              </div>
              <button
                type="button"
                onClick={handleAISuggest}
                disabled={aiLoading}
                className="text-sm flex items-center px-2 py-1 border rounded hover:bg-gray-100 disabled:opacity-50"
              >
                <Sparkles className="w-4 h-4 mr-1" /> 一键生成
              </button>
            </div>
            {aiError && (
              <div className="text-sm text-red-600 mb-2">{aiError}</div>
            )}
            {triage && (
              <div className="text-sm text-gray-700 mb-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <UserCircle className="w-4 h-4 mr-1" />
                    建议分类：{triage.category}，优先级：{triage.priority}
                    ，建议受理人ID：{triage.assignee_id}，置信度：
                    {Math.round((triage.confidence || 0) * 100)}%
                  </div>
                  <AIFeedback
                    kind="triage"
                    query={`${formData.title}\n${formData.description}`}
                    className="ml-2"
                  />
                </div>
              </div>
            )}
            {summary && (
              <div className="text-sm text-gray-600 mb-2">
                <div className="flex items-center justify-between">
                  <span>摘要：{summary}</span>
                  <AIFeedback
                    kind="summarize"
                    query={`${formData.title}\n${formData.description}`}
                    className="ml-2"
                  />
                </div>
              </div>
            )}
            {kbAnswers.length > 0 && (
              <div>
                <div className="text-sm font-semibold mb-2 flex items-center">
                  <BookOpen className="w-4 h-4 mr-1" /> 知识推荐
                </div>
                <ul className="space-y-2">
                  {kbAnswers.map((a, idx) => (
                    <li
                      key={idx}
                      className="text-sm p-2 bg-white border rounded"
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="font-medium">
                            {a.title || "相关内容"}
                          </div>
                          <div className="text-gray-600">{a.snippet}</div>
                        </div>
                        <AIFeedback
                          kind="search"
                          query={`${formData.title}\n${formData.description}`}
                          itemType="knowledge"
                          itemId={a.id}
                          className="ml-2 flex-shrink-0"
                        />
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>

          {/* 提交按钮 */}
          <div className="flex justify-end space-x-3">
            <Link
              href="/tickets"
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 transition-colors"
            >
              取消
            </Link>
            <button
              type="submit"
              disabled={loading}
              className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {loading ? (
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
              ) : (
                <Save className="w-4 h-4 mr-2" />
              )}
              {loading ? "创建中..." : "创建工单"}
            </button>
          </div>
        </form>
      </div>

      {/* 模板选择模态框 */}
      {showTemplateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[80vh] overflow-hidden">
            <div className="flex items-center justify-between p-6 border-b">
              <h2 className="text-xl font-semibold text-gray-900">
                选择工单模板
              </h2>
              <button
                onClick={() => setShowTemplateModal(false)}
                className="text-gray-400 hover:text-gray-600"
              >
                <svg
                  className="w-6 h-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>

            <div className="p-6 overflow-y-auto max-h-[60vh]">
              {templateLoading ? (
                <div className="text-center py-8">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                  <p className="mt-2 text-gray-500">加载模板中...</p>
                </div>
              ) : templates.length === 0 ? (
                <div className="text-center py-8">
                  <FileText className="w-12 h-12 text-gray-400 mx-auto mb-3" />
                  <p className="text-gray-500">暂无可用模板</p>
                </div>
              ) : (
                <div className="grid gap-4">
                  {templates.map((template) => (
                    <div
                      key={template.id}
                      className="border rounded-lg p-4 hover:border-blue-300 hover:shadow-md transition-all cursor-pointer"
                      onClick={() => applyTemplate(template)}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center mb-2">
                            <h3 className="text-lg font-medium text-gray-900 mr-3">
                              {template.name}
                            </h3>
                            <span
                              className={`px-2 py-1 text-xs rounded-full ${
                                template.is_active
                                  ? "bg-green-100 text-green-800"
                                  : "bg-gray-100 text-gray-800"
                              }`}
                            >
                              {template.is_active ? "启用" : "禁用"}
                            </span>
                          </div>
                          <p className="text-gray-600 mb-3">
                            {template.description}
                          </p>
                          <div className="flex items-center space-x-4 text-sm text-gray-500">
                            <span>分类：{template.category}</span>
                            <span>优先级：{template.priority}</span>
                            <span>
                              更新时间：
                              {new Date(template.updated_at).toLocaleDateString(
                                "zh-CN"
                              )}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              // 这里可以添加模板预览功能
                            }}
                            className="p-2 text-gray-400 hover:text-gray-600"
                            title="预览模板"
                          >
                            <Eye className="w-4 h-4" />
                          </button>
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              applyTemplate(template);
                            }}
                            className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 transition-colors"
                          >
                            应用模板
                          </button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CreateTicketPage;
