import React, { useState } from "react";
import { Button, Card, Space } from "antd";
import { Plus, RotateCcw } from "lucide-react";
import LoadingEmptyError, { useLoadingEmptyError } from "./LoadingEmptyError";

// 示例1: 基础用法
export const BasicExample = () => {
  const [state, setState] = useState<"loading" | "empty" | "error" | "success">(
    "loading"
  );

  const simulateLoading = () => setState("loading");
  const simulateEmpty = () => setState("empty");
  const simulateError = () => setState("error");
  const simulateSuccess = () => setState("success");

  return (
    <Card title="基础用法示例" className="mb-6">
      <Space className="mb-4">
        <Button onClick={simulateLoading}>加载状态</Button>
        <Button onClick={simulateEmpty}>空状态</Button>
        <Button onClick={simulateError}>错误状态</Button>
        <Button onClick={simulateSuccess}>成功状态</Button>
      </Space>

      <LoadingEmptyError
        state={state}
        loadingText="正在加载数据..."
        empty={{
          title: "暂无数据",
          description: "当前没有相关数据，您可以创建新的记录",
          actions: [
            {
              text: "创建记录",
              icon: <Plus />,
              onClick: () => console.log("创建记录"),
              type: "primary",
            },
          ],
        }}
        error={{
          title: "加载失败",
          description: "数据加载过程中发生错误，请稍后重试",
          details: "错误代码: 500, 服务器内部错误",
          onRetry: () => console.log("重试"),
          actions: [
            {
              text: "联系支持",
              onClick: () => console.log("联系支持"),
              type: "default",
            },
          ],
        }}
      >
        <div className="p-4 bg-green-50 border border-green-200 rounded">
          <h3 className="text-green-800">数据加载成功！</h3>
          <p className="text-green-600">这里是成功状态下的内容展示区域</p>
        </div>
      </LoadingEmptyError>
    </Card>
  );
};

// 示例2: 使用Hook的用法
export const HookExample = () => {
  const { state, data, error, refetch } = useLoadingEmptyError(
    async () => {
      // 模拟API调用
      await new Promise((resolve) => setTimeout(resolve, 2000));

      // 模拟随机结果
      const random = Math.random();
      if (random < 0.3) {
        throw new Error("网络连接失败");
      } else if (random < 0.6) {
        return []; // 空数据
      } else {
        return [
          { id: 1, name: "示例数据1" },
          { id: 2, name: "示例数据2" },
        ];
      }
    },
    {
      autoFetch: true,
      onSuccess: (data) => console.log("数据加载成功:", data),
      onError: (error) => console.log("数据加载失败:", error),
    }
  );

  return (
    <Card title="Hook用法示例" className="mb-6">
      <Button icon={<RotateCcw />} onClick={refetch} className="mb-4">
        重新加载
      </Button>

      <LoadingEmptyError
        state={state}
        loadingText="正在获取数据..."
        empty={{
          title: "暂无数据",
          description: "当前没有找到相关数据",
          actions: [
            {
              text: "刷新",
              icon: <RotateCcw />,
              onClick: refetch,
            },
          ],
        }}
        error={{
          title: "加载失败",
          description: error?.message || "未知错误",
          onRetry: refetch,
        }}
      >
        <div className="space-y-2">
          {data?.map((item: any) => (
            <div
              key={item.id}
              className="p-3 bg-blue-50 border border-blue-200 rounded"
            >
              {item.name}
            </div>
          ))}
        </div>
      </LoadingEmptyError>
    </Card>
  );
};

// 示例3: 工单列表场景
export const TicketListExample = () => {
  const [state, setState] = useState<"loading" | "empty" | "error" | "success">(
    "loading"
  );

  const loadTickets = () => setState("loading");
  const showEmpty = () => setState("empty");
  const showError = () => setState("error");
  const showSuccess = () => setState("success");

  return (
    <Card title="工单列表场景示例" className="mb-6">
      <Space className="mb-4">
        <Button onClick={loadTickets}>加载工单</Button>
        <Button onClick={showEmpty}>显示空状态</Button>
        <Button onClick={showError}>显示错误</Button>
        <Button onClick={showSuccess}>显示成功</Button>
      </Space>

      <LoadingEmptyError
        state={state}
        type="tickets"
        loadingText="正在加载工单列表..."
        empty={{
          title: "暂无工单",
          description: "当前没有工单记录，您可以创建新的工单",
          actions: [
            {
              text: "创建工单",
              icon: <Plus />,
              onClick: () => console.log("创建工单"),
              type: "primary",
            },
            {
              text: "刷新",
              icon: <RotateCcw />,
              onClick: loadTickets,
            },
          ],
        }}
        error={{
          title: "工单加载失败",
          description: "无法获取工单列表，请检查网络连接",
          onRetry: loadTickets,
        }}
        bordered
        minHeight={300}
      >
        <div className="space-y-3">
          <div className="p-4 bg-white border border-gray-200 rounded-lg shadow-sm">
            <h4 className="font-medium">工单 #TICKET-001</h4>
            <p className="text-gray-600">系统登录问题</p>
            <div className="mt-2">
              <span className="inline-block px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded">
                处理中
              </span>
            </div>
          </div>
          <div className="p-4 bg-white border border-gray-200 rounded-lg shadow-sm">
            <h4 className="font-medium">工单 #TICKET-002</h4>
            <p className="text-gray-600">打印机故障</p>
            <div className="mt-2">
              <span className="inline-block px-2 py-1 text-xs bg-green-100 text-green-800 rounded">
                已解决
              </span>
            </div>
          </div>
        </div>
      </LoadingEmptyError>
    </Card>
  );
};

// 示例4: 不同模块的空状态
export const ModuleExamples = () => {
  const modules = [
    { key: "incidents", name: "事件管理" },
    { key: "problems", name: "问题管理" },
    { key: "changes", name: "变更管理" },
    { key: "cmdb", name: "配置管理" },
    { key: "workflow", name: "工作流" },
  ];

  return (
    <Card title="不同模块的空状态示例">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {modules.map((module) => (
          <div
            key={module.key}
            className="border border-gray-200 rounded-lg p-4"
          >
            <h4 className="font-medium mb-3">{module.name}</h4>
            <LoadingEmptyError
              state="empty"
              type={module.key as any}
              bordered={false}
              minHeight={150}
            />
          </div>
        ))}
      </div>
    </Card>
  );
};

// 主示例组件
export const LoadingEmptyErrorExamples = () => {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">
        LoadingEmptyError 组件使用示例
      </h1>

      <BasicExample />
      <HookExample />
      <TicketListExample />
      <ModuleExamples />
    </div>
  );
};

export default LoadingEmptyErrorExamples;
