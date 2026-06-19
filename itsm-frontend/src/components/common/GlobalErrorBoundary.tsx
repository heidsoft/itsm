/**
 * GlobalErrorBoundary — 已废弃
 *
 * 请使用 @/components/common/ErrorBoundary 替代。
 * 此文件仅保留 re-export 以兼容现有导入。
 */

export { ErrorBoundary as GlobalErrorBoundary, withErrorBoundary } from './ErrorBoundary';

// 默认导出兼容
export { ErrorBoundary as default } from './ErrorBoundary';
