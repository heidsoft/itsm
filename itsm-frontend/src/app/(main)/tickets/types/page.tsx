'use client';

import { useCallback, useEffect, useState } from 'react';
import { Alert, App, Button, Card, Input, List, Modal, Space, Switch, Table, Tag, Typography } from 'antd';
import { Copy, Pencil, Plus, RotateCcw } from 'lucide-react';
import { TicketTypeFormModal } from '@/components/business/TicketTypeFormModal';
import { TicketTypeApi } from '@/lib/api/ticketTypeApi';
import { isPermissionDeniedError, serverErrorMessage } from '@/lib/api/http-client';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { formatDateTime } from '@/lib/formatters';
import type { CreateTicketTypeRequest, TicketTypeDefinition, TicketTypePresetDefinition, UpdateTicketTypeRequest } from '@/types/ticket-type';

const { Title, Text } = Typography;

// 后端路由按 ticket_type 细分动作码守卫，403 的 message 只有「权限不足」。
// 这里把需要的码点名出来，使用者才知道该申请哪一项，而不是去改筛选条件。
function describeFailure(error: unknown, fallback: string, permission: string): string {
  const detail = serverErrorMessage(error, fallback);
  return isPermissionDeniedError(error)
    ? `${detail}. This operation requires ${permission}, please contact the administrator to open for the current role`
    : detail;
}

