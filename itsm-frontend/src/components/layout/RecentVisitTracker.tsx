'use client';

import { useEffect } from 'react';
import { usePathname } from 'next/navigation';
import { useRecentVisitStore } from '@/lib/store/recent-visit-store';

// 路径到标题和图标映射
const pathMeta: Record<string, { title: string; icon?: string }> = {
  '/dashboard': { title: '首页', icon: 'home' },
  '/tickets': { title: '工单管理', icon: 'ticket' },
  '/incidents': { title: '事件管理', icon: 'alert-circle' },
  '/problems': { title: '问题管理', icon: 'alert-triangle' },
  '/changes': { title: '变更管理', icon: 'refresh-cw' },
  '/knowledge': { title: '知识库', icon: 'book' },
  '/service-catalog': { title: '服务目录', icon: 'grid' },
  '/profile': { title: '个人中心', icon: 'user' },
  '/notifications': { title: '通知中心', icon: 'bell' },
  '/admin': { title: '系统管理', icon: 'settings' },
  '/cmdb': { title: '配置管理', icon: 'database' },
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
        meta = { title: '工单详情', icon: 'ticket' };
      } else if (pathname.match(/^\/incidents\/\d+$/)) {
        meta = { title: '事件详情', icon: 'alert-circle' };
      } else if (pathname.match(/^\/problems\/\d+$/)) {
        meta = { title: '问题详情', icon: 'alert-triangle' };
      } else if (pathname.match(/^\/knowledge\/\d+$/)) {
        meta = { title: '知识详情', icon: 'book' };
      } else if (pathname.match(/^\/cmdb\/\w+$/)) {
        meta = { title: '配置详情', icon: 'database' };
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
