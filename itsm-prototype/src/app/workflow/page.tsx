"use client";

import React, { useState, useEffect, useCallback, useMemo } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Modal,
  Form,
  message,
  Tag,
  Space,
  Dropdown,
  Tooltip,
  Row,
  Col,
  Statistic,
  Alert,
  App,
} from "antd";
import {
  Plus,
  Search,
  MoreHorizontal,
  Edit,
  Eye,
  Trash2,
  PlayCircle,
  PauseCircle,
  Copy,
  Download,
  Upload,
  BarChart3,
  RefreshCw,
  FileText,
  GitBranch,
  Clock,
  CheckCircle,
  Code,
} from "lucide-react";

import { WorkflowDesigner } from "../components/WorkflowDesigner";
import { WorkflowAPI } from "../lib/workflow-api";

import { useRouter } from "next/navigation";

const { Option } = Select;

interface Workflow {
  id: number;
  name: string;
  description: string;
  category: string;
  version: string;
  status: "draft" | "active" | "inactive" | "archived";
  bpmn_xml?: string;
  created_at: string;
  updated_at: string;
  instances_count: number;
  running_instances: number;
  created_by: string;
}

const WorkflowManagementPage = () => {
  const router = useRouter();
  const { modal } = App.useApp();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [designerVisible, setDesignerVisible] = useState(false);

  const [editingWorkflow, setEditingWorkflow] = useState<Workflow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [filters, setFilters] = useState({
    status: "",
    category: "",
    keyword: "",
  });
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    running: 0,
    completed: 0,
    draft: 0,
    inactive: 0,
    todayInstances: 0,
    avgExecutionTime: 0,
  });

  // 模拟数据 - 使用useMemo避免每次渲染时重新创建
  const mockWorkflows = useMemo<Workflow[]>(
    () => [
      {
        id: 1,
        name: "工单审批流程",
        description: "标准工单审批流程，包含提交、审核、处理、验收等环节",
        category: "ticket_approval",
        version: "1.0.0",
        status: "active",
        bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_1" name="提交工单">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_2" name="审核工单">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_3" name="处理工单">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_1" name="结束">
      <bpmn:incoming>Flow_4</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="UserTask_2" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="UserTask_2" targetRef="UserTask_3" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="UserTask_3" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`,
        created_at: "2024-01-15T10:30:00Z",
        updated_at: "2024-01-15T14:20:00Z",
        instances_count: 156,
        running_instances: 23,
        created_by: "张三",
      },
      {
        id: 2,
        name: "事件处理流程",
        description: "IT事件处理标准流程，支持自动分配和升级",
        category: "事件处理",
        version: "2.1.0",
        status: "active",
        bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_3" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_3" isExecutable="true">
    <bpmn:startEvent id="StartEvent_3" name="事件报告">
      <bpmn:outgoing>Flow_11</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_7" name="事件分类">
      <bpmn:incoming>Flow_11</bpmn:incoming>
      <bpmn:outgoing>Flow_12</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:exclusiveGateway id="Gateway_2" name="优先级判断">
      <bpmn:incoming>Flow_12</bpmn:incoming>
      <bpmn:outgoing>Flow_13</bpmn:outgoing>
      <bpmn:outgoing>Flow_14</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:userTask id="UserTask_8" name="紧急处理">
      <bpmn:incoming>Flow_13</bpmn:incoming>
      <bpmn:outgoing>Flow_15</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_9" name="常规处理">
      <bpmn:incoming>Flow_14</bpmn:incoming>
      <bpmn:outgoing>Flow_16</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_3" name="事件解决">
      <bpmn:incoming>Flow_15</bpmn:incoming>
      <bpmn:incoming>Flow_16</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_11" sourceRef="StartEvent_3" targetRef="UserTask_7" />
    <bpmn:sequenceFlow id="Flow_12" sourceRef="UserTask_7" targetRef="Gateway_2" />
    <bpmn:sequenceFlow id="Flow_13" sourceRef="Gateway_2" targetRef="UserTask_8" />
    <bpmn:sequenceFlow id="Flow_14" sourceRef="Gateway_2" targetRef="UserTask_9" />
    <bpmn:sequenceFlow id="Flow_15" sourceRef="UserTask_8" targetRef="EndEvent_3" />
    <bpmn:sequenceFlow id="Flow_16" sourceRef="UserTask_9" targetRef="EndEvent_3" />
  </bpmn:process>
</bpmn:definitions>`,
        created_at: "2024-01-14T09:15:00Z",
        updated_at: "2024-01-15T11:45:00Z",
        instances_count: 89,
        running_instances: 12,
        created_by: "李四",
      },
      {
        id: 3,
        name: "变更管理流程",
        description: "IT变更管理流程，包含变更申请、审批、实施、验证",
        category: "变更管理",
        version: "1.5.0",
        status: "draft",
        bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_4" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_4" isExecutable="true">
    <bpmn:startEvent id="StartEvent_4" name="变更申请">
      <bpmn:outgoing>Flow_17</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_10" name="变更评估">
      <bpmn:incoming>Flow_17</bpmn:incoming>
      <bpmn:outgoing>Flow_18</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_11" name="变更审批">
      <bpmn:incoming>Flow_18</bpmn:incoming>
      <bpmn:outgoing>Flow_19</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_12" name="变更实施">
      <bpmn:incoming>Flow_19</bpmn:incoming>
      <bpmn:outgoing>Flow_20</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_13" name="变更验证">
      <bpmn:incoming>Flow_20</bpmn:incoming>
      <bpmn:outgoing>Flow_21</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_4" name="变更完成">
      <bpmn:incoming>Flow_21</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_17" sourceRef="StartEvent_4" targetRef="UserTask_10" />
    <bpmn:sequenceFlow id="Flow_18" sourceRef="UserTask_10" targetRef="UserTask_11" />
    <bpmn:sequenceFlow id="Flow_19" sourceRef="UserTask_11" targetRef="UserTask_12" />
    <bpmn:sequenceFlow id="Flow_20" sourceRef="UserTask_12" targetRef="UserTask_13" />
    <bpmn:sequenceFlow id="Flow_21" sourceRef="UserTask_13" targetRef="EndEvent_4" />
  </bpmn:process>
</bpmn:definitions>`,
        created_at: "2024-01-13T16:20:00Z",
        updated_at: "2024-01-14T10:30:00Z",
        instances_count: 45,
        running_instances: 8,
        created_by: "王五",
      },
    ],
    []
  );

  const loadWorkflows = useCallback(async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise((resolve) => setTimeout(resolve, 500));
      setWorkflows(mockWorkflows);
    } catch {
      message.error("加载工作流失败");
    } finally {
      setLoading(false);
    }
  }, [mockWorkflows]);

  const loadStats = useCallback(async () => {
    try {
      const currentWorkflows = workflows.length > 0 ? workflows : mockWorkflows;
      setStats({
        total: currentWorkflows.length,
        active: currentWorkflows.filter((w) => w.status === "active").length,
        draft: currentWorkflows.filter((w) => w.status === "draft").length,
        inactive: currentWorkflows.filter((w) => w.status === "inactive")
          .length,
        running: currentWorkflows.reduce(
          (sum, w) => sum + w.running_instances,
          0
        ),
        completed: currentWorkflows.reduce(
          (sum, w) => sum + (w.instances_count - w.running_instances),
          0
        ),
        todayInstances: Math.floor(Math.random() * 50) + 10, // 模拟今日实例
        avgExecutionTime: Math.floor(Math.random() * 120) + 30, // 模拟平均执行时间(分钟)
      });
    } catch {
      console.error("加载统计数据失败");
    }
  }, [workflows, mockWorkflows]);

  useEffect(() => {
    loadWorkflows();
    loadStats();
  }, [loadWorkflows, loadStats]);

  // 当filters变化时重新加载数据
  useEffect(() => {
    loadWorkflows();
  }, [filters, loadWorkflows]);

  const handleCreateWorkflow = () => {
    setEditingWorkflow(null);
    setModalVisible(true);
  };

  const handleEditWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);
    setModalVisible(true);
  };

  const handleDesignWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);

    // 根据工作流类型选择对应的设计器
    if (
      workflow.category === "ticket_approval" ||
      workflow.name.includes("审批")
    ) {
      // 工单审批流程使用专门的设计器
      router.push(`/workflow/ticket-approval?id=${workflow.id}`);
    } else {
      // 通用工作流使用BPMN设计器
      setDesignerVisible(true);
    }
  };

  const handleViewBPMN = (workflow: Workflow) => {
    if (!workflow.bpmn_xml) {
      message.warning("该工作流没有BPMN定义");
      return;
    }

    // 显示BPMN XML内容
    Modal.info({
      title: `${workflow.name} - BPMN XML`,
      width: 800,
      content: (
        <div style={{ maxHeight: "500px", overflow: "auto" }}>
          <pre
            style={{
              background: "#f5f5f5",
              padding: "16px",
              borderRadius: "4px",
              fontSize: "12px",
              lineHeight: "1.4",
            }}
          >
            {workflow.bpmn_xml}
          </pre>
        </div>
      ),
      okText: "关闭",
    });
  };

  const handleDeleteWorkflow = async (id: number) => {
    modal.confirm({
      title: "确认删除",
      content:
        "删除工作流将同时删除所有相关实例，此操作不可恢复。确定要删除吗？",
      okText: "确定删除",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          // 模拟删除API调用
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) => prev.filter((w) => w.id !== id));
          message.success("工作流删除成功");
          loadStats();
        } catch {
          message.error("删除失败");
        }
      },
    });
  };

  const handleDeployWorkflow = async (id: number) => {
    modal.confirm({
      title: "确认部署",
      content:
        "部署后工作流将变为激活状态，可以创建新的流程实例。确定要部署吗？",
      okText: "确定部署",
      cancelText: "取消",
      onOk: async () => {
        try {
          // 模拟部署API调用
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.map((w) =>
              w.id === id ? { ...w, status: "active" as const } : w
            )
          );
          message.success("工作流部署成功");
          loadStats();
        } catch {
          message.error("部署失败");
        }
      },
    });
  };

  const handleCopyWorkflow = async (workflow: Workflow) => {
    try {
      // 模拟复制API调用
      await new Promise((resolve) => setTimeout(resolve, 500));
      const newWorkflow: Workflow = {
        ...workflow,
        id: Date.now(), // 临时ID
        name: `${workflow.name} - 副本`,
        status: "draft",
        version: "1.0.0",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        instances_count: 0,
        running_instances: 0,
        created_by: "当前用户",
      };
      setWorkflows((prev) => [newWorkflow, ...prev]);
      message.success(`工作流 "${workflow.name}" 复制成功`);
      loadStats();
    } catch {
      message.error("复制失败");
    }
  };

  const handleExportWorkflow = async (workflow: Workflow) => {
    try {
      // 模拟导出API调用
      await new Promise((resolve) => setTimeout(resolve, 500));

      const exportData = {
        workflow: workflow,
        exportTime: new Date().toISOString(),
        version: "1.0",
      };

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `workflow_${workflow.name}_${workflow.version}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      message.success(`工作流 "${workflow.name}" 导出成功`);
    } catch {
      message.error("导出失败");
    }
  };

  const handleImportWorkflow = () => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".json";
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;

      try {
        const text = await file.text();
        const importData = JSON.parse(text);

        if (!importData.workflow) {
          throw new Error("无效的工作流文件");
        }

        const newWorkflow: Workflow = {
          ...importData.workflow,
          id: Date.now(),
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          status: "draft",
          instances_count: 0,
          running_instances: 0,
          created_by: "当前用户",
        };

        setWorkflows((prev) => [newWorkflow, ...prev]);
        message.success(`工作流 "${newWorkflow.name}" 导入成功`);
        loadStats();
      } catch (error) {
        message.error("导入失败：" + (error as Error).message);
      }
    };
    input.click();
  };

  const handleStopWorkflow = async (id: number) => {
    modal.confirm({
      title: "确认停用",
      content:
        "停用工作流后将无法创建新的流程实例，但已有实例会继续运行。确定要停用吗？",
      okText: "确定停用",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.map((w) =>
              w.id === id ? { ...w, status: "inactive" as const } : w
            )
          );
          message.success("工作流已停用");
          loadStats();
        } catch {
          message.error("停用失败");
        }
      },
    });
  };

  const handleActivateWorkflow = async (id: number) => {
    try {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setWorkflows((prev) =>
        prev.map((w) => (w.id === id ? { ...w, status: "active" as const } : w))
      );
      message.success("工作流已激活");
      loadStats();
    } catch {
      message.error("激活失败");
    }
  };

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的工作流");
      return;
    }

    modal.confirm({
      title: "批量删除确认",
      content: `确定要删除选中的 ${selectedRowKeys.length} 个工作流吗？此操作不可恢复。`,
      okText: "确定删除",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.filter((w) => !selectedRowKeys.includes(w.id))
          );
          setSelectedRowKeys([]);
          message.success(`成功删除 ${selectedRowKeys.length} 个工作流`);
          loadStats();
        } catch {
          message.error("批量删除失败");
        }
      },
    });
  };

  const handleBatchActivate = () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要激活的工作流");
      return;
    }

    modal.confirm({
      title: "批量激活确认",
      content: `确定要激活选中的 ${selectedRowKeys.length} 个工作流吗？`,
      okText: "确定激活",
      cancelText: "取消",
      onOk: async () => {
        try {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.map((w) =>
              selectedRowKeys.includes(w.id)
                ? { ...w, status: "active" as const }
                : w
            )
          );
          setSelectedRowKeys([]);
          message.success(`成功激活 ${selectedRowKeys.length} 个工作流`);
          loadStats();
        } catch {
          message.error("批量激活失败");
        }
      },
    });
  };

  const handleBatchStop = () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要停用的工作流");
      return;
    }

    modal.confirm({
      title: "批量停用确认",
      content: `确定要停用选中的 ${selectedRowKeys.length} 个工作流吗？`,
      okText: "确定停用",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.map((w) =>
              selectedRowKeys.includes(w.id)
                ? { ...w, status: "inactive" as const }
                : w
            )
          );
          setSelectedRowKeys([]);
          message.success(`成功停用 ${selectedRowKeys.length} 个工作流`);
          loadStats();
        } catch {
          message.error("批量停用失败");
        }
      },
    });
  };

  const handleBatchExport = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要导出的工作流");
      return;
    }

    try {
      const selectedWorkflows = workflows.filter((w) =>
        selectedRowKeys.includes(w.id)
      );
      const exportData = {
        workflows: selectedWorkflows,
        exportTime: new Date().toISOString(),
        version: "1.0",
        count: selectedWorkflows.length,
      };

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `workflows_batch_export_${
        new Date().toISOString().split("T")[0]
      }.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      message.success(`成功导出 ${selectedRowKeys.length} 个工作流`);
      setSelectedRowKeys([]);
    } catch {
      message.error("批量导出失败");
    }
  };

  const getStatusColor = (status: string) => {
    const colors = {
      draft: "orange",
      active: "green",
      inactive: "gray",
      archived: "red",
    };
    return colors[status as keyof typeof colors] || "default";
  };

  const getStatusText = (status: string) => {
    const texts = {
      draft: "草稿",
      active: "激活",
      inactive: "停用",
      archived: "归档",
    };
    return texts[status as keyof typeof texts] || status;
  };

  const columns = [
    {
      title: "工作流名称",
      dataIndex: "name",
      key: "name",
      render: (name: string, record: Workflow) => (
        <div>
          <div className="font-medium text-gray-900">{name}</div>
          <div className="text-sm text-gray-500">{record.description}</div>
        </div>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      width: 120,
      render: (category: string) => <Tag color="blue">{category}</Tag>,
    },
    {
      title: "版本",
      dataIndex: "version",
      key: "version",
      width: 100,
      render: (version: string) => (
        <span className="font-mono text-sm">{version}</span>
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
      title: "BPMN状态",
      key: "bpmn_status",
      width: 120,
      render: (record: Workflow) => (
        <Tag color={record.bpmn_xml ? "green" : "orange"}>
          {record.bpmn_xml ? "已定义" : "未定义"}
        </Tag>
      ),
    },
    {
      title: "实例统计",
      key: "instances",
      width: 150,
      render: (record: Workflow) => (
        <div className="text-sm">
          <div>总数: {record.instances_count}</div>
          <div>运行中: {record.running_instances}</div>
        </div>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => (
        <div className="text-sm">
          {new Date(date).toLocaleDateString("zh-CN")}
        </div>
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 200,
      render: (record: Workflow) => (
        <Space>
          <Tooltip title="设计工作流">
            <Button
              type="text"
              icon={<GitBranch className="w-4 h-4" />}
              onClick={() => handleDesignWorkflow(record)}
            />
          </Tooltip>
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => window.open(`/workflow/${record.id}`, "_blank")}
            />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                {
                  key: "edit",
                  label: "编辑",
                  icon: <Edit className="w-4 h-4" />,
                  onClick: () => handleEditWorkflow(record),
                },
                {
                  key: "deploy",
                  label: record.status === "draft" ? "部署" : "重新部署",
                  icon: <PlayCircle className="w-4 h-4" />,
                  onClick: () => handleDeployWorkflow(record.id),
                  disabled: record.status === "active",
                },
                {
                  key: "activate",
                  label: "激活",
                  icon: <CheckCircle className="w-4 h-4" />,
                  onClick: () => handleActivateWorkflow(record.id),
                  style: {
                    display: record.status === "inactive" ? "block" : "none",
                  },
                },
                {
                  key: "stop",
                  label: "停用",
                  icon: <PauseCircle className="w-4 h-4" />,
                  onClick: () => handleStopWorkflow(record.id),
                  style: {
                    display: record.status === "active" ? "block" : "none",
                  },
                },
                {
                  type: "divider" as const,
                },
                {
                  key: "copy",
                  label: "复制",
                  icon: <Copy className="w-4 h-4" />,
                  onClick: () => handleCopyWorkflow(record),
                },
                {
                  key: "export",
                  label: "导出",
                  icon: <Download className="w-4 h-4" />,
                  onClick: () => handleExportWorkflow(record),
                },
                {
                  key: "instances",
                  label: "查看实例",
                  icon: <BarChart3 className="w-4 h-4" />,
                  onClick: () =>
                    window.open(
                      `/workflow/instances?workflowId=${record.id}`,
                      "_blank"
                    ),
                },
                {
                  key: "versions",
                  label: "版本管理",
                  icon: <GitBranch className="w-4 h-4" />,
                  onClick: () =>
                    window.open(
                      `/workflow/versions?workflowId=${record.id}`,
                      "_blank"
                    ),
                },
                {
                  key: "viewBPMN",
                  label: "查看BPMN",
                  icon: <Code className="w-4 h-4" />,
                  onClick: () => handleViewBPMN(record),
                  disabled: !record.bpmn_xml,
                },
                {
                  type: "divider" as const,
                },
                {
                  key: "delete",
                  label: "删除",
                  icon: <Trash2 className="w-4 h-4" />,
                  danger: true,
                  onClick: () => handleDeleteWorkflow(record.id),
                  disabled:
                    record.status === "active" && record.running_instances > 0,
                },
              ].filter((item) => item.style?.display !== "none"), // 过滤隐藏项目
            }}
          >
            <Button type="text" icon={<MoreHorizontal className="w-4 h-4" />} />
          </Dropdown>
        </Space>
      ),
    },
  ];

  const filteredWorkflows = workflows.filter((workflow) => {
    if (filters.status && workflow.status !== filters.status) return false;
    if (filters.category && workflow.category !== filters.category)
      return false;
    if (
      filters.keyword &&
      !workflow.name.toLowerCase().includes(filters.keyword.toLowerCase())
    )
      return false;
    return true;
  });

  return (
    <>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总工作流"
              value={stats.total}
              prefix={<FileText className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
            <div className="mt-2 text-xs text-gray-500">
              激活 {stats.active} | 草稿 {stats.draft} | 停用 {stats.inactive}
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="运行中实例"
              value={stats.running}
              prefix={<Clock className="w-5 h-5" />}
              valueStyle={{ color: "#faad14" }}
            />
            <div className="mt-2 text-xs text-gray-500">
              今日新增 {stats.todayInstances} 个实例
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="已完成实例"
              value={stats.completed}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
            <div className="mt-2 text-xs text-gray-500">
              完成率{" "}
              {stats.total > 0
                ? Math.round(
                    (stats.completed / (stats.completed + stats.running)) * 100
                  )
                : 0}
              %
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="平均执行时间"
              value={stats.avgExecutionTime}
              suffix="分钟"
              prefix={<BarChart3 className="w-5 h-5" />}
              valueStyle={{ color: "#722ed1" }}
            />
            <div className="mt-2 text-xs text-gray-500">
              {stats.avgExecutionTime < 60 ? "执行效率良好" : "可优化空间"}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className="enterprise-card mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <Input
              placeholder="搜索工作流..."
              prefix={<Search className="w-4 h-4" />}
              value={filters.keyword}
              onChange={(e) =>
                setFilters((prev) => ({ ...prev, keyword: e.target.value }))
              }
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="状态筛选"
              value={filters.status}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, status: value }))
              }
              allowClear
              style={{ width: "100%" }}
            >
              <Option value="draft">草稿</Option>
              <Option value="active">激活</Option>
              <Option value="inactive">停用</Option>
              <Option value="archived">归档</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="分类筛选"
              value={filters.category}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, category: value }))
              }
              allowClear
              style={{ width: "100%" }}
            >
              <Option value="审批流程">审批流程</Option>
              <Option value="事件处理">事件处理</Option>
              <Option value="变更管理">变更管理</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={8}>
            <Space>
              <Dropdown
                menu={{
                  items: [
                    {
                      key: "ticket_approval",
                      label: "工单审批流程",
                      icon: <GitBranch className="w-4 h-4" />,
                      onClick: () => router.push("/workflow/ticket-approval"),
                    },
                    {
                      key: "general",
                      label: "通用工作流",
                      icon: <GitBranch className="w-4 h-4" />,
                      onClick: handleCreateWorkflow,
                    },
                    {
                      key: "bpmn",
                      label: "BPMN工作流",
                      icon: <Code className="w-4 h-4" />,
                      onClick: () => {
                        setEditingWorkflow(null);
                        setDesignerVisible(true);
                      },
                    },
                  ],
                }}
                trigger={["click"]}
              >
                <Button type="primary" icon={<Plus className="w-4 h-4" />}>
                  新建工作流 <MoreHorizontal className="w-3 h-3 ml-1" />
                </Button>
              </Dropdown>
              <Button
                icon={<Upload className="w-4 h-4" />}
                onClick={handleImportWorkflow}
              >
                导入
              </Button>
              <Button
                icon={<RefreshCw className="w-4 h-4" />}
                onClick={loadWorkflows}
              >
                刷新
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 批量操作工具栏 */}
      {selectedRowKeys.length > 0 && (
        <Card className="enterprise-card mb-4">
          <Alert
            message={
              <Space>
                <span>已选择 {selectedRowKeys.length} 个工作流</span>
                <Button size="small" onClick={() => setSelectedRowKeys([])}>
                  取消选择
                </Button>
              </Space>
            }
            type="info"
            action={
              <Space>
                <Button
                  size="small"
                  type="primary"
                  onClick={handleBatchActivate}
                >
                  批量激活
                </Button>
                <Button size="small" onClick={handleBatchStop}>
                  批量停用
                </Button>
                <Button size="small" onClick={handleBatchExport}>
                  批量导出
                </Button>
                <Button size="small" danger onClick={handleBatchDelete}>
                  批量删除
                </Button>
              </Space>
            }
          />
        </Card>
      )}

      {/* 工作流表格 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={filteredWorkflows}
          rowKey="id"
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: (selectedRowKeys: React.Key[]) => {
              setSelectedRowKeys(selectedRowKeys.map((key) => Number(key)));
            },
            selections: [
              Table.SELECTION_ALL,
              Table.SELECTION_INVERT,
              Table.SELECTION_NONE,
              {
                key: "select-active",
                text: "选择激活的",
                onSelect: () => {
                  const activeKeys = filteredWorkflows
                    .filter((w) => w.status === "active")
                    .map((w) => w.id);
                  setSelectedRowKeys(activeKeys);
                },
              },
              {
                key: "select-draft",
                text: "选择草稿",
                onSelect: () => {
                  const draftKeys = filteredWorkflows
                    .filter((w) => w.status === "draft")
                    .map((w) => w.id);
                  setSelectedRowKeys(draftKeys);
                },
              },
            ],
          }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          scroll={{ x: 1200 }}
        />
      </Card>

      {/* 创建/编辑工作流模态框 */}
      <Modal
        title={editingWorkflow ? "编辑工作流" : "新建工作流"}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
        destroyOnHidden
      >
        <Form
          layout="vertical"
          initialValues={editingWorkflow || {}}
          onFinish={async () => {
            try {
              message.success(
                editingWorkflow ? "工作流更新成功" : "工作流创建成功"
              );
              setModalVisible(false);
              loadWorkflows();
            } catch {
              message.error("操作失败");
            }
          }}
        >
          <Form.Item
            name="name"
            label="工作流名称"
            rules={[{ required: true, message: "请输入工作流名称" }]}
          >
            <Input placeholder="请输入工作流名称" />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入工作流描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="category"
                label="分类"
                rules={[{ required: true, message: "请选择分类" }]}
              >
                <Select placeholder="选择分类">
                  <Option value="审批流程">审批流程</Option>
                  <Option value="事件处理">事件处理</Option>
                  <Option value="变更管理">变更管理</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label="状态">
                <Select placeholder="选择状态">
                  <Option value="draft">草稿</Option>
                  <Option value="active">激活</Option>
                  <Option value="inactive">停用</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingWorkflow ? "更新" : "创建"}
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 工作流设计器 */}
      <Modal
        title={`BPMN工作流设计器 - ${editingWorkflow?.name || "新建工作流"}`}
        open={designerVisible}
        onCancel={() => setDesignerVisible(false)}
        footer={null}
        width="95%"
        style={{ top: 10 }}
        destroyOnHidden
      >
        <EnhancedBPMNDesigner
          initialXML={editingWorkflow?.bpmn_xml || ""}
          onSave={async (xml: string) => {
            try {
              if (editingWorkflow) {
                // 更新现有工作流
                await WorkflowAPI.updateWorkflow(editingWorkflow.id, {
                  bpmn_xml: xml,
                });
                message.success("工作流保存成功");
              } else {
                // 创建新工作流
                await WorkflowAPI.createWorkflow({
                  name: "新建BPMN工作流",
                  description: "通过BPMN设计器创建的工作流",
                  category: "审批流程",
                  bpmn_xml: xml,
                });
                message.success("工作流创建成功");
              }
              setDesignerVisible(false);
              loadWorkflows();
            } catch (error) {
              message.error("保存失败: " + (error as Error).message);
            }
          }}
          onDeploy={async (xml: string) => {
            try {
              if (editingWorkflow) {
                // 先保存BPMN XML
                await WorkflowAPI.updateWorkflow(editingWorkflow.id, {
                  bpmn_xml: xml,
                });
                // 然后部署工作流
                await WorkflowAPI.deployWorkflow(editingWorkflow.id);
                message.success("工作流部署成功");
              } else {
                // 创建并部署新工作流
                const newWorkflow = await WorkflowAPI.createWorkflow({
                  name: "新建BPMN工作流",
                  description: "通过BPMN设计器创建的工作流",
                  category: "审批流程",
                  bpmn_xml: xml,
                });
                await WorkflowAPI.deployWorkflow(newWorkflow.id);
                message.success("工作流创建并部署成功");
              }
              setDesignerVisible(false);
              loadWorkflows();
            } catch (error) {
              message.error("部署失败: " + (error as Error).message);
            }
          }}
          height={700}
          showPropertiesPanel={true}
        />
      </Modal>
    </>
  );
};

export default WorkflowManagementPage;
