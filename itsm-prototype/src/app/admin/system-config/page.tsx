"use client";

import { Settings, AlertTriangle, RefreshCw, Save, Server } from 'lucide-react';

import React, { useState } from "react";
// 系统配置分组
const CONFIG_SECTIONS = {
  GENERAL: "general",
  SECURITY: "security",
  EMAIL: "email",
  DATABASE: "database",
  INTEGRATION: "integration",
  NOTIFICATION: "notification",
} as const;

// 模拟系统配置数据
const mockSystemConfig = {
  // 基础设置
  general: {
    systemName: "企业ITSM系统",
    systemUrl: "https://itsm.company.com",
    timezone: "Asia/Shanghai",
    language: "zh-CN",
    dateFormat: "YYYY-MM-DD",
    timeFormat: "24h",
    sessionTimeout: 30,
    maxFileSize: 10,
    allowedFileTypes: ".pdf,.doc,.docx,.xls,.xlsx,.jpg,.png,.gif",
  },
  // 安全设置
  security: {
    passwordMinLength: 8,
    passwordRequireUppercase: true,
    passwordRequireLowercase: true,
    passwordRequireNumbers: true,
    passwordRequireSpecialChars: true,
    passwordExpireDays: 90,
    maxLoginAttempts: 5,
    lockoutDuration: 15,
    enableTwoFactor: false,
    enableSSOLogin: false,
    ssoProvider: "LDAP",
  },
  // 邮件设置
  email: {
    smtpHost: "smtp.company.com",
    smtpPort: 587,
    smtpUsername: "itsm@company.com",
    smtpPassword: "********",
    smtpEncryption: "TLS",
    fromEmail: "itsm@company.com",
    fromName: "ITSM系统",
    enableEmailNotification: true,
    emailTemplate: "default",
  },
  // 数据库设置
  database: {
    host: "localhost",
    port: 5432,
    database: "itsm_db",
    username: "itsm_user",
    maxConnections: 100,
    connectionTimeout: 30,
    enableBackup: true,
    backupSchedule: "daily",
    backupRetentionDays: 30,
  },
  // 集成设置
  integration: {
    enableApiAccess: true,
    apiRateLimit: 1000,
    enableWebhooks: true,
    webhookTimeout: 30,
    enableLdapSync: false,
    ldapServer: "ldap.company.com",
    ldapPort: 389,
    ldapBaseDn: "dc=company,dc=com",
  },
  // 通知设置
  notification: {
    enablePushNotification: true,
    enableEmailNotification: true,
    enableSmsNotification: false,
    smsProvider: "阿里云",
    smsApiKey: "",
    notificationRetryTimes: 3,
    notificationRetryInterval: 5,
  },
};

