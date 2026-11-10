'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Typography,
  Tag,
  Select,
  Modal,
  Timeline,
  Alert,
  Tooltip,
  Popconfirm,
  Row,
  Col,
  Descriptions,
  message,
  Spin,
} from 'antd';
import {
  GitBranch,
  History,
  RotateCcw,
  Eye,
  GitMerge,
  GitCommit,
  User,
  Calendar,
  FileText,
  AlertTriangle,
  CheckCircle,
} from 'lucide-react';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import type { ArticleVersion } from '@/types/knowledge-base';
import { format } from 'date-fns';
import DOMPurify from 'dompurify';

const { Title, Text, Paragraph } = Typography;

interface ArticleVersionControlProps {
  articleId: string;
  currentVersion: number;
  onVersionChange?: (version: number) => void;
}

const ArticleVersionControl: React.FC<ArticleVersionControlProps> = ({
  articleId,
  currentVersion,
  onVersionChange,
}) => {
  const [versions, setVersions] = useState<ArticleVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [compareModalVisible, setCompareModalVisible] = useState(false);
  const [selectedVersions, setSelectedVersions] = useState<[number, number] | null>(null);
  const [compareResult, setCompareResult] = useState<any>(null);
  const [previewVersion, setPreviewVersion] = useState<ArticleVersion | null>(null);
  const [previewModalVisible, setPreviewModalVisible] = useState(false);

  // 加载版本历史
  useEffect(() => {
    loadVersions();
  }, [articleId]);

  const loadVersions = async () => {
    try {
      setLoading(true);
      const versionHistory = await KnowledgeBaseApi.getArticleVersions(articleId);
      setVersions(versionHistory.sort((a, b) => b.version - a.version));
    } catch (error) {
      message.error('Failed to load version history');
    } finally {
      setLoading(false);
    }
  };

  // 恢复到指定版本
  const handleRestoreVersion = async (version: number) => {
    try {
      Modal.confirm({
        title: 'Confirm recovery version',
        content: `Are you sure you want to revert to version ${version} ? This will create a new version.`,
        onOk: async () => {
          await KnowledgeBaseApi.restoreVersion(articleId, version);
          message.success('Version restored successfully');
          loadVersions();
          onVersionChange?.(version);
        },
      });
    } catch (error) {
      message.error('Failed to restore version');
    }
  };

  // 比较版本
  const handleCompareVersions = async () => {
    if (!selectedVersions || selectedVersions[0] === selectedVersions[1]) {
      message.warning('Please select two different versions to compare');
      return;
    }

    try {
      setLoading(true);
      const result = await KnowledgeBaseApi.compareVersions(
        articleId,
        selectedVersions[0],
        selectedVersions[1]
      );
      setCompareResult(result);
      setCompareModalVisible(true);
    } catch (error) {
      message.error('Failed to compare versions');
    } finally {
      setLoading(false);
    }
  };

  // 预览版本
  const handlePreviewVersion = (version: ArticleVersion) => {
    setPreviewVersion(version);
    setPreviewModalVisible(true);
  };

  // 版本表格列
  const versionColumns = [
    {
      title: 'Version',
      dataIndex: 'version',
      key: 'version',
      width: 80,
      render: (version: number, record: ArticleVersion) => (
        <Space>
          <GitCommit className="w-4 h-4 text-blue-500" />
          <Tag color={version === currentVersion ? 'green' : 'default'}>
            v{version}
            {version === currentVersion && " (current)"}
          </Tag>
        </Space>
      ),
    },
    {
      title: 'Change description',
      dataIndex: 'changeLog',
      key: 'changeLog',
      render: (text: string) => (
        <Text ellipsis={{ tooltip: text }} style={{ maxWidth: 200 }}>
          {text || 'No change description'}
        </Text>
      ),
    },
    {
      title: 'Creator',
      dataIndex: 'createdByName',
      key: 'createdByName',
      render: (name: string) => (
        <Space>
          <User className="w-4 h-4 text-gray-500" />
          {name}
        </Space>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (date: string) => (
        <Space>
          <Calendar className="w-4 h-4 text-gray-500" />
          {format(new Date(date), 'yyyy-MM-dd HH:mm')}
        </Space>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 200,
      render: (_: unknown, record: ArticleVersion) => (
        <Space>
          <Tooltip title="Preview version">
            <Button
              aria-label="Preview version"
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => handlePreviewVersion(record)}
            />
          </Tooltip>

          {record.version !== currentVersion && (
            <Tooltip title="Revert to this version">
              <Popconfirm
                title="Confirm recovery version"
                description={`Are you sure you want to revert to version ${record.version} ?`}
                onConfirm={() => handleRestoreVersion(record.version)}
              >
                <Button aria-label="Revert to this version" type="text" icon={<RotateCcw className="w-4 h-4 text-orange-500" />} />
              </Popconfirm>
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  // 比较结果渲染
  const renderCompareResult = () => {
    if (!compareResult) return null;

    return (
      <div className="space-y-4">
        <Alert
          title="Version difference"
          description={`Compare versions ${selectedVersions?.[0]} and version ${selectedVersions?.[1]}`}
          type="info"
          showIcon
        />

        {compareResult.changes && compareResult.changes.length > 0 ? (
          <div>
            <Title level={5}>Change Detail</Title>
            {compareResult.changes.map((change: any, index: number) => (
              <Card key={index} size="small" className="mb-2">
                <Space>
                  <Tag
                    color={
                      change.type === 'added' ? 'green' : change.type === 'removed' ? 'red' : 'blue'
                    }
                  >
                    {change.type === 'added' ? 'New' : change.type === 'removed' ? 'Delete' : 'Modify'}
                  </Tag>
                  <Paragraph code style={{ margin: 0 }} ellipsis={{ rows: 3, expandable: true }}>
                    {change.content}
                  </Paragraph>
                </Space>
              </Card>
            ))}
          </div>
        ) : (
          <Alert title="No difference" description="No differences found between the two versions" type="success" showIcon />
        )}

        {compareResult.diff && (
          <div>
            <Title level={5}>Detailed differences</Title>
            <Card size="small">
              <pre className="whitespace-pre-wrap text-xs bg-gray-50 p-3 rounded">
                {compareResult.diff}
              </pre>
            </Card>
          </div>
        )}
      </div>
    );
  };

  return (
    <Card
      title={
        <Space>
          <History className="w-5 h-5 text-blue-600" />
          <span>Version Control</span>
        </Space>
      }
      extra={
        <Space>
          <Button
            icon={<GitMerge className="w-4 h-4" />}
            onClick={() => {
              if (versions.length >= 2) {
                setSelectedVersions([versions[0].version, versions[1].version]);
                setCompareModalVisible(true);
              } else {
                message.warning('At least two versions are required to compare');
              }
            }}
            disabled={versions.length < 2}
          >

            Compare versions
          </Button>
        </Space>
      }
    >
      {/* 版本历史表格 */}
      <Table
        columns={versionColumns}
        dataSource={versions}
        rowKey="version"
        loading={loading}
        scroll={{ x: 'max-content' }}
        pagination={false}
        size="small"
        className="version-table mb-4"
      />

      {/* 版本统计 */}
      <Row gutter={16} className="mt-4">
        <Col span={6}>
          <Card size="small">
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">{versions.length}</div>
              <Text type="secondary">Total version number</Text>
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <div className="text-center">
              <div className="text-2xl font-bold text-green-600">{currentVersion}</div>
              <Text type="secondary">Current Version</Text>
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <div className="text-center">
              <div className="text-2xl font-bold text-orange-600">
                {versions.filter(v => v.changeLog).length}
              </div>
              <Text type="secondary">There is a change record</Text>
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <div className="text-center">
              <div className="text-2xl font-bold text-purple-600">
                {versions.filter(v => v.version > currentVersion).length}
              </div>
              <Text type="secondary">Recoverable version</Text>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 版本比较模态框 */}
      <Modal
        title={
          <Space>
            <GitBranch className="w-5 h-5" />

            Version comparison
          </Space>
        }
        open={compareModalVisible}
        onCancel={() => {
          setCompareModalVisible(false);
          setCompareResult(null);
          setSelectedVersions(null);
        }}
        width={800}
        footer={null}
      >
        {!compareResult ? (
          <div className="space-y-4">
            <Title level={5}>Select Compare Version</Title>
            <Row gutter={16}>
              <Col span={12}>
                <Text strong>Source version</Text>
                <Select
                  style={{ width: '100%', marginTop: 8 }}
                  value={selectedVersions?.[0]}
                  onChange={value => setSelectedVersions([value, selectedVersions?.[1] || 0])}
                  placeholder="Select source version"
                  options={versions.map(version => ({
                    value: version.version,
                    label: `v${version.version} - ${format(new Date(version.createdAt), 'yyyy-MM-dd')}`,
                  }))}
                />
              </Col>
              <Col span={12}>
                <Text strong>target version</Text>
                <Select
                  style={{ width: '100%', marginTop: 8 }}
                  value={selectedVersions?.[1]}
                  onChange={value => setSelectedVersions([selectedVersions?.[0] || 0, value])}
                  placeholder="Select target version"
                  options={versions.map(version => ({
                    value: version.version,
                    label: `v${version.version} - ${format(new Date(version.createdAt), 'yyyy-MM-dd')}`,
                  }))}
                />
              </Col>
            </Row>
            <div className="text-center mt-4">
              <Button
                type="primary"
                onClick={handleCompareVersions}
                loading={loading}
                disabled={!selectedVersions || selectedVersions[0] === selectedVersions[1]}
              >

                Start comparing
              </Button>
            </div>
          </div>
        ) : (
          renderCompareResult()
        )}
      </Modal>

      {/* 版本预览模态框 */}
      <Modal
        title={
          <Space>
            <FileText className="w-5 h-5" />

            Version Preview - v{previewVersion?.version}
          </Space>
        }
        open={previewModalVisible}
        onCancel={() => {
          setPreviewModalVisible(false);
          setPreviewVersion(null);
        }}
        width={800}
        footer={[
          <Button key="close" onClick={() => setPreviewModalVisible(false)}>

            Close
          </Button>,
          previewVersion && previewVersion.version !== currentVersion && (
            <Button
              key="restore"
              type="primary"
              onClick={() => {
                handleRestoreVersion(previewVersion!.version);
                setPreviewModalVisible(false);
              }}
            >

              Revert to this version
            </Button>
          ),
        ]}
      >
        {previewVersion && (
          <div className="space-y-4">
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="Version">v{previewVersion.version}</Descriptions.Item>
              <Descriptions.Item label="Creator">{previewVersion.createdByName}</Descriptions.Item>
              <Descriptions.Item label="Created">
                {format(new Date(previewVersion.createdAt), 'yyyy-MM-dd HH:mm:ss')}
              </Descriptions.Item>
              <Descriptions.Item label="Change description">
                {previewVersion.changeLog || 'None'}
              </Descriptions.Item>
            </Descriptions>

            {previewVersion.summary && (
              <div>
                <Title level={5}>Summary</Title>
                <Paragraph>{previewVersion.summary}</Paragraph>
              </div>
            )}

            <div>
              <Title level={5}>Content Preview</Title>
              <Card size="small">
                <div
                  className="prose max-w-none"
                  dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(previewVersion.content) }}
                />
              </Card>
            </div>
          </div>
        )}
      </Modal>
    </Card>
  );
};

export default ArticleVersionControl;
