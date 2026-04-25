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
    Object.values(consoleSpy).forEach((spy) => spy.mockRestore());
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
              impact_scope: 'high',
              risk_level: 'medium',
              assignee_id: 2,
              assignee_name: 'John Doe',
              created_by: 1,
              created_by_name: 'Admin User',
              tenant_id: 1,
              implementation_plan: 'Step 1: Prepare\nStep 2: Execute',
              rollback_plan: 'Rollback steps',
              affected_cis: ['server-db-01'],
              related_tickets: [],
              created_at: '2024-01-01T10:00:00Z',
              updated_at: '2024-01-01T10:00:00Z',
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

      await ChangeApi.getChanges({ page: 2, page_size: 10, status: 'pending' });

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
          impact_scope: 'low',
          risk_level: 'low',
          created_by: 1,
          created_by_name: 'Test User',
          tenant_id: 1,
          implementation_plan: 'Plan',
          rollback_plan: 'Rollback',
          affected_cis: [],
          related_tickets: [],
          created_at: '2024-01-01T10:00:00Z',
          updated_at: '2024-01-01T10:00:00Z',
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
          impact_scope: 'medium',
          risk_level: 'low',
          created_by: 1,
          created_by_name: 'Test User',
          tenant_id: 1,
          implementation_plan: 'Implementation steps',
          rollback_plan: 'Rollback steps',
          affected_cis: ['server-01'],
          related_tickets: [],
          created_at: '2024-01-01T10:00:00Z',
          updated_at: '2024-01-01T10:00:00Z',
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
          impact_scope: 'medium',
          risk_level: 'low',
          created_by: 1,
          created_by_name: 'Test User',
          tenant_id: 1,
          implementation_plan: 'Plan',
          rollback_plan: 'Rollback',
          affected_cis: [],
          related_tickets: [],
          created_at: '2024-01-01T10:00:00Z',
          updated_at: '2024-01-02T10:00:00Z',
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
          in_progress: 1,
          completed: 3,
          rolled_back: 0,
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
            change_id: 1,
            approver_id: 10,
            approver_name: 'CAB Chair',
            status: 'approved',
            comment: 'Approved',
            approved_at: '2024-01-15T10:00:00Z',
            created_at: '2024-01-15T09:00:00Z',
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
          business_impact: 'High impact on customer-facing services',
          technical_impact: 'Database downtime required',
          user_impact: 'Service unavailable during maintenance',
          affected_systems: ['web-app', 'api-gateway'],
          affected_users: 5000,
          estimated_downtime: 120,
          data_risk_level: 'medium',
          service_dependencies: ['auth-service', 'cache-service'],
          backup_strategy: 'Full backup before change',
          recovery_plan: 'Restore from backup if needed',
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
          risk_level: 'high',
          risk_description: 'Database upgrade has inherent risks',
          impact_analysis: 'Data loss possible if backup fails',
          mitigation_measures: 'Multiple backup copies, dry-run in staging',
          contingency_plan: 'Rollback to previous version',
          risk_owner: 'DBA Team Lead',
          risk_score: 75,
          risk_factors: ['Data migration complexity', 'Limited rollback window'],
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
