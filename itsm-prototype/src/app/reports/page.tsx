
'use client';

import React from 'react';
import Link from 'next/link';
import { FileText} from 'lucide-react';

const ReportCard = ({ title, description, icon: Icon, link }) => (
    <Link href={link} className="block bg-white p-6 rounded-lg shadow-md hover:shadow-xl transition-shadow duration-300 ease-in-out flex items-start space-x-4">
        <div className="p-3 rounded-full bg-blue-100 text-blue-600">
            <Icon className="w-6 h-6" />
        </div>
        <div>
            <h3 className="text-xl font-semibold text-gray-800 mb-2">{title}</h3>
            <p className="text-gray-600 text-sm">{description}</p>
        </div>
    </Link>
);

const ReportsPage = () => {
    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <h2 className="text-4xl font-bold text-gray-800">报告与分析</h2>
                <p className="text-gray-500 mt-1">获取IT服务绩效的深度洞察</p>
            </header>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
                <ReportCard 
                    title="事件趋势分析"
                    description="分析事件发生频率、类型、解决时间等趋势，识别潜在问题。"
                    icon={AlertTriangle}
                    link="/reports/incident-trends"
                />
                <ReportCard 
                    title="变更成功率报告"
                    description="评估变更的成功率、失败原因和对服务的影响。"
                    icon={GitMerge}
                    link="/reports/change-success"
                />
                <ReportCard 
                    title="SLA绩效报告"
                    description="监控服务级别协议（SLA）的达成情况，识别违约风险。"
                    icon={Target}
                    link="/reports/sla-performance"
                />
                <ReportCard 
                    title="问题管理效率报告"
                    description="分析问题解决周期、根本原因分布和已知错误管理情况。"
                    icon={Search}
                    link="/reports/problem-efficiency"
                />
                <ReportCard 
                    title="服务目录使用报告"
                    description="洞察服务请求的热点、用户偏好和交付效率。"
                    icon={BookOpen}
                    link="/reports/service-catalog-usage"
                />
                <ReportCard 
                    title="CMDB数据质量报告"
                    description="评估配置项数据的完整性、准确性和关联性。"
                    icon={FileText}
                    link="/reports/cmdb-quality"
                />
            </div>
        </div>
    );
};

export default ReportsPage;
