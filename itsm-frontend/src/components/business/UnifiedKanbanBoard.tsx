import React, { useState, useMemo, useCallback } from 'react';
import {
  Card,
  Space,
  Button,
  Input,
  Select,
  Badge,
  Tag,
  Avatar,
  Tooltip,
  Modal,
  Form,
  message,
  App,
  Row,
  Col,
  Dropdown,
} from 'antd';
import {
  SearchOutlined,
  FilterOutlined,
  SettingOutlined,
  PlusOutlined,
  EyeOutlined,
  EditOutlined,
  MoreOutlined,
  SaveOutlined,
  ShareAltOutlined,
} from '@ant-design/icons';
import type { MenuProps, DropDownProps } from 'antd';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const { Search } = Input;
const { Option } = Select;

/**
 * 统一看板配置接口
 */
export interface KanbanColumnConfig<T = any> {
  /** 列的唯一标识（对应数据中的状态字段值） */
  key: string;
  /** 列标题 */
  title: string;
  /** 列背景色 */
  color: string;
  /** 自定义渲染函数（可选的） */
  renderCard?: (item: T, onClick: () => void, onEdit?: () => void) => React.ReactNode;
}

/**
 * 统一看板属性接口
 */
export interface UnifiedKanbanBoardProps<T> {
  /** 数据项数组 */
  items: T[];
  /** 加载状态 */
  loading?: boolean;
  /** 获取项的唯一标识 */
  getItemId: (item: T) => string | number;
  /** 获取项的状态字段 */
  getItemStatus: (item: T) => string;
  /** 获取项标题 */
  getItemTitle?: (item: T) => string;
  /** 获取项编号 */
  getItemNumber?: (item: T) => string;
  /** 获取项描述 */
  getItemDescription?: (item: T) => string;
  /** 获取项优先级 */
  getItemPriority?: (item: T) => string;
  /** 获取项负责人 */
  getItemAssignee?: (item: T) => { name: string; avatar?: string } | null;
  /** 获取项创建时间 */
  getItemCreatedAt?: (item: T) => string;
  /** 获取项更新时间 */
  getItemUpdatedAt?: (item: T) => string;
  /** 获取项标签 */
  getItemTags?: (item: T) => string[];
  /** 获取项分类 */
  getItemCategory?: (item: T) => string;
  /** 点击项处理函数 */
  onItemClick?: (item: T) => void;
  /** 编辑项处理函数 */
  onItemEdit?: (item: T) => void;
  /** 状态变更处理函数（拖拽） */
  onItemStatusChange?: (itemId: string | number, newStatus: string) => Promise<void>;
  /** 是否启用拖拽 */
  enableDrag?: boolean;
  /** 是否显示工具栏 */
  showToolbar?: boolean;
  /** 自定义列配置 */
  columnConfigs: KanbanColumnConfig<T>[];
  /** 自定义搜索placeholder */
  searchPlaceholder?: string;
  /** 自定义卡片渲染（完全覆盖） */
  customCardRender?: (item: T, column: KanbanColumnConfig<T>) => React.ReactNode;
  /** 是否显示状态筛选 */
  showStatusFilter?: boolean;
  /** 是否显示优先级筛选 */
  showPriorityFilter?: boolean;
  /** 可用的优先级选项 */
  priorityOptions?: Array<{ value: string; label: string; color: string }>;
}

/**
 * 默认卡片渲染器
 */
