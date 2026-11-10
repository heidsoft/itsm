'use client';

/**
 * 资产创建/编辑表单组件
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
  Tooltip,
} from 'antd';
import { useRouter, useParams } from 'next/navigation';
import { ArrowLeft, Save } from 'lucide-react';

import type { Asset, AssetRequest } from '@/lib/api/asset-api';
import { AssetApi } from '@/lib/api/asset-api';
import type { Dayjs } from 'dayjs';

const { TextArea } = Input;

// 表单验证规则
const formRules = {
  assetNumber: [
    { required: true, message: 'Please enter the asset number' },
    { pattern: /^[A-Z0-9-]+$/, message: 'Asset number can only contain uppercase letters, numbers and hyphens' },
    { min: 3, message: 'Asset number must be at least 3 characters' },
    { max: 50, message: 'Asset number cannot exceed 50 characters' },
  ],
  assetName: [
    { required: true, message: 'Please enter asset name' },
    { min: 2, message: 'Asset name must be at least 2 characters long' },
    { max: 200, message: 'Asset name cannot exceed 200 characters' },
  ],
  serialNumber: [{ max: 100, message: 'Serial number cannot exceed 100 characters' }],
  purchasePrice: [
    {
      type: 'number' as const,
      min: 0,
      max: 999999999,
      message: 'Purchase price must be between 0 and 999,999,999',
    },
  ],
};

// 表单字段的help文本
const fieldHelps = {
  assetNumber: 'Format: ASSET-001',
  assetName: 'Friendly name of the asset',
  description: 'Detailed description of the asset’s purpose and characteristics',
  assetType: 'Select the main type of asset',
  category: 'Used for sorting and filtering',
  serialNumber: 'Unique identifier provided by the manufacturer',
  model: 'Product model',
  manufacturer: 'Manufacturer name',
  vendor: 'Supplier name',
  purchaseDate: 'Purchase completion date',
  purchasePrice: 'Price at time of purchase',
  warrantyExpiry: 'Warranty service expiration date',
  supportExpiry: 'Technical support expiration date',
  location: 'Physical storage location',
  department: 'Department or team to which you belong',
  tags: 'Separate multiple tags with commas',
};

const AssetForm: React.FC = () => {
  const router = useRouter();
  const { id } = useParams() as { id: string };
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<Asset | null>(null);
  const isEdit = !!id;

  useEffect(() => {
    if (id) {
      loadDetail();
    }
  }, [id]);

  const loadDetail = async () => {
    setLoading(true);
    try {
      const data = await AssetApi.getAsset(Number(id));
      setDetail(data);
      form.setFieldsValue(data);
    } catch (error) {
      message.error('Failed to load asset details');
    } finally {
      setLoading(false);
    }
  };

  const onFinish = async (values: {
    assetNumber: string;
    name: string;
    description?: string;
    type: string;
    category?: string;
    subcategory?: string;
    ciId?: number;
    serialNumber?: string;
    model?: string;
    manufacturer?: string;
    vendor?: string;
    purchaseDate?: Dayjs;
    purchasePrice?: number;
    warrantyExpiry?: Dayjs;
    supportExpiry?: Dayjs;
    location?: string;
    department?: string;
    tags?: string[];
  }) => {
    setLoading(true);
    try {
      const data: AssetRequest = {
        assetNumber: values.assetNumber,
        name: values.name,
        description: values.description,
        type: values.type as Asset['type'],
        category: values.category,
        subcategory: values.subcategory,
        ciId: values.ciId,
        serialNumber: values.serialNumber,
        model: values.model,
        manufacturer: values.manufacturer,
        vendor: values.vendor,
        purchaseDate: values.purchaseDate?.toISOString(),
        purchasePrice: values.purchasePrice,
        warrantyExpiry: values.warrantyExpiry?.toISOString(),
        supportExpiry: values.supportExpiry?.toISOString(),
        location: values.location,
        department: values.department,
        tags: values.tags,
      };

      if (isEdit) {
        await AssetApi.updateAsset(Number(id), data);
        message.success('Updated');
      } else {
        await AssetApi.createAsset(data);
        message.success('Created');
      }
      router.push('/assets');
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
          type: 'hardware',
          status: 'available',
        }}
      >
        <div style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeft />} onClick={() => router.push('/assets')}>

            Back to List
          </Button>
        </div>

        <Divider>Basic Info</Divider>

        <Form.Item
          name="assetNumber"
          label={
            <Tooltip title={fieldHelps.assetNumber}>
              <span>Asset Number</span>
            </Tooltip>
          }
          rules={formRules.assetNumber}
          help={fieldHelps.assetNumber}
        >
          <Input placeholder="Example: ASSET-001" autoComplete="off" />
        </Form.Item>

        <Form.Item
          name="name"
          label={
            <Tooltip title={fieldHelps.assetName}>
              <span>Asset Name</span>
            </Tooltip>
          }
          rules={formRules.assetName}
          help={fieldHelps.assetName}
        >
          <Input placeholder="Asset Name" autoComplete="off" />
        </Form.Item>

        <Form.Item
          name="description"
          label={
            <Tooltip title={fieldHelps.description}>
              <span>Description</span>
            </Tooltip>
          }
          help={fieldHelps.description}
        >
          <TextArea rows={3} placeholder="Asset description" maxLength={1000} showCount />
        </Form.Item>

        <Form.Item
          name="type"
          label={
            <Tooltip title={fieldHelps.assetType}>
              <span>Asset Type</span>
            </Tooltip>
          }
          help={fieldHelps.assetType}
        >
          <Select options={[
            { value: 'hardware', label: 'Hardware' },
            { value: 'software', label: 'Software' },
            { value: 'cloud', label: 'Cloud Resources' },
            { value: 'license', label: 'Licenses' },
          ]} />
        </Form.Item>

        <Form.Item
          name="category"
          label={
            <Tooltip title={fieldHelps.category}>
              <span>Category</span>
            </Tooltip>
          }
          help={fieldHelps.category}
        >
          <Input placeholder="Asset Classification" />
        </Form.Item>

        <Form.Item name="subcategory" label="Subcategory">
          <Input placeholder="Asset subcategory" />
        </Form.Item>

        <Divider>Hardware information</Divider>

        <Form.Item
          name="serialNumber"
          label={
            <Tooltip title={fieldHelps.serialNumber}>
              <span>Serial Number</span>
            </Tooltip>
          }
          rules={formRules.serialNumber}
          help={fieldHelps.serialNumber}
        >
          <Input placeholder="Serial Number" />
        </Form.Item>

        <Form.Item name="model" label="Model">
          <Input placeholder="Model" />
        </Form.Item>

        <Form.Item name="manufacturer" label="Manufacturer">
          <Input placeholder="Manufacturer" />
        </Form.Item>

        <Form.Item name="vendor" label="Supplier">
          <Input placeholder="Supplier" />
        </Form.Item>

        <Divider>Purchasing and Finance</Divider>

        <Form.Item
          name="purchaseDate"
          label={
            <Tooltip title={fieldHelps.purchaseDate}>
              <span>Purchase date</span>
            </Tooltip>
          }
          help={fieldHelps.purchaseDate}
        >
          <DatePicker style={{ width: '100%' }} placeholder="Select purchase date" />
        </Form.Item>

        <Form.Item
          name="purchasePrice"
          label={
            <Tooltip title={fieldHelps.purchasePrice}>
              <span>Purchase price</span>
            </Tooltip>
          }
          rules={formRules.purchasePrice}
          help={fieldHelps.purchasePrice}
        >
          <InputNumber style={{ width: '100%' }} min={0} precision={2} placeholder="Purchase price" />
        </Form.Item>

        <Form.Item
          name="warrantyExpiry"
          label={
            <Tooltip title={fieldHelps.warrantyExpiry}>
              <span>Warranty expired</span>
            </Tooltip>
          }
          help={fieldHelps.warrantyExpiry}
        >
          <DatePicker style={{ width: '100%' }} placeholder="Select warranty expiration date" />
        </Form.Item>

        <Form.Item
          name="supportExpiry"
          label={
            <Tooltip title={fieldHelps.supportExpiry}>
              <span>Support period expired</span>
            </Tooltip>
          }
          help={fieldHelps.supportExpiry}
        >
          <DatePicker style={{ width: '100%' }} placeholder="Select support expiration date" />
        </Form.Item>

        <Divider>Location and Belonging</Divider>

        <Form.Item
          name="location"
          label={
            <Tooltip title={fieldHelps.location}>
              <span>Physical location</span>
            </Tooltip>
          }
          help={fieldHelps.location}
        >
          <Input placeholder="Physical location" />
        </Form.Item>

        <Form.Item
          name="department"
          label={
            <Tooltip title={fieldHelps.department}>
              <span>Department</span>
            </Tooltip>
          }
          help={fieldHelps.department}
        >
          <Input placeholder="Department" />
        </Form.Item>

        <Form.Item
          name="tags"
          label={
            <Tooltip title={fieldHelps.tags}>
              <span>Tags</span>
            </Tooltip>
          }
          help={fieldHelps.tags}
        >
          <Input placeholder="Tags, separated by commas" />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" icon={<Save />} loading={loading}>
              {isEdit ? 'Save' : 'Create'}
            </Button>
            <Button onClick={() => router.push('/assets')}>Cancel</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default AssetForm;
