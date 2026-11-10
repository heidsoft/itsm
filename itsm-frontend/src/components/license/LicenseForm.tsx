'use client';

/**
 * 许可证创建/编辑表单组件
 */

import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Space,
  Divider,
  message,
  InputNumber,
  DatePicker,
} from 'antd';
import { useRouter, useParams } from 'next/navigation';
import { ArrowLeft, Save } from 'lucide-react';

import type { License, LicenseRequest} from '@/lib/api/asset-api';
import { AssetApi, type LicenseType } from '@/lib/api/asset-api';
import type { Dayjs } from 'dayjs';

const { TextArea } = Input;

const LicenseForm: React.FC = () => {
  const router = useRouter();
  const { id } = useParams() as { id: string };
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<License | null>(null);
  const isEdit = !!id;

  useEffect(() => {
    if (id) {
      loadDetail();
    }
  }, [id]);

  const loadDetail = async () => {
    setLoading(true);
    try {
      const data = await AssetApi.getLicense(Number(id));
      setDetail(data);
      form.setFieldsValue(data);
    } catch (error) {
      message.error('Failed to load license details');
    } finally {
      setLoading(false);
    }
  };

  const onFinish = async (values: {
    name: string;
    description?: string;
    vendor?: string;
    licenseType?: LicenseType;
    licenseKey?: string;
    totalQuantity?: number;
    assetId?: number;
    purchaseDate?: Dayjs;
    purchasePrice?: number;
    expiryDate?: Dayjs;
    supportVendor?: string;
    supportContact?: string;
    renewalCost?: number;
    notes?: string;
    tags?: string[];
  }) => {
    setLoading(true);
    try {
      const data: LicenseRequest = {
        name: values.name,
        description: values.description,
        vendor: values.vendor,
        licenseType: values.licenseType,
        licenseKey: values.licenseKey,
        totalQuantity: values.totalQuantity,
        assetId: values.assetId,
        purchaseDate: values.purchaseDate?.toISOString(),
        purchasePrice: values.purchasePrice,
        expiryDate: values.expiryDate?.toISOString(),
        supportVendor: values.supportVendor,
        supportContact: values.supportContact,
        renewalCost: values.renewalCost ? String(values.renewalCost) : undefined,
        notes: values.notes,
        tags: values.tags,
      };

      if (isEdit) {
        await AssetApi.updateLicense(Number(id), data);
        message.success('Updated');
      } else {
        await AssetApi.createLicense(data);
        message.success('Created');
      }
      router.push('/licenses');
    } catch (error) {
      message.error(isEdit ? 'Update failed' : 'Creation failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <Form
        form={form}
        layout="vertical"
        onFinish={onFinish}
        initialValues={{
          licenseType: 'subscription',
          totalQuantity: 1,
        }}
      >
        <div style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeft />} onClick={() => router.push('/licenses')}>

            Back to List
          </Button>
        </div>

        <Divider>Basic Info</Divider>

        <Form.Item
          name="name"
          label="LicensesName"
          rules={[{ required: true, message: 'Please enter the license name' }]}
        >
          <Input placeholder="Example: Microsoft 365 E3" />
        </Form.Item>

        <Form.Item name="description" label="Description">
          <TextArea rows={3} placeholder="License description" />
        </Form.Item>

        <Form.Item name="vendor" label="Supplier">
          <Input placeholder="Example: Microsoft" />
        </Form.Item>

        <Form.Item name="licenseType" label="License type">
          <Select options={[
            { value: 'perpetual', label: 'Perpetual' },
            { value: 'subscription', label: 'Subscription' },
            { value: 'per-user', label: 'Per-User' },
            { value: 'per-seat', label: 'Per-Seat' },
            { value: 'site', label: 'Site' },
          ]} />
        </Form.Item>

        <Form.Item name="licenseKey" label="License Key">
          <TextArea rows={2} placeholder="License Key" />
        </Form.Item>

        <Divider>Quantity and Usage</Divider>

        <Form.Item name="totalQuantity" label="Total quantity">
          <InputNumber min={1} style={{ width: '100%' }} />
        </Form.Item>

        <Divider>Purchasing and Finance</Divider>

        <Form.Item name="purchaseDate" label="Purchase date">
          <DatePicker style={{ width: '100%' }} placeholder="Select purchase date" />
        </Form.Item>

        <Form.Item name="purchasePrice" label="Purchase price">
          <InputNumber style={{ width: '100%' }} min={0} precision={2} placeholder="Purchase price" />
        </Form.Item>

        <Form.Item name="expiryDate" label="Expiration date">
          <DatePicker style={{ width: '100%' }} placeholder="Select expiration date" />
        </Form.Item>

        <Form.Item name="renewalCost" label="Renewal cost">
          <Input placeholder="Renewal cost" />
        </Form.Item>

        <Divider>Support information</Divider>

        <Form.Item name="supportVendor" label="Support Supplier">
          <Input placeholder="Support Supplier" />
        </Form.Item>

        <Form.Item name="supportContact" label="Support Contact">
          <Input placeholder="Support Contact" />
        </Form.Item>

        <Divider>Other</Divider>

        <Form.Item name="notes" label="Notes">
          <TextArea rows={3} placeholder="Remarks information" />
        </Form.Item>

        <Form.Item name="tags" label="Tags">
          <Input placeholder="Tags, separated by commas" />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" icon={<Save />} loading={loading}>
              {isEdit ? 'Save' : 'Create'}
            </Button>
            <Button onClick={() => router.push('/licenses')}>Cancel</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default LicenseForm;
