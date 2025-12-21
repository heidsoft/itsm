'use client';

import React from 'react';
import { Modal, Form, Input, Select, Row, Col, InputNumber } from 'antd';
import { useI18n } from '@/lib/i18n';

const { Option } = Select;

interface CreateCIModalProps {
  visible: boolean;
  onCancel: () => void;
  onConfirm: () => void;
  form: any;
}

export const CreateCIModal: React.FC<CreateCIModalProps> = ({
  visible,
  onCancel,
  onConfirm,
  form,
}) => {
    const { t } = useI18n();
  return (
    <Modal
      title={t('cmdb.newCI')}
      open={visible}
      onCancel={onCancel}
      onOk={onConfirm}
      width={700}
    >
      <Form form={form} layout='vertical' style={{ marginTop: 20 }}>
        <Row gutter={24}>
          <Col span={12}>
            <Form.Item
              label={t('cmdb.ciName')}
              name='name'
              rules={[{ required: true, message: t('cmdb.ciNameRequired') }]}
            >
              <Input placeholder={t('cmdb.ciNamePlaceholder')} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label={t('cmdb.type')}
              name='type'
              rules={[{ required: true, message: t('cmdb.typeRequired') }]}
            >
              <Select placeholder={t('cmdb.typePlaceholder')}>
                <Option value='云服务器'>{t('cmdb.cloudServer')}</Option>
                <Option value='物理服务器'>{t('cmdb.physicalServer')}</Option>
                <Option value='关系型数据库'>{t('cmdb.relationalDatabase')}</Option>
                <Option value='存储设备'>{t('cmdb.storageDevice')}</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={24}>
          <Col span={12}>
            <Form.Item label={t('cmdb.cpu')} name='cpu'>
              <InputNumber
                placeholder={t('cmdb.cpuPlaceholder')}
                addonAfter={t('cmdb.cores')}
                min={1}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label={t('cmdb.memory')} name='memory'>
              <InputNumber
                placeholder={t('cmdb.memoryPlaceholder')}
                addonAfter={t('cmdb.gb')}
                min={1}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={24}>
          <Col span={12}>
            <Form.Item label={t('cmdb.disk')} name='disk'>
              <InputNumber
                placeholder={t('cmdb.diskPlaceholder')}
                addonAfter={t('cmdb.gb')}
                min={1}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label={t('cmdb.ipAddress')} name='ip'>
              <Input placeholder={t('cmdb.ipAddressPlaceholder')} />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item label={t('cmdb.ciDescription')} name='description'>
          <Input.TextArea placeholder={t('cmdb.ciDescriptionPlaceholder')} rows={4} />
        </Form.Item>
      </Form>
    </Modal>
  );
};
