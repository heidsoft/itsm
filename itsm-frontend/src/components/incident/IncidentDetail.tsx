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

import { IncidentAPI } from '@/lib/api/';
import {
  IncidentStatus,
  IncidentStatusLabels,
  IncidentPriorityLabels,
  IncidentSeverityLabels,
} from '@/constants/incident';
import type { Incident } from '@/types/biz/incident';
import { useErrorHandler } from '@/lib/hooks/useErrorHandler';
import { SafeContent, SafeTextBlock } from '@/components/common/SafeContent';
import { isValidIncidentTransition } from '@/lib/utils/workflow-state-machine';

const IncidentDetail: React.FC<{ id?: string }> = ({ id: propId }) => {
  const params = useParams();
  const router = useRouter();
  // 支持通过props传入id，或通过useParams获取
  const id = propId || (params?.id as string);
  const { handleError } = useErrorHandler();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Incident | null>(null);
  const [escalateModalVisible, setEscalateModalVisible] = useState(false);
  const [resolveModalVisible, setResolveModalVisible] = useState(false);
  const [escalating, setEscalating] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [form] = Form.useForm();
  const [resolveForm] = Form.useForm();

  const loadData = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const resp = await IncidentAPI.getIncident(Number(id));
      setData(resp as unknown as Incident);
    } catch (error) {
      handleError(error, 'loadIncident', '加载事件详情失败');
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
      escalationLevel: (data?.escalationLevel || 0) + 1,
      reason: '',
      auto_assign: true,
    });
    setEscalateModalVisible(true);
  };

  const handleEscalateSubmit = async (values: any) => {
    if (!data) return;
    setEscalating(true);
    try {
      await IncidentAPI.escalateIncident(data.id, {
        escalationLevel: values.escalationLevel || values.escalation_level,
        reason: values.reason,
        autoAssign: values.autoAssign || values.auto_assign,
      });
      message.success('事件升级成功');
      setEscalateModalVisible(false);
      loadData(); // 刷新数据
    } catch (error) {
      handleError(error, 'escalateIncident', '升级失败');
    } finally {
      setEscalating(false);
    }
  };

  // 打开解决确认弹窗
  const handleResolveClick = () => {
    resolveForm.resetFields();
    setResolveModalVisible(true);
  };

  // 提交解决方案（ITIL 合规：要求填写解决方案）
  const handleResolveSubmit = async (values: { resolution: string; resolutionCode?: string }) => {
    if (!data) return;

    // 状态转换验证
    if (!isValidIncidentTransition(data.status, 'resolved')) {
      message.error('当前状态不允许直接解决');
      return;
    }

    setResolving(true);
    try {
      // 使用专门的 resolve 端点，而非直接更新状态
      await IncidentAPI.resolveIncident(data.id, {
        resolution: values.resolution,
        resolution_code: values.resolutionCode,
      });
      message.success('事件已解决');
      setResolveModalVisible(false);
      loadData();
    } catch (error) {
      handleError(error, 'resolveIncident', '解决失败');
    } finally {
      setResolving(false);
    }
  };

  if (loading) {
    return (
      <Card>
        <Skeleton active />
      </Card>
    );
  }

  if (!data) {
    return <Card>未找到事件</Card>;
  }

  return (
    <>
      <Space orientation="vertical" style={{ width: '100%' }} size="middle">
        {/* 头部操作栏 */}
        <Card styles={{ body: { padding: '16px 24px' } }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <span style={{ fontSize: 20, fontWeight: 500, marginRight: 16 }}>
                {data.incidentNumber} {data.title}
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
              <Button icon={<ArrowUpOutlined />} onClick={handleEscalate} loading={escalating}>
                升级
              </Button>
              {data.status !== IncidentStatus.RESOLVED && (
                <Button
                  type="primary"
                  icon={<CheckCircleOutlined />}
                  onClick={handleResolveClick}
                  loading={resolving}
                >
                  解决
                </Button>
              )}
            </Space>
          </div>
        </Card>

        {/* 基本信息 */}
        <Card title="基本信息">
          <Descriptions column={2}>
            <Descriptions.Item label="报告人ID">{data.reporterId}</Descriptions.Item>
            <Descriptions.Item label="负责人ID">{data.assigneeId || '-'}</Descriptions.Item>
            <Descriptions.Item label="优先级">
              {IncidentPriorityLabels[data.priority]}
            </Descriptions.Item>
            <Descriptions.Item label="严重程度">
              {IncidentSeverityLabels[data.severity]}
            </Descriptions.Item>
            <Descriptions.Item label="分类">{data.category}</Descriptions.Item>
            <Descriptions.Item label="子分类">{data.subcategory}</Descriptions.Item>
            <Descriptions.Item label="检测时间">
              {dayjs(data.detectedAt).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
            <Descriptions.Item label="来源">{data.source}</Descriptions.Item>
          </Descriptions>
          <Divider />
          <Descriptions title="详细描述" column={1}>
            <Descriptions.Item label="描述">
              <SafeTextBlock content={data.description} fallback="暂无描述" />
            </Descriptions.Item>
          </Descriptions>

          {/* 影响分析 (如果有) */}
          {data.impactAnalysis && (
            <>
              <Divider />
              <Descriptions title="影响分析" column={1}>
                <Descriptions.Item>
                  <pre>{JSON.stringify(data.impactAnalysis, null, 2)}</pre>
                </Descriptions.Item>
              </Descriptions>
            </>
          )}
        </Card>

        {/* 解决记录 (如果有) */}
        {data.resolutionSteps && data.resolutionSteps.length > 0 && (
          <Card title="处理流程">
            <Timeline>
              {data.resolutionSteps.map((step, index) => (
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

      {/* 解决确认弹窗 (ITIL 合规：要求填写解决方案) */}
      <Modal
        title={
          <Space>
            <CheckCircleOutlined style={{ color: '#52c41a' }} />
            解决事件
          </Space>
        }
        open={resolveModalVisible}
        onCancel={() => setResolveModalVisible(false)}
        confirmLoading={resolving}
        onOk={() => resolveForm.submit()}
        okText="确认解决"
        cancelText="取消"
        width={500}
      >
        <Form form={resolveForm} layout="vertical" onFinish={handleResolveSubmit}>
          <Form.Item
            name="resolution"
            label="解决方案"
            rules={[
              { required: true, message: '请填写解决方案' },
              { min: 10, message: '解决方案至少需要10个字符' },
            ]}
          >
            <Input.TextArea
              rows={4}
              placeholder="请详细描述问题的解决方案和处理步骤..."
              showCount
              maxLength={2000}
            />
          </Form.Item>
          <Form.Item name="resolutionCode" label="解决分类">
            <Select placeholder="选择解决分类（可选）">
              <Select.Option value="fixed">已修复</Select.Option>
              <Select.Option value="workaround">临时解决方案</Select.Option>
              <Select.Option value="no_action">无需操作</Select.Option>
              <Select.Option value="third_party">第三方解决</Select.Option>
              <Select.Option value="user_error">用户错误</Select.Option>
            </Select>
          </Form.Item>
          {data?.problemId && (
            <div style={{ padding: '8px 12px', background: '#f5f5f5', borderRadius: 4 }}>
              <ExclamationCircleOutlined style={{ marginRight: 8, color: '#faad14' }} />
              <span>此事件已关联问题记录 #{data.problemId}</span>
            </div>
          )}
        </Form>
      </Modal>
    </>
  );
};

export default IncidentDetail;
