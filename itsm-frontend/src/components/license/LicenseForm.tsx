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
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons';

import { AssetApi, License, LicenseRequest } from '@/lib/api/asset-api';

const { TextArea } = Input;
const { Option } = Select;

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
      message.error('加载许可证详情失败');
    } finally {
      setLoading(false);
    }
  };

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const data: LicenseRequest = {
        name: values.name,
        description: values.description,
        vendor: values.vendor,
        license_type: values.license_type,
        total_quantity: values.total_quantity,
        asset_id: values.asset_id,
        purchase_date: values.purchase_date,
        purchase_price: values.purchase_price,
        expiry_date: values.expiry_date,
        support_vendor: values.support_vendor,
        support_contact: values.support_contact,
        renewal_cost: values.renewal_cost,
        notes: values.notes,
        tags: values.tags,
      };

      if (isEdit) {
        await AssetApi.updateLicense(Number(id), data);
        message.success('更新成功');
      } else {
        await AssetApi.createLicense(data);
        message.success('创建成功');
      }
      router.push('/licenses');
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
          license_type: 'subscription',
          total_quantity: 1,
        }}
      >
        <div style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/licenses')}>
            返回列表
          </Button>
        </div>

        <Divider>基本信息</Divider>

        <Form.Item
          name="name"
          label="许可证名称"
          rules={[{ required: true, message: '请输入许可证名称' }]}
        >
          <Input placeholder="例如: Microsoft 365 E3" />
        </Form.Item>

        <Form.Item name="description" label="描述">
          <TextArea rows={3} placeholder="许可证描述" />
        </Form.Item>

        <Form.Item name="vendor" label="供应商">
          <Input placeholder="例如: Microsoft" />
        </Form.Item>

        <Form.Item name="license_type" label="许可证类型">
          <Select>
            <Option value="perpetual">永久 (Perpetual)</Option>
            <Option value="subscription">订阅 (Subscription)</Option>
            <Option value="per-user">按用户 (Per-User)</Option>
            <Option value="per-seat">按席位 (Per-Seat)</Option>
            <Option value="site">站点 (Site)</Option>
          </Select>
        </Form.Item>

        <Form.Item name="license_key" label="许可证密钥">
          <TextArea rows={2} placeholder="许可证密钥" />
        </Form.Item>

        <Divider>数量与使用</Divider>

        <Form.Item name="total_quantity" label="总数量">
          <InputNumber min={1} style={{ width: '100%' }} />
        </Form.Item>

        <Divider>采购与财务</Divider>

        <Form.Item name="purchase_date" label="采购日期">
          <DatePicker style={{ width: '100%' }} placeholder="选择采购日期" />
        </Form.Item>

        <Form.Item name="purchase_price" label="采购价格">
          <InputNumber style={{ width: '100%' }} min={0} precision={2} placeholder="采购价格" />
        </Form.Item>

        <Form.Item name="expiry_date" label="到期日期">
          <DatePicker style={{ width: '100%' }} placeholder="选择到期日期" />
        </Form.Item>

        <Form.Item name="renewal_cost" label="续费成本">
          <Input placeholder="续费成本" />
        </Form.Item>

        <Divider>支持信息</Divider>

        <Form.Item name="support_vendor" label="支持供应商">
          <Input placeholder="支持供应商" />
        </Form.Item>

        <Form.Item name="support_contact" label="支持联系方式">
          <Input placeholder="支持联系方式" />
        </Form.Item>

        <Divider>其他</Divider>

        <Form.Item name="notes" label="备注">
          <TextArea rows={3} placeholder="备注信息" />
        </Form.Item>

        <Form.Item name="tags" label="标签">
          <Input placeholder="标签，用逗号分隔" />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={loading}>
              {isEdit ? '保存' : '创建'}
            </Button>
            <Button onClick={() => router.push('/licenses')}>取消</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default LicenseForm;
