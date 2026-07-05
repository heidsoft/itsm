// 工作流设计器类型定义
// Workflow Designer Types

export interface WorkflowDefinition {
  id: string;
  name: string;
  description: string;
  version: string;
  category: string;
  status: 'draft' | 'active' | 'inactive' | 'archived';
  xml: string;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  tags: string[];
  approvalConfig?: ApprovalConfig;
  variables?: WorkflowVariable[];
  slaConfig?: SLAConfig;
}

export interface ApprovalConfig {
  requireApproval: boolean;
  approvalType: 'single' | 'parallel' | 'sequential' | 'conditional';
  // 节点级“用户指派”兑底。可为空。
  approvers: string[];
  // 审批组是节点级，存于 BPMN userTask candidateGroups，不在此字段。需在「节点属性」面板中设。
  autoApproveRoles: string[];
  escalationRules: EscalationRule[];
}

export interface EscalationRule {
  level: number;
  timeoutHours: number;
  escalateTo: string[];
  action: 'notify' | 'auto_approve' | 'escalate';
}

export interface WorkflowVariable {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'object';
  required: boolean;
  defaultValue?: string | number | boolean | Date | Record<string, unknown>;
  description: string;
}

export interface SLAConfig {
  responseTimeHours: number;
  resolutionTimeHours: number;
  businessHoursOnly: boolean;
  excludeWeekends: boolean;
  excludeHolidays: boolean;
}

export interface WorkflowVersion {
  id: string;
  version: string;
  status: 'draft' | 'active' | 'archived';
  createdAt: string;
  createdBy: string;
  changeLog: string;
  xml: string;
}

export interface UserInfo {
  id: number;
  name: string;
  username: string;
}

export interface RoleInfo {
  id: number;
  name: string;
  code: string;
}

export interface GroupInfo {
  id: number;
  name: string;
  description?: string;
  memberCount?: number;
}

export interface WorkflowDesignerState {
  workflow: WorkflowDefinition | null;
  currentXML: string;
  hasChanges: boolean;
  activeTab: string;
  saving: boolean;
  deploying: boolean;
  approvalConfig: ApprovalConfig;
  workflowVersions: WorkflowVersion[];
  userList: UserInfo[];
  roleList: RoleInfo[];
  groupList: GroupInfo[];
  loadingUsers: boolean;
  loadingRoles: boolean;
  loadingGroups: boolean;
}

export interface WorkflowDesignerActions {
  setWorkflow: (workflow: WorkflowDefinition | null) => void;
  setCurrentXML: (xml: string) => void;
  setHasChanges: (hasChanges: boolean) => void;
  setActiveTab: (tab: string) => void;
  setSaving: (saving: boolean) => void;
  setDeploying: (deploying: boolean) => void;
  setApprovalConfig: (config: ApprovalConfig) => void;
  setWorkflowVersions: (versions: WorkflowVersion[]) => void;
  setUserList: (users: UserInfo[]) => void;
  setRoleList: (roles: RoleInfo[]) => void;
  setGroupList: (groups: GroupInfo[]) => void;
  setLoadingUsers: (loading: boolean) => void;
  setLoadingRoles: (loading: boolean) => void;
  setLoadingGroups: (loading: boolean) => void;
  updateWorkflow: (updates: Partial<WorkflowDefinition>) => void;
  updateSLAConfig: (config: Partial<SLAConfig>) => void;
  addWorkflowVersion: (version: WorkflowVersion) => void;
}

export type WorkflowDesignerContextType = WorkflowDesignerState & WorkflowDesignerActions;

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  bpmnXml: string;
  approvalConfig: {
    requireApproval: boolean;
    approvalType: 'single' | 'parallel' | 'sequential' | 'conditional';
    approvers: string[];
  };
}

export interface BPMNDesignerProps {
  xml: string;
  onSave: (xml: string) => void;
  onChange: (xml: string) => void;
}
