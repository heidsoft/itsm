'use client';

import { AuthGuard } from '@/components/auth/AuthGuard';
import { Sidebar } from '@/components/layout/Sidebar';
import { Header } from '@/components/layout/Header';
import { useUIStore } from '@/lib/store/ui-store';

/**
 * 仪表盘布局组件
 * 包含侧边栏、顶部导航和主内容区域
 */
export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { sidebar, setSidebarCollapsed } = useUIStore();

  const toggleSidebar = () => {
    setSidebarCollapsed(!sidebar.isCollapsed);
  };

  return (
    <AuthGuard>
      <div className="flex h-screen bg-gray-50">
        <Sidebar collapsed={sidebar.isCollapsed} onToggle={toggleSidebar} />
        <div className="flex-1 flex flex-col overflow-hidden">
          <Header />
          <main className="flex-1 overflow-auto p-6">
            {children}
          </main>
        </div>
      </div>
    </AuthGuard>
  );
}