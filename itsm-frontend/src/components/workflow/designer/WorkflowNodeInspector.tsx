// 工作流节点属性检查器
// Workflow Node Inspector - 监听 BPMN 画布选中节点并编辑其属性

'use client';

import React, { useEffect, useState, useRef, useCallback } from 'react';
import { Card, Empty, Select, Input, Tag, Typography, Space, Divider, Alert, Button, Switch, Tooltip, Collapse, Badge } from 'antd';
import {
  User, Users, UserCheck, Hash, Tag as TagIcon, RefreshCw,
  Code, Server, GitBranch, PlayCircle, Clock, FileText,
  Settings, AlertTriangle,
  Database, Link, Timer, MessageCircle, Radio, ChevronDown, ChevronRight,
  Save, Undo, Redo, Info, Zap, Shield, Bell
} from 'lucide-react';
import { GroupAPI, type Group } from '@/lib/api/group-api';

// bpmn-js 事件定义最小类型（库本身类型较松）
interface BpmnEventDefinition {
  $type: string;
}
import { UserApi, type User as ApiUser } from '@/lib/api/user-api';
import { RoleAPI } from '@/lib/api/role-api';
import { httpClient } from '@/lib/api/http-client';
import type { BpmnNodeSelection } from '../BPMNDesigner';
import {
  readConditionExpressionText,
  readTimerExpressionText,
  hasTimerDefinition,
  readDocumentationText,
  readReferenceId,
} from '../bpmnPropertyWrite';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';

const { Text } = Typography;
const { TextArea } = Input;

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
 * 防抖提交的受控输入。
 * 逐键 updateProperties 会触发命令栈变化 + 全量 saveXML 序列化，导致输入卡顿、
 * 命令栈被逐键污染（撤销粒度变成单个字符）。这里改为：
 * - 输入时仅更新本地 state（300ms 防抖后提交）
 * - 失焦时立即提交未保存的值
 * - 仅当外部值变化（选中节点切换 / 画布同步）时才重置本地值
 */
function useDebouncedCommit(
  externalValue: string,
  onCommit: (value: string) => void,
  delay = 300
) {
  const [localValue, setLocalValue] = useState(externalValue);
  const lastCommittedRef = useRef(externalValue);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (externalValue !== lastCommittedRef.current) {
      lastCommittedRef.current = externalValue;
      setLocalValue(externalValue);
    }
  }, [externalValue]);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const commit = useCallback(
    (value: string) => {
      lastCommittedRef.current = value;
      onCommit(value);
    },
    [onCommit]
  );

  const handleChange = useCallback(
    (value: string) => {
      setLocalValue(value);
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => {
        timerRef.current = null;
        commit(value);
      }, delay);
    },
    [commit, delay]
  );

  const handleBlur = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    if (localValue !== lastCommittedRef.current) {
      commit(localValue);
    }
  }, [localValue, commit]);

  return { value: localValue, onChange: handleChange, onBlur: handleBlur };
}

interface DebouncedInputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange' | 'onBlur' | 'value' | 'size'> {
  value: string;
  onCommit: (value: string) => void;
  delay?: number;
  size?: 'small' | 'middle' | 'large';
  addonBefore?: React.ReactNode;
  allowClear?: boolean;
}

function DebouncedInput({ value, onCommit, delay, size, addonBefore, allowClear, ...rest }: DebouncedInputProps) {
  const { value: localValue, onChange, onBlur } = useDebouncedCommit(value, onCommit, delay);
  return (
    <Input
      {...rest}
      size={size}
      addonBefore={addonBefore}
      allowClear={allowClear}
      value={localValue}
      onChange={e => onChange(e.target.value)}
      onBlur={onBlur}
    />
  );
}

interface DebouncedTextAreaProps {
  value: string;
  onCommit: (value: string) => void;
  delay?: number;
  rows?: number;
  size?: 'small' | 'middle' | 'large';
  placeholder?: string;
  className?: string;
}

function DebouncedTextArea({ value, onCommit, delay, rows, size, placeholder, className }: DebouncedTextAreaProps) {
  const { value: localValue, onChange, onBlur } = useDebouncedCommit(value, onCommit, delay);
  return (
    <TextArea
      rows={rows}
      size={size}
      placeholder={placeholder}
      className={className}
      value={localValue}
      onChange={e => onChange(e.target.value)}
      onBlur={onBlur}
    />
  );
}

