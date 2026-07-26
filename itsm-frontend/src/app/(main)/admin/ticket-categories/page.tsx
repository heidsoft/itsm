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
  message,
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

const { TextArea } = Input;
const { Title, Text } = Typography;
const { Option } = Select;

interface TicketCategory {
  id: number;
  name: string;
  code: string;
  description: string;
  parentId: number | null;
  level: number;
  sortOrder: number;
  isActive: boolean;
  workflowId: number | null;
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
  workflowId?: number;
  departmentId?: number;
}

interface WorkflowOption {
  id: number;
  name: string;
}

interface DepartmentOption {
  id: number;
  name: string;
}

const TicketCategoryManagementPage = () => {
  const [categories, setCategories] = useState<TicketCategory[]>([]);
  const [workflows, setWorkflows] = useState<WorkflowOption[]>([]);
  const [departments, setDepartments] = useState<DepartmentOption[]>([]);
  const [loading, setLoading] = useState(false);
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
    setLoading(true);
    try {
      const data = await TicketCategoryApi.getCategories({ page: 1, pageSize: 500 });
      const list = normalizeCategories(data.categories || data.items || []);
      setCategories(list);
    } catch (error) {
      console.error('Failed to load categories:', error);
      message.error('加载分类失败');
      setCategories([]);
    } finally {
      setLoading(false);
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
          (workflowRes.value?.workflows || []).map(w => ({ id: Number(w.id), name: w.name }))
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
    workflowId: category.workflowId ?? null,
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
      workflowId: category.workflowId ?? undefined,
      departmentId: category.departmentId ?? undefined,
    });
    setModalVisible(true);
  };

  const handleCopyCategory = async (category: TicketCategory) => {
    try {
      await TicketCategoryApi.createCategory({
        name: `${category.name} - 副本`,
        code: `${category.code}_COPY_${Date.now().toString(36).toUpperCase()}`,
        description: category.description,
        parentId: category.parentId || undefined,
        sortOrder: category.sortOrder,
        isActive: category.isActive,
        workflowId: category.workflowId || undefined,
        departmentId: category.departmentId || undefined,
      });
      message.success('分类复制成功');
      loadCategories();
    } catch (error) {
      console.error('Failed to copy category:', error);
      message.error(error instanceof Error ? error.message : '复制分类失败');
    }
  };

  const handleDeleteCategory = async (id: number) => {
    try {
      await TicketCategoryApi.deleteCategory(id);
      message.success('分类删除成功');
      loadCategories();
    } catch (error) {
      console.error('Failed to delete category:', error);
      // 展示后端保护性错误信息（如：无法删除有子分类的分类 / 无法删除正在使用的分类）
      message.error(error instanceof Error ? error.message : '删除失败');
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
          workflowId: values.workflowId ?? 0,
          departmentId: values.departmentId ?? 0,
        });
        message.success('分类更新成功');
      } else {
        await TicketCategoryApi.createCategory({
          name: values.name,
          code: values.code,
          description: values.description,
          parentId: values.parentId,
          sortOrder: values.sortOrder,
          isActive: values.isActive ?? true,
          workflowId: values.workflowId,
          departmentId: values.departmentId,
        });
        message.success('分类创建成功');
      }

      setModalVisible(false);
      form.resetFields();
      loadCategories();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
      console.error('保存分类失败:', error);
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
      message.success(checked ? '分类已启用' : '分类已停用');
    } catch (error) {
      console.error('Failed to update category status:', error);
      message.error('状态更新失败');
    }
  };

  const workflowNameOf = (id: number | null) =>
    id ? workflows.find(w => w.id === id)?.name || `工作流 #${id}` : null;
  const departmentNameOf = (id: number | null) =>
    id ? departments.find(d => d.id === id)?.name || `部门 #${id}` : null;

  const handleViewCategory = (category: TicketCategory) => {
    Modal.info({
      title: '分类详情',
      width: 560,
      content: (
        <div className="space-y-2 pt-2">
          <p>
            <Text strong>名称：</Text>
            {category.name}
          </p>
          <p>
            <Text strong>编码：</Text>
            {category.code || '-'}
          </p>
          <p>
            <Text strong>描述：</Text>
            {category.description || '-'}
          </p>
          <p>
            <Text strong>关联工作流：</Text>
            {workflowNameOf(category.workflowId) || '-'}
          </p>
          <p>
            <Text strong>所属部门：</Text>
            {departmentNameOf(category.departmentId) || '-'}
          </p>
          <p>
            <Text strong>状态：</Text>
            {category.isActive ? '启用' : '停用'}
          </p>
        </div>
      ),
    });
  };

  const columns = [
    {
      title: '分类名称',
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
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      render: (text: string) => (
        <div className="max-w-xs truncate" title={text}>
          {text || '-'}
        </div>
      ),
    },
    {
      title: '层级',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (level: number) => <Tag color={level === 1 ? 'blue' : 'cyan'}>{level} 级</Tag>,
    },
    {
      title: '排序',
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
      title: '关联工作流',
      dataIndex: 'workflowId',
      key: 'workflowId',
      render: (workflowId: number | null) => {
        const name = workflowNameOf(workflowId);
        return name ? <Tag color="purple">{name}</Tag> : <Text type="secondary">-</Text>;
      },
    },
    {
      title: '所属部门',
      dataIndex: 'departmentId',
      key: 'departmentId',
      render: (departmentId: number | null) => {
        const name = departmentNameOf(departmentId);
        return name ? <Tag color="geekblue">{name}</Tag> : <Text type="secondary">-</Text>;
      },
    },
    {
      title: '状态',
      key: 'status',
      width: 90,
      render: (record: TicketCategory) => (
        <Switch
          checked={record.isActive}
          size="small"
          onChange={checked => handleToggleStatus(record, checked)}
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 160,
      render: (record: TicketCategory) => (
        <Space>
          <Tooltip title="查看详情">
            <Button size="small" icon={<Eye size={14} />} onClick={() => handleViewCategory(record)} />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              size="small"
              icon={<Edit size={14} />}
              onClick={() => handleEditCategory(record)}
            />
          </Tooltip>
          <Tooltip title="复制">
            <Button
              size="small"
              icon={<Copy size={14} />}
              onClick={() => handleCopyCategory(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个分类吗？"
            onConfirm={() => handleDeleteCategory(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button size="small" danger icon={<Delete size={14} />} />
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
          {!category.isActive && <Tag color="default">已停用</Tag>}
        </div>
        <div className="flex items-center gap-1">
          <Switch
            checked={category.isActive}
            size="small"
            onChange={checked => handleToggleStatus(category, checked)}
          />
          <Button
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

  if (loading) {
    return <LoadingSkeleton type="table" rows={8} columns={7} />;
  }

  return (
    <div className="space-y-6">
      {/* 头部操作区 */}
      <Card>
        <div className="flex justify-between items-center">
          <div>
            <Title level={4} className="mb-1">
              工单分类管理
            </Title>
            <Text type="secondary">管理和配置工单分类体系，支持树形结构和工作流绑定</Text>
          </div>
          <Space>
            <Button.Group>
              <Button
                type={viewMode === 'table' ? 'primary' : 'default'}
                icon={<TableIcon size={16} />}
                onClick={() => setViewMode('table')}
              >
                表格视图
              </Button>
              <Button
                type={viewMode === 'tree' ? 'primary' : 'default'}
                icon={<Folder />}
                onClick={() => setViewMode('tree')}
              >
                树形视图
              </Button>
            </Button.Group>
            <Button type="primary" icon={<Plus size={16} />} onClick={handleCreateCategory}>
              创建分类
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
                <Empty description="暂无工单分类">
                  <Button type="primary" onClick={handleCreateCategory}>
                    创建分类
                  </Button>
                </Empty>
              ),
            }}
            pagination={{
              pageSize: 20,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
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
          <Empty description="暂无工单分类">
            <Button type="primary" onClick={handleCreateCategory}>
              创建分类
            </Button>
          </Empty>
        )}
      </Card>

      {/* 创建/编辑分类模态框 */}
      <Modal
        title={editingCategory ? '编辑分类' : '创建分类'}
        open={modalVisible}
        onOk={handleSaveCategory}
        onCancel={() => setModalVisible(false)}
        okText="保存"
        cancelText="取消"
        confirmLoading={saving}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="分类名称"
                rules={[{ required: true, message: '请输入分类名称' }]}
              >
                <Input placeholder="请输入分类名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="code"
                label="分类编码"
                tooltip="租户内唯一，创建后作为分类的稳定标识"
                rules={[
                  { required: true, message: '请输入分类编码' },
                  {
                    pattern: /^[A-Za-z][A-Za-z0-9_-]*$/,
                    message: '以字母开头，仅允许字母、数字、下划线和连字符',
                  },
                ]}
              >
                <Input placeholder="如 INCIDENT_NETWORK" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="parentId" label="父级分类">
            <TreeSelect
              placeholder="不选择则为顶级分类"
              allowClear
              treeDefaultExpandAll
              treeData={parentTreeData}
            />
          </Form.Item>

          <Form.Item name="description" label="分类描述">
            <TextArea rows={3} placeholder="请输入分类描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="workflowId" label="关联工作流" tooltip="该分类下的工单将默认走此工作流">
                <Select placeholder="请选择工作流（可选）" allowClear showSearch optionFilterProp="children">
                  {workflows.map(w => (
                    <Option key={w.id} value={w.id}>
                      {w.name}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="departmentId" label="所属部门">
                <Select placeholder="请选择部门（可选）" allowClear showSearch optionFilterProp="children">
                  {departments.map(d => (
                    <Option key={d.id} value={d.id}>
                      {d.name}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="sortOrder" label="排序顺序" initialValue={1}>
                <InputNumber min={0} style={{ width: '100%' }} placeholder="数字越小越靠前" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="isActive"
                label="启用状态"
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
