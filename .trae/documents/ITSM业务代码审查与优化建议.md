# ITSM 业务代码审查与优化建议

## 📋 执行摘要

本报告从业务角度深入审查 ITSM 系统的前后端代码，识别业务逻辑问题、性能瓶颈和用户体验改进点，提出针对性的优化建议。

### 审查范围
- ✅ 后端服务层（100+ 服务文件）
- ✅ 前端业务组件（50+ 组件）
- ✅ 业务流程完整性
- ✅ 数据一致性
- ✅ 用户体验

---

## 1. 核心业务模块审查

### 1.1 工单管理（Ticket Management）

#### 后端实现评估 ⭐⭐⭐⭐⭐

**优势**：
- ✅ 完整的生命周期管理（创建→分配→解决→关闭）
- ✅ 工作流集成（BPMN 触发）
- ✅ SLA 监控集成
- ✅ 自动化规则支持
- ✅ 批量操作支持
- ✅ MSP 多租户支持

**代码亮点**：
```go
// ticket_service.go - 服务依赖注入设计优秀
func (s *TicketService) SetNotificationService(notificationService *TicketNotificationService)
func (s *TicketService) SetAutomationRuleService(automationRuleService *TicketAutomationRuleService)
func (s *TicketService) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface)
```

**待优化项**：
1. **工单编号生成逻辑简单**
   ```go
   // 当前实现：TKT-YYYYMM-XXXXXX
   ticketNumber := fmt.Sprintf("TKT-%04d%02d-%06d", year, month, count+1)
   ```
   - 问题：跨月重置可能导致编号冲突
   - 建议：使用全局递增序列或雪花算法

2. **权限检查不够细粒度**
   ```go
   func (s *TicketService) canUpdateTicket(ticket *ent.Ticket, userID int) bool {
       // 简单的所有者检查
   }
   ```
   - 建议：集成 RBAC 权限系统，支持角色级别控制


#### 前端实现评估 ⭐⭐⭐⭐

**优势**：
- ✅ 组件化设计良好（TicketList, TicketDetail, TicketCard 等）
- ✅ 状态管理清晰
- ✅ 响应式布局
- ✅ 批量操作支持

**待优化项**：

1. **用户列表获取未实现**
   ```typescript
   // ticket-modal-service.ts
   export const fetchUserList = async (): Promise<any[]> => {
     // TODO: 替换为实际的 API 调用
     return [/* mock data */];
   }
   ```
   - 影响：无法正确分配工单处理人
   - 建议：对接 UserApi.getUsers()

2. **模板功能未完成**
   ```typescript
   export const fetchTicketTemplates = async (): Promise<any[]> => {
     // TODO: 替换为实际的 API 调用
     return [];
   }
   ```
   - 影响：无法使用工单模板快速创建
   - 建议：对接 TemplateApi.getTemplates()

3. **子任务处理人选择器为空**
   ```tsx
   <Select placeholder="请选择处理人" allowClear>
     {/* TODO: 从用户列表获取 */}
   </Select>
   ```
   - 影响：无法为子任务分配处理人
   - 建议：复用用户列表 API

**优化建议**：

```typescript
// 统一的用户选择器组件
export const UserSelector: React.FC<UserSelectorProps> = ({ 
  value, 
  onChange, 
  placeholder = "请选择用户" 
}) => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetchUsers = async () => {
      setLoading(true);
      try {
        const response = await UserApi.getUsers();
        setUsers(response.users);
      } catch (error) {
        message.error('获取用户列表失败');
      } finally {
        setLoading(false);
      }
    };
    fetchUsers();
  }, []);

  return (
    <Select
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      loading={loading}
      showSearch
      filterOption={(input, option) =>
        (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
      }
      options={users.map(user => ({
        label: user.name,
        value: user.id,
      }))}
    />
  );
};
```


---

### 1.2 事件管理（Incident Management）

#### 后端实现评估 ⭐⭐⭐⭐

**优势**：
- ✅ 完整的事件生命周期
- ✅ 事件规则引擎（自动化处理）
- ✅ 事件升级机制
- ✅ 监控和告警集成
- ✅ 工作流触发

**待完善功能**：

1. **SLA 违规升级未实现**
   ```go
   // incident_escalation_service.go
   case "sla_breach":
       // 基于SLA违规升级
       // TODO: 检查SLA状态
       return false, nil
   ```
   - 影响：无法基于 SLA 违规自动升级事件
   - 建议：集成 SLA 监控服务

2. **升级通知未实现**
   ```go
   func (s *IncidentEscalationService) sendEscalationNotification(...) {
       // TODO: 实现通知发送逻辑
       // 可以调用 notification service 发送邮件/短信/站内信
   }
   ```
   - 影响：事件升级后相关人员无法及时收到通知
   - 建议：集成通知服务

**优化建议**：

```go
// 完善 SLA 违规升级逻辑
func (s *IncidentEscalationService) shouldEscalate(ctx context.Context, rule *ent.IncidentEscalationRule, incident *ent.Incident) (bool, error) {
    switch rule.TriggerType {
    case "sla_breach":
        // 检查 SLA 状态
        slaInfo, err := s.slaService.GetIncidentSLAInfo(ctx, incident.ID, incident.TenantID)
        if err != nil {
            return false, err
        }
        
        // 判断是否违规
        if slaInfo.ResponseViolated || slaInfo.ResolutionViolated {
            s.logger.Infow("SLA breach detected, escalating incident",
                "incident_id", incident.ID,
                "response_violated", slaInfo.ResponseViolated,
                "resolution_violated", slaInfo.ResolutionViolated,
            )
            return true, nil
        }
        return false, nil
    // ... 其他情况
    }
}

// 完善升级通知
func (s *IncidentEscalationService) sendEscalationNotification(ctx context.Context, incident *ent.Incident, rule *ent.IncidentEscalationRule) {
    notification := &dto.NotificationRequest{
        Type:       "incident_escalation",
        Title:      fmt.Sprintf("事件升级通知: %s", incident.Title),
        Content:    fmt.Sprintf("事件 %s 已根据规则 %s 升级", incident.IncidentNumber, rule.Name),
        Recipients: []int{incident.AssigneeID, rule.EscalateToUserID},
        Priority:   "high",
        Metadata: map[string]interface{}{
            "incident_id": incident.ID,
            "rule_id":     rule.ID,
        },
    }
    
    if err := s.notificationService.SendNotification(ctx, notification); err != nil {
        s.logger.Errorw("Failed to send escalation notification", "error", err)
    }
}
```


