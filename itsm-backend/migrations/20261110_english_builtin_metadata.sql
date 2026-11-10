-- English display text for unchanged product baseline records.
-- Match complete original display text plus stable keys/configuration. Customized rows are untouched.
-- IDs, tenant ownership, group/role codes, categories, relationships and business data are preserved.
-- Name collisions in the same tenant are skipped. Run through the standard transactional migrator.

UPDATE departments AS t
SET name = 'Information Technology', description = 'IT management'
WHERE t.name = '信息技术部'
  AND t.description = 'IT整体管理'
  AND t.code = 'IT'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Information Technology' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT Infrastructure', description = 'Infrastructure operations'
WHERE t.name = 'IT基础架构'
  AND t.description = '基础设施运维'
  AND t.code = 'IT-INFRA'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT Infrastructure' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT Applications', description = 'Application operations'
WHERE t.name = 'IT应用服务'
  AND t.description = '应用系统运维'
  AND t.code = 'IT-APP'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT Applications' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT Security', description = 'Information security management'
WHERE t.name = 'IT安全'
  AND t.description = '信息安全管理'
  AND t.code = 'IT-SEC'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT Security' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT Project Management', description = 'IT Project Management'
WHERE t.name = 'IT项目管理'
  AND t.description = 'IT项目管理'
  AND t.code = 'IT-PMO'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT Project Management' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Operations', description = 'IT operations management'
WHERE t.name = '运营管理部'
  AND t.description = 'IT运营管理'
  AND t.code = 'OPS'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Operations' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Service Desk', description = 'First-line support'
WHERE t.name = '服务台'
  AND t.description = '一线服务支持'
  AND t.code = 'OPS-SD'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Desk' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Operations Center', description = '24/7 operations monitoring'
WHERE t.name = '运维中心'
  AND t.description = '7x24运维监控'
  AND t.code = 'OPS-NOC'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Operations Center' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Customer Service', description = 'Customer service experience'
WHERE t.name = '客户服务'
  AND t.description = '客户服务体验'
  AND t.code = 'OPS-CS'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Customer Service' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Research and Development', description = 'Product development'
WHERE t.name = '研发部'
  AND t.description = '产品研发'
  AND t.code = 'RD'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Research and Development' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Quality Assurance', description = 'Quality assurance'
WHERE t.name = '测试部'
  AND t.description = '质量保证'
  AND t.code = 'QA'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Quality Assurance' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Human Resources', description = 'Human resources management'
WHERE t.name = '人力资源部'
  AND t.description = '人力资源管理'
  AND t.code = 'HR'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Human Resources' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Finance', description = 'Financial management'
WHERE t.name = '财务部'
  AND t.description = '财务管理'
  AND t.code = 'FIN'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Finance' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Administration', description = 'Administration management'
WHERE t.name = '行政部'
  AND t.description = '行政管理'
  AND t.code = 'ADMIN'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Administration' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Service Desk L1', description = 'First-line support'
WHERE t.name = '服务台-L1'
  AND t.description = '一线服务支持'
  AND t.code = '服务台-l1'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Desk L1' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Service Desk L2', description = 'Second-line technical support'
WHERE t.name = '服务台-L2'
  AND t.description = '二线技术支持'
  AND t.code = '服务台-l2'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Desk L2' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Service Desk L3', description = 'Third-line technical specialists'
WHERE t.name = '服务台-L3'
  AND t.description = '三线技术专家'
  AND t.code = '服务台-l3'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Desk L3' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Server Operations', description = 'Server operations management'
WHERE t.name = '服务器运维'
  AND t.description = '服务器运维管理'
  AND t.code = '服务器运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Server Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Network Operations', description = 'Network device operations'
WHERE t.name = '网络运维'
  AND t.description = '网络设备运维'
  AND t.code = '网络运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Network Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Database Operations', description = 'Database operations management'
WHERE t.name = '数据库运维'
  AND t.description = '数据库运维管理'
  AND t.code = '数据库运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Database Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Cloud Operations', description = 'Cloud platform operations'
WHERE t.name = '云平台运维'
  AND t.description = '云计算平台运维'
  AND t.code = '云平台运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Cloud Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'ERP Support', description = 'ERP system support'
WHERE t.name = 'ERP支持'
  AND t.description = 'ERP系统支持'
  AND t.code = 'erp支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ERP Support' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'CRM Support', description = 'CRM system support'
WHERE t.name = 'CRM支持'
  AND t.description = 'CRM系统支持'
  AND t.code = 'crm支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'CRM Support' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Office Applications Support', description = 'Office application support'
WHERE t.name = 'OA支持'
  AND t.description = 'OA办公系统支持'
  AND t.code = 'oa支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Office Applications Support' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Security Operations', description = 'Security monitoring and response'
WHERE t.name = '安全运营'
  AND t.description = '安全监控与响应'
  AND t.code = '安全运营'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Security Compliance', description = 'Security compliance management'
WHERE t.name = '安全合规'
  AND t.description = '安全合规管理'
  AND t.code = '安全合规'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Compliance' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Backend Development', description = 'Backend development team'
WHERE t.name = '后端开发'
  AND t.description = '后端开发团队'
  AND t.code = '后端开发'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Backend Development' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Frontend Development', description = 'Frontend development team'
WHERE t.name = '前端开发'
  AND t.description = '前端开发团队'
  AND t.code = '前端开发'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Frontend Development' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Mobile Development', description = 'Mobile development team'
WHERE t.name = '移动开发'
  AND t.description = '移动端开发团队'
  AND t.code = '移动开发'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Mobile Development' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Testing Team', description = 'Testing and quality assurance'
WHERE t.name = '测试团队'
  AND t.description = '测试与质量保证'
  AND t.code = '测试团队'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Testing Team' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Customer Success', description = 'Customer success management'
WHERE t.name = '客户成功'
  AND t.description = '客户成功管理'
  AND t.code = '客户成功'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Customer Success' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Technical Support', description = 'Customer technical support'
WHERE t.name = '技术支持'
  AND t.description = '客户服务技术支持'
  AND t.code = '技术支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Technical Support' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Change Manager', description = 'Change lifecycle, CAB reviews, and release coordination'
WHERE t.name = '变更经理'
  AND t.description = '变更生命周期、CAB 与发布联动'
  AND t.code = 'change_manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Change Manager' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Problem Manager', description = 'Root cause analysis, known errors, and workarounds'
WHERE t.name = '问题经理'
  AND t.description = '根因分析、已知错误与变通方案'
  AND t.code = 'problem_manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Problem Manager' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Knowledge Administrator', description = 'Knowledge article lifecycle, review, and publishing'
WHERE t.name = '知识管理员'
  AND t.description = '知识文章生命周期与评审发布'
  AND t.code = 'knowledge_manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Knowledge Administrator' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Configuration Administrator', description = 'CI types, instances, relationships, and reconciliation'
WHERE t.name = '配置管理员'
  AND t.description = 'CI 类型/实例/关系与对账'
  AND t.code = 'cmdb_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Configuration Administrator' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Service Catalog Administrator', description = 'Service catalogs, request templates, and ticket templates'
WHERE t.name = '服务目录管理员'
  AND t.description = '服务目录、请求模板与工单模板配置'
  AND t.code = 'service_catalog_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Catalog Administrator' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP Viewer', description = 'Read-only collaboration on MSP customer tickets'
WHERE t.name = 'MSP查看者'
  AND t.description = 'MSP 客户工单只读协作'
  AND t.code = 'msp_viewer'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP Viewer' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP Technician', description = 'MSP ticket assignment and handling'
WHERE t.name = 'MSP技术员'
  AND t.description = 'MSP 派单处理协作'
  AND t.code = 'msp_tech'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP Technician' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP Specialist', description = 'MSP specialist support and customer management'
WHERE t.name = 'MSP专家'
  AND t.description = 'MSP 专家级处理与客户维护'
  AND t.code = 'msp_specialist'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP Specialist' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP Manager', description = 'MSP delivery and resource allocation'
WHERE t.name = 'MSP经理'
  AND t.description = 'MSP 交付与资源分配管理'
  AND t.code = 'msp_manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP Manager' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP Administrator', description = 'Full MSP collaboration management'
WHERE t.name = 'MSP管理员'
  AND t.description = 'MSP 全量协作管理'
  AND t.code = 'msp_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP Administrator' AND sibling.id <> t.id);

UPDATE groups AS t
SET description = 'First-line approval group: add members with the agent role'
WHERE t.description = '一线审批组：从 agent 角色人员中拉人组成'
  AND t.name = 'approvers-l1';

UPDATE groups AS t
SET description = 'Second-line approval group: add members with technician or it_admin roles'
WHERE t.description = '二线审批组：从 technician / it_admin 角色人员中拉人组成'
  AND t.name = 'approvers-l2';

UPDATE groups AS t
SET description = 'Third-line approval group: add members with it_admin or security_admin roles'
WHERE t.description = '三线审批组：从 it_admin / security_admin 角色人员中拉人组成'
  AND t.name = 'approvers-l3';

UPDATE groups AS t
SET description = 'Management approval group: add members with the manager role'
WHERE t.description = '管理审批组：从 manager 角色人员中拉人组成'
  AND t.name = 'approvers-managers';

UPDATE groups AS t
SET description = 'Security approval group: add members with the security_admin role'
WHERE t.description = '安全审批组：从 security_admin 角色人员中拉人组成'
  AND t.name = 'approvers-security';

