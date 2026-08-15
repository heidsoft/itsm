'use client';

/**
 * 问题详情组件
 */

import React, { useState, useEffect } from 'react';
import { Card, Tag, Button, Space, Skeleton, message, Typography, Tabs, Modal, Form, Input } from 'antd';
import { ArrowLeft, Search, Pencil, FlaskConical, ShieldAlert } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';

import { ProblemApi } from '@/lib/api/';
import { KEDBApi } from '@/lib/api/kedb-api';
import { ProblemStatus, ProblemStatusLabels } from '@/constants/problem';
import type { Problem } from '@/types/biz/problem';
import ProblemInvestigationTab from './ProblemInvestigationTab';
import BasicInfoCard from './BasicInfoCard';

const { Title } = Typography;
const { TextArea } = Input;

const ProblemDetail: React.FC<{ id?: string }> = ({ id: propId }) => {
  const params = useParams();
  const router = useRouter();
  // 支持通过props传入id，或通过useParams获取
  const id = propId || (params?.id as string);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Problem | null>(null);
  // 状态流转 loading：记录正在提交的目标状态，防止重复点击
  const [updatingStatus, setUpdatingStatus] = useState<ProblemStatus | null>(null);
  // 受控 Tab：用于“启动 RCA”等操作直接跳转到对应面板
  const [activeTab, setActiveTab] = useState('basic');
  // 调查 Tab 内部初始面板（启动 RCA 时跳到“根因分析”）
  const [investigationInnerTab, setInvestigationInnerTab] = useState('overview');
  const [investigationMountKey, setInvestigationMountKey] = useState(0);
  // 转为已知错误 Modal
  const [knownErrorModalOpen, setKnownErrorModalOpen] = useState(false);
  const [creatingKnownError, setCreatingKnownError] = useState(false);
  const [knownErrorForm] = Form.useForm();

  const loadData = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const problem = await ProblemApi.getProblem(Number(id));
      setData(problem as unknown as Problem);
    } catch (error) {
      message.error('加载问题详情失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [id]);

  const handleUpdateStatus = async (status: ProblemStatus) => {
    if (!id) return;
    setUpdatingStatus(status);
    try {
      await ProblemApi.updateProblem(Number(id), { status });
      message.success('状态更新成功');
      loadData();
    } catch (error) {
      message.error('状态更新失败');
    } finally {
      setUpdatingStatus(null);
    }
  };

  // 关闭问题为不可逆操作，必须二次确认
  const handleCloseProblem = () => {
    Modal.confirm({
      title: '确认关闭问题？',
      content: '关闭后问题将进入终态，无法再进行处理操作。请确认根因分析与解决方案已记录完整。',
      okText: '确认关闭',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => handleUpdateStatus(ProblemStatus.CLOSED),
    });
  };

  // 启动 RCA：跳转到问题调查 Tab 的“根因分析”面板
  const handleStartRCA = () => {
    setInvestigationInnerTab('root-cause');
    setInvestigationMountKey((k) => k + 1);
    setActiveTab('investigation');
  };

  // 转为已知错误：沉淀到 KEDB（已知错误库）
  const handleCreateKnownError = async (values: {
    title: string;
    rootCause?: string;
    workaround?: string;
    description?: string;
  }) => {
    if (!data) return;
    setCreatingKnownError(true);
    try {
      await KEDBApi.createKnownError({
        title: values.title,
        description: values.description || data.description,
        rootCause: values.rootCause || data.rootCause,
        workaround: values.workaround,
        problemId: data.id,
      });
      message.success('已转为已知错误');
      setKnownErrorModalOpen(false);
      knownErrorForm.resetFields();
      router.push('/problems/known-errors');
    } catch (error) {
      message.error('转为已知错误失败');
    } finally {
      setCreatingKnownError(false);
    }
  };

  if (loading) {
    return (
      <Card>
        <Skeleton active />
      </Card>
    );
  }

  if (!data) {
    return <Card>未找到该问题</Card>;
  }

  const tabItems = [
    {
      key: 'basic',
      label: '基本信息',
      children: <BasicInfoCard data={data} />,
    },
    {
      key: 'investigation',
      label: (
        <span>
          <Search /> 问题调查
        </span>
      ),
      children: (
        <ProblemInvestigationTab
          key={investigationMountKey}
          problemId={data.id}
          problemTitle={data.title}
          problemDescription={data.description}
          initialInnerTab={investigationInnerTab}
        />
      ),
    },
  ];

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size="middle">
      {/* 操作栏 */}
      <Card styles={{ body: { padding: '16px 24px' } }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button icon={<ArrowLeft />} onClick={() => router.push('/problems')}>
              返回列表
            </Button>
            <Title level={4} style={{ margin: 0 }}>
              {data.title}
            </Title>
            <Tag color={data.status === ProblemStatus.RESOLVED ? 'success' : 'blue'}>
              {ProblemStatusLabels[data.status]}
            </Tag>
          </Space>
          <Space>
            <Button
              icon={<Pencil />}
              onClick={() => router.push(`/problems/${data.id}/edit`)}
            >
              编辑
            </Button>
            {/* 启动 RCA：进入问题调查根因分析 */}
            {(data.status === ProblemStatus.OPEN ||
              data.status === ProblemStatus.INVESTIGATING ||
              data.status === ProblemStatus.IN_PROGRESS) && (
              <Button icon={<FlaskConical />} onClick={handleStartRCA}>
                启动 RCA
              </Button>
            )}
            {/* 转为已知错误：沉淀到 KEDB */}
            {(data.status === ProblemStatus.INVESTIGATING ||
              data.status === ProblemStatus.IN_PROGRESS ||
              data.status === ProblemStatus.RESOLVED) && (
              <Button
                icon={<ShieldAlert />}
                onClick={() => {
                  knownErrorForm.setFieldsValue({
                    title: data.title,
                    rootCause: data.rootCause || '',
                    description: data.description || '',
                  });
                  setKnownErrorModalOpen(true);
                }}
              >
                转为已知错误
              </Button>
            )}
            {data.status === ProblemStatus.OPEN && (
              <Button
                type="primary"
                loading={updatingStatus === ProblemStatus.IN_PROGRESS}
                disabled={updatingStatus !== null}
                onClick={() => handleUpdateStatus(ProblemStatus.IN_PROGRESS)}
              >
                开始处理
              </Button>
            )}
            {data.status === ProblemStatus.IN_PROGRESS && (
              <Button
                type="primary"
                loading={updatingStatus === ProblemStatus.RESOLVED}
                disabled={updatingStatus !== null}
                onClick={() => handleUpdateStatus(ProblemStatus.RESOLVED)}
              >
                标记解决
              </Button>
            )}
            {data.status === ProblemStatus.RESOLVED && (
              <Button
                loading={updatingStatus === ProblemStatus.CLOSED}
                disabled={updatingStatus !== null}
                onClick={handleCloseProblem}
              >
                关闭问题
              </Button>
            )}
          </Space>
        </div>
      </Card>

      {/* Tab 内容 */}
      <Card>
        <Tabs
          items={tabItems}
          activeKey={activeTab}
          onChange={setActiveTab}
        />
      </Card>

      {/* 转为已知错误 Modal */}
      <Modal
        title="转为已知错误"
        open={knownErrorModalOpen}
        onCancel={() => setKnownErrorModalOpen(false)}
        confirmLoading={creatingKnownError}
        onOk={() => knownErrorForm.submit()}
        okText="确认转换"
        cancelText="取消"
      >
        <Form form={knownErrorForm} layout="vertical" onFinish={handleCreateKnownError}>
          <Form.Item
            name="title"
            label="已知错误标题"
            rules={[{ required: true, message: '请输入标题' }]}
          >
            <Input placeholder="请输入已知错误标题" />
          </Form.Item>
          <Form.Item name="rootCause" label="根本原因">
            <TextArea rows={3} placeholder="根本原因（可选）" />
          </Form.Item>
          <Form.Item name="workaround" label="临时规避方案">
            <TextArea rows={3} placeholder="临时规避方案（可选）" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={3} placeholder="问题描述（可选）" />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
};

export default ProblemDetail;