---

### 1.3 问题管理（Problem Management）

#### 后端实现评估 ⭐⭐⭐⭐

**优势**：
- ✅ 完整的问题管理流程
- ✅ 工作流集成
- ✅ 统计数据支持

**待完善功能**：

1. **根因分析 AI 集成未完成**
   ```go
   // root_cause_analysis_service.go
   // 5 Whys 分析逻辑
   // 这里可以集成AI来辅助分析
   // TODO: 集成AI分析服务
   ```
   - 影响：无法利用 AI 进行智能根因分析
   - 建议：集成 LLM Gateway 进行根因推理

**优化建议**：

```go
// 集成 AI 根因分析
func (s *RootCauseAnalysisService) AnalyzeWithAI(ctx context.Context, problemID int, tenantID int) (*ent.RootCauseAnalysis, error) {
    // 获取问题详情
    problem, err := s.client.Problem.Query().
        Where(problem.ID(problemID), problem.TenantID(tenantID)).
        WithIncidents().
        Only(ctx)
    if err != nil {
        return nil, err
    }

    // 构建 AI 分析提示词
    prompt := fmt.Sprintf(`
作为 ITSM 专家，请分析以下问题的根本原因：

问题描述：%s
相关事件：%d 个
症状：%s

请使用 5 Whys 方法进行分析，并提供：
1. 根本原因
2. 分析过程
3. 预防措施建议
`, problem.Description, len(problem.Edges.Incidents), problem.Symptoms)

    // 调用 LLM Gateway
    response, err := s.llmGateway.Complete(ctx, &dto.LLMRequest{
        Model:       "gpt-4",
        Messages:    []dto.Message{{Role: "user", Content: prompt}},
        Temperature: 0.3,
        MaxTokens:   2000,
    })
    if err != nil {
        s.logger.Errorw("AI analysis failed", "error", err)
        return nil, err
    }

    // 创建根因分析记录
    rca, err := s.client.RootCauseAnalysis.Create().
        SetProblemID(problemID).
        SetTenantID(tenantID).
        SetAnalysisMethod("ai_5whys").
        SetRootCause(extractRootCause(response.Content)).
        SetAnalysisDetails(response.Content).
        SetRecommendations(extractRecommendations(response.Content)).
        SetAnalyzedBy("AI Assistant").
        Save(ctx)
    
    return rca, err
}
```


---

### 1.4 SLA 管理

#### 后端实现评估 ⭐⭐⭐⭐

**优势**：
- ✅ SLA 策略定义
- ✅ SLA 监控
- ✅ 违规记录

**待完善功能**：

1. **业务时间计算简化**
   ```go
   // sla_policy_service.go
   // 简化实现：暂不处理复杂的业务时间计算
   // TODO: 实现完整的业务时间计算逻辑
   return startTime.Add(time.Duration(policy.ResolutionTimeMinutes) * time.Minute)
   ```
   - 影响：无法正确处理工作日、节假日、工作时间
   - 建议：实现业务日历和工作时间计算

**优化建议**：

```go
// 业务时间计算器
type BusinessTimeCalculator struct {
    workingHours map[time.Weekday][]TimeRange // 工作时间
    holidays     map[string]bool               // 节假日
}

type TimeRange struct {
    Start time.Time
    End   time.Time
}

func (c *BusinessTimeCalculator) AddBusinessMinutes(start time.Time, minutes int) time.Time {
    current := start
    remainingMinutes := minutes

    for remainingMinutes > 0 {
        // 跳过非工作日
        if c.isHoliday(current) || !c.isWorkingDay(current) {
            current = c.nextWorkingDay(current)
            continue
        }

        // 获取当天工作时间
        workingRanges := c.workingHours[current.Weekday()]
        for _, timeRange := range workingRanges {
            // 如果当前时间在工作时间内
            if c.isInRange(current, timeRange) {
                // 计算到工作时间结束还有多少分钟
                minutesToEnd := int(timeRange.End.Sub(current).Minutes())
                
                if remainingMinutes <= minutesToEnd {
                    // 剩余时间在当前工作时间段内
                    return current.Add(time.Duration(remainingMinutes) * time.Minute)
                }
                
                // 消耗当前工作时间段
                remainingMinutes -= minutesToEnd
                current = timeRange.End
            }
        }
        
        // 移动到下一个工作日
        current = c.nextWorkingDay(current)
    }

    return current
}

func (c *BusinessTimeCalculator) isHoliday(t time.Time) bool {
    dateStr := t.Format("2006-01-02")
    return c.holidays[dateStr]
}

func (c *BusinessTimeCalculator) isWorkingDay(t time.Time) bool {
    weekday := t.Weekday()
    return weekday != time.Saturday && weekday != time.Sunday
}

func (c *BusinessTimeCalculator) nextWorkingDay(t time.Time) time.Time {
    next := t.Add(24 * time.Hour)
    for !c.isWorkingDay(next) || c.isHoliday(next) {
        next = next.Add(24 * time.Hour)
    }
    
    // 设置为第一个工作时间段的开始时间
    workingRanges := c.workingHours[next.Weekday()]
    if len(workingRanges) > 0 {
        return time.Date(next.Year(), next.Month(), next.Day(),
            workingRanges[0].Start.Hour(), workingRanges[0].Start.Minute(), 0, 0, next.Location())
    }
    
    return next
}

// 使用示例
func (s *SLAPolicyService) CalculateResolutionDeadline(startTime time.Time, policy *ent.SLAPolicy) time.Time {
    calculator := &BusinessTimeCalculator{
        workingHours: map[time.Weekday][]TimeRange{
            time.Monday:    {{Start: time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC), End: time.Date(0, 0, 0, 18, 0, 0, 0, time.UTC)}},
            time.Tuesday:   {{Start: time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC), End: time.Date(0, 0, 0, 18, 0, 0, 0, time.UTC)}},
            time.Wednesday: {{Start: time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC), End: time.Date(0, 0, 0, 18, 0, 0, 0, time.UTC)}},
            time.Thursday:  {{Start: time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC), End: time.Date(0, 0, 0, 18, 0, 0, 0, time.UTC)}},
            time.Friday:    {{Start: time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC), End: time.Date(0, 0, 0, 18, 0, 0, 0, time.UTC)}},
        },
        holidays: s.loadHolidays(), // 从配置或数据库加载节假日
    }
    
    return calculator.AddBusinessMinutes(startTime, policy.ResolutionTimeMinutes)
}
```


