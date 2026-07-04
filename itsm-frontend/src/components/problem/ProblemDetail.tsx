'use client';

/**
 * 问题详情组件
 */

import React, { useState, useEffect } from 'react';
import { Card, Tag, Button, Space, Skeleton, message, Typography, Tabs } from 'antd';
import { ArrowLeft, Search, Pencil } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';

import { ProblemApi } from '@/lib/api/';
import { ProblemStatus, ProblemStatusLabels } from '@/constants/problem';
import type { Problem } from '@/types/biz/problem';
import ProblemInvestigationTab from './ProblemInvestigationTab';
import BasicInfoCard from './BasicInfoCard';

const { Title } = Typography;

const ProblemDetail: React.FC<{ id?: string }> = ({ id: propId }) => {
  const params = useParams();
  const router = useRouter();
  // 支持通过props传入id，或通过useParams获取
  const id = propId || (params?.id as string);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Problem | null>(null);

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
    try {
      await ProblemApi.updateProblem(Number(id), { status });
      message.success('状态更新成功');
      loadData();
    } catch (error) {
      message.error('状态更新失败');
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
          problemId={data.id}
          problemTitle={data.title}
          problemDescription={data.description}
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
            {data.status === ProblemStatus.OPEN && (
              <Button type="primary" onClick={() => handleUpdateStatus(ProblemStatus.IN_PROGRESS)}>
                开始处理
              </Button>
            )}
            {data.status === ProblemStatus.IN_PROGRESS && (
              <Button type="primary" onClick={() => handleUpdateStatus(ProblemStatus.RESOLVED)}>
                标记解决
              </Button>
            )}
            {data.status === ProblemStatus.RESOLVED && (
              <Button onClick={() => handleUpdateStatus(ProblemStatus.CLOSED)}>关闭问题</Button>
            )}
          </Space>
        </div>
      </Card>

      {/* Tab 内容 */}
      <Card>
        <Tabs items={tabItems} defaultActiveKey="basic" />
      </Card>
    </Space>
  );
};

export default ProblemDetail;
