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
} from 'antd';
import {
  UserOutlined,
  ThunderboltOutlined,
  CheckCircleOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import {
  TicketAssignmentApi,
  AssignRecommendation,
  AutoAssignResponse,
} from '@/lib/api/ticket-assignment-api';
import { message } from 'antd';

const { Title, Text, Paragraph } = Typography;

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
  const [loading, setLoading] = useState(false);
  const [recommendations, setRecommendations] = useState<AssignRecommendation[]>([]);
  const [autoAssigning, setAutoAssigning] = useState(false);

  // 加载推荐列表
  const loadRecommendations = async () => {
    if (!ticketId) return;

    setLoading(true);
    try {
      const response = await TicketAssignmentApi.getRecommendations(ticketId);
      setRecommendations(response.recommendations || []);
    } catch (error) {
      console.error('Failed to load recommendations:', error);
      message.error('获取分配推荐失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible && ticketId) {
      loadRecommendations();
    }
  }, [visible, ticketId]);

  // 自动分配
  const handleAutoAssign = async () => {
    if (!ticketId) return;

    setAutoAssigning(true);
    try {
      const response: AutoAssignResponse = await TicketAssignmentApi.autoAssign(ticketId);
      if (response.assigned_to) {
        message.success(`工单已自动分配给推荐的处理人`);
        onSuccess(response.assigned_to);
        onCancel();
      } else {
        message.warning('自动分配失败，请手动选择处理人');
      }
    } catch (error) {
      console.error('Failed to auto assign:', error);
      message.error('自动分配失败');
    } finally {
      setAutoAssigning(false);
    }
  };

  // 手动选择处理人
  const handleSelectAssignee = (userId: number) => {
    onSuccess(userId);
    onCancel();
  };

  // 获取评分颜色
  const getScoreColor = (score: number) => {
    if (score >= 80) return '#52c41a';
    if (score >= 60) return '#faad14';
    return '#ff4d4f';
  };

  // 获取评分标签
  const getScoreLabel = (score: number) => {
    if (score >= 80) return '优秀';
    if (score >= 60) return '良好';
    return '一般';
  };

  return (
    <Modal
      title={
        <Space>
          <ThunderboltOutlined style={{ color: '#1890ff' }} />
          <span>智能分配工单</span>
        </Space>
      }
      open={visible}
      onCancel={onCancel}
      footer={null}
      width={700}
      destroyOnHidden
    >
      <Spin spinning={loading}>
        <Space orientation='vertical' style={{ width: '100%' }} size='large'>
          {/* 说明信息 */}
          <Alert
            message='智能分配说明'
            description='系统将根据处理人的技能匹配度、当前工作负载、历史处理记录等因素，为您推荐最合适的处理人。'
            type='info'
            icon={<InfoCircleOutlined />}
            showIcon
          />

          {/* 自动分配按钮 */}
          <Card>
            <Space orientation='vertical' style={{ width: '100%' }}>
              <Text strong>快速操作</Text>
              <Button
                type='primary'
                icon={<ThunderboltOutlined />}
                size='large'
                block
                loading={autoAssigning}
                onClick={handleAutoAssign}
                disabled={recommendations.length === 0}
              >
                使用智能推荐自动分配
              </Button>
              <Text type='secondary' style={{ fontSize: 12 }}>
                系统将自动分配给评分最高的推荐处理人
              </Text>
            </Space>
          </Card>

          <Divider>或手动选择</Divider>

          {/* 推荐列表 */}
          {recommendations.length === 0 ? (
            <Empty description='暂无推荐处理人' />
          ) : (
            <List
              dataSource={recommendations}
              renderItem={(item, index) => (
                <List.Item
                  actions={[
                    <Button type='primary' onClick={() => handleSelectAssignee(item.user_id)}>
                      选择
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={<Avatar src={item.user_avatar} icon={<UserOutlined />} size='large' />}
                    title={
                      <Space>
                        <Text strong>{item.user_name}</Text>
                        {index === 0 && (
                          <Tag color='gold' icon={<CheckCircleOutlined />}>
                            最佳推荐
                          </Tag>
                        )}
                        <Tag color={getScoreColor(item.score)}>
                          评分: {item.score.toFixed(1)} ({getScoreLabel(item.score)})
                        </Tag>
                      </Space>
                    }
                    description={
                      <Space orientation='vertical' size='small' style={{ width: '100%' }}>
                        <Paragraph ellipsis={{ rows: 2, expandable: false }} style={{ margin: 0 }}>
                          <Text type='secondary'>{item.reason}</Text>
                        </Paragraph>
                        <Space size='middle'>
                          {item.factors.skill_match !== undefined && (
                            <Tooltip title='技能匹配度'>
                              <Text type='secondary' style={{ fontSize: 12 }}>
                                技能: {item.factors.skill_match}%
                              </Text>
                            </Tooltip>
                          )}
                          {item.factors.workload !== undefined && (
                            <Tooltip title='工作负载'>
                              <Text type='secondary' style={{ fontSize: 12 }}>
                                负载: {item.factors.workload}%
                              </Text>
                            </Tooltip>
                          )}
                          {item.factors.history_success !== undefined && (
                            <Tooltip title='历史成功率'>
                              <Text type='secondary' style={{ fontSize: 12 }}>
                                成功率: {item.factors.history_success}%
                              </Text>
                            </Tooltip>
                          )}
                        </Space>
                        <Progress
                          percent={item.score}
                          strokeColor={getScoreColor(item.score)}
                          size='small'
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
