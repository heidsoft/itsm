'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Select,
  App,
  Modal,
  Descriptions,
  Divider,
} from 'antd';
import { Trash2, Eye, AlertTriangle, AlertCircle, CheckCircle, XCircle } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { PageContainer } from '@/components/layout/PageContainer';
import { ChangeApi, type PIRResponse, type PIROverallResult } from '@/lib/api/change-api';
import { useI18n } from '@/lib/i18n/useI18n';
import dayjs from 'dayjs';
import type { ColumnsType } from 'antd/es/table';

export default function PIRListPage() {
  const router = useRouter();
  const { t } = useI18n();
  const { message } = App.useApp();

  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空表格。
  const [listLoading, setListLoading] = useState(true);
  const [pirs, setPirs] = useState<PIRResponse[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [resultFilter, setResultFilter] = useState<PIROverallResult | '全部' | undefined>(undefined);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedPIR, setSelectedPIR] = useState<PIRResponse | null>(null);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [deleting, setDeleting] = useState(false);

  // 翻页/切过滤会并发触发多次请求，慢的旧响应不得覆盖新条件的结果。
  const listRequest = useRef(0);

  const fetchPIRs = useCallback(async () => {
    const seq = ++listRequest.current;
    setListLoading(true);
    try {
      const response = await ChangeApi.getPIRs({
        page,
        pageSize: pageSize,
        result: resultFilter,
      });
      if (seq !== listRequest.current) return;
      setPirs(response.items);
      setTotal(response.total);
    } catch (error) {
      if (seq !== listRequest.current) return;
      console.error('Failed to fetch PIRs:', error);
      message.error('Failed to obtain PIR list');
    } finally {
      if (seq === listRequest.current) setListLoading(false);
    }
  }, [page, pageSize, resultFilter]);

  useEffect(() => {
    fetchPIRs();
  }, [fetchPIRs]);

  const handleDelete = async () => {
    if (!selectedPIR) return;
    setDeleting(true);
    try {
      await ChangeApi.deletePIR(selectedPIR.id);
      message.success('PIR deleted');
      setDeleteModalVisible(false);
      setSelectedPIR(null);
      fetchPIRs();
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

  const columns: ColumnsType<PIRResponse> = [
    {
      title: 'Change ID',
      dataIndex: 'changeId',
      key: 'changeId',
      width: 100,
      render: (id: number) => (
        <Button type="link" onClick={() => router.push(`/changes/${id}`)}>
          C-{id}
        </Button>
      ),
    },
    {
      title: 'Change title',
      dataIndex: 'changeTitle',
      key: 'changeTitle',
      ellipsis: true,
    },
    {
      title: 'Reviewer',
      dataIndex: 'reviewerName',
      key: 'reviewerName',
      width: 120,
    },
    {
      title: 'Review date',
      dataIndex: 'reviewDate',
      key: 'reviewDate',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Overall results',
      dataIndex: 'overallResult',
      key: 'overallResult',
      width: 120,
      render: (result: PIROverallResult) => getResultTag(result),
    },
    {
      title: 'Goal achieved',
      dataIndex: 'objectivesAchieved',
      key: 'objectivesAchieved',
      width: 100,
      render: (achieved: boolean) =>
        achieved ? <Tag color="success">Yes</Tag> : <Tag color="error">No</Tag>,
    },
    {
      title: 'Rollback',
      dataIndex: 'rollbackPerformed',
      key: 'rollbackPerformed',
      width: 80,
      render: (performed: boolean) => (performed ? <Tag color="warning">Yes</Tag> : <Tag>No</Tag>),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            icon={<Eye />}
            onClick={() => {
              setSelectedPIR(record);
              setDetailModalVisible(true);
            }}
          >

            View
          </Button>
          <Button
            type="link"
            danger
            icon={<Trash2 />}
            onClick={() => {
              setSelectedPIR(record);
              setDeleteModalVisible(true);
            }}
          >

            Delete
          </Button>
        </Space>
      ),
    },
  ];

  const resultOptions = [
    { value: '全部', label: 'All results' },
    { value: 'successful', label: 'Success' },
    { value: 'partially_successful', label: 'Partially successful' },
    { value: 'failed', label: 'Failed' },
  ];

  return (
    <PageContainer title="Post-Implementation Review (PIR)" description="Review record after change implementation">
      <Card className="shadow-sm rounded-lg">
        <div className="mb-4">
          <Space wrap>
            <Select
              placeholder="Filter results"
              value={resultFilter}
              onChange={setResultFilter}
              allowClear
              options={resultOptions}
              style={{ width: 150 }}
            />
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={pirs}
          rowKey="id"
          loading={listLoading}
          pagination={{
            current: page,
            pageSize: pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => `Total ${total} records`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      {/* PIR详情弹窗 */}
      <Modal
        title="PIR details"
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false);
          setSelectedPIR(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>

            Close
          </Button>,
          <Button
            key="edit"
            type="primary"
            onClick={() => {
              setDetailModalVisible(false);
              if (selectedPIR) {
                router.push(`/changes/${selectedPIR.changeId}/pir`);
              }
            }}
          >

            Edit
          </Button>,
        ]}
        width={700}
      >
        {selectedPIR && (
          <div>
            <Descriptions column={2} bordered>
              <Descriptions.Item label="Change ID">
                <Button type="link" onClick={() => router.push(`/changes/${selectedPIR.changeId}`)}>
                  C-{selectedPIR.changeId}
                </Button>
              </Descriptions.Item>
              <Descriptions.Item label="Change title">{selectedPIR.changeTitle}</Descriptions.Item>
              <Descriptions.Item label="Reviewer">{selectedPIR.reviewerName}</Descriptions.Item>
              <Descriptions.Item label="Review date">
                {dayjs(selectedPIR.reviewDate).format('YYYY-MM-DD HH:mm')}
              </Descriptions.Item>
              <Descriptions.Item label="Overall results">
                {getResultTag(selectedPIR.overallResult as PIROverallResult)}
              </Descriptions.Item>
              <Descriptions.Item label="Goal achieved">
                {selectedPIR.objectivesAchieved ? (
                  <Tag color="success">Yes</Tag>
                ) : (
                  <Tag color="error">No</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="Actual duration">
                {selectedPIR.actualDurationMinutes}  minutes
              </Descriptions.Item>
              <Descriptions.Item label="Rollback">
                {selectedPIR.rollbackPerformed ? <Tag color="warning">Yes</Tag> : <Tag>No</Tag>}
              </Descriptions.Item>
            </Descriptions>

            {selectedPIR.successSummary && (
              <>
                <Divider>Successful summary</Divider>
                <p>{selectedPIR.successSummary}</p>
              </>
            )}

            {selectedPIR.issuesEncountered && (
              <>
                <Divider>Problem encountered</Divider>
                <p>{selectedPIR.issuesEncountered}</p>
              </>
            )}

            {selectedPIR.lessonsLearned && (
              <>
                <Divider>Lessons Learned</Divider>
                <p>{selectedPIR.lessonsLearned}</p>
              </>
            )}

            {selectedPIR.improvementRecommendations && (
              <>
                <Divider>ImprovementSuggestion</Divider>
                <p>{selectedPIR.improvementRecommendations}</p>
              </>
            )}

            {selectedPIR.rollbackReason && (
              <>
                <Divider>Rollback Reason</Divider>
                <p>{selectedPIR.rollbackReason}</p>
              </>
            )}
          </div>
        )}
      </Modal>

      {/* 删除确认弹窗 */}
      <Modal
        title={
          <Space>
            <AlertCircle style={{ color: '#faad14' }} />

            Confirm Delete
          </Space>
        }
        open={deleteModalVisible}
        onOk={handleDelete}
        onCancel={() => {
          setDeleteModalVisible(false);
          setSelectedPIR(null);
        }}
        okText="Delete"
        okButtonProps={{ danger: true, loading: deleting }}
        cancelText="Cancel"
      >
        <p>Are you sure you want to delete this post-implementation review record? This action cannot be undone.</p>
      </Modal>
    </PageContainer>
  );
}
