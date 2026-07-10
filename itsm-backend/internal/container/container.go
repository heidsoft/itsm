// Package container 提供依赖注入容器
// 使用构造函数注入替代 Setter 注入，确保依赖关系明确
package container

import (
	"context"
	"database/sql"
	"fmt"

	"itsm-backend/config"
	"itsm-backend/ent"
	"itsm-backend/repository/base"
	ticketRepo "itsm-backend/repository/ticket"
	"itsm-backend/service"

	"go.uber.org/zap"
)

// Container 依赖容器
// 集中管理所有服务的创建和依赖注入
type Container struct {
	cfg    *config.Config
	client *ent.Client
	db     *sql.DB
	logger *zap.SugaredLogger

	// Repositories
	ticketRepository ticketRepo.Repository

	// Core Services
	ticketService         *service.TicketService
	incidentService       *service.IncidentService
	notificationService   *service.NotificationService
	approvalService       *service.ApprovalService
	sequenceService       *service.SequenceService
	processTriggerService service.ProcessTriggerServiceInterface
	processResolver       *service.ProcessResolver
	processBindingService service.ProcessBindingServiceInterface

	// Ticket-related Services
	ticketNotificationService *service.TicketNotificationService
	ticketSLAService          *service.TicketSLAService
	ticketAutomationService   *service.TicketAutomationRuleService
}

// NewContainer 创建依赖容器
func NewContainer(cfg *config.Config, client *ent.Client, db *sql.DB, logger *zap.SugaredLogger) *Container {
	return &Container{
		cfg:    cfg,
		client: client,
		db:     db,
		logger: logger,
	}
}

// Initialize 初始化所有依赖
// 按照依赖顺序初始化，确保每个依赖都已就绪
func (c *Container) Initialize() error {
	c.logger.Info("Initializing dependency container...")

	// 1. 初始化 Repositories
	c.initRepositories()

	// 2. 初始化核心服务（无依赖或依赖已初始化）
	c.initCoreServices()

	// 3. 初始化业务服务（依赖核心服务）
	c.initBusinessServices()

	c.logger.Info("Dependency container initialized successfully")
	return nil
}

// initRepositories 初始化仓储层
func (c *Container) initRepositories() {
	// Ticket Repository
	c.ticketRepository = ticketRepo.NewEntRepository(c.client, c.logger)
}

// initCoreServices 初始化核心服务
func (c *Container) initCoreServices() {
	// Sequence Service（用于编号生成）
	c.sequenceService = service.NewSequenceService(
		c.cfg.Redis.Host,
		c.cfg.Redis.Port,
		c.cfg.Redis.Password,
		c.cfg.Redis.DB,
		c.logger,
	)
	if c.sequenceService != nil {
		c.logger.Info("Redis sequence service initialized")
		// 注入DB查询函数：Redis初始化时从DB同步序列起点
		c.sequenceService.SetDBQueryFunc(c.queryMaxTicketSeqFromDB)
		c.logger.Info("DB sequence sync function registered for Redis initialization")
	} else {
		c.logger.Warn("Redis sequence service not available, using database fallback")
	}

	// Notification Service
	c.notificationService = service.NewNotificationService(c.client)

	// Approval Service
	c.approvalService = service.NewApprovalService(c.client, c.logger)

	// Incident Service
	c.incidentService = service.NewIncidentService(c.client, c.logger)
	c.incidentService.SetSequenceService(c.sequenceService)

	// Ticket Notification Service
	c.ticketNotificationService = service.NewTicketNotificationService(c.client, c.logger)

	// Ticket SLA Service
	c.ticketSLAService = service.NewTicketSLAService(c.client, c.logger)

	// Ticket Automation Service
	c.ticketAutomationService = service.NewTicketAutomationRuleService(c.client, c.logger)

	// BPMN 子服务：Binding / Trigger / Resolver
	// 注意顺序：Binding -> Resolver（依赖 Binding）-> Trigger（依赖 Engine 与 Binding）
	c.processBindingService = service.NewProcessBindingService(c.client)
	c.processResolver = service.NewProcessResolver(c.client, c.processBindingService)
	processEngine := service.NewCustomProcessEngine(c.client, c.logger)
	c.processTriggerService = service.NewProcessTriggerService(c.client, processEngine)
}