UPDATE groups AS t
SET description = 'Change advisory board: add members with change_manager or manager roles'
WHERE t.description = '变更审批组：从 change_manager / manager 角色人员中拉人组成（变更委员会）'
  AND t.name = 'approvers-change';

UPDATE sla_definitions AS t
SET name = 'SLA-P0-Urgent', description = 'P0 urgent SLA'
WHERE t.name = 'SLA-P0-紧急'
  AND t.description = 'P0紧急级别SLA'
  AND t.service_type = 'incident'
  AND t.priority = 'urgent'
  AND t.response_time = 15
  AND t.resolution_time = 120
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-P0-Urgent' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-P1-High', description = 'P1 high priority SLA'
WHERE t.name = 'SLA-P1-高'
  AND t.description = 'P1高级别SLA'
  AND t.service_type = 'incident'
  AND t.priority = 'high'
  AND t.response_time = 30
  AND t.resolution_time = 240
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-P1-High' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-P2-Medium', description = 'P2 medium priority SLA'
WHERE t.name = 'SLA-P2-中'
  AND t.description = 'P2中级别SLA'
  AND t.service_type = 'incident'
  AND t.priority = 'medium'
  AND t.response_time = 120
  AND t.resolution_time = 480
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-P2-Medium' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-P3-Low', description = 'P3 low priority SLA'
WHERE t.name = 'SLA-P3-低'
  AND t.description = 'P3低级别SLA'
  AND t.service_type = 'incident'
  AND t.priority = 'low'
  AND t.response_time = 240
  AND t.resolution_time = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-P3-Low' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-Service Request', description = 'Standard service request SLA'
WHERE t.name = 'SLA-服务请求'
  AND t.description = '服务请求标准SLA'
  AND t.service_type = 'service_request'
  AND t.priority = 'medium'
  AND t.response_time = 480
  AND t.resolution_time = 4320
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-Service Request' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-Change', description = 'Change request SLA'
WHERE t.name = 'SLA-变更'
  AND t.description = '变更请求SLA'
  AND t.service_type = 'change'
  AND t.priority = 'high'
  AND t.response_time = 60
  AND t.resolution_time = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-Change' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'ECS Cloud Server', description = 'Elastic cloud server'
WHERE t.name = '云服务器 ECS'
  AND t.description = '弹性云服务器'
  AND t.category = '云计算'
  AND t.service_type = 'vm'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ECS Cloud Server' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'RDS Cloud Database', description = 'MySQL/PostgreSQL database'
WHERE t.name = '云数据库 RDS'
  AND t.description = 'MySQL/PostgreSQL数据库'
  AND t.category = '数据库'
  AND t.service_type = 'rds'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'RDS Cloud Database' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'OSS Object Storage', description = 'Scalable cloud storage'
WHERE t.name = '对象存储 OSS'
  AND t.description = '海量云存储'
  AND t.category = '存储'
  AND t.service_type = 'oss'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'OSS Object Storage' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'CDN Acceleration', description = 'Content delivery acceleration'
WHERE t.name = 'CDN 加速'
  AND t.description = '内容分发加速'
  AND t.category = '网络'
  AND t.service_type = 'network'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'CDN Acceleration' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'SLB Load Balancer', description = 'Traffic distribution service'
WHERE t.name = '负载均衡 SLB'
  AND t.description = '流量分发服务'
  AND t.category = '网络'
  AND t.service_type = 'network'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLB Load Balancer' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'VPN Gateway', description = 'Encrypted VPN tunnel'
WHERE t.name = 'VPN 网关'
  AND t.description = 'VPN加密通道'
  AND t.category = '安全'
  AND t.service_type = 'security'
  AND t.delivery_time = 2
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'VPN Gateway' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Enterprise Email', description = 'Corporate domain email'
WHERE t.name = '企业邮箱'
  AND t.description = '企业域名邮箱'
  AND t.category = '通讯'
  AND t.service_type = 'custom'
  AND t.delivery_time = 1
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Enterprise Email' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Enterprise File Storage', description = 'File storage and sharing'
WHERE t.name = '企业网盘'
  AND t.description = '文件存储共享'
  AND t.category = '协作'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Enterprise File Storage' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Video Conferencing', description = 'HD video conferencing'
WHERE t.name = '视频会议'
  AND t.description = '高清视频会议'
  AND t.category = '通讯'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Video Conferencing' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Enterprise Messaging', description = 'Instant messaging'
WHERE t.name = '企业IM'
  AND t.description = '即时通讯工具'
  AND t.category = '通讯'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Enterprise Messaging' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Vulnerability Scanning', description = 'Web vulnerability scanning'
WHERE t.name = '漏洞扫描'
  AND t.description = 'Web漏洞扫描'
  AND t.category = '安全'
  AND t.service_type = 'security'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Vulnerability Scanning' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Penetration Testing', description = 'Security penetration testing'
WHERE t.name = '渗透测试'
  AND t.description = '安全渗透测试'
  AND t.category = '安全'
  AND t.service_type = 'security'
  AND t.delivery_time = 5
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Penetration Testing' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Security Compliance Assessment', description = 'Classified protection compliance consulting'
WHERE t.name = '等保合规'
  AND t.description = '等级保护咨询'
  AND t.category = '安全'
  AND t.service_type = 'security'
  AND t.delivery_time = 30
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Compliance Assessment' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'IT Service Desk', description = 'IT consultation and support'
WHERE t.name = 'IT服务台'
  AND t.description = 'IT问题咨询支持'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT Service Desk' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Software Installation', description = 'Standard Software Installation'
WHERE t.name = '软件安装'
  AND t.description = '标准软件安装'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 1
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Software Installation' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Account Request', description = 'New Employee Account Setup'
WHERE t.name = '账户申请'
  AND t.description = '新员工账户开通'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Account Request' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Network Access', description = 'Network access request'
WHERE t.name = '网络接入'
  AND t.description = '网络接入申请'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 2
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Network Access' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Domain Request', description = 'Internal domain registration'
WHERE t.name = '域名申请'
  AND t.description = '内部域名注册'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 3
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Domain Request' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Code Repository', description = 'Git code repository'
WHERE t.name = '代码仓库'
  AND t.description = 'Git代码仓库'
  AND t.category = '开发'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Code Repository' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'CI/CD Pipeline', description = 'Automated deployment'
WHERE t.name = 'CI/CD流水线'
  AND t.description = '自动化部署'
  AND t.category = '开发'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'CI/CD Pipeline' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'Test Environment', description = 'Staging test environment'
WHERE t.name = '测试环境'
  AND t.description = '预发布测试环境'
  AND t.category = '开发'
  AND t.service_type = 'custom'
  AND t.delivery_time = 2
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Test Environment' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'API Gateway', description = 'API management'
WHERE t.name = 'API网关'
  AND t.description = 'API接口管理'
  AND t.category = '开发'
  AND t.service_type = 'custom'
  AND t.delivery_time = 3
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'API Gateway' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'ECS Instance Request', description = 'Request a new elastic cloud server instance'
WHERE t.name = 'ECS 实例申请'
  AND t.description = '申请新建弹性云服务器实例'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ECS Instance Request' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'ECS Configuration Change', description = 'Change the specifications of an existing ECS instance'
WHERE t.name = 'ECS 配置变更'
  AND t.description = '变更现有 ECS 实例规格'
  AND t.business_sub_type = 'change'
  AND t.process_definition_key = 'change_normal_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ECS Configuration Change' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'ECS Incident Report', description = 'Report an ECS instance incident'
WHERE t.name = 'ECS 故障报修'
  AND t.description = 'ECS 实例运行异常报修'
  AND t.business_sub_type = 'incident'
  AND t.process_definition_key = 'incident_emergency_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 0
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ECS Incident Report' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'RDS Instance Request', description = 'Request a new database instance'
WHERE t.name = 'RDS 实例申请'
  AND t.description = '申请新建数据库实例'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'RDS Instance Request' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Database Backup Recovery', description = 'Request database backup recovery'
WHERE t.name = '数据库备份恢复'
  AND t.description = '申请数据库备份恢复服务'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 2
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Database Backup Recovery' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'IT Consultation', description = 'IT questions and assistance'
WHERE t.name = 'IT 问题咨询'
  AND t.description = 'IT 相关问题咨询与解答'
  AND t.business_sub_type = 'ticket'
  AND t.process_definition_key = 'ticket_general_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 0
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT Consultation' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Password Reset', description = 'Account password reset service'
WHERE t.name = '密码重置'
  AND t.description = '账户密码重置服务'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 0
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Password Reset' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Standard Software Installation', description = 'Install approved corporate software'
WHERE t.name = '标准软件安装'
  AND t.description = '安装公司授权的标准软件'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Standard Software Installation' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'New Employee Account Setup', description = 'Set up IT accounts and access for new employees'
WHERE t.name = '新员工账户开通'
  AND t.description = '新员工 IT 账户与权限开通'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'New Employee Account Setup' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Business Access Change', description = 'Change application, database, or infrastructure access'
