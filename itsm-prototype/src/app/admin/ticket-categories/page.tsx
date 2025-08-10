"use client";

import React, { useState, useEffect, useCallback } from "react";
import {
  Card,
  Button,
  Modal,
  Form,
  Input,
  Switch,
  InputNumber,
  Select,
  message,
  Space,
  Tree,
  Row,
  Col,
  Statistic,
  Popconfirm,
  Typography,
  Empty,
  Upload,
} from "antd";
import {
  Plus,
  Edit,
  Delete,
  Folder,
  FolderOpen,
  FileText,
  GripVertical,
  Eye,
  EyeOff,
  RefreshCw,
  Search,
  Download,
} from "lucide-react";
import {
  ticketCategoryService,
  type TicketCategory,
  type CategoryTreeItem,
} from "@/app/lib/services/ticket-category-service";
import { LoadingEmptyError } from "@/app/components/ui/LoadingEmptyError";
import TicketCategoryDragSort from "@/app/components/TicketCategoryDragSort";
import TicketCategoryImport from "@/app/components/TicketCategoryImport";
import TicketCategoryExport from "@/app/components/TicketCategoryExport";

const { TextArea } = Input;
const { Option } = Select;
const { Title } = Typography;

// 定义树形数据项的类型
interface TreeDataItem {
  key: number;
  title: React.ReactNode;
  children?: TreeDataItem[];
  isLeaf: boolean;
}

