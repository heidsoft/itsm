// 工作流节点属性检查器
// Workflow Node Inspector - 监听 BPMN 画布选中节点并编辑其属性

'use client';

import React, { useEffect, useState, useRef, useCallback } from 'react';
import { Card, Empty, Select, Input, Tag, Typography, Space, Divider, Alert, Button, Switch, Tooltip, Collapse, Badge } from 'antd';
import {
  User, Users, UserCheck, Hash, Tag as TagIcon, RefreshCw,
  Code, Server, GitBranch, PlayCircle, Clock, FileText,
  Webhook, Settings, MessageSquare, Mail, AlertTriangle,
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
      const resp = await UserApi.getUsers({ page: 1, pageSize: 200, search });
      if (!cancelledRef.current) {
        setUsers(prev => {
          const existingIds = new Set(prev.map(u => u.id));
          const incoming = (resp.users as ApiUser[]).filter(u => !existingIds.has(u.id));
          return [...prev, ...incoming];
        });
      }
    } catch (err) {
      console.error('加载用户列表失败:', err);
    } finally {
      if (!cancelledRef.current) setLoadingUsers(false);
    }
  }, []);

  // 搜索防抖加载组（300ms）
  const loadGroups = useCallback(async (search = '') => {
    setLoadingGroups(true);
    try {
      const tenantId = httpClient.getTenantId();
      if (!tenantId) throw new Error('缺少有效租户上下文');
      const resp = await GroupAPI.getGroups({ page: 1, pageSize: 200, tenantId, search });
      if (!cancelledRef.current) {
        setGroups(prev => {
          const existingIds = new Set(prev.map(g => g.id));
          const incoming = (resp.groups || []).filter(g => !existingIds.has(g.id));
          return [...prev, ...incoming];
        });
      }
    } catch (err) {
      console.error('加载组列表失败:', err);
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
        const resp = await RoleAPI.getRoles();
        const list = (resp as unknown as { roles?: { id: number; name: string; code: string }[]; data?: { id: number; name: string; code: string }[] })?.roles ?? (resp as unknown as { roles?: { id: number; name: string; code: string }[]; data?: { id: number; name: string; code: string }[] })?.data ?? [];
        if (!cancelled) setRoles(list);
      } catch (err) {
        console.error('加载角色列表失败:', err);
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
        if (!tenantId) throw new Error('缺少有效租户上下文');
        const [userResp, groupResp] = await Promise.all([
          UserApi.getUsers({ page: 1, pageSize: 200, search: '' }),
          GroupAPI.getGroups({ page: 1, pageSize: 200, tenantId, search: '' }),
        ]);
        if (!cancelled) {
          setUsers((userResp.users as ApiUser[]) || []);
          setGroups(groupResp.groups || []);
        }
      } catch (err) {
        console.error('加载用户/组列表失败:', err);
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
            <span>节点属性</span>
          </Space>
        }
        className="h-full rounded-lg shadow-sm border border-gray-200"
        extra={
          onRefresh && (
            <Tooltip title="刷新选中节点属性">
              <Button
                type="text"
                size="small"
                icon={<RefreshCw className="w-3 h-3" />}
                aria-label="重读节点属性"
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
                点击画布上的节点查看/编辑属性
              </span>
              <Text type="secondary" className="text-xs">
                支持拖拽节点、连接线、条件配置
              </Text>
            </Space>
          }
        />
        {/* 快捷操作提示 */}
        <div className="mt-4 p-3 bg-gray-50 rounded-lg">
          <Text strong className="text-xs block mb-2">💡 快捷操作</Text>
          <Space orientation="vertical" size={2} className="w-full">
            <Text type="secondary" className="text-xs">• 双击节点可快速编辑名称</Text>
            <Text type="secondary" className="text-xs">• 点击连接线设置流转条件</Text>
            <Text type="secondary" className="text-xs">• 拖拽节点左侧/右侧创建新流程</Text>
          </Space>
        </div>
      </Card>
    );
  }

  // 节点类型判断
  const nodeType = selection.type.replace('bpmn:', '');
  const isUserTask = nodeType === 'UserTask';
  const isServiceTask = nodeType === 'ServiceTask';
  const isScriptTask = nodeType === 'ScriptTask';
  const isBusinessRuleTask = nodeType === 'BusinessRuleTask';
  const isSendTask = nodeType === 'SendTask';
  const isReceiveTask = nodeType === 'ReceiveTask';
  const isMailTask = nodeType === 'ServiceTask' && (selection.businessObject?.type as string) === 'mail';
  const isCCTask =
    nodeType === 'ServiceTask' &&
    ((selection.businessObject?.type as string) === 'cc' ||
      (selection.businessObject?.implementation as string) === 'cc_handler' ||
      (selection.businessObject?.operationRef as string) === 'cc_handler');
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
  
  const bo = (selection.businessObject || {}) as Record<string, unknown>;

  // 通用属性
  const currentName = (bo.name as string) || '';
  const currentDocumentation = (bo.documentation as string) || '';

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
  const currentOperationRef = (bo.operationRef as string) || '';
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

  // 邮件任务属性
  const currentMailTo = (bo.mailTo as string) || '';
  const currentMailCc = (bo.mailCc as string) || '';
  const currentMailSubject = (bo.mailSubject as string) || '';
  const currentMailTemplate = (bo.mailTemplate as string) || '';

  // 抄送任务属性
  const currentCCType = (bo.ccType as string) || 'user'; // user, group, role, variable
  const currentCCUserIds = (bo.ccUserIds as string) || '';
  const currentCCGroupIds = (bo.ccGroupIds as string) || '';
  const currentCCRoleIds = (bo.ccRoleIds as string) || '';
  const currentCCVariable = (bo.ccVariable as string) || '';
  const currentCCNotify = (bo.ccNotify as boolean) ?? true;
  const currentNotifyChannels = parseCsv((bo.notifyChannels as string) || 'in_app');


  // 网关/序列流属性
  const currentConditionExpression = (bo.conditionExpression && typeof bo.conditionExpression === 'object' 
    ? (bo.conditionExpression as unknown as { body?: string })?.body || '' 
    : (bo.conditionExpression as string) || '');
  const currentDefaultFlow = (bo.default as string) || '';

  // 事件属性
  const currentTimerDefinition = (bo.timeDuration as string) || '';
  const currentTimerCycle = (bo.timeCycle as string) || '';
  const currentTimerDate = (bo.timeDate as string) || '';
  const currentTimerType = (bo.timerType as string) || 'duration';
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
  const currentInheritVariables = (bo.inheritVariables as boolean) || true;
  const currentInheritBusinessKey = (bo.inheritBusinessKey as boolean) || true;

  // 事务子流程属性
  const currentTransactionMethod = (bo.transactionMethod as string) || 'standard';

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
    { label: 'HTTP 接口调用（未就绪）', value: 'http', disabled: true },
    { label: 'Java 类调用（未支持）', value: 'java', disabled: true },
    { label: '表达式（未就绪）', value: 'expression', disabled: true },
    { label: 'Webhook', value: 'webhook' },
    { label: '系统内置服务（请选择已注册 handler）', value: 'generic_handler' },
    { label: '邮件发送（未就绪）', value: 'mail', disabled: true },
    { label: '自动抄送', value: 'cc_handler' }
  ];

  const notifyChannelOptions = [
    { label: '站内信', value: 'in_app' },
    { label: '邮件', value: 'email' },
    { label: '短信', value: 'sms' },
    { label: '飞书', value: 'feishu' },
    { label: '钉钉', value: 'dingtalk' },
    { label: '企业微信', value: 'wecom' },
    { label: 'Webhook', value: 'webhook' }
  ];

  // 定时类型选项
  const timerTypeOptions = [
    { label: '持续时间', value: 'duration' },
    { label: '周期执行', value: 'cycle' },
    { label: '指定时间', value: 'date' }
  ];

  // 应用修改
  const apply = (patch: Record<string, unknown>) => {
    onUpdateProperties(selection.id, patch);
  };

  // 应用条件表达式修改
  const applyCondition = (value: string) => {
    // 对于条件表达式，需要符合BPMN的结构
    if (isSequenceFlow) {
      apply({ 
        conditionExpression: {
          type: 'bpmn:FormalExpression',
          body: value,
          language: 'javascript'
        } 
      });
    } else {
      apply({ conditionExpression: value });
    }
  };

  return (
    <Card
      title={
        <Space>
          <TagIcon className="w-4 h-4" />
          <span>节点属性</span>
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
            重读
          </Button>
        )
      }
      size="small"
    >
      <div className="space-y-4 pb-4">
        {/* 基础信息 - 所有节点通用 */}
        <div>
          <Text type="secondary" className="text-xs">
            节点 ID
          </Text>
          <div className="font-mono text-xs mt-1 px-2 py-1 bg-gray-50 rounded">
            {selection.id}
          </div>
          
          <div className="mt-2">
            <Text type="secondary" className="text-xs">
              节点名称
            </Text>
            <DebouncedInput
              value={currentName}
              onCommit={value => apply({ name: value })}
              placeholder="输入节点显示名称"
              size="small"
              className="mt-1"
            />
          </div>

          <div className="mt-2">
            <Text type="secondary" className="text-xs">
              描述信息
            </Text>
            <DebouncedTextArea
              value={currentDocumentation}
              onCommit={value => apply({ documentation: value })}
              placeholder="输入节点功能描述（可选）"
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
                <Shield className="w-3.5 h-3.5 mr-1" />审批语义
              </Text>
              <Select
                value={currentTaskPurpose}
                onChange={value => apply({ taskPurpose: value })}
                options={[{ label: '普通人工任务', value: 'work' }, { label: '审批任务', value: 'approval' }]}
                className="w-full" size="small"
              />
              {currentTaskPurpose === 'approval' && (
                <Space orientation="vertical" className="w-full mt-2" size="small">
                  <Select
                    value={currentApprovalMode}
                    onChange={value => apply({ approvalMode: value })}
                    options={[
                      { label: '单人审批', value: 'single' }, { label: '任一通过', value: 'any' },
                      { label: '全部通过', value: 'all' }, { label: '比例/阈值通过', value: 'threshold' },
                      { label: '顺序会签', value: 'sequential' },
                    ]}
                    className="w-full" size="small"
                  />
                  {currentApprovalMode === 'threshold' && (
                    <DebouncedInput type="number" min={1} value={String(currentApprovalThreshold)}
                      onCommit={value => apply({ approvalThreshold: Number(value || 1) })}
                      addonBefore="通过人数" size="small" />
                  )}
                  <Select value={currentRejectStrategy} onChange={value => apply({ rejectStrategy: value })}
                    options={[{ label: '终止流程', value: 'terminate' }, { label: '退回发起人（未就绪）', value: 'to_requester', disabled: true }, { label: '进入拒绝分支', value: 'gateway' }]}
                    className="w-full" size="small" />
                  <Select value={currentTimeoutAction} onChange={value => apply({ timeoutAction: value })}
                    disabled
                    options={[{ label: '仅提醒（超时调度未就绪）', value: 'notify' }, { label: '升级审批（未就绪）', value: 'escalate' }, { label: '自动拒绝（未就绪）', value: 'auto_reject' }]}
                    className="w-full" size="small" />
                  <Space wrap>
                    <Switch size="small" checked={currentAllowDelegate} onChange={v => apply({ allowDelegate: v })} />委托
                    <Switch size="small" checked={currentAllowAddApprover} onChange={v => apply({ allowAddApprover: v })} />加签
                    <Switch size="small" checked={currentCommentRequiredOnReject} onChange={v => apply({ commentRequiredOnReject: v })} />拒绝意见必填
                  </Space>
                </Space>
              )}
            </div>

            {/* 快捷操作栏 */}
            <div className="mb-3 p-2 bg-blue-50 rounded-lg">
              <Space wrap>
                <Text strong className="text-xs text-blue-700">⚡ 常用配置：</Text>
                <Button
                  size="small"
                  type="text"
                  onClick={() => apply({ assignee: currentAssignee || 'admin' })}
                >
                  设为管理员
                </Button>
                <Button
                  size="small"
                  type="text"
                  onClick={() => apply({ priority: '50' })}
                >
                  普通优先级
                </Button>
                <Button
                  size="small"
                  type="text"
                  onClick={() => apply({ dueDate: 'PT24H' })}
                >
                  24小时超时
                </Button>
              </Space>
            </div>

            {/* Assignee */}
            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <UserCheck className="w-3.5 h-3.5 mr-1" />
                受理人 (assignee)
                <Tag color="blue" className="ml-2 text-xs">单人</Tag>
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
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                指定单一用户为该任务的处理人
              </Text>
            </div>

            {/* Candidate Users */}
            <div className="mt-3">
              <Text strong className="text-sm flex items-center mb-2">
                <User className="w-3.5 h-3.5 mr-1" />
                候选人 (candidateUsers)
              </Text>
              <Select
                mode="multiple"
                placeholder="选择候选人（多选）"
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
                任一候选人可处理该任务
              </Text>
            </div>

            {/* Candidate Groups — 核心审批组入口 */}
            <div className="mt-3">
              <Space>
                <Text strong className="text-sm flex items-center">
                  <Users className="w-3.5 h-3.5 mr-1" />
                  候选组 (candidateGroups)
                </Text>
                <Badge
                  count={currentCandidateGroups.length}
                  size="small"
                  style={{ backgroundColor: currentCandidateGroups.length > 0 ? '#52c41a' : '#d9d9d9' }}
                />
              </Space>
              <Select
                mode="multiple"
                placeholder="选择审批组（多选）"
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
                options={groupOptions}
                size="small"
              />
              <Alert
                type="info"
                showIcon
                className="mt-2 text-xs"
                message="审批组中任一成员审批即视为该节点通过"
              />
            </div>

            <Divider className="my-2" />

            {/* 表单键与优先级 */}
            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <FileText className="w-3.5 h-3.5 mr-1" />
                表单键 (formKey)
              </Text>
              <DebouncedInput
                placeholder="例如：approve_form_v1"
                value={currentFormKey}
                onCommit={value => apply({ formKey: value })}
                allowClear
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Hash className="w-3.5 h-3.5 mr-1" />
                优先级 (priority)
              </Text>
              <DebouncedInput
                type="number"
                placeholder="例如：50（数值越高越优先）"
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
                截止时间 (dueDate)
              </Text>
              <DebouncedInput
                placeholder="ISO 8601 或 P0Y0M0DT2H0M0S 格式"
                value={currentDueDate}
                onCommit={value => apply({ dueDate: value })}
                allowClear
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Timer className="w-3.5 h-3.5 mr-1" />
                提醒时间 (followUpDate)
              </Text>
              <DebouncedInput
                placeholder="ISO 8601 或 P0Y0M0DT1H0M0S 格式"
                value={currentFollowUpDate}
                onCommit={value => apply({ followUpDate: value })}
                allowClear
                size="small"
              />
            </div>
          </>
        )}

        {/* 服务任务配置 */}
        {isServiceTask && !isMailTask && !isCCTask && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Server className="w-3.5 h-3.5 mr-1" />
                服务实现类型
              </Text>
              <Select
                value={currentImplementation || undefined}
                onChange={value => apply({ implementation: value || '' })}
                placeholder="选择服务实现类型"
                className="w-full"
                size="small"
                options={implementationOptions}
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Webhook className="w-3.5 h-3.5 mr-1" />
                操作引用 (operationRef)
              </Text>
              <DebouncedInput
                value={currentOperationRef}
                onCommit={value => apply({ operationRef: value })}
                placeholder="输入操作标识或URL"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Hash className="w-3.5 h-3.5 mr-1" />
                结果存储变量名
              </Text>
              <DebouncedInput
                value={currentResultVariable}
                onCommit={value => apply({ resultVariable: value })}
                placeholder="例如：httpResult（执行结果将存入该变量）"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />
                异步执行
              </Text>
              <Switch
                checked={currentAsync}
                onChange={checked => apply({ async: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                开启后任务将异步执行，不阻塞主流程
              </Text>
            </div>

            <Alert
              type="info"
              showIcon
              className="mt-2 text-xs"
              message="服务任务会在流程执行到该节点时自动调用配置的外部接口或服务，无需人工干预。"
            />
          </>
        )}

        {/* 邮件任务配置 */}
        {/* 抄送任务配置 */}
        {isCCTask && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Users className="w-3.5 h-3.5 mr-1" />
                抄送类型
              </Text>
              <Select
                value={currentCCType}
                onChange={value => apply({ ccType: value })}
                placeholder="选择抄送类型"
                className="w-full"
                size="small"
                options={[
                  { label: '单个用户', value: 'user' },
                  { label: '用户组', value: 'group' },
                  { label: '角色', value: 'role' },
                  { label: '动态变量', value: 'variable' }
                ]}
              />
            </div>

            {currentCCType === 'user' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <User className="w-3.5 h-3.5 mr-1" />
                  抄送人
                </Text>
                <Select
                  mode="multiple"
                  value={parseCsv(currentCCUserIds)}
                  onChange={values => apply({ ccUserIds: toCsv(values) })}
                  placeholder="请选择抄送用户"
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
                  用户组
                </Text>
                <Select
                  mode="multiple"
                  value={parseCsv(currentCCGroupIds)}
                  onChange={values => apply({ ccGroupIds: toCsv(values) })}
                  placeholder="请选择用户组"
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
                  角色
                </Text>
                <Select
                  mode="multiple"
                  value={parseCsv(currentCCRoleIds)}
                  onChange={values => apply({ ccRoleIds: toCsv(values) })}
                  placeholder="请选择角色"
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
                  动态变量名
                </Text>
                <DebouncedInput
                  value={currentCCVariable}
                  onCommit={value => apply({ ccVariable: value })}
                  placeholder="变量名，如 ccUserIds，变量值应为用户ID数组"
                  size="small"
                />
              </div>
            )}

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Bell className="w-3.5 h-3.5 mr-1" />
                发送通知
              </Text>
              <Switch
                checked={currentCCNotify}
                onChange={checked => apply({ ccNotify: checked })}
                checkedChildren="开启"
                unCheckedChildren="关闭"
              />
            </div>

            {currentCCNotify && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <MessageCircle className="w-3.5 h-3.5 mr-1" />
                  通知渠道
                </Text>
                <Select
                  mode="multiple"
                  value={currentNotifyChannels}
                  onChange={values => apply({ notifyChannels: toCsv(values) })}
                  placeholder="请选择通知渠道"
                  className="w-full"
                  size="small"
                  options={notifyChannelOptions}
                />
              </div>
            )}
          </>
        )}

        {isMailTask && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Mail className="w-3.5 h-3.5 mr-1" />
                收件人 (To)
              </Text>
              <DebouncedInput
                value={currentMailTo}
                onCommit={value => apply({ mailTo: value })}
                placeholder="多个收件人用逗号分隔，支持变量如 ${applyUser.email}"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Mail className="w-3.5 h-3.5 mr-1" />
                抄送人 (Cc)
              </Text>
              <DebouncedInput
                value={currentMailCc}
                onCommit={value => apply({ mailCc: value })}
                placeholder="多个抄送人用逗号分隔"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <MessageSquare className="w-3.5 h-3.5 mr-1" />
                邮件主题
              </Text>
              <DebouncedInput
                value={currentMailSubject}
                onCommit={value => apply({ mailSubject: value })}
                placeholder="邮件主题，支持变量如 ${order.title}"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <FileText className="w-3.5 h-3.5 mr-1" />
                邮件模板
              </Text>
              <DebouncedTextArea
                value={currentMailTemplate}
                onCommit={value => apply({ mailTemplate: value })}
                placeholder="邮件内容模板，支持HTML和变量"
                rows={5}
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />
                异步执行
              </Text>
              <Switch
                checked={currentAsync}
                onChange={checked => apply({ async: checked })}
                size="small"
              />
            </div>
          </>
        )}

        {/* 脚本任务配置 */}
        {isScriptTask && (
          <>
            <Divider className="my-2" />

            <Alert
              type="warning"
              title="脚本任务尚未提供隔离执行器，配置可保留但流程不可发布"
              showIcon
              className="mb-3"
            />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Code className="w-3.5 h-3.5 mr-1" />
                脚本语言
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
                脚本内容
              </Text>
              <DebouncedTextArea
                value={currentScript}
                onCommit={value => apply({ script: value })}
                placeholder="输入要执行的脚本代码"
                rows={6}
                className="font-mono text-xs"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                可以通过 execution.getVariable('变量名') 获取流程变量，通过 execution.setVariable('变量名', 值) 设置变量
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
                规则引用 (ruleRef)
              </Text>
              <DebouncedInput
                value={currentRuleRef}
                onCommit={value => apply({ ruleRef: value })}
                placeholder="业务规则ID或名称"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <FileText className="w-3.5 h-3.5 mr-1" />
                输入参数
              </Text>
              <DebouncedTextArea
                value={currentRuleInput}
                onCommit={value => apply({ ruleInput: value })}
                placeholder="输入参数映射，JSON格式"
                rows={3}
                className="font-mono text-xs"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <FileText className="w-3.5 h-3.5 mr-1" />
                输出参数
              </Text>
              <DebouncedTextArea
                value={currentRuleOutput}
                onCommit={value => apply({ ruleOutput: value })}
                placeholder="输出结果映射，JSON格式"
                rows={3}
                className="font-mono text-xs"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />
                异步执行
              </Text>
              <Switch
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
                消息引用 (messageRef)
              </Text>
              <DebouncedInput
                value={currentMessageRef}
                onCommit={value => apply({ messageRef: value })}
                placeholder="消息定义ID"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />
                操作 (operation)
              </Text>
              <DebouncedInput
                value={currentOperation}
                onCommit={value => apply({ operation: value })}
                placeholder="发送操作标识"
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
                消息引用 (messageRef)
              </Text>
              <DebouncedInput
                value={currentMessageRef}
                onCommit={value => apply({ messageRef: value })}
                placeholder="等待接收的消息定义ID"
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
                默认分支
              </Text>
              <DebouncedInput
                value={currentDefaultFlow}
                onCommit={value => apply({ default: value })}
                placeholder="输入默认流转的节点ID"
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                当所有条件都不满足时，流程将走默认分支
              </Text>
            </div>

            <Alert
              type="info"
              showIcon
              className="mt-2 text-xs"
              message="网关的具体条件需要在输出的序列流上分别配置，点击对应的连接线即可设置条件表达式。"
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
                默认分支
              </Text>
              <DebouncedInput
                value={currentDefaultFlow}
                onCommit={value => apply({ default: value })}
                placeholder="输入默认流转的节点ID"
                size="small"
              />
            </div>

            <Alert
              type="info"
              showIcon
              className="mt-2 text-xs"
              message="包容网关会执行所有条件为true的分支，全部完成后才会继续向下执行。"
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
                激活条件
              </Text>
              <DebouncedTextArea
                value={currentCondition}
                onCommit={value => apply({ activationCondition: value })}
                placeholder="输入激活条件表达式"
                rows={3}
                className="font-mono text-xs"
              />
            </div>

            <Alert
              type="info"
              showIcon
              className="mt-2 text-xs"
              message="复杂网关支持自定义的分支合并条件，适用于复杂的流程控制场景。"
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
                流转条件表达式
              </Text>
              <DebouncedTextArea
                value={currentConditionExpression}
                onCommit={value => applyCondition(value)}
                placeholder="例如：${order.amount > 10000} 或 JavaScript 表达式"
                rows={3}
                className="font-mono text-xs"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                条件表达式返回 true 时，流程将沿此连线流转
              </Text>
            </div>
          </>
        )}

        {/* 定时事件配置 */}
        {isTimerEvent && (
          <>
            <Divider className="my-2" />

            <div>
              <Text strong className="text-sm flex items-center mb-2">
                <Clock className="w-3.5 h-3.5 mr-1" />
                定时类型
              </Text>
              <Select
                value={currentTimerType}
                onChange={value => apply({ timerType: value })}
                options={timerTypeOptions}
                size="small"
                className="w-full"
              />
            </div>

            {currentTimerType === 'duration' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Timer className="w-3.5 h-3.5 mr-1" />
                  持续时间
                </Text>
                <DebouncedInput
                  value={currentTimerDefinition}
                  onCommit={value => apply({ timeDuration: value })}
                  placeholder="例如：PT1H（1小时后执行）"
                  size="small"
                />
                <Text type="secondary" className="text-xs mt-1 block">
                  支持 ISO 8601 时长格式：PnYnMnDTnHnMnS
                </Text>
              </div>
            )}

            {currentTimerType === 'cycle' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Timer className="w-3.5 h-3.5 mr-1" />
                  周期表达式
                </Text>
                <DebouncedInput
                  value={currentTimerCycle}
                  onCommit={value => apply({ timeCycle: value })}
                  placeholder="例如：R/PT1H（每小时执行一次）或 cron 表达式"
                  size="small"
                />
                <Text type="secondary" className="text-xs mt-1 block">
                  支持重复执行格式 R[次数]/[间隔时间] 或标准 cron 表达式
                </Text>
              </div>
            )}

            {currentTimerType === 'date' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Timer className="w-3.5 h-3.5 mr-1" />
                  指定时间
                </Text>
                <DebouncedInput
                  value={currentTimerDate}
                  onCommit={value => apply({ timeDate: value })}
                  placeholder="例如：2025-12-31T23:59:59Z"
                  size="small"
                />
                <Text type="secondary" className="text-xs mt-1 block">
                  支持 ISO 8601 日期时间格式
                </Text>
              </div>
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
                消息引用 (messageRef)
              </Text>
              <DebouncedInput
                value={currentMessageRef}
                onCommit={value => apply({ messageRef: value })}
                placeholder="消息定义ID或名称"
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
                信号引用 (signalRef)
              </Text>
              <DebouncedInput
                value={currentSignalRef}
                onCommit={value => apply({ signalRef: value })}
                placeholder="信号定义ID或名称"
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
                错误代码 (errorCode)
              </Text>
              <DebouncedInput
                value={currentErrorCode}
                onCommit={value => apply({ errorCode: value })}
                placeholder="要捕获或抛出的错误代码"
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
                升级代码 (escalationCode)
              </Text>
              <DebouncedInput
                value={currentEscalationCode}
                onCommit={value => apply({ escalationCode: value })}
                placeholder="升级事件代码"
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
                条件表达式
              </Text>
              <DebouncedTextArea
                value={currentCondition}
                onCommit={value => apply({ condition: value })}
                placeholder="输入条件表达式，返回true时触发事件"
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
                中断原任务
              </Text>
              <Switch
                checked={currentCancelActivity}
                onChange={checked => apply({ cancelActivity: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                开启时事件触发会中断原任务执行，关闭时事件触发后原任务继续执行
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
                事件触发
              </Text>
              <Switch
                checked={currentTriggeredByEvent}
                onChange={checked => apply({ triggeredByEvent: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                开启时该子流程为事件子流程，由事件触发执行
              </Text>
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />
                补偿流程
              </Text>
              <Switch
                checked={currentIsForCompensation}
                onChange={checked => apply({ isForCompensation: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                开启时该子流程为补偿流程，用于事务回滚时执行
              </Text>
            </div>

            {nodeType === 'Transaction' && (
              <div className="mt-2">
                <Text strong className="text-sm flex items-center mb-2">
                  <Settings className="w-3.5 h-3.5 mr-1" />
                  事务方法
                </Text>
                <Select
                  value={currentTransactionMethod}
                  onChange={value => apply({ transactionMethod: value })}
                  options={[
                    { label: '标准事务', value: 'standard' },
                    { label: '嵌套事务', value: 'nested' },
                    { label: '独立事务', value: 'requiresNew' }
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
                调用流程ID
              </Text>
              <DebouncedInput
                value={currentCalledElement}
                onCommit={value => apply({ calledElement: value })}
                placeholder="要调用的外部流程定义ID"
                size="small"
              />
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />
                继承变量
              </Text>
              <Switch
                checked={currentInheritVariables}
                onChange={checked => apply({ inheritVariables: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                开启时子流程继承父流程的所有变量
              </Text>
            </div>

            <div className="mt-2">
              <Text strong className="text-sm flex items-center mb-2">
                <Settings className="w-3.5 h-3.5 mr-1" />
                继承业务主键
              </Text>
              <Switch
                checked={currentInheritBusinessKey}
                onChange={checked => apply({ inheritBusinessKey: checked })}
                size="small"
              />
              <Text type="secondary" className="text-xs mt-1 block">
                开启时子流程继承父流程的业务主键
              </Text>
            </div>
          </>
        )}
      </div>
    </Card>
  );
}
