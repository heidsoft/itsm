'use client';

import React from 'react';
import { Modal, Form, Input, Select } from 'antd';
import { PlusCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { Option } = Select;

interface CreateServiceModalProps {
  visible: boolean;
  onCancel: () => void;
  onConfirm: () => void;
  form: any;
}

export const CreateServiceModal: React.FC<CreateServiceModalProps> = ({
  visible,
  onCancel,
  onConfirm,
  form,
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
          <span style={{ fontSize: 'large', fontWeight: 'medium' }}>{t('serviceCatalog.newService')}</span>
        </div>
      }
      open={visible}
      onCancel={onCancel}
      onOk={onConfirm}
      width={600}
    >
      <Form form={form} layout='vertical' style={{ marginTop: 20 }}>
        <Form.Item
          label={t('serviceCatalog.name')}
          name='name'
          rules={[{ required: true, message: t('serviceCatalog.serviceNameRequired') }]}
        >
          <Input placeholder={t('serviceCatalog.name')} />
        </Form.Item>

        <Form.Item
          label={t('serviceCatalog.category')}
          name='category'
          rules={[{ required: true, message: t('serviceCatalog.serviceCategoryRequired') }]}
        >
          <Select placeholder={t('serviceCatalog.category')}>
            <Option value='云资源服务'>{t('serviceCatalog.cloudResources')}</Option>
            <Option value='账号与权限'>{t('serviceCatalog.accountPermissions')}</Option>
            <Option value='安全服务'>{t('serviceCatalog.securityServices')}</Option>
          </Select>
        </Form.Item>

        <Form.Item
          label={t('serviceCatalog.deliveryTime')}
          name='deliveryTime'
          rules={[{ required: true, message: t('serviceCatalog.serviceDeliveryTimeRequired') }]}
        >
          <Input placeholder={t('serviceCatalog.deliveryTimePlaceholder')} />
        </Form.Item>

        <Form.Item label={t('serviceCatalog.status')} name='status' rules={[{ required: true, message: t('serviceCatalog.serviceStatusRequired') }]}>
          <Select placeholder={t('serviceCatalog.status')}>
            <Option value='enabled'>{t('serviceCatalog.enabled')}</Option>
            <Option value='disabled'>{t('serviceCatalog.disabled')}</Option>
          </Select>
        </Form.Item>

        <Form.Item label={t('serviceCatalog.serviceDescription')} name='description'>
          <Input.TextArea placeholder={t('serviceCatalog.serviceDescriptionPlaceholder')} rows={4} />
        </Form.Item>
      </Form>
    </Modal>
  );
};