// initBusinessServices 初始化业务服务
func (c *Container) initBusinessServices() {
	// Ticket Service V2（使用构造函数注入）
	c.ticketService = service.NewTicketService(&service.TicketServiceConfig{
		Repository:            c.ticketRepository,
		Client:                c.client,
		Logger:                c.logger,
		NotificationService:   c.ticketNotificationService,
		ApprovalService:       c.approvalService,
		AutomationRuleService: c.ticketAutomationService,
		SLAService:            c.ticketSLAService,
		ProcessTriggerService: c.processTriggerService,
		ProcessResolver:       c.processResolver,
	})
}

// ==================== Getters ====================

// GetTicketRepository 获取工单仓储
func (c *Container) GetTicketRepository() ticketRepo.Repository {
	return c.ticketRepository
}

// GetTicketService 获取工单服务
func (c *Container) GetTicketService() *service.TicketService {
	return c.ticketService
}

// GetIncidentService 获取事件服务
func (c *Container) GetIncidentService() *service.IncidentService {
	return c.incidentService
}

// GetNotificationService 获取通知服务
func (c *Container) GetNotificationService() *service.NotificationService {
	return c.notificationService
}

// GetApprovalService 获取审批服务
func (c *Container) GetApprovalService() *service.ApprovalService {
	return c.approvalService
}

// GetSequenceService 获取序列服务
func (c *Container) GetSequenceService() *service.SequenceService {
	return c.sequenceService
}

// GetTicketNotificationService 获取工单通知服务
func (c *Container) GetTicketNotificationService() *service.TicketNotificationService {
	return c.ticketNotificationService
}

// GetTicketSLAService 获取工单 SLA 服务
func (c *Container) GetTicketSLAService() *service.TicketSLAService {
	return c.ticketSLAService
}

// GetTicketAutomationService 获取工单自动化服务
func (c *Container) GetTicketAutomationService() *service.TicketAutomationRuleService {
	return c.ticketAutomationService
}

// ==================== Factory Methods ====================

// NewBaseRepository 创建基础仓储
func (c *Container) NewBaseRepository() *base.EntRepository {
	return base.NewEntRepository(c.client)
}

// NewTicketServiceWithDeps 创建工单服务（带自定义依赖）
func (c *Container) NewTicketServiceWithDeps(
	notificationSvc *service.TicketNotificationService,
	approvalSvc *service.ApprovalService,
	automationSvc *service.TicketAutomationRuleService,
	slaSvc *service.TicketSLAService,
) *service.TicketService {
	return service.NewTicketService(&service.TicketServiceConfig{
		Repository:            c.ticketRepository,
		Logger:                c.logger,
		NotificationService:   notificationSvc,
		ApprovalService:       approvalSvc,
		AutomationRuleService: automationSvc,
		SLAService:            slaSvc,
	})
}

// queryMaxTicketSeqFromDB 从DB查询指定key对应的最大工单序列号
// key格式: "sequence:ticket:tenantId:YYYYMM" → 提取tenantId和年月，查询MAX(ticket_number)
// 返回值可直接作为Redis序列的起点
func (c *Container) queryMaxTicketSeqFromDB(key string) (int64, error) {
	// 解析key: sequence:ticket:tenantId:YYYYMM
	var tenantID int
	var year, month int
	n, err := fmt.Sscanf(key, "sequence:ticket:%d:%04d%02d", &tenantID, &year, &month)
	if err != nil || n != 3 {
		return 0, fmt.Errorf("invalid sequence key format: %s", key)
	}

	prefix := fmt.Sprintf("TKT-%04d%02d-", year, month)
	query := `SELECT ticket_number FROM tickets WHERE tenant_id = $1 AND ticket_number LIKE $2 AND ticket_number IS NOT NULL AND ticket_number != '' ORDER BY ticket_number DESC LIMIT 1`
	var maxTicketNum string
	err = c.db.QueryRowContext(context.Background(), query, tenantID, prefix+"%").Scan(&maxTicketNum)
	if err == sql.ErrNoRows || maxTicketNum == "" {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query max ticket number failed: %w", err)
	}

	// 解析末尾数字
	idx := len(maxTicketNum) - 1
	for idx >= 0 && maxTicketNum[idx] != '-' {
		idx--
	}
	if idx < 0 {
		return 0, fmt.Errorf("invalid ticket number format: %s", maxTicketNum)
	}
	var seq int64
	fmt.Sscanf(maxTicketNum[idx+1:], "%d", &seq)
	return seq, nil
}
