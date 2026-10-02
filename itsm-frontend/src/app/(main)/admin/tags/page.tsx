'use client';

import React, { useState, useEffect } from 'react';
import { App, Button, ColorPicker, Form, Input, Modal, Space, Switch, Table, Tag } from 'antd';
import { Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import type { TicketTag } from '@/lib/services/ticket-tag-service';
import { ticketTagService } from '@/lib/services/ticket-tag-service';
import { useI18n } from '@/lib/i18n';

// 全局标签的真实所有者是工单标签（/api/v1/ticket-tags）。此前本页打的是只读别名
// /api/v1/tags，列表能看、新建/编辑/删除全部 404。
interface TagFormValues {
  name: string;
  color?: unknown;
  description?: string;
  isActive?: boolean;
}

function toHex(value: unknown): string | undefined {
  if (typeof value === 'string') return value;
  const color = value as { toHexString?: () => string };
  return color?.toHexString?.();
}

export default function TagsPage() {
  const { t } = useI18n();
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<TagFormValues>();
  const [tags, setTags] = useState<TicketTag[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [fetching, setFetching] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  // null = 新建；非 null = 编辑该 id，避免「编辑」实际走 create。
  const [editing, setEditing] = useState<TicketTag | null>(null);

  const fetchTags = async (targetPage = page, targetSize = pageSize) => {
    setFetching(true);
    try {
      // 后端把 pageSize 上限收敛到 100，分页范围由后端负责，前端不再二次截断。
      const data = await ticketTagService.listTags({ page: targetPage, pageSize: targetSize });
      setTags(data.items);
      setTotal(data.total);
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('common.getFailed'));
    } finally {
      setFetching(false);
    }
  };

  // 分页参数变化由 effect 统一驱动请求；显式换页时只改状态，避免同一页拉两次。
  const reload = (targetPage = page, targetSize = pageSize) => {
    if (targetPage !== page || targetSize !== pageSize) {
      setPage(targetPage);
      setPageSize(targetSize);
      return;
    }
    fetchTags(targetPage, targetSize);
  };

  useEffect(() => {
    fetchTags();
  }, [page, pageSize]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: TicketTag) => {
    setEditing(record);
    form.setFieldsValue({
      name: record.name,
      color: record.color,
      description: record.description,
      isActive: record.isActive,
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    let values: TagFormValues;
    try {
      values = await form.validateFields();
    } catch {
      // 必填/格式提示由表单项自身展示，不额外弹全局错误
      return;
    }

    const payload = {
      name: values.name,
      color: toHex(values.color),
      description: values.description,
      isActive: values.isActive ?? true,
    };

    setSaving(true);
    try {
      if (editing) {
        await ticketTagService.updateTag(editing.id, payload);
      } else {
        await ticketTagService.createTag(payload);
      }
      message.success(t('common.saveSuccess'));
      setModalOpen(false);
      form.resetFields();
      reload(editing ? page : 1);
    } catch (error) {
      // 后端已把重名/在用/无权映射成稳定消息，这里直接透出而不是统一成「操作失败」。
      message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = (record: TicketTag) => {
    modal.confirm({
      title: '确认删除标签',
      content: `删除后无法恢复。标签「${record.name}」若仍被工单使用，后端会拒绝删除。`,
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await ticketTagService.deleteTag(record.id);
          message.success(t('common.deleteSuccess'));
          // 删掉当前页最后一条时回退一页，避免停在空页上看起来像列表坏了。
          const lastPage = Math.max(1, Math.ceil(Math.max(total - 1, 0) / pageSize));
          reload(Math.min(page, lastPage));
        } catch (error) {
          message.error(error instanceof Error ? error.message : t('common.deleteFailed'));
        }
      },
    });
  };

  const columns = [
    {
      title: '标签名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: TicketTag) => <Tag color={record.color}>{text}</Tag>,
    },
    {
      title: '颜色',
      dataIndex: 'color',
      key: 'color',
      render: (color: string) => (
        <div className="flex items-center">
          <div className="mr-2 h-4 w-4 rounded" style={{ backgroundColor: color }} />
          {color}
        </div>
      ),
    },
    { title: '描述', dataIndex: 'description', key: 'description' },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (isActive: boolean) => (isActive ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: TicketTag) => (
        <Space size="middle">
          <Button type="text" aria-label="编辑" icon={<Pencil />} onClick={() => openEdit(record)} />
          <Button
            type="text"
            danger
            aria-label="删除"
            icon={<Trash2 />}
            onClick={() => handleDelete(record)}
          />
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      header={{
        title: '全局标签管理',
        breadcrumb: { items: [{ title: '首页' }, { title: '系统管理' }, { title: '标签管理' }] },
      }}
      extra={[
        <Button key="refresh" icon={<RefreshCw />} onClick={() => reload()} loading={fetching}>
          刷新
        </Button>,
        <Button key="create" type="primary" icon={<Plus />} onClick={openCreate}>
          新建标签
        </Button>,
      ]}
    >
      <Table
        columns={columns}
        dataSource={tags}
        rowKey="id"
        loading={fetching}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (nextPage, nextSize) => reload(nextPage, nextSize),
        }}
      />

      <Modal
        title={editing ? `编辑标签：${editing.name}` : '新建标签'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={saving}
      >
        <Form form={form} layout="vertical" initialValues={{ color: '#1890ff', isActive: true }}>
          <Form.Item
            name="name"
            label="标签名称"
            rules={[{ required: true, message: '请输入标签名称' }]}
          >
            <Input placeholder="请输入标签名称" maxLength={50} />
          </Form.Item>
          <Form.Item name="color" label="颜色">
            <ColorPicker showText />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item
            name="isActive"
            label="启用"
            valuePropName="checked"
            extra="停用后仍可在历史工单上看到，但不再用于新的分类。"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