WHERE t.name = '业务权限变更'
  AND t.description = '业务系统、数据库或基础设施权限调整'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 2
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Business Access Change' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Office Network Access', description = 'Request office network access'
WHERE t.name = '办公网络接入'
  AND t.description = '申请办公网络接入权限'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 2
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Office Network Access' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Internal domain registration', description = 'Register an internal system domain'
WHERE t.name = '内部域名注册'
  AND t.description = '注册内部系统域名'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 3
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Internal domain registration' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Git Repository Creation', description = 'Request a new Git repository'
WHERE t.name = 'Git 仓库创建'
  AND t.description = '申请创建新的 Git 代码仓库'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 0
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Git Repository Creation' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Test Environment Request', description = 'Request a staging test environment'
WHERE t.name = '测试环境申请'
  AND t.description = '申请预发布测试环境'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 2
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Test Environment Request' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Web Vulnerability Scan', description = 'Request a web application vulnerability scan'
WHERE t.name = 'Web 漏洞扫描'
  AND t.description = '申请 Web 应用安全漏洞扫描'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Web Vulnerability Scan' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Security penetration testing', description = 'Request a security penetration test'
WHERE t.name = '安全渗透测试'
  AND t.description = '申请系统安全渗透测试'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 5
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security penetration testing' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = 'P0/P1 Incident Approval', description = 'Urgent and high priority incidents require supervisor approval', nodes = '[{"approver_type":"manager","name":"Supervisor Approval","timeout":60,"type":"approval"}]'::jsonb
WHERE t.name = 'P0/P1事件审批'
  AND t.description = '紧急和高优先级事件需要主管审批'
  AND t.nodes::jsonb = '[{"approver_type":"manager","name":"主管审批","timeout":60,"type":"approval"}]'::jsonb
  AND t.ticket_type = 'incident'
  AND t.priority = 'urgent,high'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'P0/P1 Incident Approval' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = 'Change Approval', description = 'All change requests require multilevel approval', nodes = '[{"approver_type":"role","name":"Technical Approval","role":"engineer","timeout":240,"type":"approval"},{"approver_type":"role","name":"Manager Approval","role":"manager","timeout":480,"type":"approval"}]'::jsonb
WHERE t.name = '变更审批'
  AND t.description = '所有变更请求需要多级审批'
  AND t.nodes::jsonb = '[{"approver_type":"role","name":"技术审批","role":"engineer","timeout":240,"type":"approval"},{"approver_type":"role","name":"经理审批","role":"manager","timeout":480,"type":"approval"}]'::jsonb
  AND t.ticket_type = 'change'
  AND t.priority = ''
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Change Approval' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = 'Service Request Approval', description = 'High value service requests require approval', nodes = '[{"approver_type":"manager","name":"Service Approval","timeout":120,"type":"approval"}]'::jsonb
WHERE t.name = '服务请求审批'
  AND t.description = '高价值服务请求需要审批'
  AND t.nodes::jsonb = '[{"approver_type":"manager","name":"服务审批","timeout":120,"type":"approval"}]'::jsonb
  AND t.ticket_type = 'service_request'
  AND t.priority = 'high'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Request Approval' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = 'My Pending Tickets', description = 'Open tickets assigned to me'
WHERE t.name = '我的待办工单'
  AND t.description = '分配给我的未关闭工单'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","created_at"]'::jsonb
  AND t.is_shared = false
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'My Pending Tickets' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = 'Tickets I Created', description = 'Tickets submitted by me'
WHERE t.name = '我创建的工单'
  AND t.description = '我提交的工单'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","created_at"]'::jsonb
  AND t.is_shared = false
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Tickets I Created' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = 'Urgent Tickets', description = 'Urgent and high priority tickets'
WHERE t.name = '紧急工单'
  AND t.description = '紧急和高优先级工单'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","created_at"]'::jsonb
  AND t.is_shared = true
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Urgent Tickets' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = 'Unassigned Tickets', description = 'Tickets awaiting assignment'
WHERE t.name = '未分配工单'
  AND t.description = '尚未分配的工单'
  AND t.columns::jsonb = '["id","title","priority","status","creator","created_at"]'::jsonb
  AND t.is_shared = true
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Unassigned Tickets' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = 'Closed Tickets', description = 'Completed tickets'
WHERE t.name = '已关闭工单'
  AND t.description = '已完成的工单'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","closed_at"]'::jsonb
  AND t.is_shared = false
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Closed Tickets' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT Project Management', description = 'IT project management office'
WHERE t.name = 'IT项目管理'
  AND t.description = 'IT项目管理办公室'
  AND t.code = 'IT-PMO'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT Project Management' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'Security Operations', description = 'Security monitoring and response'
WHERE t.name = '安全运营'
  AND t.description = '安全监控与响应'
  AND t.code = 'OPS-SEC'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Service Desk L1', description = 'First-line support for inquiries, accounts, and standard requests'
WHERE t.name = '服务台-L1'
  AND t.description = '一线服务支持，处理咨询、账号和标准请求'
  AND t.code = '服务台-l1'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Desk L1' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Service Desk L2', description = 'Second-line technical support for complex incidents and escalated tickets'
WHERE t.name = '服务台-L2'
  AND t.description = '二线技术支持，处理复杂事件和升级工单'
  AND t.code = '服务台-l2'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Desk L2' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Service Desk L3', description = 'Third-line specialists for complex problems and root cause analysis'
WHERE t.name = '服务台-L3'
  AND t.description = '三线技术专家，处理疑难问题和根因分析'
  AND t.code = '服务台-l3'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Desk L3' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Server Operations', description = 'Server, virtualization, and operating system operations'
WHERE t.name = '服务器运维'
  AND t.description = '服务器、虚拟化和操作系统运维'
  AND t.code = '服务器运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Server Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Network Operations', description = 'Network device, connectivity, and security zone operations'
WHERE t.name = '网络运维'
  AND t.description = '网络设备、链路和安全域运维'
  AND t.code = '网络运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Network Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Database Operations', description = 'Database instance, backup, and performance operations'
WHERE t.name = '数据库运维'
  AND t.description = '数据库实例、备份和性能运维'
  AND t.code = '数据库运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Database Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Cloud Operations', description = 'Public cloud, private cloud, and container platform operations'
WHERE t.name = '云平台运维'
  AND t.description = '公有云、私有云和容器平台运维'
  AND t.code = '云平台运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Cloud Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Application Support', description = 'Enterprise application, API, and middleware support'
WHERE t.name = '应用支持'
  AND t.description = '企业应用、接口和中间件支持'
  AND t.code = '应用支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Application Support' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Security Operations', description = 'Security incident response, vulnerability management, and compliance'
WHERE t.name = '安全运营'
  AND t.description = '安全事件响应、漏洞和合规运营'
  AND t.code = '安全运营'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Operations' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'Change Advisory Board', description = 'Change reviews, release windows, and risk management'
WHERE t.name = '变更委员会'
  AND t.description = '变更评审、发布窗口和风险控制'
  AND t.code = '变更委员会'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Change Advisory Board' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Incident-P0-Urgent', description = 'Critical business outage or major security incident'
WHERE t.name = 'Incident-P0-紧急'
  AND t.description = '核心业务中断或重大安全事件'
  AND t.service_type = 'incident'
  AND t.priority = 'urgent'
  AND t.response_time = 15
  AND t.resolution_time = 120
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Incident-P0-Urgent' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Incident-P1-High', description = 'Important business services affected; rapid recovery required'
WHERE t.name = 'Incident-P1-高'
  AND t.description = '重要业务受影响，需要快速恢复'
  AND t.service_type = 'incident'
  AND t.priority = 'high'
  AND t.response_time = 30
  AND t.resolution_time = 240
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Incident-P1-High' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Incident-P2-Medium', description = 'Limited functionality affected with a workaround available'
WHERE t.name = 'Incident-P2-中'
  AND t.description = '局部功能受影响，有替代方案'
  AND t.service_type = 'incident'
  AND t.priority = 'medium'
  AND t.response_time = 120
  AND t.resolution_time = 480
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Incident-P2-Medium' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Incident-P3-Low', description = 'Inquiries, minor impact, or non-urgent issues'
WHERE t.name = 'Incident-P3-低'
  AND t.description = '咨询、轻微影响或非紧急问题'
  AND t.service_type = 'incident'
  AND t.priority = 'low'
  AND t.response_time = 240
  AND t.resolution_time = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Incident-P3-Low' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'ServiceRequest-Standard', description = 'Standard service request delivery SLA'
WHERE t.name = 'ServiceRequest-标准'
  AND t.description = '标准服务请求交付SLA'
  AND t.service_type = 'service_request'
  AND t.priority = 'medium'
  AND t.response_time = 480
  AND t.resolution_time = 4320
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ServiceRequest-Standard' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Change-Normal', description = 'Normal change review and implementation SLA'
WHERE t.name = 'Change-普通'
  AND t.description = '普通变更评审与实施SLA'
  AND t.service_type = 'change'
  AND t.priority = 'medium'
  AND t.response_time = 480
  AND t.resolution_time = 10080
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Change-Normal' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Change-Emergency', description = 'Emergency change review and implementation SLA'
WHERE t.name = 'Change-紧急'
  AND t.description = '紧急变更快速评审与实施SLA'
  AND t.service_type = 'change'
  AND t.priority = 'high'
  AND t.response_time = 60
  AND t.resolution_time = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Change-Emergency' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = 'Service Request Approval', description = 'Service requests involving resources, access, or expenses require department head approval', nodes = '[{"type":"approval","name":"Department Head Approval","approver_type":"role","role":"manager","timeout":480}]'::jsonb
