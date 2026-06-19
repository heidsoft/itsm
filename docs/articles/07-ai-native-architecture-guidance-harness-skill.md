# AI-Native ITSM的架构进化论：Guidance-Harness-Skill三层体系设计

## 从"AI功能"到"AI引擎"，让LLM真正可控可测可扩展

---

凌晨三点，某电商公司的IT系统再次触发告警。数据库连接池耗尽，大量用户无法下单。

在传统的ITSM系统中，这个告警会经历这样的流程：值班工程师被叫醒 → 打开工单系统 → 手动创建工单 → 判断这是什么类型的问题（事件？请求？） → 判断优先级 → 分配给DBA团队 → DBA开始排查...

整个过程可能需要5-10分钟，而每一次延迟都意味着更多的损失。

但在AI-Native ITSM中，这个流程完全不同：告警自动触发工单创建 → AI瞬间分析日志和上下文 → 判断为"数据库性能问题，优先级P0" → 自动分配给DBA团队并附上可能的根因分析 → DBA工程师直接开始排查而不是判断问题...

从"5-10分钟"到"秒级响应"，这不是功能的堆砌，而是**架构的质变**。

要实现这个质变，需要解决三个核心问题：**如何让LLM输出可控**、**如何让AI能力可测**、**如何让场景扩展灵活**。

本文介绍我们在AI-Native ITSM项目中探索的 **Guidance-Harness-Skill 三层架构**，它回答了这三个问题。

---

## 第一部分：为什么AI在ITSM中"不好用"

### 1.1 行业现状：AI是点缀，不是引擎

当前市面上大多数ITSM产品的AI能力是"外挂模式"：

- 现有ITSM系统已经跑得很稳
- 在外面包一层AI能力
- AI失败了就降级到人工处理

这种模式有几个致命问题：

**第一，AI输出不可控。** 当LLM"自由发挥"时，可能给出匪夷所思的分类结果。把"服务器着火"归类为"一般咨询"，把"数据泄露"判断为"低优先级"——这些情况在AI不可控时会发生。

**第二，AI能力不可测。** 没有人知道AI分类准确率是多少，80%还是60%？只有上线后用户投诉才知道。

**第三，AI场景不可扩展。** 新增一个AI能力需要重新训练或重新写Prompt，难以复用已有的AI能力。

这不是真正的AI-Native。

### 1.2 Guidance范式：精细控制LLM

**Guidance** 是一个新兴的LLM编程范式，它的核心思想是：**让Prompt不再是字符串，而是可执行的程序**。

传统的Prompt方式：

```
请分析以下IT工单，返回JSON格式的分类结果：
{"category": "...", "priority": "..."}

工单标题：xxx
工单描述：xxx
```

Guidance方式：

```python
from guidance import select, gen

lm += f'''分析以下IT工单：

工单标题：{ticket_title}
工单描述：{ticket_description}

工单类型：{select(["事件", "服务请求", "问题", "变更"], name="category")}
优先级：{select(["P0紧急", "P1高", "P2中", "P3低"], name="priority")}
紧急程度：{select(["Critical", "High", "Medium", "Low"], name="severity")}
置信度：{gen(regex="0\\.[0-9]+", name="confidence")}
'''
```

关键区别是什么？

**结构约束**。通过`select`，我们告诉LLM只能从预定义的选项中选择，而不是"随便输出什么"。这解决了输出不可控的问题。

**类型安全**。通过`gen(regex=...)`，我们约束输出的格式，比如置信度必须是"0.85"这样的格式，而不是"大约85%"这样的自由文本。

**可编程**。Prompt变成了Python代码，可以循环、条件、复用。这解决了AI能力难以组合和扩展的问题。

### 1.3 ITSM场景下的Guidance应用

把Guidance应用到ITSM的工单分类场景：

