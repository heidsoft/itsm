'use client';

import {
  ArrowLeft,
  PlayCircle,
  Flag,
  ClipboardCheck,
  MessageSquare,
  GitMerge,
  AlertTriangle,
} from 'lucide-react';

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import {
  changeService,
  Change,
  ChangeComment,
  ChangeApproval,
  RelatedTicket,
  AffectedCI,
} from '@/lib/services/change-service';

interface ApprovalStatusBadgeProps {
  status: '待审批' | '已批准' | '已拒绝';
}

const ApprovalStatusBadge: React.FC<ApprovalStatusBadgeProps> = ({ status }) => {
  const colors = {
    待审批: 'bg-yellow-100 text-yellow-800',
    已批准: 'bg-green-100 text-green-800',
    已拒绝: 'bg-red-100 text-red-800',
  };
  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}>{status}</span>
  );
};

const ChangeDetailPage = () => {
  const params = useParams();
  const router = useRouter();
  const changeId = params.changeId as string;
  const [change, setChange] = useState<Change | null>(null);
  const [loading, setLoading] = useState(true);
  const [newComment, setNewComment] = useState('');

  // 加载变更详情
  useEffect(() => {
    const loadChangeDetail = async () => {
      if (!changeId) return;

      try {
        setLoading(true);
        // 从URL中提取数字ID
        const numericId = parseInt(changeId.replace('CHG-', ''));
        if (isNaN(numericId)) {
          console.error('无效的变更ID');
          return;
        }

        const changeData = await changeService.getChange(numericId);
        setChange(changeData);
      } catch (error) {
        console.error('加载变更详情失败:', error);
        // 可以在这里添加错误处理UI
      } finally {
        setLoading(false);
      }
    };

    loadChangeDetail();
  }, [changeId]);

  const updateChangeStatus = async (newStatus: string, logMessage: string) => {
    if (!change) return;

    try {
      const numericId = parseInt(changeId.replace('CHG-', ''));
      if (isNaN(numericId)) {
        alert('无效的变更ID');
        return;
      }

      // 调用API更新状态
      const updatedChange = await changeService.updateChange(numericId, {
        status: newStatus as
          | 'draft'
          | 'pending'
          | 'approved'
          | 'rejected'
          | 'implementing'
          | 'completed'
          | 'cancelled',
      });

      // 更新本地状态
      const existingLogs = change.logs || [];
      setChange({
        ...change,
        ...updatedChange,
        logs: [...existingLogs, `[${new Date().toLocaleString()}] ${logMessage}`],
      } as Change);

      alert(`变更状态已更新为: ${newStatus}`);
    } catch (error) {
      console.error('更新变更状态失败:', error);
      const errorMessage = error instanceof Error ? error.message : '未知错误';
      alert(`更新状态失败: ${errorMessage}`);
    }
  };

  const handleApprove = (index: number) => {
    if (!change?.approvals) return;
    const updatedApprovals = [...change.approvals];
    updatedApprovals[index] = {
      ...updatedApprovals[index],
      status: '已批准' as const,
      approver: '当前用户',
      comment: '已批准',
    };
    setChange({ ...change, approvals: updatedApprovals });
    alert(`审批人 ${updatedApprovals[index].role} 已批准变更 ${changeId}！`);
    // 检查是否所有审批都已完成，如果是，则将变更状态更新为“已批准”
    if (updatedApprovals.every(app => app.status === '已批准')) {
      updateChangeStatus('approved', '所有审批已完成，变更已批准。');
    }
  };

  const handleReject = (index: number) => {
    if (!change?.approvals) return;

    const comment = prompt('请输入拒绝理由：');
    if (comment) {
      const updatedApprovals = [...(change.approvals || [])];
      updatedApprovals[index] = {
        ...updatedApprovals[index],
        status: '已拒绝' as const,
        approver: '当前用户',
        comment: comment,
      };
      setChange({ ...change, approvals: updatedApprovals });
      alert(`审批人 ${updatedApprovals[index].role} 已拒绝变更 ${changeId}！`);
      updateChangeStatus('rejected', `变更被拒绝。理由：${comment}`);
      // 实际应用中会更新后端状态
    }
  };

  const handleStartImplementation = () => {
    if (!change || change.status !== 'approved') {
      alert('变更未处于已批准状态，无法开始实施。');
      return;
    }
    updateChangeStatus('implementing', '变更实施已开始。');
  };

  const handleCompleteImplementation = () => {
    if (!change || change.status !== 'implementing') {
      alert('变更未处于实施中状态，无法完成实施。');
      return;
    }
    const implementationNotes = prompt('请输入实施结果描述：');
    if (implementationNotes) {
      updateChangeStatus('completed', `变更实施已完成。结果：${implementationNotes}`);
    }
  };

  const handleCompleteReview = () => {
    if (!change || change.status !== 'completed') {
      alert('变更未处于已完成状态，无法进行评审。');
      return;
    }
    const reviewNotes = prompt('请输入评审结果描述：');
    if (reviewNotes) {
      updateChangeStatus('completed', `变更评审已完成。结果：${reviewNotes}`);
    }
  };

  const handleAddComment = () => {
    if (!change || !newComment.trim()) return;

    const existingComments = change.comments || [];
    const updatedChange: Change = {
      ...change,
      comments: [
        ...existingComments,
        {
          author: '当前用户',
          timestamp: new Date().toLocaleString(),
          text: newComment.trim(),
        },
      ],
    };
    setChange(updatedChange);
    setNewComment('');
    // 在真实应用中，这里会调用API保存评论
  };

  if (!change) {
    return <div className='p-10'>变更不存在或加载失败。</div>;
  }

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <button
          onClick={() => router.back()}
          className='flex items-center text-blue-600 hover:underline mb-4'
        >
          <ArrowLeft className='w-5 h-5 mr-2' />
          返回变更列表
        </button>
        <div className='flex justify-between items-start'>
          <div>
            <h2 className='text-4xl font-bold text-gray-800'>变更：{change.title}</h2>
            <p className='text-gray-500 mt-1'>变更ID: {changeId}</p>
          </div>
          <div className='flex space-x-4'>
            {change.status === 'approved' && (
              <button
                onClick={handleStartImplementation}
                className='flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors'
              >
                <PlayCircle className='w-5 h-5 mr-2' />
                开始实施
              </button>
            )}
            {change.status === 'implementing' && (
              <button
                onClick={handleCompleteImplementation}
                className='flex items-center bg-green-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-green-700 transition-colors'
              >
                <Flag className='w-5 h-5 mr-2' />
                完成实施
              </button>
            )}
            {change.status === 'completed' && (
              <button
                onClick={handleCompleteReview}
                className='flex items-center bg-purple-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-purple-700 transition-colors'
              >
                <ClipboardCheck className='w-5 h-5 mr-2' />
                完成评审
              </button>
            )}
          </div>
        </div>
      </header>

      <div className='grid grid-cols-1 lg:grid-cols-3 gap-8'>
        {/* 左侧：变更详情 */}
        <div className='lg:col-span-2 bg-white p-8 rounded-lg shadow-md'>
          <h3 className='text-xl font-semibold text-gray-700 mb-4'>变更描述</h3>
          <p className='text-gray-600 mb-8'>{change.description}</p>

          <h3 className='text-xl font-semibold text-gray-700 mb-4'>变更理由</h3>
          <p className='text-gray-600 mb-8'>{change.justification}</p>

          <h3 className='text-xl font-semibold text-gray-700 mb-4'>变更类型</h3>
          <p className='text-gray-600 mb-8'>
            <span className='font-semibold'>{change.type}</span>
          </p>

          <h3 className='text-xl font-semibold text-gray-700 mb-4'>影响与风险</h3>
          <p className='text-gray-600 mb-2'>
            <span className='font-semibold'>影响范围:</span> {change.impactScope}
          </p>
          <p className='text-gray-600 mb-8'>
            <span className='font-semibold'>风险等级:</span> {change.riskLevel}
          </p>

          <h3 className='text-xl font-semibold text-gray-700 mb-4'>实施计划</h3>
          <p className='text-gray-600 mb-8 whitespace-pre-wrap'>{change.implementationPlan}</p>

          <h3 className='text-xl font-semibold text-gray-700 mb-4'>回滚计划</h3>
          <p className='text-gray-600 mb-8 whitespace-pre-wrap'>{change.rollbackPlan}</p>

          <h3 className='text-xl font-semibold text-gray-700 mb-4'>变更日志</h3>
          <div className='space-y-4'>
            {change.logs && change.logs.length > 0 ? (
              change.logs.map((log, i) => (
                <p key={i} className='text-sm text-gray-500 font-mono'>
                  {log}
                </p>
              ))
            ) : (
              <p className='text-sm text-gray-500'>暂无日志</p>
            )}
          </div>

          {/* 评论/备注区域 */}
          <div className='mt-8 pt-8 border-t border-gray-200'>
            <h3 className='text-xl font-semibold text-gray-700 mb-4 flex items-center'>
              <MessageSquare className='w-5 h-5 mr-2' /> 内部评论/备注
            </h3>
            <div className='space-y-4 mb-4'>
              {change.comments && change.comments.length > 0 ? (
                change.comments.map((comment: ChangeComment, i: number) => (
                  <div key={i} className='bg-gray-50 p-3 rounded-lg'>
                    <p className='text-sm font-semibold text-gray-800'>
                      {comment.author}{' '}
                      <span className='text-gray-500 font-normal'>于 {comment.timestamp}</span>
                    </p>
                    <p className='text-gray-700 mt-1'>{comment.text}</p>
                  </div>
                ))
              ) : (
                <p className='text-sm text-gray-500'>暂无评论。</p>
              )}
            </div>
            <textarea
              className='w-full p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500'
              rows={3}
              placeholder='添加内部评论或备注...'
              value={newComment}
              onChange={e => setNewComment(e.target.value)}
            ></textarea>
            <button
              onClick={handleAddComment}
              className='mt-2 bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors'
            >
              添加评论
            </button>
          </div>
        </div>

        {/* 右侧：元数据和联动功能 */}
        <div className='space-y-8'>
          <div className='bg-white p-6 rounded-lg shadow-md'>
            <h3 className='text-xl font-semibold text-gray-700 mb-4'>变更信息</h3>
            <div className='space-y-3 text-sm'>
              <div className='flex justify-between'>
                <span>状态:</span>
                <span className='font-semibold'>{change.status}</span>
              </div>
              <div className='flex justify-between'>
                <span>优先级:</span>
                <span className='font-semibold'>{change.priority}</span>
              </div>
              <div className='flex justify-between'>
                <span>负责人:</span>
                <span className='font-semibold'>
                  {typeof change.assignee === 'string'
                    ? change.assignee
                    : change.assignee?.name || change.assigneeName || '未分配'}
                </span>
              </div>
              <div className='flex justify-between'>
                <span>创建时间:</span>
                <span className='font-semibold'>{change.createdAt}</span>
              </div>
              <div className='flex justify-between'>
                <span>计划开始:</span>
                <span className='font-semibold'>{change.plannedStartDate}</span>
              </div>
              <div className='flex justify-between'>
                <span>计划结束:</span>
                <span className='font-semibold'>{change.plannedEndDate}</span>
              </div>
            </div>
          </div>

          <div className='bg-white p-6 rounded-lg shadow-md'>
            <h3 className='text-xl font-semibold text-gray-700 mb-4'>审批流程</h3>
            <div className='space-y-3'>
              {change.approvals && change.approvals.length > 0 ? (
                change.approvals.map((approval: ChangeApproval, i: number) => (
                  <div key={i} className='p-3 bg-gray-50 rounded-lg'>
                    <div className='flex justify-between items-center mb-1'>
                      <span className='font-semibold text-gray-700'>{approval.role}</span>
                      <ApprovalStatusBadge status={approval.status} />
                    </div>
                    {approval.approver && (
                      <p className='text-sm text-gray-600'>审批人: {approval.approver}</p>
                    )}
                    {approval.comment && (
                      <p className='text-sm text-gray-500 italic'>&quot;{approval.comment}&quot;</p>
                    )}
                    {approval.status === '待审批' && (
                      <div className='mt-2 flex space-x-2'>
                        <button
                          onClick={() => handleApprove(i)}
                          className='flex-1 bg-green-500 text-white text-xs py-1 rounded hover:bg-green-600'
                        >
                          批准
                        </button>
                        <button
                          onClick={() => handleReject(i)}
                          className='flex-1 bg-red-500 text-white text-xs py-1 rounded hover:bg-red-600'
                        >
                          拒绝
                        </button>
                      </div>
                    )}
                  </div>
                ))
              ) : (
                <p className='text-sm text-gray-500'>暂无审批流程</p>
              )}
            </div>
          </div>

          <div className='bg-white p-6 rounded-lg shadow-md'>
            <h3 className='text-xl font-semibold text-gray-700 mb-4 flex items-center'>
              <GitMerge className='w-5 h-5 mr-2' /> 关联工单
            </h3>
            <div className='space-y-3'>
              {change.relatedTickets && change.relatedTickets.length > 0 ? (
                change.relatedTickets.map((ticket: RelatedTicket | string) => {
                  if (typeof ticket === 'string') {
                    return (
                      <div key={ticket} className='p-3 bg-blue-50 rounded-lg'>
                        <p className='font-semibold text-blue-700'>{ticket}</p>
                      </div>
                    );
                  }
                  return (
                    <Link
                      key={typeof ticket.id === 'string' ? ticket.id : String(ticket.id)}
                      href={`/${
                        ticket.type === '事件'
                          ? 'incidents'
                          : ticket.type === '问题'
                          ? 'problems'
                          : 'changes'
                      }/${ticket.id}`}
                      className='block p-3 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors'
                    >
                      <p className='font-semibold text-blue-700'>
                        {ticket.id} ({ticket.type})
                      </p>
                      <p className='text-sm text-gray-600'>{ticket.title}</p>
                      <span className='text-xs text-gray-500'>状态: {ticket.status}</span>
                    </Link>
                  );
                })
              ) : (
                <p className='text-sm text-gray-500'>无关联工单。</p>
              )}
            </div>
          </div>

          <div className='bg-white p-6 rounded-lg shadow-md'>
            <h3 className='text-xl font-semibold text-gray-700 mb-4 flex items-center'>
              <AlertTriangle className='w-5 h-5 mr-2' /> 受影响的配置项
            </h3>
            <div className='space-y-3'>
              {change.affectedCIs && change.affectedCIs.length > 0 ? (
                change.affectedCIs.map((ci: AffectedCI | string) => {
                  if (typeof ci === 'string') {
                    return (
                      <div key={ci} className='p-3 bg-blue-50 rounded-lg'>
                        <p className='font-semibold text-blue-700'>{ci}</p>
                      </div>
                    );
                  }
                  return (
                    <Link
                      key={typeof ci.id === 'string' ? ci.id : String(ci.id)}
                      href={`/cmdb/${ci.id}`}
                      className='block p-3 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors'
                    >
                      <p className='font-semibold text-blue-700'>
                        {ci.name} ({ci.type})
                      </p>
                      <span className='text-xs text-gray-500'>CI ID: {ci.id}</span>
                    </Link>
                  );
                })
              ) : (
                <p className='text-sm text-gray-500'>无受影响的配置项。</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ChangeDetailPage;
