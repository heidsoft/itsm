'use client';

import React, { useState } from 'react';
import {
  Button,
  Dropdown,
  Modal,
  Form,
  Input,
  Select,
  Upload,
  message,
  Space,
  Tag,
  Steps,
  Timeline,
  Avatar,
  Tooltip,
} from 'antd';
import {
  CheckOutlined,
  CloseOutlined,
  RollbackOutlined,
  SendOutlined,
  MailOutlined,
  ArrowUpOutlined,
  UserSwitchOutlined,
  MoreOutlined,
  FileTextOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import {
  TicketWorkflowAction,
  TicketWorkflowState,
  ApprovalStatus,
  AcceptTicketRequest,
  RejectTicketRequest,
  WithdrawTicketRequest,
  ForwardTicketRequest,
  CCTicketRequest,
  ApproveTicketRequest,
} from '@/types/ticket-workflow';
import { Ticket } from '@/types/ticket';

const { TextArea } = Input;
const { Option } = Select;

interface TicketWorkflowActionsProps {
  ticket: Ticket;
  workflowState: TicketWorkflowState;
  onAction: (action: TicketWorkflowAction, data: any) => Promise<void>;
  onRefresh: () => void;
}

export const TicketWorkflowActions: React.FC<TicketWorkflowActionsProps> = ({
  ticket,
  workflowState,
  onAction,
  onRefresh,
}) => {
  const [modalVisible, setModalVisible] = useState(false);
  const [currentAction, setCurrentAction] = useState<TicketWorkflowAction | null>(null);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  // 打开操作模态框
  const openActionModal = (action: TicketWorkflowAction) => {
    setCurrentAction(action);
    setModalVisible(true);
    form.resetFields();
  };

  // 处理操作提交
  const handleSubmit = async () => {
    if (!currentAction) return;

    try {
      const values = await form.validateFields();
      setLoading(true);

      await onAction(currentAction, {
        ticketId: ticket.id,
        ...values,
      });

      message.success('操作成功');
      setModalVisible(false);
      onRefresh();
    } catch (error: any) {
      message.error(error.message || '操作失败');
    } finally {
      setLoading(false);
    }
  };

  // 快速操作（无需表单）
  const handleQuickAction = async (action: TicketWorkflowAction) => {
    Modal.confirm({
      title: `确认${getActionName(action)}`,
      content: `确定要${getActionName(action)}此工单吗？`,
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        try {
          await onAction(action, { ticketId: ticket.id });
          message.success('操作成功');
          onRefresh();
        } catch (error: any) {
          message.error(error.message || '操作失败');
        }
      },
    });
  };

  // 获取操作名称
  const getActionName = (action: TicketWorkflowAction): string => {
    const names: Record<TicketWorkflowAction, string> = {
      [TicketWorkflowAction.ACCEPT]: '接单',
      [TicketWorkflowAction.REJECT]: '驳回',
      [TicketWorkflowAction.WITHDRAW]: '撤回',
      [TicketWorkflowAction.FORWARD]: '转发',
      [TicketWorkflowAction.CC]: '抄送',
      [TicketWorkflowAction.ESCALATE]: '升级',
      [TicketWorkflowAction.APPROVE]: '审批通过',
      [TicketWorkflowAction.APPROVE_REJECT]: '审批拒绝',
      [TicketWorkflowAction.DELEGATE]: '委派',
      [TicketWorkflowAction.RESOLVE]: '解决',
      [TicketWorkflowAction.CLOSE]: '关闭',
      [TicketWorkflowAction.REOPEN]: '重开',
    };
    return names[action] || action;
  };

  // 渲染操作表单
  const renderActionForm = () => {
    switch (currentAction) {
      case TicketWorkflowAction.ACCEPT:
        return (
          <Form form={form} layout="vertical">
            <Form.Item label="备注" name="comment">
              <TextArea rows={3} placeholder="可选：添加接单备注" />
            </Form.Item>
          </Form>
        );

      case TicketWorkflowAction.REJECT:
        return (
          <Form form={form} layout="vertical">
            <Form.Item
              label="驳回原因"
              name="reason"
              rules={[{ required: true, message: '请输入驳回原因' }]}
            >
              <Select placeholder="选择驳回原因">
                <Option value="信息不完整">信息不完整</Option>
                <Option value="不符合要求">不符合要求</Option>
                <Option value="需要补充资料">需要补充资料</Option>
                <Option value="其他">其他</Option>
              </Select>
            </Form.Item>
            <Form.Item
              label="详细说明"
              name="comment"
              rules={[{ required: true, message: '请输入详细说明' }]}
            >
              <TextArea rows={4} placeholder="请详细说明驳回原因" />
            </Form.Item>
          </Form>
        );

      case TicketWorkflowAction.WITHDRAW:
        return (
          <Form form={form} layout="vertical">
            <Form.Item
              label="撤回原因"
              name="reason"
              rules={[{ required: true, message: '请输入撤回原因' }]}
            >
              <TextArea rows={3} placeholder="请说明撤回原因" />
            </Form.Item>
          </Form>
        );

      case TicketWorkflowAction.FORWARD:
        return (
          <Form form={form} layout="vertical">
            <Form.Item
              label="转发给"
              name="toUserId"
              rules={[{ required: true, message: '请选择接收人' }]}
            >
              <Select placeholder="选择接收人" showSearch>
                {/* TODO: 从API加载用户列表 */}
                <Option value={1}>张三</Option>
                <Option value={2}>李四</Option>
                <Option value={3}>王五</Option>
              </Select>
            </Form.Item>
            <Form.Item
              label="转发说明"
              name="comment"
              rules={[{ required: true, message: '请输入转发说明' }]}
            >
              <TextArea rows={3} placeholder="请说明转发原因" />
            </Form.Item>
            <Form.Item
              label="转移所有权"
              name="transferOwnership"
              valuePropName="checked"
            >
              <Select defaultValue={false}>
                <Option value={true}>是（转移后对方成为负责人）</Option>
                <Option value={false}>否（仅转发，不转移负责人）</Option>
              </Select>
            </Form.Item>
          </Form>
        );

      case TicketWorkflowAction.CC:
        return (
          <Form form={form} layout="vertical">
            <Form.Item
              label="抄送给"
              name="ccUsers"
              rules={[{ required: true, message: '请选择抄送人' }]}
            >
              <Select mode="multiple" placeholder="选择抄送人" showSearch>
                {/* TODO: 从API加载用户列表 */}
                <Option value={1}>张三</Option>
                <Option value={2}>李四</Option>
                <Option value={3}>王五</Option>
                <Option value={4}>赵六</Option>
              </Select>
            </Form.Item>
            <Form.Item label="抄送说明" name="comment">
              <TextArea rows={3} placeholder="可选：添加抄送说明" />
            </Form.Item>
          </Form>
        );

      case TicketWorkflowAction.APPROVE:
      case TicketWorkflowAction.APPROVE_REJECT:
        return (
          <Form form={form} layout="vertical">
            <Form.Item
              label="审批意见"
              name="comment"
              rules={[{ required: true, message: '请输入审批意见' }]}
            >
              <TextArea rows={4} placeholder="请输入审批意见" />
            </Form.Item>
            <Form.Item label="附件" name="attachments">
              <Upload>
                <Button>上传附件</Button>
              </Upload>
            </Form.Item>
          </Form>
        );

      case TicketWorkflowAction.RESOLVE:
        return (
          <Form form={form} layout="vertical">
            <Form.Item
              label="解决方案"
              name="resolution"
              rules={[{ required: true, message: '请输入解决方案' }]}
            >
              <TextArea rows={5} placeholder="请详细描述解决方案" />
            </Form.Item>
            <Form.Item label="解决类型" name="resolutionCategory">
              <Select placeholder="选择解决类型">
                <Option value="配置更改">配置更改</Option>
                <Option value="软件修复">软件修复</Option>
                <Option value="硬件更换">硬件更换</Option>
                <Option value="用户培训">用户培训</Option>
                <Option value="其他">其他</Option>
              </Select>
            </Form.Item>
            <Form.Item label="工作笔记" name="workNotes">
              <TextArea rows={3} placeholder="添加内部工作笔记（用户不可见）" />
            </Form.Item>
          </Form>
        );

      case TicketWorkflowAction.CLOSE:
        return (
          <Form form={form} layout="vertical">
            <Form.Item label="关闭原因" name="closeReason">
              <Select placeholder="选择关闭原因">
                <Option value="已解决">已解决</Option>
                <Option value="重复工单">重复工单</Option>
                <Option value="无效工单">无效工单</Option>
                <Option value="用户取消">用户取消</Option>
                <Option value="其他">其他</Option>
              </Select>
            </Form.Item>
            <Form.Item label="关闭说明" name="closeNotes">
              <TextArea rows={3} placeholder="可选：添加关闭说明" />
            </Form.Item>
          </Form>
        );

      case TicketWorkflowAction.REOPEN:
        return (
          <Form form={form} layout="vertical">
            <Form.Item
              label="重开原因"
              name="reason"
              rules={[{ required: true, message: '请输入重开原因' }]}
            >
              <TextArea rows={3} placeholder="请说明重开原因" />
            </Form.Item>
          </Form>
        );

      default:
        return null;
    }
  };

  // 主要操作按钮
  const mainActions = [];
  if (workflowState.canAccept) {
    mainActions.push(
      <Button
        key="accept"
        type="primary"
        icon={<CheckOutlined />}
        onClick={() => openActionModal(TicketWorkflowAction.ACCEPT)}
      >
        接单
      </Button>
    );
  }

  if (workflowState.canApprove) {
    mainActions.push(
      <Space key="approval">
        <Button
          type="primary"
          icon={<CheckOutlined />}
          onClick={() => openActionModal(TicketWorkflowAction.APPROVE)}
        >
          审批通过
        </Button>
        <Button
          danger
          icon={<CloseOutlined />}
          onClick={() => openActionModal(TicketWorkflowAction.APPROVE_REJECT)}
        >
          审批拒绝
        </Button>
      </Space>
    );
  }

  if (workflowState.canResolve) {
    mainActions.push(
      <Button
        key="resolve"
        type="primary"
        icon={<CheckOutlined />}
        onClick={() => openActionModal(TicketWorkflowAction.RESOLVE)}
      >
        解决
      </Button>
    );
  }

  if (workflowState.canClose) {
    mainActions.push(
      <Button
        key="close"
        type="primary"
        onClick={() => openActionModal(TicketWorkflowAction.CLOSE)}
      >
        关闭
      </Button>
    );
  }

  // 更多操作菜单
  const moreActions: MenuProps['items'] = [];
  if (workflowState.canReject) {
    moreActions.push({
      key: 'reject',
      label: '驳回',
      icon: <CloseOutlined />,
      onClick: () => openActionModal(TicketWorkflowAction.REJECT),
    });
  }

  if (workflowState.canWithdraw) {
    moreActions.push({
      key: 'withdraw',
      label: '撤回',
      icon: <RollbackOutlined />,
      onClick: () => openActionModal(TicketWorkflowAction.WITHDRAW),
    });
  }

  if (workflowState.canForward) {
    moreActions.push({
      key: 'forward',
      label: '转发',
      icon: <SendOutlined />,
      onClick: () => openActionModal(TicketWorkflowAction.FORWARD),
    });
  }

  if (workflowState.canCC) {
    moreActions.push({
      key: 'cc',
      label: '抄送',
      icon: <MailOutlined />,
      onClick: () => openActionModal(TicketWorkflowAction.CC),
    });
  }

  moreActions.push(
    {
      type: 'divider',
    },
    {
      key: 'escalate',
      label: '升级',
      icon: <ArrowUpOutlined />,
      onClick: () => openActionModal(TicketWorkflowAction.ESCALATE),
    },
    {
      key: 'reopen',
      label: '重开',
      icon: <RollbackOutlined />,
      onClick: () => openActionModal(TicketWorkflowAction.REOPEN),
    }
  );

  return (
    <>
      <div className="flex items-center space-x-2">
        {mainActions}
        {moreActions.length > 0 && (
          <Dropdown menu={{ items: moreActions }} placement="bottomRight">
            <Button icon={<MoreOutlined />}>更多操作</Button>
          </Dropdown>
        )}
      </div>

      {/* 操作模态框 */}
      <Modal
        title={currentAction ? getActionName(currentAction) : ''}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={loading}
        okText="确认"
        cancelText="取消"
        width={600}
      >
        {renderActionForm()}
      </Modal>
    </>
  );
};