export default function TicketTypesPage() {
  const { message, modal } = App.useApp();
  const { hasPermission } = usePermissions();
  const canManage = hasPermission('ticket_type', 'manage');
  const canInstallPreset = hasPermission('ticket_type', 'install_preset');
  const [items, setItems] = useState<TicketTypeDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');
  const [includeArchived, setIncludeArchived] = useState(false);
  const [editing, setEditing] = useState<TicketTypeDefinition | null>(null);
  const [open, setOpen] = useState(false);
	const [presetOpen, setPresetOpen] = useState(false);
	const [presets, setPresets] = useState<TicketTypePresetDefinition[]>([]);
	const [installingId, setInstallingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = await TicketTypeApi.list({ keyword, includeArchived, page: 1, pageSize: 100 });
      setItems(result.items ?? []);
    } catch (error) {
      message.error(describeFailure(error, 'Failed to load work order type', 'ticket:read'));
    } finally { setLoading(false); }
  }, [keyword, includeArchived, message]);

  useEffect(() => { void load(); }, [load]);

  const save = async (values: unknown) => {
    try {
      if (editing) await TicketTypeApi.update(editing.id, values as UpdateTicketTypeRequest);
      else await TicketTypeApi.create(values as CreateTicketTypeRequest);
      message.success(editing ? 'Ticket type updated' : 'Work order type has been created');
      setOpen(false); setEditing(null); await load();
    } catch (error) {
      message.error(describeFailure(error, 'Failed to save work order type', 'ticket_type:manage'));
      throw error; // 通知表单弹窗保持打开，便于修改后重试
    }
  };

  const clone = (item: TicketTypeDefinition) => {
    let code = `${item.code}_copy`;
    let name = `${item.name} Copy`;
    modal.confirm({
      title: 'Copy ticket type',
      content: <Space orientation="vertical" className="w-full"><Input defaultValue={code} onChange={e => { code = e.target.value; }} /><Input defaultValue={name} onChange={e => { name = e.target.value; }} /></Space>,
      onOk: async () => {
        try {
          await TicketTypeApi.clone(item.id, code, name);
          message.success('Ticket type copied');
          await load();
        } catch (error) {
          message.error(describeFailure(error, 'Failed to copy ticket type', 'ticket_type:manage'));
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
      message.error(describeFailure(error, 'Failed to load preset library', 'ticket_type:read'));
    }
  };

  const restore = (item: TicketTypeDefinition) => {
    modal.confirm({
      title: 'Restore work order type',
      content: `Determine restore archived type "${item.name}"?" Will be re-enabled after recovery.`,
      okText: 'Restore',
      onOk: async () => {
        try {
          await TicketTypeApi.restore(item.id);
          message.success('Work order type has been restored');
          await load();
        } catch (error) {
          message.error(describeFailure(error, 'Failed to restore work order type', 'ticket_type:manage'));
        }
      },
    });
  };

  return <div className="p-6"><Card>
    <div className="mb-5 flex items-start justify-between gap-4"><div><Title level={2} className="!mb-1">Ticket Types</Title><Text type="secondary">Tenant-level type, dynamic form and process policy configuration</Text></div><Space><Button onClick={() => void openPresets()}>Install from default</Button>{canManage && <Button type="primary" icon={<Plus className="h-4 w-4" />} onClick={() => { setEditing(null); setOpen(true); }}>New type</Button>}</Space></div>
    <Alert
      className="mb-4"
      type={canManage ? 'info' : 'warning'}
      showIcon
      title={canManage
        ? 'Enable after binding workflow, SLA, allocation rules and approval levels. It is recommended to create a test work order for acceptance.'
        : 'The current role only has view permissions. New, edit, copy, start, stop, and restore require ticket_type:manage. Preset installation requires ticket_type:install_preset.'}
    />
    <Input.Search className="mb-4 max-w-sm" allowClear placeholder="Search name or code" onSearch={setKeyword} />
    <div className="mb-3"><label className="inline-flex cursor-pointer items-center gap-2 text-sm text-gray-600"><input type="checkbox" checked={includeArchived} onChange={e => setIncludeArchived(e.target.checked)} />Show archived types</label></div>
    {/* 每列显式设置宽度并对长文本列开启 ellipsis，避免列宽被均分导致字段拥挤；总宽超出容器时通过 scroll.x 横向滚动兜底 */}
    <Table rowKey="id" loading={loading} dataSource={items} scroll={{ x: 'max-content' }} columns={[
      { title: 'Name', dataIndex: 'name', width: 160, ellipsis: true, render: (value: string) => <Text strong>{value}</Text> },
      { title: 'Encoding', dataIndex: 'code', width: 160, ellipsis: true, render: (value: string) => <Tag color="blue" style={{ maxWidth: '100%', overflow: 'hidden', textOverflow: 'ellipsis' }}>{value}</Tag> },
	  { title: 'Dynamic field', dataIndex: 'customFields', width: 90, render: (value: unknown[]) => `${value?.length ?? 0} items` },
	  { title: 'Default Priority', dataIndex: 'defaultPriority', width: 100, render: (value: string) => <Tag>{value}</Tag> },
	  { title: 'Workflows', dataIndex: 'workflowDefinitionKey', width: 200, ellipsis: true, render: (value?: string) => value || '-' },
      { title: 'SLA', dataIndex: 'slaEnabled', width: 80, render: (value: boolean) => value ? 'Bound' : '-' },
	  { title: 'Sort Order', dataIndex: 'sortOrder', width: 64 },
      { title: 'Status', dataIndex: 'status', width: 96, render: (value: string, item) => item.archivedAt ? <Tag color="default">Archived</Tag> : <Tag color={value === 'active' ? 'green' : 'default'}>{value === 'active' ? 'Enabled' : 'Disable'}</Tag> },
      { title: 'Updated At', dataIndex: 'updatedAt', width: 170, render: (value: string) => formatDateTime(value) || '-' },
      { title: 'Actions', width: 240, fixed: 'right', render: (_, item) => !canManage ? <Text type="secondary">Readonly</Text> : item.archivedAt ? <Space><Button size="small" type="primary" ghost icon={<RotateCcw className="h-3.5 w-3.5" />} onClick={() => restore(item)}>Restore</Button></Space> : <Space><Button size="small" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => { setEditing(item); setOpen(true); }}>Edit</Button><Button size="small" icon={<Copy className="h-3.5 w-3.5" />} onClick={() => clone(item)}>Copy</Button><Switch size="small" aria-label={`Switch ticket type ${item.name} enabled status`} checked={item.status === 'active'} onChange={async enabled => { try { await TicketTypeApi.setEnabled(item.id, enabled); message.success(enabled ? 'Ticket type is enabled' : 'Ticket type is deactivated'); await load(); } catch (error) { message.error(describeFailure(error, 'Operation failed', 'ticket_type:manage')); } }} /></Space> },
    ]} />
  </Card><TicketTypeFormModal visible={open} editingType={editing} onCancel={() => { setOpen(false); setEditing(null); }} onSubmit={save} /><Modal title="Work order type preset library" open={presetOpen} footer={null} onCancel={() => setPresetOpen(false)}>
    {!canInstallPreset && <Alert className="mb-3" type="warning" showIcon title="The current role can browse the preset content, but the installation requires ticket_type:install_preset. Please contact the administrator to enable it for the current role." />}
    <List dataSource={presets} renderItem={preset => <List.Item actions={canInstallPreset ? [<Button key="install" type="link" disabled={installingId !== null} onClick={async () => { setInstallingId(preset.id); try { await TicketTypeApi.installPreset(preset.id); message.success('Already installed by default'); setPresetOpen(false); await load(); } catch (error) { message.error(describeFailure(error, 'Failed to install preset', 'ticket_type:install_preset')); } finally { setInstallingId(null); } }}>Installation</Button>] : []}><List.Item.Meta title={preset.name} description={`${preset.category} · ${preset.description}`} /></List.Item>} />
  </Modal></div>;
}
