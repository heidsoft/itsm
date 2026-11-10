'use client';

import React, { useState, useCallback, useMemo } from 'react';
import { Button, Dropdown, Modal, Form, Select, message, Alert, Progress, Typography } from 'antd';
import { Trash2, Download, User as UserIcon, CheckCircle, Tag, Flag, MoreHorizontal } from 'lucide-react';
import type { MenuProps } from 'antd';
import type { Ticket, TicketStatus, TicketPriority } from '@/lib/api/types';
import { TicketAPI } from '@/lib/api/ticket-api';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { useUserListQuery } from '@/lib/hooks/useUserListQuery';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';
import { TicketStatusConfig, TicketPriorityConfig } from '@/constants/taxonomy';
const { Text } = Typography;

// 词表取自 @/constants/taxonomy（后端状态机的投影），不在组件里维护第二套清单。
const STATUS_OPTIONS = Object.entries(TicketStatusConfig).map(([value, config]) => ({
  value,
  label: config.label,
}));

const PRIORITY_OPTIONS = Object.entries(TicketPriorityConfig).map(([value, config]) => ({
  value,
  label: config.label,
}));

interface TicketBatchOperationsProps {
  selectedTickets: Ticket[];
  onOperationComplete?: () => void;
  onSelectionClear?: () => void;
}

interface BatchOperation {
  key: string;
  label: string;
  icon: React.ReactNode;
  danger?: boolean;
  description?: string;
  /** 与后端 RequirePermission 对齐的权限码；缺失时菜单项不出现。 */
  permission?: [resource: string, action: string];
}