---

### 1.5 知识库管理

#### 前端实现评估 ⭐⭐⭐⭐

**待完善功能**：

1. **实时协作未实现**
   ```typescript
   // KnowledgeCollaboration.tsx
   // TODO: 对接实时协作Session API
   // 目前后端暂未提供获取当前文档协作Session的接口
   setSession(null);
   ```
   - 影响：无法实现多人实时协作编辑
   - 建议：使用 WebSocket 实现实时协作

**优化建议**：

```typescript
// 实时协作服务
export class CollaborationService {
  private ws: WebSocket | null = null;
  private listeners: Map<string, Set<Function>> = new Map();

  connect(articleId: number) {
    const wsUrl = `${process.env.NEXT_PUBLIC_WS_URL}/collaboration/${articleId}`;
    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log('Collaboration session connected');
      this.emit('connected', { articleId });
    };

    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      this.emit(data.type, data.payload);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.emit('error', error);
    };

    this.ws.onclose = () => {
      console.log('Collaboration session closed');
      this.emit('disconnected', {});
    };
  }

  sendUpdate(update: any) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'update',
        payload: update,
      }));
    }
  }

  on(event: string, callback: Function) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback);
  }

  off(event: string, callback: Function) {
    this.listeners.get(event)?.delete(callback);
  }

  private emit(event: string, data: any) {
    this.listeners.get(event)?.forEach(callback => callback(data));
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// 使用示例
const KnowledgeCollaboration: React.FC = ({ articleId }) => {
  const [session, setSession] = useState<CollaborationSession | null>(null);
  const [activeUsers, setActiveUsers] = useState<User[]>([]);
  const collaborationService = useRef(new CollaborationService());

  useEffect(() => {
    const service = collaborationService.current;
    
    service.on('connected', () => {
      setSession({ id: articleId, status: 'active' });
    });

    service.on('user_joined', (user: User) => {
      setActiveUsers(prev => [...prev, user]);
      message.info(`${user.name} 加入了协作`);
    });

    service.on('user_left', (user: User) => {
      setActiveUsers(prev => prev.filter(u => u.id !== user.id));
      message.info(`${user.name} 离开了协作`);
    });

    service.on('content_update', (update: any) => {
      // 处理内容更新
      handleRemoteUpdate(update);
    });

    service.connect(articleId);

    return () => {
      service.disconnect();
    };
  }, [articleId]);

  return (
    <div>
      {/* 协作状态显示 */}
      <div className="flex items-center gap-2">
        <Badge status={session ? 'processing' : 'default'} />
        <span>{activeUsers.length} 人正在协作</span>
        <Avatar.Group maxCount={5}>
          {activeUsers.map(user => (
            <Tooltip key={user.id} title={user.name}>
              <Avatar src={user.avatar}>{user.name[0]}</Avatar>
            </Tooltip>
          ))}
        </Avatar.Group>
      </div>
    </div>
  );
};
```


---

## 2. 跨模块业务问题

### 2.1 权限管理不统一

**问题**：
- 前端权限配置使用 localStorage（临时方案）
- 后端权限检查分散在各个服务中
- 缺少统一的权限管理中心

```typescript
// admin/permissions/page.tsx
// TODO: 实现真实的权限保存 API
// 临时方案：将权限配置保存到 localStorage
const configToSave = { /* ... */ };
localStorage.setItem('permissions', JSON.stringify(configToSave));
```

**优化建议**：

```go
// 后端：统一权限服务
type PermissionService struct {
    client *ent.Client
    cache  *redis.Client
    logger *zap.SugaredLogger
}

// 检查用户权限
func (s *PermissionService) CheckPermission(ctx context.Context, userID int, resource string, action string) (bool, error) {
    // 1. 从缓存获取
    cacheKey := fmt.Sprintf("perm:%d:%s:%s", userID, resource, action)
    if cached, err := s.cache.Get(ctx, cacheKey).Bool(); err == nil {
        return cached, nil
    }

    // 2. 查询数据库
    user, err := s.client.User.Query().
        Where(user.ID(userID)).
        WithRoles(func(q *ent.RoleQuery) {
            q.WithPermissions()
        }).
        Only(ctx)
    if err != nil {
        return false, err
    }

    // 3. 检查权限
    hasPermission := false
    for _, role := range user.Edges.Roles {
        for _, perm := range role.Edges.Permissions {
            if perm.Resource == resource && perm.Action == action {
                hasPermission = true
                break
            }
        }
    }

    // 4. 缓存结果（5分钟）
    s.cache.Set(ctx, cacheKey, hasPermission, 5*time.Minute)

    return hasPermission, nil
}

// 权限中间件
func (s *PermissionService) RequirePermission(resource, action string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetInt("user_id")
        
        hasPermission, err := s.CheckPermission(c.Request.Context(), userID, resource, action)
        if err != nil {
            c.JSON(500, gin.H{"error": "Permission check failed"})
            c.Abort()
            return
        }

        if !hasPermission {
            c.JSON(403, gin.H{"error": "Permission denied"})
            c.Abort()
            return
        }

        c.Next()
    }
}
```

```typescript
// 前端：权限 Hook
export const usePermission = () => {
  const { user } = useAuthStore();
  const [permissions, setPermissions] = useState<Permission[]>([]);

  useEffect(() => {
    const fetchPermissions = async () => {
      try {
        const response = await PermissionApi.getUserPermissions(user.id);
        setPermissions(response.permissions);
      } catch (error) {
        console.error('Failed to fetch permissions:', error);
      }
    };
    fetchPermissions();
  }, [user.id]);

  const hasPermission = useCallback((resource: string, action: string) => {
    return permissions.some(
      p => p.resource === resource && p.action === action
    );
  }, [permissions]);

  return { hasPermission, permissions };
};

// 使用示例
const TicketList: React.FC = () => {
  const { hasPermission } = usePermission();

  return (
    <div>
      {hasPermission('ticket', 'create') && (
        <Button onClick={handleCreate}>创建工单</Button>
      )}
      {hasPermission('ticket', 'delete') && (
        <Button onClick={handleDelete}>删除工单</Button>
      )}
    </div>
  );
};
```


---

### 2.2 通知系统不完整

