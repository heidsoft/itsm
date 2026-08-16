// 流程设置模态框
// Workflow Settings Modal

'use client';

import React from 'react';
import { Modal, Form, Tabs, Select, Input, Checkbox, Row, Col } from 'antd';
import type { FormInstance } from 'antd';
import { useI18n } from '@/lib/i18n/useI18n';

interface WorkflowSettingsModalProps {
  visible: boolean;
  onClose: () => void;
  onSave: () => void;
  form: FormInstance;
}

export default function WorkflowSettingsModal({
  visible,
  onClose,
  onSave,
  form,
}: WorkflowSettingsModalProps) {
  const { t } = useI18n();

  return (
    <Modal
      title={t('workflow.settingsModal.title')}
      open={visible}
      onOk={onSave}
      onCancel={onClose}
      width={800}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
    >
      <Form form={form} layout="vertical">
        <Tabs
          items={[
            {
              key: 'approval',
              label: t('workflow.settingsModal.tabApproval'),
              children: (
                <>
                  <Form.Item
                    label={t('workflow.settingsModal.approvalType')}
                    name={['approval_config', 'approval_type']}
                    rules={[{ required: true, message: t('workflow.settingsModal.approvalTypeRequired') }]}
                  >
                    <Select options={[
                      { value: 'single', label: t('workflow.settingsModal.approvalTypeSingle') },
                      { value: 'parallel', label: t('workflow.settingsModal.approvalTypeParallel') },
                      { value: 'sequential', label: t('workflow.settingsModal.approvalTypeSequential') },
                      { value: 'conditional', label: t('workflow.settingsModal.approvalTypeConditional') },
                    ]} />
                  </Form.Item>

                  <Form.Item label={t('workflow.settingsModal.approvers')} name={['approval_config', 'approvers']}>
                    <Select mode="multiple" placeholder={t('workflow.settingsModal.approversPlaceholder')}>
                      {/* 用户列表通过 Context 获取 */}
                    </Select>
                  </Form.Item>
                </>
              ),
            },
            {
              key: 'sla',
              label: t('workflow.settingsModal.tabSla'),
              children: (
                <>
                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        label={t('workflow.settingsModal.responseTimeHours')}
                        name={['sla_config', 'response_time_hours']}
                      >
                        <Input type="number" />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        label={t('workflow.settingsModal.resolutionTimeHours')}
                        name={['sla_config', 'resolution_time_hours']}
                      >
                        <Input type="number" />
                      </Form.Item>
                    </Col>
                  </Row>

                  <Form.Item
                    label={t('workflow.settingsModal.businessHoursSetting')}
                    name={['sla_config', 'business_hours_only']}
                    valuePropName="checked"
                  >
                    <Checkbox>{t('workflow.settingsModal.businessHoursOnly')}</Checkbox>
                  </Form.Item>
                </>
              ),
            },
          ]}
        />
      </Form>
    </Modal>
  );
}