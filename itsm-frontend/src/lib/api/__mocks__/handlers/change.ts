/**
 * Mock handlers for Change API
 *
 * This module provides mock data and utilities for testing the ChangeApi.
 * It follows the existing Jest-based mock pattern used in the project.
 *
 * Usage in tests:
 * ```typescript
 * import { changeHandlers, createMockChange, resetMockChanges } from '@/lib/api/__mocks__/handlers/change';
 *
 * // Use changeHandlers to simulate API responses
 * const mockData = changeHandlers.getChanges({ status: 'pending' });
 *
 * // Create individual mock changes
 * const change = createMockChange({ id: 1, title: 'Test Change' });
 *
 * // Reset mock data between tests
 * beforeEach(() => resetMockChanges());
 * ```
 */

import type {
  Change,
  ChangeListResponse,
  ChangeStatsResponse,
  ChangeApproval,
  ChangeRequest,
  ChangeStatus,
  ChangeType,
  ChangePriority,
  ChangeImpact,
  ChangeRisk,
  RiskAssessmentData,
  ImpactAnalysisData,
} from '../../change-api';

// Mock change database
const mockChanges: Map<number, Change> = new Map();
let nextChangeId = 1;

// Reset function for tests
export function resetMockChanges(): void {
  mockChanges.clear();
  nextChangeId = 1;
  // Initialize with sample data
  initializeSampleChanges();
}

// Sample change data factory
export function createMockChange(overrides: Partial<Change> = {}): Change {
  const id = overrides.id ?? nextChangeId++;
  const now = new Date().toISOString();

  return {
    id,
    title: `Test Change ${id}`,
    description: 'Test change description',
    justification: 'Business need for this change',
    type: 'normal' as ChangeType,
    status: 'draft' as ChangeStatus,
    priority: 'medium' as ChangePriority,
    impactScope: 'medium' as ChangeImpact,
    riskLevel: 'low' as ChangeRisk,
    createdBy: 1,
    createdByName: 'Test User',
    tenantId: 1,
    implementationPlan: 'Step 1: Prepare\nStep 2: Execute\nStep 3: Verify',
    rollbackPlan: 'Step 1: Stop service\nStep 2: Restore backup\nStep 3: Verify',
    affectedCis: [],
    relatedTickets: [],
    createdAt: now,
    updatedAt: now,
    ...overrides,
  };
}

// Sample change data for initialization
function initializeSampleChanges(): void {
  const changes: Change[] = [
    createMockChange({
      id: 1,
      title: 'Database Server Upgrade',
      description: 'Upgrade PostgreSQL from 14 to 15',
      justification: 'Performance improvements and new features',
      type: 'normal',
      status: 'pending',
      priority: 'high',
      impactScope: 'high',
      riskLevel: 'medium',
      assigneeId: 2,
      assigneeName: 'John Doe',
      createdBy: 1,
      createdByName: 'Admin User',
      plannedStartDate: '2024-02-15T10:00:00Z',
      plannedEndDate: '2024-02-15T14:00:00Z',
      affectedCis: ['server-db-01', 'server-db-02'],
      relatedTickets: ['TICKET-101'],
    }),
    createMockChange({
      id: 2,
      title: 'Emergency Security Patch',
      description: 'Apply critical security patch to web servers',
      justification: 'CVE-2024-1234 requires immediate patching',
      type: 'emergency',
      status: 'approved',
      priority: 'critical',
      impactScope: 'high',
      riskLevel: 'high',
      assigneeId: 3,
      assigneeName: 'Jane Smith',
      createdBy: 2,
      createdByName: 'Security Team',
      affectedCis: ['server-web-01', 'server-web-02', 'server-web-03'],
      relatedTickets: [],
    }),
    createMockChange({
      id: 3,
      title: 'Network Switch Replacement',
      description: 'Replace aging network switch in data center',
      justification: 'Hardware end-of-life',
      type: 'standard',
      status: 'completed',
      priority: 'low',
      impactScope: 'medium',
      riskLevel: 'low',
      assigneeId: 4,
      assigneeName: 'Bob Wilson',
      createdBy: 1,
      createdByName: 'Admin User',
      actualStartDate: '2024-01-20T08:00:00Z',
      actualEndDate: '2024-01-20T12:00:00Z',
      affectedCis: ['switch-core-01'],
      relatedTickets: [],
    }),
  ];

  changes.forEach(change => {
    mockChanges.set(change.id, change);
  });
}

