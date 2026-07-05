'use client';

/**
 * 发布详情组件
 */

import React, { useState, useEffect } from 'react';
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Skeleton,
  Result,
  Space,
  Timeline,
  Steps,
  Divider,
  Typography,
  message,
} from 'antd';
import { ArrowLeft, Clock, CheckCircle, XCircle, Rocket } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import type { Release } from '@/lib/api/release-api';
import { ReleaseApi } from '@/lib/api/release-api';

const { Title, Text } = Typography;

// 状态颜色映射
const statusColors: Record<string, string> = {
  draft: 'default',
  scheduled: 'blue',
  'in-progress': 'processing',
  completed: 'success',
  cancelled: 'default',
  failed: 'error',
  rolledBack: 'warning',
};

// 类型颜色映射
const typeColors: Record<string, string> = {
  major: 'red',
  minor: 'blue',
  patch: 'green',
  hotfix: 'orange',
};

const statusLabels: Record<string, string> = {
  draft: '草稿',
  scheduled: '已计划',
  'in-progress': '进行中',
  completed: '已完成',
  cancelled: '已取消',
  failed: '失败',
  rolledBack: '已回滚',
};

const ReleaseDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [release, setRelease] = useState<Release | null>(null);

  useEffect(() => {
    if (id) {
      loadDetail();
    }
  }, [id]);

  const loadDetail = async () => {
    setLoading(true);
    try {
      const data = await ReleaseApi.getRelease(Number(id));
      setRelease(data);
    } catch (error) {
      message.error('加载发布详情失败');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <Card>
        <Skeleton active />
      </Card>
    );
  }

  if (!release) {
    return (
      <Card>
        <Result
          status="404"
          title="404"
          subTitle="抱歉，您访问的发布不存在"
          extra={
            <Button type="primary" onClick={() => router.push('/releases')}>
              返回列表
            </Button>
          }
        />
      </Card>
    );
  }

  const currentStep = ['draft', 'scheduled', 'in-progress', 'completed'].indexOf(release.status);

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="large">
      <Card>
        <div style={{ marginBottom: 24 }}>
          <Button
            icon={<ArrowLeft />}
            onClick={() => router.push('/releases')}
            style={{ marginBottom: 16 }}
          >
            返回列表
          </Button>
          <div
            style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
          >
            <div>
              <Title level={3} style={{ marginBottom: 8 }}>
                {release.title}
              </Title>
              <Text type="secondary">发布编号: {release.releaseNumber}</Text>
            </div>
            <Tag color={statusColors[release.status]} style={{ padding: '4px 12px', fontSize: 14 }}>
              {statusLabels[release.status]}
            </Tag>
          </div>
        </div>

        <Steps
          current={currentStep}
          style={{ marginBottom: 32 }}
          items={[
            { title: '草稿', icon: <Clock /> },
            { title: '已计划', icon: <Clock /> },
            { title: '进行中', icon: <Rocket /> },
            { title: '已完成', icon: <CheckCircle /> },
          ]}
        />

        <Descriptions bordered column={2}>
          <Descriptions.Item label="发布类型">
            <Tag color={typeColors[release.type]}>{release.type?.toUpperCase()}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="目标环境">
            <Tag
              color={
                release.environment === 'production'
                  ? 'red'
                  : release.environment === 'staging'
                    ? 'orange'
                    : 'default'
              }
            >
              {release.environment?.toUpperCase()}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="严重程度">
            <Tag
              color={
                release.severity === 'critical'
                  ? 'red'
                  : release.severity === 'high'
                    ? 'orange'
                    : 'default'
              }
            >
              {release.severity}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="紧急发布">
            {release.isEmergency ? <Tag color="red">是</Tag> : <Tag>否</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="需要审批">
            {release.requiresApproval ? <Tag color="blue">是</Tag> : <Tag>否</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="负责人">{release.ownerName || '-'}</Descriptions.Item>
          <Descriptions.Item label="计划发布日期">
            {release.plannedReleaseDate
              ? dayjs(release.plannedReleaseDate).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="实际发布日期">
            {release.actualReleaseDate
              ? dayjs(release.actualReleaseDate).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="计划开始时间">
            {release.plannedStartDate
              ? dayjs(release.plannedStartDate).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="计划结束时间">
            {release.plannedEndDate
              ? dayjs(release.plannedEndDate).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="创建人">{release.createdByName}</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {dayjs(release.createdAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {release.description && (
        <Card title="描述">
          <Text>{release.description}</Text>
        </Card>
      )}

      {release.releaseNotes && (
        <Card title="发布说明">
          <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
            {release.releaseNotes}
          </pre>
        </Card>
      )}

      {release.affectedSystems && release.affectedSystems.length > 0 && (
        <Card title="受影响的系统">
          <Space wrap>
            {release.affectedSystems.map((system, index) => (
              <Tag key={index}>{system}</Tag>
            ))}
          </Space>
        </Card>
      )}

      {release.affectedComponents && release.affectedComponents.length > 0 && (
        <Card title="受影响的组件">
          <Space wrap>
            {release.affectedComponents.map((component, index) => (
              <Tag key={index}>{component}</Tag>
            ))}
          </Space>
        </Card>
      )}

      {release.deploymentSteps && release.deploymentSteps.length > 0 && (
        <Card title="部署步骤">
          <Timeline>
            {release.deploymentSteps.map((step, index) => (
              <Timeline.Item key={index}>{step}</Timeline.Item>
            ))}
          </Timeline>
        </Card>
      )}

      {release.rollbackProcedure && (
        <Card title="回滚程序">
          <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
            {release.rollbackProcedure}
          </pre>
        </Card>
      )}

      {release.validationCriteria && (
        <Card title="验证标准">
          <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
            {release.validationCriteria}
          </pre>
        </Card>
      )}

      <Card>
        <Space>
          <Button type="primary" onClick={() => router.push(`/releases/${release.id}`)}>
            编辑
          </Button>
          {release.status === 'draft' && (
            <Button
              onClick={async () => {
                try {
                  await ReleaseApi.updateReleaseStatus(release.id, 'scheduled');
                  loadDetail();
                } catch (error) {
                  message.error('更新状态失败');
                }
              }}
            >
              提交计划
            </Button>
          )}
          {release.status === 'scheduled' && (
            <Button
              onClick={async () => {
                try {
                  await ReleaseApi.updateReleaseStatus(release.id, 'in-progress');
                  loadDetail();
                } catch (error) {
                  message.error('更新状态失败');
                }
              }}
            >
              开始发布
            </Button>
          )}
          {release.status === 'in-progress' && (
            <Button
              type="primary"
              onClick={async () => {
                try {
                  await ReleaseApi.updateReleaseStatus(release.id, 'completed');
                  loadDetail();
                } catch (error) {
                  message.error('更新状态失败');
                }
              }}
            >
              完成发布
            </Button>
          )}
        </Space>
      </Card>
    </Space>
  );
};

export default ReleaseDetail;
