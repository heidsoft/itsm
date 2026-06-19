# 统一表单组件使用指南

## 目的
统一使用Ant Design表单组件，确保表单交互一致性和可维护性。

## 核心原则

### 1. 永远使用Form组件包裹

#### ✅ 正确示例
```tsx
import { Form, Input, Button } from 'antd';

<Form form={form} layout="vertical" onFinish={handleSubmit}>
  <Form.Item 
    label="用户名" 
    name="username"
    rules={[{ required: true, message: '请输入用户名' }]}
  >
    <Input placeholder="请输入用户名" />
  </Form.Item>
  <Button type="primary" htmlType="submit">提交</Button>
</Form>
```

#### ❌ 错误示例
```tsx
<div className="space-y-4">
  <input 
    type="text" 
    placeholder="用户名"
    value={username}
    onChange={e => setUsername(e.target.value)}
  />
  <button onClick={handleSubmit}>提交</button>
</div>
```

### 2. 使用Form.Item管理状态

#### ✅ 正确示例
```tsx
<Form.Item
  label="邮箱"
  name="email"
  rules={[
    { required: true, message: '请输入邮箱' },
    { type: 'email', message: '邮箱格式不正确' }
  ]}
  hasFeedback
>
  <Input placeholder="example@email.com" />
</Form.Item>
```

#### ❌ 错误示例
```tsx
<div>
  <label>邮箱</label>
  <input 
    type="email" 
    value={email}
    onChange={e => setEmail(e.target.value)}
  />
  {emailError && <span className="error">{emailError}</span>}
</div>
```

## 组件对照表

| 原生HTML | Ant Design组件 | 用途 |
|---------|----------------|------|
| `<input type="text">` | `<Input>` | 文本输入 |
| `<input type="number">` | `<InputNumber>` | 数字输入 |
| `<input type="password">` | `<Input.Password>` | 密码输入 |
| `<textarea>` | `<Input.TextArea>` | 多行文本 |
| `<select>` | `<Select>` | 下拉选择 |
| `<input type="checkbox">` | `<Checkbox>` | 复选框 |
| `<input type="radio">` | `<Radio>` | 单选框 |
| `<input type="date">` | `<DatePicker>` | 日期选择 |
| `<input type="datetime-local">` | `<DatePicker showTime>` | 日期时间选择 |
| `<input type="file">` | `<Upload>` | 文件上传 |

## 表单布局

### 垂直布局 (推荐)
```tsx
<Form form={form} layout="vertical">
  <Form.Item label="用户名" name="username">
    <Input />
  </Form.Item>
  <Form.Item label="邮箱" name="email">
    <Input />
  </Form.Item>
</Form>
```

### 水平布局
```tsx
<Form form={form} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
  <Form.Item label="用户名" name="username">
    <Input />
  </Form.Item>
</Form>
```

### 行内布局
```tsx
<Form form={form} layout="inline">
  <Form.Item name="keyword">
    <Input.Search placeholder="搜索" />
  </Form.Item>
  <Form.Item>
    <Button type="primary">搜索</Button>
  </Form.Item>
</Form>
```

## 常用表单模式

### 1. 带验证的表单
```tsx
<Form form={form} onFinish={handleSubmit}>
  <Form.Item
    label="密码"
    name="password"
    rules={[
      { required: true, message: '请输入密码' },
      { min: 8, message: '密码至少8位' },
      { 
        pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/,
        message: '密码必须包含大小写字母和数字'
      }
    ]}
    hasFeedback
  >
    <Input.Password />
  </Form.Item>

  <Form.Item
    label="确认密码"
    name="confirmPassword"
    dependencies={['password']}
    rules={[
      { required: true, message: '请确认密码' },
      ({ getFieldValue }) => ({
        validator(_, value) {
          if (!value || getFieldValue('password') === value) {
            return Promise.resolve();
          }
          return Promise.reject(new Error('两次密码不一致'));
        },
      }),
    ]}
    hasFeedback
  >
    <Input.Password />
  </Form.Item>
</Form>
```

### 2. 动态表单
```tsx
<Form form={form}>
  <Form.List name="users">
    {(fields, { add, remove }) => (
      <>
        {fields.map(field => (
          <Space key={field.key} align="baseline">
            <Form.Item
              {...field}
              name={[field.name, 'name']}
              rules={[{ required: true, message: '请输入姓名' }]}
            >
              <Input placeholder="姓名" />
            </Form.Item>
            <Form.Item
              {...field}
              name={[field.name, 'email']}
              rules={[{ required: true, type: 'email' }]}
            >
              <Input placeholder="邮箱" />
            </Form.Item>
            <MinusCircleOutlined onClick={() => remove(field.name)} />
          </Space>
        ))}
        <Button onClick={() => add()} icon={<PlusOutlined />}>
          添加用户
        </Button>
      </>
    )}
  </Form.List>
</Form>
```