/**
 * 工作流节点属性面板。
 * - 支持多种BPMN节点类型的属性可视化配置
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
  // 搜索防抖状态
  const [userSearchText, setUserSearchText] = useState('');
  const [groupSearchText, setGroupSearchText] = useState('');
  // 搜索防抖计时器
  const userSearchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const groupSearchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // 共享取消标记（跨 useEffect）
  const cancelledRef = useRef(false);

  // 搜索防抖加载用户（300ms）
  const loadUsers = useCallback(async (search = '') => {
    setLoadingUsers(true);
    try {
      const resp = await UserApi.getUsers({ page: 1, pageSize: BACKEND_MAX_PAGE_SIZE, search });
      if (!cancelledRef.current) {
        setUsers(prev => {
          const existingIds = new Set(prev.map(u => u.id));
          const incoming = (resp.items as ApiUser[]).filter(u => !existingIds.has(u.id));
          return [...prev, ...incoming];
        });
      }
    } catch (err) {
      console.error('Failed to load user list:', err);
    } finally {
      if (!cancelledRef.current) setLoadingUsers(false);
    }
  }, []);

  // 搜索防抖加载组（300ms）
  const loadGroups = useCallback(async (search = '') => {
    setLoadingGroups(true);
    try {
      const tenantId = httpClient.getTenantId();
      if (!tenantId) throw new Error('Missing valid tenant context');
      const resp = await GroupAPI.getGroups({ page: 1, pageSize: 100, tenantId, search });
      if (!cancelledRef.current) {
        setGroups(prev => {
          const existingIds = new Set(prev.map(g => g.id));
          const incoming = (resp.items || []).filter(g => !existingIds.has(g.id));
          return [...prev, ...incoming];
        });
      }
    } catch (err) {
      console.error('Failed to load group list:', err);
    } finally {
      if (!cancelledRef.current) setLoadingGroups(false);
    }
  }, []);

  // 加载角色（仅一次）
  useEffect(() => {
    let cancelled = false;
    const loadRoles = async () => {
      setLoadingRoles(true);
      try {
        const resp = await RoleAPI.getRoles({ page: 1, pageSize: 100 });
        const list = resp.items.map(({ id, name, code }) => ({ id, name, code: code ?? '' }));
        if (!cancelled) setRoles(list);
      } catch (err) {
        console.error('Failed to load character list:', err);
      } finally {
        if (!cancelled) setLoadingRoles(false);
      }
    };
    loadRoles();
    return () => {
      cancelled = true;
    };
  }, []);

  // 初始加载用户和组（仅一次）
  useEffect(() => {
    let cancelled = false;
    const doLoad = async () => {
      setLoadingUsers(true);
      setLoadingGroups(true);
      try {
        const tenantId = httpClient.getTenantId();
        if (!tenantId) throw new Error('Missing valid tenant context');
        const [userResp, groupResp] = await Promise.all([
          UserApi.getUsers({ page: 1, pageSize: BACKEND_MAX_PAGE_SIZE, search: '' }),
          GroupAPI.getGroups({ page: 1, pageSize: 100, tenantId, search: '' }),
        ]);
        if (!cancelled) {
          setUsers((userResp.items as ApiUser[]) || []);
          setGroups(groupResp.items || []);
        }
      } catch (err) {
        console.error('Failed to load user/group list:', err);
      } finally {
        if (!cancelled) {
          setLoadingUsers(false);
          setLoadingGroups(false);
        }
      }
    };
    doLoad();
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
            <span>Properties</span>
          </Space>
        }
        className="h-full rounded-lg shadow-sm border border-gray-200"
        extra={
          onRefresh && (
            <Tooltip title="Refresh selected node attributes">
              <Button
                type="text"
                size="small"
                icon={<RefreshCw className="w-3 h-3" />}
                aria-label="Refresh node attributes"
                onClick={onRefresh}
              />
            </Tooltip>
          )
        }
      >
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <Space orientation="vertical" size={0} align="center">
              <span className="text-xs text-gray-500">

                Click on a node on the canvas to view/edit properties
              </span>
              <Text type="secondary" className="text-xs">

                Supports dragging nodes, connecting lines, and conditional configuration
              </Text>
            </Space>
          }
        />
        {/* 快捷操作提示 */}
        <div className="mt-4 p-3 bg-gray-50 rounded-lg">
          <Text strong className="text-xs block mb-2">💡 Quick operation</Text>
          <Space orientation="vertical" size={2} className="w-full">
            <Text type="secondary" className="text-xs">• Double-click a node to quickly edit its name</Text>
            <Text type="secondary" className="text-xs">• Click on the connection line to set flow conditions</Text>
            <Text type="secondary" className="text-xs">• Drag the left/right side of a node to create a new process</Text>
          </Space>
        </div>
      </Card>
    );
  }

  // 节点类型判断
  const bo = (selection.businessObject || {}) as Record<string, unknown>;
  const nodeType = selection.type.replace('bpmn:', '');
  const isUserTask = nodeType === 'UserTask';
  const isServiceTask = nodeType === 'ServiceTask';
  const isScriptTask = nodeType === 'ScriptTask';
  const isBusinessRuleTask = nodeType === 'BusinessRuleTask';
  const isSendTask = nodeType === 'SendTask';
  const isReceiveTask = nodeType === 'ReceiveTask';
  const isCCTask =
    nodeType === 'ServiceTask' &&
    ((bo.implementation as string) === 'cc_handler' ||
      readReferenceId(bo, 'operationRef') === 'cc_handler');
  const isExclusiveGateway = nodeType === 'ExclusiveGateway';
  const isInclusiveGateway = nodeType === 'InclusiveGateway';
  const isParallelGateway = nodeType === 'ParallelGateway';
  const isEventBasedGateway = nodeType === 'EventBasedGateway';
  const isComplexGateway = nodeType === 'ComplexGateway';
  const isSequenceFlow = nodeType === 'SequenceFlow';
  const isStartEvent = nodeType === 'StartEvent';
  const isEndEvent = nodeType === 'EndEvent';
  const isIntermediateCatchEvent = nodeType === 'IntermediateCatchEvent';
  const isIntermediateThrowEvent = nodeType === 'IntermediateThrowEvent';
  const isBoundaryEvent = nodeType === 'BoundaryEvent';
  const isSubProcess = nodeType === 'SubProcess' || nodeType === 'Transaction';
  const isCallActivity = nodeType === 'CallActivity';

  // 事件类型判断
  const isTimerEvent = selection.businessObject?.eventDefinitionType === 'timer' ||
    (selection.businessObject?.eventDefinitions as BpmnEventDefinition[])?.some(d => d.$type === 'bpmn:TimerEventDefinition');
  const isMessageEvent = selection.businessObject?.eventDefinitionType === 'message' ||
    (selection.businessObject?.eventDefinitions as BpmnEventDefinition[])?.some(d => d.$type === 'bpmn:MessageEventDefinition');
  const isSignalEvent = selection.businessObject?.eventDefinitionType === 'signal' ||
    (selection.businessObject?.eventDefinitions as BpmnEventDefinition[])?.some(d => d.$type === 'bpmn:SignalEventDefinition');
  const isErrorEvent = selection.businessObject?.eventDefinitionType === 'error' ||
    (selection.businessObject?.eventDefinitions as BpmnEventDefinition[])?.some(d => d.$type === 'bpmn:ErrorEventDefinition');
  const isEscalationEvent = selection.businessObject?.eventDefinitionType === 'escalation' ||
    (selection.businessObject?.eventDefinitions as BpmnEventDefinition[])?.some(d => d.$type === 'bpmn:EscalationEventDefinition');
  const isConditionalEvent = selection.businessObject?.eventDefinitionType === 'conditional' ||
    (selection.businessObject?.eventDefinitions as BpmnEventDefinition[])?.some(d => d.$type === 'bpmn:ConditionalEventDefinition');
  const isTerminateEvent = (selection.businessObject?.eventDefinitions as BpmnEventDefinition[])?.some(d => d.$type === 'bpmn:TerminateEventDefinition');
  const isCancelEvent = (selection.businessObject?.eventDefinitions as BpmnEventDefinition[])?.some(d => d.$type === 'bpmn:CancelEventDefinition');

  // 边界/开始事件仅在未挂接其它互斥事件定义时提供「定时」开关，
  // 开关状态必须回读真实 TimerEventDefinition，否则关闭后无法再次生效。
  const hasNonTimerEventDefinition = Boolean(
    isMessageEvent || isSignalEvent || isErrorEvent || isEscalationEvent ||
    isConditionalEvent || isTerminateEvent || isCancelEvent
  );
  const currentHasTimerDefinition = hasTimerDefinition(bo);

  // 通用属性
  const currentName = (bo.name as string) || '';
  const currentDocumentation = readDocumentationText(bo);

  // 用户任务属性
  const currentAssignee = (bo.assignee as string) || '';
  const currentCandidateUsers = parseCsv(bo.candidateUsers as string | undefined);
  const currentCandidateGroups = parseCsv(bo.candidateGroups as string | undefined);
  const currentPriority = (bo.priority as string) || '';
  const currentFormKey = (bo.formKey as string) || '';
  const currentDueDate = (bo.dueDate as string) || '';
  const currentFollowUpDate = (bo.followUpDate as string) || '';
  const currentTaskPurpose = (bo.taskPurpose as string) || 'work';
  const currentApprovalMode = (bo.approvalMode as string) || 'single';
  const currentApprovalThreshold = Number(bo.approvalThreshold || 1);
  const currentRejectStrategy = (bo.rejectStrategy as string) || 'terminate';
  const currentTimeoutAction = (bo.timeoutAction as string) || 'notify';
  const currentAllowDelegate = (bo.allowDelegate as boolean) ?? false;
  const currentAllowAddApprover = (bo.allowAddApprover as boolean) ?? false;
  const currentCommentRequiredOnReject = (bo.commentRequiredOnReject as boolean) ?? true;

  // 服务任务属性
  const currentImplementation = (bo.implementation as string) || '';
  const currentResultVariable = (bo.resultVariable as string) || '';
  const currentAsync = (bo.async as boolean) || false;

  // 脚本任务属性
  const currentScript = (bo.script as string) || '';
  const currentScriptFormat = (bo.scriptFormat as string) || 'javascript';

  // 业务规则任务属性
  const currentRuleRef = (bo.ruleRef as string) || '';
  const currentRuleInput = (bo.ruleInput as string) || '';
  const currentRuleOutput = (bo.ruleOutput as string) || '';

  // 发送/接收任务属性
  const currentMessageRef = (bo.messageRef as string) || '';
  const currentOperation = (bo.operation as string) || '';

  // 抄送任务属性
  const currentCCType = (bo.ccType as string) || 'user'; // user, group, role, variable
  const currentCCUserIds = (bo.ccUserIds as string) || '';
  const currentCCGroupIds = (bo.ccGroupIds as string) || '';
  const currentCCRoleIds = (bo.ccRoleIds as string) || '';
  const currentCCVariable = (bo.ccVariable as string) || '';
  // ccNotify 在 moddle 与后端中都是字符串属性，'false' 也是 truthy，必须显式解析
  const currentCCNotify = bo.ccNotify === undefined || bo.ccNotify === null
    ? true
    : String(bo.ccNotify) !== 'false';
  const currentNotifyChannels = parseCsv((bo.notifyChannels as string) || 'in_app');


  // 网关/序列流属性
  const currentConditionExpression = readConditionExpressionText(bo);
  const currentDefaultFlow = readReferenceId(bo, 'default');

  // 事件属性：timer 表达式存于 TimerEventDefinition 子元素（顶层 bo 上不存在）
  const currentTimerDefinition = readTimerExpressionText(bo, 'timeDuration');
  const currentTimerCycle = readTimerExpressionText(bo, 'timeCycle');
  const currentTimerDate = readTimerExpressionText(bo, 'timeDate');
  // timerType 由非空表达式推导（三种互斥），不写入 moddle 属性
  const currentTimerType = readTimerExpressionText(bo, 'timeDate')
    ? 'date'
    : readTimerExpressionText(bo, 'timeCycle')
      ? 'cycle'
      : 'duration';
  const currentSignalRef = (bo.signalRef as string) || '';
  const currentErrorCode = (bo.errorCode as string) || '';
  const currentEscalationCode = (bo.escalationCode as string) || '';
  const currentCondition = (bo.condition as string) || '';

  // 边界事件属性
  const currentCancelActivity = (bo.cancelActivity as boolean) ?? true;

  // 子流程属性
  const currentTriggeredByEvent = (bo.triggeredByEvent as boolean) || false;
  const currentIsForCompensation = (bo.isForCompensation as boolean) || false;

  // 调用活动属性
  const currentCalledElement = (bo.calledElement as string) || '';
  // 未配置时默认开启（与调用活动的期望默认一致）；显式写 false 必须保持 false，
  // 不能用 `|| true` 把关闭状态吞掉。
  const currentInheritVariables =
    bo.inheritVariables === undefined ? true : Boolean(bo.inheritVariables);
  const currentInheritBusinessKey =
    bo.inheritBusinessKey === undefined ? true : Boolean(bo.inheritBusinessKey);

  // 事务子流程属性
  const currentTransactionMethod = (bo.transactionMethod as string) || 'standard';

  // 组名选项（候选组用的是组名 group.name，对应后端 SetCandidateGroups）
  // 融合角色选项：角色 code 同名时后端 ExpandGroupsToUsers 会回退按角色解析
  // （M2M 边 ∪ 主角色枚举），groups 表为空也能按角色分派任务。
  // 同名冲突时组优先（后端语义），此处去重避免出现两个相同 value。
  const groupNameSet = new Set(groups.map(g => g.name));
  const groupOptions = [
    ...groups.map(g => ({
      label: g.name,
      value: g.name,
    })),
    ...roles
      .filter(r => r.code && !groupNameSet.has(r.code))
      .map(r => ({
        label: `Role: ${r.name} (${r.code})`,
        value: r.code,
      })),
  ];

  // 用户选项（BPMN 引擎 assignee / candidateUsers 期望用户 ID 的字符串形式）
  const userOptions = users.map(u => ({
    label: u.name || u.username || `User#${u.id}`,
    value: String(u.id),
  }));

  const ccUserOptions = users.map(u => ({
    label: `${u.name || u.username || `User#${u.id}`}${u.department ? ` (${u.department})` : ''}`,
    value: String(u.id),
  }));

  const ccGroupOptions = groups.map(g => ({
    label: g.name,
    value: String(g.id),
  }));

  const ccRoleOptions = roles.map(r => ({
    label: `${r.name}${r.code ? ` (${r.code})` : ''}`,
    value: String(r.id),
  }));

  // 脚本语言选项
  const scriptFormatOptions = [
    { label: 'JavaScript', value: 'javascript' },
    { label: 'Python', value: 'python' },
    { label: 'Groovy', value: 'groovy' },
    { label: 'Lua', value: 'lua' },
    { label: 'Ruby', value: 'ruby' },
    { label: 'Java', value: 'java' }
  ];

  // 服务实现类型选项
  const implementationOptions = [
    { label: 'HTTP interface call (not ready)', value: 'http', disabled: true },
    { label: 'Java class call (not supported)', value: 'java', disabled: true },
    { label: 'expression (not ready)', value: 'expression', disabled: true },
    { label: 'Webhook', value: 'webhook' },
    { label: 'System built-in service (please select a registered handler)', value: 'generic_handler' },
    { label: 'Email sent (not ready)', value: 'mail', disabled: true },
    { label: 'Automatic carbon copy', value: 'cc_handler' }
  ];

  const notifyChannelOptions = [
    { label: 'In-App', value: 'in_app' },
    { label: 'Email', value: 'email' },
    { label: 'SMS', value: 'sms' },
    { label: 'Feishu', value: 'feishu' },
    { label: 'DingTalk', value: 'dingtalk' },
    { label: 'WeCom', value: 'wecom' },
    { label: 'Webhook', value: 'webhook' }
  ];

  // 定时类型选项
  const timerTypeOptions = [
    { label: 'Duration', value: 'duration' },
    { label: 'Periodic execution', value: 'cycle' },
    { label: 'Specified time', value: 'date' }
  ];

  // 应用修改
  const apply = (patch: Record<string, unknown>) => {
    onUpdateProperties(selection.id, patch);
  };

  // 应用条件表达式修改：形状转换由 BPMNDesigner 的 normalizeNodeProperties 负责
  const applyCondition = (value: string) => {
    apply({ conditionExpression: value });
  };

  return (
    <Card
      title={
        <Space>
          <TagIcon className="w-4 h-4" />
          <span>Properties</span>
          <Tag
            color={
              isUserTask ? 'blue' :
              isServiceTask ? 'purple' :
              isScriptTask ? 'cyan' :
              isExclusiveGateway ? 'orange' :
              isStartEvent ? 'green' :
              isEndEvent ? 'red' :
              isBoundaryEvent ? 'geekblue' :
              isSubProcess ? 'gold' :
              'default'
            }
            className="ml-1"
          >
            {nodeType}
          </Tag>
        </Space>
      }
      className="h-full rounded-lg shadow-sm border border-gray-200 overflow-y-auto"
      extra={
        onRefresh && (
          <Button
            type="text"
            size="small"
            icon={<RefreshCw className="w-3 h-3" />}
            onClick={onRefresh}
          >

            Refresh
          </Button>
        )
      }
      size="small"
    >
      <div className="space-y-4 pb-4">
        {/* 基础信息 - 所有节点通用 */}
        <div>
          <Text type="secondary" className="text-xs">

            Node ID
          </Text>
          <div className="font-mono text-xs mt-1 px-2 py-1 bg-gray-50 rounded">
            {selection.id}
          </div>

          <div className="mt-2">
            <Text type="secondary" className="text-xs">

              Node name
            </Text>
            <DebouncedInput
              value={currentName}
              onCommit={value => apply({ name: value })}
              placeholder="Enter node display name"
              size="small"
              className="mt-1"
            />
          </div>

          <div className="mt-2">
            <Text type="secondary" className="text-xs">

              Description information
            </Text>
            <DebouncedTextArea
              value={currentDocumentation}
              onCommit={value => apply({ documentation: value })}
              placeholder="Enter node function description (optional)"
              size="small"
              rows={2}
              className="mt-1"
            />
          </div>
        </div>

        {/* 用户任务配置 */}
        {isUserTask && (
          <>
            <Divider className="my-2" />

            <div className="mb-3 p-3 border border-blue-100 rounded-lg bg-blue-50/50">
              <Text strong className="text-sm flex items-center mb-2">
                <Shield className="w-3.5 h-3.5 mr-1" />Approval Semantics
              </Text>
              <Select
                value={currentTaskPurpose}
                onChange={value => apply({ taskPurpose: value })}
                options={[{ label: 'Common manual tasks', value: 'work' }, { label: 'Approval task', value: 'approval' }]}
                className="w-full" size="small"
              />
              {currentTaskPurpose === 'approval' && (
                <Space orientation="vertical" className="w-full mt-2" size="small">
                  <Select
                    value={currentApprovalMode}
                    onChange={value => apply({ approvalMode: value })}
                    options={[
                      { label: 'Single approver', value: 'single' }, { label: 'Any pass', value: 'any' },
                      { label: 'All passed', value: 'all' },
                      { label: 'Ratio/threshold passed', value: 'threshold' },
                      { label: 'Sequential countersigning', value: 'sequential' },
                    ]}
                    className="w-full" size="small"
                  />
                  {currentApprovalMode === 'threshold' && (
                    <DebouncedInput type="number" min={1} value={String(currentApprovalThreshold)}
                      onCommit={value => apply({ approvalThreshold: Number(value || 1) })}
                      addonBefore="Number of people passing" size="small" />
                  )}
                  <Select value={currentRejectStrategy} onChange={value => apply({ rejectStrategy: value })}
                    options={[
                      { label: 'Terminating process', value: 'terminate' },
                      { label: 'Return to originator', value: 'to_requester' },
                      { label: 'Entering reject branch', value: 'gateway' },
                    ]}
                    className="w-full" size="small" />
                  <Select value={currentTimeoutAction} onChange={value => apply({ timeoutAction: value })}
                    allowClear
                    placeholder="Select timeout action"
                    options={[{ label: 'Alert only', value: 'notify' }, { label: 'Upgrade Approval', value: 'escalate' }, { label: 'Automatically rejected', value: 'auto_reject' }, { label: 'Pass automatically', value: 'auto_approve' }]}
                    className="w-full" size="small" />
                  <Space wrap>
                    <Switch size="small" aria-label="Allow delegation" checked={currentAllowDelegate} onChange={v => apply({ allowDelegate: v })} />Allow delegation
                    <Switch size="small" aria-label="Allow signature" checked={currentAllowAddApprover} onChange={v => apply({ allowAddApprover: v })} />Allow signature
                    <Switch size="small" aria-label="Required for rejecting comments" checked={currentCommentRequiredOnReject} onChange={v => apply({ commentRequiredOnReject: v })} />Required for rejecting comments
                  </Space>
                </Space>
              )}
            </div>

            {/* 快捷操作栏 */}
            <div className="mb-3 p-2 bg-blue-50 rounded-lg">
              <Space wrap>
                <Text strong className="text-xs text-blue-700">⚡ Common configuration:</Text>
                <Button
                  size="small"
                  type="text"
                  onClick={() => apply({ assignee: currentAssignee || 'admin' })}
                >

                  Set as administrator
                </Button>
                <Button
                  size="small"
                  type="text"
                  disabled
                >

                  Normal priority (not ready)
                </Button>
                <Button
                  size="small"
                  type="text"
                  disabled
                >

                  24 hours timeout (not ready)
                </Button>
              </Space>
            </div>

            {/* Assignee */}
            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <UserCheck className="w-3.5 h-3.5 mr-1" />

                assignee
                <Tag color="blue" className="ml-2 text-xs">Solo</Tag>
              </Text>
              <Select
                allowClear
                showSearch
                placeholder="Select assignee (single user)"
                value={currentAssignee || undefined}
                onChange={value => apply({ assignee: value || '' })}
                className="w-full"
                loading={loadingUsers}
                filterOption={(input, option) =>
                  (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                }
                options={userOptions}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                Designate a single user as the handler for this task
              </Text>
            </div>

            {/* Candidate Users */}
            <div className="mt-3">
              <Text strong className="text-sm flex items-center mb-2">
                <User className="w-3.5 h-3.5 mr-1" />

                candidates (candidateUsers)
              </Text>
              <Select
                mode="multiple"
                placeholder="Select candidates (multiple choices)"
                value={currentCandidateUsers}
                onChange={values => apply({ candidateUsers: toCsv(values) })}
                onSearch={value => {
                  const trimmed = value.trim();
                  setUserSearchText(trimmed);
                  if (userSearchTimerRef.current) clearTimeout(userSearchTimerRef.current);
                  userSearchTimerRef.current = setTimeout(() => {
                    if (trimmed) loadUsers(trimmed);
                  }, 300);
                }}
                filterOption={false}
                className="w-full"
                loading={loadingUsers}
                maxTagCount="responsive"
                options={userOptions}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                Any candidate can handle this task
              </Text>
            </div>

            {/* Candidate Groups — 核心审批组入口 */}
            <div className="mt-3">
              <Space>
                <Text strong className="text-sm flex items-center">
                  <Users className="w-3.5 h-3.5 mr-1" />

                  candidateGroups
                </Text>
                <Badge
                  count={currentCandidateGroups.length}
                  size="small"
                  style={{ backgroundColor: currentCandidateGroups.length > 0 ? '#52c41a' : '#d9d9d9' }}
                />
              </Space>
              <Select
                mode="multiple"
                placeholder="Select approval group (multiple selection)"
                value={currentCandidateGroups}
                onChange={values => apply({ candidateGroups: toCsv(values) })}
                onSearch={value => {
                  const trimmed = value.trim();
                  setGroupSearchText(trimmed);
                  if (groupSearchTimerRef.current) clearTimeout(groupSearchTimerRef.current);
                  groupSearchTimerRef.current = setTimeout(() => {
                    if (trimmed) loadGroups(trimmed);
                  }, 300);
                }}
                filterOption={false}
                className="w-full mt-2"
                loading={loadingGroups}
                maxTagCount="responsive"
                notFoundContent={
                  loadingGroups ? (
                    <span className="text-gray-400">Loading...</span>
                  ) : (
                    <Empty
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                      description={
                        <span className="text-xs">

                          There is no approval group yet, please come first{' '}
                          <a href="/admin/groups" target="_blank" rel="noreferrer">

                            Group Management
                          </a>{' '}

                          Create
                        </span>
                      }
                    />
                  )
                }
                options={groupOptions}
                size="small"
              />
              <Alert
                type="info"
                showIcon
                className="mt-2 text-xs"
                title="Approval by any member of the approval group is deemed as the node has passed."
              />
            </div>

            <Divider className="my-2" />

            {/* 表单键与优先级 */}
            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <FileText className="w-3.5 h-3.5 mr-1" />

                formKey (formKey)
              </Text>
              <DebouncedInput
                placeholder="For example: approve_form_v1"
                value={currentFormKey}
                onCommit={value => apply({ formKey: value })}
                allowClear
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Hash className="w-3.5 h-3.5 mr-1" />

                priority
              </Text>
              <DebouncedInput
                type="number"
                placeholder="0-100, the higher the value, the higher the priority."
                value={currentPriority}
                onCommit={value => apply({ priority: value })}
                allowClear
                size="small"
                min={0}
                max={100}
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Clock className="w-3.5 h-3.5 mr-1" />

                due date (dueDate)
              </Text>
              <DebouncedInput
                placeholder="ISO 8601 format, for example 2026-09-20T18:00:00+08:00"
                value={currentDueDate}
                onCommit={value => apply({ dueDate: value })}
                allowClear
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Timer className="w-3.5 h-3.5 mr-1" />

                Reminder time (followUpDate)
              </Text>
              <DebouncedInput
                placeholder="ISO 8601 format, for example 2026-09-19T09:00:00+08:00"
                value={currentFollowUpDate}
                onCommit={value => apply({ followUpDate: value })}
                allowClear
                size="small"
              />
            </div>
          </>
        )}

        {/* 服务任务配置 */}
        {isServiceTask && !isCCTask && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Server className="w-3.5 h-3.5 mr-1" />

                Service implementation type
              </Text>
              <Select
                value={currentImplementation || undefined}
                onChange={value => apply({ implementation: value || '' })}
                placeholder="Select service implementation type"
                className="w-full"
                size="small"
                options={implementationOptions}
              />
              <Text type="secondary" className="text-xs mt-1 block">

                The handler is uniquely determined by this attribute, and only optional runtime handlers are implemented.
              </Text>
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Hash className="w-3.5 h-3.5 mr-1" />

                Result storage variable name
              </Text>
              <DebouncedInput
                value={currentResultVariable}
                onCommit={value => apply({ resultVariable: value })}
                placeholder="The runtime does not yet support result writeback variables"
                disabled
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                Asynchronous execution
              </Text>
              <Switch
                aria-label="Asynchronous execution"
                checked={currentAsync}
                onChange={checked => apply({ async: checked })}
                disabled
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                The service task is currently executed synchronously, and the asynchronous schedule has not yet been wired.
              </Text>
            </div>

            <Alert
              type="info"
              showIcon
              className="mt-2 text-xs"
              title="The service task will automatically call the configured external interface or service when the process is executed to this node, without manual intervention."
            />
          </>
        )}

        {/* 抄送任务配置 */}
        {isCCTask && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Users className="w-3.5 h-3.5 mr-1" />

                Cc type
              </Text>
              <Select
                value={currentCCType}
                onChange={value => apply({ ccType: value })}
                placeholder="Select CC type"
                className="w-full"
                size="small"
                options={[
                  { label: 'Single user', value: 'user' },
                  { label: 'User group', value: 'group' },
                  { label: 'Role', value: 'role' },
                  { label: 'Dynamic variable', value: 'variable' }
                ]}
              />
            </div>

            {currentCCType === 'user' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <User className="w-3.5 h-3.5 mr-1" />

                  Cc person
                </Text>
                <Select
                  mode="multiple"
                  value={parseCsv(currentCCUserIds)}
                  onChange={values => apply({ ccUserIds: toCsv(values) })}
                  placeholder="Please select the user to copy"
                  className="w-full"
                  size="small"
                  loading={loadingUsers}
                  options={ccUserOptions}
                  showSearch
                  optionFilterProp="label"
                />
              </div>
            )}

            {currentCCType === 'group' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Users className="w-3.5 h-3.5 mr-1" />

                  User group
                </Text>
                <Select
                  mode="multiple"
                  value={parseCsv(currentCCGroupIds)}
                  onChange={values => apply({ ccGroupIds: toCsv(values) })}
                  placeholder="Please select a user group"
                  className="w-full"
                  size="small"
                  loading={loadingGroups}
                  options={ccGroupOptions}
                  showSearch
                  optionFilterProp="label"
                />
              </div>
            )}

            {currentCCType === 'role' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Shield className="w-3.5 h-3.5 mr-1" />

                  Role
                </Text>
                <Select
                  mode="multiple"
                  value={parseCsv(currentCCRoleIds)}
                  onChange={values => apply({ ccRoleIds: toCsv(values) })}
                  placeholder="Please select a role"
                  className="w-full"
                  size="small"
                  loading={loadingRoles}
                  options={ccRoleOptions}
                  showSearch
                  optionFilterProp="label"
                />
              </div>
            )}

            {currentCCType === 'variable' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Hash className="w-3.5 h-3.5 mr-1" />

                  Dynamic variable name
                </Text>
                <DebouncedInput
                  value={currentCCVariable}
                  onCommit={value => apply({ ccVariable: value })}
                  placeholder="Variable name, such as ccUserIds, variable value should be user ID array"
                  size="small"
                />
              </div>
            )}

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Bell className="w-3.5 h-3.5 mr-1" />

                Send notification
              </Text>
              <Switch
                aria-label="Send notification"
                checked={currentCCNotify}
                onChange={checked => apply({ ccNotify: checked })}
                checkedChildren="On"
                unCheckedChildren="Close"
              />
            </div>

            {currentCCNotify && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <MessageCircle className="w-3.5 h-3.5 mr-1" />

                  Notification Channels
                </Text>
                <Select
                  mode="multiple"
                  value={currentNotifyChannels}
                  onChange={values => apply({ notifyChannels: toCsv(values) })}
                  placeholder="Please select notification channels"
                  className="w-full"
                  size="small"
                  options={notifyChannelOptions}
                />
              </div>
            )}
          </>
        )}

        {/* 脚本任务配置 */}
        {isScriptTask && (
          <>
            <Divider className="my-2" />

            <Alert
              type="warning"
              title="The script task has not yet provided an isolation executor. The configuration can be retained but the process cannot be published."
              showIcon
              className="mb-3"
            />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Code className="w-3.5 h-3.5 mr-1" />

                Scripting language
              </Text>
              <Select
                disabled
                value={currentScriptFormat}
                onChange={value => apply({ scriptFormat: value })}
                className="w-full"
                size="small"
                options={scriptFormatOptions}
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Code className="w-3.5 h-3.5 mr-1" />

                Script content
              </Text>
              <DebouncedTextArea
                value={currentScript}
                onCommit={value => apply({ script: value })}
                placeholder="Enter the script code to be executed"
                rows={6}
                className="font-mono text-xs"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                Process variables can be obtained through execution.getVariable('variable name') and variables can be set through execution.setVariable('variable name', value)
              </Text>
            </div>
          </>
        )}

        {/* 业务规则任务配置 */}
        {isBusinessRuleTask && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Database className="w-3.5 h-3.5 mr-1" />

                Rule Reference (ruleRef)
              </Text>
              <DebouncedInput
                value={currentRuleRef}
                onCommit={value => apply({ ruleRef: value })}
                placeholder="Business rule ID or name"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <FileText className="w-3.5 h-3.5 mr-1" />

                Input parameters
              </Text>
              <DebouncedTextArea
                value={currentRuleInput}
                onCommit={value => apply({ ruleInput: value })}
                placeholder="Input parameter mapping, JSON format"
                rows={3}
                className="font-mono text-xs"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <FileText className="w-3.5 h-3.5 mr-1" />

                Output parameters
              </Text>
              <DebouncedTextArea
                value={currentRuleOutput}
                onCommit={value => apply({ ruleOutput: value })}
                placeholder="Output result mapping, JSON format"
                rows={3}
                className="font-mono text-xs"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                Asynchronous execution
              </Text>
              <Switch
                aria-label="Asynchronous execution"
                checked={currentAsync}
                onChange={checked => apply({ async: checked })}
                size="small"
              />
            </div>
          </>
        )}

        {/* 发送任务配置 */}
        {isSendTask && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <MessageCircle className="w-3.5 h-3.5 mr-1" />

                message reference (messageRef)
              </Text>
              <DebouncedInput
                value={currentMessageRef}
                onCommit={value => apply({ messageRef: value })}
                placeholder="Message definition ID"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                operation
              </Text>
              <DebouncedInput
                value={currentOperation}
                onCommit={value => apply({ operation: value })}
                placeholder="Send operation identification"
                size="small"
              />
            </div>
          </>
        )}

        {/* 接收任务配置 */}
        {isReceiveTask && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <MessageCircle className="w-3.5 h-3.5 mr-1" />

                message reference (messageRef)
              </Text>
              <DebouncedInput
                value={currentMessageRef}
                onCommit={value => apply({ messageRef: value })}
                placeholder="Message definition ID waiting to be received"
                size="small"
              />
            </div>
          </>
        )}

        {/* 排他网关配置 */}
        {isExclusiveGateway && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <GitBranch className="w-3.5 h-3.5 mr-1" />

                Default branch
              </Text>
              <DebouncedInput
                value={currentDefaultFlow}
                onCommit={value => apply({ default: value })}
                placeholder="Enter the node ID of the default transfer"
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                When all conditions are not met, the process will take the default branch
              </Text>
            </div>

            <Alert
              type="info"
              showIcon
              className="mt-2 text-xs"
              title="The specific conditions of the gateway need to be configured separately on the output sequence stream. Click the corresponding connection line to set the conditional expression."
            />
          </>
        )}

        {/* 包容网关配置 */}
        {isInclusiveGateway && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <GitBranch className="w-3.5 h-3.5 mr-1" />

                Default branch
              </Text>
              <DebouncedInput
                value={currentDefaultFlow}
                onCommit={value => apply({ default: value })}
                placeholder="Enter the node ID of the default transfer"
                size="small"
              />
            </div>

            <Alert
              type="info"
              showIcon
              className="mt-2 text-xs"
              title="The inclusive gateway will execute all branches whose conditions are true, and will continue to execute downwards after all are completed."
            />
          </>
        )}

        {/* 复杂网关配置 */}
        {isComplexGateway && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                Activation conditions
              </Text>
              <DebouncedTextArea
                value={currentCondition}
                onCommit={value => apply({ activationCondition: value })}
                placeholder="Enter activation condition expression"
                rows={3}
                className="font-mono text-xs"
              />
            </div>

            <Alert
              type="info"
              showIcon
              className="mt-2 text-xs"
              title="Complex gateway supports customized branch merge conditions, which is suitable for complex process control scenarios."
            />
          </>
        )}

        {/* 序列流配置 */}
        {isSequenceFlow && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <GitBranch className="w-3.5 h-3.5 mr-1" />

                Flow condition expression
              </Text>
              <DebouncedTextArea
                value={currentConditionExpression}
                onCommit={value => applyCondition(value)}
                placeholder={'For example: ${variables["amount"] > 10000}'}
                rows={3}
                className="font-mono text-xs"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                {'Use the form ${variables[key]} to read process variables; press expr-lang to evaluate at runtime,'}
                {'JavaScript syntax is not supported. Return true along this connection.'}
              </Text>
            </div>
          </>
        )}

        {/* 定时事件配置：中间捕获事件必配；边界/开始事件可选启用 */}
        {(isIntermediateCatchEvent || isBoundaryEvent || isStartEvent) && (
          <>
            <Divider className="my-2" />

            {(isBoundaryEvent || isStartEvent) && !hasNonTimerEventDefinition && (
              <div className="mb-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Clock className="w-3.5 h-3.5 mr-1" />
                  {isBoundaryEvent ? 'Timed boundary event' : 'Scheduled start event'}
                </Text>
                <Switch
                  size="small"
                  aria-label={isBoundaryEvent ? 'Timed boundary event' : 'Scheduled start event'}
                  checked={currentHasTimerDefinition}
                  onChange={checked => {
                    if (checked) {
                      apply({ timeDuration: 'PT1H', timeDate: '', timeCycle: '' });
                    } else {
                      // 清空三种表达式后 normalizeTimerExpressions 会移除 TimerEventDefinition
                      apply({ timeDuration: '', timeDate: '', timeCycle: '' });
                    }
                  }}
                />
                <Text type="secondary" className="text-xs ml-2">
                  {currentHasTimerDefinition
                    ? 'Enabled, turning off will remove the timer configuration'
                    : isBoundaryEvent
                      ? 'When enabled, the task will be interrupted and take an abnormal path when it times out.'
                      : 'Automatically start the process periodically after being turned on'}
                </Text>
              </div>
            )}

            {isTimerEvent && (
              <>
                <div>
                  <Text strong className="text-sm flex items-center mb-2">
                    <Clock className="w-3.5 h-3.5 mr-1" />

                    Timing type
                  </Text>
                  <Select
                    value={currentTimerType}
                    onChange={value => {
                      // timerType 由表达式推导：切换类型即写入对应表达式并清空其余两种
                      const seed: Record<string, string> = {
                        duration: currentTimerDefinition || 'PT1H',
                        date:
                          currentTimerDate ||
                          `${new Date(Date.now() + 86400000).toISOString().slice(0, 19)}Z`,
                        cycle: currentTimerCycle || 'R/PT1H',
                      };
                      apply({
                        timeDuration: value === 'duration' ? seed.duration : '',
                        timeDate: value === 'date' ? seed.date : '',
                        timeCycle: value === 'cycle' ? seed.cycle : '',
                      });
                    }}
                    options={timerTypeOptions}
                    size="small"
                    className="w-full"
                  />
                </div>

                {currentTimerType === 'duration' && (
                  <div className="mt-2">
                    <Text strong className="text-sm flex items-center mb-2">
                      <Timer className="w-3.5 h-3.5 mr-1" />

                      Duration
                    </Text>
                    <DebouncedInput
                      value={currentTimerDefinition}
                      onCommit={value => apply({ timeDuration: value })}
                      placeholder="For example: PT1H (executed after 1 hour)"
                      size="small"
                    />
                    <Text type="secondary" className="text-xs mt-1 block">

                      Support ISO 8601 duration format: PnYnMnDTnHnMnS; clear to remove the timer
                    </Text>
                  </div>
                )}

                {currentTimerType === 'cycle' && (
                  <div className="mt-2">
                    <Text strong className="text-sm flex items-center mb-2">
                      <Timer className="w-3.5 h-3.5 mr-1" />

                      Periodic expression
                    </Text>
                    <DebouncedInput
                      value={currentTimerCycle}
                      onCommit={value => apply({ timeCycle: value })}
                      placeholder="For example: R/PT1H (executed once every hour)"
                      size="small"
                    />
                    <Text type="secondary" className="text-xs mt-1 block">

                      Supports repeated execution format R[number of times]/[interval time], for example, R/PT1H means execution once every hour
                    </Text>
                  </div>
                )}

                {currentTimerType === 'date' && (
                  <div className="mt-2">
                    <Text strong className="text-sm flex items-center mb-2">
                      <Timer className="w-3.5 h-3.5 mr-1" />

                      Specified time
                    </Text>
                    <DebouncedInput
                      value={currentTimerDate}
                      onCommit={value => apply({ timeDate: value })}
                      placeholder="For example: 2026-12-31T23:59:59Z"
                      size="small"
                    />
                    <Text type="secondary" className="text-xs mt-1 block">

                      Support ISO 8601 date and time format (RFC3339)
                    </Text>
                  </div>
                )}
              </>
            )}
          </>
        )}

        {/* 消息事件配置 */}
        {isMessageEvent && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <MessageCircle className="w-3.5 h-3.5 mr-1" />

                message reference (messageRef)
              </Text>
              <DebouncedInput
                value={currentMessageRef}
                onCommit={value => apply({ messageRef: value })}
                placeholder="Message definition ID or name"
                size="small"
              />
            </div>
          </>
        )}

        {/* 信号事件配置 */}
        {isSignalEvent && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Radio className="w-3.5 h-3.5 mr-1" />

                signal reference (signalRef)
              </Text>
              <DebouncedInput
                value={currentSignalRef}
                onCommit={value => apply({ signalRef: value })}
                placeholder="Signal definition ID or name"
                size="small"
              />
            </div>
          </>
        )}

        {/* 错误事件配置 */}
        {isErrorEvent && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <AlertTriangle className="w-3.5 h-3.5 mr-1" />

                error code (errorCode)
              </Text>
              <DebouncedInput
                value={currentErrorCode}
                onCommit={value => apply({ errorCode: value })}
                placeholder="Error code to catch or throw"
                size="small"
              />
            </div>
          </>
        )}

        {/* 升级事件配置 */}
        {isEscalationEvent && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <AlertTriangle className="w-3.5 h-3.5 mr-1" />

                escalation code (escalationCode)
              </Text>
              <DebouncedInput
                value={currentEscalationCode}
                onCommit={value => apply({ escalationCode: value })}
                placeholder="Upgrade event code"
                size="small"
              />
            </div>
          </>
        )}

        {/* 条件事件配置 */}
        {isConditionalEvent && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                Conditional expression
              </Text>
              <DebouncedTextArea
                value={currentCondition}
                onCommit={value => apply({ condition: value })}
                placeholder="Enter a conditional expression and trigger an event when true is returned"
                rows={3}
                className="font-mono text-xs"
              />
            </div>
          </>
        )}

        {/* 边界事件公共配置 */}
        {isBoundaryEvent && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                Interrupt original task
              </Text>
              <Switch
                aria-label="Interrupt original task"
                checked={currentCancelActivity}
                onChange={checked => apply({ cancelActivity: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                When the event is turned on, the execution of the original task will be interrupted. When the event is turned off, the original task will continue to execute after the event is triggered.
              </Text>
            </div>
          </>
        )}

        {/* 子流程配置 */}
        {isSubProcess && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                event triggered
              </Text>
              <Switch
                aria-label="event triggered"
                checked={currentTriggeredByEvent}
                onChange={checked => apply({ triggeredByEvent: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                When enabled, this sub-process is an event sub-process and is executed triggered by events.
              </Text>
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                Compensation Process
              </Text>
              <Switch
                aria-label="Compensation Process"
                checked={currentIsForCompensation}
                onChange={checked => apply({ isForCompensation: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                When enabled, this sub-process is a compensation process and is used to execute when a transaction is rolled back.
              </Text>
            </div>

            {nodeType === 'Transaction' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Settings className="w-3.5 h-3.5 mr-1" />

                  Transaction method
                </Text>
                <Select
                  value={currentTransactionMethod}
                  onChange={value => apply({ transactionMethod: value })}
                  options={[
                    { label: 'Standard transaction', value: 'standard' },
                    { label: 'Nested transactions', value: 'nested' },
                    { label: 'Independent transaction', value: 'requiresNew' }
                  ]}
                  size="small"
                  className="w-full"
                />
              </div>
            )}
          </>
        )}

        {/* 调用活动配置 */}
        {isCallActivity && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Link className="w-3.5 h-3.5 mr-1" />

                Calling process ID
              </Text>
              <DebouncedInput
                value={currentCalledElement}
                onCommit={value => apply({ calledElement: value })}
                placeholder="ID of the external process definition to be called"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                Inherited variables
              </Text>
              <Switch
                aria-label="Inherited variables"
                checked={currentInheritVariables}
                onChange={checked => apply({ inheritVariables: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                The sub-process inherits all variables of the parent process when it is started.
              </Text>
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />

                Inherit business primary key
              </Text>
              <Switch
                aria-label="Inherit business primary key"
                checked={currentInheritBusinessKey}
                onChange={checked => apply({ inheritBusinessKey: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">

                The sub-process inherits the business primary key of the parent process when it is started.
              </Text>
            </div>
          </>
        )}
      </div>
    </Card>
  );
}
