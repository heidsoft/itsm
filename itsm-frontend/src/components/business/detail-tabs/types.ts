/**
 * 详情页 Tab 通用类型
 * 供 CommentPanel / AttachmentPanel / HistoryTimeline / ApprovalTimeline 五模块共用
 */

// ==================== Comment ====================

export interface CommentUser {
  id: number;
  username?: string;
  name?: string;
  email?: string;
  avatar?: string;
}

export interface CommentItem {
  id: number;
  userId: number;
  user?: CommentUser;
  content: string;
  isInternal?: boolean;
  mentions?: number[];
  attachments?: number[];
  createdAt: string;
  updatedAt?: string;
}

export interface CreateCommentInput {
  content: string;
  isInternal?: boolean;
  mentions?: number[];
  attachments?: number[];
}

export interface UpdateCommentInput {
  content?: string;
  isInternal?: boolean;
  mentions?: number[];
}

export interface CommentAdapter {
  list(targetId: number | string): Promise<{ comments: CommentItem[]; total: number }>;
  create(targetId: number | string, data: CreateCommentInput): Promise<CommentItem>;
  update?(
    targetId: number | string,
    commentId: number,
    data: UpdateCommentInput
  ): Promise<CommentItem>;
  remove(targetId: number | string, commentId: number): Promise<void>;
}

export type TargetType = 'ticket' | 'incident' | 'problem' | 'change' | 'release';

// ==================== Attachment ====================

export interface AttachmentItem {
  id: number;
  fileName: string;
  fileSize: number;
  mimeType: string;
  fileUrl?: string;
  createdAt: string;
  uploader?: CommentUser;
}

export interface AttachmentAdapter {
  list(targetId: number | string): Promise<AttachmentItem[]>;
  upload(
    targetId: number | string,
    file: File,
    onProgress?: (percent: number) => void
  ): Promise<AttachmentItem>;
  getDownloadUrl(targetId: number | string, attachmentId: number): string;
  getPreviewUrl?(targetId: number | string, attachmentId: number): string;
  remove(targetId: number | string, attachmentId: number): Promise<void>;
}

// ==================== History ====================

export interface HistoryRecord {
  id: number | string;
  user?: { name?: string; username?: string };
  action?: string;
  fieldName?: string;
  oldValue?: string;
  newValue?: string;
  changeReason?: string;
  createdAt: string;
  // audit-log 兜底字段
  path?: string;
  method?: string;
  statusCode?: number;
  ip?: string;
}

// ==================== Approval ====================

export type ApprovalStepStatus =
  | 'pending'
  | 'approved'
  | 'rejected'
  | 'delegated'
  | 'timeout'
  | 'skipped';

export interface ApprovalStep {
  id: number;
  level: number;
  step?: string;
  status: ApprovalStepStatus;
  approverId?: number;
  approverName?: string;
  comment?: string;
  processedAt?: string;
  createdAt?: string;
}

export interface ApprovalActionInput {
  comment: string;
  delegateToUserId?: number;
}
