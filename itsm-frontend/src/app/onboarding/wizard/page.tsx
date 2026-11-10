'use client';

/**
 * First-Run Wizard · 首次登录引导（5 步 MWP）
 *
 * 范围：仅引导，不做能力建设。
 * - Step 1: 欢迎（品牌）
 * - Step 2: 创建第一个工单（教学）
 * - Step 3: 查看服务目录
 * - Step 4: 配置 CMDB 第一个 CI
 * - Step 5: 体验 AI 分诊（调用真实后端能力）
 */

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Button,
  Card,
  Space,
  Steps,
  Typography,
  Result,
  App,
  Tag,
  Alert,
} from 'antd';
import { LayoutGrid, CheckCircle, PlusCircle, Database, Rocket, Bot } from 'lucide-react';
import { AIApi, TriageResult } from '@/lib/api/ai-api';

const { Title, Paragraph } = Typography;

// 首次登录引导里的 AI 分诊演示直接打印枚举值会显示 network / high，
// 对新用户来说既不可读也看不出这是中文产品，这里补一层展示映射。
const TRIAGE_LABELS: Record<string, string> = {
  network: 'Network failure',
  hardware: 'Hardware failure',
  software: 'Software failure',
  database: 'Database problem',
  security: 'Security issue',
  performance: 'Performance issue',
  config: 'Configuration problem',
  other: 'Other',
  critical: 'Urgent',
  urgent: 'Urgent',
  high: 'High',
  medium: 'Medium',
  low: 'Low',
};

const triageLabel = (value: string): string => TRIAGE_LABELS[value] ?? value;

type StepKey = 'welcome' | 'first-ticket' | 'catalog' | 'cmdb' | 'ai-triage' | 'done';

const STEPS: Array<{ key: StepKey; title: string; icon: React.ReactNode }> = [
  { key: 'welcome', title: 'Welcome', icon: <Rocket /> },
  { key: 'first-ticket', title: 'Create a work order', icon: <PlusCircle /> },
  { key: 'catalog', title: 'Service Catalog', icon: <LayoutGrid /> },
  { key: 'cmdb', title: 'CMDB', icon: <Database /> },
  { key: 'ai-triage', title: 'AI triage', icon: <Bot /> },
];

interface WizardProps {
  /** 已完成步骤，传入时跳过 */
  initialStep?: StepKey;
  onFinish?: () => void;
}

export default function FirstRunWizard({ initialStep, onFinish }: WizardProps) {
  const router = useRouter();
  const { message } = App.useApp();
  const [current, setCurrent] = useState<StepKey>(initialStep ?? 'welcome');
  const [currentIndex] = useState(() =>
    initialStep ? STEPS.findIndex((s) => s.key === initialStep) : 0
  );

  const goNext = () => {
    const idx = STEPS.findIndex((s) => s.key === current);
    if (idx < STEPS.length - 1) {
      setCurrent(STEPS[idx + 1].key);
    } else {
      setCurrent('done');
      onFinish?.();
    }
  };

  const goPrev = () => {
    const idx = STEPS.findIndex((s) => s.key === current);
    if (idx > 0) setCurrent(STEPS[idx - 1].key);
  };

  const skip = () => {
    message.info('Guidance has been skipped. You can view it again in "Personal Center" later.');
    setCurrent('done');
    onFinish?.();
  };

  return (
    <Card style={{ maxWidth: 880, margin: '40px auto' }}>
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        <div>
          <Title level={2} style={{ marginBottom: 8 }}>

            Welcome to AI-Native ITSM
          </Title>
          <Paragraph type="secondary">

            Intelligent service management · Make IT services more efficient
          </Paragraph>
        </div>

        {current !== 'done' && (
          <Steps
            current={currentIndex}
            items={STEPS.map((s) => ({ title: s.title, icon: s.icon }))}
          />
        )}

        <div style={{ minHeight: 280, padding: '16px 0' }}>
          {current === 'welcome' && <WelcomeStep />}
          {current === 'first-ticket' && <FirstTicketStep />}
          {current === 'catalog' && <CatalogStep />}
          {current === 'cmdb' && <CMDBStep />}
          {current === 'ai-triage' && <AITriageStep />}
          {current === 'done' && <DoneStep onEnterApp={() => router.push('/dashboard')} />}
        </div>

        {current !== 'done' && (
          <Space style={{ justifyContent: 'space-between', width: '100%' }}>
            <Button onClick={skip} type="text">

              Skipping boot
            </Button>
            <Space>
              {currentIndex > 0 && <Button onClick={goPrev}>Previous</Button>}
              <Button type="primary" onClick={goNext} icon={<CheckCircle />}>
                {currentIndex === STEPS.length - 1 ? 'Complete' : 'Next'}
              </Button>
            </Space>
          </Space>
        )}
      </Space>
    </Card>
  );
}

