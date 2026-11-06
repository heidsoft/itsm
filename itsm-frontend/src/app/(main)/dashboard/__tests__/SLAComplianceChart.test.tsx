import { render, screen } from '@testing-library/react';
import React from 'react';
import SLAComplianceChart from '../components/SLAComplianceChart';
import type { SLAComplianceData } from '../types/dashboard.types';

/**
 * 回归背景：后端 /dashboard/overview 的 slaData 是**聚合对象**，前端曾按
 * `SLAData[]`（service/target/actual）声明类型并在图表里先做 Array.isArray 过滤，
 * 于是对象被当成"没有数据"，SLA 面板在生产上长期只有"暂无数据"。以下用例锁死
 * 「对象形态必须被渲染」这条契约。
 */
const snapshot: SLAComplianceData = {
  complianceRate: 96.4,
  responseTimeCompliance: 98.2,
  resolutionTimeCompliance: 94.6,
  atRiskTickets: 2,
  breachedTickets: 4,
  totalTickets: 111,
  compliantTickets: 107,
};

describe('SLAComplianceChart 数据契约', () => {
  it('收到后端聚合对象时渲染达成率，而不是"暂无数据"', () => {
    render(<SLAComplianceChart data={snapshot} />);

    expect(screen.queryByText('暂无数据')).not.toBeInTheDocument();
    expect(screen.getByText('SLA 达成率')).toBeInTheDocument();
    // 96.4% 同时出现在 Statistic 与 Progress 文案里
    expect(screen.getAllByText(/96\.4/).length).toBeGreaterThan(0);
    expect(screen.getByText(/违约 4 张/)).toBeInTheDocument();
    expect(screen.getByText(/临期 2 张/)).toBeInTheDocument();
  });

  it('响应与解决时限各自使用自己的分项，不复制整体合规率', () => {
    render(<SLAComplianceChart data={snapshot} />);

    expect(screen.getByText('响应时限达成')).toBeInTheDocument();
    expect(screen.getByText('解决时限达成')).toBeInTheDocument();
    expect(screen.getAllByText(/98\.2/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/94\.6/).length).toBeGreaterThan(0);
  });

  it('数据缺失时用 KPI 卡的 overallValue 兜底', () => {
    render(<SLAComplianceChart data={null} overallValue={91.5} />);

    expect(screen.queryByText('暂无数据')).not.toBeInTheDocument();
    expect(screen.getAllByText(/91\.5/).length).toBeGreaterThan(0);
  });

  it('既无对象也无 overallValue 时才显示"暂无数据"', () => {
    render(<SLAComplianceChart data={null} />);

    expect(screen.getByText('暂无数据')).toBeInTheDocument();
  });

  it('统计窗口内无工单时如实说明达标率不适用，不展示百分比分项', () => {
    render(
      <SLAComplianceChart
        data={{ ...snapshot, complianceRate: 100, totalTickets: 0, breachedTickets: 0 }}
      />
    );

    expect(screen.getByText(/暂无工单，达标率不适用/)).toBeInTheDocument();
    expect(screen.queryByText('响应时限达成')).not.toBeInTheDocument();
  });
});
