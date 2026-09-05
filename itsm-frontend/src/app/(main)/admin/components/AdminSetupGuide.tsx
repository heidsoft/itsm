'use client';

import Link from 'next/link';
import { Alert, Button, Card, Space, Steps, Tag, Typography } from 'antd';
import { ArrowRight, CheckCircle2, ClipboardList, Network, ShieldCheck, Users, Workflow } from 'lucide-react';

const { Text, Title } = Typography;

const setupSteps = [
  {
    title: '组织与权限',
    description: '建立用户、用户组、角色与审批人范围。',
    href: '/admin/users',
    label: '配置用户',
    icon: Users,
  },
  {
    title: '服务策略',
    description: '准备分类、SLA、处理人分配规则和审批规则。',
    href: '/admin/sla-definitions',
    label: '配置 SLA',
    icon: ShieldCheck,
  },
  {
    title: '流程设计',
    description: '设计并发布 BPMN 流程；变更场景可补充 CAB。',
    href: '/workflow/designer',
    label: '设计流程',
    icon: Workflow,
  },
  {
    title: '工单类型绑定',
    description: '把流程、SLA、审批与分配策略绑定为可用服务。',
    href: '/tickets/types',
    label: '绑定策略',
    icon: ClipboardList,
  },
  {
    title: '联调与验收',
    description: '创建一张测试工单，检查实例、审批、SLA 和审计。',
    href: '/workflow/instances',
    label: '查看实例',
    icon: CheckCircle2,
  },
];

/** 管理员首次配置的最短闭环，避免在独立配置页之间迷失。 */
export function AdminSetupGuide() {
  return (
    <Card
      className="overflow-hidden"
      styles={{ body: { padding: 0 } }}
      title={
        <Space>
          <Network className="h-5 w-5 text-blue-600" />
          <span>管理员配置路线</span>
          <Tag color="blue">推荐顺序</Tag>
        </Space>
      }
      extra={<Text type="secondary">先绑定，再创建测试工单验证</Text>}
    >
      <div className="border-b border-slate-100 bg-slate-50 px-5 py-4">
        <Title level={5} className="!mb-1">把独立配置项组装成可执行的服务</Title>
        <Text type="secondary">
          流程、SLA、分配和审批规则不会自动对所有工单生效；需要在“工单类型”中完成绑定。
        </Text>
      </div>

      <div className="p-5">
        <Steps
          responsive
          size="small"
          current={-1}
          items={setupSteps.map((step, index) => {
            const Icon = step.icon;
            return {
              title: `${index + 1}. ${step.title}`,
              description: step.description,
              icon: <Icon className="h-4 w-4" />,
            };
          })}
        />

        <div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          {setupSteps.map(step => {
            const Icon = step.icon;
            return (
              <Link key={step.href} href={step.href} className="group no-underline">
                <div className="flex h-full items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-3 transition-colors group-hover:border-blue-300 group-hover:bg-blue-50">
                  <Space size={8} className="min-w-0">
                    <Icon className="h-4 w-4 shrink-0 text-slate-500 group-hover:text-blue-600" />
                    <span className="truncate text-sm font-medium text-slate-700">{step.label}</span>
                  </Space>
                  <ArrowRight className="h-4 w-4 shrink-0 text-slate-400 group-hover:text-blue-600" />
                </div>
              </Link>
            );
          })}
        </div>

        <Alert
          className="mt-5"
          type="info"
          showIcon
          message="验收标准：测试工单应产生处理人、SLA 截止时间和流程实例；如配置了审批，还应出现待审批任务。"
        />
      </div>
    </Card>
  );
}
