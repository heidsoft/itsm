'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Upload as AntUpload,
  Button,
  List,
  Space,
  Typography,
  Modal,
  Image as AntImage,
  Progress,
  message,
  Popconfirm,
  Tag,
  Tooltip,
  Empty,
} from 'antd';
import { Trash2, Download, File, Eye, Image, FileText, FileType, Sheet, Presentation, FileArchive } from 'lucide-react';
import type { UploadFile, UploadProps } from 'antd/es/upload/interface';
import type { TicketAttachment } from '@/lib/api/ticket-attachment-api';
import { TicketAttachmentApi } from '@/lib/api/ticket-attachment-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { App } from 'antd';
import { useI18n } from '@/lib/i18n';

const { Text } = Typography;

interface TicketAttachmentSectionProps {
  ticketId: number;
  canUpload?: boolean;
  canDelete?: (attachment: TicketAttachment) => boolean;
  onAttachmentUploaded?: (attachment: TicketAttachment) => void;
  onAttachmentDeleted?: (attachmentId: number) => void;
}

/**
 * 工单附件管理组件
 */
export const TicketAttachmentSection: React.FC<TicketAttachmentSectionProps> = ({
  ticketId,
  canUpload = true,
  canDelete = () => true,
  onAttachmentUploaded,
  onAttachmentDeleted,
}) => {
  const { t } = useI18n();
  const { message: antMessage } = App.useApp();
  const [attachments, setAttachments] = useState<TicketAttachment[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<Record<string, number>>({});
  const [previewVisible, setPreviewVisible] = useState(false);
  const [previewAttachment, setPreviewAttachment] = useState<TicketAttachment | null>(null);
  const [fileList, setFileList] = useState<UploadFile[]>([]);

  // 加载附件列表
  const loadAttachments = async () => {
    setLoading(true);
    try {
      const response = await TicketAttachmentApi.listAttachments(ticketId);
      setAttachments(response.attachments || []);
    } catch (error) {
      antMessage.error('加载附件列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (ticketId) {
      loadAttachments();
    }
  }, [ticketId]);

  // 上传文件
  const handleUpload: UploadProps['customRequest'] = async ({
    file,
    onSuccess,
    onError,
    onProgress,
  }) => {
    const uploadFile = file as File;
    setUploading(true);

    try {
      const attachment = await TicketAttachmentApi.uploadAttachment(
        ticketId,
        uploadFile,
        progress => {
          if (onProgress) {
            onProgress({ percent: progress });
          }
          setUploadProgress(prev => ({
            ...prev,
            [uploadFile.name]: progress,
          }));
        }
      );

      antMessage.success('附件上传成功');
      setFileList([]);
      setUploadProgress({});
      await loadAttachments();
      onAttachmentUploaded?.(attachment);
      if (onSuccess) {
        onSuccess(attachment);
      }
    } catch (error: any) {
      antMessage.error(error.message || '附件上传失败');
      if (onError) {
        onError(error);
      }
    } finally {
      setUploading(false);
    }
  };

  // 删除附件
  const handleDelete = async (attachment: TicketAttachment) => {
    try {
      await TicketAttachmentApi.deleteAttachment(ticketId, attachment.id);
      antMessage.success('附件删除成功');
      await loadAttachments();
      onAttachmentDeleted?.(attachment.id);
    } catch (error: any) {
      antMessage.error(error.message || '附件删除失败');
    }
  };

  // 预览附件
  const handlePreview = (attachment: TicketAttachment) => {
    // 判断是否为图片
    if (attachment.mimeType?.startsWith('image/')) {
      setPreviewAttachment(attachment);
      setPreviewVisible(true);
    } else {
      // 非图片文件，打开新窗口预览
      const previewUrl = TicketAttachmentApi.getPreviewUrl(ticketId, attachment.id);
      window.open(previewUrl, '_blank');
    }
  };

  // 下载附件
  const handleDownload = (attachment: TicketAttachment) => {
    const downloadUrl = TicketAttachmentApi.getDownloadUrl(ticketId, attachment.id);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = attachment.fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  // 获取文件图标
  const getFileIcon = (mimeType: string) => {
    const iconType = TicketAttachmentApi.getFileIconType(mimeType);
    switch (iconType) {
      case 'image':
        return <Image style={{ fontSize: 24, color: '#52c41a' }} />;
      case 'pdf':
        return <FileText style={{ fontSize: 24, color: '#ff4d4f' }} />;
      case 'word':
        return <FileType style={{ fontSize: 24, color: '#1890ff' }} />;
      case 'excel':
        return <Sheet style={{ fontSize: 24, color: '#52c41a' }} />;
      case 'powerpoint':
        return <Presentation style={{ fontSize: 24, color: '#ff9800' }} />;
      case 'archive':
        return <FileArchive style={{ fontSize: 24, color: '#722ed1' }} />;
      default:
        return <File style={{ fontSize: 24, color: '#8c8c8c' }} />;
    }
  };

  // 上传配置
  const uploadProps: UploadProps = {
    customRequest: handleUpload,
    fileList,
    onChange: ({ fileList: newFileList }) => {
      setFileList(newFileList);
    },
    beforeUpload: file => {
      // 检查文件大小（10MB）
      const maxSize = 10 * 1024 * 1024;
      if (file.size > maxSize) {
        antMessage.error('文件大小不能超过10MB');
        return false;
      }
      return true;
    },
    multiple: true,
    showUploadList: {
      showPreviewIcon: false,
      showRemoveIcon: true,
    },
  };

  return (
    <div className="space-y-4">
      {/* 上传区域 */}
      {canUpload && (
        <Card title="上传附件" size="small">
          <AntUpload.Dragger {...uploadProps} disabled={uploading}>
            <p className="ant-upload-drag-icon">
              <AntUpload style={{ fontSize: 48, color: '#1890ff' }} />
            </p>
            <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
            <p className="ant-upload-hint">
              支持多文件上传，单个文件最大10MB。支持图片、PDF、Office文档等格式
            </p>
          </AntUpload.Dragger>

          {/* 上传进度 */}
          {Object.keys(uploadProgress).length > 0 && (
            <div className="mt-4 space-y-2">
              {Object.entries(uploadProgress).map(([fileName, progress]) => (
                <div key={fileName}>
                  <Text type="secondary" className="text-xs">
                    {fileName}
                  </Text>
                  <Progress percent={progress} size="small" />
                </div>
              ))}
            </div>
          )}
        </Card>
      )}

      {/* 附件列表 */}
      <Card title={`附件列表 (${attachments.length})`} size="small" loading={loading}>
        {attachments.length === 0 ? (
          <Empty description="暂无附件" />
        ) : (
          <List
            dataSource={attachments}
            renderItem={attachment => (
              <List.Item
                actions={[
                  <Tooltip title="预览" key="preview">
                    <Button
                      type="text"
                      icon={<Eye />}
                      onClick={() => handlePreview(attachment)}
                    />
                  </Tooltip>,
                  <Tooltip title="下载" key="download">
                    <Button
                      type="text"
                      icon={<Download />}
                      onClick={() => handleDownload(attachment)}
                    />
                  </Tooltip>,
                  canDelete(attachment) && (
                    <Popconfirm
                      key="delete"
                      title={t('common.confirmDeleteContent')}
                      onConfirm={() => handleDelete(attachment)}
                      okText={t('common.confirm')}
                      cancelText={t('common.cancel')}
                    >
                      <Tooltip title={t('common.delete')}>
                        <Button type="text" danger icon={<Trash2 />} />
                      </Tooltip>
                    </Popconfirm>
                  ),
                ].filter(Boolean)}
              >
                <List.Item.Meta
                  avatar={getFileIcon(attachment.mimeType || attachment.fileType)}
                  title={
                    <Space>
                      <Text strong>{attachment.fileName}</Text>
                      <Tag color="blue">
                        {TicketAttachmentApi.formatFileSize(attachment.fileSize)}
                      </Tag>
                    </Space>
                  }
                  description={
                    <Space split={<span>•</span>}>
                      <Text type="secondary" className="text-xs">
                        {attachment.uploader?.name || '未知用户'}
                      </Text>
                      <Text type="secondary" className="text-xs">
                        {attachment.createdAt ? new Date(attachment.createdAt).toLocaleString('zh-CN') : '-'}
                      </Text>
                    </Space>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </Card>

      {/* 图片预览模态框 */}
      <Modal
        open={previewVisible}
        footer={null}
        onCancel={() => setPreviewVisible(false)}
        width={800}
        centered
      >
        {previewAttachment && (
          <div>
            <Text strong className="block mb-4">
              {previewAttachment.fileName}
            </Text>
            <AntImage
              src={TicketAttachmentApi.getPreviewUrl(ticketId, previewAttachment.id)}
              alt={previewAttachment.fileName}
              style={{ width: '100%' }}
            />
          </div>
        )}
      </Modal>
    </div>
  );
};
