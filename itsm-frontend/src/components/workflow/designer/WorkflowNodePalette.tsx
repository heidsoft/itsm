// 工作流节点面板组件
// Workflow Node Palette Component - 节点选择面板

'use client';

import React from 'react';
import { Card, Typography } from 'antd';

const { Text } = Typography;

// 节点类型定义
interface PaletteNode {
  id: string;
  name: string;
  icon: string;
  category: 'event' | 'task' | 'gateway' | 'boundary';
  description: string;
}

// 节点分类
const nodeCategories: { key: string; name: string; nodes: PaletteNode[] }[] = [
  {
    key: 'event',
    name: '事件',
    nodes: [
      {
        id: 'startEvent',
        name: '开始事件',
        icon: '⚪',
        category: 'event',
        description: '流程开始',
      },
      { id: 'endEvent', name: '结束事件', icon: '🔴', category: 'event', description: '流程结束' },
      {
        id: 'intermediateThrowEvent',
        name: '中间事件',
        icon: '🟡',
        category: 'event',
        description: '中间抛出事件',
      },
      {
        id: 'intermediateCatchEvent',
        name: '中间捕获事件',
        icon: '🟢',
        category: 'event',
        description: '中间捕获事件',
      },
    ],
  },
  {
    key: 'task',
    name: '任务',
    nodes: [
      {
        id: 'userTask',
        name: '用户任务',
        icon: '👤',
        category: 'task',
        description: '需要人工处理的任务',
      },
      {
        id: 'serviceTask',
        name: '服务任务',
        icon: '⚙️',
        category: 'task',
        description: '自动执行的服务',
      },
      {
        id: 'scriptTask',
        name: '脚本任务',
        icon: '📜',
        category: 'task',
        description: '执行脚本的任务',
      },
      {
        id: 'manualTask',
        name: '手工任务',
        icon: '✋',
        category: 'task',
        description: '手工处理的任务',
      },
    ],
  },
  {
    key: 'gateway',
    name: '网关',
    nodes: [
      {
        id: 'exclusiveGateway',
        name: '排他网关',
        icon: '🔀',
        category: 'gateway',
        description: '只能选择一条分支',
      },
      {
        id: 'parallelGateway',
        name: '并行网关',
        icon: '➕',
        category: 'gateway',
        description: '并行执行多条分支',
      },
      {
        id: 'inclusiveGateway',
        name: '包容网关',
        icon: '🔁',
        category: 'gateway',
        description: '根据条件选择分支',
      },
      {
        id: 'eventBasedGateway',
        name: '事件网关',
        icon: '⚡',
        category: 'gateway',
        description: '基于事件的分支',
      },
    ],
  },
  {
    key: 'subprocess',
    name: '子流程',
    nodes: [
      {
        id: 'subProcess',
        name: '子流程',
        icon: '📦',
        category: 'task',
        description: '嵌入的子流程',
      },
      {
        id: 'callActivity',
        name: '调用活动',
        icon: '📞',
        category: 'task',
        description: '调用外部流程',
      },
    ],
  },
];

export default function WorkflowNodePalette() {
  // 节点拖拽开始处理
  const handleDragStart = (e: React.DragEvent, node: PaletteNode) => {
    e.dataTransfer.setData('application/bpmn-node', JSON.stringify(node));
    e.dataTransfer.effectAllowed = 'copy';
  };

  return (
    <div className="w-64 bg-white border-r border-gray-200 h-full overflow-y-auto">
      <div className="p-4">
        <Text strong className="block mb-4 text-base">
          节点面板
        </Text>

        <div className="space-y-4">
          {nodeCategories.map(category => (
            <div key={category.key}>
              <Text type="secondary" className="block mb-2 text-xs uppercase">
                {category.name}
              </Text>
              <div className="grid grid-cols-2 gap-2">
                {category.nodes.map(node => (
                  <div
                    key={node.id}
                    draggable
                    onDragStart={e => handleDragStart(e, node)}
                    className="flex flex-col items-center justify-center p-3 border border-gray-200 rounded-lg cursor-move hover:border-blue-400 hover:bg-blue-50 transition-colors"
                  >
                    <span className="text-xl mb-1">{node.icon}</span>
                    <Text className="text-xs text-center">{node.name}</Text>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* 使用说明 */}
        <div className="mt-6 p-3 bg-gray-50 rounded-lg">
          <Text type="secondary" className="text-xs">
            提示: 从节点面板拖拽元素到画布上添加节点。双击节点可以编辑属性。
          </Text>
        </div>
      </div>
    </div>
  );
}
