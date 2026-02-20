# ITSM自动化测试框架
# ITSM Automated Test Framework

## 概述

本测试框架为ITSM系统提供完整的自动化测试解决方案，支持API测试、数据库测试和UI测试。

## 目录结构

```
tests/
├── api/                    # API测试
│   ├── __init__.py       # API客户端封装
│   └── test_api_cases.py # API测试用例
├── ui/                     # UI测试
│   ├── __init__.py       # UI客户端封装
│   └── test_ui_cases.py  # UI测试用例
├── database/               # 数据库测试
│   └── __init__.py       # 数据库客户端
├── fixtures/               # 测试数据生成器
│   └── __init__.py       # Fixtures
├── reports/                # 测试报告输出目录
├── config.ini              # 测试配置文件
├── pytest.ini              # Pytest配置
├── requirements.txt        # Python依赖
└── run_tests.py          # 测试运行脚本
```

## 快速开始

### 1. 安装依赖

```bash
cd tests
pip install -r requirements.txt
```

### 2. 安装Playwright浏览器

```bash
playwright install chromium firefox webkit
```

### 3. 运行测试

#### 运行所有测试
```bash
python run_tests.py --type all
```

#### 仅运行API测试
```bash
python run_tests.py --type api
```

#### 仅运行UI测试
```bash
python run_tests.py --type ui
```

#### 运行冒烟测试
```bash
python run_tests.py --markers smoke
```

#### 运行回归测试(并行)
```bash
python run_tests.py --type regression --parallel
```

## 测试标记

| 标记 | 说明 |
|------|------|
| `@pytest.mark.smoke` | 冒烟测试 |
| `@pytest.mark.regression` | 回归测试 |
| `@pytest.mark.api` | API测试 |
| `@pytest.mark.ui` | UI测试 |
| `@pytest.mark.database` | 数据库测试 |
| `@pytest.mark.critical` | 关键路径测试 |

## 测试配置

编辑 `config.ini` 文件配置测试环境:

```ini
[api]
base_url = http://localhost:8080
timeout = 30

[database]
host = localhost
port = 5432
database = itsm
user = postgres
password = password

[ui]
base_url = http://localhost:3000
headless = true
```

## 使用示例

### API测试

```python
from tests.api import ITSMAPIClient

# 创建已认证的客户端
client = ITSMAPIClient()
client.login('admin', 'admin123')

# 获取工单列表
result = client.get_tickets({'page': 1, 'page_size': 10})

# 创建工单
ticket_data = {
    'title': 'Test Ticket',
    'description': 'Test Description',
    'priority': 'medium'
}
result = client.create_ticket(ticket_data)
```

### 数据库测试

```python
from tests.database import ITSMDBClient

# 创建数据库客户端
db = ITSMDBClient()

# 查询工单
tickets = db.get_all_tickets(limit=10)

# 统计工单
stats = db.count_tickets_by_status()
```

### UI测试

```python
from tests.ui import ITSMUIClient

# 创建UI客户端
client = ITSMUIClient()
client.start()

# 登录
client.login('admin', 'admin123')

# 创建工单
client.create_ticket('Test Ticket', 'Description')
```

## 测试数据Fixtures

系统提供多种测试数据生成器:

```python
from tests.fixtures import (
    TicketFixture,
    UserFixture,
    KnowledgeArticleFixture,
    IncidentFixture,
    ProblemFixture,
    ChangeFixture,
    SLAFixture,
    ServiceCatalogFixture
)

# 生成测试数据
ticket_data = TicketFixture.create()
users = UserFixture.create_batch(5)
```

## 持续集成

GitHub Actions工作流已配置:

- 自动运行冒烟测试
- 定时运行回归测试
- 生成HTML测试报告
- 集成到CI/CD流程

触发方式:
1. 代码推送
2. Pull Request创建
3. 定时任务
4. 手动触发

## 报告查看

测试报告保存在 `tests/reports/` 目录:

- `report.html` - HTML测试报告
- `junit.xml` - JUnit XML报告

使用pytest-html生成的报告支持:

- 测试结果统计
- 失败详情
- 执行时间
- 历史趋势

## 故障排除

### 连接失败

1. 检查服务是否启动
2. 验证配置文件中的URL
3. 检查网络连接

### 数据库连接失败

1. 确认PostgreSQL服务运行
2. 验证数据库凭据
3. 检查数据库是否存在

### UI测试失败

1. 确保Playwright已安装
2. 检查浏览器驱动
3. 验证页面选择器

## 扩展测试

### 添加新API测试

1. 在 `api/test_api_cases.py` 中添加测试类
2. 使用 `api_client` fixture
3. 编写测试用例

### 添加新UI测试

1. 在 `ui/test_ui_cases.py` 中添加测试类
2. 使用 `ui_client` fixture
3. 编写测试用例

### 添加新Fixtures

1. 在 `fixtures/__init__.py` 中添加数据生成器
2. 实现 `create()` 方法
3. 可选实现 `create_batch()` 方法

## 性能测试

使用pytest-xdist实现并行测试:

```bash
pytest -n auto           # 自动CPU核心数
pytest -n 4             # 4个并行进程
```

## 监控测试

使用pytest-timeout设置超时:

```bash
pytest --timeout=300     # 5分钟超时
```

## 许可证

MIT License
