# 认证组件库文档

## 📚 概述

认证组件库提供了一套完整的认证页面组件，确保登录、注册、忘记密码等页面使用相同的设计系统。所有组件都基于 Ant Design 和自定义设计系统，支持主题切换和国际化。

## 🏗️ 组件架构

```text
认证组件库
├── AuthLayout     # 认证页面布局组件
├── AuthCard       # 认证卡片组件
├── AuthForm       # 认证表单组件
├── AuthField      # 认证字段组件
└── AuthButton     # 认证按钮组件
```

## 🎯 核心组件

### 1. AuthLayout - 认证页面布局

提供统一的认证页面布局，包括品牌区域和表单区域。

#### 基本用法

```tsx
import { AuthLayout } from "@/components/auth";

<AuthLayout>
  <div>表单内容</div>
</AuthLayout>
```

#### AuthLayout 属性配置

| 属性 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| children | ReactNode | - | 页面内容 |
| title | string | "AI-Native ITSM" | 品牌标题 |
| subtitle | string | "AI驱动的IT服务管理系统" | 品牌副标题 |
| showBranding | boolean | true | 是否显示品牌区域 |

#### 使用示例

```tsx
// 完整品牌区域
<AuthLayout title="AI-Native ITSM" subtitle="AI驱动的IT服务管理系统">
  <div>表单内容</div>
</AuthLayout>

// 隐藏品牌区域（移动端优化）
<AuthLayout showBranding={false}>
  <div>表单内容</div>
</AuthLayout>
```

### 2. AuthCard - 认证卡片

提供统一的认证表单卡片样式。

#### 基本用法

```tsx
import { AuthCard } from "@/components/auth";

<AuthCard title="欢迎回来" subtitle="请登录您的账户">
  <div>表单内容</div>
</AuthCard>
```

#### AuthCard 属性配置

| 属性 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| children | ReactNode | - | 卡片内容 |
| title | string | - | 卡片标题 |
| subtitle | string | - | 卡片副标题 |
| variant | "default" \| "elevated" \| "outlined" \| "filled" | "default" | 卡片变体 |
| size | "sm" \| "md" \| "lg" | "md" | 卡片尺寸 |
| bordered | boolean | true | 是否显示边框 |
| hoverable | boolean | false | 是否可悬停 |
| loading | boolean | false | 是否显示加载状态 |

#### 使用示例

```tsx
// 基础卡片
<AuthCard title="登录" subtitle="请输入您的账户信息">
  <form>...</form>
</AuthCard>

// 高级卡片
<AuthCard
  title="注册"
  subtitle="创建新账户"
  variant="elevated"
  size="lg"
  hoverable
  extra={<Button>帮助</Button>}
  footer={<div>底部内容</div>}
>
  <form>...</form>
</AuthCard>
```

### 3. AuthForm - 认证表单

提供统一的表单样式、验证和交互体验。

#### 基本用法

```tsx
import { AuthForm } from "@/components/auth";

const fields = [
  {
    name: "username",
    label: "用户名",
    type: "text",
    placeholder: "请输入用户名",
    required: true,
    prefix: <User />,
  },
  {
    name: "password",
    label: "密码",
    type: "password",
    placeholder: "请输入密码",
    required: true,
    prefix: <Lock />,
  },
];

const primaryButton = {
  text: "登录",
  type: "primary",
  size: "lg",
  fullWidth: true,
  icon: <ArrowRight />,
};

<AuthForm
  title="欢迎回来"
  subtitle="请登录您的账户以继续使用服务"
  fields={fields}
  primaryButton={primaryButton}
  onSubmit={handleSubmit}
/>
```

#### AuthForm 属性配置

| 属性 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| title | string | - | 表单标题 |
| subtitle | string | - | 表单副标题 |
| fields | AuthFieldConfig[] | - | 表单字段配置 |
| primaryButton | AuthButtonConfig | - | 主要按钮配置 |
| secondaryButton | AuthButtonConfig | - | 次要按钮配置 |
| onSubmit | (values) => void | - | 表单提交回调 |
| onValidationFailed | (errors) => void | - | 验证失败回调 |
| initialValues | object | - | 初始值 |
| error | string | - | 错误信息 |
| success | string | - | 成功信息 |

