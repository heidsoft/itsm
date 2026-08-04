/**
 * 批量操作配置模态框
 * 提供批量操作的参数配置界面
 */

'use client';

import React, { useEffect, useState } from 'react';
import {
  Modal,
  Form,
  Select,
  Input,
  Button,
  Alert,
  Spin,
  Space,
  Tag,
  Divider,
  Radio,
  Checkbox,
} from 'antd';
import { Eye, AlertCircle, Rocket } from 'lucide-react';
import { BatchOperationType } from '@/types/batch-operations';
import {
  useBatchAssignMutation,
  useBatchUpdateStatusMutation,
  useBatchUpdatePriorityMutation,
  useBatchUpdateFieldsMutation,
  useBatchAddTagsMutation,
  useBatchRemoveTagsMutation,
  useBatchDeleteMutation,
  useBatchCloseMutation,
  useBatchExportMutation,
} from '@/lib/hooks/useBatchOperations';
import { BatchProgressModal } from './BatchProgressModal';
import { UserApi } from '@/lib/api/user-api';
import { CommonApi } from '@/lib/api/common-api';

const { TextArea } = Input;
export interface BatchOperationModalProps {
  visible: boolean;
  operationType: BatchOperationType;
  ticketIds: number[];
  onSuccess: () => void;
  onCancel: () => void;
}

