'use client';

import React, { useState, useEffect } from 'react';
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

const { TextArea } = Input;
const { Title, Text } = Typography;
const { Option } = Select;

interface TicketCategory {
  id: number;
  name: string;
  description: string;
  parent_id: number | null;
  level: number;
  path: string;
  sort_order: number;
  is_active: boolean;
  is_default: boolean;
  color: string;
  icon?: string;
  created_at: string;
  updated_at: string;
  children?: TicketCategory[];
  ticket_count?: number;
}

type RawTicketCategory = Partial<TicketCategory> & {
  parentId?: number | null;
  sortOrder?: number;
  isActive?: boolean;
  isDefault?: boolean;
  ticketCount?: number;
  createdAt?: string;
  updatedAt?: string;
};

interface TicketCategoryFormData {
  name: string;
  description: string;
  parent_id?: number;
  color: string;
  icon?: string;
  sort_order: number;
  is_active: boolean;
}

const TicketCategoryManagementPage = () => {
  const [categories, setCategories] = useState<TicketCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState<TicketCategory | null>(null);
  const [form] = Form.useForm();
  const [viewMode, setViewMode] = useState<'table' | 'tree'>('table');

  // 模拟加载分类数据
  useEffect(() => {
    loadCategories();
  }, []);

  const loadCategories = async () => {
    setLoading(true);
    try {
      const data = await TicketCategoryApi.getCategories();
      // Handle backend response format: {categories: [...], total: X} or {items: [...], total: X}
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

  const normalizeCategory = (category: RawTicketCategory): TicketCategory => ({
    id: category.id || 0,
    name: category.name || '',
    description: category.description || '',
    parent_id: category.parent_id ?? category.parentId ?? null,
    level: category.level || 1,
    path: category.path || category.name || '',
    sort_order: category.sort_order ?? category.sortOrder ?? 1,
    is_active: category.is_active ?? category.isActive ?? true,
    is_default: category.is_default ?? category.isDefault ?? false,
    color: category.color || '#1890ff',
    icon: category.icon,
    created_at: category.created_at || category.createdAt || '',
    updated_at: category.updated_at || category.updatedAt || '',
    ticket_count: category.ticket_count ?? category.ticketCount ?? 0,
    children: category.children ? normalizeCategories(category.children) : undefined,
  });

  const normalizeCategories = (list: RawTicketCategory[]) => list.map(normalizeCategory);

  const handleCreateCategory = () => {
    setEditingCategory(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEditCategory = (category: TicketCategory) => {
    setEditingCategory(category);
    form.setFieldsValue({
      name: category.name,
      description: category.description,
      parent_id: category.parent_id,
      color: category.color,
      icon: category.icon,
      sort_order: category.sort_order,
      is_active: category.is_active,
    });
    setModalVisible(true);
  };

  const handleCopyCategory = async (category: TicketCategory) => {
    try {
      await TicketCategoryApi.createCategory({
        name: `${category.name} - 副本`,
        description: category.description,
        parent_id: category.parent_id || undefined,
        sort_order: category.sort_order,
        is_active: category.is_active,
        is_default: false,
        color: category.color,
        icon: category.icon,
      });
      message.success('分类复制成功');
      loadCategories();
    } catch (error) {
      console.error('Failed to copy category:', error);
      message.error('复制分类失败');
    }
  };

  const handleDeleteCategory = async (id: number) => {
    try {
      await TicketCategoryApi.deleteCategory(id);
      message.success('分类删除成功');
      loadCategories();
    } catch (error) {
      console.error('Failed to delete category:', error);
      message.error('删除失败');
    }
  };

  const handleSaveCategory = async () => {
    try {
      const values = await form.validateFields();

      if (editingCategory) {
        // 更新分类
        await TicketCategoryApi.updateCategory(editingCategory.id, values);
        message.success('分类更新成功');
      } else {
        // 创建新分类
        await TicketCategoryApi.createCategory(values);
        message.success('分类创建成功');
      }

      setModalVisible(false);
      form.resetFields();
      loadCategories();
    } catch (error) {
      console.error('保存分类失败:', error);
    }
  };

  const handleToggleStatus = async (category: TicketCategory, checked: boolean) => {
    try {
      await TicketCategoryApi.updateCategory(category.id, { is_active: checked });
      setCategories(prev =>
        prev.map(c => (c.id === category.id ? { ...c, is_active: checked } : c))
      );
      message.success(checked ? '分类已启用' : '分类已停用');
    } catch (error) {
      console.error('Failed to update category status:', error);
      message.error('状态更新失败');
    }
  };

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
            <Text strong>路径：</Text>
            {category.path || '-'}
          </p>
          <p>
            <Text strong>描述：</Text>
            {category.description || '-'}
          </p>
          <p>
            <Text strong>状态：</Text>
            {category.is_active ? '启用' : '停用'}
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
        <div className="flex items-center gap-2">
          <span className="text-lg">{record.icon}</span>
          <div>
            <div className="font-medium">{text}</div>
            <div className="text-sm text-gray-500">{record.path}</div>
          </div>
        </div>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      render: (text: string) => (
        <div className="max-w-xs truncate" title={text}>
          {text}
        </div>
      ),
    },
    {
      title: '层级',
      dataIndex: 'level',
      key: 'level',
      render: (level: number) => (
        <Tag color={level === 1 ? 'blue' : 'cyan'}>{level === 1 ? '一级' : '二级'}</Tag>
      ),
    },
    {
      title: '排序',
      dataIndex: 'sort_order',
      key: 'sort_order',
      render: (order: number) => (
        <div className="flex items-center gap-1">
          <GripVertical size={14} className="text-gray-400" />
          <span>{order}</span>
        </div>
      ),
    },
    {
      title: '工单数量',
      dataIndex: 'ticket_count',
      key: 'ticket_count',
      render: (count: number) => <Tag color="orange">{count}</Tag>,
    },
    {
      title: '状态',
      key: 'status',
      render: (record: TicketCategory) => (
        <div className="flex items-center gap-2">
          <Switch
            checked={record.is_active}
            size="small"
            onChange={checked => handleToggleStatus(record, checked)}
          />
          {record.is_default && <Tag color="green">默认</Tag>}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
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

  const treeData = (categories || []).map(category => ({
    key: category.id,
    title: (
      <div className="flex items-center justify-between w-full">
        <div className="flex items-center gap-2">
          <span className="text-lg">{category.icon}</span>
          <span className="font-medium">{category.name}</span>
          <Tag color="orange">{category.ticket_count}</Tag>
        </div>
        <div className="flex items-center gap-1">
          <Switch
            checked={category.is_active}
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
    children:
      category.children?.map(child => ({
        key: child.id,
        title: (
          <div className="flex items-center justify-between w-full">
            <div className="flex items-center gap-2">
              <span className="text-lg">{child.icon}</span>
              <span>{child.name}</span>
              <Tag color="orange">{child.ticket_count}</Tag>
            </div>
            <div className="flex items-center gap-1">
              <Switch
                checked={child.is_active}
                size="small"
                onChange={checked => handleToggleStatus(child, checked)}
              />
              <Button
                size="small"
                icon={<Edit size={12} />}
                onClick={e => {
                  e.stopPropagation();
                  handleEditCategory(child);
                }}
              />
            </div>
          </div>
        ),
      })) || [],
  }));

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
            <Text type="secondary">管理和配置工单分类体系，支持树形结构和权限控制</Text>
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
            dataSource={categories}
            rowKey="id"
            scroll={{ x: 960 }}
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
        ) : (
          <Tree
            treeData={treeData}
            defaultExpandAll
            showLine
            showIcon={false}
            className="category-tree"
          />
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
              <Form.Item name="parent_id" label="父级分类">
                <Select placeholder="请选择父级分类" allowClear>
                  {categories
                    .filter(c => !c.parent_id)
                    .map(category => (
                      <Option key={category.id} value={category.id}>
                        {category.name}
                      </Option>
                    ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="description" label="分类描述">
            <TextArea rows={3} placeholder="请输入分类描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="color" label="主题色彩" initialValue="#1890ff">
                <Input type="color" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="icon" label="分类图标">
                <Input placeholder="输入emoji或图标名称" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="sort_order" label="排序顺序" initialValue={1}>
                <InputNumber min={1} placeholder="数字越小越靠前" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="is_active"
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