**问题**：
- 多处通知发送逻辑未实现
- 缺少统一的通知模板
- 没有通知偏好设置

**优化建议**：

```go
// 统一通知服务
type UnifiedNotificationService struct {
    client        *ent.Client
    emailService  *EmailService
    smsService    *SMSService
    websocketService *WebSocketService
    logger        *zap.SugaredLogger
}

type NotificationRequest struct {
    Type       string                 // 通知类型
    Title      string                 // 标题
    Content    string                 // 内容
    Recipients []int                  // 接收人ID列表
    Priority   string                 // 优先级
    Channels   []string               // 通知渠道：email, sms, websocket, in_app
    Metadata   map[string]interface{} // 元数据
}

func (s *UnifiedNotificationService) SendNotification(ctx context.Context, req *NotificationRequest) error {
    // 1. 获取接收人偏好设置
    preferences, err := s.getRecipientPreferences(ctx, req.Recipients)
    if err != nil {
        return err
    }

    // 2. 根据偏好发送通知
    for _, recipientID := range req.Recipients {
        pref := preferences[recipientID]
        
        // 检查用户是否启用了该类型通知
        if !pref.IsEnabled(req.Type) {
            continue
        }

        // 根据用户偏好的渠道发送
        channels := req.Channels
        if len(pref.Channels) > 0 {
            channels = pref.Channels
        }

        for _, channel := range channels {
            switch channel {
            case "email":
                if pref.EmailEnabled {
                    go s.sendEmail(ctx, recipientID, req)
                }
            case "sms":
                if pref.SMSEnabled && req.Priority == "high" {
                    go s.sendSMS(ctx, recipientID, req)
                }
            case "websocket":
                go s.sendWebSocket(ctx, recipientID, req)
            case "in_app":
                go s.createInAppNotification(ctx, recipientID, req)
            }
        }
    }

    return nil
}

func (s *UnifiedNotificationService) sendEmail(ctx context.Context, recipientID int, req *NotificationRequest) {
    user, err := s.client.User.Get(ctx, recipientID)
    if err != nil {
        s.logger.Errorw("Failed to get user", "error", err)
        return
    }

    // 使用模板渲染邮件
    template := s.getEmailTemplate(req.Type)
    body := template.Render(req)

    if err := s.emailService.Send(ctx, &EmailRequest{
        To:      user.Email,
        Subject: req.Title,
        Body:    body,
    }); err != nil {
        s.logger.Errorw("Failed to send email", "error", err)
    }
}

func (s *UnifiedNotificationService) createInAppNotification(ctx context.Context, recipientID int, req *NotificationRequest) {
    _, err := s.client.Notification.Create().
        SetUserID(recipientID).
        SetType(req.Type).
        SetTitle(req.Title).
        SetContent(req.Content).
        SetPriority(req.Priority).
        SetMetadata(req.Metadata).
        SetRead(false).
        Save(ctx)
    
    if err != nil {
        s.logger.Errorw("Failed to create notification", "error", err)
    }
}
```


---

### 2.3 数据一致性问题

**问题**：
- 工单状态与工作流状态可能不同步
- SLA 计算可能与实际业务时间不符
- 批量操作缺少事务保护

**优化建议**：

```go
// 1. 工单状态同步机制
func (s *TicketService) SyncTicketStatusWithWorkflow(ctx context.Context, ticketID int, tenantID int) error {
    // 使用事务确保一致性
    tx, err := s.client.Tx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 获取工单
    ticket, err := tx.Ticket.Query().
        Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
        ForUpdate(). // 行锁
        Only(ctx)
    if err != nil {
        return err
    }

    // 获取工作流状态
    workflowStatus, err := s.processTriggerService.GetProcessStatus(ctx, ticketID, tenantID)
    if err != nil {
        return err
    }

    // 映射工作流状态到工单状态
    newStatus := s.mapWorkflowStatusToTicketStatus(workflowStatus.Status)
    
    // 更新工单状态
    if ticket.Status != newStatus {
        _, err = tx.Ticket.UpdateOne(ticket).
            SetStatus(newStatus).
            SetUpdatedAt(time.Now()).
            Save(ctx)
        if err != nil {
            return err
        }

        // 记录状态变更日志
        _, err = tx.TicketStatusLog.Create().
            SetTicketID(ticketID).
            SetOldStatus(ticket.Status).
            SetNewStatus(newStatus).
            SetChangedBy("system").
            SetReason("Synced with workflow").
            Save(ctx)
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}

// 2. 批量操作事务保护
func (s *TicketService) BatchUpdateTickets(ctx context.Context, ticketIDs []int, updates map[string]interface{}, tenantID int) error {
    // 使用事务
    tx, err := s.client.Tx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 批量更新
    updateQuery := tx.Ticket.Update().
        Where(ticket.IDIn(ticketIDs...), ticket.TenantID(tenantID))

    // 应用更新
    for key, value := range updates {
        switch key {
        case "status":
            updateQuery = updateQuery.SetStatus(value.(string))
        case "priority":
            updateQuery = updateQuery.SetPriority(value.(string))
        case "assignee_id":
            updateQuery = updateQuery.SetAssigneeID(value.(int))
        }
    }

    affected, err := updateQuery.Save(ctx)
    if err != nil {
        return err
    }

    s.logger.Infow("Batch update completed", 
        "affected", affected,
        "ticket_ids", ticketIDs,
    )

    // 触发后续操作（通知、SLA更新等）
    for _, ticketID := range ticketIDs {
        go s.handleTicketUpdate(context.Background(), ticketID, tenantID, updates)
    }

    return tx.Commit()
}

// 3. SLA 一致性检查
func (s *SLAMonitorService) RecalculateSLA(ctx context.Context, ticketID int, tenantID int) error {
    tx, err := s.client.Tx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    ticket, err := tx.Ticket.Query().
        Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
        WithSLAPolicy().
        ForUpdate().
        Only(ctx)
    if err != nil {
        return err
    }

    if ticket.Edges.SLAPolicy == nil {
        return nil // 无SLA策略
    }

    // 重新计算截止时间（考虑业务时间）
    calculator := NewBusinessTimeCalculator(s.client, tenantID)
    responseDeadline := calculator.AddBusinessMinutes(
        ticket.CreatedAt,
        ticket.Edges.SLAPolicy.ResponseTimeMinutes,
    )
    resolutionDeadline := calculator.AddBusinessMinutes(
        ticket.CreatedAt,
        ticket.Edges.SLAPolicy.ResolutionTimeMinutes,
    )

    // 更新工单
    _, err = tx.Ticket.UpdateOne(ticket).
        SetSLAResponseDeadline(responseDeadline).
        SetSLAResolutionDeadline(resolutionDeadline).
        Save(ctx)
    if err != nil {
        return err
    }

    // 检查是否违规
    now := time.Now()
    if ticket.FirstResponseAt == nil && now.After(responseDeadline) {
        // 响应违规
        s.createSLAViolation(ctx, tx, ticketID, "response", responseDeadline)
    }
    if ticket.ResolvedAt == nil && now.After(resolutionDeadline) {
        // 解决违规
        s.createSLAViolation(ctx, tx, ticketID, "resolution", resolutionDeadline)
    }

    return tx.Commit()
}
```


