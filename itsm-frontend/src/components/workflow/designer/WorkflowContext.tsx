// 工作流设计器上下文
// Workflow Designer Context - 状态管理

'use client';

import React, { createContext, useContext, useReducer, ReactNode, useCallback } from 'react';
import type {
  WorkflowDesignerContextType,
  WorkflowDefinition,
  ApprovalConfig,
  SLAConfig,
  WorkflowVersion,
} from './WorkflowTypes';

// 初始审批配置
const defaultApprovalConfig: ApprovalConfig = {
  require_approval: true,
  approval_type: 'sequential',
  approvers: [],
  auto_approve_roles: [],
  escalation_rules: [],
};

// 初始状态
const initialState: WorkflowDesignerContextType = {
  workflow: null,
  currentXML: '',
  hasChanges: false,
  activeTab: 'designer',
  saving: false,
  deploying: false,
  approvalConfig: defaultApprovalConfig,
  workflowVersions: [],
  userList: [],
  roleList: [],
  loadingUsers: false,
  loadingRoles: false,
  setWorkflow: () => {},
  setCurrentXML: () => {},
  setHasChanges: () => {},
  setActiveTab: () => {},
  setSaving: () => {},
  setDeploying: () => {},
  setApprovalConfig: () => {},
  setWorkflowVersions: () => {},
  setUserList: () => {},
  setRoleList: () => {},
  setLoadingUsers: () => {},
  setLoadingRoles: () => {},
  updateWorkflow: () => {},
  updateSLAConfig: () => {},
  addWorkflowVersion: () => {},
};

// Action 类型
type WorkflowDesignerAction =
  | { type: 'SET_WORKFLOW'; payload: WorkflowDefinition | null }
  | { type: 'SET_CURRENT_XML'; payload: string }
  | { type: 'SET_HAS_CHANGES'; payload: boolean }
  | { type: 'SET_ACTIVE_TAB'; payload: string }
  | { type: 'SET_SAVING'; payload: boolean }
  | { type: 'SET_DEPLOYING'; payload: boolean }
  | { type: 'SET_APPROVAL_CONFIG'; payload: ApprovalConfig }
  | { type: 'SET_WORKFLOW_VERSIONS'; payload: WorkflowVersion[] }
  | { type: 'SET_USER_LIST'; payload: { id: number; name: string; username: string }[] }
  | { type: 'SET_ROLE_LIST'; payload: { id: number; name: string; code: string }[] }
  | { type: 'SET_LOADING_USERS'; payload: boolean }
  | { type: 'SET_LOADING_ROLES'; payload: boolean }
  | { type: 'UPDATE_WORKFLOW'; payload: Partial<WorkflowDefinition> }
  | { type: 'UPDATE_SLA_CONFIG'; payload: Partial<SLAConfig> }
  | { type: 'ADD_WORKFLOW_VERSION'; payload: WorkflowVersion }
  | { type: 'RESET' };

// Reducer
function workflowDesignerReducer(
  state: WorkflowDesignerContextType,
  action: WorkflowDesignerAction
): WorkflowDesignerContextType {
  switch (action.type) {
    case 'SET_WORKFLOW':
      return { ...state, workflow: action.payload };
    case 'SET_CURRENT_XML':
      return { ...state, currentXML: action.payload };
    case 'SET_HAS_CHANGES':
      return { ...state, hasChanges: action.payload };
    case 'SET_ACTIVE_TAB':
      return { ...state, activeTab: action.payload };
    case 'SET_SAVING':
      return { ...state, saving: action.payload };
    case 'SET_DEPLOYING':
      return { ...state, deploying: action.payload };
    case 'SET_APPROVAL_CONFIG':
      return { ...state, approvalConfig: action.payload };
    case 'SET_WORKFLOW_VERSIONS':
      return { ...state, workflowVersions: action.payload };
    case 'SET_USER_LIST':
      return { ...state, userList: action.payload };
    case 'SET_ROLE_LIST':
      return { ...state, roleList: action.payload };
    case 'SET_LOADING_USERS':
      return { ...state, loadingUsers: action.payload };
    case 'SET_LOADING_ROLES':
      return { ...state, loadingRoles: action.payload };
    case 'UPDATE_WORKFLOW':
      return {
        ...state,
        workflow: state.workflow ? { ...state.workflow, ...action.payload } : null,
      };
    case 'UPDATE_SLA_CONFIG':
      return {
        ...state,
        workflow: state.workflow
          ? {
              ...state.workflow,
              sla_config: state.workflow.sla_config
                ? { ...state.workflow.sla_config, ...action.payload }
                : {
                    response_time_hours: 24,
                    resolution_time_hours: 72,
                    business_hours_only: true,
                    exclude_weekends: true,
                    exclude_holidays: true,
                    ...action.payload,
                  },
            }
          : null,
      };
    case 'ADD_WORKFLOW_VERSION':
      return {
        ...state,
        workflowVersions: [...state.workflowVersions, action.payload],
      };
    case 'RESET':
      return initialState;
    default:
      return state;
  }
}

