# ITSM产品模块架构 - 可视化图表

## 1. 系统整体架构图

```mermaid
graph TB
    subgraph "前端层 (Next.js)"
        A1[用户界面]
        A2[业务组件]
        A3[状态管理]
        A4[API客户端]
    end
    
    subgraph "API网关层"
        B1[路由管理]
        B2[认证中间件]
        B3[权限中间件]
        B4[租户中间件]
    end
    
    subgraph "业务服务层 (Go)"
        C1[工单服务]
        C2[事件服务]
        C3[问题服务]
        C4[变更服务]
        C5[服务目录服务]
        C6[CMDB服务]
        C7[知识库服务]
        C8[SLA服务]
        C9[工作流服务]
        C10[审批服务]
        C11[AI服务]
        C12[报表服务]
    end
    
    subgraph "数据层"
        D1[(PostgreSQL)]
        D2[(Redis)]
    end
    
    subgraph "外部服务"
        E1[云服务提供商]
        E2[LLM服务]
    end
    
    A1 --> A2
    A2 --> A3
    A3 --> A4
    A4 --> B1
    B1 --> B2
    B2 --> B3
    B3 --> B4
    B4 --> C1
    B4 --> C2
    B4 --> C3
    B4 --> C4
    B4 --> C5
    B4 --> C6
    B4 --> C7
    B4 --> C8
    B4 --> C9
    B4 --> C10
    B4 --> C11
    B4 --> C12
    
    C1 --> D1
    C2 --> D1
    C3 --> D1
    C4 --> D1
    C5 --> D1
    C6 --> D1
    C7 --> D1
    C8 --> D1
    C9 --> D1
    C10 --> D1
    C11 --> D1
    C12 --> D1
    
    C1 --> D2
    C5 --> D2
    C8 --> D2
    
    C5 --> E1
    C11 --> E2
```

## 2. 核心业务模块关系图

```mermaid
graph LR
    subgraph "服务请求流程"
        A[服务目录] --> B[服务请求]
        B --> C[审批流程]
        C --> D[资源交付]
    end
    
    subgraph "工单生命周期"
        E[工单创建] --> F[工单处理]
        F --> G[工单解决]
        G --> H[工单关闭]
    end
    
    subgraph "事件-问题-变更"
        I[事件] --> J[问题]
        J --> K[变更]
        K --> L[CMDB]
    end
    
    subgraph "支撑模块"
        M[SLA监控]
        N[知识库]
        O[工作流]
        P[审批管理]
    end
    
    B --> E
    E --> I
    I --> M
    J --> N
    K --> O
    C --> P
    D --> L
```

## 3. 模块依赖关系图

```mermaid
graph TD
    A[认证授权模块] --> B[用户管理模块]
    A --> C[工单管理模块]
    A --> D[事件管理模块]
    A --> E[问题管理模块]
    A --> F[变更管理模块]
    A --> G[服务目录模块]
    
    C --> H[SLA管理模块]
    C --> I[工作流模块]
    C --> J[审批管理模块]
    C --> K[通知模块]
    C --> L[知识库模块]
    
    D --> E
    E --> F
    F --> M[CMDB模块]
    
    G --> N[服务请求模块]
    N --> J
    N --> O[资源交付模块]
    
    P[AI功能模块] --> C
    P --> D
    P --> L
    
    Q[报表分析模块] --> C
    Q --> D
    Q --> E
    Q --> F
    Q --> H
    
    R[审计日志模块] --> C
    R --> D
    R --> E
    R --> F
```

## 4. 数据流架构图

```mermaid
sequenceDiagram
    participant U as 用户
    participant F as 前端
    participant API as API网关
    participant S as 业务服务
    participant DB as 数据库
    participant Cache as 缓存
    participant Ext as 外部服务
    
    U->>F: 操作请求
    F->>API: HTTP请求
    API->>API: 认证/权限检查
    API->>S: 业务处理
    S->>Cache: 查询缓存
    alt 缓存命中
        Cache-->>S: 返回缓存数据
    else 缓存未命中
        S->>DB: 查询数据库
        DB-->>S: 返回数据
        S->>Cache: 更新缓存
    end
    S->>Ext: 调用外部服务(可选)
    Ext-->>S: 返回结果
    S-->>API: 返回结果
    API-->>F: HTTP响应
    F-->>U: 展示结果
```

## 5. 工单管理模块详细架构

```mermaid
graph TB
    subgraph "工单管理模块"
        A[工单核心服务]
        B[工单分类服务]
        C[工单标签服务]
        D[工单模板服务]
        E[工单评论服务]
        F[工单附件服务]
        G[工单通知服务]
        H[工单评分服务]
        I[工单视图服务]
        J[工单分配服务]
        K[工单自动化服务]
        L[工单工作流服务]
    end
    
    subgraph "支撑服务"
        M[SLA服务]
        N[审批服务]
        O[通知服务]
        P[工作流引擎]
    end
    
    A --> B
    A --> C
    A --> D
    A --> E
    A --> F
    A --> G
    A --> H
    A --> I
    A --> J
    A --> K
    A --> L
    
    A --> M
    A --> N
    A --> O
    L --> P
```

