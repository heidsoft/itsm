/**
 * ticket-form-utils 工具函数测试
 */

import {
  MOCK_TICKET_TEMPLATES,
  MOCK_USER_LIST,
  TICKET_FORM_STEPS,
  TITLE_RULES,
  DESCRIPTION_RULES,
  TYPE_RULES,
  CATEGORY_RULES,
  PRIORITY_RULES,
  TICKET_TYPE_OPTIONS,
  PRIORITY_OPTIONS,
  CATEGORY_OPTIONS,
  getFieldsForStep,
  mergeFormData,
  resetFormData,
} from '../utils/ticket-form-utils';

describe('ticket-form-utils', () => {
  describe('MOCK_TICKET_TEMPLATES', () => {
    it('应该包含模板数据', () => {
      expect(MOCK_TICKET_TEMPLATES).toBeDefined();
      expect(Array.isArray(MOCK_TICKET_TEMPLATES)).toBe(true);
      expect(MOCK_TICKET_TEMPLATES.length).toBeGreaterThan(0);
    });

    it('每个模板应该有必需字段', () => {
      MOCK_TICKET_TEMPLATES.forEach(template => {
        expect(template).toHaveProperty('id');
        expect(template).toHaveProperty('name');
        expect(template).toHaveProperty('description');
        expect(template).toHaveProperty('category');
        expect(template).toHaveProperty('priority');
        expect(template).toHaveProperty('type');
      });
    });

    it('模板应该包含有效的描述', () => {
      MOCK_TICKET_TEMPLATES.forEach(template => {
        expect(typeof template.description).toBe('string');
        expect(template.description.length).toBeGreaterThan(0);
      });
    });
  });

  describe('MOCK_USER_LIST', () => {
    it('应该包含用户数据', () => {
      expect(MOCK_USER_LIST).toBeDefined();
      expect(Array.isArray(MOCK_USER_LIST)).toBe(true);
      expect(MOCK_USER_LIST.length).toBeGreaterThan(0);
    });

    it('每个用户应该有必需字段', () => {
      MOCK_USER_LIST.forEach(user => {
        expect(user).toHaveProperty('id');
        expect(user).toHaveProperty('name');
        expect(user).toHaveProperty('role');
      });
    });
  });

  describe('TICKET_FORM_STEPS', () => {
    it('应该定义表单步骤', () => {
      expect(TICKET_FORM_STEPS).toBeDefined();
      expect(Array.isArray(TICKET_FORM_STEPS)).toBe(true);
      expect(TICKET_FORM_STEPS.length).toBeGreaterThan(0);
    });

    it('每个步骤应该有标题', () => {
      TICKET_FORM_STEPS.forEach(step => {
        expect(step).toHaveProperty('title');
      });
    });
  });

  describe('TITLE_RULES', () => {
    it('应该定义标题验证规则', () => {
      expect(TITLE_RULES).toBeDefined();
      expect(Array.isArray(TITLE_RULES)).toBe(true);
    });
  });

  describe('DESCRIPTION_RULES', () => {
    it('应该定义描述验证规则', () => {
      expect(DESCRIPTION_RULES).toBeDefined();
      expect(Array.isArray(DESCRIPTION_RULES)).toBe(true);
    });
  });

  describe('TYPE_RULES', () => {
    it('应该定义类型验证规则', () => {
      expect(TYPE_RULES).toBeDefined();
      expect(Array.isArray(TYPE_RULES)).toBe(true);
    });
  });

  describe('CATEGORY_RULES', () => {
    it('应该定义分类验证规则', () => {
      expect(CATEGORY_RULES).toBeDefined();
      expect(Array.isArray(CATEGORY_RULES)).toBe(true);
    });
  });

  describe('PRIORITY_RULES', () => {
    it('应该定义优先级验证规则', () => {
      expect(PRIORITY_RULES).toBeDefined();
      expect(Array.isArray(PRIORITY_RULES)).toBe(true);
    });
  });

  describe('TICKET_TYPE_OPTIONS', () => {
    it('应该定义工单类型选项', () => {
      expect(TICKET_TYPE_OPTIONS).toBeDefined();
      expect(Array.isArray(TICKET_TYPE_OPTIONS)).toBe(true);
      expect(TICKET_TYPE_OPTIONS.length).toBeGreaterThan(0);
    });

    it('每个选项应该有 value 和 label', () => {
      TICKET_TYPE_OPTIONS.forEach(option => {
        expect(option).toHaveProperty('value');
        expect(option).toHaveProperty('label');
      });
    });
  });

  describe('PRIORITY_OPTIONS', () => {
    it('应该定义优先级选项', () => {
      expect(PRIORITY_OPTIONS).toBeDefined();
      expect(Array.isArray(PRIORITY_OPTIONS)).toBe(true);
      expect(PRIORITY_OPTIONS.length).toBeGreaterThan(0);
    });
  });

  describe('CATEGORY_OPTIONS', () => {
    it('应该定义分类选项', () => {
      expect(CATEGORY_OPTIONS).toBeDefined();
      expect(Array.isArray(CATEGORY_OPTIONS)).toBe(true);
      expect(CATEGORY_OPTIONS.length).toBeGreaterThan(0);
    });
  });

  describe('getFieldsForStep', () => {
    it('应该返回步骤0的字段', () => {
      const fields = getFieldsForStep(0);
      expect(fields).toContain('title');
      expect(fields).toContain('description');
    });

    it('应该返回步骤1的字段', () => {
      const fields = getFieldsForStep(1);
      expect(fields).toContain('type');
      expect(fields).toContain('category');
      expect(fields).toContain('priority');
    });

    it('应该返回步骤2的空数组', () => {
      const fields = getFieldsForStep(2);
      expect(Array.isArray(fields)).toBe(true);
      expect(fields.length).toBe(0);
    });
  });

  describe('mergeFormData', () => {
    it('应该合并表单数据', () => {
      const base = { title: 'Test', description: 'Desc' };
      const updates = { priority: 'high' };
      const result = mergeFormData(base, updates);
      expect(result.title).toBe('Test');
      expect(result.description).toBe('Desc');
      expect(result.priority).toBe('high');
    });

    it('应该覆盖已有字段', () => {
      const base = { title: 'Old' };
      const updates = { title: 'New' };
      const result = mergeFormData(base, updates);
      expect(result.title).toBe('New');
    });
  });

  describe('resetFormData', () => {
    it('应该返回空对象', () => {
      const result = resetFormData();
      expect(typeof result).toBe('object');
      expect(Object.keys(result).length).toBe(0);
    });
  });
});