---

## 3. 性能优化建议

### 3.1 数据库查询优化

**问题**：
- N+1 查询问题
- 缺少索引
- 大数据量分页性能差

**优化建议**：

```go
// 1. 使用 Eager Loading 避免 N+1 查询
func (s *TicketService) ListTicketsOptimized(ctx context.Context, req *dto.ListTicketsRequest, tenantID int) (*dto.ListTicketsResponse, error) {
    query := s.client.Ticket.Query().
        Where(ticket.TenantID(tenantID)).
        // Eager loading 关联数据
        WithAssignee().
        WithRequester().
        WithCategory().
        WithTags().
        WithSLAPolicy().
        // 只查询需要的字段
        Select(
            ticket.FieldID,
            ticket.FieldTicketNumber,
            ticket.FieldTitle,
            ticket.FieldStatus,
            ticket.FieldPriority,
            ticket.FieldCreatedAt,
            ticket.FieldUpdatedAt,
        )

    // 应用筛选
    if req.Status != nil {
        query = query.Where(ticket.StatusEQ(*req.Status))
    }
    if req.Priority != nil {
        query = query.Where(ticket.PriorityEQ(*req.Priority))
    }

    // 计数查询（使用索引）
    total, err := query.Count(ctx)
    if err != nil {
        return nil, err
    }

    // 分页查询
    tickets, err := query.
        Offset((req.Page - 1) * req.PageSize).
        Limit(req.PageSize).
        Order(ent.Desc(ticket.FieldCreatedAt)).
        All(ctx)
    if err != nil {
        return nil, err
    }

    return &dto.ListTicketsResponse{
        Tickets: tickets,
        Total:   total,
        Page:    req.Page,
        PageSize: req.PageSize,
    }, nil
}

// 2. 添加数据库索引
// 在 schema 中定义索引
func (Ticket) Indexes() []ent.Index {
    return []ent.Index{
        // 复合索引：租户ID + 状态 + 创建时间
        index.Fields("tenant_id", "status", "created_at"),
        // 复合索引：租户ID + 处理人ID + 状态
        index.Fields("tenant_id", "assignee_id", "status"),
        // 复合索引：租户ID + 优先级 + 创建时间
        index.Fields("tenant_id", "priority", "created_at"),
        // 工单号唯一索引
        index.Fields("ticket_number").Unique(),
        // 全文搜索索引（PostgreSQL）
        index.Fields("title", "description").
            Annotations(entsql.IndexType("GIN")),
    }
}

// 3. 使用游标分页优化大数据量查询
func (s *TicketService) ListTicketsCursor(ctx context.Context, cursor string, limit int, tenantID int) (*dto.CursorPaginationResponse, error) {
    query := s.client.Ticket.Query().
        Where(ticket.TenantID(tenantID)).
        Limit(limit + 1) // 多查一条判断是否有下一页

    // 如果有游标，从游标位置开始查询
    if cursor != "" {
        cursorID, err := decodeCursor(cursor)
        if err != nil {
            return nil, err
        }
        query = query.Where(ticket.IDLT(cursorID))
    }

    tickets, err := query.
        Order(ent.Desc(ticket.FieldID)).
        All(ctx)
    if err != nil {
        return nil, err
    }

    hasMore := len(tickets) > limit
    if hasMore {
        tickets = tickets[:limit]
    }

    var nextCursor string
    if hasMore && len(tickets) > 0 {
        nextCursor = encodeCursor(tickets[len(tickets)-1].ID)
    }

    return &dto.CursorPaginationResponse{
        Data:       tickets,
        NextCursor: nextCursor,
        HasMore:    hasMore,
    }, nil
}
```


---

## 4. 各业务模块深度审查

### 4.1 CMDB（配置管理数据库）

#### 后端实现评估 ⭐⭐⭐

**优势**：
- ✅ 基础 CRUD 功能完整
- ✅ CI 关系管理
- ✅ 拓扑查询支持

**待完善功能**：

1. **缺少属性定义管理**
   ```go
   // 当前实现只有基础字段，缺少动态属性管理
   type CreateCIRequest struct {
       Name       string
       CiType     string
       Attributes *map[string]interface{} // 仅支持自由格式
   }
   ```
   - 问题：无法定义 CI 类型的属性模板
   - 建议：添加 CIAttributeDefinition 管理

2. **缺少变更历史追踪**
   - 问题：无法追踪 CI 的变更历史
   - 建议：添加 CIChangeHistory 表

3. **缺少自动发现功能**
   - 问题：需要手动录入 CI
   - 建议：集成自动发现服务

**优化建议**：

