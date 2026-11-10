'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Card,
  Button,
  Input,
  Space,
  Select,
  Tag,
  Modal,
  Form,
  App,
  Popconfirm,
  Row,
  Col,
  Statistic,
  Descriptions,
  Typography,
  Empty,
  Tooltip,
} from 'antd';
import { Search, Plus, Pencil, Trash2, Eye, RotateCcw, AlertCircle, CheckCircle, XCircle } from 'lucide-react';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { KEDBResponse, KEDBStatsResponse } from '@/lib/api/kedb-api';
import { KEDBApi } from '@/lib/api/kedb-api';
import { formatDateTime } from '@/lib/formatters';

const { TextArea } = Input;
const { Text } = Typography;

// 状态配置
const statusConfig: Record<string, { color: string; text: string }> = {
  draft: { color: 'default', text: 'Draft' },
  active: { color: 'processing', text: 'Active' },
  resolved: { color: 'success', text: 'Resolved' },
  deprecated: { color: 'error', text: 'Obsolete' },
};

// 严重程度配置
const severityConfig: Record<string, { color: string; text: string }> = {
  critical: { color: 'red', text: 'Critical' },
  high: { color: 'orange', text: 'High' },
  medium: { color: 'gold', text: 'Medium' },
  low: { color: 'green', text: 'Low' },
};