## 6. 服务请求审批流程架构

```mermaid
graph TD
    A[服务目录] --> B[创建服务请求]
    B --> C{判断审批流程}
    C -->|标准流程| D[主管审批]
    C -->|简化流程| E[IT审批]
    C -->|快速流程| F[直接IT审批]
    C -->|条件流程| G[动态审批路径]
    
    D --> H[IT审批]
    H --> I{需要安全审批?}
    I -->|是| J[安全审批]
    I -->|否| K[审批通过]
    J --> K
    
    K --> L[资源交付]
    L --> M[交付完成]
    
    N[审批拒绝] --> O[通知申请人]
    
    D -.拒绝.-> N
    H -.拒绝.-> N
    J -.拒绝.-> N
```

## 7. 多租户架构图

```mermaid
graph TB
    subgraph "租户A"
        A1[用户数据A]
        A2[工单数据A]
        A3[配置数据A]
    end
    
    subgraph "租户B"
        B1[用户数据B]
        B2[工单数据B]
        B3[配置数据B]
    end
    
    subgraph "共享服务"
        C1[认证服务]
        C2[租户中间件]
        C3[数据隔离层]
    end
    
    subgraph "数据库"
        D1[(租户A数据)]
        D2[(租户B数据)]
    end
    
    A1 --> C2
    B1 --> C2
    C2 --> C3
    C3 --> D1
    C3 --> D2
```

## 8. 权限控制架构图

```mermaid
graph TD
    A[用户] --> B[角色]
    B --> C[权限]
    C --> D[资源]
    
    subgraph "角色类型"
        E[超级管理员]
        F[管理员]
        G[经理]
        H[技术人员]
        I[普通用户]
    end
    
    subgraph "权限类型"
        J[创建]
        K[读取]
        L[更新]
        M[删除]
        N[审批]
        O[分配]
    end
    
    subgraph "资源类型"
        P[工单]
        Q[变更]
        R[服务请求]
        S[CMDB]
        T[知识库]
    end
    
    B --> E
    B --> F
    B --> G
    B --> H
    B --> I
    
    C --> J
    C --> K
    C --> L
    C --> M
    C --> N
    C --> O
    
    D --> P
    D --> Q
    D --> R
    D --> S
    D --> T
```

## 9. 工作流引擎架构图

```mermaid
graph TB
    subgraph "工作流设计"
        A[BPMN设计器]
        B[流程定义]
        C[流程部署]
    end
    
    subgraph "工作流引擎"
        D[流程实例管理]
        E[任务管理]
        F[变量管理]
        G[事件处理]
    end
    
    subgraph "集成点"
        H[工单集成]
        I[审批集成]
        J[自动化集成]
    end
    
    A --> B
    B --> C
    C --> D
    D --> E
    D --> F
    D --> G
    E --> H
    E --> I
    G --> J
```

## 10. AI功能架构图

```mermaid
graph TB
    subgraph "AI服务层"
        A[LLM网关]
        B[向量存储]
        C[嵌入服务]
        D[RAG服务]
    end
    
    subgraph "AI功能"
        E[智能分类]
        F[智能搜索]
        G[智能摘要]
        H[AI对话]
        I[根因分析]
        J[趋势预测]
    end
    
    subgraph "应用场景"
        K[工单分类]
        L[知识检索]
        M[工单摘要]
        N[智能助手]
        O[问题分析]
        P[容量预测]
    end
    
    A --> E
    A --> F
    A --> G
    A --> H
    A --> I
    A --> J
    
    B --> F
    C --> B
    D --> F
    
    E --> K
    F --> L
    G --> M
    H --> N
    I --> O
    J --> P
```

## 11. 前端模块架构图

```mermaid
graph TB
    subgraph "页面层"
        A1[仪表盘]
        A2[工单管理]
        A3[事件管理]
        A4[问题管理]
        A5[变更管理]
        A6[服务目录]
        A7[CMDB]
        A8[知识库]
    end
    
    subgraph "组件层"
        B1[业务组件]
        B2[通用组件]
        B3[UI组件]
    end
    
    subgraph "状态层"
        C1[工单Store]
        C2[用户Store]
        C3[认证Store]
    end
    
    subgraph "服务层"
        D1[API客户端]
        D2[业务服务]
        D3[工具函数]
    end
    
    A1 --> B1
    A2 --> B1
    A3 --> B1
    A4 --> B1
    A5 --> B1
    A6 --> B1
    A7 --> B1
    A8 --> B1
    
    B1 --> B2
    B2 --> B3
    
    B1 --> C1
    B1 --> C2
    B1 --> C3
    
    C1 --> D1
    C2 --> D1
    C3 --> D1
    
    D1 --> D2
    D2 --> D3
```

