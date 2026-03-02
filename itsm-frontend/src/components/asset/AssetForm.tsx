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
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons';

import { AssetApi, Asset, AssetRequest } from '@/lib/api/asset-api';

const { TextArea } = Input;
const { Option } = Select;

// 表单验证规则
const formRules = {
  assetNumber: [
    { required: true, message: '请输入资产编号' },
    { pattern: /^[A-Z0-9-]+$/, message: '资产编号只能包含大写字母、数字和连字符' },
    { min: 3, message: '资产编号至少3个字符' },
    { max: 50, message: '资产编号不能超过50个字符' },
  ],
  assetName: [
    { required: true, message: '请输入资产名称' },
    { min: 2, message: '资产名称至少2个字符' },
    { max: 200, message: '资产名称不能超过200个字符' },
  ],
  serialNumber: [{ max: 100, message: '序列号不能超过100个字符' }],
  purchasePrice: [
    {
      type: 'number' as const,
      min: 0,
      max: 999999999,
      message: '采购价格必须在 0 到 999,999,999 之间',
    },
  ],
};

// 表单字段的help文本
const fieldHelps = {
  assetNumber: '格式如：ASSET-001',
  assetName: '资产的友好名称',
  description: '详细描述资产的用途和特点',
  assetType: '选择资产的主要类型',
  category: '用于分类和筛选',
  serialNumber: '厂商提供的唯一标识',
  model: '产品型号',
  manufacturer: '生产厂商名称',
  vendor: '供应商名称',
  purchaseDate: '采购完成日期',
  purchasePrice: '采购时的价格',
  warrantyExpiry: '保修服务到期日期',
  supportExpiry: '技术支持到期日期',
  location: '物理存放位置',
  department: '所属部门或团队',
  tags: '用逗号分隔多个标签',
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
      message.error('加载资产详情失败');
    } finally {
      setLoading(false);
    }
  };

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const data: AssetRequest = {
        asset_number: values.asset_number,
        name: values.name,
        description: values.description,
        type: values.type,
        category: values.category,
        subcategory: values.subcategory,
        ci_id: values.ci_id,
        serial_number: values.serial_number,
        model: values.model,
        manufacturer: values.manufacturer,
        vendor: values.vendor,
        purchase_date: values.purchase_date,
        purchase_price: values.purchase_price,
        warranty_expiry: values.warranty_expiry,
        support_expiry: values.support_expiry,
        location: values.location,
        department: values.department,
        tags: values.tags,
      };

      if (isEdit) {
        await AssetApi.updateAsset(Number(id), data);
        message.success('更新成功');
      } else {
        await AssetApi.createAsset(data);
        message.success('创建成功');
      }
      router.push('/assets');
    } catch (error) {
      message.error(isEdit ? '更新失败' : '创建失败');
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
          <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/assets')}>
            返回列表
          </Button>
        </div>

        <Divider>基本信息</Divider>

        <Form.Item
          name="asset_number"
          label={
            <Tooltip title={fieldHelps.assetNumber}>
              <span>资产编号</span>
            </Tooltip>
          }
          rules={formRules.assetNumber}
          help={fieldHelps.assetNumber}
        >
          <Input placeholder="例如: ASSET-001" autoComplete="off" />
        </Form.Item>

        <Form.Item
          name="name"
          label={
            <Tooltip title={fieldHelps.assetName}>
              <span>资产名称</span>
            </Tooltip>
          }
          rules={formRules.assetName}
          help={fieldHelps.assetName}
        >
          <Input placeholder="资产名称" autoComplete="off" />
        </Form.Item>

        <Form.Item
          name="description"
          label={
            <Tooltip title={fieldHelps.description}>
              <span>描述</span>
            </Tooltip>
          }
          help={fieldHelps.description}
        >
          <TextArea rows={3} placeholder="资产描述" maxLength={1000} showCount />
        </Form.Item>

        <Form.Item
          name="type"
          label={
            <Tooltip title={fieldHelps.assetType}>
              <span>资产类型</span>
            </Tooltip>
          }
          help={fieldHelps.assetType}
        >
          <Select>
            <Option value="hardware">硬件</Option>
            <Option value="software">软件</Option>
            <Option value="cloud">云资源</Option>
            <Option value="license">许可证</Option>
          </Select>
        </Form.Item>

        <Form.Item
          name="category"
          label={
            <Tooltip title={fieldHelps.category}>
              <span>分类</span>
            </Tooltip>
          }
          help={fieldHelps.category}
        >
          <Input placeholder="资产分类" />
        </Form.Item>

        <Form.Item name="subcategory" label="子分类">
          <Input placeholder="资产子分类" />
        </Form.Item>

        <Divider>硬件信息</Divider>

        <Form.Item
          name="serial_number"
          label={
            <Tooltip title={fieldHelps.serialNumber}>
              <span>序列号</span>
            </Tooltip>
          }
          rules={formRules.serialNumber}
          help={fieldHelps.serialNumber}
        >
          <Input placeholder="序列号" />
        </Form.Item>

        <Form.Item name="model" label="型号">
          <Input placeholder="型号" />
        </Form.Item>

        <Form.Item name="manufacturer" label="制造商">
          <Input placeholder="制造商" />
        </Form.Item>

        <Form.Item name="vendor" label="供应商">
          <Input placeholder="供应商" />
        </Form.Item>

        <Divider>采购与财务</Divider>

        <Form.Item
          name="purchase_date"
          label={
            <Tooltip title={fieldHelps.purchaseDate}>
              <span>采购日期</span>
            </Tooltip>
          }
          help={fieldHelps.purchaseDate}
        >
          <DatePicker style={{ width: '100%' }} placeholder="选择采购日期" />
        </Form.Item>

        <Form.Item
          name="purchase_price"
          label={
            <Tooltip title={fieldHelps.purchasePrice}>
              <span>采购价格</span>
            </Tooltip>
          }
          rules={formRules.purchasePrice}
          help={fieldHelps.purchasePrice}
        >
          <InputNumber style={{ width: '100%' }} min={0} precision={2} placeholder="采购价格" />
        </Form.Item>

        <Form.Item
          name="warranty_expiry"
          label={
            <Tooltip title={fieldHelps.warrantyExpiry}>
              <span>保修期到期</span>
            </Tooltip>
          }
          help={fieldHelps.warrantyExpiry}
        >
          <DatePicker style={{ width: '100%' }} placeholder="选择保修到期日期" />
        </Form.Item>

        <Form.Item
          name="support_expiry"
          label={
            <Tooltip title={fieldHelps.supportExpiry}>
              <span>支持期到期</span>
            </Tooltip>
          }
          help={fieldHelps.supportExpiry}
        >
          <DatePicker style={{ width: '100%' }} placeholder="选择支持到期日期" />
        </Form.Item>

        <Divider>位置与归属</Divider>

        <Form.Item
          name="location"
          label={
            <Tooltip title={fieldHelps.location}>
              <span>物理位置</span>
            </Tooltip>
          }
          help={fieldHelps.location}
        >
          <Input placeholder="物理位置" />
        </Form.Item>

        <Form.Item
          name="department"
          label={
            <Tooltip title={fieldHelps.department}>
              <span>所属部门</span>
            </Tooltip>
          }
          help={fieldHelps.department}
        >
          <Input placeholder="所属部门" />
        </Form.Item>

        <Form.Item
          name="tags"
          label={
            <Tooltip title={fieldHelps.tags}>
              <span>标签</span>
            </Tooltip>
          }
          help={fieldHelps.tags}
        >
          <Input placeholder="标签，用逗号分隔" />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={loading}>
              {isEdit ? '保存' : '创建'}
            </Button>
            <Button onClick={() => router.push('/assets')}>取消</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default AssetForm;