function DefaultCard<T>({
  item,
  column,
  onClick,
  onEdit,
  getItemTitle,
  getItemNumber,
  getItemDescription,
  getItemPriority,
  getItemAssignee,
  getItemCreatedAt,
  getItemTags,
  getItemCategory,
  priorityOptions = [],
}: {
  item: T;
  column: KanbanColumnConfig<T>;
  onClick: () => void;
  onEdit?: () => void;
  getItemTitle: (item: T) => string;
  getItemNumber?: (item: T) => string;
  getItemDescription?: (item: T) => string;
  getItemPriority?: (item: T) => string;
  getItemAssignee?: (item: T) => { name: string; avatar?: string } | null;
  getItemCreatedAt?: (item: T) => string;
  getItemTags?: (item: T) => string[];
  getItemCategory?: (item: T) => string;
  priorityOptions: Array<{ value: string; label: string; color: string }>;
}) {
  const title = getItemTitle(item);
  const number = getItemNumber?.(item) || '-';
  const description = getItemDescription?.(item);
  const priority = getItemPriority?.(item);
  const assignee = getItemAssignee?.(item);
  const createdAt = getItemCreatedAt?.(item);
  const tags = getItemTags?.(item) || [];
  const category = getItemCategory?.(item);

  const priorityConfig = priorityOptions.find(p => p.value === priority);

  const menuItems: MenuProps['items'] = React.useMemo(() => [
    {
      key: 'view',
      label: '查看详情',
      icon: <EyeOutlined />,
      onClick: () => onClick(),
    },
    ...(onEdit
      ? [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => onEdit(),
          },
        ]
      : []),
  ], [onClick, onEdit]);

  return (
    <Card
      size="small"
      hoverable
      className="mb-3 cursor-pointer"
      style={{
        borderLeft: priorityConfig ? `4px solid` : undefined,
        borderColor: priorityConfig?.color || undefined,
        margin: '0 0 12px 0',
      }}
      actions={menuItems.length > 0 ? [<Dropdown key="more" menu={{ items: menuItems }} trigger={['click']}><MoreOutlined /></Dropdown>] : []}
    >
      <div className="space-y-2">
        <div className="flex items-start justify-between">
          <div className="font-medium text-sm flex-1 mr-2 line-clamp-2">{title}</div>
          {priorityConfig && (
            <Badge
              count={priorityConfig.label}
              style={{ backgroundColor: priorityConfig.color, fontSize: '12px', lineHeight: '16px' }}
            />
          )}
        </div>

        <div className="flex items-center justify-between">
          <span className="text-xs text-gray-500">#{number}</span>
          {category && <Tag color="blue" className="text-xs">{category}</Tag>}
        </div>

        {description && (
          <div className="text-xs text-gray-500 line-clamp-2">{description}</div>
        )}

        <div className="flex items-center justify-between text-xs text-gray-400">
          <div className="flex items-center gap-2">
            {assignee ? (
              <>
                <Avatar size="small" className="text-xs">{assignee.name[0]}</Avatar>
                <span>{assignee.name}</span>
              </>
            ) : (
              <span>未分配</span>
            )}
          </div>
          {createdAt && (
            <span>{dayjs(createdAt).fromNow()}</span>
          )}
        </div>

        {tags.length > 0 && (
          <div className="flex gap-1 flex-wrap">
            {tags.slice(0, 2).map((tag, idx) => (
              <Tag key={idx} color="default" className="text-xs">{tag}</Tag>
            ))}
          </div>
        )}
      </div>
    </Card>
  );
}

/**
 * 统一看板组件
 * 支持Ticket、Incident等任意类型的数据展示
 */
