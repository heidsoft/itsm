'use client';

import { useEffect } from 'react';
import { usePathname } from 'next/navigation';
import { useRecentVisitStore } from '@/lib/store/recent-visit-store';

// 路径到标题和图标映射
const pathMeta: Record<string, { title: string; icon?: string }> = {
  '/dashboard': { title: 'Home page', icon: 'home' },
  '/tickets': { title: 'Ticket Management', icon: 'ticket' },
  '/incidents': { title: 'Incidents', icon: 'alert-circle' },
  '/problems': { title: 'Problems', icon: 'alert-triangle' },
  '/changes': { title: 'Changes', icon: 'refresh-cw' },
  '/knowledge': { title: 'Knowledge Base', icon: 'book' },
  '/service-catalog': { title: 'Service Catalog', icon: 'grid' },
  '/profile': { title: 'Profile', icon: 'user' },
  '/notifications': { title: 'Notification Center', icon: 'bell' },
  '/admin': { title: 'System Admin', icon: 'settings' },
  '/cmdb': { title: 'Configuration Management', icon: 'database' },
};

export const RecentVisitTracker: React.FC = () => {
  const pathname = usePathname();
  const addVisit = useRecentVisitStore(state => state.addVisit);

  useEffect(() => {
    if (!pathname) return;

    // 忽略登录等非主页面
    if (pathname.startsWith('/login') || pathname.startsWith('/auth')) {
      return;
    }

    // 获取路径元信息
    let meta = pathMeta[pathname];
    if (!meta) {
      // 匹配子路径，比如/tickets/123 -> 工单详情
      if (pathname.match(/^\/tickets\/\d+$/)) {
        meta = { title: 'Ticket Detail', icon: 'ticket' };
      } else if (pathname.match(/^\/incidents\/\d+$/)) {
        meta = { title: 'Incident Detail', icon: 'alert-circle' };
      } else if (pathname.match(/^\/problems\/\d+$/)) {
        meta = { title: 'Problem Detail', icon: 'alert-triangle' };
      } else if (pathname.match(/^\/knowledge\/\d+$/)) {
        meta = { title: 'Knowledge details', icon: 'book' };
      } else if (pathname.match(/^\/cmdb\/\w+$/)) {
        meta = { title: 'Configuration details', icon: 'database' };
      } else {
        // 未知路径，使用路径最后一段作为标题
        const segments = pathname.split('/').filter(Boolean);
        const lastSegment = segments.pop() || pathname;
        meta = { title: lastSegment.charAt(0).toUpperCase() + lastSegment.slice(1) };
      }
    }

    addVisit({
      path: pathname,
      title: meta.title,
      icon: meta.icon,
    });
  }, [pathname, addVisit]);

  return null;
};
