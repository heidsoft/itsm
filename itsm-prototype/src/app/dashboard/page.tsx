'use client';

import React from 'react';
import {
    Target,
    AlertTriangle,
    GitMerge,
    Database,
    BarChart2,
    PieChart,
    FileText,
    Zap
} from 'lucide-react';
import { Sidebar } from '../components/Sidebar';
import { ResourceDistributionChart, ResourceHealthPieChart } from './charts';
import Link from 'next/link';
import { useRouter } from 'next/navigation';

// KPI卡片组件 (代码无变化)
const KpiCard = ({ title, value, icon: Icon, trend, period, color }) => (
    <div className="bg-white p-6 rounded-2xl shadow-lg hover:shadow-xl transition-shadow duration-300 ease-in-out">
        <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-600">{title}</h3>
            <div className={`p-2 rounded-full ${color} bg-opacity-20`}>
                <Icon className={`w-6 h-6 ${color}`} />
            </div>
        </div>
        <p className="text-4xl font-bold text-gray-800">{value}</p>
        {trend && (
            <div className="flex items-center mt-2 text-sm text-gray-500">
                <svg xmlns="http://www.w3.org/2000/svg" className={`w-4 h-4 mr-1 ${trend.startsWith('+') ? 'text-green-500' : 'text-red-500'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" /></svg>
                <span>{trend}</span>
                <span className="ml-1">vs {period}</span>
            </div>
        )}
    </div>
);

// 图表容器组件 (代码无变化)
const ChartContainer = ({ title, icon: Icon, children }) => (
    <div className="bg-white p-6 rounded-2xl shadow-lg">
        <div className="flex items-center mb-4">
            <Icon className="w-6 h-6 mr-3 text-gray-700" />
            <h3 className="text-xl font-semibold text-gray-700">{title}</h3>
        </div>
        {children}
    </div>
);

const ITSMDashboard = () => {
    const router = useRouter();
    const kpis = [
        { title: "SLA 达成率", value: "98.7%", icon: Target, trend: "+1.2%", period: "上月", color: "text-green-500" },
        { title: "高优先级事件", value: "8", icon: AlertTriangle, trend: "-3", period: "上周", color: "text-red-500" },
        { title: "待审批变更", value: "4", icon: GitMerge, trend: "+1", period: "昨天", color: "text-yellow-500" },
        { title: "纳管云资源", value: "325", icon: Database, trend: "+28", period: "上周", color: "text-blue-500" },
    ];

    const handleSimulateAlert = () => {
        const newIncidentId = `INC-${Math.floor(Math.random() * 10000).toString().padStart(5, '0')}`;
        alert(`模拟外部告警：已创建新的P1事件 ${newIncidentId}！\n（实际场景中，事件会直接进入事件管理列表）`);
        // 模拟将新事件添加到事件列表数据中
        // 在真实应用中，这里会调用API向后端发送告警数据
        router.push('/incidents'); // 跳转到事件列表页
    };

    return (
        <div className="flex h-screen bg-gray-100 font-sans">
            <Sidebar />
            <main className="flex-1 p-10 overflow-y-auto">
                <header className="mb-8 flex justify-between items-center">
                    <div>
                        <h2 className="text-4xl font-bold text-gray-800">指挥中心仪表盘</h2>
                        <p className="text-gray-500 mt-1">实时洞察多云IT服务运营状况</p>
                    </div>
                    <button 
                        onClick={handleSimulateAlert}
                        className="flex items-center bg-red-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-red-700 transition-colors animate-pulse"
                    >
                        <Zap className="w-5 h-5 mr-2" />
                        模拟外部告警 (P1事件)
                    </button>
                </header>

                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-8">
                    {kpis.map((kpi, index) => (
                        <KpiCard key={index} {...kpi} />
                    ))}
                </div>

                <div className="mt-12 grid grid-cols-1 lg:grid-cols-2 gap-8">
                    <ChartContainer title="多云资源分布概览" icon={BarChart2}>
                        <ResourceDistributionChart />
                    </ChartContainer>
                    <ChartContainer title="全局资源健康状态" icon={PieChart}>
                        <ResourceHealthPieChart />
                    </ChartContainer>
                </div>

                {/* 高级报告入口 */}
                <div className="mt-12 bg-white p-6 rounded-2xl shadow-lg">
                    <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                        <FileText className="w-6 h-6 mr-3" /> 高级报告与分析
                    </h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <Link href="#" className="block p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
                            <h4 className="font-semibold text-gray-800">服务绩效报告</h4>
                            <p className="text-sm text-gray-600">查看SLA达成情况和趋势分析。</p>
                        </Link>
                        <Link href="#" className="block p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
                            <h4 className="font-semibold text-gray-800">事件趋势分析</h4>
                            <p className="text-sm text-gray-600">分析事件发生频率、类型和解决时间。</p>
                        </Link>
                        <Link href="#" className="block p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
                            <h4 className="font-semibold text-gray-800">变更成功率报告</h4>
                            <p className="text-sm text-gray-600">评估变更的成功率和对服务的影响。</p>
                        </Link>
                    </div>
                </div>
            </main>
        </div>
    );
};

export default ITSMDashboard;