// 审批进度组件
export const ApprovalProgress: React.FC<{
  workflowState: TicketWorkflowState;
}> = ({ workflowState }) => {
  if (!workflowState.approvalStatus) return null;

  const { approvalStatus, currentApprovalLevel, totalApprovalLevels, completedApprovals } =
    workflowState;

  return (
    <div className="bg-white p-4 rounded-lg shadow">
      <h3 className="font-bold mb-4">审批进度</h3>

      {/* 审批状态 */}
      <div className="mb-4">
        <Tag
          color={
            approvalStatus === ApprovalStatus.APPROVED
              ? 'success'
              : approvalStatus === ApprovalStatus.REJECTED
              ? 'error'
              : approvalStatus === ApprovalStatus.IN_PROGRESS
              ? 'processing'
              : 'default'
          }
        >
          {approvalStatus === ApprovalStatus.APPROVED && '已通过'}
          {approvalStatus === ApprovalStatus.REJECTED && '已拒绝'}
          {approvalStatus === ApprovalStatus.IN_PROGRESS && '审批中'}
          {approvalStatus === ApprovalStatus.PENDING && '待审批'}
          {approvalStatus === ApprovalStatus.CANCELLED && '已取消'}
        </Tag>
        {currentApprovalLevel && totalApprovalLevels && (
          <span className="ml-2 text-gray-500">
            第 {currentApprovalLevel} / {totalApprovalLevels} 级
          </span>
        )}
      </div>

      {/* 待审批人 */}
      {workflowState.pendingApprovers && workflowState.pendingApprovers.length > 0 && (
        <div className="mb-4">
          <div className="text-sm text-gray-500 mb-2">当前审批人：</div>
          <Space wrap>
            {workflowState.pendingApprovers.map((approver) => (
              <Tag key={approver.id} icon={<ClockCircleOutlined />} color="blue">
                {approver.fullName}
              </Tag>
            ))}
          </Space>
        </div>
      )}

      {/* 审批历史 */}
      {completedApprovals && completedApprovals.length > 0 && (
        <div>
          <div className="text-sm text-gray-500 mb-2">审批历史：</div>
          <Timeline
            items={completedApprovals.map((approval) => ({
              color:
                approval.status === ApprovalStatus.APPROVED
                  ? 'green'
                  : approval.status === ApprovalStatus.REJECTED
                  ? 'red'
                  : 'gray',
              children: (
                <div>
                  <div className="flex items-center space-x-2">
                    <Avatar size="small" src={approval.approver.avatar}>
                      {approval.approver.fullName.charAt(0)}
                    </Avatar>
                    <span className="font-medium">{approval.approver.fullName}</span>
                    <Tag
                      color={
                        approval.status === ApprovalStatus.APPROVED
                          ? 'success'
                          : approval.status === ApprovalStatus.REJECTED
                          ? 'error'
                          : 'default'
                      }
                    >
                      {approval.action === 'approve' && '通过'}
                      {approval.action === 'reject' && '拒绝'}
                      {approval.action === 'delegate' && '委派'}
                    </Tag>
                  </div>
                  <div className="text-sm text-gray-500 mt-1">{approval.levelName}</div>
                  {approval.comment && (
                    <div className="text-sm mt-1 text-gray-600">
                      <FileTextOutlined className="mr-1" />
                      {approval.comment}
                    </div>
                  )}
                  <div className="text-xs text-gray-400 mt-1">
                    {approval.processedAt &&
                      new Date(approval.processedAt).toLocaleString('zh-CN')}
                  </div>
                </div>
              ),
            }))}
          />
        </div>
      )}
    </div>
  );
};