```go
// CI 属性定义服务
type CIAttributeDefinitionService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 创建属性定义
func (s *CIAttributeDefinitionService) CreateAttributeDefinition(ctx context.Context, req *CreateAttributeDefinitionRequest) (*ent.CIAttributeDefinition, error) {
    return s.client.CIAttributeDefinition.Create().
        SetCiTypeID(req.CITypeID).
        SetName(req.Name).
        SetDisplayName(req.DisplayName).
        SetDataType(req.DataType). // string, number, boolean, date, enum
        SetRequired(req.Required).
        SetDefaultValue(req.DefaultValue).
        SetValidationRules(req.ValidationRules).
        SetOptions(req.Options). // 用于枚举类型
        SetTenantID(req.TenantID).
        Save(ctx)
}

// 验证 CI 属性
func (s *CMDBService) ValidateCI(ctx context.Context, ciTypeID int, attributes map[string]interface{}) error {
    // 获取 CI 类型的属性定义
    definitions, err := s.client.CIAttributeDefinition.Query().
        Where(ciattributedefinition.CiTypeID(ciTypeID)).
        All(ctx)
    if err != nil {
        return err
    }

    // 验证必填字段
    for _, def := range definitions {
        if def.Required {
            if _, exists := attributes[def.Name]; !exists {
                return fmt.Errorf("缺少必填属性: %s", def.DisplayName)
            }
        }

        // 验证数据类型
        if value, exists := attributes[def.Name]; exists {
            if err := s.validateAttributeValue(def, value); err != nil {
                return fmt.Errorf("属性 %s 验证失败: %w", def.DisplayName, err)
            }
        }
    }

    return nil
}

// CI 变更历史追踪
type CIChangeHistoryService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

func (s *CIChangeHistoryService) RecordChange(ctx context.Context, req *RecordCIChangeRequest) error {
    return s.client.CIChangeHistory.Create().
        SetCiID(req.CIID).
        SetChangeType(req.ChangeType). // created, updated, deleted
        SetChangedBy(req.ChangedBy).
        SetOldValue(req.OldValue).
        SetNewValue(req.NewValue).
        SetChangedFields(req.ChangedFields).
        SetReason(req.Reason).
        SetTenantID(req.TenantID).
        Exec(ctx)
}

// 自动发现服务
type CIDiscoveryService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

func (s *CIDiscoveryService) DiscoverFromCloud(ctx context.Context, provider string, credentials map[string]string, tenantID int) ([]*ent.ConfigurationItem, error) {
    var discoveredCIs []*ent.ConfigurationItem

    switch provider {
    case "aws":
        discoveredCIs, err = s.discoverFromAWS(ctx, credentials, tenantID)
    case "azure":
        discoveredCIs, err = s.discoverFromAzure(ctx, credentials, tenantID)
    case "aliyun":
        discoveredCIs, err = s.discoverFromAliyun(ctx, credentials, tenantID)
    default:
        return nil, fmt.Errorf("不支持的云服务商: %s", provider)
    }

    if err != nil {
        return nil, err
    }

    // 保存或更新 CI
    for _, ci := range discoveredCIs {
        s.saveOrUpdateCI(ctx, ci)
    }

    return discoveredCIs, nil
}
```


---

### 4.2 资产管理（Asset Management）

#### 后端实现评估 ⭐⭐⭐⭐

**优势**：
- ✅ 完整的资产生命周期管理
- ✅ 资产状态管理（可用、使用中、维护、退役、报废）
- ✅ 资产分配和退役功能
- ✅ 统计数据支持

**待完善功能**：

1. **缺少资产盘点功能**
   - 问题：无法进行定期资产盘点
   - 建议：添加资产盘点模块

2. **缺少资产折旧计算**
   - 问题：无法计算资产折旧
   - 建议：添加折旧计算服务

3. **缺少资产维护记录**
   - 问题：无法记录资产维护历史
   - 建议：添加维护记录表

**优化建议**：

```go
// 资产盘点服务
type AssetInventoryService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 创建盘点任务
func (s *AssetInventoryService) CreateInventoryTask(ctx context.Context, req *CreateInventoryTaskRequest) (*ent.AssetInventoryTask, error) {
    return s.client.AssetInventoryTask.Create().
        SetName(req.Name).
        SetDescription(req.Description).
        SetStartDate(req.StartDate).
        SetEndDate(req.EndDate).
        SetStatus("pending").
        SetCreatedBy(req.CreatedBy).
        SetTenantID(req.TenantID).
        Save(ctx)
}

// 记录盘点结果
func (s *AssetInventoryService) RecordInventoryResult(ctx context.Context, req *RecordInventoryResultRequest) error {
    // 查询资产
    asset, err := s.client.Asset.Get(ctx, req.AssetID)
    if err != nil {
        return err
    }

    // 创建盘点记录
    _, err = s.client.AssetInventoryRecord.Create().
        SetTaskID(req.TaskID).
        SetAssetID(req.AssetID).
        SetFoundStatus(req.FoundStatus). // found, missing, damaged
        SetActualLocation(req.ActualLocation).
        SetActualCondition(req.ActualCondition).
        SetNotes(req.Notes).
        SetCheckedBy(req.CheckedBy).
        SetCheckedAt(time.Now()).
        SetTenantID(req.TenantID).
        Save(ctx)

    // 如果资产丢失，更新资产状态
    if req.FoundStatus == "missing" {
        asset.Update().SetStatus("missing").Exec(ctx)
    }

    return err
}

// 资产折旧服务
type AssetDepreciationService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 计算资产折旧
func (s *AssetDepreciationService) CalculateDepreciation(ctx context.Context, assetID int) (*AssetDepreciationInfo, error) {
    asset, err := s.client.Asset.Get(ctx, assetID)
    if err != nil {
        return nil, err
    }

    if asset.PurchasePrice == nil {
        return nil, fmt.Errorf("资产未设置购买价格")
    }

    // 获取折旧策略（从系统配置或资产类型配置）
    depreciationYears := 5 // 默认5年
    depreciationMethod := "straight_line" // 直线折旧法

    // 计算已使用年限
    yearsUsed := time.Since(asset.PurchaseDate).Hours() / 24 / 365

    var currentValue float64
    var accumulatedDepreciation float64

    switch depreciationMethod {
    case "straight_line":
        // 直线折旧法
        annualDepreciation := *asset.PurchasePrice / float64(depreciationYears)
        accumulatedDepreciation = annualDepreciation * yearsUsed
        currentValue = *asset.PurchasePrice - accumulatedDepreciation
        if currentValue < 0 {
            currentValue = 0
        }
    case "declining_balance":
        // 余额递减法
        rate := 2.0 / float64(depreciationYears)
        currentValue = *asset.PurchasePrice * math.Pow(1-rate, yearsUsed)
        accumulatedDepreciation = *asset.PurchasePrice - currentValue
    }

    return &AssetDepreciationInfo{
        AssetID:                 assetID,
        PurchasePrice:           *asset.PurchasePrice,
        CurrentValue:            currentValue,
        AccumulatedDepreciation: accumulatedDepreciation,
        YearsUsed:               yearsUsed,
        DepreciationMethod:      depreciationMethod,
        CalculatedAt:            time.Now(),
    }, nil
}

// 资产维护服务
type AssetMaintenanceService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 创建维护记录
func (s *AssetMaintenanceService) CreateMaintenanceRecord(ctx context.Context, req *CreateMaintenanceRecordRequest) (*ent.AssetMaintenanceRecord, error) {
    // 更新资产状态为维护中
    asset, err := s.client.Asset.Get(ctx, req.AssetID)
    if err != nil {
        return nil, err
    }

    asset.Update().SetStatus("maintenance").Exec(ctx)

    // 创建维护记录
    return s.client.AssetMaintenanceRecord.Create().
        SetAssetID(req.AssetID).
        SetMaintenanceType(req.MaintenanceType). // preventive, corrective, emergency
        SetDescription(req.Description).
        SetScheduledDate(req.ScheduledDate).
        SetCompletedDate(req.CompletedDate).
        SetPerformedBy(req.PerformedBy).
        SetCost(req.Cost).
        SetNotes(req.Notes).
        SetStatus("scheduled").
        SetTenantID(req.TenantID).
        Save(ctx)
}

// 完成维护
func (s *AssetMaintenanceService) CompleteMaintenance(ctx context.Context, recordID int, notes string) error {
    record, err := s.client.AssetMaintenanceRecord.Get(ctx, recordID)
    if err != nil {
        return err
    }

    // 更新维护记录
    now := time.Now()
    _, err = record.Update().
        SetStatus("completed").
        SetCompletedDate(now).
        SetNotes(notes).
        Save(ctx)
    if err != nil {
        return err
    }

    // 恢复资产状态
    asset, err := s.client.Asset.Get(ctx, record.AssetID)
    if err != nil {
        return err
    }

    return asset.Update().SetStatus("available").Exec(ctx)
}
```