WHERE t.name = '服务请求审批'
  AND t.description = '涉及资源、权限或费用的服务请求需要部门负责人审批'
  AND t.nodes::jsonb = '[{"type":"approval","name":"部门负责人审批","approver_type":"role","role":"manager","timeout":480}]'::jsonb
  AND t.ticket_type = 'service_request'
  AND t.priority = ''
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Request Approval' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = 'Normal Change Approval', description = 'Normal changes require technical lead and change manager approval', nodes = '[{"type":"approval","name":"Technical Lead Approval","approver_type":"role","role":"it_admin","timeout":480},{"type":"approval","name":"Change Manager Approval","approver_type":"role","role":"change_manager","timeout":480}]'::jsonb
WHERE t.name = '普通变更审批'
  AND t.description = '普通变更需要技术负责人和变更经理审批'
  AND t.nodes::jsonb = '[{"type":"approval","name":"技术负责人审批","approver_type":"role","role":"it_admin","timeout":480},{"type":"approval","name":"变更经理审批","approver_type":"role","role":"change_manager","timeout":480}]'::jsonb
  AND t.ticket_type = 'change'
  AND t.priority = 'medium,high'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Normal Change Approval' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = 'Emergency Change Approval', description = 'Emergency changes use expedited approval and require a post-implementation review', nodes = '[{"type":"approval","name":"Duty Manager Approval","approver_type":"role","role":"manager","timeout":60},{"type":"approval","name":"Security Confirmation","approver_type":"role","role":"security_admin","timeout":120}]'::jsonb
WHERE t.name = '紧急变更审批'
  AND t.description = '紧急变更采用快速审批并要求事后复盘'
  AND t.nodes::jsonb = '[{"type":"approval","name":"值班经理审批","approver_type":"role","role":"manager","timeout":60},{"type":"approval","name":"安全确认","approver_type":"role","role":"security_admin","timeout":120}]'::jsonb
  AND t.ticket_type = 'change'
  AND t.priority = 'urgent'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Emergency Change Approval' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = 'Access Request Approval', description = 'Privileged access requires line manager and security administrator approval', nodes = '[{"type":"approval","name":"Line Manager Approval","approver_type":"role","role":"manager","timeout":480},{"type":"approval","name":"Security Administrator Approval","approver_type":"role","role":"security_admin","timeout":480}]'::jsonb
WHERE t.name = '权限申请审批'
  AND t.description = '高权限申请需要直属负责人和安全管理员审批'
  AND t.nodes::jsonb = '[{"type":"approval","name":"直属负责人审批","approver_type":"role","role":"manager","timeout":480},{"type":"approval","name":"安全管理员审批","approver_type":"role","role":"security_admin","timeout":480}]'::jsonb
  AND t.ticket_type = 'service_request'
  AND t.priority = 'high'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Access Request Approval' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = 'High Priority Tickets', description = 'P0/P1 or high priority tickets'
WHERE t.name = '高优先级工单'
  AND t.description = 'P0/P1或高优先级工单'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","created_at"]'::jsonb
  AND t.is_shared = true
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'High Priority Tickets' AND sibling.id <> t.id);

UPDATE ci_types AS t
SET description = 'Server'
WHERE t.description = '服务器'
  AND t.name = 'server'
  AND t.icon = 'server'
  AND t.color = '#28a745';

UPDATE ci_types AS t
SET description = 'Database'
WHERE t.description = '数据库'
  AND t.name = 'database'
  AND t.icon = 'database'
  AND t.color = '#fd7e14';

UPDATE ci_types AS t
SET description = 'Network Device'
WHERE t.description = '网络设备'
  AND t.name = 'network'
  AND t.icon = 'network'
  AND t.color = '#17a2b8';

UPDATE ci_types AS t
SET description = 'Storage Device'
WHERE t.description = '存储设备'
  AND t.name = 'storage'
  AND t.icon = 'storage'
  AND t.color = '#e83e8c';

UPDATE ci_types AS t
SET description = 'Application Service'
WHERE t.description = '应用服务'
  AND t.name = 'application'
  AND t.icon = 'app'
  AND t.color = '#6610f2';

UPDATE ci_types AS t
SET description = 'Middleware'
WHERE t.description = '中间件'
  AND t.name = 'middleware'
  AND t.icon = 'middleware'
  AND t.color = '#e74c3c';

UPDATE ci_types AS t
SET description = 'Cloud Virtual Machine'
WHERE t.description = '云虚拟机'
  AND t.name = 'cloud_vm'
  AND t.icon = 'cloud'
  AND t.color = '#6f42c1';

UPDATE ci_types AS t
SET description = 'Kubernetes Resource'
WHERE t.description = 'Kubernetes资源'
  AND t.name = 'kubernetes'
  AND t.icon = 'kubernetes'
  AND t.color = '#20c997';

UPDATE ticket_tags AS t
SET name = 'High Priority', description = 'Records requiring priority attention'
WHERE t.name = '高优先级'
  AND t.description = '需要优先处理的记录'
  AND t.color = '#f5222d'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'High Priority' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Approval Required', description = 'Records requiring an approval workflow'
WHERE t.name = '需审批'
  AND t.description = '需要审批流程的记录'
  AND t.color = '#faad14'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Approval Required' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'AI Suggestion', description = 'Records containing AI-assisted suggestions'
WHERE t.name = 'AI建议'
  AND t.description = '包含AI辅助建议的记录'
  AND t.color = '#52c41a'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'AI Suggestion' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Customer Impact', description = 'Records affecting customers or end users'
WHERE t.name = '客户影响'
  AND t.description = '已影响客户或终端用户的记录'
  AND t.color = '#1677ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Customer Impact' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Hardware Failure', description = 'Server, storage, or network hardware failures'
WHERE t.name = '硬件故障'
  AND t.description = '服务器、存储、网络设备等硬件故障'
  AND t.code = 'hardware'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Hardware Failure' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Software Failure', description = 'Operating system or application failures'
WHERE t.name = '软件故障'
  AND t.description = '操作系统、应用软件故障'
  AND t.code = 'software'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Software Failure' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Network Failure', description = 'Network connectivity or device issues'
WHERE t.name = '网络故障'
  AND t.description = '网络连接、网络设备问题'
  AND t.code = 'network'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Network Failure' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Database Issue', description = 'Database performance, connectivity, backup, and access issues'
WHERE t.name = '数据库问题'
  AND t.description = '数据库性能、连接、备份和权限问题'
  AND t.code = 'database'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Database Issue' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Security Issue', description = 'Security incidents, vulnerabilities, and compliance issues'
WHERE t.name = '安全问题'
  AND t.description = '安全事件、漏洞和合规问题'
  AND t.code = 'security'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Issue' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Performance Issue', description = 'Slow response, capacity constraints, or resource bottlenecks'
WHERE t.name = '性能问题'
  AND t.description = '系统响应慢、容量或资源瓶颈'
  AND t.code = 'performance'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Performance Issue' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Configuration Issue', description = 'System, release, or parameter configuration errors'
WHERE t.name = '配置问题'
  AND t.description = '系统配置、发布配置或参数错误'
  AND t.code = 'configuration'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Configuration Issue' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Other', description = 'Other incident types'
WHERE t.name = '其他'
  AND t.description = '其他类型事件'
  AND t.code = 'other'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Other' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'Standard Software Installation', description = 'Install clients or tools from the approved software allowlist', implementation_plan = 'Verify version and licensing, follow the standard installation procedure, and confirm availability.', rollback_plan = 'Uninstall the software and restore the original configuration.', justification = 'Improve endpoint support efficiency.', prerequisites = '["Software is on the approved allowlist"]'::jsonb, remarks = 'May be performed by the service desk.'
