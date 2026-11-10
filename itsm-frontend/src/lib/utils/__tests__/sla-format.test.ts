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
    expect(getPriorityLabel('urgent')).toBe("Urgent");
    expect(getPriorityColor('urgent')).toBe('red');
    expect(getPriorityLabel('URGENT')).toBe("Urgent");
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
    [1, '1 minute'],
    [15, '15 minutes'],
    [60, '1 hour'],
    [90, '1 hour 30 minutes'],
    [480, '8 hours'],
    [1440, '1 day'],
    [1500, '1 day 1 hour'],
  ] as const)('%i 分钟展示为 %s', (minutes, expected) => {
    expect(formatDuration(minutes)).toBe(expected);
  });

  it('缺失或非法值显示未设置，而不是 NaN', () => {
    expect(formatDuration(0)).toBe("Not set");
    expect(formatDuration(Number.NaN)).toBe("Not set");
    expect(formatDuration(-5)).toBe("Not set");
  });
});

describe('formatBusinessHours', () => {
  it('后端未投影出配置时明确显示未配置', () => {
    expect(formatBusinessHours(undefined)).toBe("Not configured");
    expect(formatBusinessHours({})).toBe("Not configured");
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
    expect(formatBusinessHours(config)).toBe('Mon, Tue, Wed, Thu, Fri 09:00-18:00 (Asia/Shanghai)');
  });

  it('七个工作日合并为每天', () => {
    expect(formatBusinessHours({ workDays: [1, 2, 3, 4, 5, 6, 7] })).toBe("Every day");
  });
});

describe('formatEscalationLevel', () => {
  it('带通知角色时一起展示', () => {
    expect(
      formatEscalationLevel({ level: 1, thresholdMinutes: 15, notifyRoles: ['manager', 'it_admin'] })
    ).toBe("L1 · 15 minutes before escalation → manager, it_admin");
  });

  it('无通知角色时不留下多余箭头', () => {
    expect(formatEscalationLevel({ level: 2, thresholdMinutes: 120 })).toBe("L2 · 2 hours before escalation");
  });
});

describe('服务类型与行业词表', () => {
  it('模板目录的 service_request 不再显示成原始下划线值', () => {
    expect(getServiceTypeLabel('service_request')).toBe("Service Request");
    expect(getIndustryLabel('service_request')).toBe("Service Request");
  });

  it('未知取值原样返回，不编造成中文标签', () => {
    expect(getServiceTypeLabel('problem_management')).toBe('problem_management');
    expect(getIndustryLabel('unknown_industry')).toBe('unknown_industry');
  });
});
