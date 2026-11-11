/**
 * 「能力未就绪」契约回归锁。
 *
 * 后端契约：能力未启用 / 尚未实现时返回 HTTP 503 + 业务码 5003
 * （MSPMiddleware 还带 data:{capability,state:'disabled'}）。
 * prod v1.6.54 全量黑盒回归的 3 个 WARN 中，MSP 分配 / 分配历史两项即此，
 * 知识文章评论也是同一个码。这是**永久状态**，不是「服务暂时不可用」。
 *
 * 这些断言刻意成对：一半锁「5003 必须特殊处理」，另一半锁「普通 503 行为不变」。
 * 只锁前者的话，把 503 一刀切成不可重试也能假绿。
 */

import {
  CAPABILITY_UNAVAILABLE,
  CAPABILITY_UNAVAILABLE_CODE,
  getErrorMessageByCode,
  getErrorSeverity,
  isCapabilityUnavailableError,
  isRetryableError,
  mapErrorToAction,
  parseErrorCode,
} from '../error-message-handler';
import { ApiRequestError } from '@/lib/api/http-client';

const mspDisabled = new ApiRequestError('MSP 能力在当前部署形态下未启用', 503, 5003);
const commentsMissing = new ApiRequestError('知识文章评论功能尚未实现', 503, 5003);
const transient = new ApiRequestError('服务暂时不可用', 503);

describe('isCapabilityUnavailableError', () => {
  it('识别 HTTP 503 + 业务码 5003 的真实后端错误', () => {
    expect(isCapabilityUnavailableError(mspDisabled)).toBe(true);
    expect(isCapabilityUnavailableError(commentsMissing)).toBe(true);
  });

  it('兼容字符串业务码与响应体里的 capability state', () => {
    expect(isCapabilityUnavailableError({ code: '5003', status: 503 })).toBe(true);
    expect(isCapabilityUnavailableError({ status: 503, data: { capability: 'msp', state: 'disabled' } })).toBe(true);
    expect(isCapabilityUnavailableError({ status: 503, data: { state: 'not_implemented' } })).toBe(true);
    expect(
      isCapabilityUnavailableError({ response: { data: { capability: 'msp', state: 'disabled' } } })
    ).toBe(true);
  });

  it('不把真正的临时故障或其它 4xx/5xx 误判为能力未就绪', () => {
    expect(isCapabilityUnavailableError(transient)).toBe(false);
    expect(isCapabilityUnavailableError({ status: 503 })).toBe(false);
    expect(isCapabilityUnavailableError({ status: 502, code: 5002 })).toBe(false);
    expect(isCapabilityUnavailableError({ status: 404, code: 4004 })).toBe(false);
    expect(isCapabilityUnavailableError({ status: 503, data: { state: 'degraded' } })).toBe(false);
    expect(isCapabilityUnavailableError(null)).toBe(false);
    expect(isCapabilityUnavailableError('5003')).toBe(false);
  });
});

describe('能力未就绪不得按可自愈故障处理', () => {
  it('isRetryableError 对 5003 返回 false，但普通 503 仍返回 true', () => {
    expect(isRetryableError(mspDisabled)).toBe(false);
    expect(isRetryableError({ status: 503, code: 5003 })).toBe(false);
    // 对照组：不能为了修 5003 就把所有 503 都禁掉重试
    expect(isRetryableError({ status: 503 })).toBe(true);
    expect(isRetryableError({ status: 502 })).toBe(true);
    expect(isRetryableError({ code: 'TIMEOUT' })).toBe(true);
  });

  it('parseErrorCode 不把 5003 折叠成 SERVICE_UNAVAILABLE', () => {
    expect(parseErrorCode(mspDisabled)).toBe(CAPABILITY_UNAVAILABLE);
    expect(parseErrorCode({ status: 503, code: 5003 })).toBe(CAPABILITY_UNAVAILABLE);
    expect(parseErrorCode({ status: 503 })).toBe('SERVICE_UNAVAILABLE');
  });

  it('严重级别不按 5xx 报 critical', () => {
    expect(getErrorSeverity(mspDisabled)).toBe('medium');
    expect(getErrorSeverity({ status: 503 })).toBe('critical');
  });

  it('建议动作是联系管理员而非稍后重试', () => {
    expect(mapErrorToAction(mspDisabled)).toBe('contact_admin');
    expect(mapErrorToAction({ status: 503 })).toBe('retry_later');
    expect(getErrorMessageByCode(CAPABILITY_UNAVAILABLE)).toContain('未启用');
    expect(getErrorMessageByCode('SERVICE_UNAVAILABLE')).toBe('服务暂时不可用');
  });

  it('业务码常量与后端契约一致', () => {
    expect(CAPABILITY_UNAVAILABLE_CODE).toBe(5003);
  });
});