WHERE t.title = '标准软件安装'
  AND t.description = '安装已批准的软件白名单中的客户端或工具'
  AND t.implementation_plan = '确认软件版本和授权，按标准安装步骤执行，完成后验证可用性。'
  AND t.rollback_plan = '卸载软件并恢复原配置。'
  AND t.justification = '提升终端支持效率。'
  AND t.affected_cis::jsonb = '[]'::jsonb
  AND t.prerequisites::jsonb = '["软件已进入白名单"]'::jsonb
  AND t.remarks = '可由服务台执行。'
  AND t.category = '终端支持'
  AND t.risk_level = 'low'
  AND t.impact_scope = '单用户或单设备'
  AND t.expected_duration = 30
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'Standard Software Installation' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'Low Risk Firewall Rule Change', description = 'Add or modify access control rules within the approved scope', implementation_plan = 'Verify source, destination, ports, expiry, and business owner; test connectivity after the change.', rollback_plan = 'Remove new rules or restore the original rule snapshot.', justification = 'Meet business application access requirements.', prerequisites = '["Business owner approval","Security policy review approved"]'::jsonb, remarks = 'High risk access changes require the normal change process.'
WHERE t.title = '低风险防火墙规则调整'
  AND t.description = '在已批准范围内新增或调整访问控制规则'
  AND t.implementation_plan = '核对源、目的、端口、有效期和业务负责人，变更后执行连通性验证。'
  AND t.rollback_plan = '删除新增规则或恢复原规则快照。'
  AND t.justification = '满足业务系统访问需求。'
  AND t.affected_cis::jsonb = '["firewall"]'::jsonb
  AND t.prerequisites::jsonb = '["业务负责人确认","安全策略审核通过"]'::jsonb
  AND t.remarks = '高风险开放需走普通变更。'
  AND t.category = '网络'
  AND t.risk_level = 'low'
  AND t.impact_scope = '单业务系统'
  AND t.expected_duration = 45
  AND t.approval_required = true
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'Low Risk Firewall Rule Change' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'Database Account Access Change', description = 'Adjust database account access using least privilege', implementation_plan = 'Verify the account, database and table scope, and access level; apply permissions and record an audit trail.', rollback_plan = 'Revoke added permissions or restore the original access snapshot.', justification = 'Support business operations and data access requirements.', prerequisites = '["Data owner approval","Access scope defined"]'::jsonb, remarks = 'Production write access requires security review.'
WHERE t.title = '数据库账号权限调整'
  AND t.description = '按最小权限原则调整数据库账号权限'
  AND t.implementation_plan = '确认账号、库表范围和权限级别，执行授权并记录审计。'
  AND t.rollback_plan = '回收新增权限或恢复原授权快照。'
  AND t.justification = '支持业务运维和数据访问需求。'
  AND t.affected_cis::jsonb = '["database"]'::jsonb
  AND t.prerequisites::jsonb = '["数据 owner 审批","权限范围明确"]'::jsonb
  AND t.remarks = '生产写权限需安全复核。'
  AND t.category = '数据库'
  AND t.risk_level = 'medium'
  AND t.impact_scope = '指定数据库实例'
  AND t.expected_duration = 30
  AND t.approval_required = true
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'Database Account Access Change' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = 'P1-Urgent Response', description = 'P1 urgent response SLA: respond in 15 minutes and resolve in 60 minutes'
WHERE t.name = 'P1-紧急响应'
  AND t.description = 'P1级别紧急响应SLA，15分钟响应，60分钟解决'
  AND t.priority = 'critical'
  AND t.response_time_minutes = 15
  AND t.resolution_time_minutes = 60
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'P1-Urgent Response' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = 'P2-Standard Response', description = 'P2 standard response SLA: respond in 30 minutes and resolve in 4 hours'
WHERE t.name = 'P2-标准响应'
  AND t.description = 'P2级别标准响应SLA，30分钟响应，4小时解决'
  AND t.priority = 'high'
  AND t.response_time_minutes = 30
  AND t.resolution_time_minutes = 240
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'P2-Standard Response' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = 'P3-Normal Response', description = 'P3 normal response SLA: respond in 2 hours and resolve in 8 hours'
WHERE t.name = 'P3-一般响应'
  AND t.description = 'P3级别一般响应SLA，2小时响应，8小时解决'
  AND t.priority = 'medium'
  AND t.response_time_minutes = 120
  AND t.resolution_time_minutes = 480
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'P3-Normal Response' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Super Administrator', description = 'Full access and cross-tenant initialization'
WHERE t.name = '超级管理员'
  AND t.description = '全部权限，跨租户引导'
  AND t.code = 'super_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Super Administrator' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Tenant Administrator', description = 'Manage all tenant operations and configuration'
WHERE t.name = '租户管理员'
  AND t.description = '租户内业务与配置全量管理'
  AND t.code = 'admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Tenant Administrator' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'System Administrator', description = 'System administration and integration operations'
WHERE t.name = '系统管理员'
  AND t.description = '等保三员①：系统与集成运维'
  AND t.code = 'sysadmin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'System Administrator' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Security Administrator', description = 'Security policies, compliance, and L3 approvals'
WHERE t.name = '安全管理员'
  AND t.description = '等保三员②：安全策略与合规，L3 审批人'
  AND t.code = 'security_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Administrator' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Security Auditor', description = 'Read-only access and audit trails'
WHERE t.name = '安全审计员'
  AND t.description = '等保三员③：全域只读与审计轨迹'
  AND t.code = 'audit_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Auditor' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'IT Administrator', description = 'ITIL process operations and L2 approvals'
WHERE t.name = 'IT管理员'
  AND t.description = 'ITIL 流程运营，L2 审批人'
  AND t.code = 'it_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT Administrator' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Department Manager', description = 'Department and L1 approvals'
WHERE t.name = '部门经理'
  AND t.description = '部门审批与 L1 审批人'
  AND t.code = 'manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Department Manager' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Service Desk Agent', description = 'First-line support and ticket handling'
WHERE t.name = '服务台坐席'
  AND t.description = '一线支持与工单处理'
  AND t.code = 'agent'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Service Desk Agent' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'Technician', description = 'Second-line technical support'
WHERE t.name = '技术员'
  AND t.description = '二线技术处理'
  AND t.code = 'technician'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Technician' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'End User', description = 'Submit service requests and view own tickets'
WHERE t.name = '最终用户'
  AND t.description = '服务请求与查看本人工单'
  AND t.code = 'end_user'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'End User' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Database Issue', description = 'Database performance or connection issues'
WHERE t.name = '数据库问题'
  AND t.description = '数据库性能、连接问题'
  AND t.code = 'database'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Database Issue' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Security Issue', description = 'Security incidents and vulnerabilities'
WHERE t.name = '安全问题'
  AND t.description = '安全事件、漏洞'
  AND t.code = 'security'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security Issue' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Performance Issue', description = 'Slow system response or freezes'
WHERE t.name = '性能问题'
  AND t.description = '系统响应慢、卡顿'
  AND t.code = 'performance'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Performance Issue' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = 'Configuration Issue', description = 'System configuration errors'
WHERE t.name = '配置问题'
  AND t.description = '系统配置错误'
  AND t.code = 'config'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Configuration Issue' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Urgent', description = 'Issues requiring urgent attention'
WHERE t.name = '紧急'
  AND t.description = '紧急处理的问题'
  AND t.color = '#ff4d4f'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Urgent' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Important', description = 'Important but not urgent'
WHERE t.name = '重要'
  AND t.description = '重要但不紧急'
  AND t.color = '#fa8c16'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Important' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET description = 'Software defect'
WHERE t.name = 'bug'
  AND t.description = '程序缺陷'
  AND t.color = '#f5222d';

UPDATE ticket_tags AS t
SET name = 'Feature Request', description = 'New feature request'
WHERE t.name = '功能需求'
  AND t.description = '新功能请求'
  AND t.color = '#1890ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Feature Request' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Performance Issue', description = 'System performance issues'
WHERE t.name = '性能问题'
  AND t.description = '系统性能相关'
  AND t.color = '#722ed1'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Performance Issue' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Security', description = 'Security Issue'
WHERE t.name = '安全'
  AND t.description = '安全问题'
  AND t.color = '#eb2f96'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Security' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Network', description = 'Network issues'
WHERE t.name = '网络'
  AND t.description = '网络相关问题'
  AND t.color = '#13c2c2'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Network' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Database', description = 'Database issues'
WHERE t.name = '数据库'
  AND t.description = '数据库相关问题'
  AND t.color = '#52c41a'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Database' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Awaiting Feedback', description = 'Awaiting user feedback'
WHERE t.name = '待反馈'
  AND t.description = '等待用户反馈'
  AND t.color = '#faad14'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Awaiting Feedback' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Duplicate', description = 'Duplicate issue'
WHERE t.name = '重复'
  AND t.description = '重复问题'
  AND t.color = '#8c8c8c'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Duplicate' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Cannot Reproduce', description = 'Issues that cannot be reproduced'
WHERE t.name = '无法复现'
  AND t.description = '无法复现的问题'
  AND t.color = '#d9d9d9'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Cannot Reproduce' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Resolved', description = 'Resolved issues'
WHERE t.name = '已解决'
  AND t.description = '已解决的问题'
  AND t.color = '#52c41a'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Resolved' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Needs Review', description = 'Supervisor review required'
WHERE t.name = '需要审核'
  AND t.description = '需要上级审核'
  AND t.color = '#1677ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Needs Review' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'High Availability', description = 'High availability issues'
WHERE t.name = '高可用'
  AND t.description = '高可用相关'
  AND t.color = '#fa541c'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'High Availability' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'Monitoring Alert', description = 'Monitoring and alerting'
WHERE t.name = '监控告警'
  AND t.description = '监控和告警相关'
  AND t.color = '#fa8c16'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Monitoring Alert' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'Server Restart', description = 'Standard server restart for routine maintenance', implementation_plan = '1. Notify affected users
2. Stop services
3. Restart the server
4. Verify service recovery', rollback_plan = 'If the restart fails, restore the previous state immediately', justification = 'Routine maintenance', affected_cis = '["Server"]'::jsonb, prerequisites = '["Notify users in advance","Back up important data"]'::jsonb, remarks = 'For non-critical servers only'
WHERE t.title = '服务器重启'
  AND t.description = '标准服务器重启流程，用于常规维护'
  AND t.implementation_plan = '1. 通知相关用户
