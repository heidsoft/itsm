import { WorkflowTemplateApi } from '@/lib/api/workflow-template-api';
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

describe('WorkflowTemplateApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getTemplates', () => {
    it('should return empty array', async () => {
      const res = await WorkflowTemplateApi.getTemplates();
      expect(res).toEqual([]);
    });

    it('should accept params', async () => {
      const res = await WorkflowTemplateApi.getTemplates({ category: 'general' });
      expect(res).toEqual([]);
    });
  });

  describe('getTemplate', () => {
    it('should return template object with given id', async () => {
      const res = await WorkflowTemplateApi.getTemplate('tpl-1');
      expect(res.id).toBe('tpl-1');
      expect(res.name).toBe('未命名模板');
      expect(res.definition).toBeDefined();
      expect(res.definition.nodes).toEqual([]);
    });
  });

  describe('createFromTemplate', () => {
    it('should throw because WorkflowDefinitionApi is not initialized', async () => {
      await expect(WorkflowTemplateApi.createFromTemplate('tpl-1', 'New WF'))
        .rejects.toThrow('WorkflowDefinitionApi not initialized');
    });
  });

  describe('saveAsTemplate', () => {
    it('should return template with provided data', async () => {
      const res = await WorkflowTemplateApi.saveAsTemplate('wf1', {
        name: 'My Template',
        category: 'IT',
        description: 'A template',
        isPublic: true,
        tags: ['tag1'],
      });
      expect(res.name).toBe('My Template');
      expect(res.category).toBe('IT');
      expect(res.isPublic).toBe(true);
      expect(res.tags).toEqual(['tag1']);
      expect(res.id).toContain('template_');
    });
  });
});
