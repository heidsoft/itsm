'use client';

import React from 'react';
import { Modal, Form, Input, InputNumber, Select } from 'antd';
import type { FormInstance } from 'antd';
import { PlusCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;


interface CreateServiceModalProps {
  visible: boolean;
  onCancel: () => void;
  onConfirm: () => void;
  form: FormInstance;
  loading?: boolean;
}

export const CreateServiceModal: React.FC<CreateServiceModalProps> = ({
  visible,
  onCancel,
  onConfirm,
  form,
  loading = false,
}) => {
  const { t } = useI18n();
  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <div
            style={{
              width: 32,
              height: 32,
              backgroundColor: '#1890ff',
              borderRadius: 6,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginRight: 12,
            }}
          >
            <PlusCircle size={18} style={{ color: '#fff' }} />
          </div>
          <span style={{ fontSize: 'large', fontWeight: 'medium' }}>
            {t('serviceCatalog.newService')}
          </span>
        </div>
      }
      open={visible}
      onCancel={onCancel}
      onOk={onConfirm}
      confirmLoading={loading}
      okButtonProps={{ disabled: loading }}
      width={600}
    >
      <Form form={form} layout="vertical" style={{ marginTop: 20 }}>
        <Form.Item
          label={t('serviceCatalog.name')}
          name="name"
          rules={[{ required: true, message: t('serviceCatalog.serviceNameRequired') }]}
        >
          <Input placeholder={t('serviceCatalog.name')} />
        </Form.Item>

        <Form.Item
          label={t('serviceCatalog.category')}
          name="category"
          rules={[{ required: true, message: t('serviceCatalog.serviceCategoryRequired') }]}
        >
          <Select placeholder={t('serviceCatalog.category')} options={[
            { value: "云计算", label: t('serviceCatalog.cloudResources') },
            { value: "支持", label: t('serviceCatalog.accountPermissions') },
            { value: "安全", label: t('serviceCatalog.securityServices') },
          ]} />
        </Form.Item>

        <Form.Item
          label={t('serviceCatalog.deliveryTime')}
          name="deliveryDays"
          extra="交付周期按自然日保存，范围 1–3650 天"
          rules={[
            { required: true, message: t('serviceCatalog.serviceDeliveryTimeRequired') },
            { type: 'integer', min: 1, max: 3650, message: '交付天数必须是 1 到 3650 之间的整数' },
          ]}
        >
          <InputNumber min={1} max={3650} precision={0} className="w-full" suffix="天" />
        </Form.Item>

        <Form.Item
          label={t('serviceCatalog.status')}
          name="status"
          rules={[{ required: true, message: t('serviceCatalog.serviceStatusRequired') }]}
        >
          <Select placeholder={t('serviceCatalog.status')} options={[
            { value: "published", label: t('serviceCatalog.enabled') },
            { value: "retired", label: t('serviceCatalog.disabled') },
          ]} />
        </Form.Item>

        <Form.Item label={t('serviceCatalog.serviceDescription')} name="description">
          <TextArea
            placeholder={t('serviceCatalog.serviceDescriptionPlaceholder')}
            rows={4}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};
