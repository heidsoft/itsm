/**
 * 模板卡片组件
 * 展示单个模板的信息，支持快速操作
 */

'use client';

import React from 'react';
import {
  Card,
  Space,
  Tag,
  Tooltip,
  Button,
  Badge,
  Rate,
  Avatar,
  Popconfirm,
  Dropdown,
  type MenuProps,
} from 'antd';
import {
  EditOutlined,
  CopyOutlined,
  DeleteOutlined,
  EyeOutlined,
  StarOutlined,
  StarFilled,
  MoreOutlined,
  UserOutlined,
  ClockCircleOutlined,
  FileTextOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import type { TicketTemplate } from '@/types/template';

export interface TemplateCardProps {
  template: TicketTemplate;
  onView?: (template: TicketTemplate) => void;
  onEdit?: (template: TicketTemplate) => void;
  onDuplicate?: (template: TicketTemplate) => void;
  onDelete?: (templateId: string) => void;
  onFavorite?: (templateId: string) => void;
  onUnfavorite?: (templateId: string) => void;
  isFavorite?: boolean;
  viewMode?: 'grid' | 'list';
}

export const TemplateCard: React.FC<TemplateCardProps> = ({
  template,
  onView,
  onEdit,
  onDuplicate,
  onDelete,
  onFavorite,
  onUnfavorite,
  isFavorite = false,
  viewMode = 'grid',
}) => {
  const handleFavoriteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isFavorite) {
      onUnfavorite?.(template.id);
    } else {
      onFavorite?.(template.id);
    }
  };

  const moreMenuItems: MenuProps['items'] = [
    {
      key: 'view',
      label: '查看详情',
      icon: <EyeOutlined />,
      onClick: () => onView?.(template),
    },
    {
      key: 'edit',
      label: '编辑模板',
      icon: <EditOutlined />,
      onClick: () => onEdit?.(template),
    },
    {
      key: 'duplicate',
      label: '复制模板',
      icon: <CopyOutlined />,
      onClick: () => onDuplicate?.(template),
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: '删除模板',
      icon: <DeleteOutlined />,
      danger: true,
      onClick: () => {},
    },
  ];

  const renderGridView = () => (
    <Card
      hoverable
      className="template-card h-full transition-all duration-300"
      styles={{
        body: {
          padding: '20px',
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        },
      }}
      onClick={() => onView?.(template)}
    >
      {/* 顶部：封面或图标 */}
      <div className="mb-4">
        {template.coverImage ? (
          <img
            src={template.coverImage}
            alt={template.name}
            className="w-full h-32 object-cover rounded-lg"
          />
        ) : (
          <div
            className="w-full h-32 rounded-lg flex items-center justify-center text-4xl"
            style={{
              background: template.color
                ? `linear-gradient(135deg, ${template.color}22 0%, ${template.color}44 100%)`
                : 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            }}
          >
            {template.icon || <FileTextOutlined />}
          </div>
        )}
      </div>

      {/* 标题和描述 */}
      <div className="flex-1 mb-3">
        <div className="flex items-start justify-between mb-2">
          <h3 className="text-base font-semibold m-0 flex-1 pr-2 line-clamp-2">
            {template.name}
          </h3>
          <Tooltip title={isFavorite ? '取消收藏' : '收藏'}>
            <Button
              type="text"
              size="small"
              icon={
                isFavorite ? (
                  <StarFilled style={{ color: '#faad14' }} />
                ) : (
                  <StarOutlined />
                )
              }
              onClick={handleFavoriteClick}
            />
          </Tooltip>
        </div>

        <p className="text-sm text-gray-600 m-0 line-clamp-2">
          {template.description}
        </p>
      </div>

      {/* 标签 */}
      {template.tags && template.tags.length > 0 && (
        <div className="mb-3">
          <Space size={[0, 8]} wrap>
            {template.tags.slice(0, 3).map((tag) => (
              <Tag key={tag} className="m-0">
                {tag}
              </Tag>
            ))}
            {template.tags.length > 3 && (
              <Tag className="m-0">+{template.tags.length - 3}</Tag>
            )}
          </Space>
        </div>
      )}

      {/* 统计信息 */}
      <div className="flex items-center justify-between pt-3 border-t border-gray-100">
        <Space size="large">
          <Tooltip title="使用次数">
            <Space size={4}>
              <FileTextOutlined className="text-gray-400" />
              <span className="text-sm text-gray-600">{template.usageCount}</span>
            </Space>
          </Tooltip>
          <Tooltip title="平均评分">
            <Space size={4}>
              <Rate
                disabled
                value={template.rating}
                count={5}
                style={{ fontSize: 14 }}
              />
              <span className="text-sm text-gray-600">
                {template.rating.toFixed(1)}
              </span>
            </Space>
          </Tooltip>
        </Space>

        <Dropdown menu={{ items: moreMenuItems }} trigger={['click']}>
          <Button
            type="text"
            size="small"
            icon={<MoreOutlined />}
            onClick={(e) => e.stopPropagation()}
          />
        </Dropdown>
      </div>

      {/* 状态标签 */}
      <div className="mt-3 flex items-center justify-between">
        <Space size={4}>
          {template.isDraft && (
            <Tag color="warning" icon={<ExclamationCircleOutlined />}>
              草稿
            </Tag>
          )}
          {!template.isActive && (
            <Tag color="default">已禁用</Tag>
          )}
          {template.isActive && !template.isDraft && (
            <Tag color="success" icon={<CheckCircleOutlined />}>
              已发布
            </Tag>
          )}
        </Space>
        <span className="text-xs text-gray-400">v{template.version}</span>
      </div>
    </Card>
  );

  const renderListView = () => (
    <Card
      hoverable
      className="template-card-list mb-3 transition-all duration-300"
      onClick={() => onView?.(template)}
    >
      <div className="flex items-center gap-4">
        {/* 左侧：图标/封面 */}
        <div
          className="w-16 h-16 rounded-lg flex items-center justify-center text-2xl flex-shrink-0"
          style={{
            background: template.color
              ? `linear-gradient(135deg, ${template.color}22 0%, ${template.color}44 100%)`
              : 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          }}
        >
          {template.icon || <FileTextOutlined />}
        </div>

        {/* 中间：信息 */}
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between mb-1">
            <div className="flex items-center gap-2 flex-1">
              <h3 className="text-base font-semibold m-0">{template.name}</h3>
              {template.isDraft && (
                <Tag color="warning" className="m-0">
                  草稿
                </Tag>
              )}
              {!template.isActive && (
                <Tag color="default" className="m-0">
                  已禁用
                </Tag>
              )}
            </div>
            <Tooltip title={isFavorite ? '取消收藏' : '收藏'}>
              <Button
                type="text"
                size="small"
                icon={
                  isFavorite ? (
                    <StarFilled style={{ color: '#faad14' }} />
                  ) : (
                    <StarOutlined />
                  )
                }
                onClick={handleFavoriteClick}
              />
            </Tooltip>
          </div>

          <p className="text-sm text-gray-600 m-0 mb-2 line-clamp-1">
            {template.description}
          </p>

          <div className="flex items-center gap-4">
            {/* 标签 */}
            {template.tags && template.tags.length > 0 && (
              <Space size={[0, 4]} wrap>
                {template.tags.slice(0, 3).map((tag) => (
                  <Tag key={tag} className="m-0" style={{ fontSize: 12 }}>
                    {tag}
                  </Tag>
                ))}
              </Space>
            )}

            {/* 统计 */}
            <Space size="large" className="text-sm text-gray-500">
              <Tooltip title="使用次数">
                <Space size={4}>
                  <FileTextOutlined />
                  <span>{template.usageCount}</span>
                </Space>
              </Tooltip>
              <Tooltip title="评分">
                <Space size={4}>
                  <StarFilled style={{ color: '#faad14', fontSize: 12 }} />
                  <span>{template.rating.toFixed(1)}</span>
                </Space>
              </Tooltip>
              <Tooltip title="更新时间">
                <Space size={4}>
                  <ClockCircleOutlined />
                  <span>
                    {new Date(template.updatedAt).toLocaleDateString()}
                  </span>
                </Space>
              </Tooltip>
            </Space>
          </div>
        </div>

        {/* 右侧：操作按钮 */}
        <div className="flex items-center gap-2 flex-shrink-0">
          <Tooltip title="查看">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={(e) => {
                e.stopPropagation();
                onView?.(template);
              }}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={(e) => {
                e.stopPropagation();
                onEdit?.(template);
              }}
            />
          </Tooltip>
          <Tooltip title="复制">
            <Button
              type="text"
              icon={<CopyOutlined />}
              onClick={(e) => {
                e.stopPropagation();
                onDuplicate?.(template);
              }}
            />
          </Tooltip>
          <Popconfirm
            title="确认删除此模板？"
            description="此操作不可撤销"
            onConfirm={(e) => {
              e?.stopPropagation();
              onDelete?.(template.id);
            }}
            okText="确认"
            cancelText="取消"
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              onClick={(e) => e.stopPropagation()}
            />
          </Popconfirm>
        </div>
      </div>
    </Card>
  );

  return viewMode === 'grid' ? renderGridView() : renderListView();
};

export default TemplateCard;

