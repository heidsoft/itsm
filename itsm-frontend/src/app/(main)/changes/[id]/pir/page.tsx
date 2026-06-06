'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  DatePicker,
  Button,
  Space,
  message,
  Divider,
  Tag,
  Descriptions,
  Modal,
  Alert,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  RollbackOutlined,
} from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import { PageContainer } from '@/components/layout/PageContainer';
import {
  ChangeApi,
  type PIRResponse,
  type CreatePIRRequest,
  type UpdatePIRRequest,
  type PIROverallResult,
} from '@/lib/api/change-api';
import { useI18n } from '@/lib/i18n/useI18n';
import dayjs from 'dayjs';
import { ExclamationCircleOutlined } from '@ant-design/icons';

const { TextArea } = Input;
const { Text } = Typography;

export default function PIRPage() {
  const params = useParams();
  const router = useRouter();
  const { t } = useI18n();
  const changeId = Number(params.id);

  const [loading, setLoading] = useState(false);
  const [pir, setPIR] = useState<PIRResponse | null>(null);
  const [existingPIR, setExistingPIR] = useState(false);
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    fetchPIR();
  }, [fetchPIR]);
  const fetchPIR = useCallback(async () => {
    setLoading(true);
    try {
      const pirData = await ChangeApi.getPIR(changeId);
      setPIR(pirData);
      setExistingPIR(!!pirData);
      if (pirData) {
        form.setFieldsValue({
          overallResult: pirData.overallResult,
          objectivesAchieved: pirData.objectivesAchieved,
          successSummary: pirData.successSummary,
          issuesEncountered: pirData.issuesEncountered,
          lessonsLearned: pirData.lessonsLearned,
          improvementRecommendations: pirData.improvementRecommendations,
          actualStartTime: pirData.actualStartTime ? dayjs(pirData.actualStartTime) : undefined,
          actualEndTime: pirData.actualEndTime ? dayjs(pirData.actualEndTime) : undefined,
          rollbackPerformed: pirData.rollbackPerformed,
          rollbackReason: pirData.rollbackReason,
        });
      }
    } catch (error) {
      console.error('Failed to fetch PIR:', error);
    } finally {
      setLoading(false);
    }
  }, [changeId, form]);

  useEffect(() => {
    fetchPIR();
  }, [fetchPIR]);

  const handleSubmit = async (values: any) => {
    setSubmitting(true);
    try {
      const request: CreatePIRRequest = {
        overallResult: values.overallResult as PIROverallResult,
        objectivesAchieved: values.objectivesAchieved,
        successSummary: values.successSummary,
        issuesEncountered: values.issuesEncountered,
        lessonsLearned: values.lessonsLearned,
        improvementRecommendations: values.improvementRecommendations,
        actualStartTime: values.actualStartTime?.toISOString(),
        actualEndTime: values.actualEndTime?.toISOString(),
        rollbackPerformed: values.rollbackPerformed,
        rollbackReason: values.rollbackReason,
      };

      if (existingPIR && pir) {
        // Update existing PIR
        const updateRequest: UpdatePIRRequest = {
          overallResult: values.overallResult,
          objectivesAchieved: values.objectivesAchieved,
          successSummary: values.successSummary,
          issuesEncountered: values.issuesEncountered,
          lessonsLearned: values.lessonsLearned,
          improvementRecommendations: values.improvementRecommendations,
        };
        const updated = await ChangeApi.updatePIR(pir.id, updateRequest);
        setPIR(updated);
        message.success('PIR已更新');
      } else {
        // Create new PIR
        const created = await ChangeApi.createPIR(changeId, request);
        setPIR(created);
        setExistingPIR(true);
        message.success('PIR已创建');
      }
    } catch (error: any) {
      message.error(error?.message || '操作失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!pir) return;
    setDeleting(true);
    try {
      await ChangeApi.deletePIR(pir.id);
      message.success('PIR已删除');
      setPIR(null);
      setExistingPIR(false);
      form.resetFields();
      setDeleteModalVisible(false);
    } catch (error: any) {
      message.error(error?.message || '删除失败');
    } finally {
      setDeleting(false);
    }
  };

  const getResultTag = (result: PIROverallResult) => {
    switch (result) {
      case 'successful':
        return (
          <Tag icon={<CheckCircleOutlined />} color="success">
            成功
          </Tag>
        );
      case 'partially_successful':
        return (
          <Tag icon={<WarningOutlined />} color="warning">
            部分成功
          </Tag>
        );
      case 'failed':
        return (
          <Tag icon={<CloseCircleOutlined />} color="error">
            失败
          </Tag>
        );
      default:
        return <Tag>{result}</Tag>;
    }
  };

  return (
    <PageContainer
      title="实施后审查 (PIR)"
      description="评估变更实施结果，总结经验教训"
      extra={<Button onClick={() => router.push(`/changes/${changeId}`)}>返回变更详情</Button>}
    >
      <div className="space-y-6">
        {pir && (
          <Card title="PIR概览" className="shadow-sm rounded-lg">
            <Descriptions column={2} bordered>
              <Descriptions.Item label="变更ID">{pir.changeId}</Descriptions.Item>
              <Descriptions.Item label="审查人">{pir.reviewerName}</Descriptions.Item>
              <Descriptions.Item label="审查日期">
                {dayjs(pir.reviewDate).format('YYYY-MM-DD HH:mm')}
              </Descriptions.Item>
              <Descriptions.Item label="总体结果">
                {getResultTag(pir.overallResult as PIROverallResult)}
              </Descriptions.Item>
              <Descriptions.Item label="目标达成">
                {pir.objectivesAchieved ? (
                  <Tag color="success">是</Tag>
                ) : (
                  <Tag color="error">否</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="实际持续时间">
                {pir.actualDurationMinutes} 分钟
              </Descriptions.Item>
              {pir.rollbackPerformed && (
                <Descriptions.Item label="回滚">
                  <Tag icon={<RollbackOutlined />} color="warning">
                    已回滚
                  </Tag>
                </Descriptions.Item>
              )}
            </Descriptions>
          </Card>
        )}

        <Card
          title={existingPIR ? '编辑实施后审查' : '创建实施后审查'}
          className="shadow-sm rounded-lg"
        >
          <Alert
            type="info"
            showIcon
            className="mb-5"
            message="按结果、问题、经验和改进建议四部分填写即可，不必每项都写很长。"
          />
          <Form
            form={form}
            layout="vertical"
            onFinish={handleSubmit}
            initialValues={{
              overallResult: 'successful',
              objectivesAchieved: false,
              rollbackPerformed: false,
            }}
          >
            <Divider>基本评估</Divider>

            <Form.Item
              name="overallResult"
              label="总体结果"
              rules={[{ required: true, message: '请选择总体结果' }]}
            >
              <Select placeholder="选择实施结果">
                <Select.Option value="successful">
                  <Space>
                    <CheckCircleOutlined style={{ color: '#52c41a' }} />
                    成功 - 变更完全按计划实施，达到预期目标
                  </Space>
                </Select.Option>
                <Select.Option value="partially_successful">
                  <Space>
                    <WarningOutlined style={{ color: '#faad14' }} />
                    部分成功 - 变更实施但存在一些问题
                  </Space>
                </Select.Option>
                <Select.Option value="failed">
                  <Space>
                    <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                    失败 - 变更未能达到预期目标或需要回滚
                  </Space>
                </Select.Option>
              </Select>
            </Form.Item>

            <Form.Item name="objectivesAchieved" label="目标是否达成">
              <Select
                placeholder="选择是否达成"
                options={[
                  { label: '是', value: true },
                  { label: '否', value: false },
                ]}
              />
            </Form.Item>

            <Divider>详细评估</Divider>

            <div className="mb-4">
              <Text type="secondary">建议每项控制在 1 到 3 句话，优先描述事实和影响。</Text>
            </div>

            <Form.Item name="successSummary" label="成功总结">
              <TextArea
                rows={4}
                placeholder="哪些做法有效，为什么有效？"
                showCount
                maxLength={300}
              />
            </Form.Item>

            <Form.Item name="issuesEncountered" label="遇到的问题">
              <TextArea
                rows={4}
                placeholder="实施中遇到了什么阻碍？"
                showCount
                maxLength={300}
              />
            </Form.Item>

            <Form.Item name="lessonsLearned" label="经验教训">
              <TextArea
                rows={4}
                placeholder="这次最值得复用或避免的经验是什么？"
                showCount
                maxLength={300}
              />
            </Form.Item>

            <Form.Item name="improvementRecommendations" label="改进建议">
              <TextArea
                rows={4}
                placeholder="下次如何做得更稳、更快？"
                showCount
                maxLength={300}
              />
            </Form.Item>

            <Divider>实施时间</Divider>

            <Space size="large" wrap>
              <Form.Item name="actualStartTime" label="实际开始时间">
                <DatePicker showTime format="YYYY-MM-DD HH:mm" />
              </Form.Item>

              <Form.Item name="actualEndTime" label="实际结束时间">
                <DatePicker showTime format="YYYY-MM-DD HH:mm" />
              </Form.Item>
            </Space>

            <Divider>回滚信息</Divider>

            <Form.Item name="rollbackPerformed" label="是否执行了回滚">
              <Select
                placeholder="选择是否回滚"
                options={[
                  { label: '是', value: true },
                  { label: '否', value: false },
                ]}
              />
            </Form.Item>

            <Form.Item
              name="rollbackReason"
              label="回滚原因"
              dependencies={['rollbackPerformed']}
              hidden={!form.getFieldValue('rollbackPerformed')}
            >
              <TextArea
                rows={3}
                placeholder="如果执行了回滚，请说明触发原因和处置结果。"
                showCount
                maxLength={240}
              />
            </Form.Item>

            <Divider />

            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={submitting}>
                  {existingPIR ? '更新 PIR' : '创建 PIR'}
                </Button>
                <Button onClick={() => router.push(`/changes/${changeId}`)}>取消</Button>
                {existingPIR && (
                  <Button danger onClick={() => setDeleteModalVisible(true)}>
                    删除 PIR
                  </Button>
                )}
              </Space>
            </Form.Item>
          </Form>
        </Card>
      </div>

      <Modal
        title={
          <Space>
            <ExclamationCircleOutlined style={{ color: '#faad14' }} />
            确认删除
          </Space>
        }
        open={deleteModalVisible}
        onOk={handleDelete}
        onCancel={() => setDeleteModalVisible(false)}
        okText="删除"
        okButtonProps={{ danger: true, loading: deleting }}
        cancelText="取消"
      >
        <p>确定要删除这个实施后审查记录吗？此操作不可撤销。</p>
      </Modal>
    </PageContainer>
  );
}
