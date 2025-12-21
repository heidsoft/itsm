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
} from 'antd';
import {
  Plus,
  Edit,
  Delete,
  Eye,
  FolderOpen,
  Folder,
  GripVertical,
  Settings,
  Copy,
} from 'lucide-react';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { LoadingSkeleton } from '@/components/ui/LoadingSkeleton';

const { TextArea } = Input;
const { Title, Text } = Typography;
const { Option } = Select;

interface TicketCategory {
  id: number;
  name: string;
  description: string;
  parent_id?: number;
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
  ticket_count: number;
}

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
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));

      // 模拟分类数据
      const mockCategories: TicketCategory[] = [
        {
          id: 1,
          name: '硬件故障',
          description: '计算机硬件相关问题和故障',
          parent_id: undefined,
          level: 1,
          path: '硬件故障',
          sort_order: 1,
          is_active: true,
          is_default: false,
          color: '#ff4d4f',
          icon: '💻',
          created_at: '2024-01-01 00:00:00',
          updated_at: '2024-01-15 10:00:00',
          ticket_count: 45,
          children: [
            {
              id: 2,
              name: '显示器问题',
              description: '显示器故障和显示异常',
              parent_id: 1,
              level: 2,
              path: '硬件故障/显示器问题',
              sort_order: 1,
              is_active: true,
              is_default: false,
              color: '#fa8c16',
              icon: '🖥️',
              created_at: '2024-01-01 00:00:00',
              updated_at: '2024-01-15 10:00:00',
              ticket_count: 12,
            },
            {
              id: 3,
              name: '键盘鼠标',
              description: '键盘鼠标故障和连接问题',
              parent_id: 1,
              level: 2,
              path: '硬件故障/键盘鼠标',
              sort_order: 2,
              is_active: true,
              is_default: false,
              color: '#fa8c16',
              icon: '⌨️',
              created_at: '2024-01-01 00:00:00',
              updated_at: '2024-01-15 10:00:00',
              ticket_count: 8,
            },
          ],
        },
        {
          id: 4,
          name: '软件故障',
          description: '软件安装、配置和运行问题',
          parent_id: undefined,
          level: 1,
          path: '软件故障',
          sort_order: 2,
          is_active: true,
          is_default: false,
          color: '#1890ff',
          icon: '🔧',
          created_at: '2024-01-01 00:00:00',
          updated_at: '2024-01-15 10:00:00',
          ticket_count: 67,
          children: [
            {
              id: 5,
              name: '办公软件',
              description: 'Office、WPS等办公软件问题',
              parent_id: 4,
              level: 2,
              path: '软件故障/办公软件',
              sort_order: 1,
              is_active: true,
              is_default: false,
              color: '#52c41a',
              icon: '📄',
              created_at: '2024-01-01 00:00:00',
              updated_at: '2024-01-15 10:00:00',
              ticket_count: 23,
            },
            {
              id: 6,
              name: '系统软件',
              description: '操作系统和系统工具问题',
              parent_id: 4,
              level: 2,
              path: '软件故障/系统软件',
              sort_order: 2,
              is_active: true,
              is_default: false,
              color: '#52c41a',
              icon: '⚙️',
              created_at: '2024-01-01 00:00:00',
              updated_at: '2024-01-15 10:00:00',
              ticket_count: 18,
            },
          ],
        },
        {
          id: 7,
          name: '网络故障',
          description: '网络连接和网络配置问题',
          parent_id: undefined,
          level: 1,
          path: '网络故障',
          sort_order: 3,
          is_active: true,
          is_default: false,
          color: '#722ed1',
          icon: '🌐',
          created_at: '2024-01-01 00:00:00',
          updated_at: '2024-01-15 10:00:00',
          ticket_count: 34,
        },
        {
          id: 8,
          name: '权限问题',
          description: '用户权限和访问控制问题',
          parent_id: undefined,
          level: 1,
          path: '权限问题',
          sort_order: 4,
          is_active: true,
          is_default: false,
          color: '#eb2f96',
          icon: '🔐',
          created_at: '2024-01-01 00:00:00',
          updated_at: '2024-01-15 10:00:00',
          ticket_count: 15,
        },
      ];

      setCategories(mockCategories);
    } catch (error) {
      message.error('加载分类失败');
    } finally {
      setLoading(false);
    }
  };

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

  const handleCopyCategory = (category: TicketCategory) => {
    const newCategory = {
      ...category,
      id: Date.now(),
      name: `${category.name} - 副本`,
      is_default: false,
      created_at: new Date().toLocaleString(),
      updated_at: new Date().toLocaleString(),
    };

    setCategories(prev => [newCategory, ...prev]);
    message.success('分类复制成功');
  };

  const handleDeleteCategory = async (id: number) => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));

      setCategories(prev => prev.filter(c => c.id !== id));
      message.success('分类删除成功');
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleSaveCategory = async () => {
    try {
      const values = await form.validateFields();

      if (editingCategory) {
        // 更新分类
        const updatedCategory = {
          ...editingCategory,
          ...values,
          updated_at: new Date().toLocaleString(),
        };

        setCategories(prev => prev.map(c => (c.id === editingCategory.id ? updatedCategory : c)));
        message.success('分类更新成功');
      } else {
        // 创建新分类
        const newCategory: TicketCategory = {
          id: Date.now(),
          ...values,
          level: values.parent_id ? 2 : 1,
          path: values.parent_id ? `父分类/${values.name}` : values.name,
          sort_order: values.sort_order || 1,
          is_active: values.is_active !== false,
          is_default: false,
          created_at: new Date().toLocaleString(),
          updated_at: new Date().toLocaleString(),
          ticket_count: 0,
        };

        setCategories(prev => [newCategory, ...prev]);
        message.success('分类创建成功');
      }

      setModalVisible(false);
      form.resetFields();
    } catch (error) {
      console.error('保存分类失败:', error);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'green';
      case 'inactive':
        return 'red';
      default:
        return 'default';
    }
  };

  const columns = [
    {
      title: '分类名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: TicketCategory) => (
        <div className='flex items-center gap-2'>
          <span className='text-lg'>{record.icon}</span>
          <div>
            <div className='font-medium'>{text}</div>
            <div className='text-sm text-gray-500'>{record.path}</div>
          </div>
        </div>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      render: (text: string) => (
        <div className='max-w-xs truncate' title={text}>
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
        <div className='flex items-center gap-1'>
          <GripVertical size={14} className='text-gray-400' />
          <span>{order}</span>
        </div>
      ),
    },
    {
      title: '工单数量',
      dataIndex: 'ticket_count',
      key: 'ticket_count',
      render: (count: number) => <Tag color='orange'>{count}</Tag>,
    },
    {
      title: '状态',
      key: 'status',
      render: (record: TicketCategory) => (
        <div className='flex items-center gap-2'>
          <Switch
            checked={record.is_active}
            size='small'
            onChange={checked => {
              setCategories(prev =>
                prev.map(c => (c.id === record.id ? { ...c, is_active: checked } : c))
              );
            }}
          />
          {record.is_default && <Tag color='green'>默认</Tag>}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (record: TicketCategory) => (
        <Space>
          <Tooltip title='查看详情'>
            <Button size='small' icon={<Eye size={14} />} />
          </Tooltip>
          <Tooltip title='编辑'>
            <Button
              size='small'
              icon={<Edit size={14} />}
              onClick={() => handleEditCategory(record)}
            />
          </Tooltip>
          <Tooltip title='复制'>
            <Button
              size='small'
              icon={<Copy size={14} />}
              onClick={() => handleCopyCategory(record)}
            />
          </Tooltip>
          <Popconfirm
            title='确定要删除这个分类吗？'
            onConfirm={() => handleDeleteCategory(record.id)}
            okText='确定'
            cancelText='取消'
          >
            <Button size='small' danger icon={<Delete size={14} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const treeData = categories.map(category => ({
    key: category.id,
    title: (
      <div className='flex items-center justify-between w-full'>
        <div className='flex items-center gap-2'>
          <span className='text-lg'>{category.icon}</span>
          <span className='font-medium'>{category.name}</span>
          <Tag color='orange'>{category.ticket_count}</Tag>
        </div>
        <div className='flex items-center gap-1'>
          <Switch
            checked={category.is_active}
            size='small'
            onChange={checked => {
              setCategories(prev =>
                prev.map(c => (c.id === category.id ? { ...c, is_active: checked } : c))
              );
            }}
          />
          <Button
            size='small'
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
          <div className='flex items-center justify-between w-full'>
            <div className='flex items-center gap-2'>
              <span className='text-lg'>{child.icon}</span>
              <span>{child.name}</span>
              <Tag color='orange'>{child.ticket_count}</Tag>
            </div>
            <div className='flex items-center gap-1'>
              <Switch
                checked={child.is_active}
                size='small'
                onChange={checked => {
                  setCategories(prev =>
                    prev.map(c => (c.id === child.id ? { ...c, is_active: checked } : c))
                  );
                }}
              />
              <Button
                size='small'
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
    return <LoadingSkeleton type='table' rows={8} columns={7} />;
  }

  if (categories.length === 0) {
    return (
      <LoadingEmptyError
        state='empty'
        empty={{
          title: '暂无工单分类',
          description: '创建第一个工单分类来组织工单管理',
          actionText: '创建分类',
          onAction: handleCreateCategory,
        }}
      />
    );
  }

  return (
    <div className='space-y-6'>
      {/* 头部操作区 */}
      <Card>
        <div className='flex justify-between items-center'>
          <div>
            <Title level={4} className='mb-1'>
              工单分类管理
            </Title>
            <Text type='secondary'>管理和配置工单分类体系，支持树形结构和权限控制</Text>
          </div>
          <Space>
            <Button.Group>
              <Button
                type={viewMode === 'table' ? 'primary' : 'default'}
                icon={<Table />}
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
            <Button type='primary' icon={<Plus size={16} />} onClick={handleCreateCategory}>
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
            rowKey='id'
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
            className='category-tree'
          />
        )}
      </Card>

      {/* 创建/编辑分类模态框 */}
      <Modal
        title={editingCategory ? '编辑分类' : '创建分类'}
        open={modalVisible}
        onOk={handleSaveCategory}
        onCancel={() => setModalVisible(false)}
        okText='保存'
        cancelText='取消'
        width={600}
      >
        <Form form={form} layout='vertical'>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='name'
                label='分类名称'
                rules={[{ required: true, message: '请输入分类名称' }]}
              >
                <Input placeholder='请输入分类名称' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='parent_id' label='父级分类'>
                <Select placeholder='请选择父级分类' allowClear>
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

          <Form.Item name='description' label='分类描述'>
            <TextArea rows={3} placeholder='请输入分类描述' />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name='color' label='主题色彩' initialValue='#1890ff'>
                <Input type='color' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='icon' label='分类图标'>
                <Input placeholder='输入emoji或图标名称' />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name='sort_order' label='排序顺序' initialValue={1}>
                <InputNumber min={1} placeholder='数字越小越靠前' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='is_active'
                label='启用状态'
                valuePropName='checked'
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
