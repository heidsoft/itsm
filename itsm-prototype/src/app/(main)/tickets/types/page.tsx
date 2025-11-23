'use client';

import React, { useState } from 'react';
import {
  Button,
  Table,
  Space,
  Tag,
  Modal,
  message,
  Tooltip,
  Badge,
  Popconfirm,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CopyOutlined,
  EyeOutlined,
  CheckCircleOutlined,
  StopOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { TicketTypeDefinition, TicketTypeStatus } from '@/types/ticket-type';
import { TicketTypeFormModal } from '@/components/business/TicketTypeFormModal';

export default function TicketTypesPage() {
  const [ticketTypes, setTicketTypes] = useState<TicketTypeDefinition[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingType, setEditingType] = useState<TicketTypeDefinition | null>(null);
  const [viewModalVisible, setViewModalVisible] = useState(false);
  const [viewingType, setViewingType] = useState<TicketTypeDefinition | null>(null);

  // 获取工单类型列表
  const fetchTicketTypes = async () => {
    setLoading(true);
    try {
      // TODO: 调用API获取工单类型列表
      // const response = await ticketTypeApi.list();
      // setTicketTypes(response.types);
      
      // 模拟数据
      setTicketTypes([
        {
          id: 1,
          code: 'incident',
          name: '故障工单',
          description: '用于报告和跟踪系统故障',
          icon: 'BugOutlined',
          color: '#ff4d4f',
          status: TicketTypeStatus.ACTIVE,
          customFields: [],
          approvalEnabled: true,
          slaEnabled: true,
          autoAssignEnabled: true,
          createdBy: 1,
          createdByName: '管理员',
          createdAt: '2024-01-01T00:00:00Z',
          updatedAt: '2024-01-01T00:00:00Z',
          tenantId: 1,
          usageCount: 150,
        },
        {
          id: 2,
          code: 'request',
          name: '服务请求',
          description: '用于处理各类服务请求',
          icon: 'CustomerServiceOutlined',
          color: '#1890ff',
          status: TicketTypeStatus.ACTIVE,
          customFields: [],
          approvalEnabled: true,
          slaEnabled: true,
          autoAssignEnabled: false,
          createdBy: 1,
          createdByName: '管理员',
          createdAt: '2024-01-01T00:00:00Z',
          updatedAt: '2024-01-01T00:00:00Z',
          tenantId: 1,
          usageCount: 280,
        },
      ]);
    } catch (error) {
      message.error('获取工单类型列表失败');
    } finally {
      setLoading(false);
    }
  };

  React.useEffect(() => {
    fetchTicketTypes();
  }, []);

  // 创建工单类型
  const handleCreate = () => {
    setEditingType(null);
    setModalVisible(true);
  };

  // 编辑工单类型
  const handleEdit = (record: TicketTypeDefinition) => {
    setEditingType(record);
    setModalVisible(true);
  };

  // 查看工单类型
  const handleView = (record: TicketTypeDefinition) => {
    setViewingType(record);
    setViewModalVisible(true);
  };

  // 复制工单类型
  const handleCopy = async (record: TicketTypeDefinition) => {
    try {
      // TODO: 调用API复制工单类型
      message.success('工单类型复制成功');
      fetchTicketTypes();
    } catch (error) {
      message.error('复制失败');
    }
  };

  // 删除工单类型
  const handleDelete = async (id: number) => {
    try {
      // TODO: 调用API删除工单类型
      message.success('工单类型删除成功');
      fetchTicketTypes();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 启用/停用工单类型
  const handleToggleStatus = async (id: number, currentStatus: TicketTypeStatus) => {
    try {
      const newStatus = currentStatus === TicketTypeStatus.ACTIVE 
        ? TicketTypeStatus.INACTIVE 
        : TicketTypeStatus.ACTIVE;
      // TODO: 调用API更新状态
      message.success(`工单类型已${newStatus === TicketTypeStatus.ACTIVE ? '启用' : '停用'}`);
      fetchTicketTypes();
    } catch (error) {
      message.error('操作失败');
    }
  };

  // 提交表单
  const handleSubmit = async (values: any) => {
    try {
      if (editingType) {
        // TODO: 调用API更新工单类型
        message.success('工单类型更新成功');
      } else {
        // TODO: 调用API创建工单类型
        message.success('工单类型创建成功');
      }
      setModalVisible(false);
      fetchTicketTypes();
    } catch (error) {
      message.error('操作失败');
    }
  };

  const columns: ColumnsType<TicketTypeDefinition> = [
    {
      title: '类型编码',
      dataIndex: 'code',
      key: 'code',
      width: 120,
      render: (code: string) => <code>{code}</code>,
    },
    {
      title: '类型名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (name: string, record) => (
        <Space>
          <Badge color={record.color} />
          <span>{name}</span>
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: TicketTypeStatus) => {
        const statusConfig = {
          [TicketTypeStatus.ACTIVE]: { color: 'success', text: '启用' },
          [TicketTypeStatus.INACTIVE]: { color: 'default', text: '停用' },
          [TicketTypeStatus.DRAFT]: { color: 'warning', text: '草稿' },
        };
        const config = statusConfig[status];
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '功能配置',
      key: 'features',
      width: 200,
      render: (_, record) => (
        <Space>
          {record.approvalEnabled && (
            <Tooltip title="需要审批">
              <Tag icon={<CheckCircleOutlined />} color="blue">
                审批
              </Tag>
            </Tooltip>
          )}
          {record.slaEnabled && (
            <Tooltip title="启用SLA">
              <Tag color="green">SLA</Tag>
            </Tooltip>
          )}
          {record.autoAssignEnabled && (
            <Tooltip title="自动分配">
              <Tag color="cyan">自动分配</Tag>
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: '使用次数',
      dataIndex: 'usageCount',
      key: 'usageCount',
      width: 100,
      sorter: true,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => new Date(date).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right',
      width: 220,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="text"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => handleView(record)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Tooltip title="复制">
            <Button
              type="text"
              size="small"
              icon={<CopyOutlined />}
              onClick={() => handleCopy(record)}
            />
          </Tooltip>
          <Tooltip title={record.status === TicketTypeStatus.ACTIVE ? '停用' : '启用'}>
            <Popconfirm
              title={`确定${record.status === TicketTypeStatus.ACTIVE ? '停用' : '启用'}此工单类型吗？`}
              onConfirm={() => handleToggleStatus(record.id, record.status)}
            >
              <Button
                type="text"
                size="small"
                icon={
                  record.status === TicketTypeStatus.ACTIVE ? (
                    <StopOutlined />
                  ) : (
                    <CheckCircleOutlined />
                  )
                }
                danger={record.status === TicketTypeStatus.ACTIVE}
              />
            </Popconfirm>
          </Tooltip>
          {record.status !== TicketTypeStatus.ACTIVE && (
            <Tooltip title="删除">
              <Popconfirm
                title="确定删除此工单类型吗？此操作不可恢复。"
                onConfirm={() => handleDelete(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Button
                  type="text"
                  size="small"
                  icon={<DeleteOutlined />}
                  danger
                />
              </Popconfirm>
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      {/* 页头 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">工单类型管理</h1>
          <p className="text-gray-500 mt-1">
            配置不同类型的工单，包括自定义字段、审批流程、SLA等
          </p>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={handleCreate}
          size="large"
        >
          创建工单类型
        </Button>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white p-4 rounded-lg shadow">
          <div className="text-gray-500 text-sm">总类型数</div>
          <div className="text-2xl font-bold mt-2">{ticketTypes.length}</div>
        </div>
        <div className="bg-white p-4 rounded-lg shadow">
          <div className="text-gray-500 text-sm">启用中</div>
          <div className="text-2xl font-bold mt-2 text-green-600">
            {ticketTypes.filter((t) => t.status === TicketTypeStatus.ACTIVE).length}
          </div>
        </div>
        <div className="bg-white p-4 rounded-lg shadow">
          <div className="text-gray-500 text-sm">需审批</div>
          <div className="text-2xl font-bold mt-2 text-blue-600">
            {ticketTypes.filter((t) => t.approvalEnabled).length}
          </div>
        </div>
        <div className="bg-white p-4 rounded-lg shadow">
          <div className="text-gray-500 text-sm">总使用次数</div>
          <div className="text-2xl font-bold mt-2 text-purple-600">
            {ticketTypes.reduce((sum, t) => sum + (t.usageCount || 0), 0)}
          </div>
        </div>
      </div>

      {/* 工单类型列表 */}
      <div className="bg-white rounded-lg shadow">
        <Table
          columns={columns}
          dataSource={ticketTypes}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </div>

      {/* 创建/编辑模态框 */}
      <TicketTypeFormModal
        visible={modalVisible}
        editingType={editingType}
        onCancel={() => setModalVisible(false)}
        onSubmit={handleSubmit}
      />

      {/* 查看详情模态框 */}
      <Modal
        title="工单类型详情"
        open={viewModalVisible}
        onCancel={() => setViewModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setViewModalVisible(false)}>
            关闭
          </Button>,
          <Button
            key="edit"
            type="primary"
            icon={<EditOutlined />}
            onClick={() => {
              setViewModalVisible(false);
              if (viewingType) {
                handleEdit(viewingType);
              }
            }}
          >
            编辑
          </Button>,
        ]}
        width={800}
      >
        {viewingType && (
          <div className="space-y-4">
            <div>
              <h3 className="font-bold mb-2">基本信息</h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <span className="text-gray-500">类型编码：</span>
                  <code>{viewingType.code}</code>
                </div>
                <div>
                  <span className="text-gray-500">类型名称：</span>
                  {viewingType.name}
                </div>
                <div className="col-span-2">
                  <span className="text-gray-500">描述：</span>
                  {viewingType.description || '-'}
                </div>
              </div>
            </div>

            <div>
              <h3 className="font-bold mb-2">功能配置</h3>
              <Space>
                <Tag color={viewingType.approvalEnabled ? 'blue' : 'default'}>
                  审批流程：{viewingType.approvalEnabled ? '启用' : '禁用'}
                </Tag>
                <Tag color={viewingType.slaEnabled ? 'green' : 'default'}>
                  SLA：{viewingType.slaEnabled ? '启用' : '禁用'}
                </Tag>
                <Tag color={viewingType.autoAssignEnabled ? 'cyan' : 'default'}>
                  自动分配：{viewingType.autoAssignEnabled ? '启用' : '禁用'}
                </Tag>
              </Space>
            </div>

            {viewingType.customFields && viewingType.customFields.length > 0 && (
              <div>
                <h3 className="font-bold mb-2">自定义字段</h3>
                <div className="space-y-2">
                  {viewingType.customFields.map((field) => (
                    <div key={field.id} className="border-l-4 border-blue-500 pl-3">
                      <div>
                        <span className="font-medium">{field.label}</span>
                        {field.required && (
                          <Tag color="red" className="ml-2">
                            必填
                          </Tag>
                        )}
                      </div>
                      <div className="text-sm text-gray-500">
                        字段名：{field.name} | 类型：{field.type}
                      </div>
                      {field.description && (
                        <div className="text-sm text-gray-400">{field.description}</div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {viewingType.approvalChain && viewingType.approvalChain.length > 0 && (
              <div>
                <h3 className="font-bold mb-2">审批流程</h3>
                <div className="space-y-2">
                  {viewingType.approvalChain.map((chain, index) => (
                    <div key={chain.id} className="border rounded p-3">
                      <div className="font-medium">
                        第 {chain.level} 级审批：{chain.name}
                      </div>
                      <div className="text-sm text-gray-500 mt-1">
                        审批方式：
                        {chain.approvalType === 'all' && '全部通过'}
                        {chain.approvalType === 'any' && '任一通过'}
                        {chain.approvalType === 'majority' && '多数通过'}
                      </div>
                      <div className="mt-2">
                        <Space wrap>
                          {chain.approvers.map((approver, idx) => (
                            <Tag key={idx} icon={<FileTextOutlined />}>
                              {approver.name}
                            </Tag>
                          ))}
                        </Space>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}