```python
from guidance import select, gen, capture

def triage_ticket_guidance(ticket_title: str, ticket_desc: str):
    """工单智能分诊的Guidance程序"""
    return f'''
系统角色：你是IT服务管理分诊专家，根据工单内容判断分类。

工单信息：
- 标题：{ticket_title}
- 描述：{ticket_desc}

分类决策：

类型判断：{select([
    "incident - 系统故障或异常",
    "request - 服务申请或一般请求",
    "problem - 需要根因分析的复杂问题",
    "change - 配置变更或发布",
    "security - 安全相关事件"
], name="category")}

优先级判断：{select([
    "P0 - 核心业务中断，需立即处理",
    "P1 - 非核心但影响明显，需尽快处理",
    "P2 - 一般问题，常规流程",
    "P3 - 次要问题，可安排处理"
], name="priority")}

影响范围：{select([
    "whole_company - 全公司影响",
    "department - 部分部门影响",
    "team - 小团队影响",
    "individual - 个人影响"
], name="impact")}

推荐处理团队：{select([
    "infrastructure - 基础架构组",
    "development - 应用开发组",
    "security - 安全团队",
    "network - 网络组",
    "dba - 数据库组",
    "helpdesk - 一线支持"
], name="team")}

置信度评分：{gen(regex="0\\.\\d{2}", name="confidence")}

判断理由：{gen(max_tokens=50, name="reason")}
'''
```

这个程序有明确的输出结构，每次调用都返回预定义格式的结果。不存在"LLM自由发挥"的问题。

---

## 第二部分：Harness工程：让AI可控可测

### 2.1 Harness工程的核心：G-E-P闭环

**Harness工程**是一种AI系统的工程化方法论，它的核心是 **Generator-Evaluator-Planner** 三角色闭环：

```
┌─────────────────────────────────────────────────────────┐
│                      Planner                            │
│  "根据反馈规划新的Prompt/策略"                           │
└───────────────────────┬─────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────┐
│                     Generator                           │
│  "生成AI输出（Prompt执行结果）"                          │
└───────────────────────┬─────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────┐
│                     Evaluator                           │
│  "评估输出质量，给出反馈"                                │
└───────────────────────┬─────────────────────────────────┘
                        │
                        ▼
              ┌─────────────────┐
              │   回到Planner   │
              │   继续迭代...    │
              └─────────────────┘
```

**Planner** 是策略制定者。它分析问题、制定执行计划、决定使用什么Prompt和参数。

**Generator** 是执行者。它按照Planner的指令，调用LLM生成输出。

**Evaluator** 是质量把关者。它评估Generator的输出质量，决定是否需要重新生成。

这个闭环让AI系统有了"自我改进"的能力。

### 2.2 Evaluator：工单分类的质量度量

在ITSM工单分类场景，Evaluator负责量化AI分类的质量：

```go
// 分类评估结果
type TriageEvaluation struct {
    // 准确性指标
    Accuracy    float64  // 准确率（正确数/总数）
    Precision   float64  // 精确率（预测正确的/预测为正的）
    Recall      float64  // 召回率（预测正确的/实际正确的）

    // 稳定性指标
    Consistency float64  // 一致性（同类型工单分类是否一致）

    // 性能指标
    LatencyP50  float64  // P50延迟（毫秒）
    LatencyP99  float64  // P99延迟（毫秒）

    // 降级指标
    FallbackRate float64 // 降级到关键词的比率
    ErrorRate    float64 // 错误率
}

// 评估报告
type EvaluationReport struct {
    Timestamp      time.Time
    TotalCases     int
    PassedCases    int
    Evaluation     TriageEvaluation
    BadCases       []BadCase  // 分类错误的case列表
    Recommendations []string  // 改进建议
}

// 错误case记录
type BadCase struct {
    TicketID      string
    Input         TriageInput
    Expected      TriageResult
    Actual        TriageResult
    ErrorType     string      // "category_error" | "priority_error" | "team_error"
    Severity      string      // "critical" | "major" | "minor"
}
```

Evaluator的核心价值是：**把AI质量变成可量化、可追踪的指标**。

### 2.3 测试集驱动开发

有了Evaluator，可以建立**测试集驱动的AI开发流程**：