---

### 4.3 知识库管理（Knowledge Base）

#### 后端实现评估 ⭐⭐⭐⭐

**优势**：
- ✅ 完整的文章 CRUD
- ✅ 分类管理
- ✅ 点赞功能
- ✅ 搜索支持

**待完善功能**：

1. **缺少版本控制**
   - 问题：无法追踪文章修改历史
   - 建议：添加版本控制功能

2. **缺少审核流程**
   - 问题：文章发布无审核机制
   - 建议：添加审核工作流

3. **缺少访问统计**
   - 问题：无法统计文章阅读量
   - 建议：添加访问统计

**优化建议**：

```go
// 知识库版本控制服务
type KnowledgeVersionService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 创建文章版本
func (s *KnowledgeVersionService) CreateVersion(ctx context.Context, articleID int, content string, changedBy int) (*ent.KnowledgeArticleVersion, error) {
    // 获取当前文章
    article, err := s.client.KnowledgeArticle.Get(ctx, articleID)
    if err != nil {
        return nil, err
    }

    // 获取最新版本号
    latestVersion, err := s.client.KnowledgeArticleVersion.Query().
        Where(knowledgearticleversion.ArticleID(articleID)).
        Order(ent.Desc(knowledgearticleversion.FieldVersionNumber)).
        First(ctx)

    versionNumber := 1
    if latestVersion != nil {
        versionNumber = latestVersion.VersionNumber + 1
    }

    // 创建新版本
    return s.client.KnowledgeArticleVersion.Create().
        SetArticleID(articleID).
        SetVersionNumber(versionNumber).
        SetTitle(article.Title).
        SetContent(content).
        SetChangedBy(changedBy).
        SetChangeDescription("版本更新").
        SetTenantID(article.TenantID).
        Save(ctx)
}

// 恢复到指定版本
func (s *KnowledgeVersionService) RestoreVersion(ctx context.Context, articleID int, versionNumber int) error {
    // 获取指定版本
    version, err := s.client.KnowledgeArticleVersion.Query().
        Where(
            knowledgearticleversion.ArticleID(articleID),
            knowledgearticleversion.VersionNumber(versionNumber),
        ).
        Only(ctx)
    if err != nil {
        return err
    }

    // 更新文章内容
    return s.client.KnowledgeArticle.UpdateOneID(articleID).
        SetContent(version.Content).
        Exec(ctx)
}

// 知识库审核服务
type KnowledgeReviewService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 提交审核
func (s *KnowledgeReviewService) SubmitForReview(ctx context.Context, articleID int, submittedBy int) error {
    // 创建审核记录
    _, err := s.client.KnowledgeArticleReview.Create().
        SetArticleID(articleID).
        SetStatus("pending").
        SetSubmittedBy(submittedBy).
        SetSubmittedAt(time.Now()).
        Save(ctx)
    if err != nil {
        return err
    }

    // 更新文章状态
    return s.client.KnowledgeArticle.UpdateOneID(articleID).
        SetStatus("under_review").
        Exec(ctx)
}

// 审核文章
func (s *KnowledgeReviewService) ReviewArticle(ctx context.Context, reviewID int, approved bool, comments string, reviewedBy int) error {
    review, err := s.client.KnowledgeArticleReview.Get(ctx, reviewID)
    if err != nil {
        return err
    }

    // 更新审核记录
    status := "rejected"
    if approved {
        status = "approved"
    }

    _, err = review.Update().
        SetStatus(status).
        SetReviewedBy(reviewedBy).
        SetReviewedAt(time.Now()).
        SetComments(comments).
        Save(ctx)
    if err != nil {
        return err
    }

    // 更新文章状态
    articleStatus := "draft"
    if approved {
        articleStatus = "published"
    }

    return s.client.KnowledgeArticle.UpdateOneID(review.ArticleID).
        SetStatus(articleStatus).
        SetIsPublished(approved).
        Exec(ctx)
}

// 知识库统计服务
type KnowledgeAnalyticsService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 记录文章访问
func (s *KnowledgeAnalyticsService) RecordView(ctx context.Context, articleID int, userID int, tenantID int) error {
    // 创建访问记录
    _, err := s.client.KnowledgeArticleView.Create().
        SetArticleID(articleID).
        SetUserID(userID).
        SetViewedAt(time.Now()).
        SetTenantID(tenantID).
        Save(ctx)
    if err != nil {
        return err
    }

    // 增加文章浏览量
    return s.client.KnowledgeArticle.UpdateOneID(articleID).
        AddViewCount(1).
        Exec(ctx)
}

// 获取文章统计
func (s *KnowledgeAnalyticsService) GetArticleStats(ctx context.Context, articleID int, tenantID int) (*ArticleStats, error) {
    article, err := s.client.KnowledgeArticle.Get(ctx, articleID)
    if err != nil {
        return nil, err
    }

    // 统计独立访问用户数
    uniqueViewers, _ := s.client.KnowledgeArticleView.Query().
        Where(knowledgearticleview.ArticleID(articleID)).
        GroupBy(knowledgearticleview.FieldUserID).
        Count(ctx)

    // 统计评论数
    commentCount, _ := s.client.KnowledgeArticleComment.Query().
        Where(knowledgearticlecomment.ArticleID(articleID)).
        Count(ctx)

    return &ArticleStats{
        ArticleID:      articleID,
        ViewCount:      article.ViewCount,
        UniqueViewers:  uniqueViewers,
        LikeCount:      article.LikeCount,
        CommentCount:   commentCount,
        LastViewedAt:   time.Now(),
    }, nil
}
```


