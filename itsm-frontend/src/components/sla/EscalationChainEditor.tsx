'use client';

/**
 * 升级链编辑器：SLA 超时升级规则的「按优先级逐级上报」配置。
 *
 * 数据契约（handlers/escalation_matrix）：
 * - EscalationMatrix = 优先级 → 升级级别数组（Record<string, EscalationLevel[]>）
 * - PUT /escalation-matrices 全量覆盖；保存成功即已持久化（system_configs）
 * - notifyRoles 必须取主角色词表（domain/role.All，按 users.role 精确匹配查人）
 * - afterMinutes 是距告警开始的累计分钟数，不是距上一级的间隔
 */

import React, { useEffect, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import { ArrowUpCircle, Clock, Minus, Plus } from 'lucide-react';
import {
  EscalationMatrixApi,
  DEFAULT_ESCALATION_MATRIX,
  type EscalationLevel,
  type EscalationMatrix,
} from '@/lib/api/escalation-matrix-api';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { PERM } from '@/lib/permissions';

const { Text } = Typography;

const PRIORITIES = ['critical', 'high', 'medium', 'low'] as const;

const priorityLabelMap: Record<string, string> = {
  critical: '致命',
  high: '高',
  medium: '中',
  low: '低',
};

const priorityColorMap: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'gold',
  low: 'blue',
};

// 与 domain/role.All 主角色词表对齐（升级按 users.role 精确匹配查人）
const roleOptions = [
  { value: 'manager', label: '部门经理（manager）' },
  { value: 'it_admin', label: 'IT 管理员（it_admin）' },
  { value: 'sys_admin', label: '系统管理员（sys_admin）' },
  { value: 'security_admin', label: '安全管理员（security_admin）' },
  { value: 'audit_admin', label: '审计管理员（audit_admin）' },
  { value: 'admin', label: '租户管理员（admin）' },
];

function formatMinutes(mins: number): string {
  if (mins < 60) return `${mins} 分钟`;
  if (mins < 1440) return `${(mins / 60).toFixed(1)} 小时`;
  return `${(mins / 1440).toFixed(1)} 天`;
}

const normalizeLevel = (level: Partial<EscalationLevel>): EscalationLevel => ({
  level: level.level ?? 1,
  afterMinutes: level.afterMinutes ?? 30,
  notifyRoles: level.notifyRoles ?? [],
  notifyUserIDs: level.notifyUserIDs ?? [],
  description: level.description ?? '',
});

/** 把服务端矩阵补齐成四优先级完备的编辑态（缺失优先级回填默认） */
function toEditable(server: EscalationMatrix): EscalationMatrix {
  const editable: EscalationMatrix = {};
  PRIORITIES.forEach(p => {
    editable[p] = (server[p] ?? DEFAULT_ESCALATION_MATRIX[p] ?? []).map(normalizeLevel);
  });
  // 服务端存在词表外优先级键时保留，避免保存时静默丢配置
  Object.keys(server).forEach(p => {
    if (!(p in editable)) editable[p] = server[p].map(normalizeLevel);
  });
  return editable;
}

interface EditorRow {
  priority: string;
  level: number;
  afterMinutes: number;
  notifyRoles: string[];
  notifyUserIDs: number[];
  description: string;
}

