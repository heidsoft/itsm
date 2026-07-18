'use client';

import React, { useEffect, useState } from 'react';
import { Timeline, Typography, Tag, Space, Button, Modal, Input, App, Empty, Select } from 'antd';
import { CheckCircle, XCircle, Timer, ArrowRight } from 'lucide-react';
import type { ApprovalStep, ApprovalStepStatus, ApprovalActionInput } from './types';
import { UserApi } from '@/lib/api/user-api';

const { Text } = Typography;
const { TextArea } = Input;

const approvalStatusColors: Record<ApprovalStepStatus, string> = {
  pending: 'blue',
  approved: 'green',
  rejected: 'red',
  delegated: 'purple',
  timeout: 'orange',
  skipped: 'gray',
};

const approvalStatusLabels: Record<ApprovalStepStatus, string> = {
  pending: '待审批',
  approved: '已通过',
  rejected: '已拒绝',
  delegated: '已委派',
  timeout: '已超时',
  skipped: '已跳过',
};

export interface ApprovalTimelineProps {
  approvals: ApprovalStep[];
  currentLevel?: number;
  submittedAt?: string;
  submitterName?: string;
  workflowName?: string;
  canApprove?: boolean;
  showApprovalActions?: boolean;
  onApprove?: (data: ApprovalActionInput) => Promise<void>;
  onReject?: (data: ApprovalActionInput) => Promise<void>;
  onDelegate?: (data: ApprovalActionInput) => Promise<void>;
  formatDateTime?: (s: string) => string;
}

const defaultFormat = (s?: string) => (s ? new Date(s).toLocaleString('zh-CN') : '');

type Action = 'approve' | 'reject' | 'delegate';

interface CandidateUser {
  id: number;
  name?: string;
  username?: string;
}

