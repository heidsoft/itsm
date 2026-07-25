import { smartAssignmentService, SkillLevel, AssignmentStrategy } from '../smart-assignment-service';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('SmartAssignmentService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getAssignmentSuggestions', () => {
    it('should return suggestions from API', async () => {
      const suggestions = [{ userId: 1, userName: 'Alice', score: 90, reasons: ['Skill match'] }];
      mockPost.mockResolvedValue({ suggestions });

      const result = await smartAssignmentService.getAssignmentSuggestions({
        ticketId: 100,
        ticketType: 'incident',
        category: 'network',
        priority: 'high',
        description: 'Network down',
      });

      expect(mockPost).toHaveBeenCalledWith(
        '/api/v1/tickets/assign-recommendations/100',
        expect.objectContaining({ ticketId: 100 })
      );
      expect(result).toEqual(suggestions);
    });

    it('should return empty array when no suggestions', async () => {
      mockPost.mockResolvedValue({ suggestions: null });

      const result = await smartAssignmentService.getAssignmentSuggestions({
        ticketId: 200,
        ticketType: 'request',
        category: 'general',
        priority: 'low',
        description: 'test',
      });

      expect(result).toEqual([]);
    });
  });

  describe('testAssignmentRule', () => {
    it('should test a specific rule', async () => {
      const suggestions = [{ userId: 2, userName: 'Bob', score: 80 }];
      mockPost.mockResolvedValue({ suggestions });

      const result = await smartAssignmentService.testAssignmentRule({
        ruleId: 5,
        ticketId: 100,
        ticketType: 'incident',
        category: 'network',
        priority: 'high',
        description: 'test rule',
      });

      expect(mockPost).toHaveBeenCalledWith(
        '/api/v1/tickets/assignment-rules/test',
        expect.objectContaining({ ruleId: 5 })
      );
      expect(result).toEqual(suggestions);
    });
  });

  describe('assignTicket', () => {
    it('should return success when suggestions exist', async () => {
      const suggestions = [{ userId: 1, userName: 'Alice', score: 90 }];
      mockPost.mockResolvedValue({ suggestions });

      const result = await smartAssignmentService.assignTicket({
        ticketId: 100,
        ticketType: 'incident',
        category: 'network',
        priority: 'high',
        description: 'Network down',
      });

      expect(result.success).toBe(true);
      expect(result.suggestions).toEqual(suggestions);
    });

    it('should return failure when no suggestions', async () => {
      mockPost.mockResolvedValue({ suggestions: null });

      const result = await smartAssignmentService.assignTicket({
        ticketId: 100,
        ticketType: 'incident',
        category: 'network',
        priority: 'high',
        description: 'Network down',
      });

      expect(result.success).toBe(false);
      expect(result.suggestions).toEqual([]);
    });
  });

  describe('getUserSkills', () => {
    it('should return empty array (not implemented)', async () => {
      const result = await smartAssignmentService.getUserSkills(1);
      expect(result).toEqual([]);
    });
  });

  describe('updateUserSkill', () => {
    it('should throw not implemented error', async () => {
      await expect(smartAssignmentService.updateUserSkill(1, 2, {})).rejects.toThrow(
        'User skills API not implemented'
      );
    });
  });

  describe('getUserWorkload', () => {
    it('should throw not implemented error', async () => {
      await expect(smartAssignmentService.getUserWorkload(1)).rejects.toThrow(
        'User workload API not implemented'
      );
    });
  });

  describe('getAllUserWorkloads', () => {
    it('should return empty array (not implemented)', async () => {
      const result = await smartAssignmentService.getAllUserWorkloads();
      expect(result).toEqual([]);
    });
  });

  describe('getAssignmentRules', () => {
    it('should fetch assignment rules', async () => {
      const rules = [{ id: 1, name: 'Rule 1', strategy: 'skill_based', isActive: true }];
      mockGet.mockResolvedValue(rules);

      const result = await smartAssignmentService.getAssignmentRules();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules');
      expect(result).toEqual(rules);
    });

    it('should return empty array when response is null', async () => {
      mockGet.mockResolvedValue(null);

      const result = await smartAssignmentService.getAssignmentRules();

      expect(result).toEqual([]);
    });
  });

  describe('createAssignmentRule', () => {
    it('should create a new rule', async () => {
      const ruleData = { name: 'New Rule', strategy: AssignmentStrategy.SKILL_BASED, priority: 1, conditions: [], actions: [], isActive: true, applicableTo: { ticketTypes: [], categories: [], priorities: [], departments: [] } };
      const response = { id: 1, ...ruleData, createdAt: '2024-01-01', updatedAt: '2024-01-01' };
      mockPost.mockResolvedValue(response);

      const result = await smartAssignmentService.createAssignmentRule(ruleData);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules', ruleData);
      expect(result.id).toBe(1);
    });
  });

  describe('updateAssignmentRule', () => {
    it('should update an existing rule', async () => {
      const updateData = { name: 'Updated Rule' };
      const response = { id: 1, name: 'Updated Rule' };
      mockPut.mockResolvedValue(response);

      const result = await smartAssignmentService.updateAssignmentRule(1, updateData);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules/1', updateData);
      expect(result.name).toBe('Updated Rule');
    });
  });

  describe('deleteAssignmentRule', () => {
    it('should delete a rule', async () => {
      mockDelete.mockResolvedValue(undefined);

      await smartAssignmentService.deleteAssignmentRule(1);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules/1');
    });
  });

  describe('getAssignmentHistory', () => {
    it('should throw not implemented error', async () => {
      await expect(smartAssignmentService.getAssignmentHistory()).rejects.toThrow(
        'Assignment history API not implemented'
      );
    });
  });

  describe('getAssignmentStats', () => {
    it('should throw not implemented error', async () => {
      await expect(smartAssignmentService.getAssignmentStats()).rejects.toThrow(
        'Assignment stats API not implemented'
      );
    });
  });

  describe('calculateSkillMatch', () => {
    it('should return 100 when no required skills', () => {
      const result = smartAssignmentService.calculateSkillMatch([], []);
      expect(result).toBe(100);
    });

    it('should return 0 when user has no matching skills', () => {
      const result = smartAssignmentService.calculateSkillMatch(['python', 'docker'], []);
      expect(result).toBe(0);
    });

    it('should calculate score based on skill level, experience, and success rate', () => {
      const userSkills = [
        { id: 1, userId: 1, skillId: 1, skillName: 'Python', skillCategory: 'programming', level: SkillLevel.EXPERT, experience: 80, successRate: 0.9, createdAt: '', updatedAt: '' },
      ];
      const result = smartAssignmentService.calculateSkillMatch(['python'], userSkills);
      expect(result).toBeGreaterThan(0);
      expect(result).toBeLessThanOrEqual(100);
    });

    it('should handle partial skill matches', () => {
      const userSkills = [
        { id: 1, userId: 1, skillId: 1, skillName: 'Python', skillCategory: 'programming', level: SkillLevel.INTERMEDIATE, experience: 50, successRate: 0.7, createdAt: '', updatedAt: '' },
      ];
      const result = smartAssignmentService.calculateSkillMatch(['python', 'docker'], userSkills);
      expect(result).toBeGreaterThan(0);
      expect(result).toBeLessThan(100);
    });
  });

  describe('calculateWorkloadScore', () => {
    it('should calculate workload score', () => {
      const workload = {
        userId: 1,
        userName: 'Alice',
        activeTickets: 5,
        pendingTickets: 2,
        completedToday: 3,
        averageResolutionTime: 60,
        currentUtilization: 0.5,
        maxCapacity: 10,
        availableCapacity: 5,
        lastUpdated: '',
      };
      const result = smartAssignmentService.calculateWorkloadScore(workload);
      expect(result).toBeGreaterThan(0);
      expect(result).toBeLessThanOrEqual(100);
    });
  });

  describe('calculateGeographicScore', () => {
    it('should return 100 when no ticket location', () => {
      const result = smartAssignmentService.calculateGeographicScore('Beijing', '');
      expect(result).toBe(100);
    });

    it('should return 100 for exact match', () => {
      const result = smartAssignmentService.calculateGeographicScore('Beijing', 'Beijing');
      expect(result).toBe(100);
    });

    it('should return 80 for partial match', () => {
      const result = smartAssignmentService.calculateGeographicScore('Beijing Haidian', 'Beijing');
      expect(result).toBe(80);
    });

    it('should return 50 for no match', () => {
      const result = smartAssignmentService.calculateGeographicScore('Shanghai', 'Beijing');
      expect(result).toBe(50);
    });
  });

  describe('calculateOverallScore', () => {
    it('should weight by skill-based strategy', () => {
      const result = smartAssignmentService.calculateOverallScore(100, 50, 50, AssignmentStrategy.SKILL_BASED);
      expect(result).toBe(100 * 0.6 + 50 * 0.2 + 50 * 0.2);
    });

    it('should weight by workload-balanced strategy', () => {
      const result = smartAssignmentService.calculateOverallScore(50, 100, 50, AssignmentStrategy.WORKLOAD_BALANCED);
      expect(result).toBe(50 * 0.2 + 100 * 0.6 + 50 * 0.2);
    });

    it('should weight by geographic strategy', () => {
      const result = smartAssignmentService.calculateOverallScore(50, 50, 100, AssignmentStrategy.GEOGRAPHIC);
      expect(result).toBe(50 * 0.2 + 50 * 0.2 + 100 * 0.6);
    });

    it('should use default weights for round-robin strategy', () => {
      const result = smartAssignmentService.calculateOverallScore(60, 70, 80, AssignmentStrategy.ROUND_ROBIN);
      expect(result).toBe(60 * 0.4 + 70 * 0.3 + 80 * 0.3);
    });
  });

  describe('estimateResolutionTime', () => {
    it('should estimate resolution time based on complexity', () => {
      const workload = {
        userId: 1, userName: 'Alice', activeTickets: 3, pendingTickets: 1, completedToday: 2,
        averageResolutionTime: 120, currentUtilization: 0.5, maxCapacity: 10, availableCapacity: 5, lastUpdated: '',
      };
      const resultLow = smartAssignmentService.estimateResolutionTime([], workload, 'low');
      const resultHigh = smartAssignmentService.estimateResolutionTime([], workload, 'high');
      expect(resultLow).toBeLessThan(resultHigh);
    });
  });

  describe('calculateSuccessProbability', () => {
    it('should return a probability between 0 and 1', () => {
      const workload = {
        userId: 1, userName: 'Alice', activeTickets: 3, pendingTickets: 1, completedToday: 2,
        averageResolutionTime: 120, currentUtilization: 0.3, maxCapacity: 10, availableCapacity: 7, lastUpdated: '',
      };
      const result = smartAssignmentService.calculateSuccessProbability([], workload, 'medium');
      expect(result).toBeGreaterThanOrEqual(0);
      expect(result).toBeLessThanOrEqual(1);
    });
  });
});