2. 停止服务
3. 重启服务器
4. 验证服务恢复'
  AND t.rollback_plan = '如果重启失败，立即回滚到重启前状态'
  AND t.justification = '例行维护'
  AND t.affected_cis::jsonb = '["服务器"]'::jsonb
  AND t.prerequisites::jsonb = '["提前通知用户","备份重要数据"]'::jsonb
  AND t.remarks = '仅适用于非关键业务服务器'
  AND t.category = '服务器'
  AND t.risk_level = 'low'
  AND t.impact_scope = 'low'
  AND t.expected_duration = 30
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'Server Restart' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'SSL Certificate Renewal', description = 'Renew an expiring SSL certificate', implementation_plan = '1. Obtain a new certificate
2. Validate in a test environment
3. Deploy to production
4. Verify the certificate', rollback_plan = 'Keep the previous certificate for immediate rollback if needed', justification = 'The certificate is expiring and must be renewed', affected_cis = '["Load Balancer","Web Server"]'::jsonb, prerequisites = '["New certificate requested","Obtain the certificate files"]'::jsonb
WHERE t.title = 'SSL证书更新'
  AND t.description = '更新即将过期的SSL证书'
  AND t.implementation_plan = '1. 申请新证书
2. 在测试环境验证
3. 生产环境部署
4. 验证证书生效'
  AND t.rollback_plan = '保留旧证书，发现问题可立即回滚'
  AND t.justification = '证书即将过期，必须更新'
  AND t.affected_cis::jsonb = '["负载均衡器","Web服务器"]'::jsonb
  AND t.prerequisites::jsonb = '["新证书已申请","获取证书文件"]'::jsonb
  AND t.remarks = ''
  AND t.category = '安全'
  AND t.risk_level = 'low'
  AND t.impact_scope = 'low'
  AND t.expected_duration = 60
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'SSL Certificate Renewal' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'Database Backup', description = 'Perform a full database backup', implementation_plan = '1. Pause database writes
2. Perform a full backup
3. Verify backup integrity
4. Restore database service', rollback_plan = 'Cancel the operation if the backup fails', justification = 'Data protection requirements', affected_cis = '["Database Server"]'::jsonb, prerequisites = '["Confirm sufficient backup storage","Check backup tool availability"]'::jsonb
WHERE t.title = '数据库备份'
  AND t.description = '执行数据库全量备份'
  AND t.implementation_plan = '1. 停止数据库写入
2. 执行全量备份
3. 验证备份完整性
4. 恢复数据库服务'
  AND t.rollback_plan = '备份失败时取消备份操作'
  AND t.justification = '数据安全要求'
  AND t.affected_cis::jsonb = '["数据库服务器"]'::jsonb
  AND t.prerequisites::jsonb = '["确认备份存储空间充足","检查备份工具可用性"]'::jsonb
  AND t.remarks = ''
  AND t.category = '数据库'
  AND t.risk_level = 'low'
  AND t.impact_scope = 'medium'
  AND t.expected_duration = 120
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'Database Backup' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'Add Firewall Rule', description = 'Add an allow rule to the firewall', implementation_plan = '1. Prepare the rule change request
2. Validate in a test environment
3. Apply the rule in production
4. Monitor network traffic', rollback_plan = 'Remove the new rule immediately if issues occur', justification = 'Business requirements need a new port', affected_cis = '["Firewall","Network Switch"]'::jsonb, prerequisites = '["Security assessment completed","Business department approval obtained"]'::jsonb, remarks = 'Security team approval required'
WHERE t.title = '防火墙规则添加'
  AND t.description = '添加新的防火墙放行规则'
  AND t.implementation_plan = '1. 准备规则变更申请
2. 在测试环境验证
3. 生产环境应用新规则
4. 监控网络流量'
  AND t.rollback_plan = '发现异常时立即删除新添加的规则'
  AND t.justification = '业务需要开放新端口'
  AND t.affected_cis::jsonb = '["防火墙","网络交换机"]'::jsonb
  AND t.prerequisites::jsonb = '["已完成安全评估","相关业务部门确认"]'::jsonb
  AND t.remarks = '需安全部门审批'
  AND t.category = '网络安全'
  AND t.risk_level = 'medium'
  AND t.impact_scope = 'medium'
  AND t.expected_duration = 45
  AND t.approval_required = true
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'Add Firewall Rule' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'Application Configuration Update', description = 'Update application configuration parameters', implementation_plan = '1. Back up the configuration
2. Update parameters
3. Restart the application
4. Verify functionality', rollback_plan = 'Restore the backed-up configuration file', justification = 'Optimize system performance', affected_cis = '["Application Server"]'::jsonb, prerequisites = '["New configuration tested successfully"]'::jsonb
WHERE t.title = '应用配置更新'
  AND t.description = '更新应用程序配置文件中的参数'
  AND t.implementation_plan = '1. 备份当前配置
2. 修改配置参数
3. 重启应用服务
4. 验证功能正常'
  AND t.rollback_plan = '回滚到备份的配置文件'
  AND t.justification = '优化系统性能'
  AND t.affected_cis::jsonb = '["应用服务器"]'::jsonb
  AND t.prerequisites::jsonb = '["新配置已测试通过"]'::jsonb
  AND t.remarks = ''
  AND t.category = '应用'
  AND t.risk_level = 'low'
  AND t.impact_scope = 'low'
  AND t.expected_duration = 30
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'Application Configuration Update' AND sibling.id <> t.id);

UPDATE marketplace_items AS t
SET title = 'Feishu Approval Connector', description = 'Integrate ITSM tickets with Feishu approvals, including bidirectional approval synchronization and message card callbacks', long_description = 'Native Feishu integration for approval creation, status callbacks, message cards, and user identity mapping. Supports change and service request approvals from Feishu.', tags = '["Feishu","Approval","IM","OA"]'::jsonb
WHERE t.title = '飞书审批增强连接器'
  AND t.description = '将 ITSM 工单与飞书审批原生打通，支持审批实例双向同步、消息卡片回调'
  AND t.long_description = '飞书审批增强连接器提供与飞书开放平台的原生集成：审批实例创建、状态回调、消息卡片推送、用户身份映射。可用于变更请求、服务请求等场景的飞书端审批入口。'
  AND t.tags::jsonb = '["飞书","审批","IM","OA"]'::jsonb
  AND t.name = 'feishu-connector'
  AND t.type = 'connector'
  AND t.provider = 'ITSM Foundation'
  AND t.latest_version = '1.2.0';

UPDATE marketplace_items AS t
SET title = 'DingTalk AI Ticket Assistant', description = 'An AI ticket assistant for DingTalk group chats, supporting ticket creation, status queries, and reminders', long_description = 'A skill for operations and support chats that identifies ticket intent and uses ITSM APIs to create tickets, check status, and send reminders.', tags = '["DingTalk","AI","skill","Ticket"]'::jsonb
WHERE t.title = '钉钉 AI 工单助手'
  AND t.description = '钉钉群聊内的 AI 工单助理，支持自然语言创建工单、查询状态、催办'
  AND t.long_description = '钉钉 AI 工单助手是面向运维/客服群聊的 Skill：识别用户文本中的工单意图，调用 ITSM API 创建/查询/催办；支持「帮我开个 P2 工单」「这个工单处理到哪一步了」「催一下张三」等表达工单操作。'
  AND t.tags::jsonb = '["钉钉","AI","skill","工单"]'::jsonb
  AND t.name = 'dingtalk-ai-assistant'
  AND t.type = 'skill'
  AND t.provider = 'ITSM Foundation'
  AND t.latest_version = '0.9.1';

UPDATE marketplace_items AS t
SET title = 'WeCom Connector', description = 'Integrate WeCom contacts, approvals, and application messages for internal approvals and alerts', long_description = 'Synchronize WeCom contacts, send application messages, and receive approval callbacks while reusing existing identity and approval systems.', tags = '["WeCom","Contacts","Approval"]'::jsonb
WHERE t.title = '企业微信连接器'
  AND t.description = '与企微通讯录、审批、应用消息集成，承接内网审批与告警推送'
  AND t.long_description = '企业微信连接器提供通讯录同步、自建应用消息推送、审批回调等能力，适合与企微深度绑定的客户在 IT 服务台场景中复用现有账号体系与审批流。'
  AND t.tags::jsonb = '["企业微信","通讯录","审批"]'::jsonb
  AND t.name = 'wecom-connector'
  AND t.type = 'connector'
  AND t.provider = 'ITSM Foundation'
  AND t.latest_version = '1.0.3';

UPDATE marketplace_items AS t
SET title = 'AI Ticket Classification and Priority Suggestions', description = 'An LLM skill for ticket classification, priority recommendations, and tag suggestions', long_description = 'Runs asynchronously after ticket creation and uses knowledge and ticket history to suggest classifications, priorities, and tags. Recommendations above the configured confidence threshold are accepted; others require human confirmation.', tags = '["AI","skill","Classification","triage"]'::jsonb
WHERE t.title = 'AI 工单分类与优先级建议'
  AND t.description = '基于 LLM 的工单自动分类、优先级推荐、标签补全 Skill'
  AND t.long_description = 'AI 工单分类 Skill 在工单创建后异步运行，结合知识库与历史工单给出分类、优先级、标签建议，置信度高于阈值时直接接受，否则回到人工确认流程。'
  AND t.tags::jsonb = '["AI","skill","分类","triage"]'::jsonb
  AND t.name = 'ai-ticket-classify'
  AND t.type = 'skill'
  AND t.provider = 'ITSM Foundation'
  AND t.latest_version = '1.4.0';

