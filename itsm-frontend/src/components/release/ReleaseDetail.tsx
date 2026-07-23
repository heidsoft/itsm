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
  Modal,
  Input,
} from 'antd';
import { ArrowLeft, Clock, CheckCircle, XCircle, Rocket, RotateCcw } from 'lucide-react';
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
  rolled_back: 'warning',
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
  rolled_back: '已回滚',
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

  const requestReason = (action: 'reject' | 'rollback') => {
    let reason = '';
    const isReject = action === 'reject';
    Modal.confirm({
      title: isReject ? '拒绝发布' : '回滚发布',
      content: (
        <Input.TextArea
          autoFocus
          rows={4}
          placeholder={isReject ? '请输入拒绝原因' : '请输入回滚原因'}
          onChange={(event) => {
            reason = event.target.value.trim();
          }}
        />
      ),
      okText: isReject ? '确认拒绝' : '确认回滚',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        if (!reason) {
          message.warning(isReject ? '请输入拒绝原因' : '请输入回滚原因');
          return Promise.reject();
        }
        if (isReject) {
          await ReleaseApi.rejectRelease(Number(id), reason);
        } else {
          await ReleaseApi.rollbackRelease(Number(id), reason);
        }
        message.success(isReject ? '发布已拒绝' : '发布已回滚');
        await loadDetail();
      },
    });
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
              type="primary"
              icon={<CheckCircle />}
              onClick={async () => {
                try {
                  await ReleaseApi.approveRelease(release.id);
                  message.success('发布已批准');
                  await loadDetail();
                } catch (error) {
                  message.error('批准发布失败');
                }
              }}
            >
              批准
            </Button>
          )}
          {release.status === 'draft' && (
            <Button danger icon={<XCircle />} onClick={() => requestReason('reject')}>
              拒绝
            </Button>
          )}
          {release.status === 'draft' && !release.requiresApproval && (
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
          {['in-progress', 'completed', 'failed'].includes(release.status) && (
            <Button danger icon={<RotateCcw />} onClick={() => requestReason('rollback')}>
              回滚
            </Button>
          )}
        </Space>
      </Card>
    </Space>
  );
};

export default ReleaseDetail;
