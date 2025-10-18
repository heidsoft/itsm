# 企业级 ITSM 登录组件

一个功能完整、安全可靠的企业级 ITSM 登录页面组件，基于 React + TypeScript + Tailwind CSS 构建。

## 功能特性

### 🔐 认证功能
- **多种登录方式**：用户名/邮箱 + 密码登录
- **记住我功能**：可选择保持登录状态
- **忘记密码**：密码重置链接
- **SSO 集成**：支持企业单点登录
- **多因素认证**：TOTP 和 WebAuthn 支持

### 🛡️ 安全特性
- **CSRF 保护**：自动获取和发送 CSRF token
- **安全密码处理**：不在本地持久化明文密码
- **错误信息保护**：不暴露敏感系统信息
- **防重入提交**：避免重复提交表单

### ♿ 可访问性
- **语义化 HTML**：正确的标签和结构
- **键盘导航**：完整的键盘操作支持
- **屏幕阅读器**：aria-live 和 aria-describedby 支持
- **焦点管理**：合理的焦点顺序和视觉反馈

### 🎨 用户体验
- **响应式设计**：适配各种屏幕尺寸
- **加载状态**：优雅的加载动画和禁用状态
- **实时验证**：字段级别的即时反馈
- **密码可见性**：可切换密码显示/隐藏
- **错误处理**：友好的错误提示信息

### 🌍 国际化
- **多语言支持**：内置中英文支持
- **可扩展**：易于添加新语言
- **动态切换**：运行时语言切换

## 安装和使用

### 基本使用

```tsx
import React from 'react';
import { EnterpriseLoginForm } from './components/EnterpriseLoginForm';

function App() {
  const handleLoginSuccess = (user: any) => {
    console.log('登录成功:', user);
    // 处理登录成功逻辑
  };

  const handleLoginError = (error: string) => {
    console.error('登录失败:', error);
    // 处理登录失败逻辑
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <EnterpriseLoginForm
        onLoginSuccess={handleLoginSuccess}
        onLoginError={handleLoginError}
        enableSSO={true}
        enableMFA={true}
        enableWebAuthn={true}
        language="zh"
      />
    </div>
  );
}

export default App;
```

### 高级配置

```tsx
import { EnterpriseLoginForm } from './components/EnterpriseLoginForm';

function LoginPage() {
  return (
    <EnterpriseLoginForm
      // 回调函数
      onLoginSuccess={(user) => {
        // 登录成功处理
        localStorage.setItem('user', JSON.stringify(user));
        window.location.href = '/dashboard';
      }}
      onLoginError={(error) => {
        // 错误处理
        console.error('Login failed:', error);
      }}
      
      // 功能开关
      enableSSO={true}
      enableMFA={true}
      enableWebAuthn={true}
      enableRememberMe={true}
      
      // 自定义配置
      language="zh"
      theme="light"
      companyName="Your Company"
      logoUrl="/logo.png"
      
      // 自定义样式
      className="custom-login-form"
      
      // SSO 配置
      ssoProviders={[
        { name: 'Google', url: '/auth/google' },
        { name: 'Microsoft', url: '/auth/microsoft' }
      ]}
    />
  );
}
```

## API 集成

组件需要以下 API 端点：

### 登录接口
```
POST /api/auth/login
Content-Type: application/json
X-CSRF-Token: <token>

{
  "username": "user@example.com",
  "password": "password123",
  "rememberMe": true,
  "mfaCode": "123456"
}
```

### CSRF Token 获取
```
GET /api/auth/csrf-token

Response:
{
  "token": "csrf-token-value"
}
```

### WebAuthn 认证
```
POST /api/auth/webauthn/challenge
POST /api/auth/webauthn/verify
```

详细的 API 实现请参考 `src/lib/api/auth-api.ts`。

## 测试

运行单元测试：

```bash
npm test -- EnterpriseLoginForm.test.tsx
```

测试覆盖：
- ✅ 基本渲染和表单提交
- ✅ 表单验证和错误处理
- ✅ 用户交互和状态管理
- ✅ MFA 和 WebAuthn 功能
- ✅ 可访问性和键盘导航
- ✅ 国际化和主题切换

## 自定义样式

组件使用 Tailwind CSS 构建，支持以下自定义：

### CSS 变量
```css
:root {
  --login-primary-color: #3b82f6;
  --login-secondary-color: #64748b;
  --login-error-color: #ef4444;
  --login-success-color: #10b981;
}
```

### 自定义类名
```tsx
<EnterpriseLoginForm
  className="my-custom-login"
  inputClassName="my-custom-input"
  buttonClassName="my-custom-button"
/>
```

## 浏览器支持

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## 依赖项

- React 18+
- TypeScript 4.5+
- Tailwind CSS 3.0+
- Lucide React (图标)

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 更新日志

### v1.0.0
- 初始版本发布
- 支持基本登录功能
- 集成 MFA 和 WebAuthn
- 完整的可访问性支持
- 国际化支持