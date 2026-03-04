'use client';

/**
 * 事件详情组件
 */

import React, { useState, useEffect } from 'react';
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Space,
  Timeline,
  Skeleton,
  message,
  Divider,
  Modal,
  Form,
  Select,
  Input,
} from 'antd';
import {
  EditOutlined,
  ArrowUpOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useRouter, useParams } from 'next/navigation';
import dayjs from 'dayjs';

import { IncidentApi, IncidentAPI } from '@/lib/api/';
import {
  IncidentStatus,
  IncidentStatusLabels,
  IncidentPriorityLabels,
  IncidentSeverityLabels,
} from '@/constants/incident';
import type { Incident } from '@/types/biz/incident';

const IncidentDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Incident | null>(null);
  const [escalateModalVisible, setEscalateModalVisible] = useState(false);
  const [escalating, setEscalating] = useState(false);
  const [form] = Form.useForm();

  const loadData = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const resp = await IncidentApi.getIncident(Number(id));
      setData(resp as unknown as Incident);
    } catch (error) {
      // console.error(error);
      message.error('加载事件详情失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const handleEscalate = () => {
    form.setFieldsValue({
      escalation_level: (data?.escalation_level || 0) + 1,
      reason: '',
      auto_assign: true,
    });
    setEscalateModalVisible(true);
  };

  const handleEscalateSubmit = async (values: unknown) => {
    if (!data) return;
    setEscalating(true);
    try {
      await IncidentAPI.escalateIncident({
        incident_id: data.id,
        escalation_level: values.escalation_level,
        reason: values.reason,
        auto_assign: values.auto_assign,
      });
      message.success('事件升级成功');
      setEscalateModalVisible(false);
      loadData(); // 刷新数据
    } catch (error) {
      message.error('升级失败');
    } finally {
      setEscalating(false);
    }
  };

  if (loading) {
    return (
      <Card variant="borderless">
        <Skeleton active />
      </Card>
    );
  }

  if (!data) {
    return <Card variant="borderless">未找到事件</Card>;
  }

  return (
    <>
      <Space orientation="vertical" style={{ width: '100%' }} size="middle">
        {/* 头部操作栏 */}
        <Card variant="borderless" bodyStyle={{ padding: '16px 24px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <span style={{ fontSize: 20, fontWeight: 500, marginRight: 16 }}>
                {data.incident_number} {data.title}
              </span>
              <Tag color={data.status === IncidentStatus.RESOLVED ? 'success' : 'blue'}>
                {IncidentStatusLabels[data.status]}
              </Tag>
            </div>
            <Space>
              <Button
                icon={<EditOutlined />}
                onClick={() => router.push(`/incidents/${data.id}/edit`)}
              >
                编辑
              </Button>
              <Button icon={<ArrowUpOutlined />} onClick={handleEscalate}>
                升级
              </Button>
              {data.status !== IncidentStatus.RESOLVED && (
                <Button type="primary" icon={<CheckCircleOutlined />}>
                  解决
                </Button>
              )}
            </Space>
          </div>
        </Card>

        {/* 基本信息 */}
        <Card variant="borderless" title="基本信息">
          <Descriptions column={2}>
            <Descriptions.Item label="报告人ID">{data.reporter_id}</Descriptions.Item>
            <Descriptions.Item label="负责人ID">{data.assignee_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="优先级">
              {IncidentPriorityLabels[data.priority]}
            </Descriptions.Item>
            <Descriptions.Item label="严重程度">
              {IncidentSeverityLabels[data.severity]}
            </Descriptions.Item>
            <Descriptions.Item label="分类">{data.category}</Descriptions.Item>
            <Descriptions.Item label="子分类">{data.subcategory}</Descriptions.Item>
            <Descriptions.Item label="检测时间">
              {dayjs(data.detected_at).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
            <Descriptions.Item label="来源">{data.source}</Descriptions.Item>
          </Descriptions>
          <Divider />
          <Descriptions title="详细描述" column={1}>
            <Descriptions.Item label="描述">{data.description}</Descriptions.Item>
          </Descriptions>

          {/* 影响分析 (如果有) */}
          {data.impact_analysis && (
            <>
              <Divider />
              <Descriptions title="影响分析" column={1}>
                <Descriptions.Item>
                  <pre>{JSON.stringify(data.impact_analysis, null, 2)}</pre>
                </Descriptions.Item>
              </Descriptions>
            </>
          )}
        </Card>

        {/* 解决记录 (如果有) */}
        {data.resolution_steps && data.resolution_steps.length > 0 && (
          <Card variant="borderless" title="处理流程">
            <Timeline>
              {data.resolution_steps.map((step, index) => (
                <Timeline.Item key={index}>
                  <p>{(step as any).description || '处理步骤'}</p>
                  <span style={{ fontSize: '12px', color: '#999' }}>{(step as any).timestamp}</span>
                </Timeline.Item>
              ))}
            </Timeline>
          </Card>
        )}
      </Space>

      {escalateModalVisible && (
        <Modal
          title="升级事件"
          open={escalateModalVisible}
          onCancel={() => setEscalateModalVisible(false)}
          confirmLoading={escalating}
          onOk={() => form.submit()}
        >
          <Form form={form} layout="vertical" onFinish={handleEscalateSubmit}>
            <Form.Item
              name="escalation_level"
              label="升级级别"
              rules={[{ required: true, message: '请选择升级级别' }]}
            >
              <Select placeholder="请选择升级级别">
                <Select.Option value={1}>级别 1 - 主管</Select.Option>
                <Select.Option value={2}>级别 2 - 经理</Select.Option>
                <Select.Option value={3}>级别 3 - 总监</Select.Option>
                <Select.Option value={4}>级别 4 - VP</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item
              name="reason"
              label="升级原因"
              rules={[{ required: true, message: '请输入升级原因' }]}
            >
              <Input.TextArea rows={3} placeholder="请输入升级原因" />
            </Form.Item>
            <Form.Item name="auto_assign" label="自动分配">
              <Select placeholder="是否自动分配给上级">
                <Select.Option value={true}>是</Select.Option>
                <Select.Option value={false}>否</Select.Option>
              </Select>
            </Form.Item>
          </Form>
        </Modal>
      )}
    </>
  );
};

export default IncidentDetail;
