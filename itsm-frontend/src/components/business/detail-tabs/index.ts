export { CommentPanel } from './CommentPanel';
export type { CommentPanelProps } from './CommentPanel';

export { AttachmentPanel } from './AttachmentPanel';
export type { AttachmentPanelProps } from './AttachmentPanel';

export { HistoryTimeline } from './HistoryTimeline';
export type { HistoryTimelineProps } from './HistoryTimeline';

export { ApprovalTimeline } from './ApprovalTimeline';
export type { ApprovalTimelineProps } from './ApprovalTimeline';

export { ApprovalWorkflowPanel } from './ApprovalWorkflowPanel';
export type { ApprovalWorkflowPanelProps } from './ApprovalWorkflowPanel';

export * from './types';

// Adapters
export { ticketCommentAdapter } from './adapters/ticket-comment-adapter';
export { ticketAttachmentAdapter } from './adapters/ticket-attachment-adapter';
export { fetchAuditLogHistory } from './adapters/audit-log-history-adapter';
export { incidentCommentAdapter } from './adapters/incident-comment-adapter';
