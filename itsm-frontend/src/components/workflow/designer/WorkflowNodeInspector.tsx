// 工作流节点属性检查器
// Workflow Node Inspector - 监听 BPMN 画布选中节点并编辑其属性

'use client';

import React, { useEffect, useState } from 'react';
import { Card, Empty, Select, Input, Tag, Typography, Space, Divider, Alert, Button } from 'antd';
import { User, Users, UserCheck, Hash, Tag as TagIcon, RefreshCw } from 'lucide-react';
import { GroupAPI, type Group } from '@/lib/api/group-api';
import { UserApi, type User as ApiUser } from '@/lib/api/user-api';
import { RoleAPI } from '@/lib/api/role-api';
import { httpClient } from '@/lib/api/http-client';
import type { BpmnNodeSelection } from '../BPMNDesigner';

const { Text } = Typography;

export interface WorkflowNodeInspectorProps {
  selection: BpmnNodeSelection | null;
  onUpdateProperties: (elementId: string, properties: Record<string, unknown>) => boolean;
  onRefresh?: () => void;
}

/**
 * 从 candidateGroups 字符串解析为组名列表（用逗号分隔）
 */
function parseCsv(value: string | undefined): string[] {
  if (!value) return [];
  return value
    .split(',')
    .map(s => s.trim())
    .filter(Boolean);
}

/**
 * 数组序列化为逗号分隔字符串（写入 BPMN XML 的 candidateUsers / candidateGroups 属性）
 */
function toCsv(values: string[]): string {
  return values
    .map(s => (s || '').trim())
    .filter(Boolean)
    .join(',');
}

/**
 * 工作流节点属性面板。
 * - 选中 userTask 时显示：assignee / candidateUsers / candidateGroups / priority / formKey
 * - 选中其他节点时显示通用属性（id / type）
 * - 选中空白时不渲染
 */