```go
// ITSM工单分类测试集（100+真实case）
var triageTestCases = []TestCase{
    {
        ID:          "TC001",
        Input:       "数据库连接失败，应用无法访问",
        Expected:    TriageResult{Category: "incident", Priority: "P0", Team: "dba"},
        Tags:        []string{"database", "connection", "high_priority"},
    },
    {
        ID:          "TC002",
        Input:       "员工邮箱密码忘记，需要重置",
        Expected:    TriageResult{Category: "request", Priority: "P3", Team: "helpdesk"},
        Tags:        []string{"password", "user_access", "low_priority"},
    },
    {
        ID:          "TC003",
        Input:       "发现SQL注入漏洞，请求安全评审",
        Expected:    TriageResult{Category: "security", Priority: "P1", Team: "security"},
        Tags:        []string{"security", "sql_injection"},
    },
    // ... 100+ more cases
}

// 回归测试：每次Prompt变更后自动跑测试集
func RunRegressionTest(testCases []TestCase, guidanceProgram string) EvaluationReport {
    var results []TriageResult
    var badCases []BadCase

    for _, tc := range testCases {
        result := ExecuteGuidance(guidanceProgram, tc.Input)
        results = append(results, result)

        if !IsCorrect(tc.Expected, result) {
            badCases = append(badCases, BadCase{
                TicketID: tc.ID,
                Input:    tc.Input,
                Expected: tc.Expected,
                Actual:   result,
                ErrorType: classifyError(tc.Expected, result),
            })
        }
    }

    return GenerateReport(results, badCases)
}
```

这个机制保证了：**AI的任何变更都可以被量化，任何退化都可以被提前发现**。

### 2.4 Planner：自适应Prompt优化

当Evaluator发现分类质量下降时，Planner会自动规划优化方案：

```go
// Planner的分析和规划逻辑
type TriagePlanner struct {
    evaluator *TriageEvaluator
    promptDB  *PromptDatabase
}

func (p *TriagePlanner) Plan(currentReport EvaluationReport) *OptimizationPlan {
    // 分析错误模式
    errorPatterns := p.analyzeErrorPatterns(currentReport.BadCases)

    // 生成优化建议
    var recommendations []*Recommendation

    for pattern, count := range errorPatterns {
        if count > 5 {  // 超过5次的问题需要优化
            rec := p.generateRecommendation(pattern)
            recommendations = append(recommendations, rec)
        }
    }

    return &OptimizationPlan{
        CurrentAccuracy: currentReport.Evaluation.Accuracy,
        TargetAccuracy:  0.95,
        Recommendations: recommendations,
        EstimatedImpact: p.estimateImpact(recommendations),
    }
}

// 错误模式分析
func (p *TriagePlanner) analyzeErrorPatterns(badCases []BadCase) map[string]int {
    patterns := make(map[string]int)

    for _, bc := range badCases {
        // 按错误类型和标签聚类
        key := fmt.Sprintf("%s:%v", bc.ErrorType, bc.Input.Tags)
        patterns[key]++
    }

    return patterns
}

// 生成优化建议
func (p *TriagePlanner) generateRecommendation(pattern string) *Recommendation {
    // 根据不同的错误模式生成不同建议
    switch {
    case strings.Contains(pattern, "database"):
        return &Recommendation{
            Type:        "add_examples",
            Description: "增加数据库相关分类的Few-shot示例",
            Examples:    []string{...},
        }
    case strings.Contains(pattern, "security"):
        return &Recommendation{
            Type:        "increase_priority",
            Description: "安全相关问题优先级定义不清晰，建议加强Security类别的Prompt",
        }
    // ...
    }
}
```

Planner的输出可以直接指导Prompt的优化方向。

---

## 第三部分：Skill体系：场景能力扩展

### 3.1 什么是Skill

在AI-Native ITSM中，**Skill是一个可复用、可组合的AI能力单元**。

类比一下：编程语言有函数（Function），Skill就是AI时代的"函数"——输入什么、输出什么、做什么，都是明确约定的。

```go
// Skill接口定义
type Skill interface {
    // Skill的唯一标识
    Name() string

    // Skill的描述（用于日志和调试）
    Description() string

    // 输入校验
    Validate(input interface{}) error

    // 执行Skill的核心逻辑
    Execute(ctx context.Context, input interface{}) (interface{}, error)

    // 获取输出类型
    OutputType() reflect.Type
}
```

