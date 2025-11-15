/**
 * 模板列表组件
 * 提供模板的列表展示、搜索、筛选、排序和批量操作功能
 */

'use client';

import React, { useState, useMemo, useCallback } from 'react';
import {
  Card,
  Input,
  Select,
  Button,
  Space,
  Row,
  Col,
  Empty,
  Spin,
  Pagination,
  Radio,
  Checkbox,
  Dropdown,
  Tag,
  Badge,
  message,
  Modal,
  type MenuProps,
} from 'antd';
import {
  SearchOutlined,
  PlusOutlined,
  AppstoreOutlined,
  UnorderedListOutlined,
  FilterOutlined,
  SortAscendingOutlined,
  DownloadOutlined,
  UploadOutlined,
  DeleteOutlined,
  CheckOutlined,
  CloseOutlined,
  ReloadOutlined,
  StarOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import { TemplateCard } from './TemplateCard';
import type {
  TicketTemplate,
  TemplateListQuery,
  TemplateVisibility,
} from '@/types/template';
import {
  useTemplatesQuery,
  useDeleteTemplateMutation,
  useBatchDeleteTemplatesMutation,
  useBatchToggleTemplatesMutation,
  useFavoriteTemplateMutation,
  useUnfavoriteTemplateMutation,
  useDuplicateTemplateMutation,
} from '@/lib/hooks/useTemplateQuery';

const { Option } = Select;
const { Search } = Input;

export interface TemplateListProps {
  onTemplateClick?: (template: TicketTemplate) => void;
  onCreateClick?: () => void;
  onEditClick?: (template: TicketTemplate) => void;
  showActions?: boolean;
}

export const TemplateList: React.FC<TemplateListProps> = ({
  onTemplateClick,
  onCreateClick,
  onEditClick,
  showActions = true,
}) => {
  // State
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [favoriteIds, setFavoriteIds] = useState<Set<string>>(new Set());
  const [query, setQuery] = useState<TemplateListQuery>({
    page: 1,
    pageSize: 12,
    sortBy: 'usageCount',
    sortOrder: 'desc',
  });

  // Queries & Mutations
  const { data, isLoading, refetch } = useTemplatesQuery(query);
  const deleteMutation = useDeleteTemplateMutation();
  const batchDeleteMutation = useBatchDeleteTemplatesMutation();
  const batchToggleMutation = useBatchToggleTemplatesMutation();
  const favoriteMutation = useFavoriteTemplateMutation();
  const unfavoriteMutation = useUnfavoriteTemplateMutation();
  const duplicateMutation = useDuplicateTemplateMutation();

  const templates = data?.templates || [];
  const total = data?.total || 0;

  // Handlers
  const handleSearch = (value: string) => {
    setQuery((prev) => ({ ...prev, search: value, page: 1 }));
  };

  const handleCategoryChange = (value: string) => {
    setQuery((prev) => ({ ...prev, categoryId: value, page: 1 }));
  };

  const handleVisibilityChange = (value: TemplateVisibility) => {
    setQuery((prev) => ({ ...prev, visibility: value, page: 1 }));
  };

  const handleStatusChange = (values: string[]) => {
    const filters: any = {};
    
    if (values.includes('active')) filters.isActive = true;
    if (values.includes('draft')) filters.isDraft = true;
    if (values.includes('archived')) filters.isArchived = true;
    
    setQuery((prev) => ({ ...prev, ...filters, page: 1 }));
  };

  const handleSortChange = (value: string) => {
    const [sortBy, sortOrder] = value.split('-');
    setQuery((prev) => ({
      ...prev,
      sortBy: sortBy as any,
      sortOrder: sortOrder as any,
      page: 1,
    }));
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setQuery((prev) => ({ ...prev, page, pageSize }));
  };

  const handleSelectAll = () => {
    if (selectedIds.length === templates.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(templates.map((t) => t.id));
    }
  };

  const handleSelectTemplate = (id: string, checked: boolean) => {
    if (checked) {
      setSelectedIds((prev) => [...prev, id]);
    } else {
      setSelectedIds((prev) => prev.filter((sid) => sid !== id));
    }
  };

  const handleDelete = async (templateId: string) => {
    try {
      await deleteMutation.mutateAsync(templateId);
      refetch();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  const handleBatchDelete = () => {
    Modal.confirm({
      title: '确认批量删除',
      content: `确定要删除选中的 ${selectedIds.length} 个模板吗？此操作不可撤销。`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await batchDeleteMutation.mutateAsync(selectedIds);
          setSelectedIds([]);
          refetch();
        } catch (error) {
          console.error('批量删除失败:', error);
        }
      },
    });
  };

  const handleBatchToggle = (isActive: boolean) => {
    Modal.confirm({
      title: `确认批量${isActive ? '启用' : '禁用'}`,
      content: `确定要${isActive ? '启用' : '禁用'}选中的 ${selectedIds.length} 个模板吗？`,
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        try {
          await batchToggleMutation.mutateAsync({ templateIds: selectedIds, isActive });
          setSelectedIds([]);
          refetch();
        } catch (error) {
          console.error('批量操作失败:', error);
        }
      },
    });
  };

  const handleFavorite = async (templateId: string) => {
    try {
      await favoriteMutation.mutateAsync(templateId);
      setFavoriteIds((prev) => new Set(prev).add(templateId));
    } catch (error) {
      console.error('收藏失败:', error);
    }
  };

  const handleUnfavorite = async (templateId: string) => {
    try {
      await unfavoriteMutation.mutateAsync(templateId);
      setFavoriteIds((prev) => {
        const newSet = new Set(prev);
        newSet.delete(templateId);
        return newSet;
      });
    } catch (error) {
      console.error('取消收藏失败:', error);
    }
  };

  const handleDuplicate = async (template: TicketTemplate) => {
    try {
      await duplicateMutation.mutateAsync({
        templateId: template.id,
        newName: `${template.name} (副本)`,
        copyFields: true,
        copyAutomation: true,
        copyPermission: false,
      });
      message.success('模板已复制');
      refetch();
    } catch (error) {
      console.error('复制失败:', error);
    }
  };

  const batchMenuItems: MenuProps['items'] = [
    {
      key: 'enable',
      label: '批量启用',
      icon: <CheckOutlined />,
      onClick: () => handleBatchToggle(true),
      disabled: selectedIds.length === 0,
    },
    {
      key: 'disable',
      label: '批量禁用',
      icon: <CloseOutlined />,
      onClick: () => handleBatchToggle(false),
      disabled: selectedIds.length === 0,
    },
    {
      type: 'divider',
    },
    {
      key: 'export',
      label: '批量导出',
      icon: <DownloadOutlined />,
      disabled: selectedIds.length === 0,
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: '批量删除',
      icon: <DeleteOutlined />,
      danger: true,
      onClick: handleBatchDelete,
      disabled: selectedIds.length === 0,
    },
  ];

  // Render
  return (
    <div className="template-list">
      {/* 顶部操作栏 */}
      <Card className="mb-4">
        <Row gutter={[16, 16]}>
          <Col flex="auto">
            <Space size="middle" className="w-full">
              <Search
                placeholder="搜索模板..."
                allowClear
                onSearch={handleSearch}
                style={{ width: 300 }}
                prefix={<SearchOutlined />}
              />

              <Select
                placeholder="选择分类"
                allowClear
                style={{ width: 150 }}
                onChange={handleCategoryChange}
              >
                <Option value="incident">事件</Option>
                <Option value="request">服务请求</Option>
                <Option value="problem">问题</Option>
                <Option value="change">变更</Option>
              </Select>

              <Select
                placeholder="可见性"
                allowClear
                style={{ width: 120 }}
                onChange={handleVisibilityChange as any}
              >
                <Option value="public">公开</Option>
                <Option value="private">私有</Option>
                <Option value="department">部门</Option>
                <Option value="role">角色</Option>
              </Select>

              <Select
                placeholder="排序方式"
                defaultValue="usageCount-desc"
                style={{ width: 150 }}
                onChange={handleSortChange}
              >
                <Option value="usageCount-desc">
                  使用次数 ↓
                </Option>
                <Option value="rating-desc">评分 ↓</Option>
                <Option value="createdAt-desc">创建时间 ↓</Option>
                <Option value="updatedAt-desc">更新时间 ↓</Option>
                <Option value="name-asc">名称 A-Z</Option>
              </Select>
            </Space>
          </Col>

          <Col>
            <Space>
              {selectedIds.length > 0 && (
                <>
                  <Badge count={selectedIds.length} showZero>
                    <Button
                      icon={<CheckOutlined />}
                      onClick={handleSelectAll}
                    >
                      {selectedIds.length === templates.length
                        ? '取消全选'
                        : '全选'}
                    </Button>
                  </Badge>

                  <Dropdown menu={{ items: batchMenuItems }} trigger={['click']}>
                    <Button icon={<MoreOutlined />}>
                      批量操作
                    </Button>
                  </Dropdown>
                </>
              )}

              <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)}>
                <Radio.Button value="grid">
                  <AppstoreOutlined />
                </Radio.Button>
                <Radio.Button value="list">
                  <UnorderedListOutlined />
                </Radio.Button>
              </Radio.Group>

              <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
                刷新
              </Button>

              {showActions && (
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={onCreateClick}
                >
                  创建模板
                </Button>
              )}
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 模板列表 */}
      <Spin spinning={isLoading}>
        {templates.length === 0 ? (
          <Card>
            <Empty
              description="暂无模板"
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            >
              {showActions && (
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={onCreateClick}
                >
                  创建第一个模板
                </Button>
              )}
            </Empty>
          </Card>
        ) : viewMode === 'grid' ? (
          <Row gutter={[16, 16]}>
            {templates.map((template) => (
              <Col key={template.id} xs={24} sm={12} lg={8} xl={6}>
                {selectedIds.length > 0 && (
                  <div
                    className="absolute top-2 left-2 z-10"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Checkbox
                      checked={selectedIds.includes(template.id)}
                      onChange={(e) =>
                        handleSelectTemplate(template.id, e.target.checked)
                      }
                    />
                  </div>
                )}
                <TemplateCard
                  template={template}
                  viewMode="grid"
                  onView={onTemplateClick}
                  onEdit={onEditClick}
                  onDuplicate={handleDuplicate}
                  onDelete={handleDelete}
                  onFavorite={handleFavorite}
                  onUnfavorite={handleUnfavorite}
                  isFavorite={favoriteIds.has(template.id)}
                />
              </Col>
            ))}
          </Row>
        ) : (
          <div>
            {templates.map((template) => (
              <div key={template.id} className="relative">
                {selectedIds.length > 0 && (
                  <div
                    className="absolute top-4 left-4 z-10"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Checkbox
                      checked={selectedIds.includes(template.id)}
                      onChange={(e) =>
                        handleSelectTemplate(template.id, e.target.checked)
                      }
                      style={{ paddingLeft: selectedIds.length > 0 ? '32px' : 0 }}
                    />
                  </div>
                )}
                <TemplateCard
                  template={template}
                  viewMode="list"
                  onView={onTemplateClick}
                  onEdit={onEditClick}
                  onDuplicate={handleDuplicate}
                  onDelete={handleDelete}
                  onFavorite={handleFavorite}
                  onUnfavorite={handleUnfavorite}
                  isFavorite={favoriteIds.has(template.id)}
                />
              </div>
            ))}
          </div>
        )}
      </Spin>

      {/* 分页 */}
      {total > 0 && (
        <Card className="mt-4">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">
              共 {total} 个模板
              {selectedIds.length > 0 && ` (已选择 ${selectedIds.length} 个)`}
            </span>
            <Pagination
              current={query.page}
              pageSize={query.pageSize}
              total={total}
              onChange={handlePageChange}
              showSizeChanger
              showQuickJumper
              showTotal={(total) => `共 ${total} 条`}
              pageSizeOptions={['12', '24', '48', '96']}
            />
          </div>
        </Card>
      )}
    </div>
  );
};

export default TemplateList;

