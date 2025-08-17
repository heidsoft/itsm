"use client";

import React, { useState } from "react";
import { Layout, Button, Avatar, Dropdown, Tooltip, Badge, Space, Typography, Drawer, Input, List, message } from "antd";
import {
  User,
  Search,
  LogOut,
  Bell,
  ChevronDown,
  PanelLeftClose,
  PanelLeftOpen,
  Settings,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "../../lib/store";

const { Header: AntHeader } = Layout;
const { Text } = Typography;

interface HeaderProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
  title?: string;
  breadcrumb?: Array<{ title: string; href?: string }>;
  showBackButton?: boolean;
  extra?: React.ReactNode;
  showBreadcrumb?: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  collapsed,
  onCollapse,
  title,
  breadcrumb,
  showBackButton = false,
  extra,
  showBreadcrumb = true,
}) => {
  const router = useRouter();
  const { user, logout } = useAuthStore();
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  const handleLogout = () => {
    logout();
    router.push("/login");
    message.success("已安全退出");
  };

  const userMenuItems = [
    {
      key: "profile",
      label: "个人资料",
      icon: <User size={16} />,
      onClick: () => router.push("/profile"),
    },
    {
      key: "settings",
      label: "设置",
      icon: <Settings size={16} />,
      onClick: () => router.push("/settings"),
    },
    {
      type: "divider" as const,
    },
    {
      key: "logout",
      label: "退出登录",
      icon: <LogOut size={16} />,
      onClick: handleLogout,
    },
  ];

  const notifications = [
    {
      id: 1,
      title: "新工单分配",
      content: "您有一个新的高优先级工单需要处理",
      time: "2分钟前",
      read: false,
    },
    {
      id: 2,
      title: "系统维护通知",
      content: "系统将在今晚进行维护，预计停机2小时",
      time: "1小时前",
      read: true,
    },
  ];

  return (
    <AntHeader
      style={{
        padding: "0 24px",
        background: "#fff",
        borderBottom: "1px solid #f0f0f0",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        height: 64,
      }}
    >
      {/* 左侧区域 */}
      <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
        {/* 折叠按钮 */}
        <Button
          type="text"
          icon={collapsed ? <PanelLeftOpen size={20} /> : <PanelLeftClose size={20} />}
          onClick={() => onCollapse(!collapsed)}
          style={{
            fontSize: "16px",
            width: 40,
            height: 40,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        />

        {/* 面包屑导航 - 只在有面包屑时显示 */}
        {showBreadcrumb && breadcrumb && breadcrumb.length > 0 && (
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            {breadcrumb.map((item, index) => (
              <React.Fragment key={index}>
                {index > 0 && <Text style={{ color: "#9ca3af" }}>/</Text>}
                {item.href ? (
                  <Text
                    style={{
                      color: "#6b7280",
                      cursor: "pointer",
                      fontSize: "14px",
                    }}
                    onClick={() => router.push(item.href!)}
                  >
                    {item.title}
                  </Text>
                ) : (
                  <Text style={{ color: "#374151", fontSize: "14px" }}>
                    {item.title}
                  </Text>
                )}
              </React.Fragment>
            ))}
          </div>
        )}
      </div>

      {/* 右侧区域 */}
      <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
        {/* 搜索框 */}
        <Input
          placeholder="搜索..."
          prefix={<Search size={16} style={{ color: "#9ca3af" }} />}
          style={{
            width: 240,
            borderRadius: "6px",
          }}
        />

        {/* 通知 */}
        <Tooltip title="通知">
          <Badge count={notifications.filter(n => !n.read).length} size="small">
            <Button
              type="text"
              icon={<Bell size={20} />}
              onClick={() => setNotificationsOpen(true)}
              style={{
                width: 40,
                height: 40,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            />
          </Badge>
        </Tooltip>

        {/* 用户菜单 */}
        <Dropdown
          menu={{ items: userMenuItems }}
          placement="bottomRight"
          trigger={["click"]}
          open={userMenuOpen}
          onOpenChange={setUserMenuOpen}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              cursor: "pointer",
              padding: "4px 8px",
              borderRadius: "6px",
              transition: "background-color 0.2s",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "#f3f4f6";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = "transparent";
            }}
          >
            <Avatar
              size={32}
              style={{
                backgroundColor: "#3b82f6",
                color: "#fff",
                fontWeight: "600",
              }}
            >
              {user?.name?.[0] || user?.username?.[0] || "U"}
            </Avatar>
            <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-start" }}>
              <Text style={{ fontSize: "14px", fontWeight: "500", color: "#1f2937" }}>
                {user?.name || user?.username}
              </Text>
              <Text style={{ fontSize: "12px", color: "#6b7280" }}>
                {user?.role === "admin" ? "管理员" : "用户"}
              </Text>
            </div>
            <ChevronDown size={16} style={{ color: "#9ca3af" }} />
          </div>
        </Dropdown>
      </div>

      {/* 通知抽屉 */}
      <Drawer
        title="通知中心"
        placement="right"
        width={400}
        open={notificationsOpen}
        onClose={() => setNotificationsOpen(false)}
      >
        <List
          dataSource={notifications}
          renderItem={(item) => (
            <List.Item
              style={{
                padding: "16px 0",
                borderBottom: "1px solid #f3f4f6",
                opacity: item.read ? 0.7 : 1,
              }}
            >
              <List.Item.Meta
                title={
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <Text style={{ fontWeight: item.read ? "400" : "600" }}>
                      {item.title}
                    </Text>
                    <Text style={{ fontSize: "12px", color: "#9ca3af" }}>
                      {item.time}
                    </Text>
                  </div>
                }
                description={
                  <Text style={{ color: "#6b7280", fontSize: "14px" }}>
                    {item.content}
                  </Text>
                }
              />
            </List.Item>
          )}
        />
      </Drawer>
    </AntHeader>
  );
};