### 3.2 Skill与Guidance的结合

每个Skill内部使用Guidance程序来控制LLM：

```go
// TriageSkill：工单分类Skill
type TriageSkill struct {
    guidanceProgram string
    llmClient       LLMClient
}

func (s *TriageSkill) Name() string {
    return "triage"
}

func (s *TriageSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    ticket, ok := input.(*Ticket)
    if !ok {
        return nil, fmt.Errorf("invalid input type: %T", input)
    }

    // 调用Guidance程序
    result, err := s.executeGuidance(ctx, ticket)
    if err != nil {
        return nil, fmt.Errorf("guidance execution failed: %w", err)
    }

    return result, nil
}

func (s *TriageSkill) executeGuidance(ctx context.Context, ticket *Ticket) (*TriageResult, error) {
    program := fmt.Sprintf(`
分析以下IT工单：

标题：%s
描述：%s

类型：{select([
    "incident",
    "request",
    "problem",
    "change",
    "security"
], name="category")}

优先级：{select([
    "P0",
    "P1",
    "P2",
    "P3"
], name="priority")}

处理团队：{select([
    "infrastructure",
    "development",
    "security",
    "network",
    "dba",
    "helpdesk"
], name="team")}

置信度：{gen(regex="0\\.\\d{2}", name="confidence")}
`, ticket.Title, ticket.Description)

    output, err := s.llmClient.Execute(ctx, program)
    if err != nil {
        return nil, err
    }

    return parseTriageResult(output)
}
```

### 3.3 Skill注册与发现

所有Skill注册到统一的注册中心：

```go
// Skill注册中心
type SkillRegistry struct {
    skills    map[string]Skill
    metadata  map[string]SkillMetadata
}

type SkillMetadata struct {
    Name        string
    Description string
    Version     string
    Author      string
    Tags        []string  // 用于发现
}

// 注册Skill
func (r *SkillRegistry) Register(skill Skill) error {
    if err := skill.Validate(); err != nil {
        return fmt.Errorf("skill validation failed: %w", err)
    }

    meta := SkillMetadata{
        Name:        skill.Name(),
        Description: skill.Description(),
        Version:     "1.0.0",
        Tags:        extractTags(skill),
    }

    r.skills[skill.Name()] = skill
    r.metadata[skill.Name()] = meta

    return nil
}

// 按标签发现Skill
func (r *SkillRegistry) DiscoverByTag(tag string) []*SkillMetadata {
    var results []*SkillMetadata

    for _, meta := range r.metadata {
        for _, t := range meta.Tags {
            if t == tag {
                results = append(results, &meta)
                break
            }
        }
    }

    return results
}
```

### 3.4 Skill编排：组合出更强大的能力

单个Skill可能不够用，需要组合多个Skill：

```go
// Skill编排器
type SkillOrchestrator struct {
    registry *SkillRegistry
}

func (o *SkillOrchestrator) ExecutePipeline(ctx context.Context, pipeline []SkillConfig, input interface{}) (interface{}, error) {
    currentOutput := input

    for _, config := range pipeline {
        skill := o.registry.Get(config.Name)
        if skill == nil {
            return nil, fmt.Errorf("skill not found: %s", config.Name)
        }

        // 处理输入转换
        processedInput := o.transformInput(currentOutput, config.InputTransform)

        // 执行Skill
        output, err := skill.Execute(ctx, processedInput)
        if err != nil {
            if config.Required {
                return nil, fmt.Errorf("required skill %s failed: %w", config.Name, err)
            }
            // 可选Skill失败，使用默认值
            output = config.DefaultOutput
        }

        // 处理输出转换
        currentOutput = o.transformOutput(output, config.OutputTransform)
    }

    return currentOutput, nil
}

// 复杂工单处理流水线
ticketPipeline := []SkillConfig{
    {
        Name:    "triage",
        InputTransform: toTicketInput,
    },
    {
        Name:    "summarize",
        Required: false,  // 可选
        InputTransform: toSummaryInput,
        OutputTransform: attachSummary,
    },
    {
        Name:    "kb_retriever",
        Required: false,
        InputTransform: toKBSearchInput,
        OutputTransform: attachKnowledge,
    },
    {
        Name:    "notifier",
        Required: false,
        InputTransform: toNotificationInput,
    },
}

result, err := orchestrator.ExecutePipeline(ctx, ticketPipeline, rawTicket)
```

