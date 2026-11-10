'use client';

import React, { useState, useEffect, useMemo } from 'react';
import {
  Card,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Space,
  Tag,
  App,
  Popconfirm,
  Tooltip,
  Tree,
  TreeSelect,
  Row,
  Col,
  Typography,
  Switch,
  InputNumber,
  Table,
  Empty,
} from 'antd';
import {
  Plus,
  Edit,
  Delete,
  Eye,
  Folder,
  GripVertical,
  Copy,
  Table as TableIcon,
} from 'lucide-react';
import { LoadingSkeleton } from '@/components/ui/LoadingSkeleton';
import { TicketCategoryApi } from '@/lib/api/ticket-category-api';
import { WorkflowApi } from '@/lib/api/workflow-api';
import { CommonApi } from '@/lib/api/common-api';
import { buildCategoryTree, collectDescendantIds } from './categoryTreeUtils';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';

const { TextArea } = Input;
const { Title, Text } = Typography;

interface TicketCategory {
  id: number;
  name: string;
  code: string;
  description: string;
  parentId: number | null;
  level: number;
  sortOrder: number;
  isActive: boolean;
  workflowDefinitionKey: string;
  departmentId: number | null;
  createdAt: string;
  updatedAt: string;
  children?: TicketCategory[];
}

type RawTicketCategory = Partial<Omit<TicketCategory, 'children'>> & {
  children?: RawTicketCategory[];
};

// 父级分类 TreeSelect 节点
interface ParentTreeNode {
  title: string;
  value: number;
  disabled: boolean;
  children?: ParentTreeNode[];
}

// 树形视图节点
interface CategoryTreeNode {
  key: number;
  title: React.ReactNode;
  children?: CategoryTreeNode[];
}

interface CategoryFormValues {
  name: string;
  code: string;
  description?: string;
  parentId?: number;
  sortOrder?: number;
  isActive?: boolean;
  workflowDefinitionKey?: string;
  departmentId?: number;
}

interface WorkflowOption {
  key: string;
  name: string;
}

interface DepartmentOption {
  id: number;
  name: string;
}

