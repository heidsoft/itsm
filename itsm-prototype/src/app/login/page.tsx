"use client";

import React, { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { User, AlertCircle, Shield, ArrowRight } from "lucide-react";
import { Typography, Space, Alert, Divider } from "antd";
import { Button, Input, PasswordInput } from "@/components/ui";
import { AuthLayout, AuthCard } from "@/components/auth/AuthLayout";
import { theme } from "antd";

const { Text } = Typography;

/**
 * 登录页面组件
 * 使用统一的设计系统和Ant Design组件，保持与系统内部一致的视觉风格
 */
export default function LoginPage() {
  const router = useRouter();
  const { token } = theme.useToken();

  // 表单状态管理
  const [formData, setFormData] = useState({
    username: "",
    password: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [isFormValid, setIsFormValid] = useState(false);

  // 实时表单验证
  useEffect(() => {
    setIsFormValid(
      formData.username.trim().length > 0 && formData.password.length >= 6
    );
  }, [formData]);

  // 模拟认证服务
  const mockAuthService = {
    async login(username: string, password: string) {
      // 模拟网络延迟
      await new Promise((resolve) => setTimeout(resolve, 1500));

      // 模拟认证逻辑
      if (username === "admin" && password === "admin123") {
        return {
          success: true,
          user: { id: 1, username: "admin", name: "系统管理员" },
          token: "mock-jwt-token",
        };
      }

      throw new Error("用户名或密码错误");
    },
  };

  // 处理输入变化
  const handleInputChange = (field: string, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (error) setError(""); // 清除错误信息
  };

  // 处理登录提交
  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.username.trim() || !formData.password) {
      setError("请输入用户名和密码");
      return;
    }

    setLoading(true);
    setError("");

    try {
      const result = await mockAuthService.login(
        formData.username,
        formData.password
      );

      if (result.success) {
        // 存储认证信息
        localStorage.setItem("auth_token", result.token);
        localStorage.setItem("user_info", JSON.stringify(result.user));

        // 跳转到仪表板
        router.push("/dashboard");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败，请重试");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout>
      <AuthCard title="欢迎回来" subtitle="请登录您的账户以继续使用服务">
        {/* 错误提示 */}
        {error && (
          <Alert
            message={error}
            type="error"
            icon={<AlertCircle />}
            style={{ marginBottom: token.marginLG }}
            showIcon
          />
        )}

        {/* 登录表单 */}
        <form onSubmit={handleLogin} className="space-y-4">
          {/* 用户名输入 */}
          <Input
            label="用户名"
            placeholder="请输入用户名"
            prefix={<User size={16} />}
            value={formData.username}
            onChange={(e) => handleInputChange("username", e.target.value)}
            disabled={loading}
            required
            size="lg"
          />

          {/* 密码输入 */}
          <PasswordInput
            label="密码"
            placeholder="请输入密码"
            value={formData.password}
            onChange={(e) => handleInputChange("password", e.target.value)}
            disabled={loading}
            required
            size="lg"
            showStrength={false}
          />

          {/* 记住我和忘记密码 */}
          <div className="flex items-center justify-between">
            <label className="flex items-center">
              <input
                type="checkbox"
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                disabled={loading}
              />
              <span
                className="ml-2 text-sm"
                style={{ color: token.colorTextSecondary }}
              >
                记住我
              </span>
            </label>
            <button
              type="button"
              className="text-sm font-medium hover:underline"
              style={{ color: token.colorPrimary }}
              disabled={loading}
            >
              忘记密码？
            </button>
          </div>

          {/* 登录按钮 */}
          <Button
            type="submit"
            variant="primary"
            size="lg"
            fullWidth
            loading={loading}
            disabled={!isFormValid}
            icon={<ArrowRight size={16} />}
            iconPosition="right"
            style={{
              marginTop: token.marginLG,
              height: token.controlHeightLG,
            }}
          >
            {loading ? "登录中..." : "登录"}
          </Button>
        </form>

        {/* 其他登录方式 */}
        <Divider style={{ margin: `${token.marginLG}px 0` }}>
          <Text style={{ color: token.colorTextTertiary }}>或</Text>
        </Divider>

        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Button
            variant="outline"
            size="lg"
            fullWidth
            disabled={loading}
            style={{ height: token.controlHeightLG }}
          >
            <Shield size={16} className="mr-2" />
            SSO 单点登录
          </Button>
        </Space>

        {/* 底部链接 */}
        <div className="text-center mt-6">
          <Text style={{ color: token.colorTextTertiary }}>
            还没有账户？{" "}
            <button
              type="button"
              className="font-medium hover:underline"
              style={{ color: token.colorPrimary }}
            >
              立即注册
            </button>
          </Text>
        </div>
      </AuthCard>
    </AuthLayout>
  );
}