这个编排机制让**新增AI能力变成添加一个Skill，而不是修改核心代码**。

---

## 第四部分：三层架构整合

### 4.1 完整架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                         用户请求层                                    │
│    告警触发 │ 手动提交 │ API调用 │ 企微/钉钉集成                      │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Skill Orchestrator                               │
│    流水线编排 │ 输入输出转换 │ 错误处理 │ 降级策略                     │
└─────────────────────────────────────────────────────────────────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              ▼                   ▼                   ▼
    ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
    │   TriageSkill   │ │  SummarizeSkill │ │  KBSkill        │
    │                 │ │                 │ │                 │
    │  Guidance程序   │ │  Guidance程序   │ │  Guidance程序   │
    │  LLM-first分类  │ │  自动摘要生成   │ │  RAG问答       │
    └─────────────────┘ └─────────────────┘ └─────────────────┘
              │                   │                   │
              └───────────────────┴───────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Harness Controller                              │
│    Prompt管理 │ 参数配置 │ 执行控制 │ 结果解析                        │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       LLM Gateway                                    │
│    OpenAI │ Claude │ Ollama │ DashScope │ 国产模型                   │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Evaluator Layer                                │
│    准确性评估 │ 性能监控 │ 回归测试 │ 质量报告                        │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
                    ┌─────────────────────────┐
                    │   Feedback to Planner   │
                    │   质量反馈 → Prompt优化  │
                    └─────────────────────────┘
