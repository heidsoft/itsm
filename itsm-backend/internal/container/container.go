// Package container 提供依赖注入容器
// 使用构造函数注入替代 Setter 注入，确保依赖关系明确
package container

import (
	"database/sql"

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
	ticketService        *service.TicketServiceV2
	incidentService      *service.IncidentService
	notificationService  *service.NotificationService
	approvalService      *service.ApprovalService
	sequenceService      *service.SequenceService

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
}

// initBusinessServices 初始化业务服务
func (c *Container) initBusinessServices() {
	// Ticket Service V2（使用构造函数注入）
	c.ticketService = service.NewTicketServiceV2(&service.TicketServiceV2Config{
		Repository:            c.ticketRepository,
		Logger:               c.logger,
		NotificationService:   c.ticketNotificationService,
		ApprovalService:       c.approvalService,
		AutomationRuleService: c.ticketAutomationService,
		SLAService:            c.ticketSLAService,
	})
}

// ==================== Getters ====================

// GetTicketRepository 获取工单仓储
func (c *Container) GetTicketRepository() ticketRepo.Repository {
	return c.ticketRepository
}

// GetTicketService 获取工单服务
func (c *Container) GetTicketService() *service.TicketServiceV2 {
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
) *service.TicketServiceV2 {
	return service.NewTicketServiceV2(&service.TicketServiceV2Config{
		Repository:            c.ticketRepository,
		Logger:               c.logger,
		NotificationService:   notificationSvc,
		ApprovalService:       approvalSvc,
		AutomationRuleService: automationSvc,
		SLAService:            slaSvc,
	})
}