// Context
const WorkflowDesignerContext = createContext<WorkflowDesignerContextType>(initialState);

// Provider 组件
interface WorkflowDesignerProviderProps {
  children: ReactNode;
  initialWorkflow?: WorkflowDefinition | null;
  initialXML?: string;
}

export function WorkflowDesignerProvider({
  children,
  initialWorkflow = null,
  initialXML = '',
}: WorkflowDesignerProviderProps) {
  const [state, dispatch] = useReducer(workflowDesignerReducer, {
    ...initialState,
    workflow: initialWorkflow,
    currentXML: initialXML || initialWorkflow?.xml || '',
  });

  // Actions
  const setWorkflow = useCallback((workflow: WorkflowDefinition | null) => {
    dispatch({ type: 'SET_WORKFLOW', payload: workflow });
  }, []);

  const setCurrentXML = useCallback((xml: string) => {
    dispatch({ type: 'SET_CURRENT_XML', payload: xml });
  }, []);

  const setHasChanges = useCallback((hasChanges: boolean) => {
    dispatch({ type: 'SET_HAS_CHANGES', payload: hasChanges });
  }, []);

  const setActiveTab = useCallback((tab: string) => {
    dispatch({ type: 'SET_ACTIVE_TAB', payload: tab });
  }, []);

  const setSaving = useCallback((saving: boolean) => {
    dispatch({ type: 'SET_SAVING', payload: saving });
  }, []);

  const setDeploying = useCallback((deploying: boolean) => {
    dispatch({ type: 'SET_DEPLOYING', payload: deploying });
  }, []);

  const setApprovalConfig = useCallback((config: ApprovalConfig) => {
    dispatch({ type: 'SET_APPROVAL_CONFIG', payload: config });
  }, []);

  const setWorkflowVersions = useCallback((versions: WorkflowVersion[]) => {
    dispatch({ type: 'SET_WORKFLOW_VERSIONS', payload: versions });
  }, []);

  const updateWorkflow = useCallback((updates: Partial<WorkflowDefinition>) => {
    dispatch({ type: 'UPDATE_WORKFLOW', payload: updates });
  }, []);

  const updateSLAConfig = useCallback((config: Partial<SLAConfig>) => {
    dispatch({ type: 'UPDATE_SLA_CONFIG', payload: config });
  }, []);

  const addWorkflowVersion = useCallback((version: WorkflowVersion) => {
    dispatch({ type: 'ADD_WORKFLOW_VERSION', payload: version });
  }, []);

  const value: WorkflowDesignerContextType = {
    ...state,
    setWorkflow,
    setCurrentXML,
    setHasChanges,
    setActiveTab,
    setSaving,
    setDeploying,
    setApprovalConfig,
    setWorkflowVersions,
    updateWorkflow,
    updateSLAConfig,
    addWorkflowVersion,
  };

  return (
    <WorkflowDesignerContext.Provider value={value}>
      {children}
    </WorkflowDesignerContext.Provider>
  );
}

// Hook
export function useWorkflowDesigner() {
  const context = useContext(WorkflowDesignerContext);
  if (!context) {
    throw new Error('useWorkflowDesigner must be used within a WorkflowDesignerProvider');
  }
  return context;
}

// 导出
export { WorkflowDesignerContext, defaultApprovalConfig };
export type { WorkflowDesignerAction };
