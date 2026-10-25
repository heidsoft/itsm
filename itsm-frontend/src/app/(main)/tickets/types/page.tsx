'use client';

import { useCallback, useEffect, useState } from 'react';
import { Alert, App, Button, Card, Input, List, Modal, Space, Switch, Table, Tag, Typography } from 'antd';
import { Copy, Pencil, Plus, RotateCcw } from 'lucide-react';
import { TicketTypeFormModal } from '@/components/business/TicketTypeFormModal';
import { TicketTypeApi } from '@/lib/api/ticketTypeApi';
import { isPermissionDeniedError, serverErrorMessage } from '@/lib/api/http-client';
import { usePermissions } from '@/lib/hooks/use-permissions';
import type { CreateTicketTypeRequest, TicketTypeDefinition, TicketTypePresetDefinition, UpdateTicketTypeRequest } from '@/types/ticket-type';

const { Title, Text } = Typography;

// 后端路由按 ticket_type 细分动作码守卫，403 的 message 只有「权限不足」。
// 这里把需要的码点名出来，使用者才知道该申请哪一项，而不是去改筛选条件。
function describeFailure(error: unknown, fallback: string, permission: string): string {
  const detail = serverErrorMessage(error, fallback);
  return isPermissionDeniedError(error)
    ? `${detail}。该操作需要 ${permission}，请联系管理员为当前角色开放`
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
      message.error(describeFailure(error, '加载工单类型失败', 'ticket:read'));
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
      message.error(describeFailure(error, '保存工单类型失败', 'ticket_type:manage'));
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
          message.error(describeFailure(error, '复制工单类型失败', 'ticket_type:manage'));
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
      message.error(describeFailure(error, '加载预设库失败', 'ticket_type:read'));
    }
  };

  const restore = (item: TicketTypeDefinition) => {
    modal.confirm({
      title: '恢复工单类型',
      content: `确定恢复已归档的类型「${item.name}」吗？恢复后将重新启用。`,
      okText: '恢复',
      onOk: async () => {
        try {
          await TicketTypeApi.restore(item.id);
          message.success('工单类型已恢复');
          await load();
        } catch (error) {
          message.error(describeFailure(error, '恢复工单类型失败', 'ticket_type:manage'));
        }
      },
    });
  };

  return <div className="p-6"><Card>
    <div className="mb-5 flex items-start justify-between gap-4"><div><Title level={2} className="!mb-1">工单类型</Title><Text type="secondary">租户级类型、动态表单及流程策略配置</Text></div><Space><Button onClick={() => void openPresets()}>从预设安装</Button>{canManage && <Button type="primary" icon={<Plus className="h-4 w-4" />} onClick={() => { setEditing(null); setOpen(true); }}>新建类型</Button>}</Space></div>
    <Alert
      className="mb-4"
      type={canManage ? 'info' : 'warning'}
      showIcon
      title={canManage
        ? '绑定工作流、SLA、分配规则和审批层级后启用，建议创建测试工单验收'
        : '当前角色只有查看权限，新建、编辑、复制、启停和恢复需要 ticket_type:manage，预设安装需要 ticket_type:install_preset'}
    />
    <Input.Search className="mb-4 max-w-sm" allowClear placeholder="搜索名称或编码" onSearch={setKeyword} />
    <div className="mb-3"><label className="inline-flex cursor-pointer items-center gap-2 text-sm text-gray-600"><input type="checkbox" checked={includeArchived} onChange={e => setIncludeArchived(e.target.checked)} />显示已归档类型</label></div>
    {/* 每列显式设置宽度并对长文本列开启 ellipsis，避免列宽被均分导致字段拥挤；总宽超出容器时通过 scroll.x 横向滚动兜底 */}
    <Table rowKey="id" loading={loading} dataSource={items} scroll={{ x: 'max-content' }} columns={[
      { title: '名称', dataIndex: 'name', width: 160, ellipsis: true, render: (value: string) => <Text strong>{value}</Text> },
      { title: '编码', dataIndex: 'code', width: 160, ellipsis: true, render: (value: string) => <Tag color="blue" style={{ maxWidth: '100%', overflow: 'hidden', textOverflow: 'ellipsis' }}>{value}</Tag> },
	  { title: '动态字段', dataIndex: 'customFields', width: 90, render: (value: unknown[]) => `${value?.length ?? 0} 个` },
	  { title: '默认优先级', dataIndex: 'defaultPriority', width: 100, render: (value: string) => <Tag>{value}</Tag> },
	  { title: '工作流', dataIndex: 'workflowDefinitionKey', width: 200, ellipsis: true, render: (value?: string) => value || '-' },
      { title: 'SLA', dataIndex: 'slaEnabled', width: 80, render: (value: boolean) => value ? '已绑定' : '-' },
	  { title: '排序', dataIndex: 'sortOrder', width: 64 },
      { title: '状态', dataIndex: 'status', width: 96, render: (value: string, item) => item.archivedAt ? <Tag color="default">已归档</Tag> : <Tag color={value === 'active' ? 'green' : 'default'}>{value === 'active' ? '启用' : '停用'}</Tag> },
      { title: '更新时间', dataIndex: 'updatedAt', width: 170, render: (value: string) => value ? new Date(value).toLocaleString() : '-' },
      { title: '操作', width: 240, fixed: 'right', render: (_, item) => !canManage ? <Text type="secondary">只读</Text> : item.archivedAt ? <Space><Button size="small" type="primary" ghost icon={<RotateCcw className="h-3.5 w-3.5" />} onClick={() => restore(item)}>恢复</Button></Space> : <Space><Button size="small" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => { setEditing(item); setOpen(true); }}>编辑</Button><Button size="small" icon={<Copy className="h-3.5 w-3.5" />} onClick={() => clone(item)}>复制</Button><Switch size="small" aria-label={`切换工单类型 ${item.name} 的启用状态`} checked={item.status === 'active'} onChange={async enabled => { try { await TicketTypeApi.setEnabled(item.id, enabled); message.success(enabled ? '工单类型已启用' : '工单类型已停用'); await load(); } catch (error) { message.error(describeFailure(error, '操作失败', 'ticket_type:manage')); } }} /></Space> },
    ]} />
  </Card><TicketTypeFormModal visible={open} editingType={editing} onCancel={() => { setOpen(false); setEditing(null); }} onSubmit={save} /><Modal title="工单类型预设库" open={presetOpen} footer={null} onCancel={() => setPresetOpen(false)}>
    {!canInstallPreset && <Alert className="mb-3" type="warning" showIcon title="当前角色可以浏览预设内容，但安装需要 ticket_type:install_preset，请联系管理员为当前角色开放" />}
    <List dataSource={presets} renderItem={preset => <List.Item actions={canInstallPreset ? [<Button key="install" type="link" disabled={installingId !== null} onClick={async () => { setInstallingId(preset.id); try { await TicketTypeApi.installPreset(preset.id); message.success('预设已安装'); setPresetOpen(false); await load(); } catch (error) { message.error(describeFailure(error, '安装预设失败', 'ticket_type:install_preset')); } finally { setInstallingId(null); } }}>安装</Button>] : []}><List.Item.Meta title={preset.name} description={`${preset.category} · ${preset.description}`} /></List.Item>} />
  </Modal></div>;
}