const TicketCategoryManagementPage = () => {
  const { message } = App.useApp();
  const [categories, setCategories] = useState<TicketCategory[]>([]);
  const [workflows, setWorkflows] = useState<WorkflowOption[]>([]);
  const [departments, setDepartments] = useState<DepartmentOption[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 空表格，再被真实数据覆盖。保存动作另用 saving。
  const [listLoading, setListLoading] = useState(true);
  const [modalVisible, setModalVisible] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editingCategory, setEditingCategory] = useState<TicketCategory | null>(null);
  const [form] = Form.useForm<CategoryFormValues>();
  const [viewMode, setViewMode] = useState<'table' | 'tree'>('table');

  useEffect(() => {
    loadCategories();
    loadBindingOptions();
  }, []);

  const loadCategories = async () => {
    setListLoading(true);
    try {
      const data = await TicketCategoryApi.getCategories({ page: 1, pageSize: BACKEND_MAX_PAGE_SIZE });
      const list = normalizeCategories(data.items);
      setCategories(list);
    } catch (error) {
      console.error('Failed to load categories:', error);
      message.error('Failed to load category');
      setCategories([]);
    } finally {
      setListLoading(false);
    }
  };

  const loadBindingOptions = async () => {
    try {
      const [workflowRes, departmentRes] = await Promise.allSettled([
        WorkflowApi.getWorkflows({ page: 1, pageSize: 100 }),
        CommonApi.getDepartments(),
      ]);
      if (workflowRes.status === 'fulfilled') {
        setWorkflows(
          (workflowRes.value?.workflows || []).map(w => ({ key: String(w.id), name: w.name }))
        );
      }
      if (departmentRes.status === 'fulfilled') {
        const list = Array.isArray(departmentRes.value) ? departmentRes.value : [];
        setDepartments(
          list
            .filter((d: { id?: number; name?: string }) => d?.id && d?.name)
            .map((d: { id: number; name: string }) => ({ id: d.id, name: d.name }))
        );
      }
    } catch (error) {
      // 绑定选项加载失败不阻塞分类管理主流程
      console.error('Failed to load binding options:', error);
    }
  };

  const normalizeCategory = (category: RawTicketCategory): TicketCategory => ({
    id: category.id || 0,
    name: category.name || '',
    code: category.code || '',
    description: category.description || '',
    parentId: category.parentId ?? null,
    level: category.level || 1,
    sortOrder: category.sortOrder ?? 0,
    isActive: category.isActive ?? true,
    workflowDefinitionKey: category.workflowDefinitionKey || '',
    departmentId: category.departmentId ?? null,
    createdAt: category.createdAt || '',
    updatedAt: category.updatedAt || '',
  });

  const normalizeCategories = (list: RawTicketCategory[]) => list.map(normalizeCategory);

  const categoryTree = useMemo(() => buildCategoryTree(categories), [categories]);

  // 父级分类 TreeSelect 数据：编辑时禁用自身及全部后代，防止形成环
  const parentTreeData = useMemo(() => {
    const disabledIds = editingCategory
      ? collectDescendantIds(categories, editingCategory.id)
      : new Set<number>();
    const mapNode = (node: TicketCategory): ParentTreeNode => ({
      title: node.name,
      value: node.id,
      disabled: disabledIds.has(node.id),
      children: node.children?.map(mapNode),
    });
    return categoryTree.map(mapNode);
  }, [categoryTree, categories, editingCategory]);

  const handleCreateCategory = () => {
    setEditingCategory(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEditCategory = (category: TicketCategory) => {
    setEditingCategory(category);
    form.setFieldsValue({
      name: category.name,
      code: category.code,
      description: category.description,
      parentId: category.parentId ?? undefined,
      sortOrder: category.sortOrder,
      isActive: category.isActive,
      workflowDefinitionKey: category.workflowDefinitionKey || undefined,
      departmentId: category.departmentId ?? undefined,
    });
    setModalVisible(true);
  };

  const handleCopyCategory = async (category: TicketCategory) => {
    try {
      await TicketCategoryApi.createCategory({
        name: `${category.name} - copy`,
        code: `${category.code}_COPY_${Date.now().toString(36).toUpperCase()}`,
        description: category.description,
        parentId: category.parentId || undefined,
        sortOrder: category.sortOrder,
        isActive: category.isActive,
        workflowDefinitionKey: category.workflowDefinitionKey || undefined,
        departmentId: category.departmentId || undefined,
      });
      message.success('Category copy successful');
      loadCategories();
    } catch (error) {
      console.error('Failed to copy category:', error);
      message.error(error instanceof Error ? error.message : 'Failed to copy category');
    }
  };

  const handleDeleteCategory = async (id: number) => {
    try {
      await TicketCategoryApi.deleteCategory(id);
      message.success('CategoryDeleteSuccess');
      loadCategories();
    } catch (error) {
      console.error('Failed to delete category:', error);
      // 展示后端保护性错误信息（如：无法删除有子分类的分类 / 无法删除正在使用的分类）
      message.error(error instanceof Error ? error.message : 'Delete failed');
    }
  };

  const handleSaveCategory = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);

      if (editingCategory) {
        // 更新分类：未选择即视为清除绑定（传 0 由后端 Clear）
        await TicketCategoryApi.updateCategory(editingCategory.id, {
          name: values.name,
          code: values.code,
          description: values.description,
          parentId: values.parentId ?? 0,
          sortOrder: values.sortOrder,
          isActive: values.isActive,
          workflowDefinitionKey: values.workflowDefinitionKey || '',
          departmentId: values.departmentId ?? 0,
        });
        message.success('CategoryUpdateSuccess');
      } else {
        await TicketCategoryApi.createCategory({
          name: values.name,
          code: values.code,
          description: values.description,
          parentId: values.parentId,
          sortOrder: values.sortOrder,
          isActive: values.isActive ?? true,
          workflowDefinitionKey: values.workflowDefinitionKey,
          departmentId: values.departmentId,
        });
        message.success('CategoryCreateSuccess');
      }

      setModalVisible(false);
      form.resetFields();
      loadCategories();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message ? error.message : 'Failed to save category');
      }
      console.error('Failed to save category:', error);
    } finally {
      setSaving(false);
    }
  };

  const handleToggleStatus = async (category: TicketCategory, checked: boolean) => {
    try {
      await TicketCategoryApi.updateCategory(category.id, { isActive: checked });
      setCategories(prev =>
        prev.map(c => (c.id === category.id ? { ...c, isActive: checked } : c))
      );
      message.success(checked ? 'Classification is enabled' : 'Classification is disabled');
    } catch (error) {
      console.error('Failed to update category status:', error);
      message.error('Failed to update status');
    }
  };

  const workflowNameOf = (key: string) =>
    key ? workflows.find(w => w.key === key)?.name || key : null;
  const departmentNameOf = (id: number | null) =>
    id ? departments.find(d => d.id === id)?.name || `Department #${id}` : null;

  const handleViewCategory = (category: TicketCategory) => {
    Modal.info({
      title: 'Classification details',
      width: 560,
      content: (
        <div className="space-y-2 pt-2">
          <p>
            <Text strong>Name:</Text>
            {category.name}
          </p>
          <p>
            <Text strong>Encoding:</Text>
            {category.code || '-'}
          </p>
          <p>
            <Text strong>Description:</Text>
            {category.description || '-'}
          </p>
          <p>
            <Text strong>Associated workflow:</Text>
            {workflowNameOf(category.workflowDefinitionKey) || '-'}
          </p>
          <p>
            <Text strong>Department:</Text>
            {departmentNameOf(category.departmentId) || '-'}
          </p>
          <p>
            <Text strong>Status:</Text>
            {category.isActive ? 'Enabled' : 'Disable'}
          </p>
        </div>
      ),
    });
  };

  const columns = [
    {
      title: 'Category Name',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: TicketCategory) => (
        <div>
          <div className="font-medium">{text}</div>
          <div className="text-sm text-gray-500">{record.code}</div>
        </div>
      ),
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      render: (text: string) => (
        <div className="max-w-xs truncate" title={text}>
          {text || '-'}
        </div>
      ),
    },
    {
      title: 'Hierarchy',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (level: number) => <Tag color={level === 1 ? 'blue' : 'cyan'}>{level}  level</Tag>,
    },
    {
      title: 'Sort Order',
      dataIndex: 'sortOrder',
      key: 'sortOrder',
      width: 90,
      render: (order: number) => (
        <div className="flex items-center gap-1">
          <GripVertical size={14} className="text-gray-400" />
          <span>{order}</span>
        </div>
      ),
    },
    {
      title: 'Associated Workflow',
      dataIndex: 'workflowDefinitionKey',
      key: 'workflowDefinitionKey',
      render: (workflowDefinitionKey: string) => {
        const name = workflowNameOf(workflowDefinitionKey);
        return name ? <Tag color="purple">{name}</Tag> : <Text type="secondary">-</Text>;
      },
    },
    {
      title: 'Department',
      dataIndex: 'departmentId',
      key: 'departmentId',
      render: (departmentId: number | null) => {
        const name = departmentNameOf(departmentId);
        return name ? <Tag color="geekblue">{name}</Tag> : <Text type="secondary">-</Text>;
      },
    },
    {
      title: 'Status',
      key: 'status',
      width: 90,
      render: (record: TicketCategory) => (
        <Switch
          aria-label={`Switch category ${record.name} enabled status`}
          checked={record.isActive}
          size="small"
          onChange={checked => handleToggleStatus(record, checked)}
        />
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 160,
      render: (record: TicketCategory) => (
        <Space>
          <Tooltip title="View Details">
            <Button aria-label="View Details" size="small" icon={<Eye size={14} />} onClick={() => handleViewCategory(record)} />
          </Tooltip>
          <Tooltip title="Edit">
            <Button
              aria-label="Edit"
              size="small"
              icon={<Edit size={14} />}
              onClick={() => handleEditCategory(record)}
            />
          </Tooltip>
          <Tooltip title="Copy">
            <Button
              aria-label="Copy"
              size="small"
              icon={<Copy size={14} />}
              onClick={() => handleCopyCategory(record)}
            />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this category?"
            onConfirm={() => handleDeleteCategory(record.id)}
            okText="OK"
            cancelText="Cancel"
          >
            <Button aria-label="Delete" size="small" danger icon={<Delete size={14} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 树形视图：递归渲染任意层级
  const buildTreeNode = (category: TicketCategory): CategoryTreeNode => ({
    key: category.id,
    title: (
      <div className="flex items-center justify-between w-full">
        <div className="flex items-center gap-2">
          <span className="font-medium">{category.name}</span>
          <Text type="secondary" className="text-xs">
            {category.code}
          </Text>
          {!category.isActive && <Tag color="default">Disabled</Tag>}
        </div>
        <div className="flex items-center gap-1">
          <Switch
            aria-label={`Switch category ${category.name} enabled status`}
            checked={category.isActive}
            size="small"
            onChange={checked => handleToggleStatus(category, checked)}
          />
          <Button
            aria-label="Edit"
            size="small"
            icon={<Edit size={12} />}
            onClick={e => {
              e.stopPropagation();
              handleEditCategory(category);
            }}
          />
        </div>
      </div>
    ),
    children: category.children?.map(buildTreeNode),
  });

  const treeData = categoryTree.map(buildTreeNode);

  if (listLoading) {
    return <LoadingSkeleton type="table" rows={8} columns={7} />;
  }

  return (
    <div className="space-y-6">
      {/* 头部操作区 */}
      <Card>
        <div className="flex justify-between items-center">
          <div>
            <Title level={4} className="mb-1">

              Work order classification management
            </Title>
            <Text type="secondary">Manage and configure the work order classification system, supporting tree structure and workflow binding</Text>
          </div>
          <Space>
            <Button.Group>
              <Button
                type={viewMode === 'table' ? 'primary' : 'default'}
                icon={<TableIcon size={16} />}
                onClick={() => setViewMode('table')}
              >

                Table view
              </Button>
              <Button
                type={viewMode === 'tree' ? 'primary' : 'default'}
                icon={<Folder />}
                onClick={() => setViewMode('tree')}
              >

                Tree view
              </Button>
            </Button.Group>
            <Button type="primary" icon={<Plus size={16} />} onClick={handleCreateCategory}>

              Create category
            </Button>
          </Space>
        </div>
      </Card>

      {/* 分类内容 */}
      <Card>
        {viewMode === 'table' ? (
          <Table
            columns={columns}
            dataSource={categoryTree}
            rowKey="id"
            scroll={{ x: 960 }}
            expandable={{ defaultExpandAllRows: true }}
            locale={{
              emptyText: (
                <Empty description="No work order classification yet">
                  <Button type="primary" onClick={handleCreateCategory}>

                    Create category
                  </Button>
                </Empty>
              ),
            }}
            pagination={{
              pageSize: 20,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `Total ${total} records`,
            }}
          />
        ) : treeData.length > 0 ? (
          <Tree
            treeData={treeData}
            defaultExpandAll
            showLine
            showIcon={false}
            className="category-tree"
          />
        ) : (
          <Empty description="No work order classification yet">
            <Button type="primary" onClick={handleCreateCategory}>

              Create category
            </Button>
          </Empty>
        )}
      </Card>

      {/* 创建/编辑分类模态框 */}
      <Modal
        title={editingCategory ? 'Edit Category' : 'Create category'}
        open={modalVisible}
        onOk={handleSaveCategory}
        onCancel={() => setModalVisible(false)}
        okText="Save"
        cancelText="Cancel"
        confirmLoading={saving}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="Category Name"
                rules={[{ required: true, message: 'Please enter the category name' }]}
              >
                <Input placeholder="Please enter the category name" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="code"
                label="Classification code"
                tooltip="unique within the tenant and used as a stable identifier for classification after creation"
                rules={[
                  { required: true, message: 'Please enter the classification code' },
                  {
                    pattern: /^[A-Za-z][A-Za-z0-9_-]*$/,
                    message: 'Starts with a letter, only letters, numbers, underscores and hyphens are allowed',
                  },
                ]}
              >
                <Input placeholder="Such as INCIDENT_NETWORK" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="parentId" label="Parent category">
            <TreeSelect
              placeholder="If not selected, it will be the top category"
              allowClear
              treeDefaultExpandAll
              treeData={parentTreeData}
            />
          </Form.Item>

          <Form.Item name="description" label="Category Description">
            <TextArea rows={3} placeholder="Please enter a category description" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="workflowDefinitionKey"
                label="Associated Workflow"
                tooltip="Work orders under this category follow this workflow by default; when the work order type has configured workflow, the work order type shall prevail."
              >
                <Select placeholder="Please select a workflow (optional)" allowClear showSearch optionFilterProp="children" options={workflows.map(w => ({ value: w.key, label: w.name }))} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="departmentId" label="Department">
                <Select placeholder="Please select a department (optional)" allowClear showSearch optionFilterProp="children" options={departments.map(d => ({ value: d.id, label: d.name }))} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="sortOrder" label="Sort order" initialValue={1}>
                <InputNumber min={0} style={{ width: '100%' }} placeholder="The smaller the number, the higher it is." />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="isActive"
                label="Enabled"
                valuePropName="checked"
                initialValue={true}
              >
                <Switch />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
};

export default TicketCategoryManagementPage;