function WelcomeStep() {
  return (
    <Result
      icon={<Rocket style={{ color: '#3b82f6' }} />}
      title="Quickly understand AI-Native ITSM in 5 minutes"
      subTitle="We will guide you through: creating your first work order, browsing the service catalog, configuring CMDB, and experiencing AI triage."
      extra={
        <Alert
          type="info"
          showIcon
          title="This boot is a functional demonstration and no actual data will be written."
        />
      }
    />
  );
}

function FirstTicketStep() {
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Title level={4}>Step 1: Create your first ticket</Title>
      <Paragraph>

        Ticket is the core entity of ITSM, recording the entire process of a service request or fault handling.
      </Paragraph>
      <Card type="inner" title="Operation demonstration">
        <Paragraph>1. Click on the left menu <strong>Work order management → Create new work order</strong></Paragraph>
        <Paragraph>2. Fill in title, description, priority</Paragraph>
        <Paragraph>3. After submission, the system will automatically dispatch and trigger SLA timing</Paragraph>
      </Card>
      <Tag color="blue">Estimated time: 30 seconds</Tag>
    </Space>
  );
}

function CatalogStep() {
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Title level={4}>Step 2: Browse the Services Catalog</Title>
      <Paragraph>

        Service Catalog standardizes IT services, employees can apply for them by themselves, and approvals are transferred automatically.
      </Paragraph>
      <Card type="inner" title="Built-in service">
        <Paragraph>· Apply for a new employee account</Paragraph>
        <Paragraph>· Apply for cloud resources (development/test/production)</Paragraph>
        <Paragraph>· Apply for software license</Paragraph>
        <Paragraph>· Apply for VPN / VPN group</Paragraph>
      </Card>
    </Space>
  );
}

function CMDBStep() {
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Title level={4}>Step 3: Configure CMDB First CI</Title>
      <Paragraph>

        CMDB (Configuration Management Database) records all managed configuration items (CI) and their relationships.
      </Paragraph>
      <Card type="inner" title="Recommended starting point">
        <Paragraph>· Enter a core business server</Paragraph>
        <Paragraph>· Add dependency on database</Paragraph>
        <Paragraph>· Impact analysis can be automatically performed in change management later.</Paragraph>
      </Card>
    </Space>
  );
}

function AITriageStep() {
  const { message } = App.useApp();
  const [description, setDescription] = useState('The VPN cannot be connected, and multiple colleagues cannot access the intranet system.');
  const [result, setResult] = useState<TriageResult | null>(null);
  const [loading, setLoading] = useState(false);

  const runTriage = async () => {
    const trimmed = description.trim();
    if (!trimmed) {
      message.warning('Please describe a problem first');
      return;
    }
    setLoading(true);
    try {
      setResult(await AIApi.triage('VPN connection issues', trimmed));
    } catch {
      setResult(null);
      message.error('AI triage is temporarily unavailable, please try again later.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Title level={4}>

        Step 4: Experience AI Triage
        <Tag color="blue" style={{ marginLeft: 12 }}>Early Access</Tag>
      </Title>
      <Paragraph>

        AI triage will automatically analyze the content of the work order and recommend classification, priority and handler.
      </Paragraph>
      <Alert type="info" showIcon title="AI-assisted triage" description="Recommend classification and priority based on work order content, the results are for reference only." />
      <Card type="inner" title="Effect demonstration">
        <Space orientation="vertical" style={{ width: '100%' }}>
          <textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={3} style={{ width: '100%', padding: 8 }} aria-label="Problem description" />
          <Button type="primary" icon={<Bot size={16} />} loading={loading} onClick={runTriage}>Running AI Triage</Button>
          {result && <Alert type="success" title={`Category:${triageLabel(result.category)} · Priority:${triageLabel(result.priority)}`} description={`Confidence ${(result.confidence * 100).toFixed(0)}%${result.explanation ? ` · ${result.explanation}` : ''}`} />}
        </Space>
      </Card>
    </Space>
  );
}

function DoneStep({ onEnterApp }: { onEnterApp: () => void }) {
  return (
    <Result
      status="success"
      title="Booting completed!"
      subTitle="You have understood the core capabilities of AI-Native ITSM."
      extra={[
        <Button type="primary" key="enter" onClick={onEnterApp}>

          Enter workbench
        </Button>,
      ]}
    />
  );
}