const TicketCategoriesPage: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [categories, setCategories] = useState<CategoryTreeItem[]>([]);
  const [flatCategories, setFlatCategories] = useState<TicketCategory[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState<TicketCategory | null>(
    null
  );
  const [form] = Form.useForm();
  const [searchTerm, setSearchTerm] = useState("");
  const [filterActive, setFilterActive] = useState<boolean | null>(null);
  const [viewMode, setViewMode] = useState<"tree" | "list" | "drag">("tree");
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [importModalVisible, setImportModalVisible] = useState(false);
  const [exportModalVisible, setExportModalVisible] = useState(false);

  // 加载分类数据
  const loadCategories = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [treeData, listData] = await Promise.all([
        ticketCategoryService.getCategoryTree(),
        ticketCategoryService.listCategories({ page_size: 1000 }),
      ]);

      setCategories(treeData);
      setFlatCategories(listData.categories);

      // 设置默认展开的节点
      const rootKeys = treeData.map((item: CategoryTreeItem) => item.id);
      setExpandedKeys(rootKeys);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载分类数据失败");
      message.error("加载分类数据失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  // 处理表单提交
  const handleSubmit = async (values: {
    name: string;
    code: string;
    description?: string;
    parent_id: number;
    sort_order: number;
    is_active: boolean;
  }) => {
    try {
      if (editingCategory) {
        // 更新分类
        await ticketCategoryService.updateCategory(editingCategory.id, values);
        message.success("分类更新成功");
      } else {
        // 创建分类 - 确保 description 有值
        const createData = {
          ...values,
          description: values.description || "",
        };
        await ticketCategoryService.createCategory(createData);
        message.success("分类创建成功");
      }

      setModalVisible(false);
      form.resetFields();
      setEditingCategory(null);
      loadCategories();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "操作失败");
    }
  };

  // 处理编辑
  const handleEdit = (category: TicketCategory) => {
    setEditingCategory(category);
    form.setFieldsValue({
      name: category.name,
      description: category.description,
      code: category.code,
      parent_id: category.parent_id || 0,
      sort_order: category.sort_order,
      is_active: category.is_active,
    });
    setModalVisible(true);
  };

  // 处理删除
  const handleDelete = async (id: number) => {
    try {
      await ticketCategoryService.deleteCategory(id);
      message.success("分类删除成功");
      loadCategories();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "删除失败");
    }
  };

  // 处理创建新分类
  const handleCreate = () => {
    setEditingCategory(null);
    form.resetFields();
    form.setFieldsValue({
      parent_id: 0,
      sort_order: 0,
      is_active: true,
    });
    setModalVisible(true);
  };

  // 过滤和搜索分类
  const filteredCategories = flatCategories.filter((category) => {
    const matchesSearch =
      category.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      category.code.toLowerCase().includes(searchTerm.toLowerCase()) ||
      category.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesActive =
      filterActive === null || category.is_active === filterActive;
    return matchesSearch && matchesActive;
  });

  // 构建树形数据
  const buildTreeData = (items: TicketCategory[]): TreeDataItem[] => {
    const itemMap = new Map<number, TreeDataItem>();
    const result: TreeDataItem[] = [];

    items.forEach((item: TicketCategory) => {
      itemMap.set(item.id, {
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
              <span className="ml-2 px-2 py-1 text-xs bg-green-100 text-green-800 rounded">
                {item.code}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <Switch
                checked={item.is_active}
                size="small"
                onChange={async (checked) => {
                  try {
                    await ticketCategoryService.updateCategory(item.id, {
                      is_active: checked,
                    });
                    loadCategories();
                  } catch {
                    message.error("状态更新失败");
                  }
                }}
              />
              <Button
                type="text"
                size="small"
                icon={<Edit className="w-3 h-3" />}
                onClick={(e) => {
                  e.stopPropagation();
                  handleEdit(item);
                }}
              />
              <Popconfirm
                title="确定要删除这个分类吗？"
                description="删除后无法恢复，请谨慎操作。"
                onConfirm={() => handleDelete(item.id)}
                okText="确定"
                cancelText="取消"
              >
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<Delete className="w-3 h-3" />}
                  onClick={(e) => e.stopPropagation()}
                />
              </Popconfirm>
            </div>
          </div>
        ),
        children: item.children ? buildTreeData(item.children) : [],
        isLeaf: !item.children || item.children.length === 0,
      });
    });

    items.forEach((item: TicketCategory) => {
      if (item.parent_id === 0) {
        const itemData = itemMap.get(item.id);
        if (itemData) {
          result.push(itemData);
        }
      } else {
        const parent = itemMap.get(item.parent_id);
        const itemData = itemMap.get(item.id);
        if (parent && itemData) {
          if (!parent.children) parent.children = [];
          parent.children.push(itemData);
        }
      }
    });

    return result;
  };

  // 统计信息
  const stats = {
    total: flatCategories.length,
    active: flatCategories.filter((c) => c.is_active).length,
    inactive: flatCategories.filter((c) => !c.is_active).length,
    root: flatCategories.filter((c) => c.parent_id === 0).length,
  };

  if (loading) {
    return <LoadingEmptyError state="loading" />;
  }

  if (error) {
    return (
      <LoadingEmptyError
        state="error"
        error={{
          title: "加载失败",
          description: error,
          onRetry: loadCategories,
        }}
      />
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <Title level={2} className="mb-0">
          工单分类管理
        </Title>
        <Space>
          <Button
            icon={<RefreshCw className="w-4 h-4" />}
            onClick={loadCategories}
          >
            刷新
          </Button>
          <Button
            type="primary"
            icon={<Plus className="w-4 h-4" />}
            onClick={handleCreate}
          >
            新建分类
          </Button>
          <Button
            icon={<Upload className="w-4 h-4" />}
            onClick={() => setImportModalVisible(true)}
          >
            批量导入
          </Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总分类数"
              value={stats.total}
              prefix={<Folder className="w-5 h-5 text-blue-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="启用分类"
              value={stats.active}
              prefix={<Eye className="w-5 h-5 text-green-500" />}
              valueStyle={{ color: "#3f8600" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="禁用分类"
              value={stats.inactive}
              prefix={<EyeOff className="w-5 h-5 text-red-500" />}
              valueStyle={{ color: "#cf1322" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="顶级分类"
              value={stats.root}
              prefix={<FolderOpen className="w-5 h-5 text-orange-500" />}
              valueStyle={{ color: "#fa8c16" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-4">
            <Input
              placeholder="搜索分类名称、代码或描述"
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              style={{ width: 300 }}
            />
            <Select
              placeholder="状态筛选"
              value={filterActive}
              onChange={setFilterActive}
              style={{ width: 120 }}
              allowClear
            >
              <Option value={true}>启用</Option>
              <Option value={false}>禁用</Option>
            </Select>
          </div>
          <div className="flex items-center space-x-2">
            <Button
              type={viewMode === "tree" ? "primary" : "default"}
              icon={<Folder className="w-4 h-4" />}
              onClick={() => setViewMode("tree")}
            >
              树形视图
            </Button>
            <Button
              type={viewMode === "list" ? "primary" : "default"}
              icon={<FileText className="w-4 h-4" />}
              onClick={() => setViewMode("list")}
            >
              列表视图
            </Button>
            <Button
              type={viewMode === "drag" ? "primary" : "default"}
              icon={<GripVertical className="w-4 h-4" />}
              onClick={() => setViewMode("drag")}
            >
              拖拽排序
            </Button>
            <Button
              icon={<Download className="w-4 h-4" />}
              onClick={() => setExportModalVisible(true)}
            >
              导出分类
            </Button>
          </div>
        </div>

        {/* 内容区域 */}
        {viewMode === "tree" ? (
          <div className="border rounded-lg p-4">
            {categories.length > 0 ? (
              <Tree
                treeData={buildTreeData(flatCategories)}
                expandedKeys={expandedKeys}
                selectedKeys={selectedKeys}
                onExpand={setExpandedKeys}
                onSelect={setSelectedKeys}
                showLine
                showIcon={false}
                className="category-tree"
              />
            ) : (
              <Empty description="暂无分类数据" />
            )}
          </div>
        ) : viewMode === "drag" ? (
          <TicketCategoryDragSort
            onSave={(updatedCategories: CategoryTreeItem[]) => {
              setCategories(updatedCategories);
              loadCategories();
            }}
          />
        ) : (
          <div className="border rounded-lg">
            {filteredCategories.length > 0 ? (
              <div className="divide-y">
                {filteredCategories.map((category) => (
                  <div
                    key={category.id}
                    className="p-4 hover:bg-gray-50 flex items-center justify-between"
                  >
                    <div className="flex items-center space-x-4">
                      <div>
                        <div className="font-medium">{category.name}</div>
                        <div className="text-sm text-gray-500">
                          {category.description || "暂无描述"}
                        </div>
                        <div className="text-xs text-gray-400 mt-1">
                          代码: {category.code} | 层级: {category.level} | 排序:{" "}
                          {category.sort_order}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <Switch
                        checked={category.is_active}
                        size="small"
                        onChange={async (checked) => {
                          try {
                            await ticketCategoryService.updateCategory(
                              category.id,
                              { is_active: checked }
                            );
                            loadCategories();
                          } catch {
                            message.error("状态更新失败");
                          }
                        }}
                      />
                      <Button
                        type="text"
                        size="small"
                        icon={<Edit className="w-4 h-4" />}
                        onClick={() => handleEdit(category)}
                      />
                      <Popconfirm
                        title="确定要删除这个分类吗？"
                        description="删除后无法恢复，请谨慎操作。"
                        onConfirm={() => handleDelete(category.id)}
                        okText="确定"
                        cancelText="取消"
                      >
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<Delete className="w-4 h-4" />}
                        />
                      </Popconfirm>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <Empty description="暂无符合条件的分类" />
            )}
          </div>
        )}
      </Card>

      {/* 创建/编辑分类模态框 */}
      <Modal
        title={editingCategory ? "编辑分类" : "新建分类"}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingCategory(null);
          form.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            parent_id: 0,
            sort_order: 0,
            is_active: true,
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="分类名称"
                rules={[{ required: true, message: "请输入分类名称" }]}
              >
                <Input placeholder="请输入分类名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="code"
                label="分类代码"
                rules={[{ required: true, message: "请输入分类代码" }]}
              >
                <Input placeholder="请输入分类代码" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="description" label="分类描述">
            <TextArea rows={3} placeholder="请输入分类描述（可选）" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="parent_id" label="父分类">
                <Select placeholder="选择父分类">
                  <Option value={0}>顶级分类</Option>
                  {flatCategories
                    .filter((c) => c.id !== editingCategory?.id)
                    .map((category) => (
                      <Option key={category.id} value={category.id}>
                        {category.name}
                      </Option>
                    ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="sort_order" label="排序顺序">
                <InputNumber
                  min={0}
                  placeholder="排序顺序"
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="is_active" label="是否启用" valuePropName="checked">
            <Switch />
          </Form.Item>

          <div className="flex justify-end space-x-2">
            <Button
              onClick={() => {
                setModalVisible(false);
                setEditingCategory(null);
                form.resetFields();
              }}
            >
              取消
            </Button>
            <Button type="primary" htmlType="submit">
              {editingCategory ? "更新" : "创建"}
            </Button>
          </div>
        </Form>
      </Modal>

      {/* 批量导入模态框 */}
      <TicketCategoryImport
        visible={importModalVisible}
        onCancel={() => setImportModalVisible(false)}
        onSuccess={() => {
          setImportModalVisible(false);
          loadCategories();
        }}
      />

      {/* 导出模态框 */}
      <TicketCategoryExport
        visible={exportModalVisible}
        onCancel={() => setExportModalVisible(false)}
        onSuccess={() => {
          setExportModalVisible(false);
        }}
      />
    </div>
  );
};

export default TicketCategoriesPage;
