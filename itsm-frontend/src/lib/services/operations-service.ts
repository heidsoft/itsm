import { httpClient } from '@/lib/api/http-client';

/**
 * 运维命令总线与通知投递重放的读写封装。
 *
 * 两个数据源是「死信人工重放」的两半：
 *   - operational_commands：命令级别的死信（BPMN 服务任务、通知投递命令、产品基线安装…），
 *     重放 = 用原始 payload 重新入箱，幂等键保持不变；
 *   - notification_deliveries：投递级别的失败（真正没送出去的那一条），
 *     重试 = 以原 payload + 新幂等键再入箱一次，会真实发送。
 */
export type CommandStatus = 'pending' | 'processing' | 'succeeded' | 'dead_letter' | 'cancelled';

export interface CommandSummary {
  pending: number;
  processing: number;
  deadLetter: number;
  cancelled: number;
  succeeded: number;
  oldestWaitingAt?: string;
  stuckLeases: number;
}

export interface CommandTypeStat {
  commandType: string;
  pending: number;
  processing: number;
  deadLetter: number;
  succeededRecent: number;
  failedRecent: number;
  sampleSize: number;
  failureRate: number;
}

export interface OperationalCommand {
  id: number;
  tenantId: number;
  commandType: string;
  aggregateType: string;
  aggregateId: number;
  idempotencyKey: string;
  status: CommandStatus;
  attempt: number;
  maxAttempts: number;
  availableAt: string;
  leaseOwner?: string;
  leaseExpiresAt?: string;
  fencingToken: number;
  lastError?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CommandPage {
  items: OperationalCommand[];
  total: number;
  page: number;
  pageSize: number;
  summary: CommandSummary;
  byType?: CommandTypeStat[];
}

export interface BulkReplayResult {
  matchedIds: number[];
  updated: number;
  skipped: number;
  pageSize: number;
}

export interface BulkReplayFilter {
  status?: CommandStatus;
  commandType?: string;
  aggregateType?: string;
  pageSize?: number;
}

export type DeliveryStatus = 'pending' | 'sent' | 'failed';

export interface NotificationDelivery {
  id: number;
  operationalCommandId: number;
  recipientId: number;
  channel: string;
  targetMasked: string;
  status: DeliveryStatus;
  attempt: number;
  ticketId?: number;
  providerMessageId?: string;
  errorCode?: string;
  errorMessage?: string;
  sentAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface DeliveryPage {
  items: NotificationDelivery[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

export interface RetryDeliveryResult {
  deliveryId: number;
  commandId: number;
  status: CommandStatus;
  retriedAt: string;
}

export interface ListCommandsParams {
  status?: CommandStatus;
  commandType?: string;
  aggregateType?: string;
  page?: number;
  pageSize?: number;
}

export interface ListDeliveriesParams {
  status?: DeliveryStatus;
  channel?: string;
  page?: number;
  pageSize?: number;
}

class OperationsService {
  async listCommands(params: ListCommandsParams = {}): Promise<CommandPage> {
    return httpClient.get<CommandPage>('/api/v1/admin/operations/commands', params);
  }

  async replayCommand(commandId: number): Promise<OperationalCommand> {
    return httpClient.post<OperationalCommand>(`/api/v1/admin/operations/commands/${commandId}/replay`);
  }

  async bulkReplayCommands(filter: BulkReplayFilter = {}): Promise<BulkReplayResult> {
    return httpClient.post<BulkReplayResult>('/api/v1/admin/operations/commands/bulk-replay', filter);
  }

  async listDeliveries(params: ListDeliveriesParams = {}): Promise<DeliveryPage> {
    return httpClient.get<DeliveryPage>('/api/v1/notification-deliveries', params);
  }

  /**
   * 重试一条失败投递。
   *
   * 服务端是幂等的：若该投递已有在途重试或上一次重试已成功，返回 409 并说明原因，
   * 页面直接透传该文案（不要折叠成「操作失败」）。
   */
  async retryDelivery(deliveryId: number): Promise<RetryDeliveryResult> {
    return httpClient.post<RetryDeliveryResult>(`/api/v1/notification-deliveries/${deliveryId}/retry`);
  }
}

const operationsService = new OperationsService();
export default operationsService;
