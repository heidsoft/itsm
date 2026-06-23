'use client';

import {
  RefreshCw,
  Database,
  Plus,
  Search,
  Edit,
  Trash2,
  CheckCircle,
  AlertCircle,
  Layers,
} from 'lucide-react';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Modal,
  Form,
  Alert,
  Select,
  Space,
  Row,
  Col,
  Statistic,
  Typography,
  Popconfirm,
  Tooltip,
  App,
  Tag,
  Switch,
} from 'antd';
import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CIType } from '@/types/biz/cmdb';

const { Title, Text } = Typography;
const { Option } = Select;

type AttributeTemplateField = {
  key?: string;
  label?: string;
  type?: 'select';
  options?: string;
  required?: boolean;
  placeholder?: string;
};

const DEFAULT_ATTRIBUTE_SCHEMA = JSON.stringify(
  {
    fields: [
      {
        key: 'environment',
        label: '环境',
        type: 'select',
        options: ['production', 'staging', 'development'],
        required: true,
      },
      {
        key: 'owner',
        label: '负责人',
        type: 'select',
        options: ['ops', 'platform', 'security'],
      },
    ],
  },
  null,
  2
);

const validateAttributeSchema = (value?: string) => {
  if (!value || !value.trim()) {
    return null;
  }

  let parsed: any;
  try {
    parsed = JSON.parse(value);
  } catch {
    return '请输入合法的 JSON';
  }

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return '属性模式必须是对象 JSON';
  }

  if (parsed.fields === undefined) {
    return null;
  }

  if (!Array.isArray(parsed.fields)) {
    return 'attribute_schema.fields 必须是数组';
  }

  for (let index = 0; index < parsed.fields.length; index += 1) {
    const field = parsed.fields[index];
    if (!field || typeof field !== 'object' || Array.isArray(field)) {
      return `attribute_schema.fields[${index}] 必须是对象`;
    }
    const fieldType = field.type;
    const fieldKey = field.key || field.name || `字段${index + 1}`;
    if (fieldType !== 'select') {
      return `attribute_schema.fields[${index}]（${fieldKey}）仅支持 type=select`;
    }
    if (!Array.isArray(field.options) || field.options.length === 0) {
      return `attribute_schema.fields[${index}]（${fieldKey}）必须提供非空 options`;
    }
  }

  return null;
};

const normalizeAttributeTemplateFields = (fields?: AttributeTemplateField[]) =>
  (fields || [])
    .map(field => ({
      key: field.key?.trim(),
      label: field.label?.trim(),
      type: 'select' as const,
      required: Boolean(field.required),
      placeholder: field.placeholder?.trim(),
      options: (field.options || '')
        .split(/[\n,，]/)
        .map(option => option.trim())
        .filter(Boolean),
    }))
    .filter(field => field.key || field.label || field.options.length > 0);

const buildAttributeSchemaFromFields = (fields?: AttributeTemplateField[]) => {
  const normalizedFields = normalizeAttributeTemplateFields(fields);
  if (normalizedFields.length === 0) {
    return '';
  }

  return JSON.stringify(
    {
      fields: normalizedFields.map(field => ({
        key: field.key,
        label: field.label || field.key,
        type: field.type,
        options: field.options,
        required: field.required,
        ...(field.placeholder ? { placeholder: field.placeholder } : {}),
      })),
    },
    null,
    2
  );
};

const parseAttributeSchemaToFields = (schemaText?: string): AttributeTemplateField[] => {
  if (!schemaText || !schemaText.trim()) {
    return [];
  }

  try {
    const parsed = JSON.parse(schemaText);
    const fields = Array.isArray(parsed?.fields) ? parsed.fields : [];
    return fields
      .filter((field: unknown) => field && typeof field === 'object' && !Array.isArray(field))
      .map((field: Record<string, unknown>) => ({
        key: typeof field.key === 'string' ? field.key : typeof field.name === 'string' ? field.name : '',
        label: typeof field.label === 'string' ? field.label : '',
        type: 'select' as const,
        options: Array.isArray(field.options)
          ? field.options.filter((option): option is string => typeof option === 'string').join('\n')
          : '',
        required: Boolean(field.required),
        placeholder: typeof field.placeholder === 'string' ? field.placeholder : '',
      }));
  } catch {
    return [];
  }
};

