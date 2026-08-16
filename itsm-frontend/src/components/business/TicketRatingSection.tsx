'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Rate,
  Button,
  Form,
  Input,
  Space,
  Typography,
  Modal,
  message,
  Divider,
  Tag,
  Empty,
} from 'antd';
import { Star, MessageSquare, CheckCircle, Clock } from 'lucide-react';
import type {
  TicketRating,
  SubmitTicketRatingRequest} from '@/lib/api/ticket-rating-api';
import {
  TicketRatingApi
} from '@/lib/api/ticket-rating-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { App } from 'antd';
import { useI18n } from '@/lib/i18n';

const { Text, Title } = Typography;
const { TextArea } = Input;

interface TicketRatingSectionProps {
  ticketId: number;
  ticketStatus: string;
  requesterId: number;
  canRate?: boolean;
  onRatingSubmitted?: (rating: TicketRating) => void;
}

/**
 * 工单评分组件
 */
export const TicketRatingSection: React.FC<TicketRatingSectionProps> = ({
  ticketId,
  ticketStatus,
  requesterId,
  canRate = true,
  onRatingSubmitted,
}) => {
  const { t } = useI18n();
  const { message: antMessage } = App.useApp();
  const { user } = useAuthStore();
  const [rating, setRating] = useState<TicketRating | null>(null);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [showRatingModal, setShowRatingModal] = useState(false);
  const [form] = Form.useForm();

  // 检查是否可以评分
  const canShowRating = () => {
    if (!canRate) return false;
    if (!user || user.id !== requesterId) return false; // 只有申请人可以评分
    if (ticketStatus !== 'resolved' && ticketStatus !== 'closed') return false; // 只有已解决或已关闭的工单可以评分
    if (rating && rating.rating > 0) return false; // 已经评分过了
    return true;
  };

  // 加载评分信息
  const loadRating = async () => {
    setLoading(true);
    try {
      const ratingData = await TicketRatingApi.getRating(ticketId);
      setRating(ratingData);
    } catch (error) {
      // 如果获取失败，可能是还没有评分，不显示错误
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (ticketId) {
      loadRating();
    }
  }, [ticketId]);

  // 提交评分
  const handleSubmitRating = async (values: { rating: number; comment?: string }) => {
    setSubmitting(true);
    try {
      const request: SubmitTicketRatingRequest = {
        rating: values.rating,
        comment: values.comment || '',
      };
      const newRating = await TicketRatingApi.submitRating(ticketId, request);
      setRating(newRating);
      antMessage.success(t('ticketRating.thanksForRating'));
      setShowRatingModal(false);
      form.resetFields();
      if (onRatingSubmitted) {
        onRatingSubmitted(newRating);
      }
    } catch (error: unknown) {
      antMessage.error(error instanceof Error ? error.message : t('ticketRating.submitFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  // 格式化时间
  const formatDateTime = (dateString?: string) => {
    if (!dateString) return '';
    return new Date(dateString).toLocaleString('zh-CN');
  };

  // 获取评分描述
  const getRatingDescription = (ratingValue: number) => {
    const map: Record<number, string> = {
      1: t('ticketRating.description.1'),
      2: t('ticketRating.description.2'),
      3: t('ticketRating.description.3'),
      4: t('ticketRating.description.4'),
      5: t('ticketRating.description.5'),
    };
    return map[ratingValue] || '';
  };

  // 获取评分颜色
  const getRatingColor = (ratingValue: number) => {
    if (ratingValue >= 4) return '#52c41a'; // 绿色
    if (ratingValue >= 3) return '#faad14'; // 橙色
    return '#ff4d4f'; // 红色
  };

  if (loading) {
    return (
      <Card loading={loading}>
        <Empty description={t('ticketRating.loading')} />
      </Card>
    );
  }

  // 如果已经评分，显示评分信息
  if (rating && rating.rating > 0) {
    return (
      <Card
        title={
          <Space>
            <Star style={{ color: '#faad14' }} />
            <span>{t('ticketRating.cardTitle')}</span>
          </Space>
        }
        className="shadow-sm"
      >
        <div className="space-y-4">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <Rate disabled value={rating.rating} />
              <Text strong style={{ fontSize: 18, color: getRatingColor(rating.rating) }}>
                {t('ticketRating.scoreLabel', { count: rating.rating })}
              </Text>
              <Tag
                color={rating.rating >= 4 ? 'success' : rating.rating >= 3 ? 'warning' : 'error'}
              >
                {getRatingDescription(rating.rating)}
              </Tag>
            </div>
          </div>

          {rating.comment && (
            <div>
              <div className="flex items-center space-x-2 mb-2">
                <MessageSquare style={{ fontSize: 16, color: '#8c8c8c' }} />
                <Text type="secondary" strong>
                  {t('ticketRating.commentTitle')}
                </Text>
              </div>
              <div className="p-3 bg-gray-50 rounded-md">
                <Text>{rating.comment}</Text>
              </div>
            </div>
          )}

          <Divider style={{ margin: '12px 0' }} />

          <div className="flex items-center space-x-4 text-sm text-gray-500">
            {rating.ratedAt && (
              <div className="flex items-center space-x-1">
                <Clock style={{ fontSize: 14 }} />
                <Text>{t('ticketRating.ratedAt', { time: formatDateTime(rating.ratedAt) })}</Text>
              </div>
            )}
            {rating.ratedByName && (
              <div className="flex items-center space-x-1">
                <CheckCircle style={{ fontSize: 14 }} />
                <Text>{t('ticketRating.ratedBy', { name: rating.ratedByName })}</Text>
              </div>
            )}
          </div>
        </div>
      </Card>
    );
  }

  // 如果可以评分，显示评分入口
  if (canShowRating()) {
    return (
      <Card
        title={
          <Space>
            <Star style={{ color: '#faad14' }} />
            <span>{t('ticketRating.serviceRatingTitle')}</span>
          </Space>
        }
        className="shadow-sm"
      >
        <div className="space-y-4">
          <div className="text-center py-4">
            <Text type="secondary" className="block mb-4">
              {t('ticketRating.resolvedHint')}
            </Text>
            <Button
              type="primary"
              size="large"
              icon={<Star />}
              onClick={() => setShowRatingModal(true)}
              className="bg-gradient-to-r from-yellow-400 to-orange-500 border-0 hover:from-yellow-500 hover:to-orange-600"
            >
              {t('ticketRating.rateNow')}
            </Button>
          </div>
        </div>

        {/* 评分模态框 */}
        <Modal
          title={
            <Space>
              <Star style={{ color: '#faad14' }} />
              <span>{t('ticketRating.modalTitle')}</span>
            </Space>
          }
          open={showRatingModal}
          onOk={() => form.submit()}
          onCancel={() => {
            setShowRatingModal(false);
            form.resetFields();
          }}
          okText={t('common.submit')}
          cancelText={t('common.cancel')}
          confirmLoading={submitting}
          width={500}
        >
          <Form
            form={form}
            layout="vertical"
            onFinish={handleSubmitRating}
            initialValues={{
              rating: 5,
            }}
          >
            <Form.Item
              label={t('ticketRating.fieldRating')}
              name="rating"
              rules={[{ required: true, message: t('ticketRating.ratingRequired') }]}
            >
              <Rate allowClear={false} style={{ fontSize: 32 }} character={<Star />} />
            </Form.Item>

            <Form.Item label={t('ticketRating.commentLabel')} name="comment">
              <TextArea
                rows={4}
                placeholder={t('ticketRating.commentPlaceholder')}
                showCount
                maxLength={500}
              />
            </Form.Item>

            <div className="text-sm text-gray-500 mt-4">
              <Text type="secondary">{t('ticketRating.feedbackHint')}</Text>
            </div>
          </Form>
        </Modal>
      </Card>
    );
  }

  // 如果不满足评分条件，不显示任何内容
  return null;
};
