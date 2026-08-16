/**
 * 批量操作进度模态框
 * 实时显示批量操作进度和结果
 */

'use client';

import React from 'react';
import { Modal, Progress, Space, Button, Alert, Statistic, Row, Col, Spin } from 'antd';
import { CheckCircle, XCircle, PlayCircle, PauseCircle, Square } from 'lucide-react';
import { useBatchOperationProgressQuery } from '@/lib/hooks/useBatchOperations';
import { BatchOperationStatus } from '@/types/batch-operations';
import { useI18n } from '@/lib/i18n/useI18n';

export interface BatchProgressModalProps {
  visible: boolean;
  operationId: string;
  onClose: () => void;
}

export const BatchProgressModal: React.FC<BatchProgressModalProps> = ({
  visible,
  operationId,
  onClose,
}) => {
  const { t } = useI18n();
  const { data: progress, isLoading, isError, refetch } = useBatchOperationProgressQuery(operationId, {
    enabled: visible,
    refetchInterval: 2000,
  });

  const getStatusColor = () => {
    switch (progress?.status) {
      case BatchOperationStatus.COMPLETED:
        return 'success';
      case BatchOperationStatus.RUNNING:
        return 'active';
      case BatchOperationStatus.FAILED:
        return 'exception';
      case BatchOperationStatus.PAUSED:
        return 'normal';
      default:
        return 'normal';
    }
  };

  const getStatusText = (): string => {
    switch (progress?.status) {
      case BatchOperationStatus.PENDING:
        return t('batchProgress.statusPending');
      case BatchOperationStatus.RUNNING:
        return t('batchProgress.statusRunning');
      case BatchOperationStatus.PAUSED:
        return t('batchProgress.statusPaused');
      case BatchOperationStatus.COMPLETED:
        return t('batchProgress.statusCompleted');
      case BatchOperationStatus.FAILED:
        return t('batchProgress.statusFailed');
      case BatchOperationStatus.CANCELLED:
        return t('batchProgress.statusCancelled');
      default:
        return t('batchProgress.statusUnknown');
    }
  };

  const isCompleted = progress?.status === BatchOperationStatus.COMPLETED;
  const isFailed = progress?.status === BatchOperationStatus.FAILED;
  const isRunning = progress?.status === BatchOperationStatus.RUNNING;
  const isPaused = progress?.status === BatchOperationStatus.PAUSED;

  return (
    <Modal
      title={t('batchProgress.title')}
      open={visible}
      onCancel={onClose}
      width={700}
      footer={[
        <Button key="close" type="primary" onClick={onClose} disabled={isRunning && !isPaused}>
          {isCompleted || isFailed ? t('common.close') : t('batchProgress.runInBackground')}
        </Button>,
      ]}
    >
      {isError ? (
        <Alert type="error" showIcon message={t('batchProgress.loadFailed')} description={t('batchProgress.loadFailedDesc')}
          action={<Button onClick={() => refetch()}>{t('common.retry')}</Button>} />
      ) : isLoading || !progress ? (
        <div className="flex justify-center items-center py-12">
          <Spin size="large" description={t('batchProgress.loading')} />
        </div>
      ) : (
        <Space orientation="vertical" size="large" className="w-full">
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-base font-semibold">{getStatusText()}</span>
              <span className="text-gray-600">
                {progress.processedCount} / {progress.totalCount}
              </span>
            </div>
            <Progress percent={progress.percentage} status={getStatusColor()} strokeWidth={12} />
          </div>

          <Row gutter={[16, 16]}>
            <Col xs={24} sm={8}>
              <Statistic
                title={t('batchProgress.total')}
                value={progress.totalCount}
                prefix={<CheckCircle />}
              />
            </Col>
            <Col xs={24} sm={8}>
              <Statistic
                title={t('batchProgress.success')}
                value={progress.successCount}
                styles={{ content: { color: '#3f8600' } }}
                prefix={<CheckCircle />}
              />
            </Col>
            <Col xs={24} sm={8}>
              <Statistic
                title={t('batchProgress.failed')}
                value={progress.failedCount}
                styles={{ content: { color: '#cf1322' } }}
                prefix={<XCircle />}
              />
            </Col>
          </Row>

          {isRunning && progress.currentTicket && (
            <Alert
              message={t('batchProgress.processing')}
              description={t('batchProgress.processingDesc', { ticketNumber: progress.currentTicket.ticketNumber })}
              type="info"
              showIcon
            />
          )}

          {isCompleted && (
            <Alert
              message={t('batchProgress.completed')}
              description={t('batchProgress.completedDesc', {
                successCount: progress.successCount,
                failedCount: progress.failedCount,
              })}
              type={progress.failedCount > 0 ? 'warning' : 'success'}
              showIcon
            />
          )}

          {isFailed && (
            <Alert
              message={t('batchProgress.failedTitle')}
              description={t('batchProgress.failedDesc')}
              type="error"
              showIcon
            />
          )}

          {isRunning && progress.estimatedCompletionTime && (
            <div className="text-sm text-gray-500">
              {t('batchProgress.estimatedTime')}
              {new Date(progress.estimatedCompletionTime).toLocaleTimeString()}
            </div>
          )}

          {isRunning && (
            <Space>
              {isPaused ? (
                <Button icon={<PlayCircle />}>{t('batchProgress.resume')}</Button>
              ) : (
                <Button icon={<PauseCircle />}>{t('batchProgress.pause')}</Button>
              )}
              <Button danger icon={<Square />}>
                {t('common.cancel')}
              </Button>
            </Space>
          )}
        </Space>
      )}
    </Modal>
  );
};

export default BatchProgressModal;