export const BatchOperationModal: React.FC<BatchOperationModalProps> = ({
  visible,
  operationType,
  ticketIds,
  onSuccess,
  onCancel,
}) => {
  const [form] = Form.useForm();
  const [progressVisible, setProgressVisible] = useState(false);
  const [operationId, setOperationId] = useState<string | null>(null);
  const [assignmentOptionsLoading, setAssignmentOptionsLoading] = useState(false);
  const [userOptions, setUserOptions] = useState<Array<{ value: number; label: string }>>([]);
  const [teamOptions, setTeamOptions] = useState<Array<{ value: number; label: string }>>([]);

  useEffect(() => {
    if (!visible || operationType !== BatchOperationType.ASSIGN) return;
    let active = true;
    setAssignmentOptionsLoading(true);
    Promise.all([UserApi.getUsers({ page: 1, pageSize: 100 }), CommonApi.getTeams()])
      .then(([userResponse, teams]) => {
        if (!active) return;
        setUserOptions((userResponse.users || []).map(user => ({ value: user.id, label: user.name || user.username })));
        setTeamOptions((Array.isArray(teams) ? teams : []).map(team => ({ value: team.id, label: team.name })));
      })
      .catch(() => {
        if (!active) return;
        setUserOptions([]);
        setTeamOptions([]);
      })
      .finally(() => active && setAssignmentOptionsLoading(false));
    return () => { active = false; };
  }, [visible, operationType]);

  // Mutations
  const assignMutation = useBatchAssignMutation();
  const updateStatusMutation = useBatchUpdateStatusMutation();
  const updatePriorityMutation = useBatchUpdatePriorityMutation();
  const updateFieldsMutation = useBatchUpdateFieldsMutation();
  const addTagsMutation = useBatchAddTagsMutation();
  const removeTagsMutation = useBatchRemoveTagsMutation();
  const deleteMutation = useBatchDeleteMutation();
  const closeMutation = useBatchCloseMutation();
  const exportMutation = useBatchExportMutation();

  const getMutation = () => {
    switch (operationType) {
      case BatchOperationType.ASSIGN:
        return assignMutation;
      case BatchOperationType.UPDATE_STATUS:
        return updateStatusMutation;
      case BatchOperationType.UPDATE_PRIORITY:
        return updatePriorityMutation;
      case BatchOperationType.UPDATE_FIELDS:
        return updateFieldsMutation;
      case BatchOperationType.ADD_TAGS:
        return addTagsMutation;
      case BatchOperationType.REMOVE_TAGS:
        return removeTagsMutation;
      case BatchOperationType.DELETE:
        return deleteMutation;
      case BatchOperationType.CLOSE:
        return closeMutation;
      case BatchOperationType.EXPORT:
        return exportMutation;
      default:
        return assignMutation;
    }
  };

  const mutation = getMutation();

  const getTitle = () => {
    const titles: Record<BatchOperationType, string> = {
      [BatchOperationType.ASSIGN]: '批量分配',
      [BatchOperationType.UPDATE_STATUS]: '批量更新状态',
      [BatchOperationType.UPDATE_PRIORITY]: '批量更新优先级',
      [BatchOperationType.UPDATE_TYPE]: '批量更新类型',
      [BatchOperationType.UPDATE_CATEGORY]: '批量更新分类',
      [BatchOperationType.ADD_TAGS]: '批量添加标签',
      [BatchOperationType.REMOVE_TAGS]: '批量删除标签',
      [BatchOperationType.UPDATE_FIELDS]: '批量更新字段',
      [BatchOperationType.DELETE]: '批量删除',
      [BatchOperationType.ARCHIVE]: '批量归档',
      [BatchOperationType.EXPORT]: '批量导出',
      [BatchOperationType.CLOSE]: '批量关闭',
      [BatchOperationType.REOPEN]: '批量重新打开',
    };
    return titles[operationType];
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      const data = { ticketIds, ...values };

      const result = await mutation.mutateAsync(data as never);
      const nextOperationId = (result as { operationId?: string })?.operationId;
      if (nextOperationId) {
        setOperationId(nextOperationId);
        setProgressVisible(true);
      }
      onSuccess();
    } catch (error) {
      console.error('批量操作失败:', error);
    }
  };

  const renderFormFields = () => {
    switch (operationType) {
      case BatchOperationType.ASSIGN:
        return (
          <>
            <Form.Item
              label="分配方式"
              name="assignmentRule"
              rules={[{ required: true, message: '请选择分配方式' }]}
            >
              <Radio.Group>
                <Radio value="manual">手动指定</Radio>
                <Radio value="round_robin">轮流分配</Radio>
                <Radio value="load_balance">负载均衡</Radio>
              </Radio.Group>
            </Form.Item>

            <Form.Item
              noStyle
              shouldUpdate={(prevValues, currentValues) =>
                prevValues.assignmentRule !== currentValues.assignmentRule
              }
            >
              {({ getFieldValue }) => {
                const rule = getFieldValue('assignmentRule');
                if (rule === 'manual') {
                  return (
                    <Form.Item
                      label="处理人"
                      name="assigneeId"
                      rules={[{ required: true, message: '请选择处理人' }]}
                    >
                      <Select placeholder="选择处理人" showSearch optionFilterProp="label" loading={assignmentOptionsLoading} options={userOptions} />
                    </Form.Item>
                  );
                }
                return (
                  <Form.Item
                    label="目标团队"
                    name="teamId"
                    rules={[{ required: true, message: '请选择团队' }]}
                  >
                    <Select placeholder="选择团队" showSearch optionFilterProp="label" loading={assignmentOptionsLoading} options={teamOptions} />
                  </Form.Item>
                );
              }}
            </Form.Item>
          </>
        );

      case BatchOperationType.UPDATE_STATUS:
        return (
          <>
            <Form.Item
              label="目标状态"
              name="status"
              rules={[{ required: true, message: '请选择状态' }]}
            >
              <Select placeholder="选择状态" options={[{ value: "open", label: "打开" }, { value: "in_progress", label: "处理中" }, { value: "resolved", label: "已解决" }, { value: "closed", label: "已关闭" }]} />
            </Form.Item>
            <Form.Item label="解决方案" name="resolution">
              <TextArea rows={3} placeholder="输入解决方案..." />
            </Form.Item>
          </>
        );

      case BatchOperationType.UPDATE_PRIORITY:
        return (
          <Form.Item
            label="优先级"
            name="priority"
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <Select placeholder="选择优先级" options={[{ value: "low", label: "低" }, { value: "medium", label: "中" }, { value: "high", label: "高" }, { value: "urgent", label: "紧急" }]} />
          </Form.Item>
        );

      case BatchOperationType.ADD_TAGS:
      case BatchOperationType.REMOVE_TAGS:
        return (
          <Form.Item label="标签" name="tags" rules={[{ required: true, message: '请输入标签' }]}>
            <Select mode="tags" placeholder="输入标签（回车添加）" />
          </Form.Item>
        );

      case BatchOperationType.CLOSE:
        return (
          <>
            <Form.Item label="关闭原因" name="closureReason">
              <Select placeholder="选择关闭原因" options={[{ value: "resolved", label: "问题已解决" }, { value: "duplicate", label: "重复工单" }, { value: "invalid", label: "无效工单" }, { value: "wont_fix", label: "不予修复" }]} />
            </Form.Item>
            <Form.Item label="解决方案" name="resolution">
              <TextArea rows={3} placeholder="输入解决方案..." />
            </Form.Item>
          </>
        );

      case BatchOperationType.DELETE:
        return (
          <>
            <Alert
              message="警告"
              description="删除操作不可撤销，请谨慎操作！"
              type="warning"
              showIcon
              icon={<AlertCircle />}
              className="mb-4"
            />
            <Form.Item label="删除原因" name="reason">
              <TextArea rows={2} placeholder="请说明删除原因..." showCount maxLength={500} />
            </Form.Item>
            <Form.Item name="hardDelete" valuePropName="checked">
              <Checkbox>永久删除（不可恢复）</Checkbox>
            </Form.Item>
            <Form.Item
              noStyle
              shouldUpdate={(previous, current) => previous.hardDelete !== current.hardDelete}
            >
              {({ getFieldValue }) => getFieldValue('hardDelete') ? (
                <Form.Item name="confirmPermanentDelete" valuePropName="checked"
                  rules={[{ validator: (_, checked) => checked ? Promise.resolve() : Promise.reject(new Error('请确认永久删除风险')) }]}
                >
                  <Checkbox>我已了解永久删除不可恢复，并确认继续</Checkbox>
                </Form.Item>
              ) : null}
            </Form.Item>
          </>
        );

      case BatchOperationType.EXPORT:
        return (
          <>
            <Form.Item
              label="导出格式"
              name={['config', 'format']}
              initialValue="excel"
              rules={[{ required: true }]}
            >
              <Radio.Group>
                <Radio value="excel">Excel</Radio>
                <Radio value="csv">CSV</Radio>
                <Radio value="pdf">PDF</Radio>
              </Radio.Group>
            </Form.Item>
            <Form.Item label="导出选项" name={['config', 'options']}>
              <Checkbox.Group>
                <Checkbox value="comments">包含评论</Checkbox>
                <Checkbox value="attachments">包含附件</Checkbox>
                <Checkbox value="history">包含历史记录</Checkbox>
              </Checkbox.Group>
            </Form.Item>
          </>
        );

      default:
        return null;
    }
  };

  return (
    <>
      <Modal
        title={getTitle()}
        open={visible}
        onCancel={onCancel}
        maskClosable={!mutation.isPending}
        closable={!mutation.isPending}
        width={600}
        footer={[
          <Button key="cancel" onClick={onCancel}>
            取消
          </Button>,
          <Button
            key="submit"
            type="primary"
            icon={<Rocket />}
            loading={mutation.isPending}
            disabled={mutation.isPending || ticketIds.length === 0}
            onClick={handleSubmit}
          >
            执行
          </Button>,
        ]}
      >
        <Alert
          message={`即将对 ${ticketIds.length} 个工单执行批量操作`}
          type="info"
          showIcon
          className="mb-4"
        />

        <Form form={form} layout="vertical" requiredMark="optional">
          {renderFormFields()}

          <Divider />

          <Form.Item label="备注说明" name="comment">
            <TextArea rows={2} placeholder="添加备注说明（可选）" />
          </Form.Item>
        </Form>
      </Modal>

      {progressVisible && operationId && (
        <BatchProgressModal
          visible={progressVisible}
          operationId={operationId}
          onClose={() => setProgressVisible(false)}
        />
      )}
    </>
  );
};

export default BatchOperationModal;
