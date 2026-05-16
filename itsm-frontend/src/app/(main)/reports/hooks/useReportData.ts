import { useState, useEffect } from 'react';
import { ticketAnalyticsService } from '@/lib/services/analytics-service';
import { ticketService } from '@/lib/services/ticket-service';

interface ReportMetrics {
  totalTickets: number;
  resolvedTickets: number;
  avgResolutionTime: number;
  slaCompliance: number;
  satisfactionScore: number;
  activeAgents: number;
  pendingTickets: number;
  urgentTickets: number;
}

export const useReportData = () => {
  const [metrics, setMetrics] = useState<ReportMetrics | null>(null);
  const [timeRange, setTimeRange] = useState<[string, string] | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadReportMetrics();
  }, [timeRange]);

  const loadReportMetrics = async () => {
    try {
      setLoading(true);
      
      // 调用真实API获取数据
      const [analyticsRes, statsRes] = await Promise.all([
        ticketAnalyticsService.getAnalytics({
          date_from: timeRange?.[0],
          date_to: timeRange?.[1],
        }),
        ticketService.getTicketStats(),
      ]);

      // 使用真实数据设置指标
      const fetchedMetrics: ReportMetrics = {
        totalTickets: statsRes.total || analyticsRes.total_tickets || 0,
        resolvedTickets: statsRes.resolved || analyticsRes.resolved_tickets || 0,
        avgResolutionTime: analyticsRes.processing_time_stats?.avg_resolution_time || 0,
        slaCompliance: analyticsRes.processing_time_stats?.sla_compliance_rate || 0,
        satisfactionScore: 0, // 满意度数据可能需要单独API
        activeAgents: 0, // 活跃坐席可能需要单独API
        pendingTickets: statsRes.open || 0,
        urgentTickets: statsRes.highPriority || 0,
      };

      setMetrics(fetchedMetrics);
    } catch (error) {
      console.error('加载报表数据失败:', error);
      // 不再使用模拟数据，保持null状态
      setMetrics(null);
    } finally {
      setLoading(false);
    }
  };

  return { metrics, timeRange, setTimeRange, loading, loadReportMetrics };
};
