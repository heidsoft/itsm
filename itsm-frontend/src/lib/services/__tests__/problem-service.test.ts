/**
 * ProblemService unit tests
 */
import { problemService, ProblemStatus, ProblemPriority } from '../problem-service';
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

describe('ProblemService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listProblems', () => {
    it('should call GET /api/v1/problems with params', async () => {
      const mockData = { problems: [], total: 0, page: 1, pageSize: 20 };
      mockGet.mockResolvedValueOnce(mockData);

      const result = await problemService.listProblems({ page: 2, status: ProblemStatus.OPEN });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems', { page: 2, status: 'open' });
      expect(result).toEqual(mockData);
    });

    it('should use empty params by default', async () => {
      mockGet.mockResolvedValueOnce({ problems: [], total: 0, page: 1, pageSize: 20 });
      await problemService.listProblems();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems', {});
    });
  });

  describe('getProblem', () => {
    it('should call GET /api/v1/problems/:id', async () => {
      mockGet.mockResolvedValueOnce({ id: 5, title: 'Memory leak' });
      const result = await problemService.getProblem(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/5');
      expect(result.title).toBe('Memory leak');
    });
  });

  describe('createProblem', () => {
    it('should call POST /api/v1/problems with data', async () => {
      const data = { title: 'New', description: 'Desc', priority: ProblemPriority.HIGH, category: 'infra', rootCause: 'TBD', impact: 'High' };
      mockPost.mockResolvedValueOnce({ message: 'created', problemId: 10 });
      const result = await problemService.createProblem(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/problems', data);
      expect(result.problemId).toBe(10);
    });
  });

  describe('updateProblem', () => {
    it('should call PUT /api/v1/problems/:id', async () => {
      mockPut.mockResolvedValueOnce({ message: 'updated', problemId: 3 });
      const result = await problemService.updateProblem(3, { status: ProblemStatus.RESOLVED });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/problems/3', { status: 'resolved' });
      expect(result.problemId).toBe(3);
    });
  });

  describe('deleteProblem', () => {
    it('should call DELETE /api/v1/problems/:id', async () => {
      mockDelete.mockResolvedValueOnce({ message: 'deleted', problemId: 2 });
      await problemService.deleteProblem(2);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/problems/2');
    });
  });

  describe('getProblemStats', () => {
    it('should call GET /api/v1/problems/stats', async () => {
      const stats = { total: 20, open: 5, inProgress: 8, resolved: 4, closed: 3, highPriority: 6 };
      mockGet.mockResolvedValueOnce(stats);
      const result = await problemService.getProblemStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/stats');
      expect(result.total).toBe(20);
    });
  });

  describe('addProblemComment', () => {
    it('should call POST /api/v1/problems/:id/comments', async () => {
      mockPost.mockResolvedValueOnce({ message: 'added', commentId: 99 });
      const result = await problemService.addProblemComment(5, 'Root cause found');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/problems/5/comments', { content: 'Root cause found' });
      expect(result.commentId).toBe(99);
    });
  });

  describe('getProblemComments', () => {
    it('should call GET /api/v1/problems/:id/comments', async () => {
      mockGet.mockResolvedValueOnce({ comments: [{ id: 1, content: 'Test' }], total: 1 });
      const result = await problemService.getProblemComments(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/5/comments');
      expect(result.total).toBe(1);
    });
  });

  describe('getStatusColor', () => {
    it('should return correct colors', () => {
      expect(problemService.getStatusColor(ProblemStatus.OPEN)).toBe('processing');
      expect(problemService.getStatusColor(ProblemStatus.IN_PROGRESS)).toBe('processing');
      expect(problemService.getStatusColor(ProblemStatus.RESOLVED)).toBe('success');
      expect(problemService.getStatusColor(ProblemStatus.CLOSED)).toBe('default');
      expect(problemService.getStatusColor('x' as ProblemStatus)).toBe('default');
    });
  });

  describe('getPriorityColor', () => {
    it('should return correct colors', () => {
      expect(problemService.getPriorityColor(ProblemPriority.LOW)).toBe('green');
      expect(problemService.getPriorityColor(ProblemPriority.MEDIUM)).toBe('orange');
      expect(problemService.getPriorityColor(ProblemPriority.HIGH)).toBe('red');
      expect(problemService.getPriorityColor(ProblemPriority.CRITICAL)).toBe('red');
      expect(problemService.getPriorityColor('x' as ProblemPriority)).toBe('default');
    });
  });

  describe('getStatusLabel', () => {
    it('should return Chinese labels', () => {
      expect(problemService.getStatusLabel(ProblemStatus.OPEN)).toBe('待处理');
      expect(problemService.getStatusLabel(ProblemStatus.IN_PROGRESS)).toBe('处理中');
      expect(problemService.getStatusLabel(ProblemStatus.RESOLVED)).toBe('已解决');
      expect(problemService.getStatusLabel(ProblemStatus.CLOSED)).toBe('已关闭');
      expect(problemService.getStatusLabel('other' as ProblemStatus)).toBe('other');
    });
  });

  describe('getPriorityLabel', () => {
    it('should return Chinese labels', () => {
      expect(problemService.getPriorityLabel(ProblemPriority.LOW)).toBe('低');
      expect(problemService.getPriorityLabel(ProblemPriority.MEDIUM)).toBe('中');
      expect(problemService.getPriorityLabel(ProblemPriority.HIGH)).toBe('高');
      expect(problemService.getPriorityLabel(ProblemPriority.CRITICAL)).toBe('紧急');
      expect(problemService.getPriorityLabel('x' as ProblemPriority)).toBe('x');
    });
  });

  describe('getStatusText', () => {
    it('should return English text', () => {
      expect(problemService.getStatusText(ProblemStatus.OPEN)).toBe('Open');
      expect(problemService.getStatusText(ProblemStatus.IN_PROGRESS)).toBe('In Progress');
      expect(problemService.getStatusText(ProblemStatus.RESOLVED)).toBe('Resolved');
      expect(problemService.getStatusText(ProblemStatus.CLOSED)).toBe('Closed');
      expect(problemService.getStatusText('x' as ProblemStatus)).toBe('Unknown');
    });
  });

  describe('getPriorityText', () => {
    it('should return English text', () => {
      expect(problemService.getPriorityText(ProblemPriority.LOW)).toBe('Low');
      expect(problemService.getPriorityText(ProblemPriority.MEDIUM)).toBe('Medium');
      expect(problemService.getPriorityText(ProblemPriority.HIGH)).toBe('High');
      expect(problemService.getPriorityText(ProblemPriority.CRITICAL)).toBe('Critical');
      expect(problemService.getPriorityText('x' as ProblemPriority)).toBe('Unknown');
    });
  });
});