export const ApprovalTimeline: React.FC<ApprovalTimelineProps> = ({
  approvals,
  currentLevel,
  submittedAt,
  submitterName,
  workflowName,
  canApprove = false,
  showApprovalActions = true,
  onApprove,
  onReject,
  onDelegate,
  formatDateTime = defaultFormat,
}) => {
  const { message } = App.useApp();
  const [currentAction, setCurrentAction] = useState<Action | null>(null);
  const [comment, setComment] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [delegateToUserId, setDelegateToUserId] = useState<number | undefined>();
  const [userOptions, setUserOptions] = useState<CandidateUser[]>([]);
  const [userSearchLoading, setUserSearchLoading] = useState(false);

  const iconFor = (status: ApprovalStepStatus) => {
    if (status === 'approved') return <CheckCircle size={16} />;
    if (status === 'rejected') return <XCircle size={16} />;
    return <Timer size={16} />;
  };

  const openModal = (action: Action) => {
    setCurrentAction(action);
    setComment('');
    setDelegateToUserId(undefined);
    setModalOpen(true);
  };

  const searchUsers = async (kw: string) => {
    setUserSearchLoading(true);
    try {
      const res = await UserApi.getUsers({ page: 1, pageSize: 20, search: kw });
      const list = (res.users || []).map((u) => ({
        id: u.id,
        name: u.name,
        username: u.username,
      }));
      setUserOptions(list);
    } catch (e) {
      console.warn('search users failed', e);
    } finally {
      setUserSearchLoading(false);
    }
  };

  useEffect(() => {
    if (modalOpen && currentAction === 'delegate' && userOptions.length === 0) {
      void searchUsers('');
    }
  }, [modalOpen, currentAction, userOptions.length]);

  const handleSubmit = async () => {
    if (!comment.trim()) {
      message.warning('请填写审批意见');
      return;
    }
    if (currentAction === 'delegate' && !delegateToUserId) {
      message.warning('请选择委派人');
      return;
    }
    if (!currentAction) return;
    setSubmitting(true);
    try {
      const payload: ApprovalActionInput = { comment, delegateToUserId };
      if (currentAction === 'approve') await onApprove?.(payload);
      else if (currentAction === 'reject') await onReject?.(payload);
      else if (currentAction === 'delegate') await onDelegate?.(payload);
      message.success('操作成功');
      setModalOpen(false);
    } catch (e) {
      message.error(e instanceof Error ? e.message : '操作失败');
    } finally {
      setSubmitting(false);
    }
  };

  const modalTitle =
    currentAction === 'approve'
      ? '审批通过'
      : currentAction === 'reject'
      ? '审批拒绝'
      : '委派审批';

  return (
    <div className="p-6">
      {workflowName && (
        <div className="mb-4">
          <Text type="secondary">审批流程：</Text>
          <Text strong>{workflowName}</Text>
          {currentLevel !== undefined && (
            <Tag className="ml-2" color="blue">
              当前第 {currentLevel} 级
            </Tag>
          )}
        </div>
      )}

      {approvals.length === 0 && !submittedAt ? (
        <Empty description="暂无审批记录" />
      ) : (
        <Timeline>
          {submittedAt && (
            <Timeline.Item color="green">
              <div>
                <Text strong>{submitterName || '申请人'}</Text>
                <Tag color="green" className="ml-2">
                  提交申请
                </Tag>
              </div>
              <Text type="secondary" className="text-xs">
                {formatDateTime(submittedAt)}
              </Text>
            </Timeline.Item>
          )}
          {approvals.map((app) => (
            <Timeline.Item
              key={app.id}
              color={approvalStatusColors[app.status] || 'gray'}
              dot={iconFor(app.status)}
            >
              <Space direction="vertical" size={2}>
                <Text strong>
                  {`第 ${app.level} 级`} {app.step ? `${app.step.toUpperCase()} ` : ''}审批
                </Text>
                <div>
                  <Tag color={approvalStatusColors[app.status]}>
                    {approvalStatusLabels[app.status]}
                  </Tag>
                  {app.approverName && (
                    <Text type="secondary" className="text-xs">
                      审批人：{app.approverName}
                    </Text>
                  )}
                </div>
                {app.comment && (
                  <Text type="secondary" italic className="text-xs">
                    &quot;{app.comment}&quot;
                  </Text>
                )}
                {app.processedAt && (
                  <Text type="secondary" className="text-xs">
                    {formatDateTime(app.processedAt)}
                  </Text>
                )}
              </Space>
            </Timeline.Item>
          ))}
        </Timeline>
      )}

      {showApprovalActions && canApprove && (
        <div className="mt-6 flex gap-2">
          {onApprove && (
            <Button
              type="primary"
              icon={<CheckCircle size={14} />}
              onClick={() => openModal('approve')}
            >
              通过
            </Button>
          )}
          {onReject && (
            <Button danger icon={<XCircle size={14} />} onClick={() => openModal('reject')}>
              拒绝
            </Button>
          )}
          {onDelegate && (
            <Button icon={<ArrowRight size={14} />} onClick={() => openModal('delegate')}>
              委派
            </Button>
          )}
        </div>
      )}

      <Modal
        title={modalTitle}
        open={modalOpen}
        onOk={handleSubmit}
        confirmLoading={submitting}
        onCancel={() => setModalOpen(false)}
        okText="提交"
        cancelText="取消"
        destroyOnClose
      >
        {currentAction === 'delegate' && (
          <div className="mb-3">
            <div className="mb-1 text-sm text-gray-600">委派给：</div>
            <Select
              showSearch
              value={delegateToUserId}
              placeholder="搜索并选择委派人"
              filterOption={false}
              onSearch={searchUsers}
              onChange={(v) => setDelegateToUserId(v)}
              style={{ width: '100%' }}
              loading={userSearchLoading}
              options={userOptions.map((u) => ({
                value: u.id,
                label: `${u.name || u.username || '未命名'}${u.username ? ` (${u.username})` : ''}`,
              }))}
            />
          </div>
        )}
        <div className="mb-1 text-sm text-gray-600">审批意见（必填）：</div>
        <TextArea
          rows={4}
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder="请填写审批意见"
        />
      </Modal>
    </div>
  );
};

export default ApprovalTimeline;
