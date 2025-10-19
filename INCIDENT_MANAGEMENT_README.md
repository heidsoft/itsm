# ITSM事件管理模块

## 概述

ITSM事件管理模块是一个完整的IT事件管理解决方案，提供事件监控、告警、升级、自动化处理等功能。该模块集成了监控系统（Prometheus/Grafana），支持事件规则引擎，实现实时告警通知和事件影响分析。

## 功能特性

### 核心功能

- **事件监控和告警**: 实时监控IT事件，支持多种告警渠道
- **事件关联分析**: 智能分析事件关联性，避免重复处理
- **事件升级机制**: 自动升级机制，确保重要事件得到及时处理
- **事件自动化处理**: 基于规则引擎的自动化处理流程

### 技术特性

- **监控系统集成**: 集成Prometheus/Grafana监控系统
- **事件规则引擎**: 支持复杂的事件处理规则
- **实时告警通知**: 多渠道实时告警通知
- **事件影响分析**: 智能分析事件对业务的影响
- **完整测试覆盖**: 全面的单元测试和集成测试

## 系统架构

### 后端架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   事件管理API    │    │   规则引擎服务   │    │   监控集成服务   │
│  Incident API   │    │  Rule Engine    │    │  Monitoring     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   告警通知服务   │    │   数据库层       │    │   缓存层         │
│  Alert Service  │    │  Database       │    │  Cache          │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 前端架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   事件管理界面   │    │   监控面板       │    │   告警中心       │
│  Incident UI    │    │  Dashboard      │    │  Alert Center   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   状态管理       │    │   API客户端      │    │   组件库         │
│  State Store    │    │  API Client     │    │  Components     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 数据库设计

### 核心表结构

#### incidents (事件表)

