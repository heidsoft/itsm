"use client";

import React, { useState, useEffect } from "react";
import {
  Layout,
  Menu,
  Button,
  Avatar,
  Dropdown,
  Tooltip,
  Badge,
  theme,
  Space,
  Typography,
  Drawer,
  Input,
  List,
  message,
} from "antd";
import {
  LayoutDashboard,
  FileText,
  AlertTriangle,
  Settings,
  User,
  Search,
  LogOut,
  BookOpen,
  BarChart3,
  Database,
  HelpCircle,
  ArrowLeft,
  Workflow,
  Calendar,
  Bell,
  ChevronDown,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "../lib/store";
import { httpClient } from "../lib/http-client";

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

interface AppLayoutProps {
  children: React.ReactNode;
  title?: string;
  breadcrumb?: Array<{ title: string; href?: string }>;
  showBackButton?: boolean;
  extra?: React.ReactNode;
  showBreadcrumb?: boolean;
}

// 菜单配置 - 使用 Lucide 图标
const MENU_CONFIG = {
  main: [
    {
      key: "/dashboard",
      icon: <LayoutDashboard size={16} />,
      label: "仪表盘",
      path: "/dashboard",
      permission: "dashboard:view",
    },
    {
      key: "/tickets",
      icon: <FileText size={16} />,
      label: "工单管理",
      path: "/tickets",
      permission: "ticket:view",
    },
    {
      key: "/incidents",
      icon: <AlertTriangle size={16} />,
      label: "事件管理",
      path: "/incidents",
      permission: "incident:view",
    },
    {
      key: "/problems",
      icon: <HelpCircle size={16} />,
      label: "问题管理",
      path: "/problems",
      permission: "problem:view",
    },
    {
      key: "/changes",
      icon: <BarChart3 size={16} />,
      label: "变更管理",
      path: "/changes",
      permission: "change:view",
    },
    {
      key: "/cmdb",
      icon: <Database size={16} />,
      label: "配置管理",
      path: "/cmdb",
      permission: "cmdb:view",
    },
    {
      key: "/service-catalog",
      icon: <BookOpen size={16} />,
      label: "服务目录",
      path: "/service-catalog",
      permission: "service:view",
    },
    {
      key: "/knowledge-base",
      icon: <BookOpen size={16} />,
      label: "知识库",
      path: "/knowledge-base",
      permission: "knowledge:view",
    },
    {
      key: "/sla",
      icon: <Calendar size={16} />,
      label: "SLA管理",
      path: "/sla",
      permission: "sla:view",
    },
    {
      key: "/reports",
      icon: <BarChart3 size={16} />,
      label: "报表分析",
      path: "/reports",
      permission: "report:view",
    },
  ],
  admin: [
    {
      key: "/workflow",
      icon: <Workflow size={16} />,
      label: "工作流管理",
      path: "/workflow",
      permission: "workflow:config",
    },
    {
      key: "/admin",
      icon: <Settings size={16} />,
      label: "系统管理",
      path: "/admin",
      permission: "admin:view",
    },
  ],
};

// Logo 组件 - 使用 Ant Design 样式
const TechLogo = ({ collapsed }: { collapsed: boolean }) => {
  const { token } = theme.useToken();

  const [copilotOpen, setCopilotOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [tools, setTools] = useState<
    Array<{ name: string; description: string; read_only: boolean }>
  >([]);
  const [query, setQuery] = useState("");
  const [conversationId, setConversationId] = useState<number | null>(null);
  const [answers, setAnswers] = useState<any[]>([]);

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        cursor: "pointer",
        transition: "all 0.3s ease",
      }}
    >
      {/* Logo图标 */}
      <div
        style={{
          width: 36,
          height: 36,
          background: "linear-gradient(135deg, #1890ff 0%, #722ed1 100%)",
          borderRadius: "8px",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          marginRight: collapsed ? 0 : token.marginSM,
          position: "relative",
          boxShadow: "0 2px 8px rgba(24, 144, 255, 0.2)",
          transition: "all 0.3s ease",
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.transform = "scale(1.05)";
          e.currentTarget.style.boxShadow =
            "0 4px 12px rgba(24, 144, 255, 0.3)";
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.transform = "scale(1)";
          e.currentTarget.style.boxShadow = "0 2px 8px rgba(24, 144, 255, 0.2)";
        }}
      >
        {/* IT图标设计 */}
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          {/* 服务器图标 */}
          <rect
            x="3"
            y="4"
            width="18"
            height="4"
            rx="2"
            fill="white"
            fillOpacity="0.9"
          />
          <rect
            x="3"
            y="10"
            width="18"
            height="4"
            rx="2"
            fill="white"
            fillOpacity="0.7"
          />
          <rect
            x="3"
            y="16"
            width="18"
            height="4"
            rx="2"
            fill="white"
            fillOpacity="0.5"
          />
          {/* 连接点 */}
          <circle cx="6" cy="6" r="1" fill="#1890ff" />
          <circle cx="9" cy="6" r="1" fill="#1890ff" />
          <circle cx="6" cy="12" r="1" fill="#722ed1" />
          <circle cx="9" cy="12" r="1" fill="#722ed1" />
          <circle cx="6" cy="18" r="1" fill="#52c41a" />
          <circle cx="9" cy="18" r="1" fill="#52c41a" />
        </svg>

        {/* 状态指示器 */}
        <div
          style={{
            position: "absolute",
            top: -2,
            right: -2,
            width: 10,
            height: 10,
            background: "linear-gradient(45deg, #52c41a, #73d13d)",
            borderRadius: "50%",
            border: "2px solid white",
            animation: "pulse 2s infinite",
          }}
        />
      </div>

      {/* 文字部分 */}
      {!collapsed && (
        <div style={{ display: "flex", flexDirection: "column" }}>
          <Text
            strong
            style={{
              color: token.colorText,
              fontSize: "16px",
              fontWeight: 600,
              lineHeight: "20px",
              background: "linear-gradient(135deg, #1890ff, #722ed1)",
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
              backgroundClip: "text",
            }}
          >
            ITSM Pro
          </Text>
          <Text
            style={{
              color: token.colorTextSecondary,
              fontSize: "11px",
              fontWeight: 400,
              lineHeight: "14px",
              marginTop: "1px",
            }}
          >
            智能服务管理平台
          </Text>
        </div>
      )}
    </div>
  );
};

