/**
 * Enterprise UI Components
 * 企业级UI组件库
 * 
 * 基于UI设计审计报告优化的组件系统
 */

// Phase 1: 设计一致性组件
export { 
  EnterpriseCard, 
  EnterpriseStatCard, 
  EnterpriseChartCard,
  type EnterpriseCardProps,
  type EnterpriseStatCardProps,
  type EnterpriseChartCardProps,
} from './EnterpriseCard';

export { 
  EnterpriseButton, 
  EnterpriseButtonGroup, 
  EnterpriseIconButton,
  type EnterpriseButtonProps,
  type EnterpriseButtonGroupProps,
  type EnterpriseIconButtonProps,
} from './EnterpriseButton';

export { 
  EnterpriseTable, 
  EnterpriseTableToolbar,
  renderTableActions,
  type EnterpriseTableProps,
  type EnterpriseTableToolbarProps,
  type TableAction,
} from './EnterpriseTable';

// Phase 2: 交互体验组件
export { 
  SmartForm, 
  SmartFormField,
  type SmartFormProps,
  type SmartFormFieldProps,
} from './SmartForm';

export {
  SkeletonTable,
  SkeletonCardGrid,
  SkeletonForm,
  SkeletonStatCard,
  ProgressiveImage,
  LazyImage,
  SkeletonWrapper,
  type SkeletonTableProps,
  type SkeletonCardGridProps,
  type SkeletonFormProps,
  type SkeletonStatCardProps,
  type ProgressiveImageProps,
  type LazyImageProps,
  type SkeletonWrapperProps,
} from './SkeletonLoading';

export {
  showEnhancedConfirm,
  EnhancedPopconfirm,
  showBatchOperationConfirm,
  showOperationProgress,
  showOperationSuccess,
  showDataLoadError,
  useOptimisticUpdate,
  type EnhancedConfirmOptions,
  type EnhancedPopconfirmProps,
  type BatchOperationConfirmOptions,
  type OperationProgressOptions,
  type OperationSuccessOptions,
  type DataLoadErrorOptions,
  type OptimisticUpdateOptions,
} from './EnhancedFeedback';

// Phase 3: 移动端优化组件
export {
  ResponsiveTable,
  renderResponsiveStatus,
  renderResponsiveActions,
  type ResponsiveTableProps,
  type ResponsiveTableColumn,
} from './ResponsiveTable';

export {
  MobileTabBar,
  MobileAppBar,
  MobileFab,
  type MobileTabBarProps,
  type MobileTabBarItem,
  type MobileAppBarProps,
  type MobileFabProps,
} from './MobileTabBar';

// Phase 4: 无障碍组件
export {
  VisuallyHidden,
  FocusIndicator,
  AriaLiveRegion,
  SkipLink,
  AccessibleIconButton,
  AccessibleLabel,
  AccessibleDialog,
  KeyboardNavigationHint,
  type VisuallyHiddenProps,
  type FocusIndicatorProps,
  type AriaLiveRegionProps,
  type SkipLinkProps,
  type AccessibleIconButtonProps,
  type AccessibleLabelProps,
  type AccessibleDialogProps,
  type KeyboardNavigationHintProps,
} from './AccessibilityUtils';

// 现有组件（保持兼容）
export * from './Badge';
export * from './Button';
export * from './Card';
export * from './DataTable';
export * from './DatePicker';
export * from './Form';
export * from './FormField';
export * from './Input';
export * from './LoadingState';
export * from './Modal';
export * from './Select';
export * from './Toast';
export * from './UnifiedForm';
export * from './UnifiedTable';

/**
 * 组件使用指南
 * 
 * ## 企业级卡片
 * ```tsx
 * import { EnterpriseCard } from '@/components/ui';
 * 
 * <EnterpriseCard hover gradient>
 *   <h3>标题</h3>
 *   <p>内容</p>
 * </EnterpriseCard>
 * ```
 * 
 * ## 企业级按钮
 * ```tsx
 * import { EnterpriseButton } from '@/components/ui';
 * 
 * <EnterpriseButton variant="primary" icon={<PlusOutlined />}>
 *   创建工单
 * </EnterpriseButton>
 * ```
 * 
 * ## 智能表单
 * ```tsx
 * import { SmartForm, SmartFormField } from '@/components/ui';
 * 
 * <SmartForm autoSave autoSaveKey="form-draft" onFinish={handleSubmit}>
 *   <SmartFormField name="title" label="标题" required>
 *     <Input />
 *   </SmartFormField>
 * </SmartForm>
 * ```
 * 
 * ## 响应式表格
 * ```tsx
 * import { ResponsiveTable } from '@/components/ui';
 * 
 * <ResponsiveTable
 *   columns={columns}
 *   dataSource={data}
 *   onMobileCardClick={(record) => handleClick(record)}
 * />
 * ```
 * 
 * ## 移动端导航
 * ```tsx
 * import { MobileTabBar } from '@/components/ui';
 * 
 * <MobileTabBar
 *   items={[
 *     { key: 'home', path: '/', icon: <HomeOutlined />, label: '首页' },
 *   ]}
 * />
 * ```
 */