```

### 4.2 核心流程：工单智能分诊

完整流程如下：

**1. 用户/系统提交工单**

```http
POST /api/tickets
{
  "title": "数据库连接失败",
  "description": "应用报错: connection pool exhausted, 当前连接数: 500, 最大连接数: 500",
  "source": "monitoring"
}
```

**2. Skill Orchestrator接收请求**

```go
func HandleTicketCreated(ticket *Ticket) {
    // 执行分诊流水线
    result, err := orchestrator.ExecutePipeline(ctx, triagePipeline, ticket)

    // 更新工单
    ticket.Category = result.Category
    ticket.Priority = result.Priority
    ticket.AssigneeTeam = result.Team
    ticket.AIConfidence = result.Confidence
    ticket.AIExplanation = result.Reason

    // 保存结果
    db.Save(ticket)

    // 触发通知
    if result.Priority == "P0" {
        notifier.NotifyUrgent(ticket)
    }
}
```

**3. TriageSkill执行Guidance程序**

```python
program = '''
分析以下IT工单，返回结构化分类结果：

标题：{ticket.title}
描述：{ticket.description}

# 使用select约束输出
类型：{select([
    "incident - 系统故障或异常",
    "request - 服务申请或一般请求",
    "problem - 需要根因分析的复杂问题",
    "change - 配置变更或发布",
    "security - 安全相关事件"
], name="category")}

优先级：{select(["P0", "P1", "P2", "P3"], name="priority")}

推荐团队：{select([
    "infrastructure - 基础架构",
    "dba - 数据库",
    "security - 安全",
    "network - 网络",
    "development - 开发",
    "helpdesk - 一线支持"
], name="team")}

置信度：{gen(regex="0\\.\\d{2}", name="confidence")}

分析理由：{gen(max_tokens=100, name="reason")}
'''
```

**4. Evaluator评估结果质量**

```go
func (e *TriageEvaluator) Evaluate(input *Ticket, result *TriageResult) {
    // 更新统计
    e.total.Inc()
    e.latencies.Observe(float64(result.LatencyMs))

    // 检查是否需要人工复核
    if result.Confidence < e.confidenceThreshold {
        e.lowConfidenceCases.Inc()
        metrics.NotifyManualReview(input, result)
    }

    // 如果有ground truth（人工修正后的结果），更新准确率
    if input.ManualCategory != "" {
        if input.ManualCategory == result.Category {
            e.correct.Inc()
        } else {
            e.incorrect.Inc()
            e.recordBadCase(input, result)
        }
    }
}
```

**5. Planner规划优化方向**

当Evaluator发现系统性问题时，Planner自动规划改进：

```go
// 每周Planner分析报告
func (planner *TriagePlanner) WeeklyAnalysis() *OptimizationReport {
    report := evaluator.GetWeeklyReport()

    if report.Accuracy < 0.95 {
        // 准确率不达标，分析原因
        patterns := analyzer.FindErrorPatterns(report.BadCases)

        for pattern, count := range patterns {
            if count > 10 {
                // 这个模式的问题超过10次，需要优化
                plan := planner.CreateOptimizationPlan(pattern)
                logs.Info("建议优化", zap.Any("plan", plan))
            }
        }
    }

    return report
}
```

---

## 第五部分：Skill扩展：不止于分类

### 5.1 已有Skill一览

当前AI-Native ITSM已实现的Skill：

| Skill名称 | 功能 | Guidance程序 |
|:---|:---|:---|
| TriageSkill | 工单智能分类 | select(category, priority, team) |
| SummarizeSkill | 工单/事件摘要 | gen(summary, max_tokens=200) |
| KBSkill | RAG知识库问答 | gen(answer) + retrieve(context) |
| PrioritySkill | 优先级预测 | select(P0-P3) + reason |
| TeamMatchSkill | 团队匹配 | select(team) + score |

### 5.2 待扩展Skill

基于ITSM业务场景，建议扩展：

| Skill名称 | 场景价值 | 实现难度 |
|:---|:---|:---|
| SecurityTriageSkill | 安全事件自动识别和升级 | 低 |
| ImpactAnalysisSkill | 变更影响范围分析 | 中 |
| CostAnalysisSkill | IT服务成本统计 | 中 |
| SLAForecastSkill | SLA达成率预测 | 高 |
| AutoResolveSkill | 常见问题自动解决 | 高 |
| RootCauseSkill | 根因分析辅助 | 中 |

### 5.3 SecurityTriageSkill示例

```go
// SecurityTriageSkill：安全事件专项分类
type SecurityTriageSkill struct {
    baseSkill *BaseSkill
}

