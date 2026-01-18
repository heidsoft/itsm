'use client';
// @ts-nocheck

import React, { useState, useEffect } from 'react';
import {
  FileText,
  Save,
  X,
} from 'lucide-react';

import {
  Button,
  Space,
  Typography,
  App,
  Tabs,
} from 'antd';
import { TicketApi } from '@/lib/api/ticket-api';
import { TicketNotificationApi, TicketNotification } from '@/lib/api/ticket-notification-api';
import { Ticket, Attachment, Comment, WorkflowStep, SLAInfo } from '@/lib/api-config';
import { TicketDetailHeader, TicketOverviewTab, TicketComments, TicketHistory } from '@/components/ticket/TicketDetail/index';
import { TicketAttachmentSection } from './TicketAttachmentSection';
import { TicketNotificationSection } from './TicketNotificationSection';
import { TicketSubtasks } from './TicketSubtasks';
import { TicketDependencyManager } from './TicketDependencyManager';
import { TicketMultiLevelApproval } from './TicketMultiLevelApproval';
import { TicketRootCauseAnalysis } from './TicketRootCauseAnalysis';
import { SmartAssignmentModal } from './SmartAssignmentModal';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

interface TicketDetailProps {
  ticket: Ticket;
  onApprove?: () => void;
  onReject?: () => void;
  onAssign?: (assignee: string) => void;
  onUpdate?: (updates: unknown) => void;
  canApprove?: boolean;
  canEdit?: boolean;
}

// 格式化时间
const formatDateTime = (dateString: string) => {
  return new Date(dateString).toLocaleString('zh-CN');
};

// 格式化文件大小
const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

