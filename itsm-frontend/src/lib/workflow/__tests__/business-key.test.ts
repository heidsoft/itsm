/**
 * Tests for businessKey parsing / business document deep-link.
 *
 * 锁定的是「实例 → 业务单据」这条唯一关联锚点的解析口径。
 * 关键守卫：未登记类型与非法 ID 都**不得**产出 href（宁可点不动，也不能把用户
 * 导到别人的单子上），而 label 必须仍可读。
 */

import { BUSINESS_TYPES, businessTypeLabel, parseBusinessKey } from '../business-key';

describe('parseBusinessKey', () => {
  it('解析 prod 实际形态 ticket:1 并产出工单详情路径', () => {
    const parsed = parseBusinessKey('ticket:1');
    expect(parsed.type).toBe('ticket');
    expect(parsed.id).toBe(1);
    expect(parsed.href).toBe('/tickets/1');
    expect(parsed.label).toBe('工单 #1');
  });

  it('解析 prod 实际形态 incident:2（事件详情路径）', () => {
    const parsed = parseBusinessKey('incident:2');
    expect(parsed.href).toBe('/incidents/2');
    expect(parsed.label).toBe('事件 #2');
  });

  it.each([
    ['ticket', '/tickets/7'],
    ['change', '/changes/7'],
    ['incident', '/incidents/7'],
    ['service_request', '/service-requests/7'],
    ['problem', '/problems/7'],
    ['release', '/releases/7'],
  ])('业务类型 %s 映射到 %s', (type, href) => {
    expect(parseBusinessKey(`${type}:7`).href).toBe(href);
  });

  it('大小写归一：后端 strings.ToLower 已归一，手写数据仍要容错', () => {
    expect(parseBusinessKey('TICKET:9').href).toBe('/tickets/9');
  });

  it('多段键取第一个冒号分段：定时器 timer:id:unix 不因多冒号解析失败', () => {
    const parsed = parseBusinessKey('timer:t-1:1700000000');
    // 类型未登记 → 不产出 href，但原文可读
    expect(parsed.href).toBeUndefined();
    expect(parsed.rawType).toBe('timer');
    expect(parsed.label).toBe('timer:t-1:1700000000');
  });

  it('空串与空白返回占位符，不产出任何字段', () => {
    for (const value of ['', '   ', undefined, null]) {
      const parsed = parseBusinessKey(value as string | undefined);
      expect(parsed.label).toBe('-');
      expect(parsed.href).toBeUndefined();
      expect(parsed.type).toBeUndefined();
      expect(parsed.id).toBeUndefined();
    }
  });

  it('无 type:id 结构时原样展示且不猜测 ID', () => {
    const parsed = parseBusinessKey('custom-key-without-separator');
    expect(parsed.label).toBe('custom-key-without-separator');
    expect(parsed.href).toBeUndefined();
    expect(parsed.id).toBeUndefined();
  });

  it.each(['ticket:abc', 'ticket:1.5', 'ticket:-3', 'ticket: 1', 'ticket:', 'ticket:001a'])(
    '非法 ID「%s」不产出 href（避免把 NaN 拼进路由）',
    value => {
      const parsed = parseBusinessKey(value);
      expect(parsed.href).toBeUndefined();
      expect(parsed.id).toBeUndefined();
    }
  );

  /**
   * 这组是「正则校验」与「Number() 宽松转换」唯一的分水岭。
   *
   * `Number()` 会把十六进制、科学计数法、前导正号统统转成合法数字：
   *   Number('0x10') === 16、Number('1e3') === 1000、Number('+7') === 7
   * 若判据退化成「能转成数就算有效」，这三类脏数据都会被拼成真实业务单据路由，
   * 用户点进去就是别人的单子。必须断言成 undefined。
   */
  it.each(['ticket:0x10', 'ticket:1e3', 'ticket:+7'])(
    'Number() 能转换但正则不放行的 ID「%s」不产出 href',
    value => {
      const parsed = parseBusinessKey(value);
      expect(parsed.id).toBeUndefined();
      expect(parsed.href).toBeUndefined();
    }
  );

  it('前导零是合法的（0007 → 7），正则允许而 Number() 也一致', () => {
    expect(parseBusinessKey('ticket:0007').href).toBe('/tickets/7');
  });

  it('非法 ID 但类型合法时 label 仍可读，用于定位脏数据', () => {
    expect(parseBusinessKey('ticket:abc').label).toBe('工单 #abc');
  });

  it('0 不是有效业务 ID（后端自增 ID 从 1 起）', () => {
    expect(parseBusinessKey('ticket:0').href).toBeUndefined();
  });

  it('超出安全整数范围的 ID 不进入 href', () => {
    expect(parseBusinessKey('ticket:99999999999999999999').href).toBeUndefined();
  });
});

describe('businessTypeLabel', () => {
  it('已登记类型返回中文名', () => {
    expect(businessTypeLabel('ticket')).toBe('工单');
    expect(businessTypeLabel('SERVICE_REQUEST')).toBe('服务请求');
  });

  it('未登记或空值返回 undefined 而不是 "undefined" 字符串', () => {
    expect(businessTypeLabel('unknown_type')).toBeUndefined();
    expect(businessTypeLabel('')).toBeUndefined();
    expect(businessTypeLabel(undefined)).toBeUndefined();
  });
});

describe('BUSINESS_TYPES', () => {
  it('与后端 dto.BusinessType 枚举同序同集合（6 项）', () => {
    expect(BUSINESS_TYPES).toHaveLength(6);
    expect([...BUSINESS_TYPES]).toEqual([
      'ticket',
      'change',
      'incident',
      'service_request',
      'problem',
      'release',
    ]);
  });

  it('每个类型都有详情路径映射，无漏配', () => {
    for (const type of BUSINESS_TYPES) {
      const parsed = parseBusinessKey(`${type}:1`);
      expect(parsed.href).toBeDefined();
      expect(parsed.href?.startsWith('/')).toBe(true);
    }
  });
});
