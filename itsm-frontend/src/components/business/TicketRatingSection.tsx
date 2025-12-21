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
import {
  Star,
  MessageSquare,
  CheckCircle,
  Clock,
} from 'lucide-react';
import { TicketRatingApi, TicketRating, SubmitTicketRatingRequest } from '@/lib/api/ticket-rating-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { App } from 'antd';

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
      console.error('Failed to load rating:', error);
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
      antMessage.success('感谢您的评分！');
      setShowRatingModal(false);
      form.resetFields();
      if (onRatingSubmitted) {
        onRatingSubmitted(newRating);
      }
    } catch (error: any) {
      console.error('Failed to submit rating:', error);
      antMessage.error(error.message || '提交评分失败');
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
    const descriptions: Record<number, string> = {
      1: '非常不满意',
      2: '不满意',
      3: '一般',
      4: '满意',
      5: '非常满意',
    };
    return descriptions[ratingValue] || '';
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
        <Empty description="加载中..." />
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
            <span>评分信息</span>
          </Space>
        }
        className="shadow-sm"
      >
        <div className="space-y-4">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <Rate disabled value={rating.rating} />
              <Text strong style={{ fontSize: 18, color: getRatingColor(rating.rating) }}>
                {rating.rating} 星
              </Text>
              <Tag color={rating.rating >= 4 ? 'success' : rating.rating >= 3 ? 'warning' : 'error'}>
                {getRatingDescription(rating.rating)}
              </Tag>
            </div>
          </div>

          {rating.comment && (
            <div>
              <div className="flex items-center space-x-2 mb-2">
                <MessageSquare style={{ fontSize: 16, color: '#8c8c8c' }} />
                <Text type="secondary" strong>评分评论</Text>
              </div>
              <div className="p-3 bg-gray-50 rounded-md">
                <Text>{rating.comment}</Text>
              </div>
            </div>
          )}

          <Divider style={{ margin: '12px 0' }} />

          <div className="flex items-center space-x-4 text-sm text-gray-500">
            {rating.rated_at && (
              <div className="flex items-center space-x-1">
                <Clock style={{ fontSize: 14 }} />
                <Text>评分时间: {formatDateTime(rating.rated_at)}</Text>
              </div>
            )}
            {rating.rated_by_name && (
              <div className="flex items-center space-x-1">
                <CheckCircle style={{ fontSize: 14 }} />
                <Text>评分人: {rating.rated_by_name}</Text>
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
            <span>服务评分</span>
          </Space>
        }
        className="shadow-sm"
      >
        <div className="space-y-4">
          <div className="text-center py-4">
            <Text type="secondary" className="block mb-4">
              工单已解决，请为本次服务评分
            </Text>
            <Button
              type="primary"
              size="large"
              icon={<Star />}
              onClick={() => setShowRatingModal(true)}
              className="bg-gradient-to-r from-yellow-400 to-orange-500 border-0 hover:from-yellow-500 hover:to-orange-600"
            >
              立即评分
            </Button>
          </div>
        </div>

        {/* 评分模态框 */}
        <Modal
          title={
            <Space>
              <Star style={{ color: '#faad14' }} />
              <span>为本次服务评分</span>
            </Space>
          }
          open={showRatingModal}
          onOk={() => form.submit()}
          onCancel={() => {
            setShowRatingModal(false);
            form.resetFields();
          }}
          okText="提交评分"
          cancelText="取消"
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
              label="评分"
              name="rating"
              rules={[{ required: true, message: '请选择评分' }]}
            >
              <Rate
                allowClear={false}
                style={{ fontSize: 32 }}
                character={<Star />}
              />
            </Form.Item>

            <Form.Item
              label="评分评论（可选）"
              name="comment"
            >
              <TextArea
                rows={4}
                placeholder="请分享您对本次服务的评价..."
                showCount
                maxLength={500}
              />
            </Form.Item>

            <div className="text-sm text-gray-500 mt-4">
              <Text type="secondary">
                您的评分将帮助我
们改进服务质量，感谢您的反馈！
              </Text>
            </div>
          </Form>
        </Modal>
      </Card>
    );
  }

  // 如果不满足评分条件，不显示任何内容
  return null;
};

