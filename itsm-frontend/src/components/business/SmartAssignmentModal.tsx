/**
 * 智能分配模态框组件
 * 显示分配推荐列表，支持自动分配和手动选择
 */

'use client';

import React, { useState, useEffect } from 'react';
import {
  Modal,
  List,
  Button,
  Space,
  Typography,
  Tag,
  Avatar,
  Progress,
  Card,
  Alert,
  Spin,
  Empty,
  Tooltip,
  Divider,
  App,
} from 'antd';
import { User, Info, CheckCircle, Zap } from 'lucide-react';
import type {
  AssignRecommendation,
  AutoAssignResponse,
} from '@/lib/api/ticket-assignment-api';
import { TicketAssignmentApi } from '@/lib/api/ticket-assignment-api';
import { useI18n } from '@/lib/i18n/useI18n';

const { Text, Paragraph } = Typography;

interface SmartAssignmentModalProps {
  visible: boolean;
  ticketId: number;
  onCancel: () => void;
  onSuccess: (assignedTo: number) => void;
}

export const SmartAssignmentModal: React.FC<SmartAssignmentModalProps> = ({
  visible,
  ticketId,
  onCancel,
  onSuccess,
}) => {
  const { t } = useI18n();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [recommendations, setRecommendations] = useState<AssignRecommendation[]>([]);
  const [autoAssigning, setAutoAssigning] = useState(false);

  const loadRecommendations = async () => {
    if (!ticketId) return;
    setLoading(true);
    try {
      const response = await TicketAssignmentApi.getRecommendations(ticketId);
      setRecommendations(response.recommendations || []);
    } catch (error) {
      message.error(t('smartAssignment.recommendFailed'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible && ticketId) {
      loadRecommendations();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, ticketId]);

  const handleAutoAssign = async () => {
    if (!ticketId) return;
    setAutoAssigning(true);
    try {
      const response: AutoAssignResponse = await TicketAssignmentApi.autoAssign(ticketId);
      if (response.assignedTo) {
        message.success(t('smartAssignment.autoAssigned'));
        onSuccess(response.assignedTo);
        onCancel();
      } else {
        message.warning(t('smartAssignment.autoAssignFailedFallback'));
      }
    } catch (error) {
      message.error(t('smartAssignment.autoAssignFailed'));
    } finally {
      setAutoAssigning(false);
    }
  };

  const handleSelectAssignee = (userId: number) => {
    onSuccess(userId);
    onCancel();
  };

  const getScoreColor = (score: number) => {
    if (score >= 80) return '#52c41a';
    if (score >= 60) return '#faad14';
    return '#ff4d4f';
  };

  const getScoreLabel = (score: number): string => {
    if (score >= 80) return t('smartAssignment.scoreExcellent');
    if (score >= 60) return t('smartAssignment.scoreGood');
    return t('smartAssignment.scoreAverage');
  };

  return (
    <Modal
      title={
        <Space>
          <Zap style={{ color: '#1890ff' }} />
          <span>{t('smartAssignment.title')}</span>
        </Space>
      }
      open={visible}
      onCancel={onCancel}
      footer={null}
      width={700}
      destroyOnHidden
    >
      <Spin spinning={loading}>
        <Space orientation="vertical" style={{ width: '100%' }} size="large">
          <Alert
            message={t('smartAssignment.infoTitle')}
            description={t('smartAssignment.infoDescription')}
            type="info"
            icon={<Info />}
            showIcon
          />

          <Card>
            <Space orientation="vertical" style={{ width: '100%' }}>
              <Text strong>{t('smartAssignment.quickAction')}</Text>
              <Button
                type="primary"
                icon={<Zap />}
                size="large"
                block
                loading={autoAssigning}
                onClick={handleAutoAssign}
                disabled={recommendations.length === 0}
              >
                {t('smartAssignment.autoAssignButton')}
              </Button>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('smartAssignment.autoAssignHint')}
              </Text>
            </Space>
          </Card>

          <Divider>{t('smartAssignment.orManual')}</Divider>

          {recommendations.length === 0 ? (
            <Empty description={t('smartAssignment.empty')} />
          ) : (
            <List
              dataSource={recommendations}
              renderItem={(item, index) => (
                <List.Item
                  actions={[
                    <Button
                      key="select"
                      type="primary"
                      onClick={() => handleSelectAssignee(item.userId)}
                    >
                      {t('smartAssignment.select')}
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={<Avatar src={item.userAvatar} icon={<User />} size="large" />}
                    title={
                      <Space>
                        <Text strong>{item.userName}</Text>
                        {index === 0 && (
                          <Tag color="gold" icon={<CheckCircle />}>
                            {t('smartAssignment.bestMatch')}
                          </Tag>
                        )}
                        <Tag color={getScoreColor(item.score)}>
                          {t('smartAssignment.scoreTag', {
                            score: item.score.toFixed(1),
                            label: getScoreLabel(item.score),
                          })}
                        </Tag>
                      </Space>
                    }
                    description={
                      <Space orientation="vertical" size="small" style={{ width: '100%' }}>
                        <Paragraph ellipsis={{ rows: 2, expandable: false }} style={{ margin: 0 }}>
                          <Text type="secondary">{item.reason}</Text>
                        </Paragraph>
                        <Space size="middle">
                          {item.factors.skillMatch !== undefined && (
                            <Tooltip title={t('smartAssignment.skillMatch')}>
                              <Text type="secondary" style={{ fontSize: 12 }}>
                                {t('smartAssignment.skill')}: {item.factors.skillMatch}%
                              </Text>
                            </Tooltip>
                          )}
                          {item.factors.workload !== undefined && (
                            <Tooltip title={t('smartAssignment.workload')}>
                              <Text type="secondary" style={{ fontSize: 12 }}>
                                {t('smartAssignment.load')}: {item.factors.workload}%
                              </Text>
                            </Tooltip>
                          )}
                          {item.factors.historySuccess !== undefined && (
                            <Tooltip title={t('smartAssignment.historySuccessRate')}>
                              <Text type="secondary" style={{ fontSize: 12 }}>
                                {t('smartAssignment.successRate')}: {item.factors.historySuccess}%
                              </Text>
                            </Tooltip>
                          )}
                        </Space>
                        <Progress
                          percent={item.score}
                          strokeColor={getScoreColor(item.score)}
                          size="small"
                          showInfo={false}
                        />
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          )}
        </Space>
      </Spin>
    </Modal>
  );
};