UPDATE marketplace_items AS t
SET title = 'Jira Bidirectional Sync Plugin', description = 'Synchronize Jira issues and ITSM tickets, including status, comments, and attachments', long_description = 'Connect Jira projects and issues with configurable field mappings, scheduled or event-driven synchronization, and bidirectional status, comment, and attachment updates.'
WHERE t.title = 'Jira 双向同步插件'
  AND t.description = 'Jira Issue 与 ITSM 工单双向同步，状态/评论/附件双向同步'
  AND t.long_description = 'Jira 同步插件把 Jira Project/Issue 作为外部系统接入：可配置字段映射、定时同步或事件触发同步，处理 Issue 与 ITSM 工单之间的状态/评论/附件双向同步。'
  AND t.tags::jsonb = '["Jira","plugin","sync"]'::jsonb
  AND t.name = 'jira-sync-plugin'
  AND t.type = 'plugin'
  AND t.provider = 'Community'
  AND t.latest_version = '0.8.0';

UPDATE sla_policies AS t
SET name = 'Default P1 Incident SLA', description = 'High priority incident response and resolution policy'
WHERE t.name = '默认P1事件SLA'
  AND t.description = '高优先级事件响应与解决策略'
  AND t.priority = 'high'
  AND t.response_time_minutes = 30
  AND t.resolution_time_minutes = 240
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Default P1 Incident SLA' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = 'Default P2 Incident SLA', description = 'Medium priority incident response and resolution policy'
WHERE t.name = '默认P2事件SLA'
  AND t.description = '中优先级事件响应与解决策略'
  AND t.priority = 'medium'
  AND t.response_time_minutes = 120
  AND t.resolution_time_minutes = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Default P2 Incident SLA' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = 'Default Service Request SLA', description = 'Standard service request fulfillment policy'
WHERE t.name = '默认服务请求SLA'
  AND t.description = '标准服务请求履约策略'
  AND t.priority = 'low'
  AND t.response_time_minutes = 240
  AND t.resolution_time_minutes = 2880
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Default Service Request SLA' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Kubernetes Scaling', description = 'Request to scale a Kubernetes cluster up or down'
WHERE t.name = 'K8S扩缩容'
  AND t.description = 'Kubernetes容器集群扩容或缩容请求'
  AND t.code = 'k8s_scale'
  AND t.icon = 'Container'
  AND t.color = '#1890ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Kubernetes Scaling' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'DDL Execution', description = 'Database schema changes, index creation, and other DDL operations'
WHERE t.name = 'DDL执行'
  AND t.description = '数据库表结构变更、索引创建等DDL操作'
  AND t.code = 'ddl_execute'
  AND t.icon = 'Database'
  AND t.color = '#722ed1'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'DDL Execution' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Data Export', description = 'Export data from a database or system'
WHERE t.name = '数据导出'
  AND t.description = '从数据库或系统导出数据'
  AND t.code = 'data_export'
  AND t.icon = 'Download'
  AND t.color = '#13c2c2'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Data Export' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Virtual Machine Request', description = 'Request new virtual machine resources'
WHERE t.name = '虚拟机申请'
  AND t.description = '申请新的虚拟机资源'
  AND t.code = 'vm_apply'
  AND t.icon = 'Desktop'
  AND t.color = '#2f54eb'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Virtual Machine Request' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Account Request', description = 'Request system, VPN, or bastion host accounts'
WHERE t.name = '账号申请'
  AND t.description = '申请系统账号、VPN账号、堡垒机账号等'
  AND t.code = 'account_apply'
  AND t.icon = 'User'
  AND t.color = '#52c41a'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Account Request' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'GitLab Repository Request', description = 'Request a new GitLab repository'
WHERE t.name = 'GitLab代码仓库申请'
  AND t.description = '申请创建新的GitLab代码仓库'
  AND t.code = 'gitlab_repo_apply'
  AND t.icon = 'Code'
  AND t.color = '#fa541c'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'GitLab Repository Request' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Domain Request', description = 'Request a domain or DNS change'
WHERE t.name = '域名申请'
  AND t.description = '申请新的域名或域名解析变更'
  AND t.code = 'domain_apply'
  AND t.icon = 'Global'
  AND t.color = '#eb2f96'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Domain Request' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Firewall Rule Request', description = 'Request new or modified firewall port rules'
WHERE t.name = '防火墙规则申请'
  AND t.description = '申请开放或变更防火墙端口规则'
  AND t.code = 'firewall_apply'
  AND t.icon = 'Safety'
  AND t.color = '#fa8c16'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Firewall Rule Request' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Application Request', description = 'Request deployment of an application in Kubernetes'
WHERE t.name = '应用申请'
  AND t.description = '申请在K8S集群中部署新应用服务'
  AND t.code = 'app_apply'
  AND t.icon = 'Appstore'
  AND t.color = '#1890ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Application Request' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Project Request', description = 'Request a new project or workspace'
WHERE t.name = '项目申请'
  AND t.description = '申请创建新项目或项目空间'
  AND t.code = 'project_apply'
  AND t.icon = 'Project'
  AND t.color = '#722ed1'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Project Request' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'Database Account Request', description = 'Request database read/write or read-only accounts'
WHERE t.name = '数据库账号申请'
  AND t.description = '申请数据库读写账号、只读账号等'
  AND t.code = 'db_account_apply'
  AND t.icon = 'Key'
  AND t.color = '#faad14'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Database Account Request' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'General Ticket', description = 'General ticket type for other requests'
WHERE t.name = '其他工单'
  AND t.description = '通用工单类型，用于不属于以上分类的请求'
  AND t.code = 'general'
  AND t.icon = 'FileText'
  AND t.color = '#8c8c8c'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'General Ticket' AND sibling.id <> t.id);

UPDATE users AS t
SET name = 'System Administrator', department = 'IT Department'
WHERE t.username = 'admin'
  AND t.email = 'admin@example.com'
  AND t.name = '系统管理员'
  AND t.department = 'IT部门'
  AND t.role = 'super_admin'
  AND t.tenant_id IN (SELECT id FROM tenants WHERE code = 'default');

UPDATE sla_alert_rules AS r SET name = 'SLA-P0-Urgent-Response Alert'
WHERE r.name = 'SLA-P0-紧急-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P0-紧急', 'SLA-P0-Urgent'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P0-Urgent-Resolution Alert'
WHERE r.name = 'SLA-P0-紧急-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P0-紧急', 'SLA-P0-Urgent'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P1-High-Response Alert'
WHERE r.name = 'SLA-P1-高-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P1-高', 'SLA-P1-High'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P1-High-Resolution Alert'
WHERE r.name = 'SLA-P1-高-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P1-高', 'SLA-P1-High'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P2-Medium-Response Alert'
WHERE r.name = 'SLA-P2-中-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P2-中', 'SLA-P2-Medium'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P2-Medium-Resolution Alert'
WHERE r.name = 'SLA-P2-中-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P2-中', 'SLA-P2-Medium'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P3-Low-Response Alert'
WHERE r.name = 'SLA-P3-低-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P3-低', 'SLA-P3-Low'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P3-Low-Resolution Alert'
WHERE r.name = 'SLA-P3-低-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P3-低', 'SLA-P3-Low'));

UPDATE sla_alert_rules AS r SET name = 'SLA-Service Request-Response Alert'
WHERE r.name = 'SLA-服务请求-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-服务请求', 'SLA-Service Request'));

UPDATE sla_alert_rules AS r SET name = 'SLA-Service Request-Resolution Alert'
WHERE r.name = 'SLA-服务请求-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-服务请求', 'SLA-Service Request'));

UPDATE sla_alert_rules AS r SET name = 'SLA-Change-Response Alert'
WHERE r.name = 'SLA-变更-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-变更', 'SLA-Change'));

UPDATE sla_alert_rules AS r SET name = 'SLA-Change-Resolution Alert'
WHERE r.name = 'SLA-变更-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-变更', 'SLA-Change'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P0-Urgent-Response Alert'
WHERE r.name = 'Incident-P0-紧急-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P0-紧急', 'Incident-P0-Urgent'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P0-Urgent-Resolution Alert'
WHERE r.name = 'Incident-P0-紧急-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P0-紧急', 'Incident-P0-Urgent'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P1-High-Response Alert'
WHERE r.name = 'Incident-P1-高-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P1-高', 'Incident-P1-High'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P1-High-Resolution Alert'
WHERE r.name = 'Incident-P1-高-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P1-高', 'Incident-P1-High'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P2-Medium-Response Alert'
WHERE r.name = 'Incident-P2-中-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P2-中', 'Incident-P2-Medium'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P2-Medium-Resolution Alert'
WHERE r.name = 'Incident-P2-中-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P2-中', 'Incident-P2-Medium'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P3-Low-Response Alert'
WHERE r.name = 'Incident-P3-低-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P3-低', 'Incident-P3-Low'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P3-Low-Resolution Alert'
WHERE r.name = 'Incident-P3-低-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P3-低', 'Incident-P3-Low'));

UPDATE sla_alert_rules AS r SET name = 'ServiceRequest-Standard-Response Alert'
WHERE r.name = 'ServiceRequest-标准-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('ServiceRequest-标准', 'ServiceRequest-Standard'));

UPDATE sla_alert_rules AS r SET name = 'ServiceRequest-Standard-Resolution Alert'
WHERE r.name = 'ServiceRequest-标准-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('ServiceRequest-标准', 'ServiceRequest-Standard'));

