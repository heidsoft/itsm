"use client";

import React, { useState, useEffect, useMemo } from "react";
import {
  Select,
  Tree,
  Modal,
  Button,
  Input,
  Space,
  Typography,
  Spin,
  Empty,
  message,
} from "antd";
import {
  Folder,
  FolderOpen,
  FileText,
  Search,
  Eye,
  EyeOff,
} from "lucide-react";
import {
  ticketCategoryService,
  type CategoryTreeItem,
} from "../lib/services/ticket-category-service";

const { Option } = Select;
const { Text } = Typography;

interface TicketCategorySelectorProps {
  value?: number;
  onChange?: (value: number) => void;
  placeholder?: string;
  allowClear?: boolean;
  disabled?: boolean;
  showSearch?: boolean;
  size?: 'small' | 'middle' | 'large';
  style?: React.CSSProperties;
  className?: string;
  // 是否显示分类详情
  showDetails?: boolean;
  // 是否支持多选
  multiple?: boolean;
  // 多选时的值
  multipleValue?: number[];
  // 多选时的变化回调
  onMultipleChange?: (values: number[]) => void;
}

const TicketCategorySelector: React.FC<TicketCategorySelectorProps> = ({
  value,
  onChange,
  placeholder = "选择工单分类",
  allowClear = true,
  disabled = false,
  showSearch = true,
  size = 'middle',
  style,
  className,
  showDetails = false,
  multiple = false,
  multipleValue = [],
  onMultipleChange,
}) => {
  const [categories, setCategories] = useState<CategoryTreeItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  // 加载分类数据
  const loadCategories = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await ticketCategoryService.getCategoryTree();
      setCategories(data);
      
      // 设置默认展开的节点
      const rootKeys = data.map(item => item.id);
      setExpandedKeys(rootKeys);
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

  // 构建树形数据
  const treeData = useMemo(() => {
    const buildTree = (items: CategoryTreeItem[]): any[] => {
      return items.map(item => ({
        key: item.id,
        title: (
          <div className="flex items-center justify-between w-full">
            <div className="flex items-center">
              {item.children && item.children.length > 0 ? (
                <FolderOpen className="w-4 h-4 mr-2 text-blue-500" />
              ) : (
                <FileText className="w-4 h-4 mr-2 text-gray-500" />
              )}
              <span className="font-medium">{item.name}</span>
              {showDetails && (
                <Text type="secondary" className="ml-2 text-xs">
                  {item.code}
                </Text>
              )}
            </div>
            <div className="flex items-center space-x-2">
              <div className={`px-2 py-1 rounded text-xs ${
                item.is_active 
                  ? 'bg-green-100 text-green-800' 
                  : 'bg-red-100 text-red-800'
              }`}>
                {item.is_active ? '启用' : '禁用'}
              </div>
            </div>
          </div>
        ),
        children: item.children ? buildTree(item.children) : [],
        isLeaf: !item.children || item.children.length === 0,
      }));
    };

    return buildTree(categories);
  }, [categories, showDetails]);

  // 过滤树形数据
  const filteredTreeData = useMemo(() => {
    if (!searchTerm) return treeData;

    const filterTree = (items: any[]): any[] => {
      return items.filter(item => {
        const matchesSearch = item.title.props.children[0].props.children[1].props.children
          .toLowerCase()
          .includes(searchTerm.toLowerCase());
        
        const hasMatchingChildren = item.children && filterTree(item.children).length > 0;
        
        return matchesSearch || hasMatchingChildren;
      }).map(item => ({
        ...item,
        children: item.children ? filterTree(item.children) : [],
      }));
    };

    return filterTree(treeData);
  }, [treeData, searchTerm]);

  // 获取分类名称
  const getCategoryName = (id: number): string => {
    const findCategory = (items: CategoryTreeItem[], targetId: number): string | null => {
      for (const item of items) {
        if (item.id === targetId) {
          return item.name;
        }
        if (item.children) {
          const found = findCategory(item.children, targetId);
          if (found) return found;
        }
      }
      return null;
    };

    return findCategory(categories, id) || `分类 ${id}`;
  };

  // 获取分类路径
  const getCategoryPath = (id: number): string => {
    const findPath = (items: CategoryTreeItem[], targetId: number, path: string[] = []): string[] | null => {
      for (const item of items) {
        const currentPath = [...path, item.name];
        if (item.id === targetId) {
          return currentPath;
        }
        if (item.children) {
          const found = findPath(item.children, targetId, currentPath);
          if (found) return found;
        }
      }
      return null;
    };

    const path = findPath(categories, id);
    return path ? path.join(' > ') : `分类 ${id}`;
  };

  // 处理树节点选择
  const handleTreeSelect = (keys: React.Key[]) => {
    setSelectedKeys(keys);
  };

  // 处理确认选择
  const handleConfirmSelection = () => {
    if (selectedKeys.length === 0) {
      message.warning('请选择一个分类');
      return;
    }

    const selectedId = selectedKeys[0] as number;
    
    if (multiple) {
      const newValues = [...multipleValue];
      if (!newValues.includes(selectedId)) {
        newValues.push(selectedId);
        onMultipleChange?.(newValues);
      }
    } else {
      onChange?.(selectedId);
    }
    
    setModalVisible(false);
    setSelectedKeys([]);
  };

  // 处理移除多选项目
  const handleRemoveMultiple = (id: number) => {
    if (multiple && onMultipleChange) {
      const newValues = multipleValue.filter(v => v !== id);
      onMultipleChange(newValues);
    }
  };

  // 渲染选择器
  if (multiple) {
    return (
      <div className="space-y-2">
        <div className="flex items-center space-x-2">
          <Button
            type="dashed"
            size={size}
            onClick={() => setModalVisible(true)}
            disabled={disabled}
            style={style}
            className={className}
          >
            <Folder className="w-4 h-4 mr-2" />
            选择分类
          </Button>
        </div>
        
        {/* 已选择的分类 */}
        {multipleValue.length > 0 && (
          <div className="space-y-2">
            {multipleValue.map(id => (
              <div key={id} className="flex items-center justify-between p-2 bg-gray-50 rounded">
                <span className="text-sm">{getCategoryPath(id)}</span>
                <Button
                  type="text"
                  size="small"
                  danger
                  onClick={() => handleRemoveMultiple(id)}
                >
                  移除
                </Button>
              </div>
            ))}
          </div>
        )}

        {/* 分类选择模态框 */}
        <Modal
          title="选择工单分类"
          open={modalVisible}
          onCancel={() => {
            setModalVisible(false);
            setSelectedKeys([]);
          }}
          onOk={handleConfirmSelection}
          width={600}
        >
          <div className="space-y-4">
            <Input
              placeholder="搜索分类名称"
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
            
            {loading ? (
              <div className="text-center py-8">
                <Spin size="large" />
              </div>
            ) : error ? (
              <div className="text-center py-8 text-red-500">
                {error}
              </div>
            ) : filteredTreeData.length === 0 ? (
              <Empty description="暂无分类数据" />
            ) : (
              <Tree
                treeData={filteredTreeData}
                expandedKeys={expandedKeys}
                selectedKeys={selectedKeys}
                onExpand={setExpandedKeys}
                onSelect={handleTreeSelect}
                showLine
                showIcon={false}
                className="max-h-96 overflow-auto"
              />
            )}
          </div>
        </Modal>
      </div>
    );
  }

  // 单选模式
  return (
    <>
      <Select
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        allowClear={allowClear}
        disabled={disabled}
        showSearch={showSearch}
        size={size}
        style={style}
        className={className}
        filterOption={(input, option) =>
          (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
        }
        popupRender={(menu) => (
          <div>
            {menu}
            <div className="p-2 border-t">
              <Button
                type="text"
                size="small"
                block
                onClick={() => setModalVisible(true)}
              >
                <Folder className="w-4 h-4 mr-2" />
                浏览分类树
              </Button>
            </div>
          </div>
        )}
      >
        {categories.map(category => (
          <Option key={category.id} value={category.id}>
            {category.name}
          </Option>
        ))}
      </Select>

      {/* 分类选择模态框 */}
      <Modal
        title="选择工单分类"
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setSelectedKeys([]);
        }}
        onOk={handleConfirmSelection}
        width={600}
      >
        <div className="space-y-4">
          <Input
            placeholder="搜索分类名称"
            prefix={<Search className="w-4 h-4 text-gray-400" />}
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
          
          {loading ? (
            <div className="text-center py-8">
              <Spin size="large" />
            </div>
          ) : error ? (
            <div className="text-center py-8 text-red-500">
              {error}
            </div>
          ) : filteredTreeData.length === 0 ? (
            <Empty description="暂无分类数据" />
          ) : (
            <Tree
              treeData={filteredTreeData}
              expandedKeys={expandedKeys}
              selectedKeys={selectedKeys}
              onExpand={setExpandedKeys}
              onSelect={handleTreeSelect}
              showLine
              showIcon={false}
              className="max-h-96 overflow-auto"
            />
          )}
        </div>
      </Modal>
    </>
  );
};

export default TicketCategorySelector;