#### 字段配置 (AuthFieldConfig)

| 属性 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| name | string | - | 字段名称 |
| label | string | - | 字段标签 |
| type | "text" \| "email" \| "password" \| "tel" \| "url" | "text" | 字段类型 |
| placeholder | string | - | 占位符文本 |
| required | boolean | false | 是否必填 |
| rules | any[] | - | 验证规则 |
| prefix | ReactNode | - | 前缀图标 |
| suffix | ReactNode | - | 后缀图标 |
| helpText | string | - | 帮助文本 |
| showPasswordStrength | boolean | false | 是否显示密码强度 |

#### 按钮配置 (AuthButtonConfig)

| 属性 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| text | string | - | 按钮文本 |
| type | "primary" \| "secondary" \| "outline" \| "ghost" | "primary" | 按钮类型 |
| size | "sm" \| "md" \| "lg" | "lg" | 按钮尺寸 |
| fullWidth | boolean | true | 是否全宽 |
| loading | boolean | false | 是否显示加载状态 |
| disabled | boolean | false | 是否禁用 |
| icon | ReactNode | - | 图标 |
| iconPosition | "left" \| "right" | "left" | 图标位置 |
| onClick | () => void | - | 点击回调 |

### 4. AuthField - 认证字段

提供统一的输入框样式和交互体验。

#### 基本用法

```tsx
import { AuthField } from "@/components/auth";

<AuthField
  label="用户名"
  type="text"
  placeholder="请输入用户名"
  prefix={<User />}
  required
  onChange={(value) => console.log(value)}
/>
```

#### AuthField 属性配置

| 属性 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| label | string | - | 字段标签 |
| type | "text" \| "email" \| "password" \| "tel" \| "url" \| "number" | "text" | 字段类型 |
| placeholder | string | - | 占位符文本 |
| value | string | - | 字段值 |
| required | boolean | false | 是否必填 |
| disabled | boolean | false | 是否禁用 |
| prefix | ReactNode | - | 前缀图标 |
| suffix | ReactNode | - | 后缀图标 |
| helpText | string | - | 帮助文本 |
| error | string | - | 错误信息 |
| success | string | - | 成功信息 |
| size | "sm" \| "md" \| "lg" | "lg" | 字段尺寸 |
| showPasswordStrength | boolean | false | 是否显示密码强度 |
| clearable | boolean | false | 是否可清除 |

### 5. AuthButton - 认证按钮

提供统一的按钮样式和交互体验。

#### 基本用法

```tsx
import { AuthButton } from "@/components/auth";

<AuthButton
  variant="primary"
  size="lg"
  fullWidth
  icon={<ArrowRight />}
  onClick={handleClick}
>
  登录
</AuthButton>
```

#### 属性配置

| 属性 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| children | ReactNode | - | 按钮内容 |
| variant | "primary" \| "secondary" \| "outline" \| "ghost" \| "danger" \| "success" | "primary" | 按钮类型 |
| size | "sm" \| "md" \| "lg" | "lg" | 按钮尺寸 |
| fullWidth | boolean | false | 是否全宽 |
| loading | boolean | false | 是否显示加载状态 |
| disabled | boolean | false | 是否禁用 |
| icon | ReactNode | - | 图标 |
| iconPosition | "left" \| "right" | "left" | 图标位置 |
| ripple | boolean | true | 是否显示波纹效果 |
| shape | "default" \| "round" \| "circle" | "default" | 按钮形状 |

## 🎨 使用示例

### 登录页面