export const TicketDetail: React.FC<TicketDetailProps> = ({
  ticket,
  onApprove,
  onReject,
  onAssign,
  onUpdate,
  canApprove = false,
  canEdit = false,
}) => {
  const { message: antMessage } = App.useApp();
  const [isEditing, setIsEditing] = useState(false);
  const [editedTicket, setEditedTicket] = useState<Ticket>(ticket);
  const [assigneeInput, setAssigneeInput] = useState('');
  const [activeTab, setActiveTab] = useState('overview');
  const [subtasks, setSubtasks] = useState<Ticket[]>([]);
  const [subtasksLoading, setSubtasksLoading] = useState(false);

  // 状态管理
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [comments, setComments] = useState<Comment[]>([]);
  const [workflowSteps, setWorkflowSteps] = useState<WorkflowStep[]>([]);
  const [slaInfo, setSlaInfo] = useState<SLAInfo | null>(null);
  const [ticketHistory, setTicketHistory] = useState<unknown[]>([]);
  const [notifications, setNotifications] = useState<TicketNotification[]>([]);
  const [loading, setLoading] = useState(false);
  const [showSmartAssignModal, setShowSmartAssignModal] = useState(false);

  // 获取工单相关数据
  useEffect(() => {
    fetchTicketData();
    fetchSubtasks();
  }, [ticket.id]);

  const fetchTicketData = async () => {
    setLoading(true);
    try {
      const [attachmentsRes, commentsRes, workflowRes, slaRes, historyRes, notificationsRes] =
        await Promise.allSettled([
          TicketApi.getTicketAttachments(ticket.id),
          TicketApi.getTicketComments(ticket.id),
          TicketApi.getTicketWorkflow(ticket.id),
          TicketApi.getTicketSLA(ticket.id),
          TicketApi.getTicketHistory(ticket.id),
          TicketNotificationApi.getTicketNotifications(ticket.id),
        ]);

      if (attachmentsRes.status === 'fulfilled') {
        const data = attachmentsRes.value as { attachments?: unknown[]; data?: unknown[] };
        setAttachments((data.attachments || data.data || []) as Attachment[]);
      }
      if (commentsRes.status === 'fulfilled') {
        const data = commentsRes.value as { comments?: unknown[]; data?: unknown[] };
        setComments((data.comments || data.data || []) as Comment[]);
      }
      if (workflowRes.status === 'fulfilled') {
        setWorkflowSteps((workflowRes.value as WorkflowStep[]) || []);
      }
      if (slaRes.status === 'fulfilled') {
        setSlaInfo((slaRes.value as SLAInfo) || null);
      }
      if (historyRes.status === 'fulfilled') {
        setTicketHistory(historyRes.value || []);
      }
      if (notificationsRes.status === 'fulfilled') {
        const data = notificationsRes.value as { notifications?: TicketNotification[]; data?: TicketNotification[] };
        setNotifications((data.notifications || data.data || []) as TicketNotification[]);
      }
    } catch (error) {
      console.error('获取工单数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchSubtasks = async () => {
    setSubtasksLoading(true);
    try {
      // 获取子任务逻辑
    } finally {
      setSubtasksLoading(false);
    }
  };

  const handleSave = () => {
    onUpdate?.(editedTicket);
    setIsEditing(false);
  };

  const handleCancel = () => {
    setEditedTicket(ticket);
    setIsEditing(false);
  };

  const handleAssign = () => {
    if (assigneeInput.trim()) {
      onAssign?.(assigneeInput.trim());
      setAssigneeInput('');
    }
  };

  // 智能分配成功回调
  const handleSmartAssignSuccess = async (userId: number) => {
    try {
      await TicketApi.assignTicket(ticket.id, { assignee_id: userId });
      antMessage.success('工单分配成功');
      if (onUpdate) {
        await onUpdate({ assignee_id: userId });
      }
      fetchTicketData();
    } catch (error) {
      console.error('Failed to assign ticket:', error);
      antMessage.error('分配失败');
    }
  };

  // 子任务处理函数
  const handleCreateSubtask = async (data: Partial<Ticket>) => {
    console.log('Create subtask:', data);
  };

  const handleUpdateSubtask = async (id: number, data: Partial<Ticket>) => {
    console.log('Update subtask:', id, data);
  };

  const handleDeleteSubtask = async (id: number) => {
    console.log('Delete subtask:', id);
  };

  const handleViewSubtask = (id: number) => {
    console.log('View subtask:', id);
  };

  return (
    <div className='max-w-7xl mx-auto bg-gray-50 min-h-screen'>
      {/* 头部 */}
      <TicketDetailHeader
        ticket={ticket}
        isEditing={isEditing}
        onEditToggle={() => setIsEditing(!isEditing)}
        canEdit={canEdit}
      />

      {/* 主要内容区域 */}
      <div className='px-6'>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          className='bg-white rounded-lg shadow-sm'
          type='card'
          items={[
            {
              key: 'overview',
              label: '基本信息',
              children: (
                <TicketOverviewTab
                  ticket={ticket}
                  isEditing={isEditing}
                  editedTicket={editedTicket}
                  onEditedTicketChange={(updates) =>
                    setEditedTicket({ ...editedTicket, ...updates })
                  }
                  workflowSteps={workflowSteps}
                  slaInfo={slaInfo}
                  canApprove={canApprove}
                  canEdit={canEdit}
                  onApprove={onApprove}
                  onReject={onReject}
                  onAssign={onAssign}
                  onSmartAssignClick={() => setShowSmartAssignModal(true)}
                  assigneeInput={assigneeInput}
                  onAssigneeInputChange={setAssigneeInput}
                  onManualAssign={handleAssign}
                  formatDateTime={formatDateTime}
                />
              ),
            },
            {
              key: 'attachments',
              label: '附件',
              children: (
                <div className='p-6'>
                  <TicketAttachmentSection
                    ticketId={ticket.id}
                    canUpload={canEdit}
                    canDelete={(attachment) => true}
                    onAttachmentUploaded={fetchTicketData}
                    onAttachmentDeleted={fetchTicketData}
                  />
                </div>
              ),
            },
            {
              key: 'comments',
              label: '评论',
              children: (
                <TicketComments
                  ticketId={ticket.id}
                  comments={comments}
                  onRefresh={fetchTicketData}
                  formatDateTime={formatDateTime}
                />
              ),
            },
            {
              key: 'notifications',
              label: '通知',
              children: (
                <div className='p-6'>
                  <TicketNotificationSection
                    ticketId={ticket.id}
                    canSend={canEdit}
                    onNotificationSent={fetchTicketData}
                  />
                </div>
              ),
            },
            {
              key: 'subtasks',
              label: '子任务',
              children: (
                <div className='p-6'>
                  <TicketSubtasks
                    parentTicket={ticket}
                    subtasks={subtasks as any}
                    loading={subtasksLoading}
                    onCreateSubtask={handleCreateSubtask}
                    onUpdateSubtask={handleUpdateSubtask}
                    onDeleteSubtask={handleDeleteSubtask}
                    onViewSubtask={handleViewSubtask}
                    canEdit={canEdit}
                  />
                </div>
              ),
            },
            {
              key: 'dependencies',
              label: '依赖关系',
              children: (
                <div className='p-6'>
                  <TicketDependencyManager
                    ticket={ticket}
                    canManage={canEdit}
                    onDependencyChange={fetchTicketData}
                  />
                </div>
              ),
            },
            {
              key: 'approval',
              label: '审批流程',
              children: (
                <div className='p-6'>
                  <TicketMultiLevelApproval
                    ticket={ticket}
                    canManage={canEdit}
                    onWorkflowChange={(workflowId) => {
                      console.log('Workflow changed:', workflowId);
                    }}
                  />
                </div>
              ),
            },
            {
              key: 'root-cause',
              label: '根因分析',
              children: (
                <div className='p-6'>
                  <TicketRootCauseAnalysis
                    ticketId={ticket.id}
                    autoAnalyze={false}
                    onAnalysisComplete={(report) => {
                      console.log('Root cause analysis completed:', report);
                    }}
                  />
                </div>
              ),
            },
            {
              key: 'history',
              label: '历史记录',
              children: (
                <TicketHistory
                  history={ticketHistory as any}
                  formatDateTime={formatDateTime}
                />
              ),
            },
          ]}
        />
      </div>

      {/* 智能分配模态框 */}
      <SmartAssignmentModal
        visible={showSmartAssignModal}
        ticketId={ticket.id}
        onCancel={() => setShowSmartAssignModal(false)}
        onSuccess={handleSmartAssignSuccess}
      />
    </div>
  );
};