const TicketBatchOperations: React.FC<TicketBatchOperationsProps> = ({
  selectedTickets,
  onOperationComplete,
  onSelectionClear,
}) => {
  const { hasPermission } = usePermissions();
  // 处理人下拉复用带缓存的用户列表查询，不再使用写死的示例姓名。
  const assignees = useUserListQuery({ pageSize: BACKEND_MAX_PAGE_SIZE, enabled: hasPermission('ticket', 'assign') });
  const assigneeOptions = useMemo(
    () =>
      (assignees.data?.items ?? []).map(user => ({
        value: user.id,
        label: `${user.name || user.username}${user.department ? ` (${user.department})` : ''}`,
      })),
    [assignees.data]
  );
  const [loading, setLoading] = useState(false);
  const [operationModal, setOperationModal] = useState<{
    visible: boolean;
    type: string;
    title: string;
  }>({
    visible: false,
    type: '',
    title: '',
  });
  const [form] = Form.useForm();
  const [operationProgress, setOperationProgress] = useState<{
    visible: boolean;
    current: number;
    total: number;
    status: string;
  }>({
    visible: false,
    current: 0,
    total: 0,
    status: '',
  });

  // 批量操作菜单
  const batchOperations: BatchOperation[] = useMemo(
    () => [
      {
        key: 'assign',
        label: 'Batch Assign',
        icon: <UserIcon />,
        description: 'Assign the selected work order to the designated handler',
        permission: ['ticket', 'assign'],
      },
      {
        key: 'update_status',
        label: 'Batch Update Status',
        icon: <Flag />,
        description: 'Update work order status in batches',
        permission: ['ticket', 'update'],
      },
      {
        key: 'add_tags',
        label: 'Batch Add Tags',
        icon: <Tag />,
        description: 'Add label to selected work order',
        permission: ['ticket_tag', 'create'],
      },
      {
        key: 'set_priority',
        label: 'Set priorities in batches',
        icon: <Flag />,
        description: 'Set work order priority in batches',
        permission: ['ticket', 'update'],
      },
      {
        key: 'export',
        label: 'Batch Export',
        icon: <Download />,
        description: 'Export selected work order data',
        permission: ['ticket', 'export'],
      },
      {
        key: 'divider1',
        label: '-',
        icon: null,
      },
      {
        key: 'delete',
        label: 'Batch Delete',
        icon: <Trash2 />,
        danger: true,
        description: 'Delete selected work order (unrecoverable)',
        permission: ['ticket', 'delete'],
      },
    ],
    []
  );

  // 定义操作参数类型
  type BatchOperationParams =
    | { assigneeId: number; comment?: string }
    | { status: string }
    | { tags: string[] }
    | { priority: string }
    | {};

  // 执行批量操作
  const executeBatchOperation = useCallback(
    async (operation: string, values: BatchOperationParams) => {
      setLoading(true);
      setOperationProgress({
        visible: true,
        current: 0,
        total: selectedTickets.length,
        status: "Preparing to execute...",
      });

      try {
        let successCount = 0;
        let failCount = 0;
        const errors: string[] = [];

        for (let i = 0; i < selectedTickets.length; i++) {
          const ticket = selectedTickets[i];

          setOperationProgress(prev => ({
            ...prev,
            current: i + 1,
            status: `Process work order ${ticket.ticketNumber}...`,
          }));

          try {
            switch (operation) {
              case 'assign':
                // 后端 dto.AssignTicketRequest 只绑定 assigneeId，无 reason/comment 字段
                const assignParams = values as { assigneeId: number };
                await TicketAPI.assignTicket(ticket.id, {
                  assigneeId: assignParams.assigneeId,
                });
                break;
              case 'update_status':
                const statusParams = values as { status: string };
                await TicketAPI.updateTicket(ticket.id, {
                  status: statusParams.status as TicketStatus,
                });
                break;
              case 'add_tags':
                const tagParams = values as { tags: string[] };
                await TicketAPI.addTicketTags(ticket.id, tagParams.tags);
                break;
              case 'set_priority':
                const priorityParams = values as { priority: string };
                await TicketAPI.updateTicket(ticket.id, {
                  priority: priorityParams.priority as TicketPriority,
                });
                break;
              case 'delete':
                await TicketAPI.deleteTicket(ticket.id);
                break;
              default:
                throw new Error(`Unsupported operation: ${operation}`);
            }
            successCount++;
          } catch (error) {
            failCount++;
            errors.push(
              `${ticket.ticketNumber}: ${error instanceof Error ? error.message : 'Operation failed'}`
            );
          }

          // 模拟处理延迟，避免请求过于频繁
          await new Promise(resolve => setTimeout(resolve, 100));
        }

        setOperationProgress(prev => ({
          ...prev,
          status: "Operation completed",
        }));

        // 显示操作结果
        if (failCount === 0) {
          message.success(`Batch operation successful! Processed in total ${successCount} work orders`);
        } else {
          message.warning(`Operation completed! success ${successCount} , failed ${failCount} items`);
        }

        setTimeout(() => {
          setOperationProgress({ visible: false, current: 0, total: 0, status: '' });
          setOperationModal({ visible: false, type: '', title: '' });
          form.resetFields();
          onOperationComplete?.();
        }, 1500);
      } catch (error) {
        message.error('Batch operation execution failed');
        setOperationProgress({ visible: false, current: 0, total: 0, status: '' });
      } finally {
        setLoading(false);
      }
    },
    [selectedTickets, form, onOperationComplete]
  );

  // 处理操作菜单点击
  const handleMenuClick: MenuProps['onClick'] = useCallback(
    (info: Parameters<NonNullable<MenuProps['onClick']>>[0]) => {
      const { key } = info;
      const operation = batchOperations.find(op => op.key === key);
      if (!operation || key === 'divider1') return;

      if (key === 'export') {
        handleBatchExport();
        return;
      }

      setOperationModal({
        visible: true,
        type: key,
        title: operation.label,
      });
    },
    [batchOperations]
  );

  // 批量导出
  const handleBatchExport = useCallback(async () => {
    try {
      // 后端 POST /tickets/export 的 filters 只有 status/priority（dto.ListTicketsRequest
      // 无 ticketIds，service.ExportTickets 也只读这两项），无法按选中工单导出。
      // 这里不发送 ticketIds，避免把“导出 N 条选中”伪装成“导出全量”。
      const blob = await TicketAPI.exportTickets({ format: 'excel' });

      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `tickets_export_${new Date().toISOString().split('T')[0]}.xlsx`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      message.success('Export completed: The file contains all work orders of this tenant, not limited to the currently selected items');
    } catch (error) {
      message.error('Export failed');
    }
  }, []);

  // 渲染操作表单
  const renderOperationForm = () => {
    switch (operationModal.type) {
      case 'assign':
        return (
          <>
            <Form.Item
              label="Assigned To"
              name="assigneeId"
              rules={[{ required: true, message: 'Please select an assignee' }]}
            >
              <Select
                placeholder="Select assignee"
                showSearch
                optionFilterProp="label"
                loading={assignees.isLoading}
                notFoundContent={assignees.isError ? 'Failed to load handler list' : undefined}
                options={assigneeOptions}
              />
            </Form.Item>
          </>
        );

      case 'update_status':
        return (
          <Form.Item
            label="New status"
            name="status"
            rules={[{ required: true, message: 'Please select status' }]}
          >
            <Select placeholder="Select status" options={STATUS_OPTIONS} />
          </Form.Item>
        );

      case 'add_tags':
        return (
          <Form.Item label="Tags" name="tags" rules={[{ required: true, message: 'Please enter tags' }]}>
            <Select mode="tags" placeholder="Enter the label and press Enter to add" style={{ width: '100%' }} options={[{ value: "urgent", label: "Urgent" }, { value: "sla", label: "SLA Monitor" }, { value: "escalation", label: "Escalated" }]} />
          </Form.Item>
        );

      case 'set_priority':
        return (
          <Form.Item
            label="Priority"
            name="priority"
            rules={[{ required: true, message: 'Please select priority' }]}
          >
            <Select placeholder="Select priority" options={PRIORITY_OPTIONS} />
          </Form.Item>
        );

      case 'delete':
        return (
          <Alert
            title="Delete confirmation"
            description={`About to be deleted ${selectedTickets.length} Work order, this operation cannot be restored. Please confirm whether to continue?`}
            type="error"
            showIcon
          />
        );

      default:
        return null;
    }
  };

  // 批量操作菜单项：只保留当前账号有权限码的动作。
  // 后端逐条 RequirePermission，这里只负责不给出只会 403 的入口。
  const permittedOperations = useMemo(
    () =>
      batchOperations.filter(
        op => op.key !== 'divider1' && (!op.permission || hasPermission(op.permission[0], op.permission[1]))
      ),
    [batchOperations, hasPermission]
  );

  const menuItems: MenuProps['items'] = permittedOperations.map(operation => ({
    key: operation.key,
    icon: operation.icon,
    label: operation.label,
    danger: operation.danger,
  }));

  if (selectedTickets.length === 0) {
    return null;
  }

  return (
    <div className="ticket-batch-operations">
      {/* 批量操作工具栏 */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <CheckCircle className="text-blue-600 text-lg" />
            <Text strong>Selected {selectedTickets.length}  work orders</Text>
            <Button size="small" onClick={onSelectionClear}>

              Clear selection
            </Button>
          </div>

          <Dropdown
            menu={{ items: menuItems, onClick: handleMenuClick }}
            placement="bottomRight"
            disabled={permittedOperations.length === 0}
          >
            <Button
              type="primary"
              icon={<MoreHorizontal />}
              loading={loading}
              disabled={permittedOperations.length === 0}
            >

              Batch operation
            </Button>
          </Dropdown>
        </div>
      </div>

      {/* 操作模态框 */}
      <Modal
        title={operationModal.title}
        open={operationModal.visible}
        onCancel={() => {
          setOperationModal({ visible: false, type: '', title: '' });
          form.resetFields();
        }}
        footer={[
          <Button
            key="cancel"
            onClick={() => setOperationModal({ visible: false, type: '', title: '' })}
          >

            Cancel
          </Button>,
          <Button
            key="confirm"
            type="primary"
            danger={operationModal.type === 'delete'}
            loading={loading}
            onClick={() => form.submit()}
          >
            {operationModal.type === 'delete' ? 'Confirm Delete' : 'Confirm operation'}
          </Button>,
        ]}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={values => executeBatchOperation(operationModal.type, values)}
        >
          {renderOperationForm()}
        </Form>
      </Modal>

      {/* 操作进度模态框 */}
      <Modal
        title="Batch Operation Progress"
        open={operationProgress.visible}
        closable={false}
        footer={null}
        width={500}
      >
        <div className="text-center py-6">
          <Progress
            percent={Math.round((operationProgress.current / operationProgress.total) * 100)}
            status={operationProgress.current === operationProgress.total ? 'success' : 'active'}
          />
          <div className="mt-4">
            <Text>{operationProgress.status}</Text>
          </div>
          <div className="mt-2">
            <Text type="secondary">
              {operationProgress.current} / {operationProgress.total}
            </Text>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default TicketBatchOperations;