### 3. 联动表单
```tsx
<Form form={form}>
  <Form.Item label="国家" name="country">
    <Select onChange={() => form.setFieldValue('city', undefined)}>
      <Option value="china">中国</Option>
      <Option value="usa">美国</Option>
    </Select>
  </Form.Item>

  <Form.Item noStyle shouldUpdate>
    {({ getFieldValue }) => {
      const country = getFieldValue('country');
      return (
        <Form.Item label="城市" name="city">
          <Select>
            {country === 'china' ? (
              <>
                <Option value="beijing">北京</Option>
                <Option value="shanghai">上海</Option>
              </>
            ) : (
              <>
                <Option value="newyork">纽约</Option>
                <Option value="losangeles">洛杉矶</Option>
              </>
            )}
          </Select>
        </Form.Item>
      );
    }}
  </Form.Item>
</Form>
```

## 表单验证规则

### 常用验证规则
```tsx
const rules = {
  // 必填
  required: { required: true, message: '此字段必填' },

  // 邮箱
  email: { type: 'email', message: '邮箱格式不正确' },

  // 手机号
  phone: { 
    pattern: /^1[3-9]\d{9}$/, 
    message: '手机号格式不正确' 
  },

  // 身份证
  idCard: { 
    pattern: /\d{17}[\dXx]/, 
    message: '身份证号格式不正确' 
  },

  // 长度限制
  length: { min: 6, max: 20, message: '长度在6-20个字符之间' },

  // 只允许数字字母
  alphanumeric: { 
    pattern: /^[a-zA-Z0-9]+$/, 
    message: '只能包含字母和数字' 
  },
};

// 使用
<Form.Item rules={[rules.required, rules.email]}>
  <Input />
</Form.Item>
```

## 表单提交处理

### 异步提交
```tsx
const [form] = Form.useForm();
const [loading, setLoading] = useState(false);

const handleSubmit = async (values: any) => {
  try {
    setLoading(true);
    await api.submitForm(values);
    message.success('提交成功');
    form.resetFields();
  } catch (error) {
    message.error('提交失败');
  } finally {
    setLoading(false);
  }
};

<Form form={form} onFinish={handleSubmit}>
  {/* 表单字段 */}
  <Button type="primary" htmlType="submit" loading={loading}>
    提交
  </Button>
</Form>
```

## 最佳实践

### 1. 表单重置
```tsx
const handleReset = () => {
  form.resetFields(); // 重置所有字段
  // 或者重置特定字段
  form.resetFields(['username', 'email']);
};
```

### 2. 表单赋值
```tsx
const initData = {
  username: 'admin',
  email: 'admin@example.com',
};

useEffect(() => {
  form.setFieldsValue(initData);
}, [form, initData]);
```

### 3. 表单监听
```tsx
<Form 
  form={form} 
  onValuesChange={(changedValues, allValues) => {
    console.log('变化的值:', changedValues);
    console.log('所有值:', allValues);
  }}
>
  {/* 表单字段 */}
</Form>
```

### 4. 自定义验证
```tsx
<Form.Item
  name="age"
  rules={[
    {
      validator: (_, value) => {
        if (value >= 18 && value <= 60) {
          return Promise.resolve();
        }
        return Promise.reject(new Error('年龄必须在18-60岁之间'));
      },
    },
  ]}
>
  <InputNumber />
</Form.Item>
```

## 迁移清单

### 已迁移的表单
- [x] ApplyRdsForm - 数据库申请表单
- [x] ApplyVmForm - 虚拟机申请表单
- [x] ExpandOssForm - OSS扩容表单
- [x] GroupManagement - 用户组管理

### 待迁移的表单
- [ ] ServiceRequestPage - 服务请求页面
- [ ] 其他使用原生HTML表单的页面

## 检查工具

### 自动化检查
```bash
# 检查原生HTML表单元素
grep -rn '<input' src --include='*.tsx' | grep -v 'type="hidden"'
grep -rn '<select' src --include='*.tsx'
grep -rn '<textarea' src --include='*.tsx'
```

### ESLint规则
```json
{
  "rules": {
    "react/button-has-type": "error",
    "react/no-unescaped-entities": "error"
  }
}
```

---

**更新日期**: 2026-04-26  
**维护者**: ITSM前端团队
