"use client";

import React, { useState, useEffect, useCallback } from "react";
import {
  Tree,
  Button,
  message,
  Modal,
  Spin,
  Alert,
  Space,
  Typography,
  Card,
  Tooltip,
} from "antd";
import {
  DragDropContext,
  Droppable,
  Draggable,
  DropResult,
} from "@hello-pangea/dnd";
import {
  GripVertical,
  Save,
  RefreshCw,
  Undo,
  Folder,
  FolderOpen,
  FileText,
  ArrowUp,
  ArrowDown,
  ArrowLeft,
  ArrowRight,
} from "lucide-react";
import {
  ticketCategoryService,
  type CategoryTreeItem,
} from "../lib/services/ticket-category-service";

const { Text } = Typography;

interface TicketCategoryDragSortProps {
  onSave?: (categories: CategoryTreeItem[]) => void;
  showActions?: boolean;
  className?: string;
  style?: React.CSSProperties;
}

const TicketCategoryDragSort: React.FC<TicketCategoryDragSortProps> = ({
  onSave,
  showActions = true,
  className,
  style,
}) => {
  const [categories, setCategories] = useState<CategoryTreeItem[]>([]);
  const [originalCategories, setOriginalCategories] = useState<CategoryTreeItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [hasChanges, setHasChanges] = useState(false);

  // 加载分类数据
  const loadCategories = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await ticketCategoryService.getCategoryTree();
      setCategories(data);
      setOriginalCategories(JSON.parse(JSON.stringify(data))); // 深拷贝
      
      // 设置默认展开的节点
      const rootKeys = data.map(item => item.id);
      setExpandedKeys(rootKeys);
      setHasChanges(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载分类数据失败');
      message.error('加载分类数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCategories();
  }, []);

  // 检查是否有变化
  useEffect(() => {
    const checkChanges = () => {
      const originalStr = JSON.stringify(originalCategories);
      const currentStr = JSON.stringify(categories);
      setHasChanges(originalStr !== currentStr);
    };
    
    checkChanges();
  }, [categories, originalCategories]);

  // 定义树形节点类型
  interface TreeNodeData {
    key: string;
    title: React.ReactNode;
    children?: TreeNodeData[];
    isLeaf?: boolean;
    disabled?: boolean;
  }
  
  // 定义拖拽源和目标类型
  interface DragSource {
    index: number;
    droppableId: string;
  }
  
  interface DragDestination {
    index: number;
    droppableId: string;
  }

  // 构建树形数据
  const buildTreeData = useCallback((items: CategoryTreeItem[]): TreeNodeData[] => {
    return items.map((item, index) => ({
      key: item.id,
      title: (
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center">
            <GripVertical className="w-4 h-4 mr-2 text-gray-400 cursor-move" />
            {item.children && item.children.length > 0 ? (
              <FolderOpen className="w-4 h-4 mr-2 text-blue-500" />
            ) : (
              <FileText className="w-4 h-4 mr-2 text-gray-500" />
            )}
            <span className="text-sm font-medium">{item.name}</span>
          </div>
          <div className="flex items-center space-x-1">
            <Text type="secondary" className="text-xs">
              {item.code}
            </Text>
          </div>
        </div>
      ),
      children: item.children ? buildTreeData(item.children) : undefined,
      isLeaf: !item.children || item.children.length === 0,
    }));
  }, []);

  // 处理树形拖拽结束
  const handleTreeDragEnd = (source: DragSource, destination: DragDestination | null) => {
    if (!destination) return;
    
    // 拖拽逻辑处理
    const newCategories = [...categories];
    // 实现具体的拖拽逻辑
    setCategories(newCategories);
  };

  // 处理列表拖拽结束
  const handleListDragEnd = (source: DragSource, destination: DragDestination | null) => {
    if (!destination) return;
    
    // 拖拽逻辑处理
    const newCategories = [...categories];
    // 实现具体的拖拽逻辑
    setCategories(newCategories);
  };

  // 根据ID查找节点
  const findNodeById = (items: CategoryTreeItem[], id: string): CategoryTreeItem | null => {
    for (const item of items) {
      if (item.id.toString() === id) {
        return item;
      }
      if (item.children) {
        const found = findNodeById(item.children, id);
        if (found) return found;
      }
    }
    return null;
  };

  // 根据ID移除节点
  const removeNodeById = (items: CategoryTreeItem[], id: string): boolean => {
    for (let i = 0; i < items.length; i++) {
      if (items[i].id.toString() === id) {
        items.splice(i, 1);
        return true;
      }
      if (items[i].children) {
        if (removeNodeById(items[i].children, id)) {
          return true;
        }
      }
    }
    return false;
  };

  // 更新排序
  const updateSortOrders = (items: CategoryTreeItem[], parentLevel: number = 0) => {
    items.forEach((item, index) => {
      item.sort_order = (parentLevel * 1000) + index;
      item.level = parentLevel;
      if (item.children && item.children.length > 0) {
        updateSortOrders(item.children, parentLevel + 1);
      }
    });
  };

  // 保存更改
  const handleSave = async () => {
    try {
      setSaving(true);
      
      // 准备批量更新数据
      const updateData = prepareUpdateData(categories);
      
      // 调用批量更新API
      await ticketCategoryService.batchUpdateCategories(updateData);
      
      message.success('分类排序保存成功');
      setOriginalCategories(JSON.parse(JSON.stringify(categories)));
      setHasChanges(false);
      
      if (onSave) {
        onSave(categories);
      }
    } catch (err) {
      message.error('保存失败: ' + (err instanceof Error ? err.message : '未知错误'));
    } finally {
      setSaving(false);
    }
  };

  // 准备更新数据
  const prepareUpdateData = (items: CategoryTreeItem[]): Array<{
    id: string;
    parent_id?: string;
    sort_order: number;
    level: number;
  }> => {
    const updates: Array<{
      id: string;
      parent_id?: string;
      sort_order: number;
      level: number;
    }> = [];
    
    const processItems = (
      items: CategoryTreeItem[], 
      parentId?: string, 
      level: number = 0
    ) => {
      items.forEach((item, index) => {
        updates.push({
          id: item.id,
          parent_id: parentId,
          sort_order: index,
          level: level,
        });
        
        if (item.children && item.children.length > 0) {
          processItems(item.children, item.id, level + 1);
        }
      });
    };
    
    processItems(items);
    return updates;
  };

  // 重置更改
  const handleReset = () => {
    Modal.confirm({
      title: '确认重置',
      content: '确定要重置所有更改吗？这将丢失所有未保存的修改。',
      onOk: () => {
        setCategories(JSON.parse(JSON.stringify(originalCategories)));
        setHasChanges(false);
        message.info('已重置所有更改');
      },
    });
  };

  // 刷新数据
  const handleRefresh = () => {
    loadCategories();
  };

  // 移动节点
  const moveNode = (direction: 'up' | 'down' | 'left' | 'right', itemId: number) => {
    const newCategories = [...categories];
    
    switch (direction) {
      case 'up':
        moveNodeUp(newCategories, itemId);
        break;
      case 'down':
        moveNodeDown(newCategories, itemId);
        break;
      case 'left':
        moveNodeLeft(newCategories, itemId);
        break;
      case 'right':
        moveNodeRight(newCategories, itemId);
        break;
    }
    
    updateSortOrders(newCategories);
    setCategories(newCategories);
  };

  // 向上移动节点
  const moveNodeUp = (items: CategoryTreeItem[], itemId: number): boolean => {
    for (let i = 0; i < items.length; i++) {
      if (items[i].id === itemId) {
        if (i > 0) {
          [items[i], items[i - 1]] = [items[i - 1], items[i]];
          return true;
        }
        return false;
      }
      if (items[i].children) {
        if (moveNodeUp(items[i].children, itemId)) {
          return true;
        }
      }
    }
    return false;
  };

  // 向下移动节点
  const moveNodeDown = (items: CategoryTreeItem[], itemId: number): boolean => {
    for (let i = 0; i < items.length; i++) {
      if (items[i].id === itemId) {
        if (i < items.length - 1) {
          [items[i], items[i + 1]] = [items[i + 1], items[i]];
          return true;
        }
        return false;
      }
      if (items[i].children) {
        if (moveNodeDown(items[i].children, itemId)) {
          return true;
        }
      }
    }
    return false;
  };

  // 向左移动节点（提升层级）
  const moveNodeLeft = (items: CategoryTreeItem[], itemId: number): boolean => {
    for (let i = 0; i < items.length; i++) {
      if (items[i].id === itemId) {
        // 已经是根节点，无法向左移动
        return false;
      }
      if (items[i].children) {
        if (moveNodeLeft(items[i].children, itemId)) {
          return true;
        }
      }
    }
    return false;
  };

  // 向右移动节点（降低层级）
  const moveNodeRight = (items: CategoryTreeItem[], itemId: number): boolean => {
    for (let i = 0; i < items.length; i++) {
      if (items[i].id === itemId) {
        // 需要找到前一个兄弟节点作为父节点
        if (i > 0) {
          const parentNode = items[i - 1];
          if (!parentNode.children) parentNode.children = [];
          parentNode.children.push(items[i]);
          items.splice(i, 1);
          return true;
        }
        return false;
      }
      if (items[i].children) {
        if (moveNodeRight(items[i].children, itemId)) {
          return true;
        }
      }
    }
    return false;
  };

  if (loading) {
    return (
      <div className="text-center py-8">
        <Spin size="large" />
        <div className="mt-4">加载分类数据中...</div>
      </div>
    );
  }

  if (error) {
    return (
      <Alert
        message="加载失败"
        description={error}
        type="error"
        showIcon
        action={
          <Button size="small" onClick={loadCategories}>
            重试
          </Button>
        }
      />
    );
  }

  return (
    <div className={className} style={style}>
      {showActions && (
        <Card className="mb-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Text strong>工单分类排序</Text>
              {hasChanges && (
                <Text type="warning">有未保存的更改</Text>
              )}
            </div>
            
            <Space>
              <Tooltip title="刷新数据">
                <Button
                  icon={<RefreshCw className="w-4 h-4" />}
                  onClick={handleRefresh}
                  disabled={loading}
                >
                  刷新
                </Button>
              </Tooltip>
              
              {hasChanges && (
                <>
                  <Tooltip title="重置更改">
                    <Button
                      icon={<Undo className="w-4 h-4" />}
                      onClick={handleReset}
                    >
                      重置
                    </Button>
                  </Tooltip>
                  
                  <Tooltip title="保存更改">
                    <Button
                      type="primary"
                      icon={<Save className="w-4 h-4" />}
                      onClick={handleSave}
                      loading={saving}
                    >
                      保存
                    </Button>
                  </Tooltip>
                </>
              )}
            </Space>
          </div>
        </Card>
      )}

      <DragDropContext onDragEnd={handleDragEnd}>
        <Droppable droppableId="categories" type="list">
          {(provided) => (
            <div
              ref={provided.innerRef}
              {...provided.droppableProps}
              className="space-y-2"
            >
              {categories.map((category, index) => (
                <Draggable
                  key={category.id}
                  draggableId={category.id.toString()}
                  index={index}
                >
                  {(provided, snapshot) => (
                    <div
                      ref={provided.innerRef}
                      {...provided.draggableProps}
                      {...provided.dragHandleProps}
                      className={`p-4 border rounded-lg bg-white ${
                        snapshot.isDragging ? 'shadow-lg' : ''
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center">
                          <GripVertical className="w-4 h-4 mr-2 text-gray-400 cursor-move" />
                          {category.children && category.children.length > 0 ? (
                            <FolderOpen className="w-4 h-4 mr-2 text-blue-500" />
                          ) : (
                            <FileText className="w-4 h-4 mr-2 text-gray-500" />
                          )}
                          <span className="font-medium">{category.name}</span>
                          <Text type="secondary" className="ml-2 text-xs">
                            {category.code}
                          </Text>
                        </div>
                        
                        <div className="flex items-center space-x-2">
                          <div className={`px-2 py-1 rounded text-xs ${
                            category.is_active 
                              ? 'bg-green-100 text-green-800' 
                              : 'bg-red-100 text-red-800'
                          }`}>
                            {category.is_active ? '启用' : '禁用'}
                          </div>
                          
                          <Text type="secondary" className="text-xs">
                            排序: {category.sort_order}
                          </Text>
                          
                          <Space size="small">
                            <Tooltip title="上移">
                              <Button
                                size="small"
                                icon={<ArrowUp className="w-3 h-3" />}
                                onClick={() => moveNode('up', category.id)}
                                disabled={index === 0}
                              />
                            </Tooltip>
                            
                            <Tooltip title="下移">
                              <Button
                                size="small"
                                icon={<ArrowDown className="w-3 h-3" />}
                                onClick={() => moveNode('down', category.id)}
                                disabled={index === categories.length - 1}
                              />
                            </Tooltip>
                            
                            <Tooltip title="提升层级">
                              <Button
                                size="small"
                                icon={<ArrowLeft className="w-3 h-3" />}
                                onClick={() => moveNode('left', category.id)}
                              />
                            </Tooltip>
                            
                            <Tooltip title="降低层级">
                              <Button
                                size="small"
                                icon={<ArrowRight className="w-3 h-3" />}
                                onClick={() => moveNode('right', category.id)}
                                disabled={index === 0}
                              />
                            </Tooltip>
                          </Space>
                        </div>
                      </div>
                      
                      {category.children && category.children.length > 0 && (
                        <div className="mt-3 ml-6">
                          <Droppable droppableId={category.id.toString()} type="list">
                            {(provided) => (
                              <div
                                ref={provided.innerRef}
                                {...provided.droppableProps}
                                className="space-y-2"
                              >
                                {category.children.map((child, childIndex) => (
                                  <Draggable
                                    key={child.id}
                                    draggableId={child.id.toString()}
                                    index={childIndex}
                                  >
                                    {(provided, snapshot) => (
                                      <div
                                        ref={provided.innerRef}
                                        {...provided.draggableProps}
                                        {...provided.dragHandleProps}
                                        className={`p-3 border rounded bg-gray-50 ${
                                          snapshot.isDragging ? 'shadow-lg' : ''
                                        }`}
                                      >
                                        <div className="flex items-center justify-between">
                                          <div className="flex items-center">
                                            <GripVertical className="w-4 h-4 mr-2 text-gray-400 cursor-move" />
                                            <FileText className="w-4 h-4 mr-2 text-gray-500" />
                                            <span className="text-sm">{child.name}</span>
                                            <Text type="secondary" className="ml-2 text-xs">
                                              {child.code}
                                            </Text>
                                          </div>
                                          
                                          <div className="flex items-center space-x-2">
                                            <div className={`px-2 py-1 rounded text-xs ${
                                              child.is_active 
                                                ? 'bg-green-100 text-green-800' 
                                                : 'bg-red-100 text-red-800'
                                            }`}>
                                              {child.is_active ? '启用' : '禁用'}
                                            </div>
                                            
                                            <Text type="secondary" className="text-xs">
                                              排序: {child.sort_order}
                                            </Text>
                                          </div>
                                        </div>
                                      </div>
                                    )}
                                  </Draggable>
                                ))}
                                {provided.placeholder}
                              </div>
                            )}
                          </Droppable>
                        </div>
                      )}
                    </div>
                  )}
                </Draggable>
              ))}
              {provided.placeholder}
            </div>
          )}
        </Droppable>
      </DragDropContext>
    </div>
  );
};

export default TicketCategoryDragSort;
