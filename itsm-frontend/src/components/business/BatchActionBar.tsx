'use client';

import React from 'react';
import { Alert, Button, Dropdown, Popconfirm, Space, Tooltip } from 'antd';
import type { MenuProps } from 'antd';
import { ChevronDown, X } from 'lucide-react';

/**
 * 单个批量动作定义。
 * - 简单动作：直接触发 onClick
 * - 危险动作：设置 danger=true 会走 Popconfirm 二次确认
 * - 分组溢出：多个 action 可放到 more（Dropdown 内的 MenuItem）
 */
export interface BatchAction {
  /** 稳定 key，用于 Dropdown items key，与 React 列表 key */
  key: string;
  /** 按钮文案 */
  label: React.ReactNode;
  /** 前置图标（可选） */
  icon?: React.ReactNode;
  /** 点击触发。禁止在此函数内自行 message.success，交由容器统一处理 */
  onClick: () => void | Promise<void>;
  /** 危险动作 → 红色按钮 + Popconfirm 二次确认 */
  danger?: boolean;
  /** danger 场景下自定义确认文案；缺省用 "确定要执行「{label}」吗？" */
  confirmTitle?: React.ReactNode;
  /** 禁用（例如权限不足） */
  disabled?: boolean;
  /** 禁用时的提示，配合 Tooltip */
  disabledTooltip?: string;
  /** loading 态由容器控制（避免每按钮独立 loading 混乱） */
  loading?: boolean;
  /** 是否折叠到"更多"下拉。适合低频动作 */
  overflow?: boolean;
  /** 按钮类型 */
  type?: 'primary' | 'default' | 'dashed' | 'link' | 'text';
}

export interface BatchActionBarProps {
  /** 选中数量。=0 时组件返回 null，不渲染 */
  selectedCount: number;
  /** 主体动作按钮 */
  actions: BatchAction[];
  /** 清除选择的回调（右侧"取消选择"按钮） */
  onClear?: () => void;
  /** 主体单位名词，例如 "工单" / "事件" / "服务目录"。默认 "项" */
  itemLabel?: string;
  /** 自定义左侧描述节点，覆盖默认的"已选择 N 个 XXX" */
  leftExtra?: React.ReactNode;
  /** 整栏加载态（例如获取权限中） */
  loading?: boolean;
  /** 附加样式类 */
  className?: string;
  /** 使用 Alert 或 Card 风格。默认 alert */
  variant?: 'alert' | 'card';
}

/**
 * 批量操作工具栏 —— 各列表页共享的选中态操作条
 *
 * 使用示例:
 *   const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
 *
 *   <BatchActionBar
 *     selectedCount={selectedRowKeys.length}
 *     itemLabel="事件"
 *     onClear={() => setSelectedRowKeys([])}
 *     actions={[
 *       { key: 'assign', label: '批量分派', onClick: handleAssign },
 *       { key: 'close', label: '批量关闭', onClick: handleClose },
 *       {
 *         key: 'delete',
 *         label: '批量删除',
 *         danger: true,
 *         confirmTitle: `确定删除选中的 ${selectedRowKeys.length} 项？此操作不可撤销`,
 *         onClick: handleBatchDelete,
 *       },
 *     ]}
 *   />
 */
export const BatchActionBar: React.FC<BatchActionBarProps> = ({
  selectedCount,
  actions,
  onClear,
  itemLabel = '项',
  leftExtra,
  loading = false,
  className,
  variant = 'alert',
}) => {
  if (selectedCount <= 0) return null;

  const primaryActions = actions.filter((a) => !a.overflow);
  const overflowActions = actions.filter((a) => a.overflow);

  const renderActionButton = (action: BatchAction) => {
    const btn = (
      <Button
        size="small"
        type={action.type || 'default'}
        danger={action.danger && !action.confirmTitle}
        icon={action.icon as React.ReactElement | undefined}
        disabled={action.disabled || loading}
        loading={action.loading}
        onClick={action.danger && action.confirmTitle ? undefined : () => void action.onClick()}
      >
        {action.label}
      </Button>
    );

    const wrapped = action.danger && action.confirmTitle ? (
      <Popconfirm
        key={action.key}
        title={action.confirmTitle}
        onConfirm={() => void action.onClick()}
        okText="确定"
        cancelText="取消"
        okButtonProps={{ danger: true }}
        disabled={action.disabled || loading}
      >
        <Button
          size="small"
          type={action.type || 'default'}
          danger
          icon={action.icon as React.ReactElement | undefined}
          disabled={action.disabled || loading}
          loading={action.loading}
        >
          {action.label}
        </Button>
      </Popconfirm>
    ) : (
      btn
    );

    if (action.disabled && action.disabledTooltip) {
      return (
        <Tooltip key={action.key} title={action.disabledTooltip}>
          <span>{wrapped}</span>
        </Tooltip>
      );
    }
    return <React.Fragment key={action.key}>{wrapped}</React.Fragment>;
  };

  const overflowMenu: MenuProps = {
    items: overflowActions.map((a) => ({
      key: a.key,
      label: a.label,
      icon: a.icon,
      danger: a.danger,
      disabled: a.disabled || loading,
      onClick: () => void a.onClick(),
    })),
  };

  const content = (
    <div className="flex items-center justify-between gap-4 w-full">
      <div className="text-sm">
        {leftExtra ?? (
          <>
            已选择 <span className="font-semibold text-blue-600">{selectedCount}</span> 个{itemLabel}
          </>
        )}
      </div>
      <Space size={8} wrap>
        {primaryActions.map(renderActionButton)}
        {overflowActions.length > 0 && (
          <Dropdown menu={overflowMenu} disabled={loading}>
            <Button size="small">
              更多
              <ChevronDown size={14} className="ml-1" />
            </Button>
          </Dropdown>
        )}
        {onClear && (
          <Button
            size="small"
            type="text"
            icon={<X size={14} />}
            onClick={onClear}
            disabled={loading}
          >
            取消选择
          </Button>
        )}
      </Space>
    </div>
  );

  if (variant === 'card') {
    return (
      <div className={`bg-blue-50 border border-blue-200 rounded-md px-4 py-2 mb-4 ${className || ''}`}>
        {content}
      </div>
    );
  }

  return (
    <Alert
      type="info"
      showIcon={false}
      className={`mb-4 ${className || ''}`}
      message={content}
    />
  );
};

export default BatchActionBar;
