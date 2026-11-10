'use client';

import Link from 'next/link';
import { Alert, Button, Card, Space, Steps, Tag, Typography } from 'antd';
import { ArrowRight, CheckCircle2, ClipboardList, Network, ShieldCheck, Users, Workflow } from 'lucide-react';

const { Text, Title } = Typography;

const setupSteps = [
  {
    title: 'Organization and Permissions',
    description: 'Establish users, user groups, roles and approver ranges.',
    href: '/admin/users',
    label: 'Configure user',
    icon: Users,
  },
  {
    title: 'Service Policy',
    description: 'Prepare classification, SLA, handler assignment rules and approval rules.',
    href: '/admin/sla-definitions',
    label: 'Configure SLA',
    icon: ShieldCheck,
  },
  {
    title: 'Process Design',
    description: 'Design and release a BPMN process; change scenarios complement the review team.',
    href: '/workflow/designer',
    label: 'Design Process',
    icon: Workflow,
  },
  {
    title: 'Work order type binding',
    description: 'Bind processes, SLAs, approvals and allocation policies into available services.',
    href: '/tickets/types',
    label: 'Binding strategy',
    icon: ClipboardList,
  },
  {
    title: 'Joint debugging and acceptance',
    description: 'Create a test ticket to check the instance, approvals, SLA and auditing.',
    href: '/workflow/instances',
    label: 'View Instances',
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
          <span>Administrator configures route</span>
          <Tag color="blue">Recommended order</Tag>
        </Space>
      }
      extra={<Text type="secondary">Bind first, then create a test work order for verification</Text>}
    >
      <div className="border-b border-slate-100 bg-slate-50 px-5 py-4">
        <Title level={5} className="!mb-1">Assemble independent configuration items into executable services</Title>
        <Text type="secondary">

          Process, SLA, allocation and approval rules will not automatically take effect on all work orders; binding needs to be completed in "Work Order Type".
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
          title="Acceptance criteria: The test work order should generate a handler, SLA deadline and process instance; if approval is configured, tasks to be approved should also appear."
        />
      </div>
    </Card>
  );
}
