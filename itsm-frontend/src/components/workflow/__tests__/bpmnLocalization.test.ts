import { zhTranslate } from '../bpmnZhTranslate';
import { defaultNodeName } from '../bpmnDefaultNames';

describe('bpmnZhTranslate', () => {
  it('translates known palette strings', () => {
    expect(zhTranslate('User Task')).toBe('用户任务');
    expect(zhTranslate('Activate the hand tool')).toBe('激活手型工具');
  });

  it('interpolates replacements after lookup', () => {
    expect(zhTranslate('Append {type}', { type: '结束事件' })).toBe('追加结束事件');
  });

  it('falls back to the original template for unknown keys', () => {
    expect(zhTranslate('Some future bpmn-js string')).toBe('Some future bpmn-js string');
  });

  it('keeps the placeholder when a replacement is missing', () => {
    expect(zhTranslate('Append {type}', {})).toBe('追加{type}');
  });
});

describe('defaultNodeName', () => {
  it('gives natural Chinese defaults for common node types', () => {
    expect(defaultNodeName('bpmn:UserTask')).toBe('人工任务');
    expect(defaultNodeName('bpmn:ServiceTask')).toBe('自动任务');
    expect(defaultNodeName('bpmn:StartEvent')).toBe('开始');
    expect(defaultNodeName('bpmn:ExclusiveGateway')).toBe('排他网关');
  });

  it('returns null for types without a default', () => {
    expect(defaultNodeName('bpmn:TextAnnotation')).toBeNull();
    expect(defaultNodeName('')).toBeNull();
  });
});