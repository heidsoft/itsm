import React, { useState } from "react";
import { Menu, X, Search, Bell } from "lucide-react";
import { GlobalSearch } from "./GlobalSearch";
import { NotificationCenter } from "./NotificationCenter";

interface MobileLayoutProps {
  children: React.ReactNode;
}

export const MobileLayout: React.FC<MobileLayoutProps> = ({ children }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);

  return (
    <div className="flex h-screen bg-gray-100">
      {/* 移动端侧边栏遮罩 */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 z-40 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* 侧边栏 */}
      <aside
        className={`
        fixed inset-y-0 left-0 z-50 w-64 bg-gray-900 transform transition-transform duration-300 ease-in-out lg:translate-x-0 lg:static lg:inset-0
        ${sidebarOpen ? "translate-x-0" : "-translate-x-full"}
      `}
      >
        {/* 侧边栏内容 */}
        <div className="flex items-center justify-between h-16 px-4 border-b border-gray-700">
          <h1 className="text-xl font-bold text-white">ITSM Pro</h1>
          <button
            onClick={() => setSidebarOpen(false)}
            className="text-gray-400 hover:text-white lg:hidden"
          >
            <X className="w-6 h-6" />
          </button>
        </div>
        {/* 导航菜单 */}
        <nav className="mt-4 px-2">{/* 这里放置导航菜单项 */}</nav>
      </aside>

      {/* 主内容区域 */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* 移动端顶部栏 */}
        <header className="bg-white border-b border-gray-200 lg:hidden">
          <div className="flex items-center justify-between h-16 px-4">
            <button
              onClick={() => setSidebarOpen(true)}
              className="text-gray-600 hover:text-gray-900"
            >
              <Menu className="w-6 h-6" />
            </button>

            <div className="flex items-center space-x-2">
              <button
                onClick={() => setSearchOpen(true)}
                className="text-gray-600 hover:text-gray-900 p-2"
              >
                <Search className="w-6 h-6" />
              </button>
              <NotificationCenter />
            </div>
          </div>
        </header>

        {/* 移动端全屏搜索 */}
        {searchOpen && (
          <div className="fixed inset-0 bg-white z-50 lg:hidden">
            <div className="flex items-center h-16 px-4 border-b border-gray-200">
              <button
                onClick={() => setSearchOpen(false)}
                className="text-gray-600 hover:text-gray-900 mr-4"
              >
                <X className="w-6 h-6" />
              </button>
              <GlobalSearch className="flex-1" placeholder="搜索..." />
            </div>
          </div>
        )}

        {/* 主内容 */}
        <main className="flex-1 overflow-auto">{children}</main>
      </div>
    </div>
  );
};
