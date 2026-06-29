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
  category: 'event' | 'task' | 'gateway' | 'subprocess' | 'boundary';
  description: string;
  bpmnType: string;
}

// 节点分类
const nodeCategories: { key: string; name: string; nodes: PaletteNode[] }[] = [
  {
    key: 'event',
    name: '事件',
    nodes: [
      // 启动事件
      {
        id: 'startEvent',
        name: '开始事件',
        icon: '⚪',
        category: 'event',
        description: '流程开始',
        bpmnType: 'bpmn:StartEvent'
      },
      {
        id: 'timerStartEvent',
        name: '定时启动',
        icon: '⏰',
        category: 'event',
        description: '定时触发流程',
        bpmnType: 'bpmn:StartEvent'
      },
      {
        id: 'messageStartEvent',
        name: '消息启动',
        icon: '✉️',
        category: 'event',
        description: '接收消息启动流程',
        bpmnType: 'bpmn:StartEvent'
      },
      {
        id: 'signalStartEvent',
        name: '信号启动',
        icon: '📡',
        category: 'event',
        description: '接收信号启动流程',
        bpmnType: 'bpmn:StartEvent'
      },
      // 结束事件
      {
        id: 'endEvent',
        name: '结束事件',
        icon: '🔴',
        category: 'event',
        description: '流程结束',
        bpmnType: 'bpmn:EndEvent'
      },
      {
        id: 'errorEndEvent',
        name: '错误结束',
        icon: '❌',
        category: 'event',
        description: '抛出错误结束流程',
        bpmnType: 'bpmn:EndEvent'
      },
      {
        id: 'terminateEndEvent',
        name: '终止结束',
        icon: '⏹️',
        category: 'event',
        description: '立即终止所有流程分支',
        bpmnType: 'bpmn:EndEvent'
      },
      {
        id: 'cancelEndEvent',
        name: '取消结束',
        icon: '🚫',
        category: 'event',
        description: '取消事务子流程',
        bpmnType: 'bpmn:EndEvent'
      },
      // 中间事件
      {
        id: 'intermediateThrowEvent',
        name: '中间抛出',
        icon: '🟡',
        category: 'event',
        description: '中间抛出事件',
        bpmnType: 'bpmn:IntermediateThrowEvent'
      },
      {
        id: 'intermediateCatchEvent',
        name: '中间捕获',
        icon: '🟢',
        category: 'event',
        description: '中间捕获事件',
        bpmnType: 'bpmn:IntermediateCatchEvent'
      },
      {
        id: 'timerIntermediateCatchEvent',
        name: '定时捕获',
        icon: '⏱️',
        category: 'event',
        description: '等待指定时间后继续',
        bpmnType: 'bpmn:IntermediateCatchEvent'
      },
      {
        id: 'messageIntermediateCatchEvent',
        name: '消息捕获',
        icon: '📩',
        category: 'event',
        description: '等待接收消息后继续',
        bpmnType: 'bpmn:IntermediateCatchEvent'
      },
      {
        id: 'signalIntermediateCatchEvent',
        name: '信号捕获',
        icon: '📶',
        category: 'event',
        description: '等待接收信号后继续',
        bpmnType: 'bpmn:IntermediateCatchEvent'
      }
    ],
  },
  {
    key: 'boundary',
    name: '边界事件',
    nodes: [
      {
        id: 'timerBoundaryEvent',
        name: '定时边界',
        icon: '⏰',
        category: 'boundary',
        description: '附加到任务上的定时触发事件',
        bpmnType: 'bpmn:BoundaryEvent'
      },
      {
        id: 'messageBoundaryEvent',
        name: '消息边界',
        icon: '✉️',
        category: 'boundary',
        description: '附加到任务上的消息触发事件',
        bpmnType: 'bpmn:BoundaryEvent'
      },
      {
        id: 'signalBoundaryEvent',
        name: '信号边界',
        icon: '📡',
        category: 'boundary',
        description: '附加到任务上的信号触发事件',
        bpmnType: 'bpmn:BoundaryEvent'
      },
      {
        id: 'errorBoundaryEvent',
        name: '错误边界',
        icon: '❌',
        category: 'boundary',
        description: '捕获任务抛出的错误',
        bpmnType: 'bpmn:BoundaryEvent'
      },
      {
        id: 'escalationBoundaryEvent',
        name: '升级边界',
        icon: '⬆️',
        category: 'boundary',
        description: '处理任务升级事件',
        bpmnType: 'bpmn:BoundaryEvent'
      },
      {
        id: 'conditionalBoundaryEvent',
        name: '条件边界',
        icon: '🔍',
        category: 'boundary',
        description: '满足条件时触发',
        bpmnType: 'bpmn:BoundaryEvent'
      }
    ]
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
        bpmnType: 'bpmn:UserTask'
      },
      {
        id: 'serviceTask',
        name: '服务任务',
        icon: '⚙️',
        category: 'task',
        description: '自动执行的服务',
        bpmnType: 'bpmn:ServiceTask'
      },
      {
        id: 'scriptTask',
        name: '脚本任务',
        icon: '📜',
        category: 'task',
        description: '执行脚本的任务',
        bpmnType: 'bpmn:ScriptTask'
      },
      {
        id: 'businessRuleTask',
        name: '规则任务',
        icon: '📋',
        category: 'task',
        description: '执行业务规则',
        bpmnType: 'bpmn:BusinessRuleTask'
      },
      {
        id: 'sendTask',
        name: '发送任务',
        icon: '📤',
        category: 'task',
        description: '发送消息或通知',
        bpmnType: 'bpmn:SendTask'
      },
      {
        id: 'receiveTask',
        name: '接收任务',
        icon: '📥',
        category: 'task',
        description: '等待接收消息',
        bpmnType: 'bpmn:ReceiveTask'
      },
      {
        id: 'manualTask',
        name: '手工任务',
        icon: '✋',
        category: 'task',
        description: '手工处理的任务',
        bpmnType: 'bpmn:ManualTask'
      },
      {
        id: 'mailTask',
        name: '邮件任务',
        icon: '📧',
        category: 'task',
        description: '发送邮件',
        bpmnType: 'bpmn:ServiceTask'
      }
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
        bpmnType: 'bpmn:ExclusiveGateway'
      },
      {
        id: 'parallelGateway',
        name: '并行网关',
        icon: '➕',
        category: 'gateway',
        description: '并行执行多条分支',
        bpmnType: 'bpmn:ParallelGateway'
      },
      {
        id: 'inclusiveGateway',
        name: '包容网关',
        icon: '🔁',
        category: 'gateway',
        description: '根据条件选择分支',
        bpmnType: 'bpmn:InclusiveGateway'
      },
      {
        id: 'eventBasedGateway',
        name: '事件网关',
        icon: '⚡',
        category: 'gateway',
        description: '基于事件的分支',
        bpmnType: 'bpmn:EventBasedGateway'
      },
      {
        id: 'complexGateway',
        name: '复杂网关',
        icon: '⚙️🔀',
        category: 'gateway',
        description: '复杂条件的分支控制',
        bpmnType: 'bpmn:ComplexGateway'
      }
    ],
  },
  {
    key: 'subprocess',
    name: '子流程',
    nodes: [
      {
        id: 'subProcess',
        name: '嵌入式子流程',
        icon: '📦',
        category: 'subprocess',
        description: '嵌入在当前流程中的子流程',
        bpmnType: 'bpmn:SubProcess'
      },
      {
        id: 'transactionSubProcess',
        name: '事务子流程',
        icon: '🔄',
        category: 'subprocess',
        description: '具有事务特性的子流程',
        bpmnType: 'bpmn:Transaction'
      },
      {
        id: 'eventSubProcess',
        name: '事件子流程',
        icon: '⚡📦',
        category: 'subprocess',
        description: '由事件触发的子流程',
        bpmnType: 'bpmn:SubProcess'
      },
      {
        id: 'callActivity',
        name: '调用活动',
        icon: '📞',
        category: 'subprocess',
        description: '调用外部独立流程',
        bpmnType: 'bpmn:CallActivity'
      }
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
                    title={node.description}
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