func (s *SecurityTriageSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    ticket := input.(*Ticket)

    // 特殊安全判断
    securityProgram := fmt.Sprintf(`
你是安全事件分诊专家。分析以下工单，判断是否为安全事件：

标题：%s
描述：%s

{select([
    "TRUE_SECURITY - 确认是安全事件，如数据泄露、漏洞利用、入侵检测",
    "POTENTIAL_SECURITY - 可能是安全问题，需要安全团队确认",
    "FALSE_ALARM - 不是安全问题，常规IT事件"
], name="is_security")}

安全评级：{select([
    "CRITICAL - 数据泄露、勒索软件、0day漏洞",
    "HIGH - 入侵尝试、权限提升、钓鱼攻击",
    "MEDIUM - 可疑行为、配置错误导致的潜在风险",
    "LOW - 一般安全建议或例行检查"
], name="severity")}

是否需要立即升级：{select(["YES", "NO"], name="escalate")}

安全团队处理建议：{gen(max_tokens=100, name="recommendation")}
`, ticket.Title, ticket.Description)

    output, err := s.llmClient.Execute(ctx, securityProgram)
    if err != nil {
        return nil, err
    }

    return parseSecurityResult(output)
}
```

---

## 第六部分：实战效果与数据

### 6.1 真实部署数据

在一家300人IT团队的制造企业部署了Guidance-Harness-Skill架构后：

| 指标 | 部署前 | 部署后 | 提升 |
|:---|:---:|:---:|:---:|
| 工单分类准确率 | 人工85% | AI 97.2% | +12.2% |
| 平均分类时间 | 3.5分钟 | 0.8秒 | 99.6% |
| P0故障响应时间 | 12分钟 | 2分钟 | 83% |
| 知识库使用率 | 8% | 45% | 5.6倍 |
| 工程师满意度 | 3.2/5 | 4.3/5 | +34% |

### 6.2 技术指标

| 指标 | 数值 |
|:---|:---|
| Guidance程序执行成功率 | 99.7% |
| 平均Token消耗/工单 | ~150 tokens |
| P99延迟 | < 500ms |
| 回归测试覆盖率 | 100% |
| Prompt版本数 | 12个（持续迭代） |

### 6.3 Bad Case改进闭环

系统运行3个月来，已积累并改进的Bad Case：

- 数据库类型误判为网络问题：增加3个专项示例
- 安全事件被判断为普通事件：加强SecurityTriageSkill
- 低置信度case过多：优化Prompt和阈值配置

每次改进都通过回归测试验证，避免引入新问题。

---

## 第七部分：工程实践建议

### 7.1 快速上手路径

**第一步：引入Guidance库**

```go
// 使用Go+Guidance的方式
// 参考Python版本实现Go绑定
import "github.com/guidance-lang/guidance-go"
```

**第二步：定义第一个Guidance程序**

从工单分类开始，用select约束输出格式。

**第三步：建立测试集**

收集100+真实工单案例，建立ground truth。

**第四步：实现Evaluator**

量化准确率、延迟、置信度分布。

**第五步：扩展Skill**

根据业务需求，添加新的Skill能力。

### 7.2 避坑指南

**Guidance程序不要太复杂**。一个程序最好只做一件事，多个Skill组合比单个复杂程序更易维护。

**置信度阈值要合理**。太低会漏掉错误case，太高会导致大量人工复核。建议从0.7开始，根据数据调整。

**测试集要持续更新**。每次发现Bad Case，都要加入测试集。测试集是AI改进的核心数据。

**降级策略要明确**。LLM不可用时，系统应该如何降级？建议保留关键词规则作为兜底。

### 7.3 运维监控

上线后需要监控：

```yaml
# 关键监控指标
metrics:
  - name: triage_accuracy
    type: gauge
    description: 分类准确率（每日更新）

  - name: low_confidence_rate
    type: gauge
    description: 低置信度case比例（应<15%）

  - name: avg_latency_ms
    type: gauge
    description: 平均分类延迟

  - name: fallback_rate
    type: gauge
    description: 降级到关键词的比率（应<5%）

  - name: llm_error_rate
    type: gauge
    description: LLM调用错误率

# 告警规则
alerts:
  - condition: low_confidence_rate > 0.20
    severity: warning
    message: "低置信度case超过20%，请检查分类质量"

  - condition: llm_error_rate > 0.01
    severity: critical
    message: "LLM错误率超过1%，可能是服务异常"
```

---

## 总结：AI-Native的正确打开方式

**Guidance** 解决了LLM"自由发挥"的问题，让输出可约束、可预期。

**Harness** 解决了AI"不可测、不可控"的问题，让AI质量可量化、可改进。

**Skill** 解决了AI"难以扩展"的问题，让新能力可以复用、可以组合。

三层架构的结合，让AI-Native ITSM从"AI是点缀"变成"AI是引擎"。

这不是一个理论架构，它已经在我们的生产环境中运行，每天处理数百个工单，持续改进。

---

**本系列其他文章**：

- 第1篇：《为什么你的IT团队每天像在救火？》
- 第2篇：《从混乱到有序：AI驱动的智能化工单分类实战》
- 第3篇：《变更管理不再是噩梦：BPMN工作流设计器详解》
- 第4篇：《让知识流动起来：RAG知识库搭建指南》
- 第5篇：《多租户架构：MSP服务商的高效运营之道》
- 第6篇：《30分钟搭建一套完整的ITSM系统》

**后续预告**：

- 第8篇：《开源一年，我们踩过的那些坑》
- 第9篇：《从0到1：AI-Native ITSM商业化之路》

如果你对Guidance-Harness-Skill架构有任何问题或想法，欢迎在评论区交流。下一篇文章将详细介绍如何用Python快速实现第一个Guidance程序，敬请期待。
