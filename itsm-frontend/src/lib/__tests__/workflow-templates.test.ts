/**
 * Tests for src/lib/workflow-templates.ts
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { WORKFLOW_TEMPLATES } from '../workflow-templates';
import type { WorkflowTemplate } from '../workflow-templates';

describe('workflow-templates', () => {
  it('exports WORKFLOW_TEMPLATES array', () => {
    expect(Array.isArray(WORKFLOW_TEMPLATES)).toBe(true);
    expect(WORKFLOW_TEMPLATES.length).toBeGreaterThan(0);
  });

  it('each template has required fields', () => {
    WORKFLOW_TEMPLATES.forEach((tmpl: WorkflowTemplate) => {
      expect(tmpl.id).toBeDefined();
      expect(typeof tmpl.id).toBe('string');
      expect(tmpl.name).toBeDefined();
      expect(typeof tmpl.name).toBe('string');
      expect(tmpl.description).toBeDefined();
      expect(tmpl.category).toBeDefined();
      expect(tmpl.icon).toBeDefined();
      expect(tmpl.bpmnXml).toBeDefined();
      expect(tmpl.approvalConfig).toBeDefined();
      expect(tmpl.approvalConfig.requireApproval).toBeDefined();
      expect(['single', 'parallel', 'sequential']).toContain(tmpl.approvalConfig.approvalType);
      expect(Array.isArray(tmpl.approvalConfig.approvers)).toBe(true);
    });
  });

  it('has leave_request template', () => {
    const tmpl = WORKFLOW_TEMPLATES.find(t => t.id === 'leave_request');
    expect(tmpl).toBeDefined();
    expect(tmpl!.category).toBe('hr');
  });

  it('templates have unique ids', () => {
    const ids = WORKFLOW_TEMPLATES.map(t => t.id);
    const uniqueIds = new Set(ids);
    expect(uniqueIds.size).toBe(ids.length);
  });

  it('bpmnXml contains valid XML structure', () => {
    WORKFLOW_TEMPLATES.forEach((tmpl: WorkflowTemplate) => {
      expect(tmpl.bpmnXml).toContain('<?xml');
      expect(tmpl.bpmnXml).toContain('bpmn:');
    });
  });
});
