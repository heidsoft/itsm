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
        label: '批量分配',
        icon: <UserIcon />,
        description: '将选中的工单分配给指定处理人',
        permission: ['ticket', 'assign'],
      },
      {
        key: 'update_status',
        label: '批量更新状态',
        icon: <Flag />,
        description: '批量更新工单状态',
        permission: ['ticket', 'update'],
      },
      {
        key: 'add_tags',
        label: '批量添加标签',
        icon: <Tag />,
        description: '为选中的工单添加标签',
        permission: ['ticket_tag', 'create'],
      },
      {
        key: 'set_priority',
        label: '批量设置优先级',
        icon: <Flag />,
        description: '批量设置工单优先级',
        permission: ['ticket', 'update'],
      },
      {
        key: 'export',
        label: '批量导出',
        icon: <Download />,
        description: '导出选中的工单数据',
        permission: ['ticket', 'export'],
      },
      {
        key: 'divider1',
        label: '-',
        icon: null,
      },
      {
        key: 'delete',
        label: '批量删除',
        icon: <Trash2 />,
        danger: true,
        description: '删除选中的工单（不可恢复）',
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
        status: '准备执行...',
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
            status: `处理工单 ${ticket.ticketNumber}...`,
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
                throw new Error(`不支持的操作: ${operation}`);
            }
            successCount++;
          } catch (error) {
            failCount++;
            errors.push(
              `${ticket.ticketNumber}: ${error instanceof Error ? error.message : '操作失败'}`
            );
          }

          // 模拟处理延迟，避免请求过于频繁
          await new Promise(resolve => setTimeout(resolve, 100));
        }

        setOperationProgress(prev => ({
          ...prev,
          status: '操作完成',
        }));

        // 显示操作结果
        if (failCount === 0) {
          message.success(`批量操作成功！共处理 ${successCount} 个工单`);
        } else {
          message.warning(`操作完成！成功 ${successCount} 个，失败 ${failCount} 个`);
        }

        setTimeout(() => {
          setOperationProgress({ visible: false, current: 0, total: 0, status: '' });
          setOperationModal({ visible: false, type: '', title: '' });
          form.resetFields();
          onOperationComplete?.();
        }, 1500);
      } catch (error) {
        message.error('批量操作执行失败');
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

      message.success('导出完成：文件包含本租户全部工单，不仅限于当前选中项');
    } catch (error) {
      message.error('导出失败');
    }
  }, []);

  // 渲染操作表单
  const renderOperationForm = () => {
    switch (operationModal.type) {
      case 'assign':
        return (
          <>
            <Form.Item
              label="分配给"
              name="assigneeId"
              rules={[{ required: true, message: '请选择处理人' }]}
            >
              <Select
                placeholder="选择处理人"
                showSearch
                optionFilterProp="label"
                loading={assignees.isLoading}
                notFoundContent={assignees.isError ? '处理人列表加载失败' : undefined}
                options={assigneeOptions}
              />
            </Form.Item>
          </>
        );

      case 'update_status':
        return (
          <Form.Item
            label="新状态"
            name="status"
            rules={[{ required: true, message: '请选择状态' }]}
          >
            <Select placeholder="选择状态" options={STATUS_OPTIONS} />
          </Form.Item>
        );

      case 'add_tags':
        return (
          <Form.Item label="标签" name="tags" rules={[{ required: true, message: '请输入标签' }]}>
            <Select mode="tags" placeholder="输入标签，按回车添加" style={{ width: '100%' }} options={[{ value: "urgent", label: "紧急" }, { value: "sla", label: "SLA监控" }, { value: "escalation", label: "已升级" }]} />
          </Form.Item>
        );

      case 'set_priority':
        return (
          <Form.Item
            label="优先级"
            name="priority"
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <Select placeholder="选择优先级" options={PRIORITY_OPTIONS} />
          </Form.Item>
        );

      case 'delete':
        return (
          <Alert
            title="删除确认"
            description={`即将删除 ${selectedTickets.length} 个工单，此操作不可恢复。请确认是否继续？`}
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
            <Text strong>已选择 {selectedTickets.length} 个工单</Text>
            <Button size="small" onClick={onSelectionClear}>
              清空选择
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
              批量操作
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
            取消
          </Button>,
          <Button
            key="confirm"
            type="primary"
            danger={operationModal.type === 'delete'}
            loading={loading}
            onClick={() => form.submit()}
          >
            {operationModal.type === 'delete' ? '确认删除' : '确认操作'}
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
        title="批量操作进度"
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
