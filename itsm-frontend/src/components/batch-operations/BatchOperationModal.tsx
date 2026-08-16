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
  Space,
  Radio,
  Checkbox,
} from 'antd';
import { AlertCircle, Rocket } from 'lucide-react';
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
import { useI18n } from '@/lib/i18n/useI18n';

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
  const { t } = useI18n();
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

  const getTitle = (): string => {
    const titles: Record<BatchOperationType, string> = {
      [BatchOperationType.ASSIGN]: t('batchOperation.titleAssign'),
      [BatchOperationType.UPDATE_STATUS]: t('batchOperation.titleUpdateStatus'),
      [BatchOperationType.UPDATE_PRIORITY]: t('batchOperation.titleUpdatePriority'),
      [BatchOperationType.UPDATE_TYPE]: t('batchOperation.titleUpdateType'),
      [BatchOperationType.UPDATE_CATEGORY]: t('batchOperation.titleUpdateCategory'),
      [BatchOperationType.ADD_TAGS]: t('batchOperation.titleAddTags'),
      [BatchOperationType.REMOVE_TAGS]: t('batchOperation.titleRemoveTags'),
      [BatchOperationType.UPDATE_FIELDS]: t('batchOperation.titleUpdateFields'),
      [BatchOperationType.DELETE]: t('batchOperation.titleDelete'),
      [BatchOperationType.ARCHIVE]: t('batchOperation.titleArchive'),
      [BatchOperationType.EXPORT]: t('batchOperation.titleExport'),
      [BatchOperationType.CLOSE]: t('batchOperation.titleClose'),
      [BatchOperationType.REOPEN]: t('batchOperation.titleReopen'),
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
      console.error('Batch operation failed:', error);
    }
  };

  const renderFormFields = () => {
    switch (operationType) {
      case BatchOperationType.ASSIGN:
        return (
          <>
            <Form.Item
              label={t('batchOperation.assignmentRule')}
              name="assignmentRule"
              rules={[{ required: true, message: t('batchOperation.assignmentRuleRequired') }]}
            >
              <Radio.Group>
                <Radio value="manual">{t('batchOperation.ruleManual')}</Radio>
                <Radio value="round_robin">{t('batchOperation.ruleRoundRobin')}</Radio>
                <Radio value="load_balance">{t('batchOperation.ruleLoadBalance')}</Radio>
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
                      label={t('batchOperation.assignee')}
                      name="assigneeId"
                      rules={[{ required: true, message: t('batchOperation.assigneeRequired') }]}
                    >
                      <Select placeholder={t('batchOperation.assigneePlaceholder')} showSearch optionFilterProp="label" loading={assignmentOptionsLoading} options={userOptions} />
                    </Form.Item>
                  );
                }
                return (
                  <Form.Item
                    label={t('batchOperation.targetTeam')}
                    name="teamId"
                    rules={[{ required: true, message: t('batchOperation.teamRequired') }]}
                  >
                    <Select placeholder={t('batchOperation.teamPlaceholder')} showSearch optionFilterProp="label" loading={assignmentOptionsLoading} options={teamOptions} />
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
              label={t('batchOperation.targetStatus')}
              name="status"
              rules={[{ required: true, message: t('batchOperation.statusRequired') }]}
            >
              <Select placeholder={t('batchOperation.statusPlaceholder')} options={[
                { value: "open", label: t('batchOperation.statusOpen') },
                { value: "in_progress", label: t('batchOperation.statusInProgress') },
                { value: "resolved", label: t('batchOperation.statusResolved') },
                { value: "closed", label: t('batchOperation.statusClosed') },
              ]} />
            </Form.Item>
            <Form.Item label={t('batchOperation.resolution')} name="resolution">
              <TextArea rows={3} placeholder={t('batchOperation.resolutionPlaceholder')} />
            </Form.Item>
          </>
        );

      case BatchOperationType.UPDATE_PRIORITY:
        return (
          <Form.Item
            label={t('batchOperation.priority')}
            name="priority"
            rules={[{ required: true, message: t('batchOperation.priorityRequired') }]}
          >
            <Select placeholder={t('batchOperation.priorityPlaceholder')} options={[
              { value: "low", label: t('batchOperation.priorityLow') },
              { value: "medium", label: t('batchOperation.priorityMedium') },
              { value: "high", label: t('batchOperation.priorityHigh') },
              { value: "urgent", label: t('batchOperation.priorityUrgent') },
            ]} />
          </Form.Item>
        );

      case BatchOperationType.ADD_TAGS:
      case BatchOperationType.REMOVE_TAGS:
        return (
          <Form.Item label={t('batchOperation.tags')} name="tags" rules={[{ required: true, message: t('batchOperation.tagsRequired') }]}>
            <Select mode="tags" placeholder={t('batchOperation.tagsPlaceholder')} />
          </Form.Item>
        );

      case BatchOperationType.CLOSE:
        return (
          <>
            <Form.Item label={t('batchOperation.closureReason')} name="closureReason">
              <Select placeholder={t('batchOperation.closureReasonPlaceholder')} options={[
                { value: "resolved", label: t('batchOperation.closureReasonResolved') },
                { value: "duplicate", label: t('batchOperation.closureReasonDuplicate') },
                { value: "invalid", label: t('batchOperation.closureReasonInvalid') },
                { value: "wont_fix", label: t('batchOperation.closureReasonWontFix') },
              ]} />
            </Form.Item>
            <Form.Item label={t('batchOperation.resolution')} name="resolution">
              <TextArea rows={3} placeholder={t('batchOperation.resolutionPlaceholder')} />
            </Form.Item>
          </>
        );

      case BatchOperationType.DELETE:
        return (
          <>
            <Alert
              message={t('batchOperation.deleteWarning')}
              description={t('batchOperation.deleteWarningDesc')}
              type="warning"
              showIcon
              icon={<AlertCircle />}
              className="mb-4"
            />
            <Form.Item label={t('batchOperation.deleteReason')} name="reason">
              <TextArea rows={2} placeholder={t('batchOperation.deleteReasonPlaceholder')} showCount maxLength={500} />
            </Form.Item>
            <Form.Item name="hardDelete" valuePropName="checked">
              <Checkbox>{t('batchOperation.hardDelete')}</Checkbox>
            </Form.Item>
            <Form.Item
              noStyle
              shouldUpdate={(previous, current) => previous.hardDelete !== current.hardDelete}
            >
              {({ getFieldValue }) => getFieldValue('hardDelete') ? (
                <Form.Item name="confirmPermanentDelete" valuePropName="checked"
                  rules={[{ validator: (_, checked) => checked ? Promise.resolve() : Promise.reject(new Error(t('batchOperation.confirmPermanentDelete'))) }]}
                >
                  <Checkbox>{t('batchOperation.confirmPermanentDeleteLabel')}</Checkbox>
                </Form.Item>
              ) : null}
            </Form.Item>
          </>
        );

      case BatchOperationType.EXPORT:
        return (
          <>
            <Form.Item
              label={t('batchOperation.exportFormat')}
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
            <Form.Item label={t('batchOperation.exportOptions')} name={['config', 'options']}>
              <Checkbox.Group>
                <Checkbox value="comments">{t('batchOperation.exportIncludeComments')}</Checkbox>
                <Checkbox value="attachments">{t('batchOperation.exportIncludeAttachments')}</Checkbox>
                <Checkbox value="history">{t('batchOperation.exportIncludeHistory')}</Checkbox>
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
            {t('common.cancel')}
          </Button>,
          <Button
            key="submit"
            type="primary"
            icon={<Rocket />}
            loading={mutation.isPending}
            disabled={mutation.isPending || ticketIds.length === 0}
            onClick={handleSubmit}
          >
            {t('batchOperation.execute')}
          </Button>,
        ]}
      >
        <Alert
          message={t('batchOperation.aboutToApply', { count: ticketIds.length })}
          type="info"
          showIcon
          className="mb-4"
        />

        <Form form={form} layout="vertical" requiredMark="optional">
          {renderFormFields()}

          <Form.Item label={t('batchOperation.comment')} name="comment">
            <TextArea rows={2} placeholder={t('batchOperation.commentPlaceholder')} />
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