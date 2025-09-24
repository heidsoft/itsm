'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { 
  Shield, 
  Sparkles, 
  Zap, 
  User, 
  Lock, 
  Globe, 
  ArrowRight, 
  AlertCircle 
} from 'lucide-react';
import { 
  Card, 
  Form, 
  Input, 
  Button, 
  Typography, 
  Space, 
  Alert 
} from 'antd';
import { AuthService } from '@/lib/auth/auth-service';
import { useNotifications } from '@/lib/store/ui-store';

const { Title, Text } = Typography;

/**
 * 登录页面组件
 */
export default function LoginPage() {
  const router = useRouter();
  const { success, error: showError } = useNotifications();
  
  // 状态管理
  const [isLoading, setIsLoading] = useState(false);
  const [showTenantField, setShowTenantField] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    tenantCode: ''
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  // 表单验证
  const validateForm = () => {
    const newErrors: Record<string, string> = {};
    
    if (!formData.username.trim()) {
      newErrors.username = '请输入用户名';
    }
    
    if (!formData.password) {
      newErrors.password = '请输入密码';
    } else if (formData.password.length < 6) {
      newErrors.password = '密码长度至少6位';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // 处理登录提交
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    setIsLoading(true);
    setErrorMessage(''); // 清除之前的错误信息
    
    try {
      await AuthService.login(formData);
      success('登录成功', '欢迎回来！');
      router.push('/dashboard');
    } catch (err) {
      console.error('Login failed:', err);
      const errorMsg = err instanceof Error ? err.message : '用户名或密码错误';
      setErrorMessage(errorMsg);
      showError('登录失败', errorMsg);
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
      {/* Advanced background effects */}
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

      {/* Dynamic particle effects */}
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

      {/* Main content container */}
      <div
        style={{
          width: "100%",
          maxWidth: "480px",
          position: "relative",
          zIndex: 10,
        }}
      >
        {/* Brand area */}
        <div
          style={{
            textAlign: "center",
            marginBottom: "3rem",
          }}
        >
          {/* Main Logo */}
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

          {/* Brand title */}
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
              Intelligent Service Management Platform
            </Text>
          </div>

          {/* Feature tags */}
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
                Smart Management
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
                Efficient Service
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
                Secure & Reliable
              </Text>
            </div>
          </Space>
        </div>

        {/* Login form */}
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
          {/* Form header */}
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
              System Login
            </Title>
            <Text
              style={{
                fontSize: "0.9rem",
                color: "#94a3b8",
                fontWeight: "500",
              }}
            >
              Please enter your account information
            </Text>
          </div>

          {/* Error message */}
          {errorMessage && (
            <Alert
              message={errorMessage}
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
            {/* Username input */}
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
                  Username
                </Text>
              }
              rules={[{ required: true, message: "Please enter username" }]}
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
                placeholder="Please enter username"
                autoComplete="username"
                size="large"
                value={formData.username}
                onChange={(e) => setFormData({ ...formData, username: e.target.value })}
              />
            </Form.Item>

            {/* Password input */}
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
                  Password
                </Text>
              }
              rules={[{ required: true, message: "Please enter password" }]}
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
                placeholder="Please enter password"
                autoComplete="current-password"
                size="large"
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
              />
            </Form.Item>

            {/* Tenant code input */}
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
                      Tenant Code
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
                    placeholder="Please enter tenant code (optional)"
                    autoComplete="organization"
                    size="large"
                    value={formData.tenantCode}
                    onChange={(e) => setFormData({ ...formData, tenantCode: e.target.value })}
                  />
                </Form.Item>
              )}

            {/* Login button */}
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
                Login to System
              </Button>
            </Form.Item>

            {/* Tenant toggle */}
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
                {showTenantField ? "Hide Tenant Settings" : "Show Tenant Settings"}
              </Button>
            </Form.Item>
          </Form>
        </Card>
      </div>

      {/* CSS animations */}
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