```tsx
import { AuthLayout, AuthForm } from "@/components/auth";
import { User, Lock, ArrowRight, Shield } from "lucide-react";

export default function LoginPage() {
  const fields = [
    {
      name: "username",
      label: "用户名",
      type: "text",
      placeholder: "请输入用户名",
      required: true,
      prefix: <User size={16} />,
    },
    {
      name: "password",
      label: "密码",
      type: "password",
      placeholder: "请输入密码",
      required: true,
      prefix: <Lock size={16} />,
    },
  ];

  const primaryButton = {
    text: "登录",
    type: "primary",
    size: "lg",
    fullWidth: true,
    icon: <ArrowRight size={16} />,
    iconPosition: "right",
  };

  const secondaryButton = {
    text: "SSO 单点登录",
    type: "outline",
    size: "lg",
    fullWidth: true,
    icon: <Shield size={16} />,
  };

  const handleSubmit = async (values: Record<string, any>) => {
    console.log("登录数据:", values);
    // 处理登录逻辑
  };

  return (
    <AuthLayout>
      <AuthForm
        title="欢迎回来"
        subtitle="请登录您的账户以继续使用服务"
        fields={fields}
        primaryButton={primaryButton}
        secondaryButton={secondaryButton}
        showDivider={true}
        dividerText="或"
        onSubmit={handleSubmit}
        extraActions={
          <div className="text-center mt-4">
            <span className="text-sm text-gray-500">
              还没有账户？{" "}
              <a href="/register" className="text-blue-600 hover:underline">
                立即注册
              </a>
            </span>
          </div>
        }
      />
    </AuthLayout>
  );
}
```

### 注册页面

```tsx
import { AuthLayout, AuthForm } from "@/components/auth";
import { User, Mail, Lock, ArrowRight } from "lucide-react";

export default function RegisterPage() {
  const fields = [
    {
      name: "username",
      label: "用户名",
      type: "text",
      placeholder: "请输入用户名",
      required: true,
      prefix: <User size={16} />,
      rules: [
        { required: true, message: "请输入用户名" },
        { min: 3, message: "用户名至少3个字符" },
      ],
    },
    {
      name: "email",
      label: "邮箱",
      type: "email",
      placeholder: "请输入邮箱地址",
      required: true,
      prefix: <Mail size={16} />,
      rules: [
        { required: true, message: "请输入邮箱" },
        { type: "email", message: "请输入有效的邮箱地址" },
      ],
    },
    {
      name: "password",
      label: "密码",
      type: "password",
      placeholder: "请输入密码",
      required: true,
      prefix: <Lock size={16} />,
      showPasswordStrength: true,
      rules: [
        { required: true, message: "请输入密码" },
        { min: 8, message: "密码至少8个字符" },
      ],
    },
    {
      name: "confirmPassword",
      label: "确认密码",
      type: "password",
      placeholder: "请再次输入密码",
      required: true,
      prefix: <Lock size={16} />,
      rules: [
        { required: true, message: "请确认密码" },
        ({ getFieldValue }) => ({
          validator(_, value) {
            if (!value || getFieldValue("password") === value) {
              return Promise.resolve();
            }
            return Promise.reject(new Error("两次输入的密码不一致"));
          },
        }),
      ],
    },
  ];

  const primaryButton = {
    text: "注册",
    type: "primary",
    size: "lg",
    fullWidth: true,
    icon: <ArrowRight size={16} />,
    iconPosition: "right",
  };

  const handleSubmit = async (values: Record<string, any>) => {
    console.log("注册数据:", values);
    // 处理注册逻辑
  };

  return (
    <AuthLayout>
      <AuthForm
        title="创建账户"
        subtitle="请填写信息以创建新账户"
        fields={fields}
        primaryButton={primaryButton}
        onSubmit={handleSubmit}
        extraActions={
          <div className="text-center mt-4">
            <span className="text-sm text-gray-500">
              已有账户？{" "}
              <a href="/login" className="text-blue-600 hover:underline">
                立即登录
              </a>
            </span>
          </div>
        }
      />
    </AuthLayout>
  );
}
```

### 忘记密码页面