const getAttributeSchemaFieldCount = (value?: string) => {
  if (!value || !value.trim()) {
    return 0;
  }

  try {
    const parsed = JSON.parse(value);
    if (Array.isArray(parsed?.fields)) {
      return parsed.fields.length;
    }
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return Object.keys(parsed).length;
    }
  } catch {
    return 0;
  }

  return 0;
};

const CMDBTypesManagement = () => {
  const { message } = App.useApp();
  const formId = 'cmdb-type-form';
  const [ciTypes, setCiTypes] = useState<CIType[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingType, setEditingType] = useState<CIType | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [form] = Form.useForm();
  const schemaFields = Form.useWatch('schema_fields', form) as AttributeTemplateField[] | undefined;
  const attributeSchemaPreview = React.useMemo(
    () => buildAttributeSchemaFromFields(schemaFields),
    [schemaFields]
  );

  // 预设图标列表
  const iconOptions = [
    { value: 'server', label: '服务器' },
    { value: 'database', label: '数据库' },
    { value: 'cloud', label: '云服务' },
    { value: 'network', label: '网络' },
    { value: 'storage', label: '存储' },
    { value: 'desktop', label: '终端' },
    { value: 'code', label: '应用' },
    { value: 'monitor', label: '监控' },
    { value: 'security', label: '安全' },
    { value: 'mail', label: '邮件' },
    { value: 'phone', label: '电话' },
    { value: 'printer', label: '打印机' },
  ];

  // 预设颜色列表
  const colorOptions = [
    { value: '#1890ff', label: '蓝色' },
    { value: '#52c41a', label: '绿色' },
    { value: '#faad14', label: '橙色' },
    { value: '#f5222d', label: '红色' },
    { value: '#722ed1', label: '紫色' },
    { value: '#13c2c2', label: '青色' },
    { value: '#eb2f96', label: '粉色' },
    { value: '#fa541c', label: '橙红' },
  ];

  // 统计数据
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

  // 获取CI类型列表
  const fetchCITypes = async () => {
    try {
      setLoading(true);
      const types = await CMDBApi.getCMDBTypes();
      setCiTypes(types);

      // 计算统计数据
      const activeCount = types.filter((t: CIType) => t.is_active).length;

      setStats({
        total: types.length,
        active: activeCount,
        inactive: types.length - activeCount,
      });
    } catch (error) {
      message.error('获取CI类型失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCITypes();
  }, []);

  // 处理表单提交
  const handleSubmit = async (values: Record<string, any>) => {
    try {
      const normalizedFields = normalizeAttributeTemplateFields(values.schema_fields);
      const duplicateKeys = normalizedFields
        .map(field => field.key)
        .filter((key, index, keys): key is string => Boolean(key) && keys.indexOf(key) !== index);
      if (duplicateKeys.length > 0) {
        message.error(`字段标识不能重复：${Array.from(new Set(duplicateKeys)).join('、')}`);
        return;
      }

      const schemaText = buildAttributeSchemaFromFields(values.schema_fields);
      const schemaError = validateAttributeSchema(schemaText);
      if (schemaError) {
        message.error(schemaError);
        return;
      }

      const payload = {
        name: values.name,
        description: values.description || '',
        icon: values.icon || '',
        color: values.color || '#1890ff',
        attribute_schema: schemaText,
        is_active: values.is_active ?? true,
      };

      if (editingType) {
        await CMDBApi.updateCITypes(editingType.id, payload);
        message.success('CI类型更新成功');
      } else {
        await CMDBApi.createCITypes(payload);
        message.success('CI类型创建成功');
      }
      setShowModal(false);
      setEditingType(null);
      form.resetFields();
      fetchCITypes();
    } catch (error: any) {
      message.error(error?.message || (editingType ? '更新失败' : '创建失败'));
    }
  };

  // 编辑CI类型
  const handleEdit = (type: CIType) => {
    setEditingType(type);
    form.setFieldsValue({
      name: type.name,
      description: type.description,
      icon: type.icon,
      color: type.color,
      attribute_schema: type.attribute_schema
        ? (() => {
            try {
              return JSON.stringify(JSON.parse(type.attribute_schema), null, 2);
            } catch {
              return type.attribute_schema;
            }
          })()
        : '',
      schema_fields: parseAttributeSchemaToFields(type.attribute_schema),
      is_active: type.is_active,
    });
    setShowModal(true);
  };

  // 删除CI类型
  const handleDelete = async (id: number) => {
    try {
      await CMDBApi.deleteCITypes(id);
      message.success('删除成功');
      fetchCITypes();
    } catch (error: any) {
      message.error(error?.message || '删除失败');
    }
  };

  const handleUseDefaultAttributeSchema = () => {
    form.setFieldsValue({
      attribute_schema: DEFAULT_ATTRIBUTE_SCHEMA,
      schema_fields: parseAttributeSchemaToFields(DEFAULT_ATTRIBUTE_SCHEMA),
    });
  };

  const handleFormatAttributeSchema = () => {
    const schemaText = buildAttributeSchemaFromFields(form.getFieldValue('schema_fields'));
    if (!schemaText) {
      message.info('请先添加至少一个字段');
      return;
    }

    try {
      form.setFieldValue('attribute_schema', JSON.stringify(JSON.parse(schemaText), null, 2));
      message.success('属性模板预览已更新');
    } catch {
      message.error('字段配置不完整，暂时无法生成 JSON');
    }
  };

  // 过滤CI类型
  const filteredTypes = ciTypes.filter(type => {
    const matchesSearch =
      !searchTerm ||
      type.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      type.description?.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesStatus =
      !statusFilter || (statusFilter === 'active' ? type.is_active : !type.is_active);

    return matchesSearch && matchesStatus;
  });

  // 表格列定义
  const columns = [
    {
      title: '类型名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: CIType) => (
        <div className="flex items-center">
          <div
            className="w-8 h-8 rounded flex items-center justify-center mr-3 text-white text-sm font-medium"
            style={{ backgroundColor: record.color || '#1890ff' }}
          >
            {record.icon ? (
              <span className="text-xs">{record.icon.charAt(0).toUpperCase()}</span>
            ) : (
              <Layers className="w-4 h-4" />
            )}
          </div>
          <div>
            <div className="font-medium text-gray-900">{name}</div>
            {record.description && (
              <div className="text-sm text-gray-500 mt-1">{record.description}</div>
            )}
          </div>
        </div>
      ),
    },
    {
      title: '图标',
      dataIndex: 'icon',
      key: 'icon',
      width: 100,
      render: (icon: string) => <Tag color="default">{icon || '-'}</Tag>,
    },
    {
      title: '颜色',
      dataIndex: 'color',
      key: 'color',
      width: 100,
      render: (color: string) => (
        <div className="flex items-center">
          <div className="w-4 h-4 rounded mr-2" style={{ backgroundColor: color || '#1890ff' }} />
          <span className="text-sm">{color || '-'}</span>
        </div>
      ),
    },
    {
      title: '属性模板',
      dataIndex: 'attribute_schema',
      key: 'attribute_schema',
      width: 120,
      render: (schema: string) => {
        const fieldCount = getAttributeSchemaFieldCount(schema);
        return fieldCount > 0 ? <Tag color="blue">{fieldCount} 个字段</Tag> : <Tag>未配置</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 100,
      render: (isActive: boolean) => (
        <Tag
          color={isActive ? 'green' : 'default'}
          icon={
            isActive ? <CheckCircle className="w-3 h-3" /> : <AlertCircle className="w-3 h-3" />
          }
        >
          {isActive ? '激活' : '停用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date: string) => (
        <span className="text-sm text-gray-600">
          {date ? new Date(date).toLocaleDateString('zh-CN') : '-'}
        </span>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: CIType) => (
        <Space>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => handleEdit(record)}
              size="small"
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个CI类型吗？"
            description="如果存在使用该类型的CI实例，将无法删除"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button type="text" icon={<Trash2 className="w-4 h-4" />} danger size="small" />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      {/* 页面标题 */}
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <Database className="inline-block w-6 h-6 mr-2" />
          CI类型管理
        </Title>
        <Text type="secondary">管理CMDB配置项类型，自定义IT基础设施分类</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="总类型数"
              value={stats.total}
              prefix={<Database className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="激活类型"
              value={stats.active}
              prefix={<CheckCircle className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="停用类型"
              value={stats.inactive}
              prefix={<AlertCircle className="w-5 h-5" />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className="enterprise-card mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <Input
              placeholder="搜索CI类型..."
              prefix={<Search className="w-4 h-4" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="状态筛选"
              value={statusFilter}
              onChange={setStatusFilter}
              allowClear
              style={{ width: '100%' }}
            >
              <Option value="active">激活</Option>
              <Option value="inactive">停用</Option>
            </Select>
          </Col>
          <Col xs={24} sm={24} md={12}>
            <Space>
              <Button
                type="primary"
                icon={<Plus className="w-4 h-4" />}
                onClick={() => {
                  setEditingType(null);
                  form.resetFields();
                  form.setFieldsValue({ is_active: true, color: '#1890ff', schema_fields: [] });
                  setShowModal(true);
                }}
              >
                新建CI类型
              </Button>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={fetchCITypes}>
                刷新
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* CI类型表格 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={filteredTypes}
          rowKey="id"
          loading={loading}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          scroll={{ x: 800 }}
        />
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={
          <span>
            <Database className="w-4 h-4 mr-2" />
            {editingType ? '编辑CI类型' : '新建CI类型'}
          </span>
        }
        open={showModal}
        onCancel={() => {
          setShowModal(false);
          setEditingType(null);
          form.resetFields();
        }}
        footer={
          <Space>
            <Button
              onClick={() => {
                setShowModal(false);
                setEditingType(null);
                form.resetFields();
              }}
            >
              取消
            </Button>
            <Button type="primary" htmlType="submit" form={formId}>
              {editingType ? '更新' : '创建'}
            </Button>
          </Space>
        }
        width={760}
      >
        <Form
          id={formId}
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{ is_active: true, color: '#1890ff' }}
        >
          <Form.Item
            name="name"
            label="类型名称"
            rules={[
              { required: true, message: '请输入类型名称' },
              { max: 255, message: '类型名称不能超过255个字符' },
            ]}
          >
            <Input placeholder="请输入类型名称，如：服务器、数据库、网络设备" />
          </Form.Item>

          <Form.Item
            name="description"
            label="类型描述"
            rules={[{ max: 1000, message: '描述不能超过1000个字符' }]}
          >
            <Input.TextArea rows={3} placeholder="请输入类型描述" showCount maxLength={1000} />
          </Form.Item>

          <Form.Item
            label="属性模板"
            extra="这些字段会出现在配置项录入表单中。当前先支持枚举选择，适合环境、负责人团队、实例规格等标准字段。"
          >
            <Alert
              className="mb-3"
              type="info"
              showIcon
              message="不用手写 JSON，按字段逐项配置即可"
              description="字段标识用于保存数据，建议使用英文小写和下划线；选项可用换行或逗号分隔。"
            />
            <Form.List name="schema_fields">
              {(fields, { add, remove }) => (
                <div className="space-y-3">
                  {fields.map(({ key, name, ...restField }, index) => (
                    <Card
                      key={key}
                      size="small"
                      title={`字段 ${index + 1}`}
                      extra={
                        <Button size="small" danger type="text" onClick={() => remove(name)}>
                          删除
                        </Button>
                      }
                    >
                      <Row gutter={12}>
                        <Col xs={24} md={12}>
                          <Form.Item
                            {...restField}
                            name={[name, 'key']}
                            label="字段标识"
                            rules={[
                              { required: true, message: '请输入字段标识' },
                              {
                                pattern: /^[a-z][a-z0-9_]*$/,
                                message: '仅支持英文小写、数字和下划线，并以字母开头',
                              },
                            ]}
                          >
                            <Input placeholder="例如 environment" allowClear />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={12}>
                          <Form.Item
                            {...restField}
                            name={[name, 'label']}
                            label="字段名称"
                            rules={[{ required: true, message: '请输入字段名称' }]}
                          >
                            <Input placeholder="例如 环境" allowClear />
                          </Form.Item>
                        </Col>
                      </Row>
                      <Row gutter={12}>
                        <Col xs={24} md={12}>
                          <Form.Item
                            {...restField}
                            name={[name, 'options']}
                            label="可选项"
                            rules={[{ required: true, message: '请输入至少一个可选项' }]}
                          >
                            <Input.TextArea
                              rows={3}
                              placeholder={'生产\n预发布\n开发'}
                              allowClear
                            />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={12}>
                          <Form.Item {...restField} name={[name, 'placeholder']} label="输入提示">
                            <Input placeholder="例如 请选择环境" allowClear />
                          </Form.Item>
                          <Form.Item
                            {...restField}
                            name={[name, 'required']}
                            label="是否必填"
                            valuePropName="checked"
                          >
                            <Switch checkedChildren="必填" unCheckedChildren="选填" />
                          </Form.Item>
                        </Col>
                      </Row>
                    </Card>
                  ))}
                  <Button
                    type="dashed"
                    block
                    icon={<Plus className="w-4 h-4" />}
                    onClick={() =>
                      add({
                        key: '',
                        label: '',
                        type: 'select',
                        options: '',
                        required: false,
                      })
                    }
                  >
                    添加字段
                  </Button>
                </div>
              )}
            </Form.List>
          </Form.Item>

          <div className="-mt-4 mb-4 flex flex-wrap gap-2">
            <Button size="small" onClick={handleUseDefaultAttributeSchema}>
              使用示例模板
            </Button>
            <Button size="small" onClick={handleFormatAttributeSchema}>
              生成 JSON 预览
            </Button>
            <Button
              size="small"
              onClick={() =>
                form.setFieldsValue({
                  schema_fields: [],
                  attribute_schema: '',
                })
              }
            >
              清空模板
            </Button>
          </div>

          <Form.Item label="JSON 预览" extra="提交时会自动保存该 JSON。高级用户可用于核对最终结构。">
            <Input.TextArea
              rows={5}
              value={attributeSchemaPreview || ''}
              placeholder={DEFAULT_ATTRIBUTE_SCHEMA}
              readOnly
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="icon" label="图标标识">
                <Select placeholder="选择图标标识" allowClear>
                  {iconOptions.map(option => (
                    <Option key={option.value} value={option.value}>
                      {option.label}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="color" label="颜色">
                <Select placeholder="选择颜色">
                  {colorOptions.map(option => (
                    <Option key={option.value} value={option.value}>
                      <div className="flex items-center">
                        <div
                          className="w-3 h-3 rounded mr-2"
                          style={{ backgroundColor: option.value }}
                        />
                        {option.label}
                      </div>
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="is_active" label="状态" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="激活" unCheckedChildren="停用" />
          </Form.Item>

        </Form>
      </Modal>
    </div>
  );
};

export default CMDBTypesManagement;