export default function EscalationChainEditor() {
  const { message } = App.useApp();
  const { hasPermission } = usePermissions();
  const canEdit = hasPermission(PERM.sla.write);

  const [matrix, setMatrix] = useState<EscalationMatrix>({});
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setLoadError(null);
    EscalationMatrixApi.getMatrix()
      .then(m => {
        if (alive) setMatrix(toEditable(m));
      })
      .catch((err: Error) => {
        if (!alive) return;
        setMatrix(toEditable(DEFAULT_ESCALATION_MATRIX));
        setLoadError(err.message || '升级链加载失败');
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, []);

  const rows: EditorRow[] = [];
  Object.entries(matrix).forEach(([priority, levels]) => {
    levels.forEach(l => rows.push({ priority, ...l }));
  });

  const mutatePriority = (priority: string, updater: (levels: EscalationLevel[]) => EscalationLevel[]) => {
    setMatrix(prev => ({ ...prev, [priority]: updater(prev[priority] ?? []) }));
    setDirty(true);
  };

  const addLevel = (priority: string) => {
    mutatePriority(priority, levels => {
      const nextLevel = levels.length + 1;
      const prevMinutes = levels.length > 0 ? levels[levels.length - 1].afterMinutes : 0;
      return [...levels, {
        level: nextLevel,
        afterMinutes: prevMinutes + 30,
        notifyRoles: [],
        notifyUserIDs: [],
        description: '',
      }];
    });
  };

  const removeLevel = (priority: string, level: number) => {
    mutatePriority(priority, levels => levels
      .filter(l => l.level !== level)
      .map((l, i) => ({ ...l, level: i + 1 })));
  };

  const updateLevel = (priority: string, level: number, patch: Partial<EscalationLevel>) => {
    mutatePriority(priority, levels => levels.map(l => (l.level === level ? { ...l, ...patch } : l)));
  };

  const handleSave = async () => {
    if (saving) return;
    // 前端先做与运行时一致的硬校验：级别无通知目标 = 永远空转的升级级
    const invalid: string[] = [];
    Object.entries(matrix).forEach(([priority, levels]) => {
      levels.forEach(l => {
        if (l.notifyRoles.length === 0 && l.notifyUserIDs.length === 0) {
          invalid.push(`${priorityLabelMap[priority] ?? priority} L${l.level}`);
        }
      });
    });
    if (invalid.length > 0) {
      message.warning(`以下级别未配置任何通知对象，升级时不会通知任何人：${invalid.join('、')}`);
      return;
    }
    setSaving(true);
    try {
      await EscalationMatrixApi.setMatrix(matrix);
      message.success('升级链已保存');
      setDirty(false);
      const fresh = await EscalationMatrixApi.getMatrix();
      setMatrix(toEditable(fresh));
    } catch (error) {
      console.error('Failed to save escalation matrix:', error);
      message.error(error instanceof Error ? error.message : '保存升级链失败');
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      onCell: (_: unknown, index?: number) => {
        // 同优先级多行只在第一行显示标签（rowSpan 合并）
        const row = rows[index ?? 0];
        const firstIdx = rows.findIndex(r => r.priority === row.priority);
        const span = rows.filter(r => r.priority === row.priority).length;
        return { rowSpan: index === firstIdx ? span : 0 };
      },
      render: (v: string) => (
        <Tag color={priorityColorMap[v] ?? 'default'}>
          {priorityLabelMap[v] ? `${priorityLabelMap[v]}` : v}
        </Tag>
      ),
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 70,
      render: (v: number) => <Tag color="blue">L{v}</Tag>,
    },
    {
      title: '超时后（累计分钟）',
      dataIndex: 'afterMinutes',
      key: 'afterMinutes',
      width: 170,
      render: (v: number, record: EditorRow) => canEdit ? (
        <Space direction="vertical" size={0}>
          <InputNumber
            min={1}
            max={43200}
            value={v}
            onChange={val => updateLevel(record.priority, record.level, { afterMinutes: val ?? 1 })}
            addonAfter={<Clock size={12} />}
            style={{ width: '100%' }}
            aria-label={`升级到 L${record.level} 的累计分钟数`}
          />
          <Text type="secondary" style={{ fontSize: 12 }}>{formatMinutes(v)}</Text>
        </Space>
      ) : (
        <Space><Clock size={14} /><Text strong>{formatMinutes(v)}</Text></Space>
      ),
    },
    {
      title: '通知角色',
      dataIndex: 'notifyRoles',
      key: 'notifyRoles',
      render: (roles: string[], record: EditorRow) => canEdit ? (
        <Select
          mode="multiple"
          allowClear
          style={{ minWidth: 260 }}
          placeholder="选择超时后通知的角色"
          value={roles}
          onChange={val => updateLevel(record.priority, record.level, { notifyRoles: val })}
          options={roleOptions}
          aria-label={`L${record.level} 通知角色`}
        />
      ) : (
        roles.length === 0
          ? <Text type="secondary">未指定</Text>
          : <Space size={4} wrap>{roles.map(r => <Tag key={r} color="geekblue">{r}</Tag>)}</Space>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 90,
      hidden: !canEdit,
      render: (_: unknown, record: EditorRow) => (
        <Popconfirm
          title={`删除 ${priorityLabelMap[record.priority] ?? record.priority} 的 L${record.level}？`}
          description="保存后生效；同优先级其余级别会自动重新编号。"
          okText="删除"
          cancelText="取消"
          okType="danger"
          onConfirm={() => removeLevel(record.priority, record.level)}
        >
          <Button size="small" danger icon={<Minus size={14} />} aria-label={`删除 L${record.level}`} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      {loadError && (
        <Alert
          title="升级链加载失败"
          description={loadError + '。当前展示的是系统内置默认，并非生产配置；保存任意修改会以当前表格为准整体覆盖。'}
          type="error"
          showIcon
        />
      )}

      <Card
        title="按优先级的升级链"
        extra={canEdit && (
          <Space>
            <Button
              type="primary"
              icon={<ArrowUpCircle size={14} />}
              loading={saving}
              disabled={!dirty}
              onClick={handleSave}
            >
              保存{dirty ? '' : '（无改动）'}
            </Button>
          </Space>
        )}
      >
        <Table
          rowKey={r => `${r.priority}-${r.level}`}
          loading={loading}
          dataSource={rows}
          columns={columns}
          pagination={false}
          size="middle"
        />
        {canEdit && (
          <Space wrap style={{ marginTop: 16 }}>
            {Object.keys(matrix).map(priority => (
              <Button
                key={priority}
                size="small"
                icon={<Plus size={14} />}
                onClick={() => addLevel(priority)}
              >
                给{priorityLabelMap[priority] ?? priority}加一级
              </Button>
            ))}
          </Space>
        )}
      </Card>
    </div>
  );
}