// Initialize on module load
initializeSampleChanges();

// Mock response helper
function createMockResponse<T>(
  data: T,
  code: number = 0,
  message: string = 'success'
): { code: number; message: string; data: T } {
  return { code, message, data };
}

// Handler functions that simulate API responses
export const changeHandlers = {
  // GET /api/v1/changes - Get change list
  getChanges: (params?: {
    page?: number;
    page_size?: number;
    status?: ChangeStatus;
    type?: ChangeType;
    priority?: ChangePriority;
    search?: string;
  }): ChangeListResponse => {
    let changes = Array.from(mockChanges.values());

    // Apply filters
    if (params?.status) {
      changes = changes.filter(c => c.status === params.status);
    }
    if (params?.type) {
      changes = changes.filter(c => c.type === params.type);
    }
    if (params?.priority) {
      changes = changes.filter(c => c.priority === params.priority);
    }
    if (params?.search) {
      const searchLower = params.search.toLowerCase();
      changes = changes.filter(
        c =>
          c.title.toLowerCase().includes(searchLower) ||
          c.description.toLowerCase().includes(searchLower)
      );
    }

    // Pagination
    const page = params?.page ?? 1;
    const pageSize = params?.page_size ?? 20;
    const startIndex = (page - 1) * pageSize;
    const paginatedChanges = changes.slice(startIndex, startIndex + pageSize);

    return {
      total: changes.length,
      changes: paginatedChanges,
    };
  },

  // GET /api/v1/changes/:id - Get single change
  getChange: (id: number): Change | null => {
    return mockChanges.get(id) ?? null;
  },

  // POST /api/v1/changes - Create change
  createChange: (data: ChangeRequest): Change => {
    const change = createMockChange({
      title: data.title,
      description: data.description,
      justification: data.justification,
      type: data.type,
      priority: data.priority,
      impactScope: data.impactScope,
      riskLevel: data.riskLevel,
      plannedStartDate: data.plannedStartDate,
      plannedEndDate: data.plannedEndDate,
      implementationPlan: data.implementationPlan,
      rollbackPlan: data.rollbackPlan,
      affectedCis: data.affectedCis,
      relatedTickets: data.relatedTickets,
    });

    mockChanges.set(change.id, change);
    return change;
  },

  // PUT /api/v1/changes/:id - Update change
  updateChange: (id: number, data: Partial<ChangeRequest>): Change | null => {
    const existing = mockChanges.get(id);
    if (!existing) return null;

    const updated: Change = {
      ...existing,
      ...data,
      updatedAt: new Date().toISOString(),
    };

    mockChanges.set(id, updated);
    return updated;
  },

  // DELETE /api/v1/changes/:id - Delete change
  deleteChange: (id: number): boolean => {
    return mockChanges.delete(id);
  },

  // POST /api/v1/changes/:id/submit - Submit for approval
  submitForApproval: (id: number): boolean => {
    const change = mockChanges.get(id);
    if (!change) return false;
    if (change.status !== 'draft') return false;

    change.status = 'pending';
    change.updatedAt = new Date().toISOString();
    mockChanges.set(id, change);
    return true;
  },

  // POST /api/v1/changes/:id/approve - Approve change
  approveChange: (id: number, comment?: string): boolean => {
    const change = mockChanges.get(id);
    if (!change) return false;
    if (change.status !== 'pending') return false;

    change.status = 'approved';
    change.updatedAt = new Date().toISOString();
    mockChanges.set(id, change);
    return true;
  },

  // POST /api/v1/changes/:id/reject - Reject change
  rejectChange: (id: number, comment?: string): boolean => {
    const change = mockChanges.get(id);
    if (!change) return false;
    if (change.status !== 'pending') return false;

    change.status = 'rejected';
    change.updatedAt = new Date().toISOString();
    mockChanges.set(id, change);
    return true;
  },

  // GET /api/v1/changes/stats - Get change statistics
  getChangeStats: (): ChangeStatsResponse => {
    const changes = Array.from(mockChanges.values());
    return {
      total: changes.length,
      pending: changes.filter(c => c.status === 'pending').length,
      approved: changes.filter(c => c.status === 'approved').length,
      inProgress: changes.filter(c => c.status === 'in_progress').length,
      completed: changes.filter(c => c.status === 'completed').length,
      rolledBack: changes.filter(c => c.status === 'rolled_back').length,
      rejected: changes.filter(c => c.status === 'rejected').length,
      cancelled: changes.filter(c => c.status === 'cancelled').length,
    };
  },

  // POST /api/v1/changes/:id/start - Start implementation
  startImplementation: (id: number): boolean => {
    const change = mockChanges.get(id);
    if (!change) return false;
    if (change.status !== 'approved') return false;

    change.status = 'in_progress';
    change.actualStartDate = new Date().toISOString();
    change.updatedAt = new Date().toISOString();
    mockChanges.set(id, change);
    return true;
  },

  // POST /api/v1/changes/:id/complete - Complete implementation
  completeImplementation: (id: number): boolean => {
    const change = mockChanges.get(id);
    if (!change) return false;
    if (change.status !== 'in_progress') return false;

    change.status = 'completed';
    change.actualEndDate = new Date().toISOString();
    change.updatedAt = new Date().toISOString();
    mockChanges.set(id, change);
    return true;
  },

  // POST /api/v1/changes/:id/rollback - Rollback change
  rollbackChange: (id: number, reason?: string): boolean => {
    const change = mockChanges.get(id);
    if (!change) return false;
    if (!['in_progress', 'completed'].includes(change.status)) return false;

    change.status = 'rolled_back';
    change.updatedAt = new Date().toISOString();
    mockChanges.set(id, change);
    return true;
  },

  // POST /api/v1/changes/:id/cancel - Cancel change
  cancelChange: (id: number, reason?: string): boolean => {
    const change = mockChanges.get(id);
    if (!change) return false;
    if (['completed', 'cancelled', 'rolled_back'].includes(change.status)) return false;

    change.status = 'cancelled';
    change.updatedAt = new Date().toISOString();
    mockChanges.set(id, change);
    return true;
  },

  // POST /api/v1/changes/:id/assign - Assign change
  assignChange: (id: number, assigneeId: number, assigneeName: string): boolean => {
    const change = mockChanges.get(id);
    if (!change) return false;

    change.assigneeId = assigneeId;
    change.assigneeName = assigneeName;
    change.updatedAt = new Date().toISOString();
    mockChanges.set(id, change);
    return true;
  },

  // GET /api/v1/changes/:id/approvals - Get approval history
  getChangeApprovals: (id: number): ChangeApproval[] => {
    const change = mockChanges.get(id);
    if (!change) return [];

    // Return mock approval history
    const approvals: ChangeApproval[] = [];

    if (
      change.status === 'approved' ||
      change.status === 'in_progress' ||
      change.status === 'completed'
    ) {
      approvals.push({
        id: 1,
        changeId: id,
        approverId: 10,
        approverName: 'CAB Chair',
        status: 'approved',
        comment: 'Approved with conditions',
        approvedAt: new Date().toISOString(),
        createdAt: new Date().toISOString(),
      });
    }

    return approvals;
  },
};

// Export mock data for direct access in tests
export { mockChanges };

// Export response helpers
export { createMockResponse };
