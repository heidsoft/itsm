# 技术栈学习指南

## 📚 学习路径

### 第一阶段：前端基础 (2-3周)

#### 1. JavaScript基础

```javascript
// 现代JavaScript特性
// 1. 变量声明
const name = "张三";        // 常量，不可重新赋值
let age = 25;             // 变量，可以重新赋值
var oldWay = "不推荐";     // 旧方式，不推荐使用

// 2. 箭头函数
const add = (a, b) => a + b;
const greet = name => `Hello, ${name}!`;

// 3. 解构赋值
const user = { name: "张三", age: 25 };
const { name, age } = user;

// 4. 模板字符串
const message = `用户 ${name} 的年龄是 ${age} 岁`;

// 5. 数组方法
const numbers = [1, 2, 3, 4, 5];
const doubled = numbers.map(n => n * 2);
const evens = numbers.filter(n => n % 2 === 0);
const sum = numbers.reduce((acc, n) => acc + n, 0);
```

#### 2. React基础

```jsx
// React组件基础
import React, { useState, useEffect } from 'react';

// 函数组件
function Counter() {
  // useState Hook：管理状态
  const [count, setCount] = useState(0);
  
  // useEffect Hook：处理副作用
  useEffect(() => {
    document.title = `计数: ${count}`;
  }, [count]); // 依赖数组，只有count变化时才执行
  
  return (
    <div>
      <p>当前计数: {count}</p>
      <button onClick={() => setCount(count + 1)}>
        增加
      </button>
    </div>
  );
}

// 组件通信
function Parent() {
  const [data, setData] = useState("Hello");
  
  return (
    <div>
      <Child 
        data={data} 
        onUpdate={(newData) => setData(newData)} 
      />
    </div>
  );
}

function Child({ data, onUpdate }) {
  return (
    <div>
      <p>从父组件接收: {data}</p>
      <button onClick={() => onUpdate("Updated!")}>
        更新父组件
      </button>
    </div>
  );
}
```

#### 3. TypeScript基础

```typescript
// 类型定义
interface User {
  id: number;
  name: string;
  email?: string; // 可选属性
  age: number;
}

// 函数类型
type GreetFunction = (name: string) => string;

// 泛型
function identity<T>(arg: T): T {
  return arg;
}

// React组件类型
interface UserCardProps {
  user: User;
  onEdit: (user: User) => void;
  onDelete: (id: number) => void;
}

const UserCard: React.FC<UserCardProps> = ({ user, onEdit, onDelete }) => {
  return (
    <div>
      <h3>{user.name}</h3>
      <p>年龄: {user.age}</p>
      {user.email && <p>邮箱: {user.email}</p>}
      <button onClick={() => onEdit(user)}>编辑</button>
      <button onClick={() => onDelete(user.id)}>删除</button>
    </div>
  );
};
```

### 第二阶段：前端框架 (2-3周)

#### 1. Next.js基础

```typescript
// 文件系统路由
// app/page.tsx → /
// app/dashboard/page.tsx → /dashboard
// app/users/[id]/page.tsx → /users/123

// 页面组件
export default function DashboardPage() {
  return (
    <div>
      <h1>仪表盘</h1>
      <p>这是仪表盘页面</p>
    </div>
  );
}

// API路由
// app/api/users/route.ts
import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest) {
  // 处理GET请求
  const users = await fetchUsers();
  return NextResponse.json(users);
}

export async function POST(request: NextRequest) {
  // 处理POST请求
  const body = await request.json();
  const newUser = await createUser(body);
  return NextResponse.json(newUser, { status: 201 });
}
```

#### 2. Tailwind CSS

```css
/* 实用优先的CSS框架 */
/* 不需要写CSS，直接使用预定义的类名 */

/* 布局 */
<div class="flex items-center justify-between p-4">
  <h1 class="text-2xl font-bold">标题</h1>
  <button class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">
    按钮
  </button>
</div>

/* 响应式设计 */
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
  <div class="bg-white p-4 rounded shadow">卡片1</div>
  <div class="bg-white p-4 rounded shadow">卡片2</div>
  <div class="bg-white p-4 rounded shadow">卡片3</div>
</div>

/* 状态样式 */
<button class="bg-blue-500 hover:bg-blue-600 focus:bg-blue-700 disabled:bg-gray-300">
  按钮
</button>
```

#### 3. Ant Design

