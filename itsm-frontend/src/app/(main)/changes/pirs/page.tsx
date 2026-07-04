'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Select,
  message,
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
import { Trash2, Eye, AlertTriangle, AlertCircle, CheckCircle, XCircle } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';

export default function PIRListPage() {
  const router = useRouter();
  const { t } = useI18n();

  const [loading, setLoading] = useState(false);
  const [pirs, setPirs] = useState<PIRResponse[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [resultFilter, setResultFilter] = useState<string | undefined>(undefined);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedPIR, setSelectedPIR] = useState<PIRResponse | null>(null);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const fetchPIRs = useCallback(async () => {
    setLoading(true);
    try {
      const response = await ChangeApi.getPIRs({
        page,
        page_size: pageSize,
        result: resultFilter as any,
      });
      setPirs(response.items || []);
      setTotal(response.total || 0);
    } catch (error) {
      console.error('Failed to fetch PIRs:', error);
      message.error('获取PIR列表失败');
    } finally {
      setLoading(false);
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
      message.success('PIR已删除');
      setDeleteModalVisible(false);
      setSelectedPIR(null);
      fetchPIRs();
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
          <Tag icon={<CheckCircle />} color="success">
            成功
          </Tag>
        );
      case 'partially_successful':
        return (
          <Tag icon={<AlertTriangle />} color="warning">
            部分成功
          </Tag>
        );
      case 'failed':
        return (
          <Tag icon={<XCircle />} color="error">
            失败
          </Tag>
        );
      default:
        return <Tag>{result}</Tag>;
    }
  };

  const columns: ColumnsType<PIRResponse> = [
    {
      title: '变更ID',
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
      title: '变更标题',
      dataIndex: 'changeTitle',
      key: 'changeTitle',
      ellipsis: true,
    },
    {
      title: '审查人',
      dataIndex: 'reviewerName',
      key: 'reviewerName',
      width: 120,
    },
    {
      title: '审查日期',
      dataIndex: 'reviewDate',
      key: 'reviewDate',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '总体结果',
      dataIndex: 'overallResult',
      key: 'overallResult',
      width: 120,
      render: (result: PIROverallResult) => getResultTag(result),
    },
    {
      title: '目标达成',
      dataIndex: 'objectivesAchieved',
      key: 'objectivesAchieved',
      width: 100,
      render: (achieved: boolean) =>
        achieved ? <Tag color="success">是</Tag> : <Tag color="error">否</Tag>,
    },
    {
      title: '回滚',
      dataIndex: 'rollbackPerformed',
      key: 'rollbackPerformed',
      width: 80,
      render: (performed: boolean) => (performed ? <Tag color="warning">是</Tag> : <Tag>否</Tag>),
    },
    {
      title: '操作',
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
            查看
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
            删除
          </Button>
        </Space>
      ),
    },
  ];

  const resultOptions = [
    { value: '全部', label: '全部结果' },
    { value: 'successful', label: '成功' },
    { value: 'partially_successful', label: '部分成功' },
    { value: 'failed', label: '失败' },
  ];

  return (
    <PageContainer title="实施后审查列表 (PIR)" description="查看所有变更的实施后审查记录">
      <Card className="shadow-sm rounded-lg">
        <div className="mb-4">
          <Space wrap>
            <Select
              placeholder="结果筛选"
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
          loading={loading}
          pagination={{
            current: page,
            pageSize: pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => `共 ${total} 条记录`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      {/* PIR详情弹窗 */}
      <Modal
        title="PIR详情"
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false);
          setSelectedPIR(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
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
            编辑
          </Button>,
        ]}
        width={700}
      >
        {selectedPIR && (
          <div>
            <Descriptions column={2} bordered>
              <Descriptions.Item label="变更ID">
                <Button type="link" onClick={() => router.push(`/changes/${selectedPIR.changeId}`)}>
                  C-{selectedPIR.changeId}
                </Button>
              </Descriptions.Item>
              <Descriptions.Item label="变更标题">{selectedPIR.changeTitle}</Descriptions.Item>
              <Descriptions.Item label="审查人">{selectedPIR.reviewerName}</Descriptions.Item>
              <Descriptions.Item label="审查日期">
                {dayjs(selectedPIR.reviewDate).format('YYYY-MM-DD HH:mm')}
              </Descriptions.Item>
              <Descriptions.Item label="总体结果">
                {getResultTag(selectedPIR.overallResult as PIROverallResult)}
              </Descriptions.Item>
              <Descriptions.Item label="目标达成">
                {selectedPIR.objectivesAchieved ? (
                  <Tag color="success">是</Tag>
                ) : (
                  <Tag color="error">否</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="实际持续时间">
                {selectedPIR.actualDurationMinutes} 分钟
              </Descriptions.Item>
              <Descriptions.Item label="回滚">
                {selectedPIR.rollbackPerformed ? <Tag color="warning">是</Tag> : <Tag>否</Tag>}
              </Descriptions.Item>
            </Descriptions>

            {selectedPIR.successSummary && (
              <>
                <Divider>成功总结</Divider>
                <p>{selectedPIR.successSummary}</p>
              </>
            )}

            {selectedPIR.issuesEncountered && (
              <>
                <Divider>遇到的问题</Divider>
                <p>{selectedPIR.issuesEncountered}</p>
              </>
            )}

            {selectedPIR.lessonsLearned && (
              <>
                <Divider>经验教训</Divider>
                <p>{selectedPIR.lessonsLearned}</p>
              </>
            )}

            {selectedPIR.improvementRecommendations && (
              <>
                <Divider>改进建议</Divider>
                <p>{selectedPIR.improvementRecommendations}</p>
              </>
            )}

            {selectedPIR.rollbackReason && (
              <>
                <Divider>回滚原因</Divider>
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
            确认删除
          </Space>
        }
        open={deleteModalVisible}
        onOk={handleDelete}
        onCancel={() => {
          setDeleteModalVisible(false);
          setSelectedPIR(null);
        }}
        okText="删除"
        okButtonProps={{ danger: true, loading: deleting }}
        cancelText="取消"
      >
        <p>确定要删除这个实施后审查记录吗？此操作不可撤销。</p>
      </Modal>
    </PageContainer>
  );
}