```sql
CREATE TABLE incidents (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'new',
    priority VARCHAR(50) NOT NULL DEFAULT 'medium',
    severity VARCHAR(50) NOT NULL DEFAULT 'medium',
    incident_number VARCHAR(100) UNIQUE NOT NULL,
    reporter_id INTEGER NOT NULL,
    assignee_id INTEGER,
    configuration_item_id INTEGER,
    category VARCHAR(100),
    subcategory VARCHAR(100),
    impact_analysis JSONB,
    root_cause JSONB,
    resolution_steps JSONB,
    detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP,
    escalated_at TIMESTAMP,
    escalation_level INTEGER DEFAULT 0,
    is_automated BOOLEAN DEFAULT false,
    source VARCHAR(50) DEFAULT 'manual',
    metadata JSONB,
    tenant_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

#### incident_events (事件活动记录表)

```sql
CREATE TABLE incident_events (
    id SERIAL PRIMARY KEY,
    incident_id INTEGER NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    severity VARCHAR(50) NOT NULL DEFAULT 'medium',
    data JSONB,
    occurred_at TIMESTAMP NOT NULL DEFAULT NOW(),
    user_id INTEGER,
    source VARCHAR(50) NOT NULL DEFAULT 'system',
    metadata JSONB,
    tenant_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

#### incident_alerts (事件告警表)

```sql
CREATE TABLE incident_alerts (
    id SERIAL PRIMARY KEY,
    incident_id INTEGER NOT NULL,
    alert_type VARCHAR(100) NOT NULL,
    alert_name VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL DEFAULT 'medium',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    channels JSONB,
    recipients JSONB,
    triggered_at TIMESTAMP NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMP,
    resolved_at TIMESTAMP,
    acknowledged_by INTEGER,
    metadata JSONB,
    tenant_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

#### incident_metrics (事件指标表)

```sql
CREATE TABLE incident_metrics (
    id SERIAL PRIMARY KEY,
    incident_id INTEGER NOT NULL,
    metric_type VARCHAR(100) NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    metric_value DECIMAL(10,2) NOT NULL,
    unit VARCHAR(50),
    measured_at TIMESTAMP NOT NULL DEFAULT NOW(),
    tags JSONB,
    metadata JSONB,
    tenant_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

#### incident_rules (事件规则表)

```sql
CREATE TABLE incident_rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(100) NOT NULL,
    conditions JSONB,
    actions JSONB,
    priority VARCHAR(50) NOT NULL DEFAULT 'medium',
    is_active BOOLEAN NOT NULL DEFAULT true,
    execution_count INTEGER NOT NULL DEFAULT 0,
    last_executed_at TIMESTAMP,
    metadata JSONB,
    tenant_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## API接口

### 事件管理接口

#### 创建事件

```http
POST /api/v1/incidents
Content-Type: application/json

{
    "title": "服务器CPU使用率过高",
    "description": "生产环境Web服务器CPU使用率持续超过90%",
    "priority": "high",
    "severity": "high",
    "category": "performance",
    "subcategory": "cpu",
    "source": "monitoring"
}
```

#### 获取事件列表

```http
GET /api/v1/incidents?page=1&size=10&status=new&priority=high
```

#### 获取事件详情

```http
GET /api/v1/incidents/{id}
```

#### 更新事件

```http
PUT /api/v1/incidents/{id}
Content-Type: application/json

{
    "status": "in_progress",
    "assignee_id": 123,
    "priority": "urgent"
}
```

#### 删除事件

```http
DELETE /api/v1/incidents/{id}
```

### 事件升级接口

#### 升级事件

```http
POST /api/v1/incidents/escalate
Content-Type: application/json

{
    "incident_id": 123,
    "escalation_level": 1,
    "reason": "事件处理时间过长",
    "notify_users": [1, 2, 3],
    "auto_assign": true
}
```

### 监控接口

#### 获取监控数据

```http
POST /api/v1/incidents/monitoring
Content-Type: application/json

{
    "start_time": "2024-01-01T00:00:00Z",
    "end_time": "2024-01-31T23:59:59Z",
    "category": "performance"
}
```

#### 分析事件影响

```http
GET /api/v1/incidents/{id}/impact
```

### 告警接口

#### 创建告警

```http
POST /api/v1/incidents/alerts
Content-Type: application/json

{
    "incident_id": 123,
    "alert_type": "escalation",
    "alert_name": "事件升级告警",
    "message": "事件已升级到下一级别",
    "severity": "high",
    "channels": ["email", "sms"],
    "recipients": ["manager@company.com"]
}
```

#### 确认告警

```http
POST /api/v1/incidents/alerts/{id}/acknowledge
```

#### 解决告警

```http
POST /api/v1/incidents/alerts/{id}/resolve
```

#### 获取活跃告警

```http
GET /api/v1/incidents/alerts/active?page=1&size=10
```

## 规则引擎

### 规则类型

#### 1. 升级规则 (escalation)

```json
{
    "name": "高优先级事件自动升级",
    "description": "当事件优先级为high或urgent时，自动升级到下一级别",
    "rule_type": "escalation",
    "conditions": {
        "priority": ["high", "urgent"],
        "status": "new"
    },
    "actions": [
        {
            "type": "escalate",
            "level": 1,
            "message": "事件优先级较高，已自动升级"
        },
        {
            "type": "notify",
            "channels": ["email", "sms"],
            "recipients": ["manager@company.com"]
        }
    ],
    "priority": "high",
    "is_active": true
}
```

#### 2. 告警规则 (alert)

```json
{
    "name": "长时间未处理事件告警",
    "description": "当事件超过24小时未处理时，发送告警通知",
    "rule_type": "alert",
    "conditions": {
        "status": "in_progress",
        "time": {
            "field": "created_at",
            "operator": ">",
            "duration": "24h"
        }
    },
    "actions": [
        {
            "type": "notify",
            "channels": ["email", "slack"],
            "recipients": ["team@company.com"],
            "message": "事件长时间未处理，需要关注"
        }
    ],
    "priority": "medium",
    "is_active": true
}
```

#### 3. 分配规则 (assignment)

```json
{
    "name": "严重事件自动分配",
    "description": "当事件严重程度为critical时，自动分配给高级工程师",
    "rule_type": "assignment",
    "conditions": {
        "severity": "critical",
        "status": "new"
    },
    "actions": [
        {
            "type": "assign",
            "assignee_id": 1,
            "message": "严重事件已自动分配给高级工程师"
        }
    ],
    "priority": "high",
    "is_active": true
}
```

### 条件类型

#### 优先级条件

```json
{
    "priority": ["high", "urgent"]
}
```

#### 严重程度条件

```json
{
    "severity": ["critical", "high"]
}
```

#### 状态条件

```json
{
    "status": ["new", "in_progress"]
}
```

#### 时间条件

```json
{
    "time": {
        "field": "created_at",
        "operator": ">",
        "duration": "24h"
    }
}
```

#### 分类条件

```json
{
    "category": ["performance", "security"]
}
```

### 动作类型

#### 升级动作

```json
{
    "type": "escalate",
    "level": 1,
    "reason": "自动升级",
    "notify_users": [1, 2, 3],
    "auto_assign": true
}
```

#### 通知动作

```json
{
    "type": "notify",
    "channels": ["email", "sms", "slack"],
    "recipients": ["manager@company.com"],
    "message": "事件需要关注",
    "severity": "high"
}
```

#### 分配动作

```json
{
    "type": "assign",
    "assignee_id": 123,
    "reason": "自动分配"
}
```

#### 状态变更动作

```json
{
    "type": "change_status",
    "status": "in_progress",
    "reason": "自动状态变更"
}
```

#### 指标收集动作

```json
{
    "type": "collect_metric",
    "metric_type": "response_time",
    "metric_name": "平均响应时间",
    "metric_value": 2.5,
    "unit": "秒",
    "tags": {
        "environment": "production"
    }
}
```

## 监控集成

### Prometheus集成

#### 指标收集

```go
// 从Prometheus收集指标
func (s *IncidentMonitoringService) CollectMetricsFromPrometheus(ctx context.Context, queries []string, tenantID int) error {
    for _, query := range queries {
        metrics, err := s.queryPrometheus(query)
        if err != nil {
            continue
        }
        
        for _, result := range metrics.Data.Result {
            err = s.createIncidentMetricFromPrometheus(ctx, result.Metric, result.Value, tenantID)
            if err != nil {
                s.logger.Errorw("Failed to create incident metric", "error", err)
            }
        }
    }
    return nil
}
```

#### 常用查询

```promql
# CPU使用率
cpu_usage_percent{instance="server-01"}

# 内存使用率
memory_usage_percent{instance="server-01"}

# 磁盘使用率
disk_usage_percent{instance="server-01"}

# 网络延迟
network_latency_ms{instance="server-01"}

# 服务响应时间
service_response_time_ms{service="web"}
```

### Grafana集成

#### 仪表板配置

```json
{
    "dashboard": {
        "title": "ITSM Incident Monitoring Dashboard",
        "panels": [
            {
                "title": "Incident Count by Status",
                "type": "stat",
                "targets": [
                    {
                        "expr": "incident_count_by_status"
                    }
                ]
            },
            {
                "title": "Incident Resolution Time",
                "type": "graph",
                "targets": [
                    {
                        "expr": "incident_resolution_time"
                    }
                ]
            },
            {
                "title": "Critical Incidents",
                "type": "table",
                "targets": [
                    {
                        "expr": "critical_incidents"
                    }
                ]
            }
        ]
    }
}
```

## 告警通知

### 告警渠道

#### 邮件告警

```go
type EmailChannel struct {
    smtpHost     string
    smtpPort     int
    smtpUsername string
    smtpPassword string
    fromEmail    string
}

func (c *EmailChannel) Send(ctx context.Context, alert *dto.IncidentAlertResponse) error {
    // 实现邮件发送逻辑
    return nil
}
```

#### 短信告警

```go
type SMSChannel struct {
    apiKey    string
    apiSecret string
    signName  string
}

func (c *SMSChannel) Send(ctx context.Context, alert *dto.IncidentAlertResponse) error {
    // 实现短信发送逻辑
    return nil
}
```

#### Slack告警

```go
type SlackChannel struct {
    webhookURL string
    channel    string
}

func (c *SlackChannel) Send(ctx context.Context, alert *dto.IncidentAlertResponse) error {
    // 实现Slack发送逻辑
    return nil
}
```

#### Webhook告警

```go
type WebhookChannel struct {
    url     string
    method  string
    headers map[string]string
}

func (c *WebhookChannel) Send(ctx context.Context, alert *dto.IncidentAlertResponse) error {
    // 实现Webhook发送逻辑
    return nil
}
```

### 告警模板

#### 升级告警模板

```
主题: 事件升级告警 - {incident_number}
内容: 
事件 {incident_number} 已升级到级别 {escalation_level}

事件标题: {title}
严重程度: {severity}
优先级: {priority}
升级原因: {reason}

请及时处理此事件。
```

#### SLA违规告警模板

```
主题: SLA违规告警 - {incident_number}
内容:
事件 {incident_number} 违反SLA: {violation_type}

事件标题: {title}
违规类型: {violation_type}
当前状态: {status}
处理时间: {processing_time}

请立即处理此事件以避免进一步违规。
```

## 前端组件

### 事件管理主组件

```tsx
export const IncidentManagement: React.FC = () => {
    const [incidents, setIncidents] = useState<Incident[]>([]);
    const [loading, setLoading] = useState(false);
    const [filters, setFilters] = useState<any>({});

    // 获取事件列表
    const fetchIncidents = useCallback(async () => {
        setLoading(true);
        try {
            const response = await fetch("/api/v1/incidents", {
                method: "GET",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${localStorage.getItem("access_token")}`,
                    "X-Tenant-ID": localStorage.getItem("tenant_id") || "",
                },
            });

            if (!response.ok) {
                throw new Error("获取事件列表失败");
            }

            const data = await response.json();
            setIncidents(data.data || []);
        } catch (error) {
            message.error("获取事件列表失败");
        } finally {
            setLoading(false);
        }
    }, []);

    return (
        <div className="incident-management">
            {/* 统计卡片 */}
            <Row gutter={16} className="mb-6">
                <Col span={6}>
                    <Card>
                        <Statistic
                            title="总事件数"
                            value={total}
                            prefix={<InfoCircleOutlined />}
                        />
                    </Card>
                </Col>
                {/* 其他统计卡片 */}
            </Row>

            {/* 筛选器 */}
            <Card className="mb-6">
                <Row gutter={16} align="middle">
                    <Col span={6}>
                        <Input
                            placeholder="搜索事件标题或编号"
                            prefix={<SearchOutlined />}
                            onChange={(e) => handleFilterChange("search", e.target.value)}
                        />
                    </Col>
                    {/* 其他筛选器 */}
                </Row>
            </Card>

            {/* 事件列表 */}
            <Card>
                <Table
                    columns={columns}
                    dataSource={incidents}
                    loading={loading}
                    rowKey="id"
                    pagination={{
                        current: currentPage,
                        pageSize: pageSize,
                        total: total,
                        showSizeChanger: true,
                        showQuickJumper: true,
                        showTotal: (total, range) =>
                            `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
                    }}
                    onChange={handleTableChange}
                />
            </Card>
        </div>
    );
};
```

### 事件详情组件

```tsx
const IncidentDetailDrawer: React.FC<{
    visible: boolean;
    incident: Incident | null;
    onClose: () => void;
}> = ({ visible, incident, onClose }) => {
    const [events, setEvents] = useState<IncidentEvent[]>([]);
    const [alerts, setAlerts] = useState<IncidentAlert[]>([]);
    const [metrics, setMetrics] = useState<IncidentMetric[]>([]);

    return (
        <Drawer
            title={`事件详情 - ${incident?.incident_number}`}
            width={800}
            open={visible}
            onClose={onClose}
        >
            <Tabs defaultActiveKey="overview">
                <TabPane tab="概览" key="overview">
                    <IncidentOverview incident={incident} />
                </TabPane>
                <TabPane tab="活动记录" key="events">
                    <IncidentEvents events={events} />
                </TabPane>
                <TabPane tab="告警" key="alerts">
                    <IncidentAlerts alerts={alerts} />
                </TabPane>
                <TabPane tab="指标" key="metrics">
                    <IncidentMetrics metrics={metrics} />
                </TabPane>
                <TabPane tab="影响分析" key="impact">
                    <IncidentImpactAnalysis incident={incident} />
                </TabPane>
            </Tabs>
        </Drawer>
    );
};
```

## 部署和配置

### 环境要求

- Go 1.21+
- Node.js 18+
- PostgreSQL 13+
- Redis 6+
- Prometheus 2.40+
- Grafana 9.0+

### 配置文件

```yaml
# config.yaml
database:
  host: localhost
  port: 5432
  name: itsm
  username: dev
  password: "123456!@#$%^"
  ssl_mode: disable

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

monitoring:
  prometheus:
    url: http://localhost:9090
    timeout: 30s
  grafana:
    url: http://localhost:3000
    api_key: "your_api_key"

alerting:
  email:
    smtp_host: smtp.example.com
    smtp_port: 587
    smtp_username: alerts@company.com
    smtp_password: "password"
    from_email: alerts@company.com
  sms:
    api_key: "sms_api_key"
    api_secret: "sms_api_secret"
    sign_name: "ITSM系统"
  slack:
    webhook_url: "https://hooks.slack.com/services/xxx/xxx/xxx"
    channel: "#incidents"
```

### 数据库迁移

```bash
# 运行数据库迁移
go run migrate_incident_management.go
```

### 启动服务

```bash
# 启动后端服务
go run main.go

# 启动前端服务
cd itsm-prototype
npm run dev
```

## 测试

### 单元测试

```bash
# 运行后端测试
go test ./service/... -v

# 运行前端测试
cd itsm-prototype
npm test
```

### 集成测试

```bash
# 运行集成测试
go test ./controller/... -v
```

### 性能测试

```bash
# 运行基准测试
go test ./service/... -bench=.
```

## 监控和运维

### 日志监控

- 使用Zap进行结构化日志记录
- 支持日志级别配置
- 集成日志聚合系统

### 性能监控

- 集成Prometheus指标收集
- Grafana仪表板监控
- 告警规则配置

### 健康检查

```http
GET /health
```

### 指标端点

```http
GET /metrics
```

## 故障排除

### 常见问题

#### 1. 事件创建失败

- 检查数据库连接
- 验证租户ID
- 检查权限配置

#### 2. 告警发送失败

- 检查告警渠道配置
- 验证网络连接
- 检查认证信息

#### 3. 规则引擎不工作

- 检查规则配置
- 验证条件语法
- 查看执行日志

#### 4. 监控数据缺失

- 检查Prometheus连接
- 验证查询语句
- 检查数据权限

### 日志分析

```bash
# 查看事件服务日志
tail -f logs/incident_service.log

# 查看规则引擎日志
tail -f logs/rule_engine.log

# 查看告警服务日志
tail -f logs/alerting_service.log
```

## 扩展开发

### 添加新的告警渠道

1. 实现AlertChannel接口
2. 在getAlertChannels方法中注册
3. 添加配置参数
4. 编写测试用例

### 添加新的规则类型

1. 定义规则条件接口
2. 实现条件评估逻辑
3. 定义规则动作接口
4. 实现动作执行逻辑
5. 更新规则解析器

### 添加新的监控指标

1. 定义指标类型
2. 实现指标收集逻辑
3. 添加Grafana面板
4. 配置告警规则

## 贡献指南

### 代码规范

- 遵循Go代码规范
- 使用gofmt格式化代码
- 编写完整的测试用例
- 添加必要的注释

### 提交规范

- 使用语义化提交信息
- 一个提交只做一件事
- 添加测试用例
- 更新文档

### 代码审查

- 所有代码都需要审查
- 检查代码质量
- 验证测试覆盖
- 确认文档更新

## 许可证

本项目采用MIT许可证，详见LICENSE文件。

## 联系方式

如有问题或建议，请联系：

- 邮箱: <support@company.com>
- 文档: <https://docs.company.com/itsm>
- 问题反馈: <https://github.com/company/itsm/issues>
