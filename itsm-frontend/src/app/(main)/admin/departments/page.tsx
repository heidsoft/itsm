'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Typography,
  Modal,
  Form,
  Input,
  Select,
  TreeSelect,
  message,
  Popconfirm,
  Tag,
  Row,
  Col,
  Statistic,
} from 'antd';
import {
  Plus,
  Edit,
  Trash2,
  Users,
  RefreshCw,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { departmentService, Department, CreateDepartmentRequest } from '@/lib/services/department-service';
import { UserApi } from '@/lib/api/user-api';

const { Title, Text } = Typography;
const { TextArea } = Input;

export default function DepartmentManagement() {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [treeData, setTreeData] = useState<Department[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedDepartment, setSelectedDepartment] = useState<Department | null>(null);
  const [form] = Form.useForm();
  const [users, setUsers] = useState<{ label: string; value: number }[]>([]);

  // 加载部门数据
  const loadDepartments = useCallback(async () => {
    setFetching(true);
    try {
      const data = await departmentService.getDepartmentTree();
      setDepartments(data);
      // 构建树形数据用于TreeSelect
      const buildTreeData = (depts: Department[]): Department[] => {
        return depts.map(dept => ({
          ...dept,
          value: dept.id,
          title: dept.name,
          children: dept.children ? buildTreeData(dept.children) : undefined,
        }));
      };
      setTreeData(buildTreeData(data));
    } catch (error) {
      console.error('Failed to load departments:', error);
      message.error('加载部门数据失败');
    } finally {
      setFetching(false);
    }
  }, []);

  // 加载用户列表（用于选择部门经理）
  const loadUsers = useCallback(async () => {
    try {
      const response = await UserApi.getUsers({ page: 1, page_size: 100 });
      setUsers(
        response.users.map(user => ({
          label: user.name || user.username,
          value: user.id,
        }))
      );
    } catch (error) {
      console.error('Failed to load users:', error);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    loadDepartments();
    loadUsers();
  }, [loadDepartments, loadUsers]);

  // 扁平化部门树用于表格展示
  const flattenDepartments = (depts: Department[], level = 0): Department[] => {
    const result: Department[] = [];
    depts.forEach(dept => {
      result.push({ ...dept, key: dept.id });
      if (dept.children && dept.children.length > 0) {
        result.push(...flattenDepartments(dept.children, level + 1));
      }
    });
    return result;
  };

  const flatDepartments = flattenDepartments(departments);

  // 统计信息
  const stats = {
    totalDepartments: flatDepartments.length,
    activeDepartments: flatDepartments.filter(d => d.name).length,
  };

  // 处理保存
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      if (selectedDepartment) {
        // 更新
        await departmentService.updateDepartment(selectedDepartment.id, values);
        message.success('部门更新成功');
      } else {
        // 创建
        await departmentService.createDepartment(values as CreateDepartmentRequest);
        message.success('部门创建成功');
      }

      setShowModal(false);
      form.resetFields();
      setSelectedDepartment(null);
      loadDepartments();
    } catch (error) {
      console.error('Failed to save department:', error);
      message.error('保存部门失败');
    } finally {
      setLoading(false);
    }
  };

  // 处理删除
  const handleDelete = async (id: number) => {
    try {
      await departmentService.deleteDepartment(id);
      message.success('部门删除成功');
      loadDepartments();
    } catch (error) {
      console.error('Failed to delete department:', error);
      message.error('删除部门失败');
    }
  };

  // 处理编辑
  const handleEdit = (record: Department) => {
    setSelectedDepartment(record);
    form.setFieldsValue({
      name: record.name,
      code: record.code,
      description: record.description,
      manager_id: record.manager_id,
      parent_id: record.parent_id,
    });
    setShowModal(true);
  };

  // 表格列定义
  const columns: ColumnsType<Department> = [
    {
      title: '部门名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Department) => (
        <Space>
          <Users />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    {
      title: '部门编码',
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: '部门经理',
      dataIndex: 'manager_id',
      key: 'manager',
      render: (managerId: number) => {
        const user = users.find(u => u.value === managerId);
        return <span>{user?.label || '-'}</span>;
      },
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Department) => (
        <Space size="small">
          <Button
            type="text"
            icon={<Edit size={16} />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title="确认删除"
            description={`确定要删除部门"${record.name}"吗？`}
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button type="text" danger icon={<Trash2 size={16} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <Users className="mr-2" />
          部门管理
        </Title>
        <Text type="secondary">管理系统部门结构和组织架构</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="部门总数"
              value={stats.totalDepartments}
              prefix={<Users />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="活跃部门"
              value={stats.activeDepartments}
              prefix={<Users />}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作栏 */}
      <Card className="mb-6">
        <Space>
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={() => {
              setSelectedDepartment(null);
              form.resetFields();
              setShowModal(true);
            }}
          >
            新建部门
          </Button>
          <Button
            icon={<RefreshCw size={16} />}
            onClick={() => loadDepartments()}
            loading={fetching}
          >
            刷新
          </Button>
        </Space>
      </Card>

      {/* 部门列表 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={flatDepartments}
          rowKey="id"
          loading={fetching}
          pagination={false}
          className="enterprise-table"
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={
          <span>
            <Edit className="w-4 h-4 mr-2" />
            {selectedDepartment ? '编辑部门' : '新建部门'}
          </span>
        }
        open={showModal}
        onOk={handleSave}
        onCancel={() => {
          setShowModal(false);
          setSelectedDepartment(null);
          form.resetFields();
        }}
        width={600}
        confirmLoading={loading}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label="部门名称"
            name="name"
            rules={[{ required: true, message: '请输入部门名称' }]}
          >
            <Input placeholder="请输入部门名称" />
          </Form.Item>
          <Form.Item
            label="部门编码"
            name="code"
            rules={[{ required: true, message: '请输入部门编码' }]}
          >
            <Input placeholder="请输入部门编码（如：DEPT001）" />
          </Form.Item>
          <Form.Item
            label="上级部门"
            name="parent_id"
          >
            <TreeSelect
              placeholder="选择上级部门（可选）"
              treeData={treeData}
              treeNodeFilterProp="title"
              allowClear
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item
            label="部门经理"
            name="manager_id"
          >
            <Select
              placeholder="选择部门经理"
              options={users}
              allowClear
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item
            label="描述"
            name="description"
          >
            <TextArea rows={3} placeholder="请输入部门描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
