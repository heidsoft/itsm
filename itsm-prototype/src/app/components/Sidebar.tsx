
'use client';

import React from 'react';
import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { 
    LayoutDashboard, 
    BookOpen, 
    AlertTriangle, 
    Search, 
    GitMerge, 
    Target, 
    TrendingUp, 
    Database,
    User,
    Ticket,
    BarChart
} from 'lucide-react';

const NavItem = ({ href, icon: Icon, label }) => {
    const pathname = usePathname();
    const active = pathname === href || (href !== '/' && pathname.startsWith(href + '/'));

    return (
        <Link
            href={href}
            className={`flex items-center px-4 py-3 transition-colors duration-200 ease-in-out 
                ${active 
                    ? "bg-blue-600 text-white font-semibold shadow-lg rounded-lg"
                    : "text-gray-300 hover:bg-gray-700 hover:text-white rounded-lg"
                }`}
        >
            <Icon className="w-5 h-5 mr-3" />
            <span>{label}</span>
        </Link>
    );
};

export const Sidebar = () => {
    const navItems = [
        { href: "/dashboard", icon: LayoutDashboard, label: "仪表盘" },
        { href: "/tickets", icon: Ticket, label: "所有工单" }, 
        { href: "/service-catalog", icon: BookOpen, label: "服务目录" },
        { href: "/my-requests", icon: User, label: "我的请求" }, 
        { href: "/incidents", icon: AlertTriangle, label: "事件管理" },
        { href: "/problems", icon: Search, label: "问题管理" },
        { href: "/changes", icon: GitMerge, label: "变更管理" },
        { href: "/sla", icon: Target, label: "服务级别管理" },
        { href: "/improvements", icon: TrendingUp, label: "持续改进" },
        { href: "/cmdb", icon: Database, label: "配置管理 (CMDB)" },
        { href: "/knowledge-base", icon: BookOpen, label: "知识库" }, 
        { href: "/reports", icon: BarChart, label: "报告与分析" }, // 新增报告与分析
    ];

    return (
        <aside className="w-72 bg-gray-900 text-white flex flex-col p-4">
            <div className="flex items-center justify-center p-4 mb-6 border-b border-gray-700">
                <h1 className="text-2xl font-bold tracking-wider">ITSM<span className="text-blue-400">Pro</span></h1>
            </div>
            <nav className="flex flex-col space-y-2">
                {navItems.map((item) => (
                    <NavItem key={item.href} {...item} />
                ))}
            </nav>
            
        </aside>
    );
};