```jsx
// 企业级UI组件库
import { Button, Card, Table, Form, Input, Select } from 'antd';

// 表单组件
function UserForm() {
  const [form] = Form.useForm();
  
  const onFinish = (values) => {
    console.log('表单数据:', values);
  };
  
  return (
    <Form form={form} onFinish={onFinish}>
      <Form.Item name="name" label="姓名" rules={[{ required: true }]}>
        <Input placeholder="请输入姓名" />
      </Form.Item>
      
      <Form.Item name="role" label="角色">
        <Select>
          <Select.Option value="admin">管理员</Select.Option>
          <Select.Option value="user">普通用户</Select.Option>
        </Select>
      </Form.Item>
      
      <Form.Item>
        <Button type="primary" htmlType="submit">
          提交
        </Button>
      </Form.Item>
    </Form>
  );
}

// 表格组件
function UserTable() {
  const columns = [
    { title: '姓名', dataIndex: 'name', key: 'name' },
    { title: '邮箱', dataIndex: 'email', key: 'email' },
    { title: '角色', dataIndex: 'role', key: 'role' },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Button type="link" onClick={() => handleEdit(record)}>
          编辑
        </Button>
      ),
    },
  ];
  
  return <Table columns={columns} dataSource={users} />;
}
```

### 第三阶段：后端基础 (3-4周)

#### 1. Go语言基础

```go
// Go语言基础语法
package main

import (
    "fmt"
    "time"
)

// 结构体定义
type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// 方法定义
func (u *User) GetFullName() string {
    return fmt.Sprintf("用户: %s", u.Name)
}

// 接口定义
type UserService interface {
    GetUser(id int) (*User, error)
    CreateUser(user *User) error
    UpdateUser(user *User) error
    DeleteUser(id int) error
}

// 实现接口
type userService struct {
    db Database
}

func (s *userService) GetUser(id int) (*User, error) {
    // 实现获取用户的逻辑
    return &User{ID: id, Name: "张三"}, nil
}

// 错误处理
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, fmt.Errorf("除数不能为零")
    }
    return a / b, nil
}

// 并发处理
func fetchUsers() {
    ch := make(chan *User, 10)
    
    // 启动goroutine
    go func() {
        for i := 0; i < 10; i++ {
            user := &User{ID: i, Name: fmt.Sprintf("用户%d", i)}
            ch <- user
        }
        close(ch)
    }()
    
    // 接收数据
    for user := range ch {
        fmt.Printf("收到用户: %s\n", user.Name)
    }
}
```

#### 2. Gin框架

```go
// Gin Web框架
package main

import (
    "github.com/gin-gonic/gin"
    "net/http"
)

func main() {
    r := gin.Default() // 创建默认的Gin实例
    
    // 中间件
    r.Use(gin.Logger())    // 日志中间件
    r.Use(gin.Recovery())  // 恢复中间件
    
    // 路由组
    api := r.Group("/api")
    {
        // GET请求
        api.GET("/users", getUsers)
        api.GET("/users/:id", getUserByID)
        
        // POST请求
        api.POST("/users", createUser)
        
        // PUT请求
        api.PUT("/users/:id", updateUser)
        
        // DELETE请求
        api.DELETE("/users/:id", deleteUser)
    }
    
    r.Run(":8080")
}

// 处理函数
func getUsers(c *gin.Context) {
    users := []User{
        {ID: 1, Name: "张三", Email: "zhangsan@example.com"},
        {ID: 2, Name: "李四", Email: "lisi@example.com"},
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": users,
    })
}

func getUserByID(c *gin.Context) {
    id := c.Param("id") // 获取路径参数
    
    user := User{ID: 1, Name: "张三"}
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": user,
    })
}

func createUser(c *gin.Context) {
    var user User
    
    // 绑定JSON数据
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": err.Error(),
        })
        return
    }
    
    // 创建用户逻辑...
    c.JSON(http.StatusCreated, gin.H{
        "success": true,
        "data": user,
    })
}
```

#### 3. PostgreSQL基础

