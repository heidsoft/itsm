import {
  buildAttributeSchemaFromFields,
  getAttributeSchemaFieldCount,
  isAttributeSchemaSafelyEditable,
  normalizeAttributeTemplateFields,
  parseAttributeSchemaToFields,
  validateAttributeSchema,
} from '../attributeSchemaUtils';

describe('attributeSchemaUtils', () => {
  describe('isAttributeSchemaSafelyEditable', () => {
    it('拒绝对象旧格式和未知扩展字段，避免有损保存', () => {
      expect(isAttributeSchemaSafelyEditable(JSON.stringify({ owner: { type: 'string' } }))).toBe(
        false
      );
      expect(
        isAttributeSchemaSafelyEditable(
          JSON.stringify({ fields: [{ key: 'owner', type: 'string', pattern: '.+' }] })
        )
      ).toBe(false);
      expect(
        isAttributeSchemaSafelyEditable(
          JSON.stringify({ fields: [{ key: 'owner', type: 'string' }] })
        )
      ).toBe(true);
    });
  });
  describe('buildAttributeSchemaFromFields', () => {
    it('空字段列表返回空字符串', () => {
      expect(buildAttributeSchemaFromFields([])).toBe('');
      expect(buildAttributeSchemaFromFields(undefined)).toBe('');
    });

    it('select 字段包含 options，其他类型不含 options', () => {
      const schema = buildAttributeSchemaFromFields([
        {
          key: 'env',
          label: '环境',
          type: 'select',
          options: '生产\n预发布, 开发',
          required: true,
        },
        { key: 'cpu', label: 'CPU核数', type: 'number' },
      ]);
      const parsed = JSON.parse(schema);
      expect(parsed.fields).toHaveLength(2);
      expect(parsed.fields[0]).toMatchObject({
        key: 'env',
        type: 'select',
        options: ['生产', '预发布', '开发'],
        required: true,
      });
      expect(parsed.fields[1].type).toBe('number');
      expect(parsed.fields[1].options).toBeUndefined();
    });
  });

  describe('parseAttributeSchemaToFields', () => {
    it('往返转换保持字段类型与选项', () => {
      const schema = buildAttributeSchemaFromFields([
        { key: 'env', label: '环境', type: 'select', options: 'a,b' },
        { key: 'expireAt', label: '到期日', type: 'date' },
      ]);
      const fields = parseAttributeSchemaToFields(schema);
      expect(fields).toEqual([
        expect.objectContaining({ key: 'env', type: 'select', options: 'a\nb' }),
        expect.objectContaining({ key: 'expireAt', type: 'date', options: '' }),
      ]);
    });

    it('未知类型回退为 string，非法 JSON 返回空数组', () => {
      const fields = parseAttributeSchemaToFields(
        JSON.stringify({ fields: [{ key: 'x', type: 'unknown' }] })
      );
      expect(fields[0].type).toBe('string');
      expect(parseAttributeSchemaToFields('{bad json')).toEqual([]);
    });
  });

  describe('validateAttributeSchema', () => {
    it('允许 string/number/boolean/date 无 options', () => {
      const schema = JSON.stringify({
        fields: [
          { key: 'a', type: 'string' },
          { key: 'b', type: 'number' },
          { key: 'c', type: 'boolean' },
          { key: 'd', type: 'date' },
        ],
      });
      expect(validateAttributeSchema(schema)).toBeNull();
    });

    it('select 缺少 options 时报错', () => {
      const schema = JSON.stringify({ fields: [{ key: 'env', type: 'select' }] });
      expect(validateAttributeSchema(schema)).toContain('options');
    });

    it('非法类型报错，空值/非法 JSON 有对应结果', () => {
      const schema = JSON.stringify({ fields: [{ key: 'x', type: 'file' }] });
      expect(validateAttributeSchema(schema)).toContain('type 仅支持');
      expect(validateAttributeSchema('')).toBeNull();
      expect(validateAttributeSchema('{oops')).toBe('请输入合法的 JSON');
    });
  });

  describe('normalizeAttributeTemplateFields', () => {
    it('去除空白字段并规范化选项', () => {
      const fields = normalizeAttributeTemplateFields([
        { key: ' env ', label: ' 环境 ', type: 'select', options: ' a ,, b ，c ' },
        { key: '', label: '', options: '' },
      ]);
      expect(fields).toHaveLength(1);
      expect(fields[0]).toMatchObject({ key: 'env', label: '环境', options: ['a', 'b', 'c'] });
    });
  });

  describe('getAttributeSchemaFieldCount', () => {
    it('统计 fields 数量或对象键数量', () => {
      expect(getAttributeSchemaFieldCount(JSON.stringify({ fields: [{}, {}] }))).toBe(2);
      expect(getAttributeSchemaFieldCount(JSON.stringify({ a: 1, b: 2, c: 3 }))).toBe(3);
      expect(getAttributeSchemaFieldCount('')).toBe(0);
      expect(getAttributeSchemaFieldCount('{bad')).toBe(0);
    });
  });
});
