import { departmentService } from '../department-service';
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

describe('DepartmentService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getDepartmentTree', () => {
    it('should fetch department tree and normalize data', async () => {
      const rawData = [
        {
          id: 1,
          name: 'Engineering',
          code: 'ENG',
          description: 'Engineering dept',
          managerId: 10,
          parentId: undefined,
          children: [
            { id: 2, name: 'Frontend', code: 'FE', parentId: 1 },
          ],
          createdAt: '2024-01-01',
          updatedAt: '2024-01-02',
        },
      ];
      mockGet.mockResolvedValue(rawData);

      const result = await departmentService.getDepartmentTree();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/org/departments/tree');
      expect(result).toHaveLength(1);
      expect(result[0].id).toBe(1);
      expect(result[0].name).toBe('Engineering');
      expect(result[0].children).toHaveLength(1);
      expect(result[0].children![0].id).toBe(2);
      expect(result[0].children![0].name).toBe('Frontend');
    });

    it('should handle empty tree', async () => {
      mockGet.mockResolvedValue([]);

      const result = await departmentService.getDepartmentTree();

      expect(result).toEqual([]);
    });

    it('should normalize missing fields with defaults', async () => {
      mockGet.mockResolvedValue([{ }]);

      const result = await departmentService.getDepartmentTree();

      expect(result[0].id).toBe(0);
      expect(result[0].name).toBe('');
      expect(result[0].code).toBe('');
    });
  });

  describe('createDepartment', () => {
    it('should create a department', async () => {
      const request = { name: 'HR', code: 'HR', description: 'Human Resources' };
      const response = { id: 3, ...request };
      mockPost.mockResolvedValue(response);

      const result = await departmentService.createDepartment(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/org/departments', request);
      expect(result).toEqual(response);
    });

    it('should create a department with parentId', async () => {
      const request = { name: 'Payroll', code: 'PAY', parentId: 3 };
      const response = { id: 4, ...request };
      mockPost.mockResolvedValue(response);

      const result = await departmentService.createDepartment(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/org/departments', request);
      expect(result.parentId).toBe(3);
    });
  });

  describe('updateDepartment', () => {
    it('should update a department by id', async () => {
      const updateData = { name: 'Engineering Updated' };
      const response = { id: 1, name: 'Engineering Updated', code: 'ENG' };
      mockPut.mockResolvedValue(response);

      const result = await departmentService.updateDepartment(1, updateData);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/org/departments/1', updateData);
      expect(result.name).toBe('Engineering Updated');
    });
  });

  describe('deleteDepartment', () => {
    it('should delete a department by id', async () => {
      mockDelete.mockResolvedValue(undefined);

      await departmentService.deleteDepartment(5);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/org/departments/5');
    });
  });
});
