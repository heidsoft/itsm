/**
 * 工单页面工具栏组件
 * 包含刷新、导出、批量操作等功能
 */

import React from 'react';
import { Button, Space, Dropdown, Tooltip } from 'antd';
import {
  ReloadOutlined,
  DownloadOutlined,
  DeleteOutlined,
  MoreOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';

export interface TicketsToolbarProps {
  selectedCount: number;
  onRefresh: () => void;
  onCreate: () => void;
  onExport: () => void;
  onBatchDelete: () => void;
  onBatchUpdate: (action: string) => void;
  loading?: boolean;
  canCreate?: boolean;
  canDelete?: boolean;
  canExport?: boolean;
}

/**
 * 工单工具栏组件
 */
export const TicketsToolbar: React.FC<TicketsToolbarProps> = ({
  selectedCount,
  onRefresh,
  onCreate,
  onExport,
  onBatchDelete,
  onBatchUpdate,
  loading = false,
  canCreate = true,
  canDelete = true,
  canExport = true,
}) => {
  // 批量操作菜单
  const batchMenuItems: MenuProps['items'] = [
    {
      key: 'assign',
      label: '批量分配',
      onClick: () => onBatchUpdate('assign'),
    },
    {
      key: 'update-status',
      label: '批量更新状态',
      onClick: () => onBatchUpdate('update-status'),
    },
    {
      key: 'update-priority',
      label: '批量更新优先级',
      onClick: () => onBatchUpdate('update-priority'),
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: '批量删除',
      danger: true,
      onClick: onBatchDelete,
      disabled: !canDelete,
    },
  ];

  // 更多操作菜单
  const moreMenuItems: MenuProps['items'] = [
    {
      key: 'import',
      label: '导入工单',
    },
    {
      key: 'export-template',
      label: '导出模板',
    },
    {
      type: 'divider',
    },
    {
      key: 'settings',
      label: '列表设置',
    },
  ];

  return (
    <div className="flex items-center justify-between mb-4 p-4 bg-white rounded-lg shadow-sm">
      {/* 左侧：标题和信息 */}
      <div className="flex items-center space-x-4">
        <h2 className="text-lg font-semibold text-gray-800">工单列表</h2>
        {selectedCount > 0 && (
          <span className="text-sm text-gray-600">
            已选择 <span className="font-semibold text-blue-600">{selectedCount}</span> 项
          </span>
        )}
      </div>

      {/* 右侧：操作按钮 */}
      <Space size="middle">
        {/* 刷新按钮 */}
        <Tooltip title="刷新数据">
          <Button
            icon={<ReloadOutlined spin={loading} />}
            onClick={onRefresh}
            loading={loading}
            data-testid="refresh-button"
          >
            刷新
          </Button>
        </Tooltip>

        {/* 导出按钮 */}
        {canExport && (
          <Tooltip title="导出工单">
            <Button
              icon={<DownloadOutlined />}
              onClick={onExport}
            >
              导出
            </Button>
          </Tooltip>
        )}

        {/* 批量操作按钮（仅在有选中项时显示） */}
        {selectedCount > 0 && (
          <Dropdown menu={{ items: batchMenuItems }} placement="bottomRight">
            <Button icon={<MoreOutlined />}>
              批量操作
            </Button>
          </Dropdown>
        )}

        {/* 创建工单按钮 */}
        {canCreate && (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={onCreate}
          >
            创建工单
          </Button>
        )}

        {/* 更多操作 */}
        <Dropdown menu={{ items: moreMenuItems }} placement="bottomRight">
          <Button icon={<MoreOutlined />} />
        </Dropdown>
      </Space>
    </div>
  );
};

export default TicketsToolbar;

