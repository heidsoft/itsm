'use client';

import { useEffect, useState } from 'react';
import { 
  BarChart3, 
  TrendingUp, 
  AlertTriangle, 
  CheckCircle, 
  Clock, 
  Users,
  Ticket,
  Activity
} from 'lucide-react';
import { useUIStore } from '@/lib/store/ui-store';

/**
 * 仪表盘统计数据接口
 */
interface DashboardStats {
  totalTickets: number;
  openTickets: number;
  resolvedTickets: number;
  criticalIncidents: number;
  activeUsers: number;
  avgResolutionTime: number;
}

/**
 * 统计卡片组件
 */
interface StatCardProps {
  title: string;
  value: string | number;
  icon: React.ComponentType<{ className?: string }>;
  trend?: {
    value: number;
    isPositive: boolean;
  };
  color: 'blue' | 'green' | 'red' | 'yellow' | 'purple' | 'indigo';
}

function StatCard({ title, value, icon: Icon, trend, color }: StatCardProps) {
  const colorClasses = {
    blue: 'bg-blue-50 text-blue-600 border-blue-200',
    green: 'bg-green-50 text-green-600 border-green-200',
    red: 'bg-red-50 text-red-600 border-red-200',
    yellow: 'bg-yellow-50 text-yellow-600 border-yellow-200',
    purple: 'bg-purple-50 text-purple-600 border-purple-200',
    indigo: 'bg-indigo-50 text-indigo-600 border-indigo-200',
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-md transition-shadow">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-600">{title}</p>
          <p className="text-2xl font-bold text-gray-900 mt-2">{value}</p>
          {trend && (
            <div className={`flex items-center mt-2 text-sm ${
              trend.isPositive ? 'text-green-600' : 'text-red-600'
            }`}>
              <TrendingUp className={`w-4 h-4 mr-1 ${
                trend.isPositive ? '' : 'rotate-180'
              }`} />
              {Math.abs(trend.value)}%
            </div>
          )}
        </div>
        <div className={`p-3 rounded-lg border ${colorClasses[color]}`}>
          <Icon className="w-6 h-6" />
        </div>
      </div>
    </div>
  );
}

/**
 * 仪表盘页面组件
 */
export default function DashboardPage() {
  const { setPageTitle, setBreadcrumbs } = useUIStore();
  const [stats, setStats] = useState<DashboardStats>({
    totalTickets: 0,
    openTickets: 0,
    resolvedTickets: 0,
    criticalIncidents: 0,
    activeUsers: 0,
    avgResolutionTime: 0,
  });
  const [isLoading, setIsLoading] = useState(true);

  // 设置页面标题和面包屑
  useEffect(() => {
    setPageTitle('仪表盘');
    setBreadcrumbs([
      { label: '首页', href: '/dashboard' },
      { label: '仪表盘' },
    ]);
  }, [setPageTitle, setBreadcrumbs]);

  // 模拟数据加载
  useEffect(() => {
    const loadDashboardData = async () => {
      try {
        setIsLoading(true);
        
        // 模拟API调用延迟
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // 模拟统计数据
        setStats({
          totalTickets: 1247,
          openTickets: 89,
          resolvedTickets: 1158,
          criticalIncidents: 3,
          activeUsers: 156,
          avgResolutionTime: 4.2,
        });
      } catch (error) {
        console.error('Failed to load dashboard data:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadDashboardData();
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-2 text-gray-600">加载中...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">仪表盘</h1>
        <p className="text-gray-600 mt-1">系统运行状态和关键指标概览</p>
      </div>

      {/* 统计卡片网格 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="总工单数"
          value={stats.totalTickets.toLocaleString()}
          icon={Ticket}
          trend={{ value: 12, isPositive: true }}
          color="blue"
        />
        <StatCard
          title="待处理工单"
          value={stats.openTickets}
          icon={Clock}
          trend={{ value: 5, isPositive: false }}
          color="yellow"
        />
        <StatCard
          title="已解决工单"
          value={stats.resolvedTickets.toLocaleString()}
          icon={CheckCircle}
          trend={{ value: 8, isPositive: true }}
          color="green"
        />
        <StatCard
          title="紧急事件"
          value={stats.criticalIncidents}
          icon={AlertTriangle}
          color="red"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <StatCard
          title="活跃用户"
          value={stats.activeUsers}
          icon={Users}
          trend={{ value: 3, isPositive: true }}
          color="purple"
        />
        <StatCard
          title="平均解决时间"
          value={`${stats.avgResolutionTime}小时`}
          icon={Activity}
          trend={{ value: 15, isPositive: true }}
          color="indigo"
        />
      </div>

      {/* 快速操作区域 */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">快速操作</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <button className="flex items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
            <Ticket className="w-5 h-5 text-blue-600 mr-3" />
            <span className="text-gray-700">创建工单</span>
          </button>
          <button className="flex items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
            <AlertTriangle className="w-5 h-5 text-red-600 mr-3" />
            <span className="text-gray-700">报告事件</span>
          </button>
          <button className="flex items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
            <BarChart3 className="w-5 h-5 text-green-600 mr-3" />
            <span className="text-gray-700">查看报表</span>
          </button>
        </div>
      </div>

      {/* 最近活动 */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">最近活动</h2>
        <div className="space-y-3">
          {[
            { type: 'ticket', message: '工单 #1234 已被分配给张三', time: '5分钟前' },
            { type: 'incident', message: '紧急事件 #INC-001 已解决', time: '15分钟前' },
            { type: 'user', message: '新用户李四已注册', time: '1小时前' },
            { type: 'system', message: '系统维护已完成', time: '2小时前' },
          ].map((activity, index) => (
            <div key={index} className="flex items-center justify-between py-2 border-b border-gray-100 last:border-b-0">
              <div className="flex items-center">
                <div className={`w-2 h-2 rounded-full mr-3 ${
                  activity.type === 'ticket' ? 'bg-blue-500' :
                  activity.type === 'incident' ? 'bg-red-500' :
                  activity.type === 'user' ? 'bg-green-500' : 'bg-gray-500'
                }`} />
                <span className="text-gray-700">{activity.message}</span>
              </div>
              <span className="text-sm text-gray-500">{activity.time}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}