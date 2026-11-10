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
import { formatDateTime } from '@/lib/formatters';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
import type { CIType } from '@/types/biz/cmdb';
import {
  ATTRIBUTE_FIELD_TYPE_OPTIONS,
  DEFAULT_ATTRIBUTE_SCHEMA,
  buildAttributeSchemaFromFields,
  getAttributeSchemaFieldCount,
  isAttributeSchemaSafelyEditable,
  normalizeAttributeTemplateFields,
  parseAttributeSchemaToFields,
  validateAttributeSchema,
} from './attributeSchemaUtils';
import type { AttributeTemplateField } from './attributeSchemaUtils';

const { Title, Text } = Typography;

const CMDBTypesManagement = () => {
  const { message } = App.useApp();
  const formId = 'cmdb-type-form';
  const [ciTypes, setCiTypes] = useState<CIType[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingType, setEditingType] = useState<CIType | null>(null);
  const [hasUnsupportedSchema, setHasUnsupportedSchema] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [form] = Form.useForm();
  const schemaFields = Form.useWatch('schemaFields', form) as AttributeTemplateField[] | undefined;
  const attributeSchemaPreview = React.useMemo(
    () => buildAttributeSchemaFromFields(schemaFields),
    [schemaFields]
  );

  // 预设图标列表
  const iconOptions = [
    { value: 'server', label: 'Servers' },
    { value: 'database', label: 'Database' },
    { value: 'cloud', label: 'Cloud services' },
    { value: 'network', label: 'Network' },
    { value: 'storage', label: 'Storage' },
    { value: 'desktop', label: 'Terminal' },
    { value: 'code', label: 'Application' },
    { value: 'monitor', label: 'Monitoring' },
    { value: 'security', label: 'Safe' },
    { value: 'mail', label: 'Email' },
    { value: 'phone', label: 'Phone' },
    { value: 'printer', label: 'Printer' },
  ];

  // 预设颜色列表
  const colorOptions = [
    { value: '#1890ff', label: 'blue' },
    { value: '#52c41a', label: 'green' },
    { value: '#faad14', label: 'Orange' },
    { value: '#f5222d', label: 'red' },
    { value: '#722ed1', label: 'Purple' },
    { value: '#13c2c2', label: 'cyan' },
    { value: '#eb2f96', label: 'pink' },
    { value: '#fa541c', label: 'orange red' },
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
      const activeCount = types.filter((t: CIType) => t.isActive).length;

      setStats({
        total: types.length,
        active: activeCount,
        inactive: types.length - activeCount,
      });
    } catch (error) {
      message.error('Failed to obtain CI type');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCITypes();
  }, []);

  // 处理表单提交
  const handleSubmit = async (values: Record<string, unknown>) => {
    try {
      if (editingType && hasUnsupportedSchema) {
        message.error('This type contains historical attribute formats that are not supported by the current visual editor. Please migrate the template first before editing.');
        return;
      }
      const normalizedFields = normalizeAttributeTemplateFields(values.schemaFields as AttributeTemplateField[]);
      const duplicateKeys = normalizedFields
        .map(field => field.key)
        .filter((key, index, keys): key is string => Boolean(key) && keys.indexOf(key) !== index);
      if (duplicateKeys.length > 0) {
        message.error(`Field ID cannot be repeated:${Array.from(new Set(duplicateKeys)).join('、')}`);
        return;
      }

      const schemaText = buildAttributeSchemaFromFields(values.schemaFields as AttributeTemplateField[]);
      const schemaError = validateAttributeSchema(schemaText);
      if (schemaError) {
        message.error(schemaError);
        return;
      }

      const payload = {
        name: values.name as string,
        description: (values.description as string) || '',
        icon: (values.icon as string) || '',
        color: (values.color as string) || '#1890ff',
        attributeSchema: schemaText,
        parentTypeId: values.parentTypeId as number | undefined,
        ...(editingType?.parentTypeId && !values.parentTypeId ? { clearParent: true } : {}),
        isActive: (values.isActive as boolean) ?? true,
      };

      if (editingType) {
        await CMDBApi.updateCITypes(editingType.id, payload);
        message.success('CI type updated successfully');
      } else {
        await CMDBApi.createCITypes(payload);
        message.success('CI type created successfully');
      }
      setShowModal(false);
      setEditingType(null);
      setHasUnsupportedSchema(false);
      form.resetFields();
      fetchCITypes();
    } catch (error: unknown) {
      message.error((error instanceof Error ? error.message : undefined) || (editingType ? 'Update failed' : 'Creation failed'));
    }
  };

  // 编辑CI类型
  const handleEdit = (type: CIType) => {
    setEditingType(type);
    setHasUnsupportedSchema(!isAttributeSchemaSafelyEditable(type.attributeSchema));
    form.setFieldsValue({
      name: type.name,
      description: type.description,
      icon: type.icon,
      color: type.color,
      parentTypeId: type.parentTypeId,
      schemaFields: parseAttributeSchemaToFields(type.attributeSchema),
      isActive: type.isActive,
    });
    setShowModal(true);
  };

  // 删除CI类型
  const handleDelete = async (id: number) => {
    try {
      await CMDBApi.deleteCITypes(id);
      message.success('Deleted');
      fetchCITypes();
    } catch (error: unknown) {
      message.error((error instanceof Error ? error.message : undefined) || 'Delete failed');
    }
  };

  const handleUseDefaultAttributeSchema = () => {
    form.setFieldsValue({
      schemaFields: parseAttributeSchemaToFields(DEFAULT_ATTRIBUTE_SCHEMA),
    });
  };

  const handleFormatAttributeSchema = () => {
    const schemaText = buildAttributeSchemaFromFields(form.getFieldValue('schemaFields'));
    if (!schemaText) {
      message.info('Please add at least one field first');
      return;
    }

    const schemaError = validateAttributeSchema(schemaText);
    if (schemaError) {
      message.error(schemaError);
      return;
    }
    message.success('The attribute template verification passed, which can be checked in the JSON preview below');
  };

  // 过滤CI类型
  const filteredTypes = ciTypes.filter(type => {
    const matchesSearch =
      !searchTerm ||
      type.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      type.description?.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesStatus =
      !statusFilter || (statusFilter === 'active' ? type.isActive : !type.isActive);

    return matchesSearch && matchesStatus;
  });

  // 表格列定义
  const columns = [
    {
      title: 'Type Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: CIType) => (
        <div className='flex items-center'>
          <div
            className='w-8 h-8 rounded flex items-center justify-center mr-3 text-white text-sm font-medium'
            style={{ backgroundColor: record.color || '#1890ff' }}
          >
            {record.icon ? (
              <span className='text-xs'>{record.icon.charAt(0).toUpperCase()}</span>
            ) : (
              <Layers className='w-4 h-4' />
            )}
          </div>
          <div>
            <div className='font-medium text-gray-900'>{name}</div>
            {record.description && (
              <div className='text-sm text-gray-500 mt-1'>{record.description}</div>
            )}
          </div>
        </div>
      ),
    },
    {
      title: 'Icon',
      dataIndex: 'icon',
      key: 'icon',
      width: 100,
      render: (icon: string) => <Tag color='default'>{icon || '-'}</Tag>,
    },
    {
      title: 'Color',
      dataIndex: 'color',
      key: 'color',
      width: 100,
      render: (color: string) => (
        <div className='flex items-center'>
          <div className='w-4 h-4 rounded mr-2' style={{ backgroundColor: color || '#1890ff' }} />
          <span className='text-sm'>{color || '-'}</span>
        </div>
      ),
    },
    {
      title: 'Property template',
      dataIndex: 'attributeSchema',
      key: 'attributeSchema',
      width: 120,
      render: (schema: string) => {
        const fieldCount = getAttributeSchemaFieldCount(schema);
        return fieldCount > 0 ? <Tag color='blue'>{fieldCount}  fields</Tag> : <Tag>Not configured</Tag>;
      },
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 100,
      render: (isActive: boolean) => (
        <Tag
          color={isActive ? 'green' : 'default'}
          icon={
            isActive ? <CheckCircle className='w-3 h-3' /> : <AlertCircle className='w-3 h-3' />
          }
        >
          {isActive ? 'Active' : 'Disable'}
        </Tag>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date: string) => (
        <span className='text-sm text-gray-600'>
          {formatDateTime(date) || '-'}
        </span>
      ),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 120,
      render: (_: unknown, record: CIType) => (
        <Space>
          <Tooltip title="Edit">
            <Button
              aria-label="Edit"
              type='text'
              icon={<Edit className='w-4 h-4' />}
              onClick={() => handleEdit(record)}
              size='small'
            />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this CI type?"
            description="If there is a CI instance using this type, it cannot be deleted."
            onConfirm={() => handleDelete(record.id)}
            okText="OK"
            cancelText="Cancel"
          >
            <Tooltip title="Delete">
              <Button aria-label="Delete" type='text' icon={<Trash2 className='w-4 h-4' />} danger size='small' />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className='p-6'>
      {/* 页面标题 */}
      <div className='mb-6'>
        <Title level={2} className='!mb-2'>
          <Database className='inline-block w-6 h-6 mr-2' />

          CI type management
        </Title>
        <Text type='secondary'>Manage CMDB configuration item type, customize IT infrastructure classification</Text>
      </div>

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="How to use CI type"
        intro="CI type defines the field template of the configuration item. When creating a configuration item (CI instance) in the CMDB, first select the type, and then enter the fields according to the attribute template of the type."
        steps={[
          'Click "Create Type" to fill in basic information such as name; it is recommended to organize by asset hierarchy, such as "Server → Physical Server/Virtual Machine".',
          'Attribute template: maintain fields (name, type, required) line by line in the form, supporting five types of text, number, Boolean, date, and enumeration selection; after making changes, click "Verify Template Configuration" to confirm that it is correct, and "JSON Preview" is used to check the final structure, which is automatically saved when submitting.',
          'Inherited type (optional): After selecting a parent type, it automatically inherits all its attributes. When the current type defines attributes with the same name, it overwrites the parent type to avoid repeated maintenance of public fields.',
          'Enable/disable type through inline switch; disabling will not delete existing configuration item data.',
          'Restriction on deletion: CI instances using this type cannot be deleted, and related instances must be migrated or cleaned up first.',
        ]}
      />

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={8}>
          <Card className='enterprise-card'>
            <Statistic
              title="Total number of types"
              value={stats.total}
              prefix={<Database className='w-5 h-5' />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className='enterprise-card'>
            <Statistic
              title="Activation type"
              value={stats.active}
              prefix={<CheckCircle className='w-5 h-5' />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className='enterprise-card'>
            <Statistic
              title="Deactivation type"
              value={stats.inactive}
              prefix={<AlertCircle className='w-5 h-5' />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className='enterprise-card mb-6'>
        <Row gutter={[16, 16]} align='middle'>
          <Col xs={24} sm={12} md={8}>
            <Input
              placeholder="Search CI type..."
              prefix={<Search className='w-4 h-4' />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
              size='middle'
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="Status"
              value={statusFilter}
              onChange={setStatusFilter}
              allowClear
              style={{ width: '100%' }}
              size='middle'
              options={[{ value: 'active', label: 'Active' }, { value: 'inactive', label: 'Disable' }]}
            />
          </Col>
          <Col xs={24} sm={24} md={12}>
            <Space>
              <Button
                type='primary'
                icon={<Plus className='w-4 h-4' />}
                onClick={() => {
                  setEditingType(null);
                  setHasUnsupportedSchema(false);
                  form.resetFields();
                  form.setFieldsValue({ isActive: true, color: '#1890ff', schemaFields: [] });
                  setShowModal(true);
                }}
              >

                Create a new CI type
              </Button>
              <Button icon={<RefreshCw className='w-4 h-4' />} onClick={fetchCITypes}>

                Refresh
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* CI类型表格 */}
      <Card className='enterprise-card'>
        <Table
          columns={columns}
          dataSource={filteredTypes}
          rowKey='id'
          loading={loading}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => `Total ${total} records`,
          }}
          scroll={{ x: 800 }}
        />
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={
          <span>
            <Database className='w-4 h-4 mr-2' />
            {editingType ? 'Edit CI type' : 'Create a new CI type'}
          </span>
        }
        open={showModal}
        onCancel={() => {
          setShowModal(false);
          setEditingType(null);
          setHasUnsupportedSchema(false);
          form.resetFields();
        }}
        footer={
          <Space>
            <Button
              onClick={() => {
                setShowModal(false);
                setEditingType(null);
                setHasUnsupportedSchema(false);
                form.resetFields();
              }}
            >

              Cancel
            </Button>
            <Button type='primary' htmlType='submit' form={formId}>
              {editingType ? 'Update' : 'Create'}
            </Button>
          </Space>
        }
        width={760}
      >
        <Form
          id={formId}
          form={form}
          layout='vertical'
          onFinish={handleSubmit}
          initialValues={{ isActive: true, color: '#1890ff' }}
        >
          <Form.Item
            name='name'
            label="Type Name"
            rules={[
              { required: true, message: 'Please enter the type name' },
              { max: 255, message: 'Type name cannot exceed 255 characters' },
            ]}
          >
            <Input placeholder="Please enter the type name, such as: server, database, network device" />
          </Form.Item>

          <Form.Item
            name='description'
            label="Description"
            rules={[{ max: 1000, message: 'Description cannot exceed 1000 characters' }]}
          >
            <Input.TextArea rows={3} placeholder="Please enter a type description" showCount maxLength={1000} />
          </Form.Item>

          <Form.Item
            name='parentTypeId'
            label="Inherited type"
            extra="Inherit the attribute definition of the parent type; when the current type defines an attribute with the same name, it will overwrite the parent type."
          >
            <Select
              allowClear
              showSearch
              optionFilterProp='label'
              placeholder="Optional, for example &quot;physical server&quot; inherits &quot;server&quot;"
              options={ciTypes
                .filter(type => type.id !== editingType?.id)
                .map(type => ({ label: type.name, value: type.id }))}
            />
          </Form.Item>

          <Form.Item
            label="Property template"
            extra="These fields will appear in the configuration item entry form, supporting five types of text/number/Boolean/date/enumeration selection, suitable for standard fields such as environment, responsible team, instance specifications, etc."
          >
            {hasUnsupportedSchema && (
              <Alert
                className='mb-3'
                type='warning'
                showIcon
                title="Historical attribute format detected"
                description="To avoid silent loss of unknown fields, overwriting and saving of the current page is prohibited. Please convert the template format via controlled migration first."
              />
            )}
            <Alert
              className='mb-3'
              type='info'
              showIcon
              title="No need to write JSON by hand, just configure it item by field"
              description="Field identifier is used to save data. It is recommended to use English lowercase and underscores; enumeration type options can be separated by newlines or commas."
            />
            <Form.List name='schemaFields'>
              {(fields, { add, remove }) => (
                <div className='space-y-3'>
                  {fields.map(({ key, name, ...restField }, index) => (
                    <Card
                      key={key}
                      size='small'
                      title={`Field ${index + 1}`}
                      extra={
                        <Button size='small' danger type='text' onClick={() => remove(name)}>

                          Delete
                        </Button>
                      }
                    >
                      <Row gutter={12}>
                        <Col xs={24} md={12}>
                          <Form.Item
                            {...restField}
                            name={[name, 'key']}
                            label="Field Key"
                            rules={[
                              { required: true, message: 'Please enter the field ID' },
                              {
                                pattern: /^[a-z][a-z0-9_]*$/,
                                message: 'Only supports English lowercase, numbers and underscores, and starts with a letter',
                              },
                            ]}
                          >
                            <Input placeholder="for example environment" allowClear />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={12}>
                          <Form.Item
                            {...restField}
                            name={[name, 'label']}
                            label="Field Name"
                            rules={[{ required: true, message: 'Please enter the field name' }]}
                          >
                            <Input placeholder="example environment" allowClear />
                          </Form.Item>
                        </Col>
                      </Row>
                      <Row gutter={12}>
                        <Col xs={24} md={12}>
                          <Form.Item
                            {...restField}
                            name={[name, 'type']}
                            label="Field type"
                            initialValue='string'
                            rules={[{ required: true, message: 'Please select field type' }]}
                          >
                            <Select options={ATTRIBUTE_FIELD_TYPE_OPTIONS} />
                          </Form.Item>
                          {schemaFields?.[name]?.type === 'select' && (
                            <Form.Item
                              {...restField}
                              name={[name, 'options']}
                              label="optional"
                              rules={[{ required: true, message: 'Please enter at least one option' }]}
                            >
                              <Input.TextArea
                                rows={3}
                                placeholder={'Production\nPre-release\nDevelopment'}
                                allowClear
                              />
                            </Form.Item>
                          )}
                        </Col>
                        <Col xs={24} md={12}>
                          <Form.Item {...restField} name={[name, 'placeholder']} label="Input prompt">
                            <Input placeholder="For example please select environment" allowClear />
                          </Form.Item>
                          <Form.Item
                            {...restField}
                            name={[name, 'required']}
                            label="Is it required?"
                            valuePropName='checked'
                          >
                            <Switch checkedChildren="Required" unCheckedChildren="Optional" />
                          </Form.Item>
                        </Col>
                      </Row>
                    </Card>
                  ))}
                  <Button
                    type='dashed'
                    block
                    icon={<Plus className='w-4 h-4' />}
                    onClick={() =>
                      add({
                        key: '',
                        label: '',
                        type: 'string',
                        options: '',
                        required: false,
                      })
                    }
                  >

                    Add field
                  </Button>
                </div>
              )}
            </Form.List>
          </Form.Item>

          <div className='-mt-4 mb-4 flex flex-wrap gap-2'>
            <Button size='small' onClick={handleUseDefaultAttributeSchema}>

              Using sample templates
            </Button>
            <Button size='small' onClick={handleFormatAttributeSchema}>

              Verification template configuration
            </Button>
            <Button
              size='small'
              onClick={() =>
                form.setFieldsValue({
                  schemaFields: [],
                })
              }
            >

              Clear template
            </Button>
          </div>

          <Form.Item
            label="JSON Preview"
            extra="The JSON is automatically saved on submission. It can be used by advanced users to check the final structure."
          >
            <Input.TextArea
              rows={5}
              value={attributeSchemaPreview || ''}
              placeholder={DEFAULT_ATTRIBUTE_SCHEMA}
              readOnly
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name='icon' label="Icon ID">
                <Select placeholder="Select icon ID" allowClear options={iconOptions} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='color' label="Color">
                <Select placeholder="Select color" options={colorOptions.map(opt => ({ value: opt.value, label: opt.label }))} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name='isActive' label="Status" valuePropName='checked' initialValue={true}>
            <Switch checkedChildren="Active" unCheckedChildren="Disable" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default CMDBTypesManagement;
