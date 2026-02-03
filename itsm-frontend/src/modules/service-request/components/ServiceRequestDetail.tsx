'use client';

/**
 * 服务请求详情组件
 */

import React, { useState, useEffect } from 'react';
import {
  Card,
  Descriptions,
  Tag,
  Timeline,
  Button,
  Typography,
  Space,
  Modal,
  Input,
  message,
  Spin,
  Divider,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  FieldTimeOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ServiceRequestApi } from '../api';
import { ServiceRequestStatus, ApprovalStatus, ApprovalAction } from '../constants';
import type { ServiceRequest, ServiceRequestApproval } from '../types';

const { Title, Text } = Typography;
const { TextArea } = Input;

// 审批状态颜色映射
const approvalStatusColors: Record<string, string> = {
  [ApprovalStatus.PENDING]: 'orange',
  [ApprovalStatus.APPROVED]: 'green',
  [ApprovalStatus.REJECTED]: 'red',
};

const ServiceRequestDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [request, setRequest] = useState<ServiceRequest | null>(null);
  const [approvals, setApprovals] = useState<ServiceRequestApproval[]>([]);

  // 审批动作弹窗状态
  const [actionModalVisible, setActionModalVisible] = useState(false);
  const [currentAction, setCurrentAction] = useState<ApprovalAction | null>(null);
  const [comment, setComment] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // 加载详情
  const loadDetail = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const data = await ServiceRequestApi.getServiceRequest(id);
      setRequest(data);
      setApprovals(data.approvals || []);
    } catch (error) {
      // console.error(error);
      message.error('加载详情失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDetail();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  // 提交审批
  const handleSubmitApproval = async () => {
    if (!id || !currentAction) return;

    if (currentAction === ApprovalAction.REJECT && !comment.trim()) {
      message.error('拒绝操作必须填写原因');
      return;
    }

    setSubmitting(true);
    try {
      await ServiceRequestApi.applyApproval(id, {
        action: currentAction === ApprovalAction.APPROVE ? 'approve' : 'reject',
        comment: comment,
      });
      message.success('操作成功');
      setActionModalVisible(false);
      loadDetail(); // 刷新数据
    } catch (error: any) {
      // console.error(error);
      message.error(error.message || '操作失败');
    } finally {
      setSubmitting(false);
    }
  };

  const openActionModal = (action: ApprovalAction) => {
    setCurrentAction(action);
    setComment('');
    setActionModalVisible(true);
  };

  // 渲染审批时间轴
  const renderApprovalTimeline = () => {
    return (
      <Timeline>
        <Timeline.Item color='green'>
          <p>提交申请</p>
          <small>{dayjs(request?.created_at).format('YYYY-MM-DD HH:mm')}</small>
        </Timeline.Item>
        {approvals.map((app, index) => (
          <Timeline.Item
            key={app.id}
            color={approvalStatusColors[app.status] || 'gray'}
            dot={
              app.status === ApprovalStatus.APPROVED ? (
                <CheckCircleOutlined />
              ) : app.status === ApprovalStatus.REJECTED ? (
                <CloseCircleOutlined />
              ) : (
                <FieldTimeOutlined />
              )
            }
          >
            <Space orientation='vertical' size={2}>
              <Text strong>{`${app.level}. ${app.step.toUpperCase()} 审批`}</Text>
              <div>
                <Tag color={approvalStatusColors[app.status]}>{app.status}</Tag>
                {app.approver_name && <Text type='secondary'>by {app.approver_name}</Text>}
              </div>
              {app.comment && (
                <Text type='secondary' italic>
                  &quot;{app.comment}&quot;
                </Text>
              )}
              {app.processed_at && (
                <div style={{ fontSize: '12px', color: '#999' }}>
                  {dayjs(app.processed_at).format('YYYY-MM-DD HH:mm')}
                </div>
              )}
            </Space>
          </Timeline.Item>
        ))}
      </Timeline>
    );
  };

  // 判断当前用户是否可以审批 (简化逻辑：只要有 pending 状态且页面显示了按钮，前端暂不深度校验 user role，依赖后端拦截)
  // 实际生产中应结合当前 userInfo 判断
  const canApprove = approvals.some(
    a => a.status === ApprovalStatus.PENDING && a.level === request?.current_level
  );

  if (loading) return <Spin size='large' style={{ display: 'block', margin: '50px auto' }} />;
  if (!request) return <div>未找到请求</div>;

  return (
    <div style={{ padding: '24px' }}>
      <Space orientation='vertical' size='large' style={{ width: '100%' }}>
        {/* 头部信息 */}
        <Card variant="borderless">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
            <div>
              <Title level={3}>{request.title || '服务请求'}</Title>
              <Space size='middle'>
                <Tag>{request.catalog?.category}</Tag>
                <Text type='secondary'>ID: {request.id}</Text>
                <Text type='secondary'>
                  提交于: {dayjs(request.created_at).format('YYYY-MM-DD HH:mm')}
                </Text>
              </Space>
            </div>
            <div style={{ textAlign: 'right' }}>
              <Title level={4} style={{ margin: 0 }}>
                <Tag color={request.status === 'completed' ? 'green' : 'blue'}>
                  {request.status.toUpperCase()}
                </Tag>
              </Title>
            </div>
          </div>
        </Card>

        <div style={{ display: 'flex', gap: '24px' }}>
          {/* 左侧：详情 */}
          <div style={{ flex: 2 }}>
            <Card title='请求详情' variant="borderless">
              <Descriptions column={1} bordered>
                <Descriptions.Item label='服务名称'>{request.catalog?.name}</Descriptions.Item>
                <Descriptions.Item label='申请原因'>{request.reason}</Descriptions.Item>
                <Descriptions.Item label='成本中心'>{request.cost_center || '-'}</Descriptions.Item>
                <Descriptions.Item label='数据分类'>
                  {request.data_classification || 'Public'}
                </Descriptions.Item>
                <Descriptions.Item label='需要公网IP'>
                  {request.needs_public_ip ? '是' : '否'}
                </Descriptions.Item>
                {request.form_data && (
                  <Descriptions.Item label='表单数据'>
                    <pre style={{ margin: 0, fontSize: '12px' }}>
                      {JSON.stringify(request.form_data, null, 2)}
                    </pre>
                  </Descriptions.Item>
                )}
              </Descriptions>
            </Card>
          </div>

          {/* 右侧：审批流 & 操作 */}
          <div style={{ flex: 1 }}>
            <Card title='审批流程' variant="borderless">
              {renderApprovalTimeline()}

              {/* 审批操作区 */}
              {canApprove &&
                request.status !== ServiceRequestStatus.REJECTED &&
                request.status !== ServiceRequestStatus.CANCELLED && (
                  <>
                    <Divider />
                    <Title level={5}>审批操作</Title>
                    <Space style={{ width: '100%', justifyContent: 'center' }}>
                      <Button
                        type='primary'
                        icon={<CheckCircleOutlined />}
                        onClick={() => openActionModal(ApprovalAction.APPROVE)}
                      >
                        通过
                      </Button>
                      <Button
                        danger
                        icon={<CloseCircleOutlined />}
                        onClick={() => openActionModal(ApprovalAction.REJECT)}
                      >
                        拒绝
                      </Button>
                    </Space>
                  </>
                )}
            </Card>
          </div>
        </div>
      </Space>

      {/* 审批确认弹窗 */}
      <Modal
        title={currentAction === ApprovalAction.APPROVE ? '确认批准' : '确认拒绝'}
        open={actionModalVisible}
        onOk={handleSubmitApproval}
        onCancel={() => setActionModalVisible(false)}
        confirmLoading={submitting}
        okText='提交'
        cancelText='取消'
        okButtonProps={{ danger: currentAction === ApprovalAction.REJECT }}
      >
        <p>确定要{currentAction === ApprovalAction.APPROVE ? '批准' : '拒绝'}此请求吗？</p>
        <TextArea
          rows={4}
          value={comment}
          onChange={e => setComment(e.target.value)}
          placeholder={
            currentAction === ApprovalAction.REJECT ? '请填写拒绝原因 (必填)' : '审批意见 (选填)'
          }
        />
      </Modal>
    </div>
  );
};

export default ServiceRequestDetail;
