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
              <Select>
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

            <Form.Item name="objectivesAchieved" label="目标是否达成" valuePropName="checked">
              <Select>
                <Select.Option value={true}>是</Select.Option>
                <Select.Option value={false}>否</Select.Option>
              </Select>
            </Form.Item>

            <Divider>详细评估</Divider>

            <Form.Item
              name="successSummary"
              label="成功总结"
              extra="描述变更实施过程中做得好的方面"
            >
              <TextArea rows={4} placeholder="描述成功实施的关键因素..." />
            </Form.Item>

            <Form.Item
              name="issuesEncountered"
              label="遇到的问题"
              extra="记录实施过程中遇到的问题和挑战"
            >
              <TextArea rows={4} placeholder="描述遇到的问题..." />
            </Form.Item>

            <Form.Item
              name="lessonsLearned"
              label="经验教训"
              extra="总结可以从这次变更中学到的经验"
            >
              <TextArea rows={4} placeholder="经验教训..." />
            </Form.Item>

            <Form.Item
              name="improvementRecommendations"
              label="改进建议"
              extra="为未来的变更提供改进建议"
            >
              <TextArea rows={4} placeholder="改进建议..." />
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

            <Form.Item name="rollbackPerformed" label="是否执行了回滚" valuePropName="checked">
              <Select>
                <Select.Option value={true}>是</Select.Option>
                <Select.Option value={false}>否</Select.Option>
              </Select>
            </Form.Item>

            <Form.Item
              name="rollbackReason"
              label="回滚原因"
              dependencies={['rollbackPerformed']}
              hidden={!form.getFieldValue('rollbackPerformed')}
            >
              <TextArea rows={3} placeholder="如果执行了回滚，请说明原因..." />
            </Form.Item>

            <Divider />

            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={submitting}>
                  {existingPIR ? '更新PIR' : '创建PIR'}
                </Button>
                {existingPIR && (
                  <Button danger onClick={() => setDeleteModalVisible(true)}>
                    删除PIR
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
