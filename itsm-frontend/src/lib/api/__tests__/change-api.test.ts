import { ChangeApi, type ChangeRequest } from '@/lib/api/change-api';

// Mock security module to avoid CSRF fetch calls
jest.mock('@/lib/security', () => ({
  security: {
    csrf: {
      getToken: jest.fn().mockResolvedValue('mock-csrf-token'),
      clearToken: jest.fn(),
    },
    network: {
      getSecureHeaders: jest.fn().mockReturnValue({
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
      }),
    },
  },
}));

// Mock fetch globally
global.fetch = jest.fn();

// Mock console methods to avoid noise in tests
const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

describe('ChangeApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (fetch as jest.Mock).mockClear();
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  describe('getChanges', () => {
    it('should fetch changes successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          changes: [
            {
              id: 1,
              title: 'Database Server Upgrade',
              description: 'Upgrade PostgreSQL from 14 to 15',
              justification: 'Performance improvements',
              type: 'normal',
              status: 'pending',
              priority: 'high',
              impactScope: 'high',
              riskLevel: 'medium',
              assigneeId: 2,
              assigneeName: 'John Doe',
              createdBy: 1,
              createdByName: 'Admin User',
              tenantId: 1,
              implementationPlan: 'Step 1: Prepare\nStep 2: Execute',
              rollbackPlan: 'Rollback steps',
              affectedCis: ['server-db-01'],
              relatedTickets: [],
              createdAt: '2024-01-01T10:00:00Z',
              updatedAt: '2024-01-01T10:00:00Z',
            },
          ],
          total: 1,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.getChanges();

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.changes).toHaveLength(1);
      expect(result.changes[0].title).toBe('Database Server Upgrade');
    });

    it('should handle empty change list', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          changes: [],
          total: 0,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.getChanges();

      expect(result.changes).toHaveLength(0);
      expect(result.total).toBe(0);
    });

    it('should pass query parameters correctly', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: { changes: [], total: 0 },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.getChanges({ page: 2, pageSize: 10, status: 'pending' });

      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('page=2'), expect.any(Object));
    });

    it('should handle API error response', async () => {
      const mockResponse = {
        code: 5001,
        message: 'Internal server error',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await expect(ChangeApi.getChanges({})).rejects.toThrow('Internal server error');
    });
  });

  describe('getChange', () => {
    it('should fetch single change successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          title: 'Test Change',
          description: 'Test description',
          justification: 'Test justification',
          type: 'normal',
          status: 'draft',
          priority: 'medium',
          impactScope: 'low',
          riskLevel: 'low',
          createdBy: 1,
          createdByName: 'Test User',
          tenantId: 1,
          implementationPlan: 'Plan',
          rollbackPlan: 'Rollback',
          affectedCis: [],
          relatedTickets: [],
          createdAt: '2024-01-01T10:00:00Z',
          updatedAt: '2024-01-01T10:00:00Z',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.getChange(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.id).toBe(1);
      expect(result.title).toBe('Test Change');
    });

    it('should handle change not found', async () => {
      const mockResponse = {
        code: 4004,
        message: 'Change not found',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await expect(ChangeApi.getChange(999)).rejects.toBeDefined();
    });
  });

  describe('createChange', () => {
    it('should create change successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 10,
          title: 'New Change Request',
          description: 'Description for new change',
          justification: 'Business justification',
          type: 'normal',
          status: 'draft',
          priority: 'high',
          impactScope: 'medium',
          riskLevel: 'low',
          createdBy: 1,
          createdByName: 'Test User',
          tenantId: 1,
          implementationPlan: 'Implementation steps',
          rollbackPlan: 'Rollback steps',
          affectedCis: ['server-01'],
          relatedTickets: [],
          createdAt: '2024-01-01T10:00:00Z',
          updatedAt: '2024-01-01T10:00:00Z',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 201,
        json: async () => mockResponse,
      });

      const newChange: ChangeRequest = {
        title: 'New Change Request',
        description: 'Description for new change',
        justification: 'Business justification',
        type: 'normal',
        priority: 'high',
        impactScope: 'medium',
        riskLevel: 'low',
        implementationPlan: 'Implementation steps',
        rollbackPlan: 'Rollback steps',
        affectedCis: ['server-01'],
        relatedTickets: [],
      };

      const result = await ChangeApi.createChange(newChange);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes'),
        expect.objectContaining({
          method: 'POST',
        })
      );

      expect(result.title).toBe('New Change Request');
    });
  });

  describe('updateChange', () => {
    it('should update change successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          title: 'Updated Change',
          description: 'Test description',
          justification: 'Test justification',
          type: 'normal',
          status: 'draft',
          priority: 'critical',
          impactScope: 'medium',
          riskLevel: 'low',
          createdBy: 1,
          createdByName: 'Test User',
          tenantId: 1,
          implementationPlan: 'Plan',
          rollbackPlan: 'Rollback',
          affectedCis: [],
          relatedTickets: [],
          createdAt: '2024-01-01T10:00:00Z',
          updatedAt: '2024-01-02T10:00:00Z',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.updateChange(1, { title: 'Updated Change' });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1'),
        expect.objectContaining({
          method: 'PUT',
        })
      );

      expect(result.title).toBe('Updated Change');
    });
  });

  describe('deleteChange', () => {
    it('should delete change successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.deleteChange(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1'),
        expect.objectContaining({
          method: 'DELETE',
        })
      );
    });
  });

  describe('submitForApproval', () => {
    it('should submit change for approval', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.submitForApproval(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/submit'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('approveChange', () => {
    it('should approve change', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.approveChange(1, { status: 'approved', comment: 'LGTM' });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/approve'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('rejectChange', () => {
    it('should reject change', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.rejectChange(1, { status: 'rejected', comment: 'Not approved' });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/reject'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('getChangeStats', () => {
    it('should fetch change statistics', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          total: 10,
          pending: 3,
          approved: 2,
          inProgress: 1,
          completed: 3,
          rolledBack: 0,
          rejected: 1,
          cancelled: 0,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.getChangeStats();

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/stats'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.total).toBe(10);
      expect(result.pending).toBe(3);
    });
  });

  describe('startImplementation', () => {
    it('should start implementation', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.startImplementation(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/start'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('completeImplementation', () => {
    it('should complete implementation', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.completeImplementation(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/complete'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('rollbackChange', () => {
    it('should rollback change with reason', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.rollbackChange(1, 'Unexpected issues occurred');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/rollback'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('cancelChange', () => {
    it('should cancel change', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.cancelChange(1, 'No longer needed');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/cancel'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('assignChange', () => {
    it('should assign change to user', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.assignChange(1, 42);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/assign'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('getChangeApprovals', () => {
    it('should fetch approval history', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: [
          {
            id: 1,
            changeId: 1,
            approverId: 10,
            approverName: 'CAB Chair',
            status: 'approved',
            comment: 'Approved',
            approvedAt: '2024-01-15T10:00:00Z',
            createdAt: '2024-01-15T09:00:00Z',
          },
        ],
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.getChangeApprovals(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/approvals'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result).toHaveLength(1);
      expect(result[0].approverName).toBe('CAB Chair');
    });
  });

  describe('getImpactAnalysis', () => {
    it('should fetch impact analysis', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          businessImpact: 'High impact on customer-facing services',
          technicalImpact: 'Database downtime required',
          userImpact: 'Service unavailable during maintenance',
          affectedSystems: ['web-app', 'api-gateway'],
          affectedUsers: 5000,
          estimatedDowntime: 120,
          dataRiskLevel: 'medium',
          serviceDependencies: ['auth-service', 'cache-service'],
          backupStrategy: 'Full backup before change',
          recoveryPlan: 'Restore from backup if needed',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.getImpactAnalysis(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/impact'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.businessImpact).toBe('High impact on customer-facing services');
    });
  });

  describe('getRiskAssessment', () => {
    it('should fetch risk assessment', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          riskLevel: 'high',
          riskDescription: 'Database upgrade has inherent risks',
          impactAnalysis: 'Data loss possible if backup fails',
          mitigationMeasures: 'Multiple backup copies, dry-run in staging',
          contingencyPlan: 'Rollback to previous version',
          riskOwner: 'DBA Team Lead',
          riskScore: 75,
          riskFactors: ['Data migration complexity', 'Limited rollback window'],
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.getRiskAssessment(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/risk'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.riskLevel).toBe('high');
    });
  });

  describe('batchUpdateChanges', () => {
    it('should batch update changes', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      await ChangeApi.batchUpdateChanges([1, 2, 3], 'cancel', { reason: 'Bulk cancel' });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/batch'),
        expect.objectContaining({
          method: 'POST',
        })
      );
    });
  });

  describe('getApprovalSummary (deprecated alias)', () => {
    it('should call getChangeApprovals internally', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: [],
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        status: 200,
        json: async () => mockResponse,
      });

      const result = await ChangeApi.getApprovalSummary(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/changes/1/approvals'),
        expect.any(Object)
      );

      expect(result).toEqual([]);
    });
  });
});