export default function WorkflowNodeInspector({
  selection,
  onUpdateProperties,
  onRefresh,
}: WorkflowNodeInspectorProps) {
  const [users, setUsers] = useState<ApiUser[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [roles, setRoles] = useState<{ id: number; name: string; code: string }[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [loadingGroups, setLoadingGroups] = useState(false);
  const [loadingRoles, setLoadingRoles] = useState(false);

  // 加载候选数据
  useEffect(() => {
    let cancelled = false;
    const loadUsers = async () => {
      setLoadingUsers(true);
      try {
        const resp = await UserApi.getUsers({ page: 1, page_size: 200 });
        if (!cancelled) setUsers((resp.users as ApiUser[]) || []);
      } catch (err) {
        console.error('加载用户列表失败:', err);
      } finally {
        if (!cancelled) setLoadingUsers(false);
      }
    };
    const loadGroups = async () => {
      setLoadingGroups(true);
      try {
        const tenantId = httpClient.getTenantId() || 1;
        const resp = await GroupAPI.getGroups({ page: 1, page_size: 200, tenant_id: tenantId });
        if (!cancelled) setGroups(resp.groups || []);
      } catch (err) {
        console.error('加载组列表失败:', err);
      } finally {
        if (!cancelled) setLoadingGroups(false);
      }
    };
    const loadRoles = async () => {
      setLoadingRoles(true);
      try {
        const resp = (await RoleAPI.getRoles()) as any;
        const list = (resp.roles || resp.data || []) as { id: number; name: string; code: string }[];
        if (!cancelled) setRoles(list);
      } catch (err) {
        console.error('加载角色列表失败:', err);
      } finally {
        if (!cancelled) setLoadingRoles(false);
      }
    };
    loadUsers();
    loadGroups();
    loadRoles();
    return () => {
      cancelled = true;
    };
  }, []);

  // 未选中时
  if (!selection) {
    return (
      <Card
        title={
          <Space>
            <TagIcon className="w-4 h-4" />
            <span>节点属性</span>
          </Space>
        }
        className="h-full rounded-lg shadow-sm border border-gray-200"
        extra={
          onRefresh && (
            <Button
              type="text"
              size="small"
              icon={<RefreshCw className="w-3 h-3" />}
              onClick={onRefresh}
            >
              重读
            </Button>
          )
        }
      >
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <span className="text-xs text-gray-500">
              点击画布上的节点查看/编辑属性
            </span>
          }
        />
      </Card>
    );
  }

  const isUserTask = selection.type === 'bpmn:UserTask' || selection.type === 'UserTask';
  const bo = (selection.businessObject || {}) as Record<string, unknown>;

  // 当前值
  const currentAssignee = (bo.assignee as string) || '';
  const currentCandidateUsers = parseCsv(bo.candidateUsers as string | undefined);
  const currentCandidateGroups = parseCsv(bo.candidateGroups as string | undefined);
  const currentPriority = (bo.priority as string) || '';
  const currentFormKey = (bo.formKey as string) || '';
  const currentDueDate = (bo.dueDate as string) || '';

  // 组名选项（候选组用的是组名 group.name，对应后端 SetCandidateGroups）
  const groupOptions = groups.map(g => ({
    label: g.name,
    value: g.name,
  }));

  // 用户选项（候选用户用 username，对应后端 lib-bpmn-engine candidateUsers 解析方式）
  const userOptions = users.map(u => ({
    label: u.name || u.username || `User#${u.id}`,
    value: u.username,
  }));

  // 应用修改
  const apply = (patch: Record<string, unknown>) => {
    onUpdateProperties(selection.id, patch);
  };

  return (
    <Card
      title={
        <Space>
          <TagIcon className="w-4 h-4" />
          <span>节点属性</span>
          <Tag color={isUserTask ? 'blue' : 'default'} className="ml-1">
            {selection.type.replace('bpmn:', '')}
          </Tag>
        </Space>
      }
      className="h-full rounded-lg shadow-sm border border-gray-200"
      extra={
        onRefresh && (
          <Button
            type="text"
            size="small"
            icon={<RefreshCw className="w-3 h-3" />}
            onClick={onRefresh}
          >
            重读
          </Button>
        )
      }
      size="small"
    >
      <div className="space-y-4">
        {/* 基础信息 */}
        <div>
          <Text type="secondary" className="text-xs">
            节点 ID
          </Text>
          <div className="font-mono text-xs mt-1 px-2 py-1 bg-gray-50 rounded">
            {selection.id}
          </div>
          {selection.name && (
            <div className="mt-2">
              <Text type="secondary" className="text-xs">
                名称
              </Text>
              <div className="text-sm mt-1">{selection.name}</div>
            </div>
          )}
        </div>

        {isUserTask ? (
          <>
            <Divider className="my-2" />

            {/* Assignee */}
            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <UserCheck className="w-3.5 h-3.5 mr-1" />
                受理人 (assignee)
              </Text>
              <Select
                allowClear
                showSearch
                placeholder="选择受理人（单一用户）"
                value={currentAssignee || undefined}
                onChange={value => apply({ assignee: value || '' })}
                className="w-full"
                loading={loadingUsers}
                filterOption={(input, option) =>
                  (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                }
                options={userOptions}
              />
            </div>

            {/* Candidate Users */}
            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <User className="w-3.5 h-3.5 mr-1" />
                候选人 (candidateUsers)
              </Text>
              <Select
                mode="multiple"
                placeholder="选择候选人（多选）"
                value={currentCandidateUsers}
                onChange={values => apply({ candidateUsers: toCsv(values) })}
                className="w-full"
                loading={loadingUsers}
                maxTagCount="responsive"
                filterOption={(input, option) =>
                  (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                }
                options={userOptions}
              />
              <Text type="secondary" className="text-xs mt-1 block">
                任一候选人审批即通过该节点
              </Text>
            </div>

            {/* Candidate Groups — 核心审批组入口 */}
            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Users className="w-3.5 h-3.5 mr-1" />
                候选组 (candidateGroups)
              </Text>
              <Select
                mode="multiple"
                placeholder="选择审批组（多选）"
                value={currentCandidateGroups}
                onChange={values => apply({ candidateGroups: toCsv(values) })}
                className="w-full"
                loading={loadingGroups}
                maxTagCount="responsive"
                notFoundContent={
                  loadingGroups ? (
                    <span className="text-gray-400">加载中...</span>
                  ) : (
                    <Empty
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                      description={
                        <span className="text-xs">
                          暂无审批组，请先到{' '}
                          <a href="/admin/groups" target="_blank" rel="noreferrer">
                            组管理
                          </a>{' '}
                          创建
                        </span>
                      }
                    />
                  )
                }
                filterOption={(input, option) =>
                  (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                }
                options={groupOptions}
              />
              <Alert
                type="info"
                showIcon
                className="mt-2 text-xs"
                message="提示：审批组中任一成员审批即视为该节点通过；保存后写入 BPMN XML 的 candidateGroups 属性，后端引擎据此分配任务。"
              />
            </div>

            <Divider className="my-2" />

            {/* 表单键与优先级 */}
            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Hash className="w-3.5 h-3.5 mr-1" />
                表单键 (formKey)
              </Text>
              <Input
                placeholder="例如：approve_form_v1"
                value={currentFormKey}
                onChange={e => apply({ formKey: e.target.value })}
                allowClear
              />
            </div>

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Hash className="w-3.5 h-3.5 mr-1" />
                优先级 (priority)
              </Text>
              <Input
                type="number"
                placeholder="例如：50（数值越高越优先）"
                value={currentPriority}
                onChange={e => apply({ priority: e.target.value })}
                allowClear
              />
            </div>

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Hash className="w-3.5 h-3.5 mr-1" />
                截止时间 (dueDate)
              </Text>
              <Input
                placeholder="ISO 8601 或 P0Y0M0DT2H0M0S 格式"
                value={currentDueDate}
                onChange={e => apply({ dueDate: e.target.value })}
                allowClear
              />
            </div>
          </>
        ) : (
          <Alert
            type="info"
            showIcon
            className="text-xs"
            message={
              <span>
                <strong>{selection.type.replace('bpmn:', '')}</strong> 节点的属性编辑请直接在画布上操作（拖拽、连线等）。
              </span>
            }
          />
        )}
      </div>
    </Card>
  );
}
