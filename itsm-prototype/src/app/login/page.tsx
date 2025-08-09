"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { AuthService } from "../lib/auth-service";
import { Form, Input, Button, Card, Typography, Space, Alert } from "antd";
import {
  User,
  Lock,
  Globe,
  AlertCircle,
  Shield,
  Zap,
  ArrowRight,
  Sparkles,
} from "lucide-react";

const { Title, Text } = Typography;

interface LoginFormValues {
  username: string;
  password: string;
  tenantCode?: string;
}

export default function LoginPage() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [showTenantField, setShowTenantField] = useState(false);
  const router = useRouter();

  const handleSubmit = async (values: LoginFormValues) => {
    setIsLoading(true);
    setError("");

    try {
      console.log("开始登录...", {
        username: values.username,
        password: "***",
        tenantCode: values.tenantCode,
      });

      const success = await AuthService.login(
        values.username,
        values.password,
        values.tenantCode || undefined
      );

      console.log("登录结果:", success);
      if (success) {
        console.log("登录成功，准备跳转到dashboard");
        router.push("/dashboard");
      } else {
        console.log("登录失败");
        setError("用户名或密码错误");
      }
    } catch (error) {
      console.error("Login error:", error);
      setError("登录失败，请稍后重试");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div
      className="login-page"
      style={{
        minHeight: "100vh",
        background:
          "linear-gradient(135deg, #0a0f1c 0%, #1a1f2e 50%, #2d3748 100%)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: "2rem",
        position: "relative",
        overflow: "hidden",
      }}
    >
      {/* 高级背景效果 */}
      <div
        style={{
          position: "absolute",
          top: "0",
          left: "0",
          width: "100%",
          height: "100%",
          background: `
            radial-gradient(circle at 25% 25%, rgba(59, 130, 246, 0.1) 0%, transparent 50%),
            radial-gradient(circle at 75% 75%, rgba(147, 51, 234, 0.1) 0%, transparent 50%),
            radial-gradient(circle at 50% 50%, rgba(16, 185, 129, 0.05) 0%, transparent 50%)
          `,
        }}
      />

      {/* 动态粒子效果 */}
      <div
        style={{
          position: "absolute",
          top: "20%",
          left: "10%",
          width: "2px",
          height: "2px",
          background: "#3b82f6",
          borderRadius: "50%",
          boxShadow: "0 0 10px #3b82f6",
          animation: "float1 8s ease-in-out infinite",
        }}
      />
      <div
        style={{
          position: "absolute",
          top: "60%",
          right: "15%",
          width: "3px",
          height: "3px",
          background: "#10b981",
          borderRadius: "50%",
          boxShadow: "0 0 15px #10b981",
          animation: "float2 10s ease-in-out infinite",
        }}
      />
      <div
        style={{
          position: "absolute",
          top: "30%",
          right: "30%",
          width: "1px",
          height: "1px",
          background: "#8b5cf6",
          borderRadius: "50%",
          boxShadow: "0 0 8px #8b5cf6",
          animation: "float3 12s ease-in-out infinite",
        }}
      />

      {/* 主要内容容器 */}
      <div
        style={{
          width: "100%",
          maxWidth: "480px",
          position: "relative",
          zIndex: 10,
        }}
      >
        {/* 品牌区域 */}
        <div
          style={{
            textAlign: "center",
            marginBottom: "3rem",
          }}
        >
          {/* 主 Logo */}
          <div
            style={{
              display: "inline-flex",
              alignItems: "center",
              justifyContent: "center",
              width: "80px",
              height: "80px",
              background: "linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)",
              borderRadius: "20px",
              boxShadow:
                "0 20px 40px rgba(59, 130, 246, 0.3), 0 0 0 1px rgba(59, 130, 246, 0.1)",
              marginBottom: "1.5rem",
              position: "relative",
              overflow: "hidden",
            }}
          >
            <div
              style={{
                position: "absolute",
                top: "0",
                left: "0",
                width: "100%",
                height: "100%",
                background:
                  "linear-gradient(45deg, transparent 30%, rgba(255,255,255,0.1) 50%, transparent 70%)",
                animation: "shimmer 3s ease-in-out infinite",
              }}
            />
            <Shield
              style={{
                width: "40px",
                height: "40px",
                color: "white",
                zIndex: 1,
              }}
            />
            <div
              style={{
                position: "absolute",
                top: "-3px",
                right: "-3px",
                width: "24px",
                height: "24px",
                background: "linear-gradient(135deg, #10b981 0%, #059669 100%)",
                borderRadius: "50%",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                boxShadow: "0 4px 8px rgba(16, 185, 129, 0.3)",
              }}
            >
              <Sparkles
                style={{
                  width: "12px",
                  height: "12px",
                  color: "white",
                }}
              />
            </div>
          </div>

          {/* 品牌标题 */}
          <div style={{ marginBottom: "1rem" }}>
            <Title
              level={1}
              style={{
                fontSize: "2.5rem",
                fontWeight: "800",
                color: "#f1f5f9",
                marginBottom: "0.75rem",
                margin: 0,
                background: "linear-gradient(135deg, #f1f5f9 0%, #cbd5e1 100%)",
                WebkitBackgroundClip: "text",
                WebkitTextFillColor: "transparent",
                backgroundClip: "text",
              }}
            >
              ITSM
            </Title>
            <Text
              style={{
                fontSize: "1.1rem",
                color: "#94a3b8",
                fontWeight: "500",
                display: "block",
                marginBottom: "1rem",
              }}
            >
              智能服务管理平台
            </Text>
          </div>

          {/* 功能标签 */}
          <Space size="middle" wrap>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "6px",
                padding: "6px 12px",
                background: "rgba(59, 130, 246, 0.1)",
                borderRadius: "20px",
                border: "1px solid rgba(59, 130, 246, 0.2)",
              }}
            >
              <Sparkles size={12} style={{ color: "#3b82f6" }} />
              <Text style={{ fontSize: "0.8rem", color: "#94a3b8" }}>
                智能管理
              </Text>
            </div>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "6px",
                padding: "6px 12px",
                background: "rgba(16, 185, 129, 0.1)",
                borderRadius: "20px",
                border: "1px solid rgba(16, 185, 129, 0.2)",
              }}
            >
              <Zap size={12} style={{ color: "#10b981" }} />
              <Text style={{ fontSize: "0.8rem", color: "#94a3b8" }}>
                高效服务
              </Text>
            </div>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "6px",
                padding: "6px 12px",
                background: "rgba(139, 92, 246, 0.1)",
                borderRadius: "20px",
                border: "1px solid rgba(139, 92, 246, 0.2)",
              }}
            >
              <Shield size={12} style={{ color: "#8b5cf6" }} />
              <Text style={{ fontSize: "0.8rem", color: "#94a3b8" }}>
                安全可靠
              </Text>
            </div>
          </Space>
        </div>

        {/* 登录表单 */}
        <Card
          style={{
            background: "rgba(30, 41, 59, 0.9)",
            borderRadius: "24px",
            boxShadow:
              "0 32px 64px -12px rgba(0, 0, 0, 0.6), 0 0 0 1px rgba(59, 130, 246, 0.1)",
            border: "1px solid rgba(59, 130, 246, 0.2)",
            backdropFilter: "blur(24px)",
            position: "relative",
          }}
          styles={{
            body: {
              padding: "2rem",
            },
          }}
        >
          {/* 表单头部 */}
          <div style={{ textAlign: "center", marginBottom: "2rem" }}>
            <Title
              level={3}
              style={{
                fontSize: "1.5rem",
                fontWeight: "700",
                color: "#f1f5f9",
                marginBottom: "0.5rem",
                margin: 0,
              }}
            >
              系统登录
            </Title>
            <Text
              style={{
                fontSize: "0.9rem",
                color: "#94a3b8",
                fontWeight: "500",
              }}
            >
              请输入您的账户信息
            </Text>
          </div>

          {/* 错误提示 */}
          {error && (
            <Alert
              message={error}
              type="error"
              showIcon
              icon={<AlertCircle size={16} />}
              style={{
                marginBottom: "1.5rem",
                background: "rgba(239, 68, 68, 0.1)",
                border: "1px solid rgba(239, 68, 68, 0.3)",
                borderRadius: "12px",
              }}
            />
          )}

          <Form
            name="login"
            onFinish={handleSubmit}
            layout="vertical"
            size="large"
            style={{
              display: "flex",
              flexDirection: "column",
              gap: "1.5rem",
            }}
          >
            {/* 用户名输入框 */}
            <Form.Item
              name="username"
              label={
                <Text
                  style={{
                    color: "#cbd5e1",
                    fontSize: "0.9rem",
                    fontWeight: "600",
                  }}
                >
                  用户名
                </Text>
              }
              rules={[{ required: true, message: "请输入用户名" }]}
              style={{ marginBottom: 0 }}
            >
              <Input
                prefix={
                  <User
                    size={18}
                    style={{
                      color: "#64748b",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                    }}
                  />
                }
                placeholder="请输入用户名"
                autoComplete="username"
                size="large"
              />
            </Form.Item>

            {/* 密码输入框 */}
            <Form.Item
              name="password"
              label={
                <Text
                  style={{
                    color: "#cbd5e1",
                    fontSize: "0.9rem",
                    fontWeight: "600",
                  }}
                >
                  密码
                </Text>
              }
              rules={[{ required: true, message: "请输入密码" }]}
              style={{ marginBottom: 0 }}
            >
              <Input.Password
                prefix={
                  <Lock
                    size={18}
                    style={{
                      color: "#64748b",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                    }}
                  />
                }
                placeholder="请输入密码"
                autoComplete="current-password"
                size="large"
              />
            </Form.Item>

            {/* 租户代码输入框 */}
            {showTenantField && (
              <Form.Item
                name="tenantCode"
                label={
                  <Text
                    style={{
                      color: "#cbd5e1",
                      fontSize: "0.9rem",
                      fontWeight: "600",
                    }}
                  >
                    租户代码
                  </Text>
                }
                style={{ marginBottom: 0 }}
              >
                <Input
                  prefix={
                    <Globe
                      size={18}
                      style={{
                        color: "#64748b",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                      }}
                    />
                  }
                  placeholder="请输入租户代码（可选）"
                  autoComplete="organization"
                  size="large"
                />
              </Form.Item>
            )}

            {/* 登录按钮 */}
            <Form.Item style={{ marginBottom: 0, marginTop: "2rem" }}>
              <Button
                type="primary"
                htmlType="submit"
                loading={isLoading}
                block
                size="large"
                icon={<ArrowRight size={20} />}
                style={{
                  background:
                    "linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)",
                  border: "none",
                  borderRadius: "12px",
                  height: "52px",
                  fontSize: "1.1rem",
                  fontWeight: "600",
                  boxShadow: "0 8px 24px rgba(59, 130, 246, 0.3)",
                }}
              >
                登录系统
              </Button>
            </Form.Item>

            {/* 租户切换 */}
            <Form.Item style={{ marginBottom: 0, marginTop: "1.5rem" }}>
              <Button
                type="link"
                onClick={() => setShowTenantField(!showTenantField)}
                style={{
                  color: "#94a3b8",
                  fontSize: "0.9rem",
                  padding: "6px 12px",
                  height: "auto",
                }}
              >
                {showTenantField ? "隐藏租户设置" : "显示租户设置"}
              </Button>
            </Form.Item>
          </Form>
        </Card>
      </div>

      {/* CSS 动画 */}
      <style jsx>{`
        @keyframes float1 {
          0%,
          100% {
            transform: translateY(0px) rotate(0deg);
          }
          50% {
            transform: translateY(-20px) rotate(180deg);
          }
        }
        @keyframes float2 {
          0%,
          100% {
            transform: translateY(0px) rotate(0deg);
          }
          50% {
            transform: translateY(-15px) rotate(-180deg);
          }
        }
        @keyframes float3 {
          0%,
          100% {
            transform: translateY(0px) rotate(0deg);
          }
          50% {
            transform: translateY(-25px) rotate(90deg);
          }
        }
        @keyframes shimmer {
          0% {
            transform: translateX(-100%);
          }
          100% {
            transform: translateX(100%);
          }
        }
      `}</style>
    </div>
  );
}
