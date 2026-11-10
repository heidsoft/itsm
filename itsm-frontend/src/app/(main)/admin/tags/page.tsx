'use client';

import React, { useState, useEffect } from 'react';
import { App, Button, ColorPicker, Form, Input, Modal, Space, Switch, Table, Tag } from 'antd';
import { Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import type { TicketTag } from '@/lib/services/ticket-tag-service';
import { ticketTagService } from '@/lib/services/ticket-tag-service';
import { useI18n } from '@/lib/i18n';

// 全局标签的真实所有者是工单标签（/api/v1/ticket-tags）；
// /api/v1/tags 只是只读别名，写操作打到它上面会 404。
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
      title: 'Confirm tag deletion',
      content: `Cannot be restored after deletion. Tag "${record.name}"If it is still used by the work order, the backend will refuse to delete it.`,
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
      title: 'Tag Name',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: TicketTag) => <Tag color={record.color}>{text}</Tag>,
    },
    {
      title: 'Color',
      dataIndex: 'color',
      key: 'color',
      render: (color: string) => (
        <div className="flex items-center">
          <div className="mr-2 h-4 w-4 rounded" style={{ backgroundColor: color }} />
          {color}
        </div>
      ),
    },
    { title: 'Description', dataIndex: 'description', key: 'description' },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (isActive: boolean) => (isActive ? <Tag color="green">Enabled</Tag> : <Tag>Disable</Tag>),
    },
    {
      title: 'Actions',
      key: 'action',
      render: (_: unknown, record: TicketTag) => (
        <Space size="middle">
          <Button type="text" aria-label="Edit" icon={<Pencil />} onClick={() => openEdit(record)} />
          <Button
            type="text"
            danger
            aria-label="Delete"
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
        title: 'Global tag management',
        breadcrumb: { items: [{ title: 'Home page' }, { title: 'System Admin' }, { title: 'Tag Management' }] },
      }}
      extra={[
        <Button key="refresh" icon={<RefreshCw />} onClick={() => reload()} loading={fetching}>

          Refresh
        </Button>,
        <Button key="create" type="primary" icon={<Plus />} onClick={openCreate}>

          Create Tag
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
        title={editing ? `Edit tag:${editing.name}` : 'Create Tag'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={saving}
      >
        <Form form={form} layout="vertical" initialValues={{ color: '#1890ff', isActive: true }}>
          <Form.Item
            name="name"
            label="Tag Name"
            rules={[{ required: true, message: 'Please enter the tag name' }]}
          >
            <Input placeholder="Please enter the tag name" maxLength={50} />
          </Form.Item>
          <Form.Item name="color" label="Color">
            <ColorPicker showText />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item
            name="isActive"
            label="Enabled"
            valuePropName="checked"
            extra="After deactivation, it can still be seen on historical tickets, but will no longer be used for new categories."
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