## 12. 后端分层架构图

```mermaid
graph TB
    subgraph "Controller层"
        A1[工单Controller]
        A2[事件Controller]
        A3[变更Controller]
        A4[服务Controller]
    end
    
    subgraph "Service层"
        B1[工单Service]
        B2[事件Service]
        B3[变更Service]
        B4[服务Service]
    end
    
    subgraph "Repository层 (Ent)"
        C1[工单Entity]
        C2[事件Entity]
        C3[变更Entity]
        C4[服务Entity]
    end
    
    subgraph "数据层"
        D1[(PostgreSQL)]
        D2[(Redis)]
    end
    
    A1 --> B1
    A2 --> B2
    A3 --> B3
    A4 --> B4
    
    B1 --> C1
    B2 --> C2
    B3 --> C3
    B4 --> C4
    
    C1 --> D1
    C2 --> D1
    C3 --> D1
    C4 --> D1
    
    B1 --> D2
    B4 --> D2
```

## 13. 数据模型关系图（核心实体）

```mermaid
erDiagram
    User ||--o{ Ticket : creates
    User ||--o{ Ticket : assigned_to
    User ||--o{ ServiceRequest : creates
    User ||--o{ ServiceRequestApproval : approves
    
    Ticket ||--o{ TicketComment : has
    Ticket ||--o{ TicketAttachment : has
    Ticket ||--o{ TicketNotification : triggers
    Ticket }o--|| TicketCategory : belongs_to
    Ticket }o--o{ TicketTag : tagged_with
    
    ServiceRequest ||--o{ ServiceRequestApproval : requires
    ServiceRequest }o--|| ServiceCatalog : requests
    ServiceRequest ||--o{ ProvisioningTask : triggers
    
    Change ||--o{ ChangeApproval : requires
    Change }o--o{ ConfigurationItem : affects
    
    Problem ||--o{ RootCauseAnalysis : analyzed_by
    Problem }o--o{ Incident : caused_by
    
    Incident }o--o{ Problem : creates
    Incident }o--o{ Change : requires
    
    ConfigurationItem }o--o{ CIRelationship : relates_to
    ConfigurationItem }o--|| CIType : is_type
    
    Ticket }o--o{ SLADefinition : monitored_by
    SLADefinition ||--o{ SLAViolation : violates
    
    Workflow ||--o{ WorkflowInstance : instantiates
    WorkflowInstance ||--o{ ProcessTask : contains
    
    Tenant ||--o{ User : contains
    Tenant ||--o{ Ticket : contains
    Tenant ||--o{ ServiceRequest : contains
```

## 14. API端点组织图

```mermaid
graph LR
    subgraph "/api/v1"
        A[tickets/]
        B[incidents/]
        C[problems/]
        D[changes/]
        E[service-catalogs/]
        F[service-requests/]
        G[cmdb/]
        H[knowledge-articles/]
        I[sla/]
        J[workflows/]
        K[users/]
        L[ai/]
        M[dashboard/]
    end
    
    A --> A1[CRUD操作]
    A --> A2[分配/解决]
    A --> A3[评论/附件]
    A --> A4[审批]
    A --> A5[分析]
    
    E --> E1[目录管理]
    F --> F1[请求管理]
    F --> F2[审批流程]
    F --> F3[资源交付]
    
    I --> I1[定义管理]
    I --> I2[监控]
    I --> I3[告警]
```

## 15. 部署架构图

```mermaid
graph TB
    subgraph "负载均衡层"
        LB[负载均衡器]
    end
    
    subgraph "应用层"
        APP1[应用实例1]
        APP2[应用实例2]
        APP3[应用实例N]
    end
    
    subgraph "数据层"
        DB1[(PostgreSQL主)]
        DB2[(PostgreSQL从)]
        CACHE[(Redis集群)]
    end
    
    subgraph "外部服务"
        EXT1[云服务API]
        EXT2[LLM服务]
    end
    
    LB --> APP1
    LB --> APP2
    LB --> APP3
    
    APP1 --> DB1
    APP2 --> DB1
    APP3 --> DB1
    
    DB1 --> DB2
    
    APP1 --> CACHE
    APP2 --> CACHE
    APP3 --> CACHE
    
    APP1 --> EXT1
    APP2 --> EXT2
```

---

## 图表说明

### 符号说明

- **矩形**：模块/组件
- **圆角矩形**：服务/功能
- **菱形**：判断节点
- **圆形**：数据存储
- **箭头**：依赖/调用关系
- **虚线箭头**：可选/条件关系

### 颜色说明

- **蓝色**：核心业务模块
- **绿色**：支撑模块
- **橙色**：数据层
- **紫色**：外部服务

---

**文档版本**：V1.0  
**创建日期**：2025-12-17
