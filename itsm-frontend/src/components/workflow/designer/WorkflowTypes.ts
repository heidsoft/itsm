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
  created_at: string;
  updated_at: string;
  created_by: string;
  tags: string[];
  approval_config?: ApprovalConfig;
  variables?: WorkflowVariable[];
  sla_config?: SLAConfig;
}

export interface ApprovalConfig {
  require_approval: boolean;
  approval_type: 'single' | 'parallel' | 'sequential' | 'conditional';
  approvers: string[];
  auto_approve_roles: string[];
  escalation_rules: EscalationRule[];
}

export interface EscalationRule {
  level: number;
  timeout_hours: number;
  escalate_to: string[];
  action: 'notify' | 'auto_approve' | 'escalate';
}

export interface WorkflowVariable {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'object';
  required: boolean;
  default_value?: string | number | boolean | Date | Record<string, unknown>;
  description: string;
}

export interface SLAConfig {
  response_time_hours: number;
  resolution_time_hours: number;
  business_hours_only: boolean;
  exclude_weekends: boolean;
  exclude_holidays: boolean;
}

export interface WorkflowVersion {
  id: string;
  version: string;
  status: 'draft' | 'active' | 'archived';
  created_at: string;
  created_by: string;
  change_log: string;
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
  loadingUsers: boolean;
  loadingRoles: boolean;
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
  setLoadingUsers: (loading: boolean) => void;
  setLoadingRoles: (loading: boolean) => void;
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
  bpmn_xml: string;
  approval_config: {
    require_approval: boolean;
    approval_type: 'single' | 'parallel' | 'sequential' | 'conditional';
    approvers: string[];
  };
}

export interface BPMNDesignerProps {
  xml: string;
  onSave: (xml: string) => void;
  onChange: (xml: string) => void;
}