```tsx
import { AuthLayout, AuthForm } from "@/components/auth";
import { Mail, ArrowRight } from "lucide-react";

export default function ForgotPasswordPage() {
  const fields = [
    {
      name: "email",
      label: "邮箱地址",
      type: "email",
      placeholder: "请输入您的邮箱地址",
      required: true,
      prefix: <Mail size={16} />,
      helpText: "我们将向此邮箱发送重置密码的链接",
      rules: [
        { required: true, message: "请输入邮箱地址" },
        { type: "email", message: "请输入有效的邮箱地址" },
      ],
    },
  ];

  const primaryButton = {
    text: "发送重置链接",
    type: "primary",
    size: "lg",
    fullWidth: true,
    icon: <ArrowRight size={16} />,
    iconPosition: "right",
  };

  const handleSubmit = async (values: Record<string, any>) => {
    console.log("重置密码:", values);
    // 处理重置密码逻辑
  };

  return (
    <AuthLayout>
      <AuthForm
        title="重置密码"
        subtitle="请输入您的邮箱地址，我们将发送重置密码的链接"
        fields={fields}
        primaryButton={primaryButton}
        onSubmit={handleSubmit}
        extraActions={
          <div className="text-center mt-4">
            <span className="text-sm text-gray-500">
              记起密码了？{" "}
              <a href="/login" className="text-blue-600 hover:underline">
                返回登录
              </a>
            </span>
          </div>
        }
      />
    </AuthLayout>
  );
}
```

## 🎨 主题定制

所有组件都支持主题定制，通过 Ant Design 的 theme 系统：

```tsx
import { ConfigProvider } from "antd";

const customTheme = {
  token: {
    colorPrimary: "#1890ff",
    borderRadius: 8,
    // 其他主题配置...
  },
};

<ConfigProvider theme={customTheme}>
  <AuthLayout>
    <AuthForm {...props} />
  </AuthLayout>
</ConfigProvider>
```

## 🌍 国际化支持

组件支持国际化，可以通过配置实现多语言：

```tsx
// 中文配置
const zhCN = {
  login: {
    title: "欢迎回来",
    subtitle: "请登录您的账户以继续使用服务",
    username: "用户名",
    password: "密码",
    submit: "登录",
  },
};

// 英文配置
const enUS = {
  login: {
    title: "Welcome Back",
    subtitle: "Please sign in to your account to continue",
    username: "Username",
    password: "Password",
    submit: "Sign In",
  },
};
```

## 📱 响应式设计

所有组件都支持响应式设计：

- **桌面端**: 显示完整的品牌区域和表单区域
- **平板端**: 自适应布局，保持良好体验
- **移动端**: 隐藏品牌区域，显示移动端Logo，优化触摸交互

## 🔧 最佳实践

### 1. 表单验证

```tsx
const validationRules = {
  username: [
    { required: true, message: "请输入用户名" },
    { min: 3, max: 20, message: "用户名长度在3-20个字符" },
    { pattern: /^[a-zA-Z0-9_]+$/, message: "用户名只能包含字母、数字和下划线" },
  ],
  email: [
    { required: true, message: "请输入邮箱" },
    { type: "email", message: "请输入有效的邮箱地址" },
  ],
  password: [
    { required: true, message: "请输入密码" },
    { min: 8, message: "密码至少8个字符" },
    { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/, message: "密码必须包含大小写字母和数字" },
  ],
};
```

### 2. 错误处理

```tsx
const [error, setError] = useState("");
const [success, setSuccess] = useState("");

const handleSubmit = async (values: Record<string, any>) => {
  try {
    setError("");
    setSuccess("");
    
    const response = await authService.login(values);
    
    if (response.success) {
      setSuccess("登录成功！");
      // 跳转到仪表板
      router.push("/dashboard");
    }
  } catch (err) {
    setError(err.message || "登录失败，请重试");
  }
};
```

### 3. 加载状态

```tsx
const [loading, setLoading] = useState(false);

const primaryButton = {
  text: "登录",
  type: "primary",
  size: "lg",
  fullWidth: true,
  loading: loading, // 显示加载状态
  disabled: loading, // 禁用按钮
};
```

## 🚀 总结

认证组件库提供了完整的认证页面解决方案：

- ✅ **统一设计**: 所有组件使用相同的设计系统
- ✅ **类型安全**: 完整的 TypeScript 支持
- ✅ **响应式**: 支持各种屏幕尺寸
- ✅ **可定制**: 支持主题和样式定制
- ✅ **国际化**: 支持多语言
- ✅ **易用性**: 简单的API和丰富的配置选项
- ✅ **可维护**: 清晰的组件结构和文档

使用这套组件库，可以快速构建一致、美观、功能完整的认证页面。