export function UnifiedKanbanBoard<T>({
  items = [],
  loading = false,
  getItemId,
  getItemStatus,
  getItemTitle,
  getItemNumber,
  getItemDescription,
  getItemPriority,
  getItemAssignee,
  getItemCreatedAt,
  getItemUpdatedAt,
  getItemTags,
  getItemCategory,
  onItemClick,
  onItemEdit,
  onItemStatusChange,
  enableDrag = true,
  showToolbar = true,
  columnConfigs,
  searchPlaceholder = "搜索...",
  showStatusFilter = true,
  showPriorityFilter = true,
  priorityOptions = [],
  customCardRender,
}: UnifiedKanbanBoardProps<T>) {
  const { message: antMessage } = App.useApp();
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterPriority, setFilterPriority] = useState<string>('all');

  // 必需属性检查
  if (!getItemTitle) {
    throw new Error('UnifiedKanbanBoard: getItemTitle is required');
  }

  // 筛选后的项目
  const filteredItems = useMemo(() => {
    let result = [...items];

    if (searchKeyword) {
      const keyword = searchKeyword.toLowerCase();
      result = result.filter(item => {
        const title = getItemTitle(item).toLowerCase();
        const number = getItemNumber?.(item)?.toLowerCase() || '';
        const desc = getItemDescription?.(item)?.toLowerCase() || '';
        return title.includes(keyword) || number.includes(keyword) || desc.includes(keyword);
      });
    }

    if (filterStatus !== 'all') {
      result = result.filter(item => getItemStatus(item) === filterStatus);
    }

    if (filterPriority !== 'all' && getItemPriority) {
      result = result.filter(item => getItemPriority(item) === filterPriority);
    }

    return result;
  }, [items, searchKeyword, filterStatus, filterPriority, getItemTitle, getItemNumber, getItemDescription, getItemStatus, getItemPriority]);

  // 按列分组的项目
  const columnsData = useMemo(() => {
    return columnConfigs.map(column => ({
      ...column,
      items: filteredItems.filter(item => getItemStatus(item) === column.key),
    }));
  }, [filteredItems, columnConfigs, getItemStatus]);

  // 状态处理
  const handleItemClick = useCallback((item: T) => {
    onItemClick?.(item);
  }, [onItemClick]);

  const handleItemEdit = useCallback((item: T) => {
    onItemEdit?.(item);
  }, [onItemEdit]);

  // 渲染卡片
  const renderCard = useCallback((item: T, column: KanbanColumnConfig<T>) => {
    if (customCardRender) {
      return customCardRender(item, column);
    }

    return (
      <DefaultCard
        key={getItemId(item)}
        item={item}
        column={column}
        onClick={() => handleItemClick(item)}
        onEdit={() => handleItemEdit(item)}
        getItemTitle={getItemTitle}
        getItemNumber={getItemNumber}
        getItemDescription={getItemDescription}
        getItemPriority={getItemPriority}
        getItemAssignee={getItemAssignee}
        getItemCreatedAt={getItemCreatedAt}
        getItemTags={getItemTags}
        getItemCategory={getItemCategory}
        priorityOptions={priorityOptions}
      />
    );
  }, [
    customCardRender, getItemId, handleItemClick, handleItemEdit,
    getItemTitle, getItemNumber, getItemDescription, getItemPriority,
    getItemAssignee, getItemCreatedAt, getItemTags, getItemCategory, priorityOptions
  ]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-gray-500">加载中...</div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* 工具栏 */}
      {showToolbar && (
        <div className="flex items-center justify-between flex-wrap gap-4 bg-white p-4 rounded-lg shadow-sm">
          <div className="flex items-center gap-3 flex-1 min-w-[300px]">
            <Search
              placeholder={searchPlaceholder}
              allowClear
              value={searchKeyword}
              onChange={e => setSearchKeyword(e.target.value)}
              style={{ width: 250 }}
              prefix={<SearchOutlined />}
            />
            {showStatusFilter && (
              <Select
                value={filterStatus}
                onChange={setFilterStatus}
                style={{ width: 120 }}
                placeholder="状态"
              >
                <Option value="all">全部状态</Option>
                {columnConfigs.map(col => (
                  <Option key={col.key} value={col.key}>{col.title}</Option>
                ))}
              </Select>
            )}
            {showPriorityFilter && priorityOptions.length > 0 && (
              <Select
                value={filterPriority}
                onChange={setFilterPriority}
                style={{ width: 120 }}
                placeholder="优先级"
              >
                <Option value="all">全部优先级</Option>
                {priorityOptions.map(p => (
                  <Option key={p.value} value={p.value}>{p.label}</Option>
                ))}
              </Select>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Dropdown menu={{ items: [
              {
                key: 'save',
                label: '保存当前视图',
                icon: <SaveOutlined />,
                onClick: () => antMessage.info('保存视图功能开发中'),
              },
              {
                key: 'share',
                label: '共享视图',
                icon: <ShareAltOutlined />,
                onClick: () => antMessage.info('共享功能开发中'),
              },
            ] }} trigger={['click']}>
              <Button icon={<SettingOutlined />}>视图设置</Button>
            </Dropdown>
          </div>
        </div>
      )}

      {/* 看板区域 */}
      <Row gutter={[16, 0]}>
        {columnsData.map(column => (
          <Col key={column.key} span={4}>
            <Card
              title={
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <div
                      className="w-2 h-2 rounded-full mr-2"
                      style={{ backgroundColor: column.color }}
                    />
                    <span className="font-medium text-gray-900">{column.title}</span>
                  </div>
                  <Badge
                    count={column.items.length}
                    style={{ backgroundColor: column.color }}
                  />
                </div>
              }
              size="small"
              className="h-full"
              style={{ minHeight: '600px' }}
            >
              <div className="space-y-2">
                {column.items.map(item => renderCard(item, column))}
                {column.items.length === 0 && (
                  <div className="text-center text-gray-400 py-8 text-sm">暂无数据</div>
                )}
              </div>
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
}

export default UnifiedKanbanBoard;
