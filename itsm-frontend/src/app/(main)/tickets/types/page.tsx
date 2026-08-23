'use client';

import { useCallback, useEffect, useState } from 'react';
import { App, Button, Card, Input, List, Modal, Space, Switch, Table, Tag, Typography } from 'antd';
import { Copy, Pencil, Plus, RotateCcw } from 'lucide-react';
import { TicketTypeFormModal } from '@/components/business/TicketTypeFormModal';
import { TicketTypeApi } from '@/lib/api/ticketTypeApi';
import type { CreateTicketTypeRequest, TicketTypeDefinition, TicketTypePresetDefinition, UpdateTicketTypeRequest } from '@/types/ticket-type';

const { Title, Text } = Typography;

export default function TicketTypesPage() {
  const { message, modal } = App.useApp();
  const [items, setItems] = useState<TicketTypeDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');
  const [includeArchived, setIncludeArchived] = useState(false);
  const [editing, setEditing] = useState<TicketTypeDefinition | null>(null);
  const [open, setOpen] = useState(false);
	const [presetOpen, setPresetOpen] = useState(false);
	const [presets, setPresets] = useState<TicketTypePresetDefinition[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = await TicketTypeApi.list({ keyword, includeArchived, page: 1, pageSize: 100 });
      setItems(result.types ?? []);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载工单类型失败');
    } finally { setLoading(false); }
  }, [keyword, includeArchived, message]);

  useEffect(() => { void load(); }, [load]);

  const save = async (values: unknown) => {
    try {
      if (editing) await TicketTypeApi.update(editing.id, values as UpdateTicketTypeRequest);
      else await TicketTypeApi.create(values as CreateTicketTypeRequest);
      message.success(editing ? '工单类型已更新' : '工单类型已创建');
      setOpen(false); setEditing(null); await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存工单类型失败');
      throw error; // 通知表单弹窗保持打开，便于修改后重试
    }
  };

  const clone = (item: TicketTypeDefinition) => {
    let code = `${item.code}_copy`;
    let name = `${item.name} 副本`;
    modal.confirm({
      title: '复制工单类型',
      content: <Space orientation="vertical" className="w-full"><Input defaultValue={code} onChange={e => { code = e.target.value; }} /><Input defaultValue={name} onChange={e => { name = e.target.value; }} /></Space>,
      onOk: async () => {
        try {
          await TicketTypeApi.clone(item.id, code, name);
          message.success('工单类型已复制');
          await load();
        } catch (error) {
          message.error(error instanceof Error ? error.message : '复制工单类型失败');
          throw error;
        }
      },
    });
  };

	const openPresets = async () => {
    try {
      setPresets(await TicketTypeApi.listPresets());
      setPresetOpen(true);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载预设库失败');
    }
  };

  const restore = (item: TicketTypeDefinition) => {
    modal.confirm({
      title: '恢复工单类型',
      content: `确定恢复已归档的类型「${item.name}」吗？恢复后将重新启用。`,
      okText: '恢复',
      onOk: async () => { await TicketTypeApi.restore(item.id); message.success('工单类型已恢复'); await load(); },
    });
  };

  return <div className="p-6"><Card>
    <div className="mb-5 flex items-start justify-between gap-4"><div><Title level={2} className="!mb-1">工单类型</Title><Text type="secondary">租户级类型、动态表单及流程策略配置</Text></div><Space><Button onClick={() => void openPresets()}>从预设安装</Button><Button type="primary" icon={<Plus className="h-4 w-4" />} onClick={() => { setEditing(null); setOpen(true); }}>新建类型</Button></Space></div>
    <Input.Search className="mb-4 max-w-sm" allowClear placeholder="搜索名称或编码" onSearch={setKeyword} />
    <div className="mb-3"><label className="inline-flex cursor-pointer items-center gap-2 text-sm text-gray-600"><input type="checkbox" checked={includeArchived} onChange={e => setIncludeArchived(e.target.checked)} />显示已归档类型</label></div>
    <Table rowKey="id" loading={loading} dataSource={items} columns={[
      { title: '名称', dataIndex: 'name', render: (value: string) => <Text strong>{value}</Text> },
      { title: '编码', dataIndex: 'code', render: (value: string) => <Tag color="blue">{value}</Tag> },
      { title: '动态字段', dataIndex: 'customFields', render: (value: unknown[]) => `${value?.length ?? 0} 个` },
	  { title: '默认优先级', dataIndex: 'defaultPriority', render: (value: string) => <Tag>{value}</Tag> },
	  { title: '工作流', dataIndex: 'workflowDefinitionKey', render: (value?: string) => value || '-' },
      { title: 'SLA', dataIndex: 'slaEnabled', render: (value: boolean) => value ? '已绑定' : '-' },
	  { title: '排序', dataIndex: 'sortOrder', width: 70 },
      { title: '状态', dataIndex: 'status', render: (value: string, item) => item.archivedAt ? <Tag color="default">已归档</Tag> : <Tag color={value === 'active' ? 'green' : 'default'}>{value === 'active' ? '启用' : '停用'}</Tag> },
      { title: '更新时间', dataIndex: 'updatedAt', render: (value: string) => value ? new Date(value).toLocaleString() : '-' },
      { title: '操作', width: 300, render: (_, item) => item.archivedAt ? <Space><Button size="small" type="primary" ghost icon={<RotateCcw className="h-3.5 w-3.5" />} onClick={() => restore(item)}>恢复</Button></Space> : <Space><Button size="small" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => { setEditing(item); setOpen(true); }}>编辑</Button><Button size="small" icon={<Copy className="h-3.5 w-3.5" />} onClick={() => clone(item)}>复制</Button><Switch size="small" checked={item.status === 'active'} onChange={async enabled => { try { await TicketTypeApi.setEnabled(item.id, enabled); message.success(enabled ? '工单类型已启用' : '工单类型已停用'); await load(); } catch (error) { message.error(error instanceof Error ? error.message : '操作失败'); } }} /></Space> },
    ]} />
  </Card><TicketTypeFormModal visible={open} editingType={editing} onCancel={() => { setOpen(false); setEditing(null); }} onSubmit={save} /><Modal title="工单类型预设库" open={presetOpen} footer={null} onCancel={() => setPresetOpen(false)}><List dataSource={presets} renderItem={preset => <List.Item actions={[<Button key="install" type="link" onClick={async () => { await TicketTypeApi.installPreset(preset.id); message.success('预设已安装'); setPresetOpen(false); await load(); }}>安装</Button>]}><List.Item.Meta title={preset.name} description={`${preset.category} · ${preset.description}`} /></List.Item>} /></Modal></div>;
}
