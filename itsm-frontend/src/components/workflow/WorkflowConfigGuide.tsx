'use client';

import React from 'react';
import Link from 'next/link';
import { Card, Space, Steps, Tag, Typography } from 'antd';
import { ArrowRight, ClipboardList, Link2, Workflow } from 'lucide-react';

const { Text } = Typography;

export type ConfigSurface = 'designer' | 'approval' | 'binding';

const surfaces = [
  {
    key: 'designer',
    title: '设计流程',
    href: '/admin/workflows',
    label: '流程定义',
    icon: Workflow,
    description: '新建流程，进设计器编排节点、分支与超时。',
  },
  {
    key: 'approval',
    title: '配置审批链',
    href: '/admin/approval-chains',
    label: '审批链配置',
    icon: Link2,
    description: '为审批节点按业务类型配置审批人、会签/或签与兜底。',
  },
  {
    key: 'binding',
    title: '绑定工单类型',
    href: '/tickets/types',
    label: '工单类型',
    icon: ClipboardList,
    description: '把流程与审批策略绑定到工单类型，决定哪些工单走这条流程。',
  },
] as const;

export interface WorkflowConfigGuideProps {
  /** 当前所在的配置面，用于高亮“你在这里” */
  current: ConfigSurface;
  style?: React.CSSProperties;
}

/**
 * 流程配置三角导航：流程定义、审批链、工单类型绑定三个页面互相串联，
 * 让用户在任一配置面都知道上一步/下一步去哪，而不是在孤立页面间迷失。
 */
export const WorkflowConfigGuide: React.FC<WorkflowConfigGuideProps> = ({ current, style }) => {
  const currentIndex = surfaces.findIndex(s => s.key === current);
  const currentSurface = surfaces[currentIndex];

  return (
    <Card
      size="small"
      style={style}
      styles={{ body: { padding: '12px 16px' } }}
      title={
        <Space size={8}>
          <Workflow className="h-4 w-4 text-blue-600" />
          <span className="text-sm">流程配置路线</span>
          <Tag color="blue">你在这里：{currentSurface.title}</Tag>
        </Space>
      }
    >
      <Steps
        size="small"
        current={currentIndex}
        items={surfaces.map(s => ({ title: s.title }))}
      />
      <div className="mt-3 grid gap-2 md:grid-cols-3">
        {surfaces.map(s => {
          const Icon = s.icon;
          const isCurrent = s.key === current;
          const inner = (
            <div
              className={`flex h-full items-center justify-between rounded-lg border px-3 py-2 ${
                isCurrent
                  ? 'border-blue-300 bg-blue-50'
                  : 'border-slate-200 bg-white transition-colors hover:border-blue-300 hover:bg-blue-50'
              }`}
            >
              <Space size={8} className="min-w-0">
                <Icon className={`h-4 w-4 shrink-0 ${isCurrent ? 'text-blue-600' : 'text-slate-500'}`} />
                <span className="truncate text-sm font-medium text-slate-700">{s.label}</span>
              </Space>
              {isCurrent ? (
                <Tag color="blue" className="!mr-0">当前</Tag>
              ) : (
                <ArrowRight className="h-4 w-4 shrink-0 text-slate-400" />
              )}
            </div>
          );
          return isCurrent ? (
            <div key={s.key}>{inner}</div>
          ) : (
            <Link key={s.key} href={s.href} className="group no-underline">
              {inner}
            </Link>
          );
        })}
      </div>
      <Text type="secondary" className="mt-2 block text-xs">
        先设计流程并编排审批节点，再为审批节点配置审批链规则，最后绑定到工单类型生效。
      </Text>
    </Card>
  );
};

export default WorkflowConfigGuide;