const SystemConfiguration = () => {
  const [config, setConfig] = useState(mockSystemConfig);
  const [activeSection, setActiveSection] = useState(CONFIG_SECTIONS.GENERAL);
  const [hasChanges, setHasChanges] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  // 处理配置更新
  const handleConfigChange = (section: string, field: string, value: unknown) => {
    setConfig((prev) => ({
      ...prev,
      [section]: {
        ...prev[section],
        [field]: value,
      },
    }));
    setHasChanges(true);
  };

  // 保存配置
  const handleSave = async () => {
    setIsSaving(true);
    try {
      // 模拟API调用
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setHasChanges(false);
      alert("配置保存成功！");
    } catch (error) {
      alert("保存失败，请重试！");
    } finally {
      setIsSaving(false);
    }
  };

  // 重置配置
  const handleReset = () => {
    if (confirm("确定要重置所有配置吗？此操作不可撤销。")) {
      setConfig(mockSystemConfig);
      setHasChanges(false);
    }
  };

  // 测试连接
  const handleTestConnection = async (type: string) => {
    try {
      // 模拟测试连接
      await new Promise((resolve) => setTimeout(resolve, 1000));
      alert(`${type}连接测试成功！`);
    } catch (error) {
      alert(`${type}连接测试失败！`);
    }
  };

  // 配置节点菜单
  const sectionMenus = [
    {
      key: CONFIG_SECTIONS.GENERAL,
      label: "基础设置",
      icon: Settings,
      description: "系统基本信息和通用配置",
    },
    {
      key: CONFIG_SECTIONS.SECURITY,
      label: "安全设置",
      icon: Shield,
      description: "密码策略和安全选项",
    },
    {
      key: CONFIG_SECTIONS.EMAIL,
      label: "邮件设置",
      icon: Mail,
      description: "SMTP服务器和邮件模板",
    },
    {
      key: CONFIG_SECTIONS.DATABASE,
      label: "数据库设置",
      icon: Database,
      description: "数据库连接和备份配置",
    },
    {
      key: CONFIG_SECTIONS.INTEGRATION,
      label: "集成设置",
      icon: Globe,
      description: "API接口和第三方集成",
    },
    {
      key: CONFIG_SECTIONS.NOTIFICATION,
      label: "通知设置",
      icon: Bell,
      description: "消息推送和通知渠道",
    },
  ];

  // 渲染基础设置
  const renderGeneralSettings = () => (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            系统名称
          </label>
          <input
            type="text"
            value={config.general.systemName}
            onChange={(e) =>
              handleConfigChange("general", "systemName", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            系统URL
          </label>
          <input
            type="url"
            value={config.general.systemUrl}
            onChange={(e) =>
              handleConfigChange("general", "systemUrl", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            时区
          </label>
          <select
            value={config.general.timezone}
            onChange={(e) =>
              handleConfigChange("general", "timezone", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="Asia/Shanghai">Asia/Shanghai (UTC+8)</option>
            <option value="UTC">UTC (UTC+0)</option>
            <option value="America/New_York">America/New_York (UTC-5)</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            默认语言
          </label>
          <select
            value={config.general.language}
            onChange={(e) =>
              handleConfigChange("general", "language", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="zh-CN">简体中文</option>
            <option value="en-US">English</option>
            <option value="ja-JP">日本語</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            会话超时时间（分钟）
          </label>
          <input
            type="number"
            value={config.general.sessionTimeout}
            onChange={(e) =>
              handleConfigChange(
                "general",
                "sessionTimeout",
                parseInt(e.target.value)
              )
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            最大文件大小（MB）
          </label>
          <input
            type="number"
            value={config.general.maxFileSize}
            onChange={(e) =>
              handleConfigChange(
                "general",
                "maxFileSize",
                parseInt(e.target.value)
              )
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          允许的文件类型
        </label>
        <input
          type="text"
          value={config.general.allowedFileTypes}
          onChange={(e) =>
            handleConfigChange("general", "allowedFileTypes", e.target.value)
          }
          placeholder="例如: .pdf,.doc,.jpg"
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
        <p className="text-sm text-gray-500 mt-1">用逗号分隔多个文件扩展名</p>
      </div>
    </div>
  );

  // 渲染安全设置
  const renderSecuritySettings = () => (
    <div className="space-y-6">
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
        <div className="flex items-center">
          <AlertTriangle className="w-5 h-5 text-yellow-600 mr-2" />
          <h4 className="text-sm font-medium text-yellow-800">安全设置提醒</h4>
        </div>
        <p className="text-sm text-yellow-700 mt-1">
          修改安全设置可能影响所有用户的登录体验，请谨慎操作。
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            密码最小长度
          </label>
          <input
            type="number"
            min="6"
            max="20"
            value={config.security.passwordMinLength}
            onChange={(e) =>
              handleConfigChange(
                "security",
                "passwordMinLength",
                parseInt(e.target.value)
              )
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            密码过期天数
          </label>
          <input
            type="number"
            value={config.security.passwordExpireDays}
            onChange={(e) =>
              handleConfigChange(
                "security",
                "passwordExpireDays",
                parseInt(e.target.value)
              )
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            最大登录尝试次数
          </label>
          <input
            type="number"
            value={config.security.maxLoginAttempts}
            onChange={(e) =>
              handleConfigChange(
                "security",
                "maxLoginAttempts",
                parseInt(e.target.value)
              )
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            锁定时长（分钟）
          </label>
          <input
            type="number"
            value={config.security.lockoutDuration}
            onChange={(e) =>
              handleConfigChange(
                "security",
                "lockoutDuration",
                parseInt(e.target.value)
              )
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
      </div>

      <div className="space-y-4">
        <h4 className="text-lg font-medium text-gray-900">密码复杂度要求</h4>
        <div className="space-y-3">
          {[
            { key: "passwordRequireUppercase", label: "包含大写字母" },
            { key: "passwordRequireLowercase", label: "包含小写字母" },
            { key: "passwordRequireNumbers", label: "包含数字" },
            { key: "passwordRequireSpecialChars", label: "包含特殊字符" },
          ].map((item) => (
            <label key={item.key} className="flex items-center">
              <input
                type="checkbox"
                checked={config.security[item.key]}
                onChange={(e) =>
                  handleConfigChange("security", item.key, e.target.checked)
                }
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm text-gray-700">{item.label}</span>
            </label>
          ))}
        </div>
      </div>

      <div className="space-y-4">
        <h4 className="text-lg font-medium text-gray-900">高级安全选项</h4>
        <div className="space-y-3">
          {[
            { key: "enableTwoFactor", label: "启用双因子认证" },
            { key: "enableSSOLogin", label: "启用单点登录(SSO)" },
          ].map((item) => (
            <label key={item.key} className="flex items-center">
              <input
                type="checkbox"
                checked={config.security[item.key]}
                onChange={(e) =>
                  handleConfigChange("security", item.key, e.target.checked)
                }
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm text-gray-700">{item.label}</span>
            </label>
          ))}
        </div>
      </div>
    </div>
  );

  // 渲染邮件设置
  const renderEmailSettings = () => (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            SMTP服务器
          </label>
          <input
            type="text"
            value={config.email.smtpHost}
            onChange={(e) =>
              handleConfigChange("email", "smtpHost", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            SMTP端口
          </label>
          <input
            type="number"
            value={config.email.smtpPort}
            onChange={(e) =>
              handleConfigChange("email", "smtpPort", parseInt(e.target.value))
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            用户名
          </label>
          <input
            type="text"
            value={config.email.smtpUsername}
            onChange={(e) =>
              handleConfigChange("email", "smtpUsername", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            密码
          </label>
          <input
            type="password"
            value={config.email.smtpPassword}
            onChange={(e) =>
              handleConfigChange("email", "smtpPassword", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            加密方式
          </label>
          <select
            value={config.email.smtpEncryption}
            onChange={(e) =>
              handleConfigChange("email", "smtpEncryption", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="TLS">TLS</option>
            <option value="SSL">SSL</option>
            <option value="NONE">无加密</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            发件人邮箱
          </label>
          <input
            type="email"
            value={config.email.fromEmail}
            onChange={(e) =>
              handleConfigChange("email", "fromEmail", e.target.value)
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
      </div>

      <div className="flex gap-3">
        <button
          onClick={() => handleTestConnection("邮件服务器")}
          className="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 flex items-center gap-2"
        >
          <Server className="w-4 h-4" />
          测试连接
        </button>
      </div>
    </div>
  );

  // 根据当前选中的节点渲染对应的设置内容
  const renderCurrentSection = () => {
    switch (activeSection) {
      case CONFIG_SECTIONS.GENERAL:
        return renderGeneralSettings();
      case CONFIG_SECTIONS.SECURITY:
        return renderSecuritySettings();
      case CONFIG_SECTIONS.EMAIL:
        return renderEmailSettings();
      default:
        return (
          <div className="text-center py-12">
            <Settings className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-medium text-gray-900">
              功能开发中
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              该配置模块正在开发中，敬请期待！
            </p>
          </div>
        );
    }
  };

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">系统配置</h1>
          <p className="text-gray-600 mt-1">
            管理系统的基础设置、安全策略和集成配置
          </p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={handleReset}
            className="bg-gray-600 text-white px-4 py-2 rounded-lg hover:bg-gray-700 flex items-center gap-2"
          >
            <RefreshCw className="w-4 h-4" />
            重置
          </button>
          <button
            onClick={handleSave}
            disabled={!hasChanges || isSaving}
            className={`px-4 py-2 rounded-lg flex items-center gap-2 ${
              hasChanges && !isSaving
                ? "bg-blue-600 text-white hover:bg-blue-700"
                : "bg-gray-300 text-gray-500 cursor-not-allowed"
            }`}
          >
            <Save className="w-4 h-4" />
            {isSaving ? "保存中..." : "保存配置"}
          </button>
        </div>
      </div>

      {/* 配置内容 */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* 左侧菜单 */}
        <div className="lg:col-span-1">
          <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
            <div className="p-4 border-b border-gray-200">
              <h3 className="text-lg font-medium text-gray-900">配置分类</h3>
            </div>
            <nav className="space-y-1 p-2">
              {sectionMenus.map((menu) => {
                const Icon = menu.icon;
                const isActive = activeSection === menu.key;
                return (
                  <button
                    key={menu.key}
                    onClick={() => setActiveSection(menu.key)}
                    className={`w-full text-left px-3 py-2 rounded-lg flex items-center gap-3 transition-colors ${
                      isActive
                        ? "bg-blue-50 text-blue-700 border border-blue-200"
                        : "text-gray-700 hover:bg-gray-50"
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    <div>
                      <div className="text-sm font-medium">{menu.label}</div>
                      <div className="text-xs text-gray-500">
                        {menu.description}
                      </div>
                    </div>
                  </button>
                );
              })}
            </nav>
          </div>
        </div>

        {/* 右侧配置内容 */}
        <div className="lg:col-span-3">
          <div className="bg-white rounded-lg shadow-sm border">
            <div className="p-6 border-b border-gray-200">
              <h3 className="text-lg font-medium text-gray-900">
                {sectionMenus.find((m) => m.key === activeSection)?.label}
              </h3>
              <p className="text-sm text-gray-600 mt-1">
                {sectionMenus.find((m) => m.key === activeSection)?.description}
              </p>
            </div>
            <div className="p-6">{renderCurrentSection()}</div>
          </div>
        </div>
      </div>

      {/* 变更提示 */}
      {hasChanges && (
        <div className="fixed bottom-4 right-4 bg-blue-600 text-white px-4 py-2 rounded-lg shadow-lg flex items-center gap-2">
          <AlertTriangle className="w-4 h-4" />
          <span className="text-sm">您有未保存的更改</span>
        </div>
      )}
    </div>
  );
};

export default SystemConfiguration;
