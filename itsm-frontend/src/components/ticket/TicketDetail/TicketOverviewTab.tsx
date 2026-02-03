'use client';

import React from 'react';
import { CheckCircle, XCircle, Clock, Zap } from 'lucide-react';
import { Card, Typography, Space, Button, Badge, Timeline, Input, Divider } from 'antd';
import { Ticket, WorkflowStep, SLAInfo } from '@/app/lib/api-config';
import { TicketRatingSection } from '@/components/business/TicketRatingSection';
import { useAuthStore } from '@/lib/store/auth-store';

const { Paragraph } = Typography;
const { TextArea } = Input;

interface TicketOverviewTabProps {
  ticket: Ticket;
  isEditing: boolean;
  editedTicket: Ticket;
  onEditedTicketChange: (updates: Partial<Ticket>) => void;
  workflowSteps: WorkflowStep[];
  slaInfo: SLAInfo | null;
  canApprove: boolean;
  canEdit: boolean;
  onApprove?: () => void;
  onReject?: () => void;
  onAssign?: (assignee: string) => void;
  onSmartAssignClick: () => void;
  assigneeInput: string;
  onAssigneeInputChange: (value: string) => void;
  onManualAssign: () => void;
  formatDateTime: (dateString: string) => string;
}

export const TicketOverviewTab: React.FC<TicketOverviewTabProps> = ({
  ticket,
  isEditing,
  editedTicket,
  onEditedTicketChange,
  workflowSteps,
  slaInfo,
  canApprove,
  canEdit,
  onApprove,
  onReject,
  onAssign,
  onSmartAssignClick,
  assigneeInput,
  onAssigneeInputChange,
  onManualAssign,
  formatDateTime,
}) => {
  const { user } = useAuthStore();

  return (
    <div className='p-6'>
      <div className='grid grid-cols-1 lg:grid-cols-3 gap-6'>
        {/* 左侧：详细信息 */}
        <div className='lg:col-span-2 space-y-6'>
          {/* 问题描述 */}
          <Card title='问题描述' className='shadow-sm'>
            {isEditing ? (
              <TextArea
                value={editedTicket.description}
                onChange={e => onEditedTicketChange({ description: e.target.value })}
                rows={4}
              />
            ) : (
              <Paragraph className='whitespace-pre-wrap'>{ticket.description}</Paragraph>
            )}
          </Card>

          {/* 解决方案 */}
          {ticket.resolution && (
            <Card title='解决方案' className='shadow-sm'>
              {isEditing ? (
                <TextArea
                  value={editedTicket.resolution}
                  onChange={e => onEditedTicketChange({ resolution: e.target.value })}
                  rows={3}
                />
              ) : (
                <Paragraph className='whitespace-pre-wrap'>{ticket.resolution}</Paragraph>
              )}
            </Card>
          )}

          {/* 工作流程图 */}
          {workflowSteps.length > 0 && (
            <Card title='处理流程' className='shadow-sm'>
              <Timeline>
                {workflowSteps.map(step => (
                  <Timeline.Item
                    key={step.id}
                    color={
                      step.status === 'completed'
                        ? 'green'
                        : step.status === 'in_progress'
                          ? 'blue'
                          : 'gray'
                    }
                    dot={
                      step.status === 'completed' ? (
                        <CheckCircle className='w-4 h-4' />
                      ) : step.status === 'in_progress' ? (
                        <Clock className='w-4 h-4' />
                      ) : undefined
                    }
                  >
                    <div className='flex items-center justify-between'>
                      <div>
                        <Typography.Text strong>{step.step_name}</Typography.Text>
                        {step.assignee && (
                          <div className='text-sm text-gray-500'>负责人: {step.assignee.name}</div>
                        )}
                        {step.comments && (
                          <div className='text-sm text-gray-600 mt-1'>{step.comments}</div>
                        )}
                      </div>
                      <div className='text-right'>
                        <Badge
                          status={
                            step.status === 'completed'
                              ? 'success'
                              : step.status === 'in_progress'
                                ? 'processing'
                                : 'default'
                          }
                          text={
                            step.status === 'completed'
                              ? '已完成'
                              : step.status === 'in_progress'
                                ? '进行中'
                                : '待处理'
                          }
                        />
                        {step.completed_at && (
                          <div className='text-xs text-gray-500 mt-1'>
                            {formatDateTime(step.completed_at)}
                          </div>
                        )}
                      </div>
                    </div>
                  </Timeline.Item>
                ))}
              </Timeline>
            </Card>
          )}
        </div>

        {/* 右侧：操作和SLA */}
        <div className='space-y-6'>
          {/* 审批操作 */}
          {canApprove && ticket.status === '待审批' && (
            <Card title='审批操作' className='shadow-sm'>
              <Space orientation='vertical' className='w-full'>
                <Button
                  type='primary'
                  icon={<CheckCircle />}
                  onClick={onApprove}
                  className='w-full'
                >
                  批准
                </Button>
                <Button danger icon={<XCircle />} onClick={onReject} className='w-full'>
                  拒绝
                </Button>
              </Space>
            </Card>
          )}

          {/* 分配操作 */}
          <Card title='分配工单' className='shadow-sm'>
            <Space orientation='vertical' className='w-full' style={{ width: '100%' }}>
              <Button
                type='primary'
                icon={<Zap />}
                onClick={onSmartAssignClick}
                className='w-full'
                style={{ marginBottom: 8 }}
              >
                智能分配
              </Button>
              <Divider style={{ margin: '8px 0' }}>或手动分配</Divider>
              <Input
                value={assigneeInput}
                onChange={e => onAssigneeInputChange(e.target.value)}
                placeholder='输入负责人姓名'
              />
              <Button onClick={onManualAssign} disabled={!assigneeInput.trim()} className='w-full'>
                手动分配
              </Button>
            </Space>
          </Card>

          {/* 评分信息 */}
          <TicketRatingSection
            ticketId={ticket.id}
            ticketStatus={ticket.status}
            requesterId={ticket.requester_id}
            canRate={canEdit}
            onRatingSubmitted={newRating => {
              console.log('Rating submitted:', newRating);
            }}
          />

          {/* SLA信息 */}
          {slaInfo && (
            <Card title='SLA信息' className='shadow-sm'>
              <div className='space-y-3'>
                <div>
                  <Typography.Text type='secondary' className='text-sm'>
                    SLA名称
                  </Typography.Text>
                  <div className='font-medium'>{slaInfo.sla_name}</div>
                </div>
                <div>
                  <Typography.Text type='secondary' className='text-sm'>
                    响应时间
                  </Typography.Text>
                  <div className='font-medium'>{slaInfo.response_time} 分钟</div>
                </div>
                <div>
                  <Typography.Text type='secondary' className='text-sm'>
                    解决时间
                  </Typography.Text>
                  <div className='font-medium'>{slaInfo.resolution_time} 分钟</div>
                </div>
                <div>
                  <Typography.Text type='secondary' className='text-sm'>
                    到期时间
                  </Typography.Text>
                  <div className='font-medium'>{formatDateTime(slaInfo.due_time)}</div>
                </div>
                <div>
                  <Typography.Text type='secondary' className='text-sm'>
                    状态
                  </Typography.Text>
                  <Badge
                    status={
                      slaInfo.status === 'active'
                        ? 'processing'
                        : slaInfo.status === 'breached'
                          ? 'error'
                          : 'success'
                    }
                    text={
                      slaInfo.status === 'active'
                        ? '进行中'
                        : slaInfo.status === 'breached'
                          ? '已违反'
                          : '已完成'
                    }
                  />
                </div>
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
};
