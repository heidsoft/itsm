/**
 * 批量操作工具栏组件
 * 显示已选择的工单数量和批量操作按钮
 */

'use client';

import React, { useState } from 'react';
import { Space, Button, Dropdown, Badge, Tooltip, type MenuProps } from 'antd';
import {
  CheckOutlined,
  CloseOutlined,
  UserAddOutlined,
  EditOutlined,
  TagsOutlined,
  DeleteOutlined,
  CloseCircleOutlined,
  DownloadOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import { BatchOperationModal } from './BatchOperationModal';
import { BatchOperationType } from '@/types/batch-operations';

export interface BatchOperationBarProps {
  selectedCount: number;
  selectedIds: number[];
  onClearSelection: () => void;
  onOperationComplete?: () => void;
  maxBatchSize?: number;
}

export const BatchOperationBar: React.FC<BatchOperationBarProps> = ({
  selectedCount,
  selectedIds,
  onClearSelection,
  onOperationComplete,
  maxBatchSize = 1000,
}) => {
  const [modalVisible, setModalVisible] = useState(false);
  const [operationType, setOperationType] = useState<BatchOperationType | null>(null);

  const handleOpenModal = (type: BatchOperationType) => {
    if (selectedCount > maxBatchSize) {
      return;
    }
    setOperationType(type);
    setModalVisible(true);
  };

  const handleModalClose = () => {
    setModalVisible(false);
    setOperationType(null);
  };

  const handleOperationSuccess = () => {
    setModalVisible(false);
    setOperationType(null);
    onOperationComplete?.();
    onClearSelection();
  };

  const moreMenuItems: MenuProps['items'] = [
    {
      key: 'update-type',
      label: '批量更新类型',
      icon: <EditOutlined />,
      onClick: () => handleOpenModal(BatchOperationType.UPDATE_TYPE),
    },
    {
      key: 'update-category',
      label: '批量更新分类',
      icon: <EditOutlined />,
      onClick: () => handleOpenModal(BatchOperationType.UPDATE_CATEGORY),
    },
    {
      key: 'update-fields',
      label: '批量更新字段',
      icon: <EditOutlined />,
      onClick: () => handleOpenModal(BatchOperationType.UPDATE_FIELDS),
    },
    {
      type: 'divider',
    },
    {
      key: 'add-tags',
      label: '批量添加标签',
      icon: <TagsOutlined />,
      onClick: () => handleOpenModal(BatchOperationType.ADD_TAGS),
    },
    {
      key: 'remove-tags',
      label: '批量删除标签',
      icon: <TagsOutlined />,
      onClick: () => handleOpenModal(BatchOperationType.REMOVE_TAGS),
    },
    {
      type: 'divider',
    },
    {
      key: 'archive',
      label: '批量归档',
      icon: <CloseCircleOutlined />,
      onClick: () => handleOpenModal(BatchOperationType.ARCHIVE),
    },
  ];

  const isOverLimit = selectedCount > maxBatchSize;

  return (
    <div
      className='fixed bottom-8 left-1/2 transform -translate-x-1/2 z-50 
                 bg-white rounded-lg shadow-2xl border border-gray-200 
                 px-6 py-4 animate-slide-up'
      style={{
        minWidth: '600px',
        animation: 'slideUp 0.3s ease-out',
      }}
    >
      <style>
        {`
          @keyframes slideUp {
            from {
              transform: translate(-50%, 100px);
              opacity: 0;
            }
            to {
              transform: translate(-50%, 0);
              opacity: 1;
            }
          }
        `}
      </style>

      <div className='flex items-center justify-between'>
        <div className='flex items-center gap-4'>
          <Badge count={selectedCount} showZero={false} overflowCount={9999}>
            <div className='flex items-center gap-2'>
              <CheckOutlined className='text-blue-500' style={{ fontSize: 18 }} />
              <span className='text-base font-semibold'>已选择 {selectedCount} 个工单</span>
            </div>
          </Badge>

          {isOverLimit && (
            <span className='text-red-500 text-sm'>（超出最大批量操作数量 {maxBatchSize}）</span>
          )}
        </div>

        <Space size='middle'>
          {/* 批量分配 */}
          <Tooltip title='批量分配'>
            <Button
              icon={<UserAddOutlined />}
              onClick={() => handleOpenModal(BatchOperationType.ASSIGN)}
              disabled={isOverLimit}
            >
              分配
            </Button>
          </Tooltip>

          {/* 批量更新状态 */}
          <Tooltip title='批量更新状态'>
            <Button
              icon={<EditOutlined />}
              onClick={() => handleOpenModal(BatchOperationType.UPDATE_STATUS)}
              disabled={isOverLimit}
            >
              状态
            </Button>
          </Tooltip>

          {/* 批量更新优先级 */}
          <Tooltip title='批量更新优先级'>
            <Button
              icon={<EditOutlined />}
              onClick={() => handleOpenModal(BatchOperationType.UPDATE_PRIORITY)}
              disabled={isOverLimit}
            >
              优先级
            </Button>
          </Tooltip>

          {/* 批量关闭 */}
          <Tooltip title='批量关闭'>
            <Button
              icon={<CloseCircleOutlined />}
              onClick={() => handleOpenModal(BatchOperationType.CLOSE)}
              disabled={isOverLimit}
            >
              关闭
            </Button>
          </Tooltip>

          {/* 批量导出 */}
          <Tooltip title='批量导出'>
            <Button
              icon={<DownloadOutlined />}
              onClick={() => handleOpenModal(BatchOperationType.EXPORT)}
              disabled={isOverLimit}
            >
              导出
            </Button>
          </Tooltip>

          {/* 更多操作 */}
          <Dropdown menu={{ items: moreMenuItems }} trigger={['click']}>
            <Button icon={<MoreOutlined />} disabled={isOverLimit}>
              更多
            </Button>
          </Dropdown>

          {/* 批量删除 */}
          <Tooltip title='批量删除'>
            <Button
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleOpenModal(BatchOperationType.DELETE)}
              disabled={isOverLimit}
            >
              删除
            </Button>
          </Tooltip>

          {/* 取消选择 */}
          <Button icon={<CloseOutlined />} onClick={onClearSelection}>
            取消
          </Button>
        </Space>
      </div>

      {/* 批量操作模态框 */}
      {modalVisible && operationType && (
        <BatchOperationModal
          visible={modalVisible}
          operationType={operationType}
          ticketIds={selectedIds}
          onSuccess={handleOperationSuccess}
          onCancel={handleModalClose}
        />
      )}
    </div>
  );
};

export default BatchOperationBar;
