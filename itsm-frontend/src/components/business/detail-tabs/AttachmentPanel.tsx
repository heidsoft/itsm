'use client';

import React, { useCallback, useEffect, useState } from 'react';
import {
  Upload,
  Button,
  App,
  List,
  Typography,
  Progress,
  Modal,
  Space,
  Empty,
  Spin,
  Alert,
} from 'antd';
import type { UploadFile, RcFile } from 'antd/es/upload/interface';
import {
  File as FileIcon,
  Image as ImageIcon,
  FileText,
  Music,
  Video,
  Archive,
  Download,
  Eye,
  Trash2,
  Upload as UploadIcon,
} from 'lucide-react';
import type { AttachmentAdapter, AttachmentItem, TargetType } from './types';

const { Text } = Typography;

export interface AttachmentPanelProps {
  targetType: TargetType;
  targetId: number | string;
  adapter: AttachmentAdapter;
  maxSize?: number; // bytes，默认 50MB
  accept?: string;
  currentUserId?: number;
  formatDateTime?: (dateString: string) => string;
}

const defaultFormat = (s: string) => (s ? new Date(s).toLocaleString('zh-CN') : '');

const formatFileSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
};

const getFileIcon = (mimeType: string) => {
  if (!mimeType) return <FileIcon size={20} />;
  if (mimeType.startsWith('image/')) return <ImageIcon size={20} />;
  if (mimeType.startsWith('video/')) return <Video size={20} />;
  if (mimeType.startsWith('audio/')) return <Music size={20} />;
  if (mimeType.includes('pdf') || mimeType.includes('word') || mimeType.includes('excel'))
    return <FileText size={20} />;
  if (mimeType.includes('zip') || mimeType.includes('rar')) return <Archive size={20} />;
  return <FileIcon size={20} />;
};

export const AttachmentPanel: React.FC<AttachmentPanelProps> = ({
  targetId,
  adapter,
  maxSize = 50 * 1024 * 1024,
  accept,
  currentUserId,
  formatDateTime = defaultFormat,
}) => {
  const { message } = App.useApp();
  const [items, setItems] = useState<AttachmentItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [uploadProgress, setUploadProgress] = useState<number>(0);
  const [uploading, setUploading] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewName, setPreviewName] = useState<string>('');

  const fetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await adapter.list(targetId);
      setItems(res || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载附件失败');
    } finally {
      setLoading(false);
    }
  }, [adapter, targetId]);

  useEffect(() => {
    void fetch();
  }, [fetch]);

  const beforeUpload = (file: RcFile) => {
    if (file.size > maxSize) {
      message.error(`文件大小超过 ${formatFileSize(maxSize)} 限制`);
      return Upload.LIST_IGNORE;
    }
    return true;
  };

  const customRequest = async (options: {
    file: File | RcFile | Blob;
    onSuccess?: (response: unknown) => void;
    onError?: (err: Error) => void;
  }) => {
    const file = options.file as File;
    setUploading(true);
    setUploadProgress(0);
    try {
      await adapter.upload(targetId, file, (p) => setUploadProgress(p));
      options.onSuccess?.({});
      message.success(`${file.name} 上传成功`);
      await fetch();
    } catch (e) {
      const err = e instanceof Error ? e : new Error('上传失败');
      options.onError?.(err);
      message.error(err.message);
    } finally {
      setUploading(false);
      setUploadProgress(0);
    }
  };

  const handleDelete = (item: AttachmentItem) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除附件 "${item.fileName}" 吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await adapter.remove(targetId, item.id);
          message.success('删除成功');
          await fetch();
        } catch (e) {
          message.error(e instanceof Error ? e.message : '删除失败');
        }
      },
    });
  };

  const handleDownload = (item: AttachmentItem) => {
    const url = adapter.getDownloadUrl(targetId, item.id);
    // 通过 <a download> 触发浏览器下载
    const a = document.createElement('a');
    a.href = url;
    a.download = item.fileName;
    a.target = '_blank';
    a.rel = 'noreferrer';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  const handlePreview = (item: AttachmentItem) => {
    if (!adapter.getPreviewUrl) {
      message.warning('该附件不支持预览');
      return;
    }
    setPreviewUrl(adapter.getPreviewUrl(targetId, item.id));
    setPreviewName(item.fileName);
  };

  const isPreviewable = (mime: string) =>
    mime.startsWith('image/') || mime.includes('pdf') || mime.startsWith('text/');

  const canManage = (a: AttachmentItem) =>
    currentUserId ? a.uploader?.id === currentUserId : true;

  if (loading && items.length === 0) {
    return (
      <div className="p-6 text-center">
        <Spin />
      </div>
    );
  }

  return (
    <div className="p-6">
      {error && (
        <Alert
          message={error}
          type="error"
          showIcon
          closable
          className="mb-4"
          onClose={() => setError(null)}
          action={
            <Button size="small" type="link" onClick={() => void fetch()}>
              重试
            </Button>
          }
        />
      )}

      <div className="mb-6">
        <Upload
          multiple
          showUploadList={false}
          beforeUpload={beforeUpload}
          customRequest={customRequest as never}
          accept={accept}
        >
          <Button icon={<UploadIcon size={14} />} loading={uploading}>
            上传附件
          </Button>
        </Upload>
        {uploading && (
          <div className="mt-3">
            <Progress percent={Math.round(uploadProgress)} size="small" />
          </div>
        )}
        <Text type="secondary" className="ml-3 text-xs">
          单文件最大 {formatFileSize(maxSize)}
        </Text>
      </div>

      {items.length === 0 ? (
        <Empty description="暂无附件" />
      ) : (
        <List
          itemLayout="horizontal"
          dataSource={items}
          renderItem={(item) => (
            <List.Item
              actions={[
                adapter.getPreviewUrl && isPreviewable(item.mimeType) ? (
                  <Button
                    key="preview"
                    type="link"
                    size="small"
                    icon={<Eye size={14} />}
                    onClick={() => handlePreview(item)}
                  >
                    预览
                  </Button>
                ) : null,
                <Button
                  key="download"
                  type="link"
                  size="small"
                  icon={<Download size={14} />}
                  onClick={() => handleDownload(item)}
                >
                  下载
                </Button>,
                canManage(item) ? (
                  <Button
                    key="delete"
                    type="link"
                    size="small"
                    danger
                    icon={<Trash2 size={14} />}
                    onClick={() => handleDelete(item)}
                  >
                    删除
                  </Button>
                ) : null,
              ].filter(Boolean) as React.ReactNode[]}
            >
              <List.Item.Meta
                avatar={getFileIcon(item.mimeType)}
                title={item.fileName}
                description={
                  <Space size="small" wrap>
                    <Text type="secondary" className="text-xs">
                      {formatFileSize(item.fileSize)}
                    </Text>
                    <Text type="secondary" className="text-xs">
                      {item.uploader?.name || item.uploader?.username || '未知'}
                    </Text>
                    <Text type="secondary" className="text-xs">
                      {formatDateTime(item.createdAt)}
                    </Text>
                  </Space>
                }
              />
            </List.Item>
          )}
        />
      )}

      <Modal
        title={previewName}
        open={!!previewUrl}
        onCancel={() => setPreviewUrl(null)}
        footer={null}
        width={900}
      >
        {previewUrl && (
          <iframe
            src={previewUrl}
            style={{ width: '100%', height: '70vh', border: 0 }}
            title={previewName}
          />
        )}
      </Modal>
    </div>
  );
};

export default AttachmentPanel;
