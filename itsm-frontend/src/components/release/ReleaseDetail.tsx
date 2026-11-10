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
  draft: 'Draft',
  scheduled: 'Planned',
  'in-progress': 'In Progress',
  completed: 'Completed',
  cancelled: 'Cancelled',
  failed: 'Failed',
  rolled_back: 'Rolled Back',
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
      message.error('Failed to load publication details');
    } finally {
      setLoading(false);
    }
  };

  const requestReason = (action: 'reject' | 'rollback') => {
    let reason = '';
    const isReject = action === 'reject';
    Modal.confirm({
      title: isReject ? 'Refuse to publish' : 'Rollback release',
      content: (
        <Input.TextArea
          autoFocus
          rows={4}
          placeholder={isReject ? 'Please enter rejection reason' : 'Please enter rollback reason'}
          onChange={(event) => {
            reason = event.target.value.trim();
          }}
        />
      ),
      okText: isReject ? 'Confirm Rejection' : 'Confirm rollback',
      okButtonProps: { danger: true },
      cancelText: 'Cancel',
      onOk: async () => {
        if (!reason) {
          message.warning(isReject ? 'Please enter rejection reason' : 'Please enter rollback reason');
          return Promise.reject();
        }
        if (isReject) {
          await ReleaseApi.rejectRelease(Number(id), reason);
        } else {
          await ReleaseApi.rollbackRelease(Number(id), reason);
        }
        message.success(isReject ? 'Post rejected' : 'Release rolled back');
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
          subTitle="Sorry, the publication you visited does not exist"
          extra={
            <Button type="primary" onClick={() => router.push('/releases')}>

              Back to List
            </Button>
          }
        />
      </Card>
    );
  }

  const currentStep = ['draft', 'scheduled', 'in-progress', 'completed'].indexOf(release.status);

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size="large">
      <Card>
        <div style={{ marginBottom: 24 }}>
          <Button
            icon={<ArrowLeft />}
            onClick={() => router.push('/releases')}
            style={{ marginBottom: 16 }}
          >

            Back to List
          </Button>
          <div
            style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
          >
            <div>
              <Title level={3} style={{ marginBottom: 8 }}>
                {release.title}
              </Title>
              <Text type="secondary">Release number: {release.releaseNumber}</Text>
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
            { title: 'Draft', icon: <Clock /> },
            { title: 'Planned', icon: <Clock /> },
            { title: 'In Progress', icon: <Rocket /> },
            { title: 'Completed', icon: <CheckCircle /> },
          ]}
        />

        <Descriptions bordered column={2}>
          <Descriptions.Item label="Release type">
            <Tag color={typeColors[release.type]}>{release.type?.toUpperCase()}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Target environment">
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
          <Descriptions.Item label="Severity">
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
          <Descriptions.Item label="Urgent release">
            {release.isEmergency ? <Tag color="red">Yes</Tag> : <Tag>No</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="Approval required">
            {release.requiresApproval ? <Tag color="blue">Yes</Tag> : <Tag>No</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="Assignee">{release.ownerName || '-'}</Descriptions.Item>
          <Descriptions.Item label="Planned release date">
            {release.plannedReleaseDate
              ? dayjs(release.plannedReleaseDate).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Actual release date">
            {release.actualReleaseDate
              ? dayjs(release.actualReleaseDate).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Planned start time">
            {release.plannedStartDate
              ? dayjs(release.plannedStartDate).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Planned end time">
            {release.plannedEndDate
              ? dayjs(release.plannedEndDate).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Creator">{release.createdByName}</Descriptions.Item>
          <Descriptions.Item label="Created">
            {dayjs(release.createdAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {release.description && (
        <Card title="Description">
          <Text>{release.description}</Text>
        </Card>
      )}

      {release.releaseNotes && (
        <Card title="Release Notes">
          <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
            {release.releaseNotes}
          </pre>
        </Card>
      )}

      {release.affectedSystems && release.affectedSystems.length > 0 && (
        <Card title="Affected systems">
          <Space wrap>
            {release.affectedSystems.map((system, index) => (
              <Tag key={index}>{system}</Tag>
            ))}
          </Space>
        </Card>
      )}

      {release.affectedComponents && release.affectedComponents.length > 0 && (
        <Card title="Affected components">
          <Space wrap>
            {release.affectedComponents.map((component, index) => (
              <Tag key={index}>{component}</Tag>
            ))}
          </Space>
        </Card>
      )}

      {release.deploymentSteps && release.deploymentSteps.length > 0 && (
        <Card title="Deployment steps">
          <Timeline>
            {release.deploymentSteps.map((step, index) => (
              <Timeline.Item key={index}>{step}</Timeline.Item>
            ))}
          </Timeline>
        </Card>
      )}

      {release.rollbackProcedure && (
        <Card title="Rollback procedure">
          <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
            {release.rollbackProcedure}
          </pre>
        </Card>
      )}

      {release.validationCriteria && (
        <Card title="Verification Standard">
          <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
            {release.validationCriteria}
          </pre>
        </Card>
      )}

      <Card>
        <Space>
          <Button type="primary" onClick={() => router.push(`/releases/${release.id}/edit`)}>

            Edit
          </Button>
          {release.status === 'draft' && (
            <Button
              type="primary"
              icon={<CheckCircle />}
              onClick={async () => {
                try {
                  await ReleaseApi.approveRelease(release.id);
                  message.success('Release approved');
                  await loadDetail();
                } catch (error) {
                  message.error('Failed to approve release');
                }
              }}
            >

              Approve
            </Button>
          )}
          {release.status === 'draft' && (
            <Button danger icon={<XCircle />} onClick={() => requestReason('reject')}>

              Rejected
            </Button>
          )}
          {release.status === 'draft' && !release.requiresApproval && (
            <Button
              onClick={async () => {
                try {
                  await ReleaseApi.updateReleaseStatus(release.id, 'scheduled');
                  loadDetail();
                } catch (error) {
                  message.error('Failed to update status');
                }
              }}
            >

              Submit plan
            </Button>
          )}
          {release.status === 'scheduled' && (
            <Button
              onClick={async () => {
                try {
                  await ReleaseApi.updateReleaseStatus(release.id, 'in-progress');
                  loadDetail();
                } catch (error) {
                  message.error('Failed to update status');
                }
              }}
            >

              Start publishing
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
                  message.error('Failed to update status');
                }
              }}
            >

              Complete publishing
            </Button>
          )}
          {['in-progress', 'completed', 'failed'].includes(release.status) && (
            <Button danger icon={<RotateCcw />} onClick={() => requestReason('rollback')}>

              Rollback
            </Button>
          )}
        </Space>
      </Card>
    </Space>
  );
};

export default ReleaseDetail;
