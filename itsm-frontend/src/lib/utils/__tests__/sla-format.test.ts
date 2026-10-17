import {
  SLA_PRIORITY_VALUES,
  formatBusinessHours,
  formatDuration,
  formatEscalationLevel,
  getIndustryLabel,
  getPriorityColor,
  getPriorityLabel,
  getServiceTypeLabel,
  matchesPriority,
} from '../sla-format';
import type { SLABusinessHours } from '@/lib/api/sla-api';

describe('SLA 优先级词表', () => {
  it('与后端匹配器的取值一致：critical/high/medium/low', () => {
    expect(SLA_PRIORITY_VALUES).toEqual(['critical', 'high', 'medium', 'low']);
  });

  it('urgent 归一成 critical，与后端 matcher 的别名口径相同', () => {
    expect(getPriorityLabel('urgent')).toBe('紧急');
    expect(getPriorityColor('urgent')).toBe('red');
    expect(getPriorityLabel('URGENT')).toBe('紧急');
  });

  it('未识别的优先级原样展示并退回默认色，不显示空白', () => {
    expect(getPriorityLabel('p1')).toBe('p1');
    expect(getPriorityColor('p1')).toBe('default');
  });

  it('筛选时行值与筛选值都归一，urgent 定义能被 critical 筛选命中', () => {
    expect(matchesPriority('urgent', 'critical')).toBe(true);
    expect(matchesPriority('critical', 'urgent')).toBe(true);
    expect(matchesPriority('high', 'all')).toBe(true);
    expect(matchesPriority('high', 'low')).toBe(false);
  });
});

describe('formatDuration', () => {
  it.each([
    [15, '15 分钟'],
    [60, '1 小时'],
    [90, '1 小时 30 分钟'],
    [480, '8 小时'],
    [1440, '1 天'],
    [1500, '1 天 1 小时'],
  ] as const)('%i 分钟展示为 %s', (minutes, expected) => {
    expect(formatDuration(minutes)).toBe(expected);
  });

  it('缺失或非法值显示未设置，而不是 NaN', () => {
    expect(formatDuration(0)).toBe('未设置');
    expect(formatDuration(Number.NaN)).toBe('未设置');
    expect(formatDuration(-5)).toBe('未设置');
  });
});

describe('formatBusinessHours', () => {
  it('后端未投影出配置时明确显示未配置', () => {
    expect(formatBusinessHours(undefined)).toBe('未配置');
    expect(formatBusinessHours({})).toBe('未配置');
  });

  it('alwaysOn 显示 7×24', () => {
    expect(formatBusinessHours({ alwaysOn: true })).toBe('7×24');
  });

  it('工作日加时间窗并带上时区', () => {
    const config: SLABusinessHours = {
      timeZone: 'Asia/Shanghai',
      workDays: [1, 2, 3, 4, 5],
      startTime: '09:00',
      endTime: '18:00',
    };
    expect(formatBusinessHours(config)).toBe('一、二、三、四、五 09:00-18:00 (Asia/Shanghai)');
  });

  it('七个工作日合并为每天', () => {
    expect(formatBusinessHours({ workDays: [1, 2, 3, 4, 5, 6, 7] })).toBe('每天');
  });
});

describe('formatEscalationLevel', () => {
  it('带通知角色时一起展示', () => {
    expect(
      formatEscalationLevel({ level: 1, thresholdMinutes: 15, notifyRoles: ['manager', 'it_admin'] })
    ).toBe('L1 · 15 分钟后升级 → manager、it_admin');
  });

  it('无通知角色时不留下多余箭头', () => {
    expect(formatEscalationLevel({ level: 2, thresholdMinutes: 120 })).toBe('L2 · 2 小时后升级');
  });
});

describe('服务类型与行业词表', () => {
  it('模板目录的 service_request 不再显示成原始下划线值', () => {
    expect(getServiceTypeLabel('service_request')).toBe('服务请求');
    expect(getIndustryLabel('service_request')).toBe('服务请求');
  });

  it('未知取值原样返回，不编造成中文标签', () => {
    expect(getServiceTypeLabel('problem_management')).toBe('problem_management');
    expect(getIndustryLabel('unknown_industry')).toBe('unknown_industry');
  });
});
