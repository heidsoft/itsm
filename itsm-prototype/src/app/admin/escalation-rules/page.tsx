"use client";

import React, { useState } from "react";
import {
  Zap,
  Clock,
  Users,
  AlertTriangle,
  CheckCircle,
  Plus,
  Search,
  Filter,
  Edit,
  Trash2,
  Eye,
  ArrowUp,
  Settings,
  Target,
} from "lucide-react";

// 升级规则的数据类型
interface EscalationRule {
  id: string;
  name: string;
  description: string;
  triggerCondition: string;
  priority: "P1" | "P2" | "P3" | "P4";
  serviceType: string;
  escalationLevels: EscalationLevel[];
  status: "active" | "inactive" | "draft";
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}

interface EscalationLevel {
  level: number;
  timeThreshold: string;
  escalateTo: string;
  notificationMethod: string[];
  action: string;
}

// 模拟升级规则数据
const mockEscalationRules: EscalationRule[] = [
  {
    id: "ESC-001",
    name: "P1事件升级规则",
    description: "针对P1级别紧急事件的自动升级规则，确保关键问题得到及时处理",
    triggerCondition: '工单状态为"进行中"且优先级为P1',
    priority: "P1",
    serviceType: "关键业务系统",
    escalationLevels: [
      {
        level: 1,
        timeThreshold: "15分钟",
        escalateTo: "高级工程师",
        notificationMethod: ["邮件", "短信", "企业微信"],
        action: "自动分配给高级工程师组",
      },
      {
        level: 2,
        timeThreshold: "1小时",
        escalateTo: "技术经理",
        notificationMethod: ["邮件", "短信", "电话"],
        action: "通知技术经理并创建紧急会议",
      },
      {
        level: 3,
        timeThreshold: "4小时",
        escalateTo: "CTO",
        notificationMethod: ["邮件", "电话"],
        action: "升级至CTO并启动应急响应流程",
      },
    ],
    status: "active",
    createdAt: "2024-01-15",
    updatedAt: "2024-06-01",
    createdBy: "张三",
  },
  {
    id: "ESC-002",
    name: "P2事件升级规则",
    description: "针对P2级别高优先级事件的升级规则",
    triggerCondition: '工单状态为"进行中"且优先级为P2',
    priority: "P2",
    serviceType: "一般业务系统",
    escalationLevels: [
      {
        level: 1,
        timeThreshold: "30分钟",
        escalateTo: "高级工程师",
        notificationMethod: ["邮件", "企业微信"],
        action: "自动分配给高级工程师组",
      },
      {
        level: 2,
        timeThreshold: "2小时",
        escalateTo: "技术经理",
        notificationMethod: ["邮件", "短信"],
        action: "通知技术经理",
      },
    ],
    status: "active",
    createdAt: "2024-01-20",
    updatedAt: "2024-05-15",
    createdBy: "李四",
  },
  {
    id: "ESC-003",
    name: "服务请求超时升级",
    description: "服务请求长时间未处理的升级规则",
    triggerCondition: '服务请求状态为"待处理"超过预定时间',
    priority: "P3",
    serviceType: "服务请求",
    escalationLevels: [
      {
        level: 1,
        timeThreshold: "4小时",
        escalateTo: "服务台主管",
        notificationMethod: ["邮件"],
        action: "提醒服务台主管关注",
      },
      {
        level: 2,
        timeThreshold: "1工作日",
        escalateTo: "服务经理",
        notificationMethod: ["邮件", "企业微信"],
        action: "升级至服务经理处理",
      },
    ],
    status: "active",
    createdAt: "2024-02-01",
    updatedAt: "2024-06-10",
    createdBy: "王五",
  },
  {
    id: "ESC-004",
    name: "SLA违约升级规则",
    description: "当服务级别协议即将违约时的升级规则",
    triggerCondition: "SLA剩余时间少于20%",
    priority: "P2",
    serviceType: "所有服务",
    escalationLevels: [
      {
        level: 1,
        timeThreshold: "即时",
        escalateTo: "责任工程师",
        notificationMethod: ["邮件", "短信"],
        action: "立即通知责任工程师",
      },
      {
        level: 2,
        timeThreshold: "10%剩余时间",
        escalateTo: "服务经理",
        notificationMethod: ["邮件", "短信", "电话"],
        action: "紧急升级至服务经理",
      },
    ],
    status: "draft",
    createdAt: "2024-06-01",
    updatedAt: "2024-06-15",
    createdBy: "赵六",
  },
];

// 优先级颜色映射
const priorityColors = {
  P1: "bg-red-100 text-red-800",
  P2: "bg-orange-100 text-orange-800",
  P3: "bg-yellow-100 text-yellow-800",
  P4: "bg-green-100 text-green-800",
};

