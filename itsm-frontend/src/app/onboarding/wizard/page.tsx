'use client';

/**
 * First-Run Wizard · 首次登录引导（5 步 MWP）
 *
 * 范围：仅引导，不做能力建设。
 * - Step 1: 欢迎（品牌）
 * - Step 2: 创建第一个工单（教学）
 * - Step 3: 查看服务目录
 * - Step 4: 配置 CMDB 第一个 CI
 * - Step 5: 体验 AI 分诊（mock + Coming Soon 标签）
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
import {
  RocketOutlined,
  PlusCircleOutlined,
  AppstoreOutlined,
  DatabaseOutlined,
  RobotOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';

const { Title, Paragraph } = Typography;

type StepKey = 'welcome' | 'first-ticket' | 'catalog' | 'cmdb' | 'ai-triage' | 'done';

const STEPS: Array<{ key: StepKey; title: string; icon: React.ReactNode }> = [
  { key: 'welcome', title: '欢迎', icon: <RocketOutlined /> },
  { key: 'first-ticket', title: '创建工单', icon: <PlusCircleOutlined /> },
  { key: 'catalog', title: '服务目录', icon: <AppstoreOutlined /> },
  { key: 'cmdb', title: 'CMDB', icon: <DatabaseOutlined /> },
  { key: 'ai-triage', title: 'AI 分诊', icon: <RobotOutlined /> },
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
    message.info('已跳过引导，可稍后在「个人中心」再次查看');
    setCurrent('done');
    onFinish?.();
  };

  return (
    <Card style={{ maxWidth: 880, margin: '40px auto' }}>
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        <div>
          <Title level={2} style={{ marginBottom: 8 }}>
            欢迎使用 AI-Native ITSM
          </Title>
          <Paragraph type="secondary">
            智能服务管理 · 让 IT 服务更高效
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
              跳过引导
            </Button>
            <Space>
              {currentIndex > 0 && <Button onClick={goPrev}>上一步</Button>}
              <Button type="primary" onClick={goNext} icon={<CheckCircleOutlined />}>
                {currentIndex === STEPS.length - 1 ? '完成' : '下一步'}
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
      icon={<RocketOutlined style={{ color: '#1677ff' }} />}
      title="5 分钟快速了解 AI-Native ITSM"
      subTitle="我们将引导你完成：创建第一个工单、浏览服务目录、配置 CMDB、体验 AI 分诊。"
      extra={
        <Alert
          type="info"
          showIcon
          message="本引导不影响生产数据，所有演示均使用 mock 接口"
        />
      }
    />
  );
}

function FirstTicketStep() {
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Title level={4}>第 1 步：创建你的第一个工单</Title>
      <Paragraph>
        工单（Ticket）是 ITSM 的核心实体，记录一次服务请求或故障处理的全过程。
      </Paragraph>
      <Card type="inner" title="操作演示">
        <Paragraph>1. 点击左侧菜单 <strong>工单管理 → 新建工单</strong></Paragraph>
        <Paragraph>2. 填写标题、描述、优先级</Paragraph>
        <Paragraph>3. 提交后系统会自动分派并触发 SLA 计时</Paragraph>
      </Card>
      <Tag color="blue">预计耗时：30 秒</Tag>
    </Space>
  );
}

function CatalogStep() {
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Title level={4}>第 2 步：浏览服务目录</Title>
      <Paragraph>
        服务目录（Service Catalog）将 IT 服务标准化，员工可自助申请，审批自动流转。
      </Paragraph>
      <Card type="inner" title="内置服务">
        <Paragraph>· 申请新员工账号</Paragraph>
        <Paragraph>· 申请云资源（开发/测试/生产）</Paragraph>
        <Paragraph>· 申请软件许可证</Paragraph>
        <Paragraph>· 申请 VPN / VPN 组</Paragraph>
      </Card>
    </Space>
  );
}

function CMDBStep() {
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Title level={4}>第 3 步：配置 CMDB 第一个 CI</Title>
      <Paragraph>
        CMDB（Configuration Management Database）记录所有受管的配置项（CI）及其关系。
      </Paragraph>
      <Card type="inner" title="推荐起点">
        <Paragraph>· 录入一台核心业务服务器</Paragraph>
        <Paragraph>· 添加与数据库的依赖关系</Paragraph>
        <Paragraph>· 后续可在变更管理中自动做影响分析</Paragraph>
      </Card>
    </Space>
  );
}

function AITriageStep() {
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Title level={4}>
        第 4 步：体验 AI 分诊
        <Tag color="orange" style={{ marginLeft: 12 }}>Coming Soon</Tag>
      </Title>
      <Paragraph>
        AI 分诊会自动分析工单内容，推荐分类、优先级和处理人。
      </Paragraph>
      <Alert
        type="warning"
        showIcon
        message="实验性功能"
        description="当前版本仅展示 mock 响应，正式 RAG 能力将在 v2.0 推出。"
      />
      <Card type="inner" title="Mock 演示">
        <Paragraph>
          <Tag color="blue">mock</Tag> 工单：「VPN 连不上」→ 建议分类：网络 / 优先级：高 / 处理人：网络组
        </Paragraph>
        <Paragraph type="secondary" style={{ fontSize: 12 }}>
          数据来源：/api/v1/ai/mock/triage
        </Paragraph>
      </Card>
    </Space>
  );
}

function DoneStep({ onEnterApp }: { onEnterApp: () => void }) {
  return (
    <Result
      status="success"
      title="引导完成！"
      subTitle="你已经了解了 AI-Native ITSM 的核心能力。"
      extra={[
        <Button type="primary" key="enter" onClick={onEnterApp}>
          进入工作台
        </Button>,
      ]}
    />
  );
}