---

### 4.4 发布管理（Release Management）

#### 后端实现评估 ⭐⭐⭐⭐

**优势**：
- ✅ 完整的发布生命周期
- ✅ 发布状态管理
- ✅ 关联变更管理
- ✅ 回滚计划支持

**待完善功能**：

1. **缺少发布阶段管理**
   - 问题：无法管理发布的多个阶段（开发→测试→预发布→生产）
   - 建议：添加发布阶段管理

2. **缺少发布验证**
   - 问题：无法记录发布后的验证结果
   - 建议：添加验证检查清单

3. **缺少发布通知**
   - 问题：发布状态变更无自动通知
   - 建议：集成通知服务

**优化建议**：

```go
// 发布阶段管理服务
type ReleasePhaseService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 创建发布阶段
func (s *ReleasePhaseService) CreatePhase(ctx context.Context, req *CreateReleasePhaseRequest) (*ent.ReleasePhase, error) {
    return s.client.ReleasePhase.Create().
        SetReleaseID(req.ReleaseID).
        SetName(req.Name).
        SetEnvironment(req.Environment). // dev, test, staging, production
        SetStatus("pending").
        SetPlannedStartDate(req.PlannedStartDate).
        SetPlannedEndDate(req.PlannedEndDate).
        SetOrder(req.Order).
        SetTenantID(req.TenantID).
        Save(ctx)
}

// 开始发布阶段
func (s *ReleasePhaseService) StartPhase(ctx context.Context, phaseID int) error {
    phase, err := s.client.ReleasePhase.Get(ctx, phaseID)
    if err != nil {
        return err
    }

    // 检查前置阶段是否完成
    if phase.Order > 1 {
        previousPhase, err := s.client.ReleasePhase.Query().
            Where(
                releasephase.ReleaseID(phase.ReleaseID),
                releasephase.Order(phase.Order-1),
            ).
            Only(ctx)
        if err != nil {
            return err
        }

        if previousPhase.Status != "completed" {
            return fmt.Errorf("前置阶段未完成")
        }
    }

    // 更新阶段状态
    now := time.Now()
    return phase.Update().
        SetStatus("in_progress").
        SetActualStartDate(now).
        Exec(ctx)
}

// 完成发布阶段
func (s *ReleasePhaseService) CompletePhase(ctx context.Context, phaseID int, success bool, notes string) error {
    phase, err := s.client.ReleasePhase.Get(ctx, phaseID)
    if err != nil {
        return err
    }

    status := "completed"
    if !success {
        status = "failed"
    }

    now := time.Now()
    return phase.Update().
        SetStatus(status).
        SetActualEndDate(now).
        SetNotes(notes).
        Exec(ctx)
}

// 发布验证服务
type ReleaseValidationService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// 创建验证检查清单
func (s *ReleaseValidationService) CreateChecklist(ctx context.Context, req *CreateValidationChecklistRequest) (*ent.ReleaseValidationChecklist, error) {
    return s.client.ReleaseValidationChecklist.Create().
        SetReleaseID(req.ReleaseID).
        SetName(req.Name).
        SetDescription(req.Description).
        SetCheckItems(req.CheckItems).
        SetTenantID(req.TenantID).
        Save(ctx)
}

// 执行验证检查
func (s *ReleaseValidationService) PerformValidation(ctx context.Context, req *PerformValidationRequest) (*ent.ReleaseValidationResult, error) {
    checklist, err := s.client.ReleaseValidationChecklist.Get(ctx, req.ChecklistID)
    if err != nil {
        return nil, err
    }

    // 创建验证结果
    result, err := s.client.ReleaseValidationResult.Create().
        SetChecklistID(req.ChecklistID).
        SetReleaseID(checklist.ReleaseID).
        SetValidatedBy(req.ValidatedBy).
        SetValidatedAt(time.Now()).
        SetResults(req.Results).
        SetOverallStatus(req.OverallStatus). // passed, failed, partial
        SetNotes(req.Notes).
        SetTenantID(req.TenantID).
        Save(ctx)
    if err != nil {
        return nil, err
    }

    // 如果验证失败，更新发布状态
    if req.OverallStatus == "failed" {
        s.client.Release.UpdateOneID(checklist.ReleaseID).
            SetStatus("validation_failed").
            Exec(ctx)
    }

    return result, nil
}

// 发布通知服务
type ReleaseNotificationService struct {
    client              *ent.Client
    notificationService *UnifiedNotificationService
    logger              *zap.SugaredLogger
}

// 发送发布通知
func (s *ReleaseNotificationService) NotifyReleaseStatusChange(ctx context.Context, releaseID int, oldStatus, newStatus string) error {
    release, err := s.client.Release.Get(ctx, releaseID)
    if err != nil {
        return err
    }

    // 获取相关人员
    recipients := []int{release.CreatedBy}
    if release.OwnerID != nil {
        recipients = append(recipients, *release.OwnerID)
    }

    // 构建通知内容
    title := fmt.Sprintf("发布状态变更: %s", release.Title)
    content := fmt.Sprintf("发布 %s 的状态从 %s 变更为 %s", release.ReleaseNumber, oldStatus, newStatus)

    // 发送通知
    return s.notificationService.SendNotification(ctx, &NotificationRequest{
        Type:       "release_status_change",
        Title:      title,
        Content:    content,
        Recipients: recipients,
        Priority:   "medium",
        Channels:   []string{"email", "in_app"},
        Metadata: map[string]interface{}{
            "release_id":     releaseID,
            "release_number": release.ReleaseNumber,
            "old_status":     oldStatus,
            "new_status":     newStatus,
        },
    })
}
```