export default function KnownErrorsPage() {
  const { message } = App.useApp();
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空表格。
  const [listLoading, setListLoading] = useState(true);
  const [data, setData] = useState<KEDBResponse[]>([]);
  const [stats, setStats] = useState<KEDBStatsResponse | null>(null);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [searchParams, setSearchParams] = useState({
    keyword: '',
    status: '',
    category: '',
    severity: '',
  });
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isViewModalOpen, setIsViewModalOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<KEDBResponse | null>(null);
  const [viewRecord, setViewRecord] = useState<KEDBResponse | null>(null);
  const [form] = Form.useForm();
  const [categories, setCategories] = useState<string[]>([]);

  useEffect(() => {
    fetchStats();
    fetchCategories();
  }, []);

  // 状态驱动重取：分页/筛选变化即重新拉取。
  // 2026-10-14 修复：此前 handleSearch/handleTableChange 先 setState 后立即调 fetchData()，
  // fetchData 闭包里读的是旧 pagination（停第 N 页搜索会拿第 N 页数据），且无 effect 补救。
  const fetchData = useCallback(async () => {
    setListLoading(true);
    try {
      const params = {
        page: pagination.current,
        pageSize: pagination.pageSize,
        ...searchParams,
      };
      // 移除空参数
      Object.keys(params).forEach((key) => {
        if (params[key as keyof typeof params] === '') {
          delete params[key as keyof typeof params];
        }
      });
      const response = await KEDBApi.getKnownErrors(params);
      setData(response.items);
      setPagination((prev) => ({ ...prev, total: response.total }));
    } catch (error) {
      message.error('Failed to obtain data');
    } finally {
      setListLoading(false);
    }
  }, [pagination.current, pagination.pageSize, searchParams]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const fetchStats = async () => {
    try {
      const response = await KEDBApi.getStats();
      setStats(response);
    } catch (error) {
      console.error('Failed to obtain statistics', error);
      message.error('Failed to obtain known error statistics');
    }
  };

  const fetchCategories = async () => {
    try {
      const response = await KEDBApi.getCategories();
      setCategories(response.items);
    } catch (error) {
      console.error('Failed to obtain classification', error);
      message.error('Failed to obtain classification');
    }
  };

  const handleSearch = () => {
    // 只改状态，重取由 [fetchData] effect 驱动（带着最新 searchParams）
    setPagination((prev) => ({ ...prev, current: 1 }));
  };

  const handleReset = () => {
    setSearchParams({ keyword: '', status: '', category: '', severity: '' });
    setPagination((prev) => ({ ...prev, current: 1 }));
  };

  const handleTableChange = (pag: TablePaginationConfig) => {
    // 只改状态，重取由 [fetchData] effect 驱动（带着最新 pagination）
    setPagination({ ...pagination, current: pag.current || 1, pageSize: pag.pageSize || 20 });
  };

  const handleAdd = () => {
    setEditingRecord(null);
    form.resetFields();
    setIsModalOpen(true);
  };

  const handleEdit = (record: KEDBResponse) => {
    setEditingRecord(record);
    form.setFieldsValue({
      title: record.title,
      description: record.description,
      symptoms: record.symptoms,
      rootCause: record.rootCause,
      workaround: record.workaround,
      resolution: record.resolution,
      category: record.category,
      severity: record.severity,
      status: record.status,
      affectedProducts: record.affectedProducts,
      affectedCis: record.affectedCis,
      keywords: record.keywords,
    });
    setIsModalOpen(true);
  };

  const handleView = (record: KEDBResponse) => {
    setViewRecord(record);
    setIsViewModalOpen(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await KEDBApi.deleteKnownError(id);
      message.success('Deleted');
      fetchData();
      fetchStats();
    } catch (error) {
      message.error('Delete failed');
    }
  };

  const handlePromote = async (id: number) => {
    try {
      await KEDBApi.promoteToKnownError(id);
      message.success('Promotion successful');
      fetchData();
      fetchStats();
    } catch (error) {
      message.error('Promotion failed');
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingRecord) {
        await KEDBApi.updateKnownError(editingRecord.id, values);
        message.success('Updated');
      } else {
        await KEDBApi.createKnownError(values);
        message.success('Created');
      }
      setIsModalOpen(false);
      fetchData();
      fetchStats();
    } catch (error) {
      message.error('Operation failed');
    }
  };

  const columns: ColumnsType<KEDBResponse> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
      fixed: 'left',
    },
    {
      title: 'Title',
      dataIndex: 'title',
      fixed: 'left',
      width: 200,
      ellipsis: true,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => {
        const config = statusConfig[status] || { color: 'default', text: status };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: 'Severity',
      dataIndex: 'severity',
      width: 100,
      render: (severity: string) => {
        const config = severityConfig[severity] || { color: 'default', text: severity };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: 'Category',
      dataIndex: 'category',
      width: 100,
    },
    {
      title: 'Number of occurrences',
      dataIndex:'occurrenceCount',
      width: 100,
    },
    {
      title: 'Keywords',
      dataIndex: 'keywords',
      width: 150,
      ellipsis: true,
      render: (keywords: string[]) =>
        keywords?.slice(0, 3).map((k, i) => (
          <Tag key={i} style={{ marginRight: 4 }}>
            {k}
          </Tag>
        )),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      width: 160,
      render: (text: string) => formatDateTime(text),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 180,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="View Details">
            <Button
              aria-label="View Details"
              type="link"
              size="small"
              icon={<Eye />}
              onClick={() => handleView(record)}
            />
          </Tooltip>
          <Tooltip title="Edit">
            <Button
              aria-label="Edit"
              type="link"
              size="small"
              icon={<Pencil />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          {record.status === 'draft' && (
            <Tooltip title="Promoted to officially known bug">
              <Button
                aria-label="Promoted to officially known bug"
                type="link"
                size="small"
                icon={<CheckCircle />}
                onClick={() => handlePromote(record.id)}
              />
            </Tooltip>
          )}
          <Popconfirm
            title="Confirm Delete"
            description="Are you sure you want to delete this known error?"
            onConfirm={() => handleDelete(record.id)}
            okText="Confirm"
            cancelText="Cancel"
          >
            <Tooltip title="Delete">
              <Button aria-label="Delete" type="link" size="small" danger icon={<Trash2 />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        {/* 统计卡片 */}
        {stats && (
          <Row gutter={16}>
            <Col span={6}>
              <Card>
                <Statistic title="Total" value={stats.total} />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="Active"
                  value={stats.active}
                  styles={{ content: { color: '#1890ff' } }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="Resolved"
                  value={stats.resolved}
                  styles={{ content: { color: '#52c41a' } }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="Obsolete"
                  value={stats.deprecated}
                  styles={{ content: { color: '#ff4d4f' } }}
                />
              </Card>
            </Col>
          </Row>
        )}

        {/* 搜索栏 */}
        <Card>
          <Space wrap>
            <Input.Search
              placeholder="Search title, description, symptoms..."
              style={{ width: 250 }}
              allowClear
              onSearch={handleSearch}
              onChange={(e) =>
                setSearchParams((prev) => ({ ...prev, keyword: e.target.value }))
              }
            />
            <Select
              placeholder="Status"
              style={{ width: 120 }}
              allowClear
              value={searchParams.status || undefined}
              onChange={(value) =>
                setSearchParams((prev) => ({ ...prev, status: value || '' }))
              }
            >
              <Select.Option value="draft">Draft</Select.Option>
              <Select.Option value="active">Active</Select.Option>
              <Select.Option value="resolved">Resolved</Select.Option>
              <Select.Option value="deprecated">Obsolete</Select.Option>
            </Select>
            <Select
              placeholder="Severity"
              style={{ width: 120 }}
              allowClear
              value={searchParams.severity || undefined}
              onChange={(value) =>
                setSearchParams((prev) => ({ ...prev, severity: value || '' }))
              }
            >
              <Select.Option value="critical">Critical</Select.Option>
              <Select.Option value="high">High</Select.Option>
              <Select.Option value="medium">Medium</Select.Option>
              <Select.Option value="low">Low</Select.Option>
            </Select>
            <Select
              placeholder="Category"
              style={{ width: 150 }}
              allowClear
              value={searchParams.category || undefined}
              onChange={(value) =>
                setSearchParams((prev) => ({ ...prev, category: value || '' }))
              }
            >
              {categories.map((cat) => (
                <Select.Option key={cat} value={cat}>
                  {cat}
                </Select.Option>
              ))}
            </Select>
            <Space>
              <Button type="primary" icon={<Search />} onClick={handleSearch}>

                Search
              </Button>
              <Button onClick={handleReset}>Reset</Button>
            </Space>
          </Space>
        </Card>

        {/* 数据表格 */}
        <Card
          title="Known bug list"
          extra={
            <Button type="primary" icon={<Plus />} onClick={handleAdd}>

              New known error
            </Button>
          }
        >
          <Table
            columns={columns}
            dataSource={data}
            loading={listLoading}
            rowKey="id"
            scroll={{ x: 1200 }}
            pagination={{
              ...pagination,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total) => `Total ${total} items`,
              pageSizeOptions: ['10', '20', '50', '100'],
            }}
            onChange={handleTableChange}
          />
        </Card>
      </Space>

      {/* 新建/编辑 Modal */}
      <Modal
        title={editingRecord ? 'Edit known error' : 'New known error'}
        open={isModalOpen}
        onOk={handleSubmit}
        onCancel={() => setIsModalOpen(false)}
        width={700}
        okText="Confirm"
        cancelText="Cancel"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="title"
            label="Title"
            rules={[{ required: true, message: 'Please enter a title' }]}
          >
            <Input placeholder="Please enter a title" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <TextArea rows={2} placeholder="Please enter a description" />
          </Form.Item>
          <Form.Item name="symptoms" label="Symptom description">
            <TextArea rows={2} placeholder="Please enter a symptom description" />
          </Form.Item>
          <Form.Item name="rootCause" label="Root Cause">
            <TextArea rows={2} placeholder="Please enter root cause" />
          </Form.Item>
          <Form.Item name="workaround" label="Workaround">
            <TextArea rows={2} placeholder="Please enter a temporary solution" />
          </Form.Item>
          <Form.Item name="resolution" label="Permanent solution">
            <TextArea rows={2} placeholder="Please enter a permanent solution" />
          </Form.Item>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="category" label="Category">
                <Input placeholder="Please enter the category" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="severity" label="Severity">
                <Select placeholder="Please select">
                  <Select.Option value="critical">Critical</Select.Option>
                  <Select.Option value="high">High</Select.Option>
                  <Select.Option value="medium">Medium</Select.Option>
                  <Select.Option value="low">Low</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            {editingRecord && (
              <Col span={8}>
                <Form.Item name="status" label="Status">
                  <Select placeholder="Please select">
                    <Select.Option value="draft">Draft</Select.Option>
                    <Select.Option value="active">Active</Select.Option>
                    <Select.Option value="resolved">Resolved</Select.Option>
                    <Select.Option value="deprecated">Obsolete</Select.Option>
                  </Select>
                </Form.Item>
              </Col>
            )}
          </Row>
          <Form.Item name="affectedProducts" label="Affected products/services">
            <Select
              mode="tags"
              placeholder="Press enter after entering to add"
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item name="affectedCis" label="Affected configuration items">
            <Select
              mode="tags"
              placeholder="Press enter after entering to add"
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item name="keywords" label="Keywords">
            <Select
              mode="tags"
              placeholder="Press enter after entering to add"
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 查看详情 Modal */}
      <Modal
        title="Known error details"
        open={isViewModalOpen}
        onCancel={() => setIsViewModalOpen(false)}
        footer={
          <Button onClick={() => setIsViewModalOpen(false)}>Close</Button>
        }
        width={800}
      >
        {viewRecord && (
          <Descriptions column={2} bordered>
            <Descriptions.Item label="ID">{viewRecord.id}</Descriptions.Item>
            <Descriptions.Item label="Status">
              <Tag color={statusConfig[viewRecord.status]?.color}>
                {statusConfig[viewRecord.status]?.text}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Title" span={2}>
              {viewRecord.title}
            </Descriptions.Item>
            <Descriptions.Item label="Description" span={2}>
              {viewRecord.description || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Symptom description" span={2}>
              {viewRecord.symptoms || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Root Cause" span={2}>
              {viewRecord.rootCause || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Workaround" span={2}>
              {viewRecord.workaround || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Permanent solution" span={2}>
              {viewRecord.resolution || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Category">{viewRecord.category}</Descriptions.Item>
            <Descriptions.Item label="Severity">
              <Tag color={severityConfig[viewRecord.severity]?.color}>
                {severityConfig[viewRecord.severity]?.text}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Number of occurrences">
              {viewRecord.occurrenceCount}
            </Descriptions.Item>
            <Descriptions.Item label="Keywords" span={2}>
              {viewRecord.keywords?.map((k, i) => (
                <Tag key={i}>{k}</Tag>
              )) || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Affected products" span={2}>
              {viewRecord.affectedProducts?.join(', ') || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Affected configuration items" span={2}>
              {viewRecord.affectedCis?.join(', ') || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Created">
              {formatDateTime(viewRecord.createdAt)}
            </Descriptions.Item>
            <Descriptions.Item label="Updated At">
              {formatDateTime(viewRecord.updatedAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}