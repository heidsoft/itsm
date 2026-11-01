/**
 * 云信号类型：云监控告警与安全漏洞。
 * 与后端 handlers/cloudsignal/dto.go 的 camelCase DTO 逐字段一致。
 */

// 告警级别：P1/P2 会由后端自动建事件单，其余只做告警留档。
export interface CloudAlert {
  id: number;
  alertId: string;
  name: string;
  description: string;
  severity: string;
  status: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  sourceIp?: string;
  service?: string;
  firedAt: string;
  acknowledgedAt?: string;
  resolvedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface SecurityFinding {
  id: number;
  cloudAccountId: number;
  provider: string;
  vulnType: string;
  findingKey: string;
  name: string;
  aliasName?: string;
  severity: string;
  status: string;
  necessity?: string;
  instanceId?: string;
  instanceName?: string;
  uuid?: string;
  internetIp?: string;
  intranetIp?: string;
  region?: string;
  ciId?: number;
  ciNumber?: string;
  ciName?: string;
  cvssScore?: number;
  firstSeenAt?: string;
  lastSeenAt?: string;
  lastSyncedAt: string;
  resolvedAt?: string;
  createdAt: string;
  updatedAt: string;
}

// 只统计 open/fixing，已修复存量不计入。
export interface FindingSummary {
  total: number;
  bySeverity: Record<string, number>;
  byStatus: Record<string, number>;
}

// unconfigured / unavailable / ready：空列表不等于同步正常。
export interface CloudSignalSyncState {
  cloudAccountId: number;
  accountName: string;
  provider: string;
  accountActive: boolean;
  signalKind: string;
  state: string;
  lastAttemptAt?: string;
  lastSuccessAt?: string;
  watermark?: string;
  lastErrorCode?: string;
  lastErrorMessage?: string;
  itemsSynced: number;
  consecutiveFailures: number;
  updatedAt: string;
}
