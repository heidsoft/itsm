// 工作流设计器组件导出
// Workflow Designer Components Export

export { default as WorkflowDesigner } from './WorkflowDesigner';
export { default as WorkflowToolbar } from './WorkflowToolbar';
export { default as WorkflowCanvas } from './WorkflowCanvas';
export { default as WorkflowProperties } from './WorkflowProperties';
export { default as WorkflowNodePalette } from './WorkflowNodePalette';
export { default as WorkflowNewModal } from './WorkflowNewModal';
export { default as WorkflowVersionModal } from './WorkflowVersionModal';
export { default as WorkflowSettingsModal } from './WorkflowSettingsModal';
export { default as WorkflowMetadataModal } from './WorkflowMetadataModal';

export { WorkflowDesignerProvider, useWorkflowDesigner } from './WorkflowContext';

export type {
  WorkflowDefinition,
  ApprovalConfig,
  EscalationRule,
  WorkflowVariable,
  SLAConfig,
  WorkflowVersion,
  UserInfo,
  RoleInfo,
  WorkflowDesignerState,
  WorkflowDesignerActions,
  WorkflowDesignerContextType,
  WorkflowTemplate,
  BPMNDesignerProps,
} from './WorkflowTypes';
