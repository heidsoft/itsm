'use client';

import React, { useState, useCallback, useMemo } from 'react';
import {
  Button,
  Dropdown,
  Modal,
  Form,
  Select,
  Input,
  message,
  Space,
  Alert,
  Progress,
  Divider,
  Typography,
  Checkbox,
} from 'antd';
import {
  UserOutlined,
  TagOutlined,
  FlagOutlined,
  DeleteOutlined,
  ExportOutlined,
  BellOutlined,
  MoreOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { Ticket } from '@/app/lib/api-config';
import { TicketAPI } from '@/lib/api/ticket-api';

const { Option } = Select;
const { TextArea } = Input;
const { Text, Title } = Typography;

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
}

const TicketBatchOperations: React.FC<TicketBatchOperationsProps> = ({
  selectedTickets,
  onOperationComplete,
  onSelectionClear,
}) => {
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
  const batchOperations: BatchOperation[] = useMemo(() => [
    {
      key: 'assign',
      label: '批量分配',
      icon: <UserOutlined />,
      description: '将选中的工单分配给指定处理人',
    },
    {
      key: 'update_status',
      label: '批量更新状态',
      icon: <FlagOutlined />,
      description: '批量更新工单状态',
    },
    {
      key: 'add_tags',
      label: '批量添加标签',
      icon: <TagOutlined />,
      description: '为选中的工单添加标签',
    },
    {
      key: 'set_priority',
      label: '批量设置优先级',
      icon: <FlagOutlined />,
      description: '批量设置工单优先级',
    },
    {
      key: 'notify',
      label: '批量通知',
      icon: <BellOutlined />,
      description: '向工单相关人员发送通知',
    },
    {
      key: 'export',
      label: '批量导出',
      icon: <ExportOutlined />,
      description: '导出选中的工单数据',
    },
    {
      key: 'divider1',
      label: '-',
      icon: null,
    },
    {
      key: 'delete',
      label: '批量删除',
      icon: <DeleteOutlined />,
      danger: true,
      description: '删除选中的工单（不可恢复）',
    },
  ], []);

  // 执行批量操作
  const executeBatchOperation = useCallback(async (operation: string, values: any) => {
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
          status: `处理工单 ${ticket.ticket_number}...`,
        }));

        try {
          switch (operation) {
            case 'assign':
              await TicketAPI.assignTicket(ticket.id, {
                assignee_id: values.assignee_id,
                comment: values.comment,
              });
              break;
            case 'update_status':
              await TicketAPI.updateTicket(ticket.id, {
                status: values.status,
              });
              break;
            case 'add_tags':
              await TicketAPI.addTicketTags(ticket.id, values.tags);
              break;
            case 'set_priority':
              await TicketAPI.updateTicket(ticket.id, {
                priority: values.priority,
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
          errors.push(`${ticket.ticket_number}: ${error instanceof Error ? error.message : '操作失败'}`);
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

      if (errors.length > 0 && operation === 'delete') {
        console.error('批量操作错误:', errors);
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
  }, [selectedTickets, form, onOperationComplete]);

  // 处理操作菜单点击
  const handleMenuClick: MenuProps['onClick'] = useCallback((info: Parameters<NonNullable<MenuProps['onClick']>>[0]) => {
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
  }, [batchOperations]);

  // 批量导出
  const handleBatchExport = useCallback(async () => {
    try {
      const ticketIds = selectedTickets.map(ticket => ticket.id);
      const blob = await TicketAPI.exportTickets({
        format: 'excel',
        filters: { ticket_ids: ticketIds },
      });
      
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `tickets_batch_${new Date().toISOString().split('T')[0]}.xlsx`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      
      message.success(`成功导出 ${selectedTickets.length} 个工单`);
    } catch (error) {
      message.error('导出失败');
    }
  }, [selectedTickets]);

  // 渲染操作表单
  const renderOperationForm = () => {
    switch (operationModal.type) {
      case 'assign':
        return (
          <>
            <Form.Item
              label="分配给"
              name="assignee_id"
              rules={[{ required: true, message: '请选择处理人' }]}
            >
              <Select placeholder="选择处理人" showSearch filterOption>
                {/* 这里应该从API获取用户列表 */}
                <Option value={1}>张三</Option>
                <Option value={2}>李四</Option>
                <Option value={3}>王五</Option>
              </Select>
            </Form.Item>
            <Form.Item label="分配备注" name="comment">
              <TextArea rows={3} placeholder="可选的分配备注" />
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
            <Select placeholder="选择状态">
              <Option value="new">新建</Option>
              <Option value="open">待处理</Option>
              <Option value="in_progress">处理中</Option>
              <Option value="pending">等待中</Option>
              <Option value="resolved">已解决</Option>
              <Option value="closed">已关闭</Option>
            </Select>
          </Form.Item>
        );

      case 'add_tags':
        return (
          <Form.Item
            label="标签"
            name="tags"
            rules={[{ required: true, message: '请输入标签' }]}
          >
            <Select
              mode="tags"
              placeholder="输入标签，按回车添加"
              style={{ width: '100%' }}
            >
              <Option value="urgent">紧急</Option>
              <Option value="sla">SLA监控</Option>
              <Option value="escalation">已升级</Option>
            </Select>
          </Form.Item>
        );

      case 'set_priority':
        return (
          <Form.Item
            label="优先级"
            name="priority"
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <Select placeholder="选择优先级">
              <Option value="low">低</Option>
              <Option value="medium">中</Option>
              <Option value="high">高</Option>
              <Option value="urgent">紧急</Option>
              <Option value="critical">严重</Option>
            </Select>
          </Form.Item>
        );

      case 'notify':
        return (
          <>
            <Form.Item label="通知内容" name="notification" rules={[{ required: true }]}>
              <TextArea rows={4} placeholder="输入通知内容" />
            </Form.Item>
            <Form.Item name="notify_assignee" valuePropName="checked" initialValue={true}>
              <Checkbox>通知处理人</Checkbox>
            </Form.Item>
            <Form.Item name="notify_reporter" valuePropName="checked">
              <Checkbox>通知报告人</Checkbox>
            </Form.Item>
          </>
        );

      case 'delete':
        return (
          <Alert
            message="删除确认"
            description={`即将删除 ${selectedTickets.length} 个工单，此操作不可恢复。请确认是否继续？`}
            type="error"
            showIcon
          />
        );

      default:
        return null;
    }
  };

  // 批量操作菜单项
  const menuItems: MenuProps['items'] = batchOperations
    .filter(op => op.key !== 'divider1')
    .map(operation => ({
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
            <CheckCircleOutlined className="text-blue-600 text-lg" />
            <Text strong>已选择 {selectedTickets.length} 个工单</Text>
            <Button size="small" onClick={onSelectionClear}>
              清空选择
            </Button>
          </div>
          
          <Dropdown
            menu={{ items: menuItems, onClick: handleMenuClick }}
            placement="bottomRight"
          >
            <Button type="primary" icon={<MoreOutlined />} loading={loading}>
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
          <Button key="cancel" onClick={() => setOperationModal({ visible: false, type: '', title: '' })}>
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
          onFinish={(values) => executeBatchOperation(operationModal.type, values)}
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
