/**
 * 批量操作进度模态框
 * 实时显示批量操作进度和结果
 */

'use client';

import React from 'react';
import {
  Modal,
  Progress,
  Space,
  Button,
  Alert,
  Statistic,
  Row,
  Col,
  List,
  Tag,
  Spin,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  StopOutlined,
  DownloadOutlined,
} from '@ant-design/icons';
import { useBatchOperationProgressQuery } from '@/lib/hooks/useBatchOperations';
import { BatchOperationStatus } from '@/types/batch-operations';

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
  const { data: progress, isLoading } = useBatchOperationProgressQuery(
    operationId,
    {
      enabled: visible,
      refetchInterval: 2000,
    }
  );

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

  const getStatusText = () => {
    switch (progress?.status) {
      case BatchOperationStatus.PENDING:
        return '等待中';
      case BatchOperationStatus.RUNNING:
        return '执行中';
      case BatchOperationStatus.PAUSED:
        return '已暂停';
      case BatchOperationStatus.COMPLETED:
        return '已完成';
      case BatchOperationStatus.FAILED:
        return '失败';
      case BatchOperationStatus.CANCELLED:
        return '已取消';
      default:
        return '未知';
    }
  };

  const isCompleted = progress?.status === BatchOperationStatus.COMPLETED;
  const isFailed = progress?.status === BatchOperationStatus.FAILED;
  const isRunning = progress?.status === BatchOperationStatus.RUNNING;
  const isPaused = progress?.status === BatchOperationStatus.PAUSED;

  return (
    <Modal
      title="批量操作进度"
      open={visible}
      onCancel={onClose}
      width={700}
      footer={[
        <Button key="close" type="primary" onClick={onClose} disabled={isRunning && !isPaused}>
          {isCompleted || isFailed ? '关闭' : '后台运行'}
        </Button>,
      ]}
    >
      {isLoading || !progress ? (
        <div className="flex justify-center items-center py-12">
          <Spin size="large" />
        </div>
      ) : (
        <Space direction="vertical" size="large" className="w-full">
          {/* 进度条 */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-base font-semibold">
                {getStatusText()}
              </span>
              <span className="text-gray-600">
                {progress.processedCount} / {progress.totalCount}
              </span>
            </div>
            <Progress
              percent={progress.percentage}
              status={getStatusColor()}
              strokeWidth={12}
            />
          </div>

          {/* 统计信息 */}
          <Row gutter={16}>
            <Col span={8}>
              <Statistic
                title="总数"
                value={progress.totalCount}
                prefix={<CheckCircleOutlined />}
              />
            </Col>
            <Col span={8}>
              <Statistic
                title="成功"
                value={progress.successCount}
                valueStyle={{ color: '#3f8600' }}
                prefix={<CheckCircleOutlined />}
              />
            </Col>
            <Col span={8}>
              <Statistic
                title="失败"
                value={progress.failedCount}
                valueStyle={{ color: '#cf1322' }}
                prefix={<CloseCircleOutlined />}
              />
            </Col>
          </Row>

          {/* 当前处理工单 */}
          {isRunning && progress.currentTicket && (
            <Alert
              message="正在处理"
              description={`工单 #${progress.currentTicket.ticketNumber}`}
              type="info"
              showIcon
            />
          )}

          {/* 完成提示 */}
          {isCompleted && (
            <Alert
              message="批量操作完成"
              description={`成功处理 ${progress.successCount} 个工单${
                progress.failedCount > 0
                  ? `，失败 ${progress.failedCount} 个`
                  : ''
              }`}
              type={progress.failedCount > 0 ? 'warning' : 'success'}
              showIcon
            />
          )}

          {/* 失败提示 */}
          {isFailed && (
            <Alert
              message="批量操作失败"
              description="请查看错误日志了解详情"
              type="error"
              showIcon
            />
          )}

          {/* 预计完成时间 */}
          {isRunning && progress.estimatedCompletionTime && (
            <div className="text-sm text-gray-500">
              预计完成时间：
              {new Date(progress.estimatedCompletionTime).toLocaleTimeString()}
            </div>
          )}

          {/* 操作按钮 */}
          {isRunning && (
            <Space>
              {isPaused ? (
                <Button icon={<PlayCircleOutlined />}>继续</Button>
              ) : (
                <Button icon={<PauseCircleOutlined />}>暂停</Button>
              )}
              <Button danger icon={<StopOutlined />}>
                取消
              </Button>
            </Space>
          )}
        </Space>
      )}
    </Modal>
  );
};

export default BatchProgressModal;