```sql
-- PostgreSQL基础语法

-- 创建表
CREATE TABLE users (
    id SERIAL PRIMARY KEY,           -- 自增主键
    name VARCHAR(100) NOT NULL,      -- 姓名，非空
    email VARCHAR(255) UNIQUE,       -- 邮箱，唯一
    age INTEGER CHECK (age > 0),     -- 年龄，大于0
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);

-- 插入数据
INSERT INTO users (name, email, age) VALUES 
    ('张三', 'zhangsan@example.com', 25),
    ('李四', 'lisi@example.com', 30);

-- 查询数据
SELECT id, name, email, age 
FROM users 
WHERE age > 20 
ORDER BY created_at DESC 
LIMIT 10;

-- 更新数据
UPDATE users 
SET name = '张三丰', updated_at = CURRENT_TIMESTAMP 
WHERE id = 1;

-- 删除数据
DELETE FROM users WHERE id = 1;

-- 复杂查询
SELECT 
    u.name,
    COUNT(t.id) as ticket_count,
    AVG(t.priority) as avg_priority
FROM users u
LEFT JOIN tickets t ON u.id = t.assignee_id
WHERE u.created_at >= '2024-01-01'
GROUP BY u.id, u.name
HAVING COUNT(t.id) > 0
ORDER BY ticket_count DESC;
```

#### 4. Ent ORM

```go
// Ent ORM使用示例

// 定义Schema
type User struct {
    ent.Schema
}

func (User) Fields() []ent.Field {
    return []ent.Field{
        field.String("name").NotEmpty(),
        field.String("email").Unique(),
        field.Int("age").Positive(),
        field.Time("created_at").Default(time.Now),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

func (User) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("tickets", Ticket.Type),
    }
}

// 数据库操作
func CreateUser(ctx context.Context, client *ent.Client, user *ent.User) error {
    return client.User.
        Create().
        SetName(user.Name).
        SetEmail(user.Email).
        SetAge(user.Age).
        Exec(ctx)
}

func GetUserByID(ctx context.Context, client *ent.Client, id int) (*ent.User, error) {
    return client.User.Get(ctx, id)
}

func GetUsersWithTickets(ctx context.Context, client *ent.Client) ([]*ent.User, error) {
    return client.User.
        Query().
        WithTickets(). // 预加载关联数据
        All(ctx)
}

func UpdateUser(ctx context.Context, client *ent.Client, id int, updates map[string]interface{}) error {
    update := client.User.UpdateOneID(id)
    
    if name, ok := updates["name"].(string); ok {
        update.SetName(name)
    }
    if email, ok := updates["email"].(string); ok {
        update.SetEmail(email)
    }
    
    return update.Exec(ctx)
}

func DeleteUser(ctx context.Context, client *ent.Client, id int) error {
    return client.User.DeleteOneID(id).Exec(ctx)
}
```

## 🎯 学习建议

### 1. 学习顺序

1. **JavaScript基础** → **React基础** → **TypeScript**
2. **Next.js** → **Tailwind CSS** → **Ant Design**
3. **Go基础** → **Gin框架** → **PostgreSQL** → **Ent ORM**

### 2. 实践项目

- 前端：从简单的Todo应用开始，逐步增加复杂度
- 后端：从简单的CRUD API开始，逐步添加业务逻辑

### 3. 学习资源

- **官方文档**：React、Next.js、Go、PostgreSQL官方文档
- **在线课程**：Udemy、Coursera、慕课网等平台
- **实践项目**：GitHub上找开源项目学习

### 4. 开发工具

- **IDE**：VS Code（前端）、GoLand（后端）
- **数据库工具**：DBeaver、pgAdmin
- **API测试**：Postman、Insomnia

## 📖 推荐书籍

### 前端

- 《JavaScript高级程序设计》
- 《React学习手册》
- 《TypeScript编程》

### 后端

- 《Go语言实战》
- 《PostgreSQL必知必会》
- 《数据库系统概念》

## 🚀 快速上手

### 1. 环境搭建

```bash
# 前端环境
node --version  # 检查Node.js版本
npm install -g create-next-app  # 安装Next.js脚手架

# 后端环境
go version  # 检查Go版本
go install entgo.io/ent/cmd/ent@latest  # 安装Ent CLI

# 数据库
# 下载并安装PostgreSQL
```

### 2. 第一个项目

```bash
# 创建Next.js项目
npx create-next-app@latest my-app --typescript --tailwind

# 创建Go项目
mkdir my-api
cd my-api
go mod init my-api
go get github.com/gin-gonic/gin
go get entgo.io/ent
```

### 3. 学习时间安排

- **第1-2周**：JavaScript + React基础
- **第3-4周**：TypeScript + Next.js
- **第5-6周**：Go + Gin基础
- **第7-8周**：PostgreSQL + Ent ORM
- **第9-10周**：项目实战

记住：**实践是最好的学习方式**，多写代码，多调试，多思考！