export const AppLayout: React.FC<AppLayoutProps> = ({
  children,
  title = "ITSM系统",
  breadcrumb = [],
  showBackButton = false,
  extra,
  showBreadcrumb = true,
}) => {
  const [collapsed, setCollapsed] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [userPermissions] = useState<string[]>([
    "dashboard:view",
    "ticket:view",
    "incident:view",
    "problem:view",
    "change:view",
    "cmdb:view",
    "service:view",
    "knowledge:view",
    "sla:view",
    "report:view",
    "admin:view",
    "user:manage",
    "workflow:config",
    "system:config",
  ]);

  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuthStore();
  const { token } = theme.useToken();
  // Copilot 抽屉状态
  const [copilotOpen, setCopilotOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [tools, setTools] = useState<
    Array<{ name: string; description: string; read_only: boolean }>
  >([]);
  const [query, setQuery] = useState("");
  const [conversationId, setConversationId] = useState<number | null>(null);
  const [answers, setAnswers] = useState<any[]>([]);

  // 检测屏幕尺寸
  useEffect(() => {
    const checkScreenSize = () => {
      setIsMobile(window.innerWidth < 768);
    };

    checkScreenSize();
    window.addEventListener("resize", checkScreenSize);
    return () => window.removeEventListener("resize", checkScreenSize);
  }, []);

  const hasPermission = (permission: string) => {
    return (
      userPermissions.includes(permission) || userPermissions.includes("admin")
    );
  };

  const getMenuItems = () => {
    const mainItems = MENU_CONFIG.main.filter((item) =>
      hasPermission(item.permission)
    );
    const adminItems = MENU_CONFIG.admin.filter((item) =>
      hasPermission(item.permission)
    );

    return [...mainItems, ...adminItems];
  };

  const getCurrentMenuKey = () => {
    const items = getMenuItems();
    for (const item of items) {
      if (item.path === pathname) {
        return item.key;
      }
    }
    return "/dashboard";
  };

  const handleMenuClick = ({ key }: { key: string }) => {
    const items = getMenuItems();
    for (const item of items) {
      if (item.key === key) {
        if (item.path) {
          router.push(item.path);
        }
        return;
      }
    }
  };

  const handleUserMenuClick = ({ key }: { key: string }) => {
    switch (key) {
      case "profile":
        router.push("/profile");
        break;
      case "settings":
        router.push("/settings");
        break;
      case "logout":
        logout();
        router.push("/login");
        break;
    }
  };

  // 用户菜单项
  const userMenuItems = [
    {
      key: "profile",
      icon: <User size={16} />,
      label: "个人资料",
    },
    {
      key: "settings",
      icon: <Settings size={16} />,
      label: "设置",
    },
    {
      type: "divider" as const,
    },
    {
      key: "logout",
      icon: <LogOut size={16} />,
      label: "退出登录",
    },
  ];

  return (
    <>
      <Layout style={{ minHeight: "100vh" }}>
        {/* 侧边栏 */}
        <Sider
          trigger={null}
          collapsible
          collapsed={collapsed}
          breakpoint="lg"
          collapsedWidth={64}
          width={240}
          onBreakpoint={(broken) => {
            console.log("Breakpoint:", broken);
          }}
          onCollapse={(collapsed, type) => {
            console.log("Collapsed:", collapsed, "Type:", type);
            setCollapsed(collapsed);
          }}
          style={{
            background: token.colorBgContainer,
            borderRight: `1px solid ${token.colorBorder}`,
            display: "flex",
            flexDirection: "column",
          }}
        >
          {/* Logo 区域 */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: collapsed ? "center" : "space-between",
              padding: collapsed
                ? `${token.paddingMD} ${token.paddingXS}`
                : `${token.paddingSM} ${token.paddingSM}`,
              borderBottom: `1px solid ${token.colorBorderSecondary}`,
              marginBottom: token.marginXS,
              height: 64, // 固定高度保持一致性
            }}
          >
            {!collapsed && <TechLogo collapsed={collapsed} />}
            {/* 收缩/展开按钮 - 放在侧边栏内部 */}
            <Button
              type="text"
              icon={
                collapsed ? (
                  <PanelLeftOpen size={16} />
                ) : (
                  <PanelLeftClose size={16} />
                )
              }
              onClick={() => setCollapsed(!collapsed)}
              title={collapsed ? "展开侧边栏" : "收缩侧边栏"}
              style={{
                color: token.colorTextSecondary,
                background: "transparent",
                border: "none",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                minWidth: 32,
                height: 32,
                borderRadius: token.borderRadius,
                transition: "all 0.2s ease",
              }}
              size="small"
              onMouseEnter={(e) => {
                e.currentTarget.style.background = token.colorFillQuaternary;
                e.currentTarget.style.color = token.colorText;
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = "transparent";
                e.currentTarget.style.color = token.colorTextSecondary;
              }}
            />
          </div>

          {/* 菜单 */}
          <Menu
            theme="light"
            mode="inline"
            selectedKeys={[getCurrentMenuKey()]}
            items={getMenuItems()}
            onClick={handleMenuClick}
            inlineCollapsed={collapsed}
            style={{
              border: "none",
              background: "transparent",
              flex: 1,
            }}
          />
        </Sider>

        <Layout>
          {/* 头部 */}
          <Header
            style={{
              padding: `0 ${token.paddingSM}`,
              background: token.colorBgContainer,
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              height: 64,
              borderBottom: `1px solid ${token.colorBorder}`,
            }}
          >
            {/* 左侧区域 */}
            <div
              style={{
                display: "flex",
                alignItems: "center",
                flex: 1,
                minWidth: 0,
              }}
            >
              {/* 返回按钮 */}
              {showBackButton && (
                <Button
                  type="text"
                  icon={<ArrowLeft size={16} />}
                  onClick={() => router.back()}
                  style={{
                    marginRight: token.marginMD,
                    flexShrink: 0,
                  }}
                >
                  返回
                </Button>
              )}

              {/* 页面标题和面包屑 */}
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  minWidth: 0,
                  flex: 1,
                }}
              >
                <div
                  style={{
                    width: 2,
                    height: 16,
                    background: "linear-gradient(to bottom, #1890ff, #722ed1)",
                    borderRadius: 1,
                    marginRight: token.marginSM,
                    flexShrink: 0,
                  }}
                />
                <div style={{ minWidth: 0, flex: 1 }}>
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      flexWrap: "wrap",
                      gap: token.marginXS,
                    }}
                  >
                    <Text
                      strong
                      style={{
                        fontSize: token.fontSizeLG,
                        whiteSpace: "nowrap",
                      }}
                    >
                      {title}
                    </Text>
                    {showBreadcrumb && breadcrumb.length > 0 && (
                      <div
                        style={{
                          display: "flex",
                          alignItems: "center",
                          marginLeft: token.marginSM,
                        }}
                      >
                        <Text type="secondary" style={{ margin: "0 4px" }}>
                          /
                        </Text>
                        <Space
                          split={
                            <Text type="secondary" style={{ margin: "0 4px" }}>
                              /
                            </Text>
                          }
                        >
                          {breadcrumb.map((item, index) => (
                            <Link key={index} href={item.href || "#"}>
                              <Text
                                type="secondary"
                                style={{
                                  fontSize: token.fontSizeSM,
                                  whiteSpace: "nowrap",
                                }}
                              >
                                {item.title}
                              </Text>
                            </Link>
                          ))}
                        </Space>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>

            {/* 右侧工具栏 */}
            <div
              style={{ display: "flex", alignItems: "center", flexShrink: 0 }}
            >
              <Space size="small">
                {/* 全局搜索 */}
                {!isMobile && (
                  <Tooltip title="全局搜索">
                    <Button
                      type="text"
                      icon={<Search size={16} />}
                      onClick={() => console.log("全局搜索功能")}
                    />
                  </Tooltip>
                )}

                {/* 通知中心 */}
                <Badge count={3} size="small">
                  <Button type="text" icon={<Bell size={16} />} />
                </Badge>

                {/* Copilot 按钮 */}
                <Button
                  type="primary"
                  onClick={async () => {
                    setCopilotOpen(true);
                    setLoading(true);
                    try {
                      const data = await httpClient.get<
                        Array<{
                          name: string;
                          description: string;
                          read_only: boolean;
                        }>
                      >("/api/v1/ai/tools");
                      setTools(data);
                    } catch (e: any) {
                      message.error(e?.message || "加载工具失败");
                    } finally {
                      setLoading(false);
                    }
                  }}
                >
                  Copilot
                </Button>

                {/* 用户菜单 */}
                <Dropdown
                  menu={{
                    items: userMenuItems,
                    onClick: handleUserMenuClick,
                  }}
                  placement="bottomRight"
                >
                  <Button
                    type="text"
                    style={{ height: "auto", padding: token.paddingXS }}
                  >
                    <Space>
                      <Avatar
                        size="small"
                        icon={<User size={16} />}
                        style={{
                          background:
                            "linear-gradient(135deg, #1890ff, #722ed1)",
                        }}
                      />
                      {/* 在小屏幕上隐藏用户信息文字 */}
                      {!isMobile && (
                        <div style={{ textAlign: "left" }}>
                          <div
                            style={{
                              fontSize: token.fontSizeSM,
                              fontWeight: 500,
                            }}
                          >
                            {user?.name || "用户"}
                          </div>
                          <div
                            style={{
                              fontSize: 10,
                              color: token.colorTextSecondary,
                            }}
                          >
                            在线
                          </div>
                        </div>
                      )}
                      {!isMobile && (
                        <ChevronDown
                          size={12}
                          style={{ color: token.colorTextSecondary }}
                        />
                      )}
                    </Space>
                  </Button>
                </Dropdown>
              </Space>
            </div>
          </Header>

          {/* 主内容区域 */}
          <Content
            style={{
              margin: token.margin,
              minHeight: `calc(100vh - ${64 + token.margin * 2}px)`,
            }}
          >
            {/* 额外操作区域 */}
            {extra && (
              <div
                style={{
                  padding: token.paddingLG,
                  marginBottom: token.margin,
                  background: token.colorBgContainer,
                  borderRadius: token.borderRadius,
                  border: `1px solid ${token.colorBorder}`,
                }}
              >
                {extra}
              </div>
            )}

            {/* 主要内容 */}
            <div
              style={{
                padding: token.paddingLG,
                minHeight: 360,
                background: token.colorBgContainer,
                borderRadius: token.borderRadius,
                border: `1px solid ${token.colorBorder}`,
              }}
            >
              {children}
            </div>
          </Content>
        </Layout>
      </Layout>

      {/* Copilot 抽屉 */}
      <Drawer
        title="Copilot"
        placement="right"
        width={420}
        open={copilotOpen}
        onClose={() => setCopilotOpen(false)}
      >
        <Space direction="vertical" style={{ width: "100%" }} size="middle">
          <Input.TextArea
            rows={3}
            placeholder="输入你的问题..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <Button
            loading={loading}
            type="primary"
            onClick={async () => {
              if (!query.trim()) return;
              setLoading(true);
              try {
                const res = await httpClient.post<{
                  answers: any[];
                  conversation_id: number;
                }>("/api/v1/ai/chat", {
                  query,
                  limit: 5,
                  conversation_id: conversationId || undefined,
                });
                setAnswers(res.answers || []);
                if (res.conversation_id) {
                  setConversationId(res.conversation_id);
                }
              } catch (e: any) {
                message.error(e?.message || "对话失败");
              } finally {
                setLoading(false);
              }
            }}
          >
            发送
          </Button>

          <Typography.Title level={5}>工具</Typography.Title>
          <List
            loading={loading}
            dataSource={tools}
            renderItem={(item) => (
              <List.Item
                actions={[
                  <Button
                    key="exec"
                    size="small"
                    onClick={async () => {
                      try {
                        const res = await httpClient.post<any>(
                          "/api/v1/ai/tools/execute",
                          { name: item.name, args: {} }
                        );
                        if (res?.invocation_id) {
                          message.success("已提交，等待审批");
                        } else {
                          message.success("执行成功");
                        }
                      } catch (e: any) {
                        message.error(e?.message || "执行失败");
                      }
                    }}
                  >
                    {item.read_only ? "执行" : "申请审批"}
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={item.name}
                  description={item.description}
                />
              </List.Item>
            )}
          />

          <Typography.Title level={5}>回答</Typography.Title>
          <List
            bordered
            dataSource={answers}
            locale={{ emptyText: "暂无结果" }}
            renderItem={(a: any) => (
              <List.Item>
                <div style={{ width: "100%" }}>
                  {a.title && <div style={{ fontWeight: 600 }}>{a.title}</div>}
                  {a.snippet && (
                    <div style={{ color: "#666", marginTop: 4 }}>
                      {a.snippet}
                    </div>
                  )}
                  <div style={{ marginTop: 6, fontSize: 12, color: "#999" }}>
                    {a.source && <span>来源: {a.source} </span>}
                    {a.score !== undefined && (
                      <span>
                        {" "}
                        置信度: {a.score.toFixed ? a.score.toFixed(3) : a.score}
                      </span>
                    )}
                  </div>
                </div>
              </List.Item>
            )}
          />
        </Space>
      </Drawer>
    </>
  );
};

export default AppLayout;
