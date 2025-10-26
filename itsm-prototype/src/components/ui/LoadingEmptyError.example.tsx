import React, { useState } from "react";
import { Button, Card, Space } from "antd";
import LoadingEmptyError from "./LoadingEmptyError";

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
          actionText: "创建记录",
          onAction: () => console.log("创建记录"),
          showAction: true,
        }}
        error={{
          title: "加载失败",
          description: "数据加载过程中发生错误，请稍后重试",
          actionText: "重试",
          onAction: () => console.log("重试"),
          showRetry: true,
          showAction: true,
        }}
      >
        {state === "success" && (
          <div className="p-4 text-center">
            <h3 className="text-lg font-semibold text-green-600">数据加载成功！</h3>
            <p className="text-gray-600">这里显示实际的数据内容</p>
          </div>
        )}
      </LoadingEmptyError>
    </Card>
  );
};

// 示例2: 工单列表示例
interface Ticket {
  id: number;
  title: string;
}

export const TicketListExample = () => {
  const [loading, setLoading] = useState(false);
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [error, setError] = useState<string | null>(null);

  const fetchTickets = async () => {
    setLoading(true);
    setError(null);
    
    // 模拟API调用
    setTimeout(() => {
      const random = Math.random();
      if (random < 0.3) {
        setError("网络错误");
        setTickets([]);
      } else if (random < 0.6) {
        setTickets([]);
      } else {
        setTickets([{ id: 1, title: "示例工单" }]);
      }
      setLoading(false);
    }, 2000);
  };

  const getState = () => {
    if (loading) return "loading";
    if (error) return "error";
    if (tickets.length === 0) return "empty";
    return "success";
  };

  return (
    <Card title="工单列表示例" className="mb-6">
      <Space className="mb-4">
        <Button onClick={fetchTickets} loading={loading}>
          获取工单列表
        </Button>
      </Space>

      <LoadingEmptyError
        state={getState()}
        loadingText="正在加载工单列表..."
        empty={{
          title: "暂无工单",
          description: "当前没有工单数据，点击下方按钮创建第一个工单",
          actionText: "创建工单",
          onAction: () => console.log("创建工单"),
          showAction: true,
        }}
        error={{
          title: "加载失败",
          description: error || "工单列表加载失败，请稍后重试",
          actionText: "重试",
          onAction: fetchTickets,
          showRetry: true,
          showAction: true,
        }}
      >
        {tickets.length > 0 && (
          <div className="space-y-2">
            {tickets.map((ticket) => (
              <div key={ticket.id} className="p-3 border rounded">
                {ticket.title}
              </div>
            ))}
          </div>
        )}
      </LoadingEmptyError>
    </Card>
  );
};

// 示例3: 模块化示例
export const ModuleExamples = () => {
  return (
    <div className="space-y-6">
      <Card title="不同模块的空状态">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <LoadingEmptyError
            state="empty"
            empty={{
              title: "暂无事件",
              description: "当前没有事件数据",
              actionText: "创建事件",
              onAction: () => console.log("创建事件"),
              showAction: true,
            }}
            minHeight={150}
          />
          
          <LoadingEmptyError
            state="empty"
            empty={{
              title: "暂无问题",
              description: "当前没有问题数据",
              actionText: "创建问题",
              onAction: () => console.log("创建问题"),
              showAction: true,
            }}
            minHeight={150}
          />
        </div>
      </Card>
    </div>
  );
};

// 主示例组件
export const LoadingEmptyErrorExamples = () => {
  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-bold mb-6">LoadingEmptyError 组件示例</h1>
      <BasicExample />
      <TicketListExample />
      <ModuleExamples />
    </div>
  );
};

export default LoadingEmptyErrorExamples;
