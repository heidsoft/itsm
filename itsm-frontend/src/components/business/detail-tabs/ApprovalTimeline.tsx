'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Timeline, Typography, Tag, Space, Button, Modal, Input, App, Empty, Select } from 'antd';
import { CheckCircle, XCircle, Timer, ArrowRight } from 'lucide-react';
import type { ApprovalStep, ApprovalStepStatus, ApprovalActionInput } from './types';
import { UserApi } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n/useI18n';

const { Text } = Typography;
const { TextArea } = Input;

type Action = 'approve' | 'reject' | 'delegate';

interface CandidateUser {
  id: number;
  name?: string;
  username?: string;
}

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
  formatDateTime,
}) => {
  const { t, language } = useI18n();
  const { message } = App.useApp();
  const [currentAction, setCurrentAction] = useState<Action | null>(null);
  const [comment, setComment] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [delegateToUserId, setDelegateToUserId] = useState<number | undefined>();
  const [userOptions, setUserOptions] = useState<CandidateUser[]>([]);
  const [userSearchLoading, setUserSearchLoading] = useState(false);

  const approvalStatusColors: Record<ApprovalStepStatus, string> = {
    pending: 'blue',
    approved: 'green',
    rejected: 'red',
    delegated: 'purple',
    timeout: 'orange',
    skipped: 'gray',
  };

  const approvalStatusLabels = useMemo(
    () =>
      ({
        pending: t('detailTabs.approvalStatusPending'),
        approved: t('detailTabs.approvalStatusApproved'),
        rejected: t('detailTabs.approvalStatusRejected'),
        delegated: t('detailTabs.approvalStatusDelegated'),
        timeout: t('detailTabs.approvalStatusTimeout'),
        skipped: t('detailTabs.approvalStatusSkipped'),
      }) as Record<ApprovalStepStatus, string>,
    [t],
  );

  const defaultFormat = useCallback(
    (s?: string) =>
      s ? new Date(s).toLocaleString(language === 'en-US' ? 'en-US' : 'zh-CN') : '',
    [language],
  );
  const fmt = formatDateTime ?? defaultFormat;

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
      message.warning(t('detailTabs.approvalCommentRequired'));
      return;
    }
    if (currentAction === 'delegate' && !delegateToUserId) {
      message.warning(t('detailTabs.delegateUserRequired'));
      return;
    }
    if (!currentAction) return;
    setSubmitting(true);
    try {
      const payload: ApprovalActionInput = { comment, delegateToUserId };
      if (currentAction === 'approve') await onApprove?.(payload);
      else if (currentAction === 'reject') await onReject?.(payload);
      else if (currentAction === 'delegate') await onDelegate?.(payload);
      message.success(t('detailTabs.approvalSuccess'));
      setModalOpen(false);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('detailTabs.approvalFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const modalTitle =
    currentAction === 'approve'
      ? t('detailTabs.approveAction')
      : currentAction === 'reject'
      ? t('detailTabs.rejectAction')
      : t('detailTabs.delegateAction');

  return (
    <div className="p-6">
      {workflowName && (
        <div className="mb-4">
          <Text type="secondary">{t('detailTabs.workflowLabel')}：</Text>
          <Text strong>{workflowName}</Text>
          {currentLevel !== undefined && (
            <Tag className="ml-2" color="blue">
              {t('detailTabs.currentLevel', { level: currentLevel })}
            </Tag>
          )}
        </div>
      )}

      {approvals.length === 0 && !submittedAt ? (
        <Empty description={t('detailTabs.noApprovals')} />
      ) : (
        <Timeline>
          {submittedAt && (
            <Timeline.Item color="green">
              <div>
                <Text strong>{submitterName || t('detailTabs.submitter')}</Text>
                <Tag color="green" className="ml-2">
                  {t('detailTabs.submitted')}
                </Tag>
              </div>
              <Text type="secondary" className="text-xs">
                {fmt(submittedAt)}
              </Text>
            </Timeline.Item>
          )}
          {approvals.map((app) => (
            <Timeline.Item
              key={app.id}
              color={approvalStatusColors[app.status] || 'gray'}
              dot={iconFor(app.status)}
            >
              <Space orientation="vertical" size={2}>
                <Text strong>
                  {t('detailTabs.levelLabel', { level: app.level })}{' '}
                  {app.step ? `${app.step.toUpperCase()} ` : ''}
                  {t('detailTabs.approval')}
                </Text>
                <div>
                  <Tag color={approvalStatusColors[app.status]}>
                    {approvalStatusLabels[app.status]}
                  </Tag>
                  {app.approverName && (
                    <Text type="secondary" className="text-xs">
                      {t('detailTabs.approverLabel')}：{app.approverName}
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
                    {fmt(app.processedAt)}
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
              {t('detailTabs.approve')}
            </Button>
          )}
          {onReject && (
            <Button danger icon={<XCircle size={14} />} onClick={() => openModal('reject')}>
              {t('detailTabs.reject')}
            </Button>
          )}
          {onDelegate && (
            <Button icon={<ArrowRight size={14} />} onClick={() => openModal('delegate')}>
              {t('detailTabs.delegate')}
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
        okText={t('common.submit')}
        cancelText={t('common.cancel')}
        destroyOnHidden
      >
        {currentAction === 'delegate' && (
          <div className="mb-3">
            <div className="mb-1 text-sm text-gray-600">{t('detailTabs.delegateTo')}：</div>
            <Select
              showSearch
              value={delegateToUserId}
              placeholder={t('detailTabs.delegatePlaceholder')}
              filterOption={false}
              onSearch={searchUsers}
              onChange={(v) => setDelegateToUserId(v)}
              style={{ width: '100%' }}
              loading={userSearchLoading}
              options={userOptions.map((u) => ({
                value: u.id,
                label: `${u.name || u.username || t('detailTabs.unknownUser')}${u.username ? ` (${u.username})` : ''}`,
              }))}
            />
          </div>
        )}
        <div className="mb-1 text-sm text-gray-600">{t('detailTabs.approvalCommentRequired')}</div>
        <TextArea
          rows={4}
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder={t('detailTabs.approvalCommentPlaceholder')}
        />
      </Modal>
    </div>
  );
};

export default ApprovalTimeline;
