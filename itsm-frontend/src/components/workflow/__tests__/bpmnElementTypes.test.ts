import { formatElementType, toShortType, typeSearchTerms } from '../bpmnElementTypes';

describe('bpmnElementTypes', () => {
  const registeredTranslate = (key: string): string =>
    key === 'bpmnDesigner.nodeTypes.userTask' ? '用户任务' : key;

  describe('toShortType', () => {
    it('去掉 BPMN 命名空间前缀', () => {
      expect(toShortType('bpmn:UserTask')).toBe('UserTask');
      expect(toShortType('itsm:CustomTask')).toBe('CustomTask');
    });

    it('无命名空间时原样返回，空值返回空串', () => {
      expect(toShortType('UserTask')).toBe('UserTask');
      expect(toShortType(undefined)).toBe('');
      expect(toShortType('')).toBe('');
    });
  });

  describe('typeSearchTerms', () => {
    it('同时给出短类型、去冒号全名与译文别名', () => {
      expect(typeSearchTerms('bpmn:UserTask', '用户任务')).toEqual([
        'usertask',
        'bpmnusertask',
        '用户任务',
      ]);
    });

    it('没有译文时仍可用英文名搜索', () => {
      expect(typeSearchTerms('bpmn:UserTask', '')).toEqual(['usertask', 'bpmnusertask']);
    });

    it('不产生重复别名', () => {
      expect(typeSearchTerms('UserTask', 'UserTask')).toEqual(['usertask']);
    });
  });

  describe('formatElementType', () => {
    it('已注册类型使用译文展示', () => {
      expect(formatElementType('bpmn:UserTask', registeredTranslate)).toBe('用户任务');
    });

    it('i18n key 未注册时回落英文短类型而不是点路径', () => {
      expect(formatElementType('bpmn:ExclusiveGateway', registeredTranslate)).toBe(
        'ExclusiveGateway'
      );
    });

    it('词表外的类型直接展示短名', () => {
      expect(formatElementType('itsm:CustomNode', registeredTranslate)).toBe('CustomNode');
    });

    it('空类型返回空串', () => {
      expect(formatElementType('', registeredTranslate)).toBe('');
    });
  });
});