// 状态颜色映射
const statusColors = {
  active: "bg-green-100 text-green-800",
  inactive: "bg-gray-100 text-gray-800",
  draft: "bg-blue-100 text-blue-800",
};

// 状态标签映射
const statusLabels = {
  active: "启用",
  inactive: "停用",
  draft: "草稿",
};

const EscalationRulesPage = () => {
  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState("all");
  const [filterPriority, setFilterPriority] = useState("all");
  const [selectedRule, setSelectedRule] = useState<EscalationRule | null>(null);
  const [showDetails, setShowDetails] = useState(false);

  // 过滤升级规则
  const filteredRules = mockEscalationRules.filter((rule) => {
    const matchesSearch =
      rule.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      rule.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      rule.serviceType.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus =
      filterStatus === "all" || rule.status === filterStatus;
    const matchesPriority =
      filterPriority === "all" || rule.priority === filterPriority;

    return matchesSearch && matchesStatus && matchesPriority;
  });

  // 统计数据
  const stats = {
    total: mockEscalationRules.length,
    active: mockEscalationRules.filter((rule) => rule.status === "active")
      .length,
    draft: mockEscalationRules.filter((rule) => rule.status === "draft").length,
    inactive: mockEscalationRules.filter((rule) => rule.status === "inactive")
      .length,
  };

  const handleViewDetails = (rule: EscalationRule) => {
    setSelectedRule(rule);
    setShowDetails(true);
  };

  return (
    <div className="p-8 bg-gray-50 min-h-full">
      {/* 页面头部 */}
      <div className="mb-8">
        <div className="flex justify-between items-start mb-6">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">
              升级规则管理
            </h1>
            <p className="text-gray-600">
              配置和管理工单自动升级规则，确保问题得到及时处理
            </p>
          </div>
          <button className="flex items-center bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors">
            <Plus className="w-5 h-5 mr-2" />
            新建升级规则
          </button>
        </div>

        {/* 统计卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">总计</p>
                <p className="text-2xl font-bold text-gray-900">
                  {stats.total}
                </p>
              </div>
              <div className="p-3 bg-blue-100 rounded-full">
                <Zap className="w-6 h-6 text-blue-600" />
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">启用中</p>
                <p className="text-2xl font-bold text-green-600">
                  {stats.active}
                </p>
              </div>
              <div className="p-3 bg-green-100 rounded-full">
                <CheckCircle className="w-6 h-6 text-green-600" />
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">草稿</p>
                <p className="text-2xl font-bold text-blue-600">
                  {stats.draft}
                </p>
              </div>
              <div className="p-3 bg-blue-100 rounded-full">
                <Settings className="w-6 h-6 text-blue-600" />
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">停用</p>
                <p className="text-2xl font-bold text-gray-600">
                  {stats.inactive}
                </p>
              </div>
              <div className="p-3 bg-gray-100 rounded-full">
                <AlertTriangle className="w-6 h-6 text-gray-600" />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200 mb-6">
        <div className="flex flex-col md:flex-row gap-4">
          {/* 搜索框 */}
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
            <input
              type="text"
              placeholder="搜索升级规则名称、描述或服务类型..."
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>

          {/* 状态过滤 */}
          <div className="flex items-center gap-2">
            <Filter className="w-5 h-5 text-gray-400" />
            <select
              className="border border-gray-300 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
            >
              <option value="all">所有状态</option>
              <option value="active">启用</option>
              <option value="draft">草稿</option>
              <option value="inactive">停用</option>
            </select>
          </div>

          {/* 优先级过滤 */}
          <div>
            <select
              className="border border-gray-300 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={filterPriority}
              onChange={(e) => setFilterPriority(e.target.value)}
            >
              <option value="all">所有优先级</option>
              <option value="P1">P1 - 紧急</option>
              <option value="P2">P2 - 高</option>
              <option value="P3">P3 - 中</option>
              <option value="P4">P4 - 低</option>
            </select>
          </div>
        </div>
      </div>

      {/* 升级规则列表 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  升级规则
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  服务类型
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  优先级
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  触发条件
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  升级层级
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {filteredRules.map((rule) => (
                <tr key={rule.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4">
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        {rule.name}
                      </div>
                      <div className="text-sm text-gray-500 max-w-xs truncate">
                        {rule.description}
                      </div>
                      <div className="text-xs text-gray-400 mt-1">
                        ID: {rule.id}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span className="text-sm text-gray-900">
                      {rule.serviceType}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                        priorityColors[rule.priority]
                      }`}
                    >
                      {rule.priority}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className="text-sm text-gray-900 max-w-xs truncate block">
                      {rule.triggerCondition}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center text-sm text-gray-900">
                      <ArrowUp className="w-4 h-4 mr-1 text-gray-400" />
                      {rule.escalationLevels.length} 级
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                        statusColors[rule.status]
                      }`}
                    >
                      {statusLabels[rule.status]}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center space-x-2">
                      <button
                        onClick={() => handleViewDetails(rule)}
                        className="text-blue-600 hover:text-blue-800 p-1"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                      <button className="text-green-600 hover:text-green-800 p-1">
                        <Edit className="w-4 h-4" />
                      </button>
                      <button className="text-red-600 hover:text-red-800 p-1">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {filteredRules.length === 0 && (
          <div className="text-center py-12">
            <Zap className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              没有找到升级规则
            </h3>
            <p className="text-gray-500">尝试调整搜索条件或创建新的升级规则</p>
          </div>
        )}
      </div>

      {/* 详情弹窗 */}
      {showDetails && selectedRule && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b border-gray-200">
              <div className="flex justify-between items-start">
                <div>
                  <h2 className="text-2xl font-bold text-gray-900">
                    {selectedRule.name}
                  </h2>
                  <p className="text-gray-600 mt-1">
                    {selectedRule.description}
                  </p>
                </div>
                <button
                  onClick={() => setShowDetails(false)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  ×
                </button>
              </div>
            </div>

            <div className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                <div>
                  <h3 className="text-lg font-semibold text-gray-900 mb-3">
                    基本信息
                  </h3>
                  <div className="space-y-3">
                    <div>
                      <span className="text-sm font-medium text-gray-500">
                        规则ID:
                      </span>
                      <span className="ml-2 text-sm text-gray-900">
                        {selectedRule.id}
                      </span>
                    </div>
                    <div>
                      <span className="text-sm font-medium text-gray-500">
                        服务类型:
                      </span>
                      <span className="ml-2 text-sm text-gray-900">
                        {selectedRule.serviceType}
                      </span>
                    </div>
                    <div>
                      <span className="text-sm font-medium text-gray-500">
                        优先级:
                      </span>
                      <span
                        className={`ml-2 inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                          priorityColors[selectedRule.priority]
                        }`}
                      >
                        {selectedRule.priority}
                      </span>
                    </div>
                    <div>
                      <span className="text-sm font-medium text-gray-500">
                        状态:
                      </span>
                      <span
                        className={`ml-2 inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                          statusColors[selectedRule.status]
                        }`}
                      >
                        {statusLabels[selectedRule.status]}
                      </span>
                    </div>
                  </div>
                </div>

                <div>
                  <h3 className="text-lg font-semibold text-gray-900 mb-3">
                    触发条件
                  </h3>
                  <p className="text-sm text-gray-700 bg-gray-50 p-3 rounded-lg">
                    {selectedRule.triggerCondition}
                  </p>
                </div>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4">
                  升级层级配置
                </h3>
                <div className="space-y-4">
                  {selectedRule.escalationLevels.map((level, index) => (
                    <div
                      key={index}
                      className="border border-gray-200 rounded-lg p-4"
                    >
                      <div className="flex items-center mb-3">
                        <div className="flex items-center justify-center w-8 h-8 bg-blue-100 text-blue-600 rounded-full text-sm font-semibold mr-3">
                          {level.level}
                        </div>
                        <h4 className="text-md font-medium text-gray-900">
                          第 {level.level} 级升级
                        </h4>
                      </div>

                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 ml-11">
                        <div>
                          <span className="text-sm font-medium text-gray-500">
                            时间阈值:
                          </span>
                          <div className="flex items-center mt-1">
                            <Clock className="w-4 h-4 text-gray-400 mr-1" />
                            <span className="text-sm text-gray-900">
                              {level.timeThreshold}
                            </span>
                          </div>
                        </div>

                        <div>
                          <span className="text-sm font-medium text-gray-500">
                            升级对象:
                          </span>
                          <div className="flex items-center mt-1">
                            <Users className="w-4 h-4 text-gray-400 mr-1" />
                            <span className="text-sm text-gray-900">
                              {level.escalateTo}
                            </span>
                          </div>
                        </div>

                        <div>
                          <span className="text-sm font-medium text-gray-500">
                            通知方式:
                          </span>
                          <div className="flex flex-wrap gap-1 mt-1">
                            {level.notificationMethod.map((method, idx) => (
                              <span
                                key={idx}
                                className="inline-flex px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded"
                              >
                                {method}
                              </span>
                            ))}
                          </div>
                        </div>

                        <div>
                          <span className="text-sm font-medium text-gray-500">
                            执行动作:
                          </span>
                          <p className="text-sm text-gray-900 mt-1">
                            {level.action}
                          </p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default EscalationRulesPage;