UPDATE sla_alert_rules AS r SET name = 'Change-Normal-Response Alert'
WHERE r.name = 'Change-普通-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Change-普通', 'Change-Normal'));

UPDATE sla_alert_rules AS r SET name = 'Change-Normal-Resolution Alert'
WHERE r.name = 'Change-普通-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Change-普通', 'Change-Normal'));

UPDATE sla_alert_rules AS r SET name = 'Change-Emergency-Response Alert'
WHERE r.name = 'Change-紧急-响应告警' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Change-紧急', 'Change-Emergency'));

UPDATE sla_alert_rules AS r SET name = 'Change-Emergency-Resolution Alert'
WHERE r.name = 'Change-紧急-解决告警' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Change-紧急', 'Change-Emergency'));

-- Fixed configuration defaults and template-installed SLA display text.
UPDATE system_configs AS t SET description = 'System Name'
WHERE t.key = 'systemName' AND t.deleted_at IS NULL AND t.description = '系统名称';

UPDATE system_configs AS t SET value = 'ITSM System'
WHERE t.key = 'systemName' AND t.deleted_at IS NULL AND t.value = 'ITSM系统';

UPDATE system_configs AS t SET description = 'System URL'
WHERE t.key = 'systemUrl' AND t.deleted_at IS NULL AND t.description = '系统URL';

UPDATE system_configs AS t SET description = 'Time Zone'
WHERE t.key = 'timezone' AND t.deleted_at IS NULL AND t.description = '时区';

UPDATE system_configs AS t SET description = 'Language'
WHERE t.key = 'language' AND t.deleted_at IS NULL AND t.description = '语言';

UPDATE system_configs AS t SET description = 'Date Format'
WHERE t.key = 'dateFormat' AND t.deleted_at IS NULL AND t.description = '日期格式';

UPDATE system_configs AS t SET description = 'Time Format'
WHERE t.key = 'timeFormat' AND t.deleted_at IS NULL AND t.description = '时间格式';

UPDATE system_configs AS t SET description = 'Session Timeout (minutes)'
WHERE t.key = 'sessionTimeout' AND t.deleted_at IS NULL AND t.description = '会话超时时间(分钟)';

UPDATE system_configs AS t SET description = 'Maximum File Size (MB)'
WHERE t.key = 'maxFileSize' AND t.deleted_at IS NULL AND t.description = '最大文件大小(MB)';

UPDATE system_configs AS t SET description = 'Allowed File Types'
WHERE t.key = 'allowedFileTypes' AND t.deleted_at IS NULL AND t.description = '允许的文件类型';

UPDATE system_configs AS t SET description = 'Minimum Password Length'
WHERE t.key = 'passwordMinLength' AND t.deleted_at IS NULL AND t.description = '密码最小长度';

UPDATE system_configs AS t SET description = 'Require Uppercase Letters'
WHERE t.key = 'passwordRequireUppercase' AND t.deleted_at IS NULL AND t.description = '需要大写字母';

UPDATE system_configs AS t SET description = 'Require Lowercase Letters'
WHERE t.key = 'passwordRequireLowercase' AND t.deleted_at IS NULL AND t.description = '需要小写字母';

UPDATE system_configs AS t SET description = 'Require Numbers'
WHERE t.key = 'passwordRequireNumbers' AND t.deleted_at IS NULL AND t.description = '需要数字';

UPDATE system_configs AS t SET description = 'Require Special Characters'
WHERE t.key = 'passwordRequireSpecialChars' AND t.deleted_at IS NULL AND t.description = '需要特殊字符';

UPDATE system_configs AS t SET description = 'Maximum Failed Login Attempts'
WHERE t.key = 'loginMaxAttempts' AND t.deleted_at IS NULL AND t.description = '登录失败次数限制';

UPDATE system_configs AS t SET description = 'Account Lockout Duration (minutes)'
WHERE t.key = 'accountLockoutDuration' AND t.deleted_at IS NULL AND t.description = '账户锁定时间(分钟)';

UPDATE system_configs AS t SET description = 'Enable Two-Factor Authentication'
WHERE t.key = 'enable2FA' AND t.deleted_at IS NULL AND t.description = '启用双因素认证';

UPDATE system_configs AS t SET description = 'SMTP Server'
WHERE t.key = 'smtpHost' AND t.deleted_at IS NULL AND t.description = 'SMTP服务器';

UPDATE system_configs AS t SET description = 'SMTP Port'
WHERE t.key = 'smtpPort' AND t.deleted_at IS NULL AND t.description = 'SMTP端口';

UPDATE system_configs AS t SET description = 'SMTP Username'
WHERE t.key = 'smtpUsername' AND t.deleted_at IS NULL AND t.description = 'SMTP用户名';

UPDATE system_configs AS t SET description = 'SMTP Password'
WHERE t.key = 'smtpPassword' AND t.deleted_at IS NULL AND t.description = 'SMTP密码';

UPDATE system_configs AS t SET description = 'Enable SSL/TLS'
WHERE t.key = 'smtpEnableSSL' AND t.deleted_at IS NULL AND t.description = '启用SSL/TLS';

UPDATE system_configs AS t SET description = 'Sender Email Address'
WHERE t.key = 'emailFrom' AND t.deleted_at IS NULL AND t.description = '发件人邮箱';

UPDATE system_configs AS t SET description = 'System Notification Template'
WHERE t.key = 'systemNotificationTemplate' AND t.deleted_at IS NULL AND t.description = '系统通知模板';

UPDATE system_configs AS t SET value = 'Hello {username},
You have a new system notification: {message}
— ITSM Team'
WHERE t.key = 'systemNotificationTemplate' AND t.deleted_at IS NULL AND t.value = '您好 {username}：
您有一条新的系统通知：{message}
— ITSM 团队';

UPDATE users AS t SET name = 'System Administrator'
WHERE t.username = 'admin' AND t.is_bootstrap_admin = true AND t.name = '系统管理员';

UPDATE users AS t SET name = 'System Administrator (emergency)'
WHERE t.username = 'admin' AND t.is_bootstrap_admin = true AND t.name = '系统管理员 (emergency)';

UPDATE users AS t SET department = 'IT Department'
WHERE t.username = 'admin' AND t.is_bootstrap_admin = true AND t.department = 'IT部门';

UPDATE sla_definitions AS t SET name = '[Template] Incident P1 Critical'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p1_critical' AND t.name = '[Template] 事件 P1 紧急';

UPDATE sla_definitions AS t SET description = '[Template] incident_p1_critical - For critical incidents with a complete business outage: respond within 15 minutes and resolve within 4 hours'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p1_critical' AND t.description = '[Template] incident_p1_critical - 适用于业务完全中断的紧急事件，要求 15 分钟内响应、4 小时内解决';

UPDATE sla_definitions AS t SET name = '[Template] Incident P2 High'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p2_high' AND t.name = '[Template] 事件 P2 高';

UPDATE sla_definitions AS t SET description = '[Template] incident_p2_high - For high-priority incidents affecting some users: respond within 1 hour and resolve within 8 hours'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p2_high' AND t.description = '[Template] incident_p2_high - 适用于影响部分用户的高优先级事件，要求 1 小时内响应、8 小时内解决';

UPDATE sla_definitions AS t SET name = '[Template] Incident P3 Medium'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p3_medium' AND t.name = '[Template] 事件 P3 中';

UPDATE sla_definitions AS t SET description = '[Template] incident_p3_medium - For medium-priority incidents affecting a few users: respond within 4 hours and resolve within 24 hours'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p3_medium' AND t.description = '[Template] incident_p3_medium - 适用于影响少数用户的中优先级事件，要求 4 小时内响应、24 小时内解决';

UPDATE sla_definitions AS t SET name = '[Template] Change - Normal'
WHERE t.conditions::jsonb ->> 'templateKey' = 'change_normal' AND t.name = '[Template] 变更 - 普通';

UPDATE sla_definitions AS t SET description = '[Template] change_normal - For low-risk routine changes: approve within 4 hours and implement within 24 hours'
WHERE t.conditions::jsonb ->> 'templateKey' = 'change_normal' AND t.description = '[Template] change_normal - 适用于风险较低的常规变更，要求 4 小时内审批、24 小时内实施';

UPDATE sla_definitions AS t SET name = '[Template] Change - Emergency'
WHERE t.conditions::jsonb ->> 'templateKey' = 'change_emergency' AND t.name = '[Template] 变更 - 紧急';

UPDATE sla_definitions AS t SET description = '[Template] change_emergency - For emergency fixes: approve within 30 minutes and implement within 4 hours'
WHERE t.conditions::jsonb ->> 'templateKey' = 'change_emergency' AND t.description = '[Template] change_emergency - 适用于紧急修复变更，要求 30 分钟内审批、4 小时内实施';

UPDATE sla_definitions AS t SET name = '[Template] Service Request - Standard'
WHERE t.conditions::jsonb ->> 'templateKey' = 'service_request_standard' AND t.name = '[Template] 服务请求 - 标准';

UPDATE sla_definitions AS t SET description = '[Template] service_request_standard - For standard service requests such as account creation and access requests: respond within 8 hours and resolve within 72 hours'
WHERE t.conditions::jsonb ->> 'templateKey' = 'service_request_standard' AND t.description = '[Template] service_request_standard - 适用于标准服务请求（账号开通、权限申请等），要求 8 小时内响应、72 小时内解决';
