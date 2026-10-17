import { formatMetricWindow, formatUptime } from '../metric-format';

describe('formatUptime', () => {
  it.each([
    [0, '—'],
    [-1, '—'],
    [Number.NaN, '—'],
    [59, '59秒'],
    [3600, '1小时 0分钟'],
    [90061, '1天 1小时'],
  ])('%s 秒显示为 %s', (seconds, expected) => {
    expect(formatUptime(seconds)).toBe(expected);
  });
});

describe('formatMetricWindow', () => {
  it.each([
    [0, '—'],
    [-0.5, '—'],
    // 亚秒级区间取整会变成「0s」，看起来像没有数据。
    [0.027, '27ms'],
    [0.4, '400ms'],
    [1, '1.0s'],
    [9.94, '9.9s'],
    [10, '10s'],
    [59, '59s'],
    [90, '1.5min'],
    [3600, '1.0h'],
  ])('%s 秒显示为 %s', (seconds, expected) => {
    expect(formatMetricWindow(seconds)).toBe(expected);
  });
});
