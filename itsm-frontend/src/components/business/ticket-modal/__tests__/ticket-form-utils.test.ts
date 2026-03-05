/**
 * ticket-form-utils 工具函数测试
 */

import {
  MOCK_TICKET_TEMPLATES,
  MOCK_USER_LIST,
  getTicketTemplates,
  getUsersByDepartment,
  filterTemplatesByCategory,
  validateTicketForm,
  formatTicketData,
  generateTicketNumber,
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
        expect(template).toHaveProperty('defaultValues');
      });
    });

    it('模板应该包含有效的默认值', () => {
      MOCK_TICKET_TEMPLATES.forEach(template => {
        expect(template.defaultValues).toHaveProperty('title');
        expect(template.defaultValues).toHaveProperty('type');
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
        expect(user).toHaveProperty('email');
      });
    });
  });

  describe('getTicketTemplates', () => {
    it('应该返回所有模板', () => {
      const templates = getTicketTemplates();
      expect(templates).toEqual(MOCK_TICKET_TEMPLATES);
    });

    it('返回的应该是数组', () => {
      const templates = getTicketTemplates();
      expect(Array.isArray(templates)).toBe(true);
    });
  });

  describe('getUsersByDepartment', () => {
    it('应该返回指定部门的用户', () => {
      const users = getUsersByDepartment('IT');
      expect(Array.isArray(users)).toBe(true);
      users.forEach(user => {
        expect(user.department).toBe('IT');
      });
    });

    it('不存在的部门应返回空数组', () => {
      const users = getUsersByDepartment('NonExistent');
      expect(users).toEqual([]);
    });

    it('应该正确处理空参数', () => {
      const users = getUsersByDepartment('');
      expect(Array.isArray(users)).toBe(true);
    });
  });

  describe('filterTemplatesByCategory', () => {
    it('应该按分类筛选模板', () => {
      const filtered = filterTemplatesByCategory('network');
      expect(filtered.every(t => t.category === 'network')).toBe(true);
    });

    it('返回的应该是数组', () => {
      const filtered = filterTemplatesByCategory('hardware');
      expect(Array.isArray(filtered)).toBe(true);
    });

    it('不存在的分类应返回空数组', () => {
      const filtered = filterTemplatesByCategory('unknown');
      expect(filtered).toEqual([]);
    });

    it('应该保留所有模板当分类为 all', () => {
      const filtered = filterTemplatesByCategory('all');
      expect(filtered.length).toBe(MOCK_TICKET_TEMPLATES.length);
    });
  });

  describe('validateTicketForm', () => {
    it('应该验证必填字段', () => {
      const validData = {
        title: 'Test Ticket',
        description: 'Test description',
        type: 'incident',
      };

      const result = validateTicketForm(validData);
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it('应该拒绝空标题', () => {
      const invalidData = {
        title: '',
        description: 'Test description',
        type: 'incident',
      };

      const result = validateTicketForm(invalidData);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'title')).toBe(true);
    });

    it('应该拒绝空描述', () => {
      const invalidData = {
        title: 'Test Ticket',
        description: '',
        type: 'incident',
      };

      const result = validateTicketForm(invalidData);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'description')).toBe(true);
    });

    it('应该拒绝过长的标题', () => {
      const longTitle = 'a'.repeat(201); // 超过 200 字符限制

      const result = validateTicketForm({
        title: longTitle,
        description: 'Test description',
        type: 'incident',
      });

      expect(result.valid).toBe(false);
    });

    it('应该接受最小数据', () => {
      const minimal = {
        title: 'Minimal Ticket',
        description: 'Minimal description',
        type: 'incident',
      };

      const result = validateTicketForm(minimal);
      expect(result.valid).toBe(true);
    });

    it('应该处理可选字段', () => {
      const withOptional = {
        title: 'Test Ticket',
        description: 'Test description',
        type: 'incident',
        priority: 'high',
        category: 'hardware',
        assigneeId: 'user1',
        dueDate: '2024-12-31',
      };

      const result = validateTicketForm(withOptional);
      expect(result.valid).toBe(true);
    });

    it('应该验证有效类型', () => {
      const validTypes = ['incident', 'problem', 'change', 'request'];

      validTypes.forEach(type => {
        const result = validateTicketForm({
          title: 'Test',
          description: 'Test',
          type,
        });
        expect(result.valid).toBe(true);
      });
    });

    it('应该拒绝无效类型', () => {
      const result = validateTicketForm({
        title: 'Test',
        description: 'Test',
        type: 'invalid-type',
      });

      expect(result.valid).toBe(false);
    });
  });

  describe('formatTicketData', () => {
    it('应该格式化日期字段', () => {
      const rawData = {
        title: 'Test',
        description: 'Test',
        type: 'incident',
        dueDate: '2024-12-31T00:00:00Z',
      };

      const formatted = formatTicketData(rawData);
      expect(formatted.dueDate).toBeDefined();
    });

    it('应该保留所有原始字段', () => {
      const rawData = {
        title: 'Test',
        description: 'Test',
        type: 'incident',
        customField: 'custom',
      };

      const formatted = formatTicketData(rawData);
      expect(formatted.customField).toBe('custom');
    });

    it('应该处理空日期', () => {
      const rawData = {
        title: 'Test',
        description: 'Test',
        type: 'incident',
        dueDate: null,
      };

      const formatted = formatTicketData(rawData);
      expect(formatted.dueDate).toBeNull();
    });

    it('应该添加时间戳', () => {
      const formatted = formatTicketData({
        title: 'Test',
        description: 'Test',
        type: 'incident',
      });

      expect(formatted.createdAt).toBeDefined();
      expect(formatted.updatedAt).toBeDefined();
    });
  });

  describe('generateTicketNumber', () => {
    it('应该生成工单编号', () => {
      const number = generateTicketNumber();
      expect(number).toMatch(/^INC-\d{8}-\d{4}$/);
    });

    it('每次调用应生成不同编号', () => {
      const number1 = generateTicketNumber();
      const number2 = generateTicketNumber();
      expect(number1).not.toBe(number2);
    });

    it('应该包含日期前缀', () => {
      const number = generateTicketNumber();
      const today = new Date().toISOString().slice(0, 10).replace(/-/g, '');
      expect(number).toContain(today);
    });

    it('应该生成有效格式', () => {
      const number = generateTicketNumber();
      const parts = number.split('-');
      expect(parts.length).toBe(3);
      expect(parts[0]).toBe('INC');
      expect(parseInt(parts[1])).not.toBeNaN();
      expect(parseInt(parts[2])).not.toBeNaN();
    });
  });
});
