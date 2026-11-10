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
  App,
  Divider,
  Tag,
  Descriptions,
  Modal,
  Alert,
  Spin,
  Typography,
} from 'antd';
import { Undo, AlertTriangle, AlertCircle, CheckCircle, XCircle } from 'lucide-react';
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
import dayjs, { type Dayjs } from 'dayjs';

const { TextArea } = Input;
const { Text } = Typography;

export default function PIRPage() {
  const params = useParams();
  const router = useRouter();
  const { t } = useI18n();
  const { message } = App.useApp();
  const changeId = Number(params.id);

  // 加载初值必须为 true：mount 必定发起首次拉取。existingPIR 决定表单标题
  // （创建 vs 编辑），初值 false 会让首帧先显示「创建实施后审查」再翻转。
  const [listLoading, setListLoading] = useState(true);
  const [pir, setPIR] = useState<PIRResponse | null>(null);
  const [existingPIR, setExistingPIR] = useState(false);
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const fetchPIR = useCallback(async () => {
    setListLoading(true);
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
      setListLoading(false);
    }
  }, [changeId, form]);

  useEffect(() => {
    fetchPIR();
  }, [fetchPIR]);

  const handleSubmit = async (values: Record<string, unknown>) => {
    setSubmitting(true);
    try {
      const request: CreatePIRRequest = {
        overallResult: values.overallResult as PIROverallResult,
        objectivesAchieved: values.objectivesAchieved as boolean,
        successSummary: values.successSummary as string,
        issuesEncountered: values.issuesEncountered as string,
        lessonsLearned: values.lessonsLearned as string,
        improvementRecommendations: values.improvementRecommendations as string,
        actualStartTime: (values.actualStartTime as Dayjs)?.toISOString(),
        actualEndTime: (values.actualEndTime as Dayjs)?.toISOString(),
        rollbackPerformed: values.rollbackPerformed as boolean,
        rollbackReason: values.rollbackReason as string,
      };

      if (existingPIR && pir) {
        // Update existing PIR
        const updateRequest: UpdatePIRRequest = {
          overallResult: values.overallResult as PIROverallResult,
          objectivesAchieved: values.objectivesAchieved as boolean,
          successSummary: values.successSummary as string,
          issuesEncountered: values.issuesEncountered as string,
          lessonsLearned: values.lessonsLearned as string,
          improvementRecommendations: values.improvementRecommendations as string,
        };
        const updated = await ChangeApi.updatePIR(pir.id, updateRequest);
        setPIR(updated);
        message.success('PIR updated');
      } else {
        // Create new PIR
        const created = await ChangeApi.createPIR(changeId, request);
        setPIR(created);
        setExistingPIR(true);
        message.success('PIR has been created');
      }
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : 'Operation failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!pir) return;
    setDeleting(true);
    try {
      await ChangeApi.deletePIR(pir.id);
      message.success('PIR deleted');
      setPIR(null);
      setExistingPIR(false);
      form.resetFields();
      setDeleteModalVisible(false);
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : 'Delete failed');
    } finally {
      setDeleting(false);
    }
  };

  const getResultTag = (result: PIROverallResult) => {
    switch (result) {
      case 'successful':
        return (
          <Tag icon={<CheckCircle />} color="success">

            Success
          </Tag>
        );
      case 'partially_successful':
        return (
          <Tag icon={<AlertTriangle />} color="warning">

            Partially successful
          </Tag>
        );
      case 'failed':
        return (
          <Tag icon={<XCircle />} color="error">

            Failed
          </Tag>
        );
      default:
        return <Tag>{result}</Tag>;
    }
  };

  return (
    <PageContainer
      title="Post-Implementation Review (PIR)"
      description="Evaluate the implementation results and capture lessons learned"
      extra={<Button onClick={() => router.push(`/changes/${changeId}`)}>Return change details</Button>}
    >
      <div className="space-y-6">
        {/* 加载期不渲染表单：existingPIR 决定标题是「创建」还是「编辑」，
            表单字段也要等回填完成才有意义，首帧渲染必然产生错态。 */}
        {listLoading ? (
          <Card className="shadow-sm rounded-lg">
            <Spin className="flex justify-center py-12" />
          </Card>
        ) : (
          <>
            {pir && (
              <Card title="PIR Overview" className="shadow-sm rounded-lg">
                <Descriptions column={2} bordered>
                  <Descriptions.Item label="Change ID">{pir.changeId}</Descriptions.Item>
                  <Descriptions.Item label="Reviewer">{pir.reviewerName}</Descriptions.Item>
                  <Descriptions.Item label="Review date">
                    {dayjs(pir.reviewDate).format('YYYY-MM-DD HH:mm')}
                  </Descriptions.Item>
                  <Descriptions.Item label="Overall results">
                    {getResultTag(pir.overallResult as PIROverallResult)}
                  </Descriptions.Item>
                  <Descriptions.Item label="Goal achieved">
                    {pir.objectivesAchieved ? (
                      <Tag color="success">Yes</Tag>
                    ) : (
                      <Tag color="error">No</Tag>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="Actual duration">
                    {pir.actualDurationMinutes} minutes
                  </Descriptions.Item>
                  {pir.rollbackPerformed && (
                    <Descriptions.Item label="Rollback">
                      <Tag icon={<Undo />} color="warning">
                        Rolled Back
                      </Tag>
                    </Descriptions.Item>
                  )}
                </Descriptions>
              </Card>
            )}

            <Card
              title={existingPIR ? "Edit post-implementation review" : "Create a post-implementation review"}
              className="shadow-sm rounded-lg"
            >
              <Alert
                type="info"
                showIcon
                className="mb-5"
                title="Just fill in the four parts: results, problems, experiences and suggestions for improvement. It is not necessary to write a long article for each item."
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
                <Divider>Basic Assessment</Divider>

                <Form.Item
                  name="overallResult"
                  label="Overall results"
                  rules={[{ required: true, message: "Please select overall results" }]}
                >
                  <Select placeholder="Select implementation results">
                    <Select.Option value="successful">
                      <Space>
                        <CheckCircle style={{ color: '#52c41a' }} />
                        Success - the change was implemented exactly as planned and achieved the expected goals
                      </Space>
                    </Select.Option>
                    <Select.Option value="partially_successful">
                      <Space>
                        <AlertTriangle style={{ color: '#faad14' }} />
                        Partially successful - Change implemented with some issues
                      </Space>
                    </Select.Option>
                    <Select.Option value="failed">
                      <Space>
                        <XCircle style={{ color: '#ff4d4f' }} />
                        Failure - the change did not achieve the desired goal or needs to be rolled back
                      </Space>
                    </Select.Option>
                  </Select>
                </Form.Item>

                <Form.Item name="objectivesAchieved" label="Is the goal achieved?">
                  <Select
                    placeholder="Choose whether to achieve"
                    options={[
                      { label: "Yes", value: true },
                      { label: "No", value: false },
                    ]}
                  />
                </Form.Item>

                <Divider>Detailed Assessment</Divider>

                <div className="mb-4">
                  <Text type="secondary">It is recommended that each control be limited to 1 to 3 sentences, giving priority to describing the facts and impact.</Text>
                </div>

                <Form.Item name="successSummary" label="Successful summary">
                  <TextArea
                    rows={4}
                    placeholder="What works and why?"
                    showCount
                    maxLength={300}
                  />
                </Form.Item>

                <Form.Item name="issuesEncountered" label="Problem encountered">
                  <TextArea rows={4} placeholder="What obstacles were encountered during implementation?" showCount maxLength={300} />
                </Form.Item>

                <Form.Item name="lessonsLearned" label="Lessons Learned">
                  <TextArea
                    rows={4}
                    placeholder="What is the most valuable experience from this time that is worth reusing or avoiding?"
                    showCount
                    maxLength={300}
                  />
                </Form.Item>

                <Form.Item name="improvementRecommendations" label="ImprovementSuggestion">
                  <TextArea rows={4} placeholder="How can we make it more stable and faster next time?" showCount maxLength={300} />
                </Form.Item>

                <Divider>Implementation time</Divider>

                <Space size="large" wrap>
                  <Form.Item name="actualStartTime" label="Actual start time">
                    <DatePicker showTime format="YYYY-MM-DD HH:mm" />
                  </Form.Item>

                  <Form.Item name="actualEndTime" label="Actual end time">
                    <DatePicker showTime format="YYYY-MM-DD HH:mm" />
                  </Form.Item>
                </Space>

                <Divider>Rollback information</Divider>

                <Form.Item name="rollbackPerformed" label="Whether a rollback was performed">
                  <Select
                    placeholder="Choose whether to roll back"
                    options={[
                      { label: "Yes", value: true },
                      { label: "No", value: false },
                    ]}
                  />
                </Form.Item>

                <Form.Item
                  name="rollbackReason"
                  label="Rollback Reason"
                  dependencies={['rollbackPerformed']}
                  hidden={!form.getFieldValue('rollbackPerformed')}
                >
                  <TextArea
                    rows={3}
                    placeholder="If a rollback is performed, please explain the triggering reason and the disposition result."
                    showCount
                    maxLength={240}
                  />
                </Form.Item>

                <Divider />

                <Form.Item>
                  <Space>
                    <Button type="primary" htmlType="submit" loading={submitting}>
                      {existingPIR ? "Update PIR" : "Create PIR"}
                    </Button>
                    <Button onClick={() => router.push(`/changes/${changeId}`)}>Cancel</Button>
                    {existingPIR && (
                      <Button danger onClick={() => setDeleteModalVisible(true)}>
                        Delete PIR
                      </Button>
                    )}
                  </Space>
                </Form.Item>
              </Form>
            </Card>
          </>
        )}
      </div>

      <Modal
        title={
          <Space>
            <AlertCircle style={{ color: '#faad14' }} />

            Confirm Delete
          </Space>
        }
        open={deleteModalVisible}
        onOk={handleDelete}
        onCancel={() => setDeleteModalVisible(false)}
        okText="Delete"
        okButtonProps={{ danger: true, loading: deleting }}
        cancelText="Cancel"
      >
        <p>Are you sure you want to delete this post-implementation review record? This action cannot be undone.</p>
      </Modal>
    </PageContainer>
  );
}
