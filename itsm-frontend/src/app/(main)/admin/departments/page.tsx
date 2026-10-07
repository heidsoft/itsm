'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
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
  App,
  Popconfirm,
  Tag,
  Row,
  Col,
  Statistic,
  Empty,
} from 'antd';
import {
  Plus,
  Edit,
  Trash2,
  Users,
  RefreshCw,
  Search,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import type { Department, CreateDepartmentRequest } from '@/lib/services/department-service';
import { departmentService } from '@/lib/services/department-service';
import { UserApi } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;
const { TextArea } = Input;

export default function DepartmentManagement() {
  const { message } = App.useApp();
  const { t } = useI18n();
  const [departments, setDepartments] = useState<Department[]>([]);
  const [treeData, setTreeData] = useState<Department[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedDepartment, setSelectedDepartment] = useState<Department | null>(null);
  const [form] = Form.useForm();
  const [users, setUsers] = useState<{ label: string; value: number }[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [expandedRowKeys, setExpandedRowKeys] = useState<React.Key[]>([]);

  // 加载部门数据
  const loadDepartments = useCallback(async () => {
    setFetching(true);
    try {
      const data = await departmentService.getDepartmentTree();
      const addKeys = (depts: Department[]): Department[] =>
        depts.map(dept => ({
          ...dept,
          key: dept.id,
          children: dept.children ? addKeys(dept.children) : undefined,
        }));
      const keyedData = addKeys(data);
      setDepartments(keyedData);
      setExpandedRowKeys(keyedData.map(d => d.id));
      // 构建树形数据用于TreeSelect
      const buildTreeData = (depts: Department[]): Department[] => {
        return depts.map(dept => ({
          ...dept,
          value: dept.id,
          title: dept.name,
          children: dept.children ? buildTreeData(dept.children) : undefined,
        }));
      };
      setTreeData(buildTreeData(keyedData));
    } catch (error) {
      console.error('Failed to load departments:', error);
      message.error(t('departments.loadFailed'));
    } finally {
      setFetching(false);
    }
  }, []);

  // 加载用户列表（用于选择部门经理）
  const loadUsers = useCallback(async () => {
    try {
      const response = await UserApi.getUsers({ page: 1, pageSize: 100 });
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

  // 递归过滤部门树（保留匹配节点及其祖先）
  const filterTree = useCallback(
    (depts: Department[], keyword: string): Department[] => {
      if (!keyword) return depts;
      return depts.reduce<Department[]>((acc, dept) => {
        const children = dept.children ? filterTree(dept.children, keyword) : [];
        const selfMatch =
          dept.name.toLowerCase().includes(keyword) ||
          dept.code.toLowerCase().includes(keyword) ||
          (dept.description || '').toLowerCase().includes(keyword);
        if (selfMatch || children.length > 0) {
          acc.push({ ...dept, children: children.length > 0 ? children : dept.children?.length ? [] : undefined });
        }
        return acc;
      }, []);
    },
    []
  );

  const keyword = searchTerm.trim().toLowerCase();
  const filteredDepartments = useMemo(
    () => filterTree(departments, keyword),
    [departments, keyword, filterTree]
  );

  const collectKeys = useCallback((depts: Department[]): React.Key[] => {
    return depts.flatMap(d => [d.key ?? d.id, ...(d.children ? collectKeys(d.children) : [])]);
  }, []);

  const tableDataSource = keyword ? filteredDepartments : departments;
  const tableExpandableKeys = keyword ? collectKeys(filteredDepartments) : expandedRowKeys;

  // 统计信息
  const countDepartments = (depts: Department[]): number =>
    depts.reduce((sum, d) => sum + 1 + (d.children ? countDepartments(d.children) : 0), 0);

  const stats = {
    totalDepartments: countDepartments(tableDataSource),
  };

  // 处理保存
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      if (selectedDepartment) {
        // 更新
        await departmentService.updateDepartment(selectedDepartment.id, values);
        message.success(t('departments.updateSuccess'));
      } else {
        // 创建
        await departmentService.createDepartment(values as CreateDepartmentRequest);
        message.success(t('departments.createSuccess'));
      }

      setShowModal(false);
      form.resetFields();
      setSelectedDepartment(null);
      loadDepartments();
    } catch (error) {
      console.error('Failed to save department:', error);
      message.error(t('departments.saveFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 处理删除
  const handleDelete = async (id: number) => {
    try {
      await departmentService.deleteDepartment(id);
      message.success(t('departments.deleteSuccess'));
      loadDepartments();
    } catch (error) {
      console.error('Failed to delete department:', error);
      message.error(t('departments.deleteFailed'));
    }
  };

  // 处理编辑
  const handleEdit = (record: Department) => {
    setSelectedDepartment(record);
    form.setFieldsValue({
      name: record.name,
      code: record.code,
      description: record.description,
      managerId: record.managerId,
      parentId: record.parentId,
    });
    setShowModal(true);
  };

  // 表格列定义
  const columns: ColumnsType<Department> = [
    {
      title: t('departments.departmentName'),
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (text: string) => (
        <Space>
          <Users />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    {
      title: t('departments.departmentCode'),
      dataIndex: 'code',
      key: 'code',
      width: 120,
      render: (text: string) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: t('departments.manager'),
      dataIndex:'managerId',
      key: 'manager',
      width: 130,
      render: (managerId: number) => {
        const user = users.find(u => u.value === managerId);
        return <span>{user?.label || '-'}</span>;
      },
    },
    {
      title: t('departments.departmentDescription'),
      dataIndex: 'description',
      key: 'description',
      width: 160,
      ellipsis: true,
    },
    {
      title: t('common.action'),
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Department) => (
        <Space size="small">
          <Button
            aria-label={t('common.edit')}
            type="text"
            icon={<Edit size={16} />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title={t('departments.confirmDelete')}
            description={t('departments.deleteWarning', { name: record.name })}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Button aria-label={t('common.delete')} type="text" danger icon={<Trash2 size={16} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2">
          <Users className="mr-2" />
          {t('departments.title')}
        </Title>
        <Text type="secondary">{t('departments.description')}</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title={t('departments.totalDepartments')}
              value={stats.totalDepartments}
              prefix={<Users />}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作栏 */}
      <Card>
        <Space wrap>
          <Input
            allowClear
            placeholder={t('departments.searchPlaceholder')}
            prefix={<Search size={16} />}
            value={searchTerm}
            onChange={event => setSearchTerm(event.target.value)}
            style={{ width: 280 }}
          />
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={() => {
              setSelectedDepartment(null);
              form.resetFields();
              setShowModal(true);
            }}
          >
            {t('departments.create')}
          </Button>
          <Button
            icon={<RefreshCw size={16} />}
            onClick={() => loadDepartments()}
            loading={fetching}
          >
            {t('common.refresh')}
          </Button>
        </Space>
      </Card>

      {/* 部门列表 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={tableDataSource}
          rowKey="id"
          loading={fetching}
          pagination={false}
          scroll={{ x: 760 }}
          expandable={{
            expandedRowKeys: tableExpandableKeys,
            onExpandedRowsChange: (keys) => {
              if (!keyword) setExpandedRowKeys(keys as React.Key[]);
            },
          }}
          locale={{
            emptyText: (
              <Empty description={searchTerm ? t('departments.emptySearch') : t('departments.empty')}>
                <Button type="primary" onClick={() => setShowModal(true)}>
                  {t('departments.create')}
                </Button>
              </Empty>
            ),
          }}
          className="enterprise-table"
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={
          <span>
            <Edit className="w-4 h-4 mr-2" />
            {selectedDepartment ? t('departments.edit') : t('departments.create')}
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
        okText={t('common.save')}
        cancelText={t('common.cancel')}
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label={t('departments.departmentName')}
            name="name"
            rules={[{ required: true, message: t('departments.form.nameRequired') }]}
          >
            <Input placeholder={t('departments.form.namePlaceholder')} />
          </Form.Item>
          <Form.Item
            label={t('departments.departmentCode')}
            name="code"
            rules={[{ required: true, message: t('departments.form.codeRequired') }]}
          >
            <Input placeholder={t('departments.form.codePlaceholder')} />
          </Form.Item>
          <Form.Item
            label={t('departments.parentDepartment')}
            name="parentId"
          >
            <TreeSelect
              placeholder={t('departments.form.parentPlaceholder')}
              treeData={treeData.filter(dept => dept.id !== selectedDepartment?.id)}
              treeNodeFilterProp="title"
              allowClear
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item
            label={t('departments.manager')}
            name="managerId"
          >
            <Select
              placeholder={t('departments.form.managerPlaceholder')}
              options={users}
              allowClear
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item
            label={t('departments.departmentDescription')}
            name="description"
          >
            <TextArea rows={3} placeholder={t('departments.form.descriptionPlaceholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
