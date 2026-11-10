UPDATE sla_definitions AS t SET description = '[Template] service_request_standard - 适用于标准服务请求（账号开通、权限申请等），要求 8 小时内响应、72 小时内解决'
WHERE t.conditions::jsonb ->> 'templateKey' = 'service_request_standard' AND t.description = '[Template] service_request_standard - For standard service requests such as account creation and access requests: respond within 8 hours and resolve within 72 hours';

UPDATE sla_definitions AS t SET name = '[Template] 服务请求 - 标准'
WHERE t.conditions::jsonb ->> 'templateKey' = 'service_request_standard' AND t.name = '[Template] Service Request - Standard';

UPDATE sla_definitions AS t SET description = '[Template] change_emergency - 适用于紧急修复变更，要求 30 分钟内审批、4 小时内实施'
WHERE t.conditions::jsonb ->> 'templateKey' = 'change_emergency' AND t.description = '[Template] change_emergency - For emergency fixes: approve within 30 minutes and implement within 4 hours';

UPDATE sla_definitions AS t SET name = '[Template] 变更 - 紧急'
WHERE t.conditions::jsonb ->> 'templateKey' = 'change_emergency' AND t.name = '[Template] Change - Emergency';

UPDATE sla_definitions AS t SET description = '[Template] change_normal - 适用于风险较低的常规变更，要求 4 小时内审批、24 小时内实施'
WHERE t.conditions::jsonb ->> 'templateKey' = 'change_normal' AND t.description = '[Template] change_normal - For low-risk routine changes: approve within 4 hours and implement within 24 hours';

UPDATE sla_definitions AS t SET name = '[Template] 变更 - 普通'
WHERE t.conditions::jsonb ->> 'templateKey' = 'change_normal' AND t.name = '[Template] Change - Normal';

UPDATE sla_definitions AS t SET description = '[Template] incident_p3_medium - 适用于影响少数用户的中优先级事件，要求 4 小时内响应、24 小时内解决'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p3_medium' AND t.description = '[Template] incident_p3_medium - For medium-priority incidents affecting a few users: respond within 4 hours and resolve within 24 hours';

UPDATE sla_definitions AS t SET name = '[Template] 事件 P3 中'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p3_medium' AND t.name = '[Template] Incident P3 Medium';

UPDATE sla_definitions AS t SET description = '[Template] incident_p2_high - 适用于影响部分用户的高优先级事件，要求 1 小时内响应、8 小时内解决'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p2_high' AND t.description = '[Template] incident_p2_high - For high-priority incidents affecting some users: respond within 1 hour and resolve within 8 hours';

UPDATE sla_definitions AS t SET name = '[Template] 事件 P2 高'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p2_high' AND t.name = '[Template] Incident P2 High';

UPDATE sla_definitions AS t SET description = '[Template] incident_p1_critical - 适用于业务完全中断的紧急事件，要求 15 分钟内响应、4 小时内解决'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p1_critical' AND t.description = '[Template] incident_p1_critical - For critical incidents with a complete business outage: respond within 15 minutes and resolve within 4 hours';

UPDATE sla_definitions AS t SET name = '[Template] 事件 P1 紧急'
WHERE t.conditions::jsonb ->> 'templateKey' = 'incident_p1_critical' AND t.name = '[Template] Incident P1 Critical';

UPDATE users AS t SET department = 'IT部门'
WHERE t.username = 'admin' AND t.is_bootstrap_admin = true AND t.department = 'IT Department';

UPDATE users AS t SET name = '系统管理员 (emergency)'
WHERE t.username = 'admin' AND t.is_bootstrap_admin = true AND t.name = 'System Administrator (emergency)';

UPDATE users AS t SET name = '系统管理员'
WHERE t.username = 'admin' AND t.is_bootstrap_admin = true AND t.name = 'System Administrator';

UPDATE system_configs AS t SET value = '您好 {username}：
您有一条新的系统通知：{message}
— ITSM 团队'
WHERE t.key = 'systemNotificationTemplate' AND t.deleted_at IS NULL AND t.value = 'Hello {username},
You have a new system notification: {message}
— ITSM Team';

UPDATE system_configs AS t SET description = '系统通知模板'
WHERE t.key = 'systemNotificationTemplate' AND t.deleted_at IS NULL AND t.description = 'System Notification Template';

UPDATE system_configs AS t SET description = '发件人邮箱'
WHERE t.key = 'emailFrom' AND t.deleted_at IS NULL AND t.description = 'Sender Email Address';

UPDATE system_configs AS t SET description = '启用SSL/TLS'
WHERE t.key = 'smtpEnableSSL' AND t.deleted_at IS NULL AND t.description = 'Enable SSL/TLS';

UPDATE system_configs AS t SET description = 'SMTP密码'
WHERE t.key = 'smtpPassword' AND t.deleted_at IS NULL AND t.description = 'SMTP Password';

UPDATE system_configs AS t SET description = 'SMTP用户名'
WHERE t.key = 'smtpUsername' AND t.deleted_at IS NULL AND t.description = 'SMTP Username';

UPDATE system_configs AS t SET description = 'SMTP端口'
WHERE t.key = 'smtpPort' AND t.deleted_at IS NULL AND t.description = 'SMTP Port';

UPDATE system_configs AS t SET description = 'SMTP服务器'
WHERE t.key = 'smtpHost' AND t.deleted_at IS NULL AND t.description = 'SMTP Server';

UPDATE system_configs AS t SET description = '启用双因素认证'
WHERE t.key = 'enable2FA' AND t.deleted_at IS NULL AND t.description = 'Enable Two-Factor Authentication';

UPDATE system_configs AS t SET description = '账户锁定时间(分钟)'
WHERE t.key = 'accountLockoutDuration' AND t.deleted_at IS NULL AND t.description = 'Account Lockout Duration (minutes)';

UPDATE system_configs AS t SET description = '登录失败次数限制'
WHERE t.key = 'loginMaxAttempts' AND t.deleted_at IS NULL AND t.description = 'Maximum Failed Login Attempts';

UPDATE system_configs AS t SET description = '需要特殊字符'
WHERE t.key = 'passwordRequireSpecialChars' AND t.deleted_at IS NULL AND t.description = 'Require Special Characters';

UPDATE system_configs AS t SET description = '需要数字'
WHERE t.key = 'passwordRequireNumbers' AND t.deleted_at IS NULL AND t.description = 'Require Numbers';

UPDATE system_configs AS t SET description = '需要小写字母'
WHERE t.key = 'passwordRequireLowercase' AND t.deleted_at IS NULL AND t.description = 'Require Lowercase Letters';

UPDATE system_configs AS t SET description = '需要大写字母'
WHERE t.key = 'passwordRequireUppercase' AND t.deleted_at IS NULL AND t.description = 'Require Uppercase Letters';

UPDATE system_configs AS t SET description = '密码最小长度'
WHERE t.key = 'passwordMinLength' AND t.deleted_at IS NULL AND t.description = 'Minimum Password Length';

UPDATE system_configs AS t SET description = '允许的文件类型'
WHERE t.key = 'allowedFileTypes' AND t.deleted_at IS NULL AND t.description = 'Allowed File Types';

UPDATE system_configs AS t SET description = '最大文件大小(MB)'
WHERE t.key = 'maxFileSize' AND t.deleted_at IS NULL AND t.description = 'Maximum File Size (MB)';

UPDATE system_configs AS t SET description = '会话超时时间(分钟)'
WHERE t.key = 'sessionTimeout' AND t.deleted_at IS NULL AND t.description = 'Session Timeout (minutes)';

UPDATE system_configs AS t SET description = '时间格式'
WHERE t.key = 'timeFormat' AND t.deleted_at IS NULL AND t.description = 'Time Format';

UPDATE system_configs AS t SET description = '日期格式'
WHERE t.key = 'dateFormat' AND t.deleted_at IS NULL AND t.description = 'Date Format';

UPDATE system_configs AS t SET description = '语言'
WHERE t.key = 'language' AND t.deleted_at IS NULL AND t.description = 'Language';

UPDATE system_configs AS t SET description = '时区'
WHERE t.key = 'timezone' AND t.deleted_at IS NULL AND t.description = 'Time Zone';

UPDATE system_configs AS t SET description = '系统URL'
WHERE t.key = 'systemUrl' AND t.deleted_at IS NULL AND t.description = 'System URL';

UPDATE system_configs AS t SET value = 'ITSM系统'
WHERE t.key = 'systemName' AND t.deleted_at IS NULL AND t.value = 'ITSM System';

UPDATE system_configs AS t SET description = '系统名称'
WHERE t.key = 'systemName' AND t.deleted_at IS NULL AND t.description = 'System Name';

UPDATE sla_alert_rules AS r SET name = 'Change-紧急-解决告警'
WHERE r.name = 'Change-Emergency-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Change-紧急', 'Change-Emergency'));

UPDATE sla_alert_rules AS r SET name = 'Change-紧急-响应告警'
WHERE r.name = 'Change-Emergency-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Change-紧急', 'Change-Emergency'));

UPDATE sla_alert_rules AS r SET name = 'Change-普通-解决告警'
WHERE r.name = 'Change-Normal-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Change-普通', 'Change-Normal'));

UPDATE sla_alert_rules AS r SET name = 'Change-普通-响应告警'
WHERE r.name = 'Change-Normal-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Change-普通', 'Change-Normal'));

UPDATE sla_alert_rules AS r SET name = 'ServiceRequest-标准-解决告警'
WHERE r.name = 'ServiceRequest-Standard-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('ServiceRequest-标准', 'ServiceRequest-Standard'));

UPDATE sla_alert_rules AS r SET name = 'ServiceRequest-标准-响应告警'
WHERE r.name = 'ServiceRequest-Standard-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('ServiceRequest-标准', 'ServiceRequest-Standard'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P3-低-解决告警'
WHERE r.name = 'Incident-P3-Low-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P3-低', 'Incident-P3-Low'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P3-低-响应告警'
WHERE r.name = 'Incident-P3-Low-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P3-低', 'Incident-P3-Low'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P2-中-解决告警'
WHERE r.name = 'Incident-P2-Medium-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P2-中', 'Incident-P2-Medium'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P2-中-响应告警'
WHERE r.name = 'Incident-P2-Medium-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P2-中', 'Incident-P2-Medium'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P1-高-解决告警'
WHERE r.name = 'Incident-P1-High-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P1-高', 'Incident-P1-High'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P1-高-响应告警'
WHERE r.name = 'Incident-P1-High-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P1-高', 'Incident-P1-High'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P0-紧急-解决告警'
WHERE r.name = 'Incident-P0-Urgent-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P0-紧急', 'Incident-P0-Urgent'));

UPDATE sla_alert_rules AS r SET name = 'Incident-P0-紧急-响应告警'
WHERE r.name = 'Incident-P0-Urgent-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('Incident-P0-紧急', 'Incident-P0-Urgent'));

UPDATE sla_alert_rules AS r SET name = 'SLA-变更-解决告警'
WHERE r.name = 'SLA-Change-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-变更', 'SLA-Change'));

UPDATE sla_alert_rules AS r SET name = 'SLA-变更-响应告警'
WHERE r.name = 'SLA-Change-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-变更', 'SLA-Change'));

UPDATE sla_alert_rules AS r SET name = 'SLA-服务请求-解决告警'
WHERE r.name = 'SLA-Service Request-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-服务请求', 'SLA-Service Request'));

UPDATE sla_alert_rules AS r SET name = 'SLA-服务请求-响应告警'
WHERE r.name = 'SLA-Service Request-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-服务请求', 'SLA-Service Request'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P3-低-解决告警'
WHERE r.name = 'SLA-P3-Low-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P3-低', 'SLA-P3-Low'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P3-低-响应告警'
WHERE r.name = 'SLA-P3-Low-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P3-低', 'SLA-P3-Low'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P2-中-解决告警'
WHERE r.name = 'SLA-P2-Medium-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P2-中', 'SLA-P2-Medium'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P2-中-响应告警'
WHERE r.name = 'SLA-P2-Medium-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P2-中', 'SLA-P2-Medium'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P1-高-解决告警'
WHERE r.name = 'SLA-P1-High-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P1-高', 'SLA-P1-High'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P1-高-响应告警'
WHERE r.name = 'SLA-P1-High-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P1-高', 'SLA-P1-High'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P0-紧急-解决告警'
WHERE r.name = 'SLA-P0-Urgent-Resolution Alert' AND r.threshold_percentage = 80
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P0-紧急', 'SLA-P0-Urgent'));

UPDATE sla_alert_rules AS r SET name = 'SLA-P0-紧急-响应告警'
WHERE r.name = 'SLA-P0-Urgent-Response Alert' AND r.threshold_percentage = 50
  AND EXISTS (SELECT 1 FROM sla_definitions AS s WHERE s.id = r.sla_definition_id AND s.tenant_id = r.tenant_id AND s.name IN ('SLA-P0-紧急', 'SLA-P0-Urgent'));

-- Restore only unchanged English baseline records, with the same identity/configuration guards.

UPDATE users AS t
SET name = '系统管理员', department = 'IT部门'
WHERE t.username = 'admin'
  AND t.email = 'admin@example.com'
  AND t.name = 'System Administrator'
  AND t.department = 'IT Department'
  AND t.role = 'super_admin'
  AND t.tenant_id IN (SELECT id FROM tenants WHERE code = 'default');

UPDATE ticket_types AS t
SET name = '其他工单', description = '通用工单类型，用于不属于以上分类的请求'
WHERE t.name = 'General Ticket'
  AND t.description = 'General ticket type for other requests'
  AND t.code = 'general'
  AND t.icon = 'FileText'
  AND t.color = '#8c8c8c'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '其他工单' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = '数据库账号申请', description = '申请数据库读写账号、只读账号等'
WHERE t.name = 'Database Account Request'
  AND t.description = 'Request database read/write or read-only accounts'
  AND t.code = 'db_account_apply'
  AND t.icon = 'Key'
  AND t.color = '#faad14'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '数据库账号申请' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = '项目申请', description = '申请创建新项目或项目空间'
WHERE t.name = 'Project Request'
  AND t.description = 'Request a new project or workspace'
  AND t.code = 'project_apply'
  AND t.icon = 'Project'
  AND t.color = '#722ed1'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '项目申请' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = '应用申请', description = '申请在K8S集群中部署新应用服务'
WHERE t.name = 'Application Request'
  AND t.description = 'Request deployment of an application in Kubernetes'
  AND t.code = 'app_apply'
  AND t.icon = 'Appstore'
  AND t.color = '#1890ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '应用申请' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = '防火墙规则申请', description = '申请开放或变更防火墙端口规则'
WHERE t.name = 'Firewall Rule Request'
  AND t.description = 'Request new or modified firewall port rules'
  AND t.code = 'firewall_apply'
  AND t.icon = 'Safety'
  AND t.color = '#fa8c16'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '防火墙规则申请' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = '域名申请', description = '申请新的域名或域名解析变更'
WHERE t.name = 'Domain Request'
  AND t.description = 'Request a domain or DNS change'
  AND t.code = 'domain_apply'
  AND t.icon = 'Global'
  AND t.color = '#eb2f96'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '域名申请' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'GitLab代码仓库申请', description = '申请创建新的GitLab代码仓库'
WHERE t.name = 'GitLab Repository Request'
  AND t.description = 'Request a new GitLab repository'
  AND t.code = 'gitlab_repo_apply'
  AND t.icon = 'Code'
  AND t.color = '#fa541c'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'GitLab代码仓库申请' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = '账号申请', description = '申请系统账号、VPN账号、堡垒机账号等'
WHERE t.name = 'Account Request'
  AND t.description = 'Request system, VPN, or bastion host accounts'
  AND t.code = 'account_apply'
  AND t.icon = 'User'
  AND t.color = '#52c41a'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '账号申请' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = '虚拟机申请', description = '申请新的虚拟机资源'
WHERE t.name = 'Virtual Machine Request'
  AND t.description = 'Request new virtual machine resources'
  AND t.code = 'vm_apply'
  AND t.icon = 'Desktop'
  AND t.color = '#2f54eb'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '虚拟机申请' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = '数据导出', description = '从数据库或系统导出数据'
WHERE t.name = 'Data Export'
  AND t.description = 'Export data from a database or system'
  AND t.code = 'data_export'
  AND t.icon = 'Download'
  AND t.color = '#13c2c2'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '数据导出' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'DDL执行', description = '数据库表结构变更、索引创建等DDL操作'
WHERE t.name = 'DDL Execution'
  AND t.description = 'Database schema changes, index creation, and other DDL operations'
  AND t.code = 'ddl_execute'
  AND t.icon = 'Database'
  AND t.color = '#722ed1'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'DDL执行' AND sibling.id <> t.id);

UPDATE ticket_types AS t
SET name = 'K8S扩缩容', description = 'Kubernetes容器集群扩容或缩容请求'
WHERE t.name = 'Kubernetes Scaling'
  AND t.description = 'Request to scale a Kubernetes cluster up or down'
  AND t.code = 'k8s_scale'
  AND t.icon = 'Container'
  AND t.color = '#1890ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_types AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'K8S扩缩容' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = '默认服务请求SLA', description = '标准服务请求履约策略'
WHERE t.name = 'Default Service Request SLA'
  AND t.description = 'Standard service request fulfillment policy'
  AND t.priority = 'low'
  AND t.response_time_minutes = 240
  AND t.resolution_time_minutes = 2880
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '默认服务请求SLA' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = '默认P2事件SLA', description = '中优先级事件响应与解决策略'
WHERE t.name = 'Default P2 Incident SLA'
  AND t.description = 'Medium priority incident response and resolution policy'
  AND t.priority = 'medium'
  AND t.response_time_minutes = 120
  AND t.resolution_time_minutes = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '默认P2事件SLA' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = '默认P1事件SLA', description = '高优先级事件响应与解决策略'
WHERE t.name = 'Default P1 Incident SLA'
  AND t.description = 'High priority incident response and resolution policy'
  AND t.priority = 'high'
  AND t.response_time_minutes = 30
  AND t.resolution_time_minutes = 240
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '默认P1事件SLA' AND sibling.id <> t.id);

UPDATE marketplace_items AS t
SET title = 'Jira 双向同步插件', description = 'Jira Issue 与 ITSM 工单双向同步，状态/评论/附件双向同步', long_description = 'Jira 同步插件把 Jira Project/Issue 作为外部系统接入：可配置字段映射、定时同步或事件触发同步，处理 Issue 与 ITSM 工单之间的状态/评论/附件双向同步。'
WHERE t.title = 'Jira Bidirectional Sync Plugin'
  AND t.description = 'Synchronize Jira issues and ITSM tickets, including status, comments, and attachments'
  AND t.long_description = 'Connect Jira projects and issues with configurable field mappings, scheduled or event-driven synchronization, and bidirectional status, comment, and attachment updates.'
  AND t.tags::jsonb = '["Jira","plugin","sync"]'::jsonb
  AND t.name = 'jira-sync-plugin'
  AND t.type = 'plugin'
  AND t.provider = 'Community'
  AND t.latest_version = '0.8.0';

UPDATE marketplace_items AS t
SET title = 'AI 工单分类与优先级建议', description = '基于 LLM 的工单自动分类、优先级推荐、标签补全 Skill', long_description = 'AI 工单分类 Skill 在工单创建后异步运行，结合知识库与历史工单给出分类、优先级、标签建议，置信度高于阈值时直接接受，否则回到人工确认流程。', tags = '["AI","skill","分类","triage"]'::jsonb
WHERE t.title = 'AI Ticket Classification and Priority Suggestions'
  AND t.description = 'An LLM skill for ticket classification, priority recommendations, and tag suggestions'
  AND t.long_description = 'Runs asynchronously after ticket creation and uses knowledge and ticket history to suggest classifications, priorities, and tags. Recommendations above the configured confidence threshold are accepted; others require human confirmation.'
  AND t.tags::jsonb = '["AI","skill","Classification","triage"]'::jsonb
  AND t.name = 'ai-ticket-classify'
  AND t.type = 'skill'
  AND t.provider = 'ITSM Foundation'
  AND t.latest_version = '1.4.0';

UPDATE marketplace_items AS t
SET title = '企业微信连接器', description = '与企微通讯录、审批、应用消息集成，承接内网审批与告警推送', long_description = '企业微信连接器提供通讯录同步、自建应用消息推送、审批回调等能力，适合与企微深度绑定的客户在 IT 服务台场景中复用现有账号体系与审批流。', tags = '["企业微信","通讯录","审批"]'::jsonb
WHERE t.title = 'WeCom Connector'
  AND t.description = 'Integrate WeCom contacts, approvals, and application messages for internal approvals and alerts'
  AND t.long_description = 'Synchronize WeCom contacts, send application messages, and receive approval callbacks while reusing existing identity and approval systems.'
  AND t.tags::jsonb = '["WeCom","Contacts","Approval"]'::jsonb
  AND t.name = 'wecom-connector'
  AND t.type = 'connector'
  AND t.provider = 'ITSM Foundation'
  AND t.latest_version = '1.0.3';

UPDATE marketplace_items AS t
SET title = '钉钉 AI 工单助手', description = '钉钉群聊内的 AI 工单助理，支持自然语言创建工单、查询状态、催办', long_description = '钉钉 AI 工单助手是面向运维/客服群聊的 Skill：识别用户文本中的工单意图，调用 ITSM API 创建/查询/催办；支持「帮我开个 P2 工单」「这个工单处理到哪一步了」「催一下张三」等表达工单操作。', tags = '["钉钉","AI","skill","工单"]'::jsonb
WHERE t.title = 'DingTalk AI Ticket Assistant'
  AND t.description = 'An AI ticket assistant for DingTalk group chats, supporting ticket creation, status queries, and reminders'
  AND t.long_description = 'A skill for operations and support chats that identifies ticket intent and uses ITSM APIs to create tickets, check status, and send reminders.'
  AND t.tags::jsonb = '["DingTalk","AI","skill","Ticket"]'::jsonb
  AND t.name = 'dingtalk-ai-assistant'
  AND t.type = 'skill'
  AND t.provider = 'ITSM Foundation'
  AND t.latest_version = '0.9.1';

UPDATE marketplace_items AS t
SET title = '飞书审批增强连接器', description = '将 ITSM 工单与飞书审批原生打通，支持审批实例双向同步、消息卡片回调', long_description = '飞书审批增强连接器提供与飞书开放平台的原生集成：审批实例创建、状态回调、消息卡片推送、用户身份映射。可用于变更请求、服务请求等场景的飞书端审批入口。', tags = '["飞书","审批","IM","OA"]'::jsonb
WHERE t.title = 'Feishu Approval Connector'
  AND t.description = 'Integrate ITSM tickets with Feishu approvals, including bidirectional approval synchronization and message card callbacks'
  AND t.long_description = 'Native Feishu integration for approval creation, status callbacks, message cards, and user identity mapping. Supports change and service request approvals from Feishu.'
  AND t.tags::jsonb = '["Feishu","Approval","IM","OA"]'::jsonb
  AND t.name = 'feishu-connector'
  AND t.type = 'connector'
  AND t.provider = 'ITSM Foundation'
  AND t.latest_version = '1.2.0';

UPDATE standard_changes AS t
SET title = '应用配置更新', description = '更新应用程序配置文件中的参数', implementation_plan = '1. 备份当前配置
2. 修改配置参数
3. 重启应用服务
4. 验证功能正常', rollback_plan = '回滚到备份的配置文件', justification = '优化系统性能', affected_cis = '["应用服务器"]'::jsonb, prerequisites = '["新配置已测试通过"]'::jsonb
WHERE t.title = 'Application Configuration Update'
  AND t.description = 'Update application configuration parameters'
  AND t.implementation_plan = '1. Back up the configuration
2. Update parameters
3. Restart the application
4. Verify functionality'
  AND t.rollback_plan = 'Restore the backed-up configuration file'
  AND t.justification = 'Optimize system performance'
  AND t.affected_cis::jsonb = '["Application Server"]'::jsonb
  AND t.prerequisites::jsonb = '["New configuration tested successfully"]'::jsonb
  AND t.remarks = ''
  AND t.category = '应用'
  AND t.risk_level = 'low'
  AND t.impact_scope = 'low'
  AND t.expected_duration = 30
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = '应用配置更新' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = '防火墙规则添加', description = '添加新的防火墙放行规则', implementation_plan = '1. 准备规则变更申请
2. 在测试环境验证
3. 生产环境应用新规则
4. 监控网络流量', rollback_plan = '发现异常时立即删除新添加的规则', justification = '业务需要开放新端口', affected_cis = '["防火墙","网络交换机"]'::jsonb, prerequisites = '["已完成安全评估","相关业务部门确认"]'::jsonb, remarks = '需安全部门审批'
WHERE t.title = 'Add Firewall Rule'
  AND t.description = 'Add an allow rule to the firewall'
  AND t.implementation_plan = '1. Prepare the rule change request
2. Validate in a test environment
3. Apply the rule in production
4. Monitor network traffic'
  AND t.rollback_plan = 'Remove the new rule immediately if issues occur'
  AND t.justification = 'Business requirements need a new port'
  AND t.affected_cis::jsonb = '["Firewall","Network Switch"]'::jsonb
  AND t.prerequisites::jsonb = '["Security assessment completed","Business department approval obtained"]'::jsonb
  AND t.remarks = 'Security team approval required'
  AND t.category = '网络安全'
  AND t.risk_level = 'medium'
  AND t.impact_scope = 'medium'
  AND t.expected_duration = 45
  AND t.approval_required = true
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = '防火墙规则添加' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = '数据库备份', description = '执行数据库全量备份', implementation_plan = '1. 停止数据库写入
2. 执行全量备份
3. 验证备份完整性
4. 恢复数据库服务', rollback_plan = '备份失败时取消备份操作', justification = '数据安全要求', affected_cis = '["数据库服务器"]'::jsonb, prerequisites = '["确认备份存储空间充足","检查备份工具可用性"]'::jsonb
WHERE t.title = 'Database Backup'
  AND t.description = 'Perform a full database backup'
  AND t.implementation_plan = '1. Pause database writes
2. Perform a full backup
3. Verify backup integrity
4. Restore database service'
  AND t.rollback_plan = 'Cancel the operation if the backup fails'
  AND t.justification = 'Data protection requirements'
  AND t.affected_cis::jsonb = '["Database Server"]'::jsonb
  AND t.prerequisites::jsonb = '["Confirm sufficient backup storage","Check backup tool availability"]'::jsonb
  AND t.remarks = ''
  AND t.category = '数据库'
  AND t.risk_level = 'low'
  AND t.impact_scope = 'medium'
  AND t.expected_duration = 120
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = '数据库备份' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = 'SSL证书更新', description = '更新即将过期的SSL证书', implementation_plan = '1. 申请新证书
2. 在测试环境验证
3. 生产环境部署
4. 验证证书生效', rollback_plan = '保留旧证书，发现问题可立即回滚', justification = '证书即将过期，必须更新', affected_cis = '["负载均衡器","Web服务器"]'::jsonb, prerequisites = '["新证书已申请","获取证书文件"]'::jsonb
WHERE t.title = 'SSL Certificate Renewal'
  AND t.description = 'Renew an expiring SSL certificate'
  AND t.implementation_plan = '1. Obtain a new certificate
2. Validate in a test environment
3. Deploy to production
4. Verify the certificate'
  AND t.rollback_plan = 'Keep the previous certificate for immediate rollback if needed'
  AND t.justification = 'The certificate is expiring and must be renewed'
  AND t.affected_cis::jsonb = '["Load Balancer","Web Server"]'::jsonb
  AND t.prerequisites::jsonb = '["New certificate requested","Obtain the certificate files"]'::jsonb
  AND t.remarks = ''
  AND t.category = '安全'
  AND t.risk_level = 'low'
  AND t.impact_scope = 'low'
  AND t.expected_duration = 60
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = 'SSL证书更新' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = '服务器重启', description = '标准服务器重启流程，用于常规维护', implementation_plan = '1. 通知相关用户
2. 停止服务
3. 重启服务器
4. 验证服务恢复', rollback_plan = '如果重启失败，立即回滚到重启前状态', justification = '例行维护', affected_cis = '["服务器"]'::jsonb, prerequisites = '["提前通知用户","备份重要数据"]'::jsonb, remarks = '仅适用于非关键业务服务器'
WHERE t.title = 'Server Restart'
  AND t.description = 'Standard server restart for routine maintenance'
  AND t.implementation_plan = '1. Notify affected users
2. Stop services
3. Restart the server
4. Verify service recovery'
  AND t.rollback_plan = 'If the restart fails, restore the previous state immediately'
  AND t.justification = 'Routine maintenance'
  AND t.affected_cis::jsonb = '["Server"]'::jsonb
  AND t.prerequisites::jsonb = '["Notify users in advance","Back up important data"]'::jsonb
  AND t.remarks = 'For non-critical servers only'
  AND t.category = '服务器'
  AND t.risk_level = 'low'
  AND t.impact_scope = 'low'
  AND t.expected_duration = 30
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = '服务器重启' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '监控告警', description = '监控和告警相关'
WHERE t.name = 'Monitoring Alert'
  AND t.description = 'Monitoring and alerting'
  AND t.color = '#fa8c16'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '监控告警' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '高可用', description = '高可用相关'
WHERE t.name = 'High Availability'
  AND t.description = 'High availability issues'
  AND t.color = '#fa541c'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '高可用' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '需要审核', description = '需要上级审核'
WHERE t.name = 'Needs Review'
  AND t.description = 'Supervisor review required'
  AND t.color = '#1677ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '需要审核' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '已解决', description = '已解决的问题'
WHERE t.name = 'Resolved'
  AND t.description = 'Resolved issues'
  AND t.color = '#52c41a'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '已解决' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '无法复现', description = '无法复现的问题'
WHERE t.name = 'Cannot Reproduce'
  AND t.description = 'Issues that cannot be reproduced'
  AND t.color = '#d9d9d9'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '无法复现' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '重复', description = '重复问题'
WHERE t.name = 'Duplicate'
  AND t.description = 'Duplicate issue'
  AND t.color = '#8c8c8c'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '重复' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '待反馈', description = '等待用户反馈'
WHERE t.name = 'Awaiting Feedback'
  AND t.description = 'Awaiting user feedback'
  AND t.color = '#faad14'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '待反馈' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '数据库', description = '数据库相关问题'
WHERE t.name = 'Database'
  AND t.description = 'Database issues'
  AND t.color = '#52c41a'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '数据库' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '网络', description = '网络相关问题'
WHERE t.name = 'Network'
  AND t.description = 'Network issues'
  AND t.color = '#13c2c2'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '网络' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '安全', description = '安全问题'
WHERE t.name = 'Security'
  AND t.description = 'Security Issue'
  AND t.color = '#eb2f96'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '性能问题', description = '系统性能相关'
WHERE t.name = 'Performance Issue'
  AND t.description = 'System performance issues'
  AND t.color = '#722ed1'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '性能问题' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '功能需求', description = '新功能请求'
WHERE t.name = 'Feature Request'
  AND t.description = 'New feature request'
  AND t.color = '#1890ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '功能需求' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET description = '程序缺陷'
WHERE t.name = 'bug'
  AND t.description = 'Software defect'
  AND t.color = '#f5222d';

UPDATE ticket_tags AS t
SET name = '重要', description = '重要但不紧急'
WHERE t.name = 'Important'
  AND t.description = 'Important but not urgent'
  AND t.color = '#fa8c16'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '重要' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '紧急', description = '紧急处理的问题'
WHERE t.name = 'Urgent'
  AND t.description = 'Issues requiring urgent attention'
  AND t.color = '#ff4d4f'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '紧急' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '配置问题', description = '系统配置错误'
WHERE t.name = 'Configuration Issue'
  AND t.description = 'System configuration errors'
  AND t.code = 'config'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '配置问题' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '性能问题', description = '系统响应慢、卡顿'
WHERE t.name = 'Performance Issue'
  AND t.description = 'Slow system response or freezes'
  AND t.code = 'performance'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '性能问题' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '安全问题', description = '安全事件、漏洞'
WHERE t.name = 'Security Issue'
  AND t.description = 'Security incidents and vulnerabilities'
  AND t.code = 'security'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全问题' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '数据库问题', description = '数据库性能、连接问题'
WHERE t.name = 'Database Issue'
  AND t.description = 'Database performance or connection issues'
  AND t.code = 'database'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '数据库问题' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '最终用户', description = '服务请求与查看本人工单'
WHERE t.name = 'End User'
  AND t.description = 'Submit service requests and view own tickets'
  AND t.code = 'end_user'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '最终用户' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '技术员', description = '二线技术处理'
WHERE t.name = 'Technician'
  AND t.description = 'Second-line technical support'
  AND t.code = 'technician'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '技术员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '服务台坐席', description = '一线支持与工单处理'
WHERE t.name = 'Service Desk Agent'
  AND t.description = 'First-line support and ticket handling'
  AND t.code = 'agent'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务台坐席' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '部门经理', description = '部门审批与 L1 审批人'
WHERE t.name = 'Department Manager'
  AND t.description = 'Department and L1 approvals'
  AND t.code = 'manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '部门经理' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'IT管理员', description = 'ITIL 流程运营，L2 审批人'
WHERE t.name = 'IT Administrator'
  AND t.description = 'ITIL process operations and L2 approvals'
  AND t.code = 'it_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT管理员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '安全审计员', description = '等保三员③：全域只读与审计轨迹'
WHERE t.name = 'Security Auditor'
  AND t.description = 'Read-only access and audit trails'
  AND t.code = 'audit_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全审计员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '安全管理员', description = '等保三员②：安全策略与合规，L3 审批人'
WHERE t.name = 'Security Administrator'
  AND t.description = 'Security policies, compliance, and L3 approvals'
  AND t.code = 'security_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全管理员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '系统管理员', description = '等保三员①：系统与集成运维'
WHERE t.name = 'System Administrator'
  AND t.description = 'System administration and integration operations'
  AND t.code = 'sysadmin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '系统管理员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '租户管理员', description = '租户内业务与配置全量管理'
WHERE t.name = 'Tenant Administrator'
  AND t.description = 'Manage all tenant operations and configuration'
  AND t.code = 'admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '租户管理员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '超级管理员', description = '全部权限，跨租户引导'
WHERE t.name = 'Super Administrator'
  AND t.description = 'Full access and cross-tenant initialization'
  AND t.code = 'super_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '超级管理员' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = 'P3-一般响应', description = 'P3级别一般响应SLA，2小时响应，8小时解决'
WHERE t.name = 'P3-Normal Response'
  AND t.description = 'P3 normal response SLA: respond in 2 hours and resolve in 8 hours'
  AND t.priority = 'medium'
  AND t.response_time_minutes = 120
  AND t.resolution_time_minutes = 480
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'P3-一般响应' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = 'P2-标准响应', description = 'P2级别标准响应SLA，30分钟响应，4小时解决'
WHERE t.name = 'P2-Standard Response'
  AND t.description = 'P2 standard response SLA: respond in 30 minutes and resolve in 4 hours'
  AND t.priority = 'high'
  AND t.response_time_minutes = 30
  AND t.resolution_time_minutes = 240
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'P2-标准响应' AND sibling.id <> t.id);

UPDATE sla_policies AS t
SET name = 'P1-紧急响应', description = 'P1级别紧急响应SLA，15分钟响应，60分钟解决'
WHERE t.name = 'P1-Urgent Response'
  AND t.description = 'P1 urgent response SLA: respond in 15 minutes and resolve in 60 minutes'
  AND t.priority = 'critical'
  AND t.response_time_minutes = 15
  AND t.resolution_time_minutes = 60
  AND NOT EXISTS (SELECT 1 FROM sla_policies AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'P1-紧急响应' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = '数据库账号权限调整', description = '按最小权限原则调整数据库账号权限', implementation_plan = '确认账号、库表范围和权限级别，执行授权并记录审计。', rollback_plan = '回收新增权限或恢复原授权快照。', justification = '支持业务运维和数据访问需求。', prerequisites = '["数据 owner 审批","权限范围明确"]'::jsonb, remarks = '生产写权限需安全复核。'
WHERE t.title = 'Database Account Access Change'
  AND t.description = 'Adjust database account access using least privilege'
  AND t.implementation_plan = 'Verify the account, database and table scope, and access level; apply permissions and record an audit trail.'
  AND t.rollback_plan = 'Revoke added permissions or restore the original access snapshot.'
  AND t.justification = 'Support business operations and data access requirements.'
  AND t.affected_cis::jsonb = '["database"]'::jsonb
  AND t.prerequisites::jsonb = '["Data owner approval","Access scope defined"]'::jsonb
  AND t.remarks = 'Production write access requires security review.'
  AND t.category = '数据库'
  AND t.risk_level = 'medium'
  AND t.impact_scope = '指定数据库实例'
  AND t.expected_duration = 30
  AND t.approval_required = true
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = '数据库账号权限调整' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = '低风险防火墙规则调整', description = '在已批准范围内新增或调整访问控制规则', implementation_plan = '核对源、目的、端口、有效期和业务负责人，变更后执行连通性验证。', rollback_plan = '删除新增规则或恢复原规则快照。', justification = '满足业务系统访问需求。', prerequisites = '["业务负责人确认","安全策略审核通过"]'::jsonb, remarks = '高风险开放需走普通变更。'
WHERE t.title = 'Low Risk Firewall Rule Change'
  AND t.description = 'Add or modify access control rules within the approved scope'
  AND t.implementation_plan = 'Verify source, destination, ports, expiry, and business owner; test connectivity after the change.'
  AND t.rollback_plan = 'Remove new rules or restore the original rule snapshot.'
  AND t.justification = 'Meet business application access requirements.'
  AND t.affected_cis::jsonb = '["firewall"]'::jsonb
  AND t.prerequisites::jsonb = '["Business owner approval","Security policy review approved"]'::jsonb
  AND t.remarks = 'High risk access changes require the normal change process.'
  AND t.category = '网络'
  AND t.risk_level = 'low'
  AND t.impact_scope = '单业务系统'
  AND t.expected_duration = 45
  AND t.approval_required = true
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = '低风险防火墙规则调整' AND sibling.id <> t.id);

UPDATE standard_changes AS t
SET title = '标准软件安装', description = '安装已批准的软件白名单中的客户端或工具', implementation_plan = '确认软件版本和授权，按标准安装步骤执行，完成后验证可用性。', rollback_plan = '卸载软件并恢复原配置。', justification = '提升终端支持效率。', prerequisites = '["软件已进入白名单"]'::jsonb, remarks = '可由服务台执行。'
WHERE t.title = 'Standard Software Installation'
  AND t.description = 'Install clients or tools from the approved software allowlist'
  AND t.implementation_plan = 'Verify version and licensing, follow the standard installation procedure, and confirm availability.'
  AND t.rollback_plan = 'Uninstall the software and restore the original configuration.'
  AND t.justification = 'Improve endpoint support efficiency.'
  AND t.affected_cis::jsonb = '[]'::jsonb
  AND t.prerequisites::jsonb = '["Software is on the approved allowlist"]'::jsonb
  AND t.remarks = 'May be performed by the service desk.'
  AND t.category = '终端支持'
  AND t.risk_level = 'low'
  AND t.impact_scope = '单用户或单设备'
  AND t.expected_duration = 30
  AND t.approval_required = false
  AND NOT EXISTS (SELECT 1 FROM standard_changes AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.title = '标准软件安装' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '其他', description = '其他类型事件'
WHERE t.name = 'Other'
  AND t.description = 'Other incident types'
  AND t.code = 'other'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '其他' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '配置问题', description = '系统配置、发布配置或参数错误'
WHERE t.name = 'Configuration Issue'
  AND t.description = 'System, release, or parameter configuration errors'
  AND t.code = 'configuration'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '配置问题' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '性能问题', description = '系统响应慢、容量或资源瓶颈'
WHERE t.name = 'Performance Issue'
  AND t.description = 'Slow response, capacity constraints, or resource bottlenecks'
  AND t.code = 'performance'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '性能问题' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '安全问题', description = '安全事件、漏洞和合规问题'
WHERE t.name = 'Security Issue'
  AND t.description = 'Security incidents, vulnerabilities, and compliance issues'
  AND t.code = 'security'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全问题' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '数据库问题', description = '数据库性能、连接、备份和权限问题'
WHERE t.name = 'Database Issue'
  AND t.description = 'Database performance, connectivity, backup, and access issues'
  AND t.code = 'database'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '数据库问题' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '网络故障', description = '网络连接、网络设备问题'
WHERE t.name = 'Network Failure'
  AND t.description = 'Network connectivity or device issues'
  AND t.code = 'network'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '网络故障' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '软件故障', description = '操作系统、应用软件故障'
WHERE t.name = 'Software Failure'
  AND t.description = 'Operating system or application failures'
  AND t.code = 'software'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '软件故障' AND sibling.id <> t.id);

UPDATE ticket_categories AS t
SET name = '硬件故障', description = '服务器、存储、网络设备等硬件故障'
WHERE t.name = 'Hardware Failure'
  AND t.description = 'Server, storage, or network hardware failures'
  AND t.code = 'hardware'
  AND NOT EXISTS (SELECT 1 FROM ticket_categories AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '硬件故障' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '客户影响', description = '已影响客户或终端用户的记录'
WHERE t.name = 'Customer Impact'
  AND t.description = 'Records affecting customers or end users'
  AND t.color = '#1677ff'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '客户影响' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = 'AI建议', description = '包含AI辅助建议的记录'
WHERE t.name = 'AI Suggestion'
  AND t.description = 'Records containing AI-assisted suggestions'
  AND t.color = '#52c41a'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'AI建议' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '需审批', description = '需要审批流程的记录'
WHERE t.name = 'Approval Required'
  AND t.description = 'Records requiring an approval workflow'
  AND t.color = '#faad14'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '需审批' AND sibling.id <> t.id);

UPDATE ticket_tags AS t
SET name = '高优先级', description = '需要优先处理的记录'
WHERE t.name = 'High Priority'
  AND t.description = 'Records requiring priority attention'
  AND t.color = '#f5222d'
  AND NOT EXISTS (SELECT 1 FROM ticket_tags AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '高优先级' AND sibling.id <> t.id);

UPDATE ci_types AS t
SET description = 'Kubernetes资源'
WHERE t.description = 'Kubernetes Resource'
  AND t.name = 'kubernetes'
  AND t.icon = 'kubernetes'
  AND t.color = '#20c997';

UPDATE ci_types AS t
SET description = '云虚拟机'
WHERE t.description = 'Cloud Virtual Machine'
  AND t.name = 'cloud_vm'
  AND t.icon = 'cloud'
  AND t.color = '#6f42c1';

UPDATE ci_types AS t
SET description = '中间件'
WHERE t.description = 'Middleware'
  AND t.name = 'middleware'
  AND t.icon = 'middleware'
  AND t.color = '#e74c3c';

UPDATE ci_types AS t
SET description = '应用服务'
WHERE t.description = 'Application Service'
  AND t.name = 'application'
  AND t.icon = 'app'
  AND t.color = '#6610f2';

UPDATE ci_types AS t
SET description = '存储设备'
WHERE t.description = 'Storage Device'
  AND t.name = 'storage'
  AND t.icon = 'storage'
  AND t.color = '#e83e8c';

UPDATE ci_types AS t
SET description = '网络设备'
WHERE t.description = 'Network Device'
  AND t.name = 'network'
  AND t.icon = 'network'
  AND t.color = '#17a2b8';

UPDATE ci_types AS t
SET description = '数据库'
WHERE t.description = 'Database'
  AND t.name = 'database'
  AND t.icon = 'database'
  AND t.color = '#fd7e14';

UPDATE ci_types AS t
SET description = '服务器'
WHERE t.description = 'Server'
  AND t.name = 'server'
  AND t.icon = 'server'
  AND t.color = '#28a745';

UPDATE ticket_views AS t
SET name = '高优先级工单', description = 'P0/P1或高优先级工单'
WHERE t.name = 'High Priority Tickets'
  AND t.description = 'P0/P1 or high priority tickets'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","created_at"]'::jsonb
  AND t.is_shared = true
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '高优先级工单' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = '权限申请审批', description = '高权限申请需要直属负责人和安全管理员审批', nodes = '[{"type":"approval","name":"直属负责人审批","approver_type":"role","role":"manager","timeout":480},{"type":"approval","name":"安全管理员审批","approver_type":"role","role":"security_admin","timeout":480}]'::jsonb
WHERE t.name = 'Access Request Approval'
  AND t.description = 'Privileged access requires line manager and security administrator approval'
  AND t.nodes::jsonb = '[{"type":"approval","name":"Line Manager Approval","approver_type":"role","role":"manager","timeout":480},{"type":"approval","name":"Security Administrator Approval","approver_type":"role","role":"security_admin","timeout":480}]'::jsonb
  AND t.ticket_type = 'service_request'
  AND t.priority = 'high'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '权限申请审批' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = '紧急变更审批', description = '紧急变更采用快速审批并要求事后复盘', nodes = '[{"type":"approval","name":"值班经理审批","approver_type":"role","role":"manager","timeout":60},{"type":"approval","name":"安全确认","approver_type":"role","role":"security_admin","timeout":120}]'::jsonb
WHERE t.name = 'Emergency Change Approval'
  AND t.description = 'Emergency changes use expedited approval and require a post-implementation review'
  AND t.nodes::jsonb = '[{"type":"approval","name":"Duty Manager Approval","approver_type":"role","role":"manager","timeout":60},{"type":"approval","name":"Security Confirmation","approver_type":"role","role":"security_admin","timeout":120}]'::jsonb
  AND t.ticket_type = 'change'
  AND t.priority = 'urgent'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '紧急变更审批' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = '普通变更审批', description = '普通变更需要技术负责人和变更经理审批', nodes = '[{"type":"approval","name":"技术负责人审批","approver_type":"role","role":"it_admin","timeout":480},{"type":"approval","name":"变更经理审批","approver_type":"role","role":"change_manager","timeout":480}]'::jsonb
WHERE t.name = 'Normal Change Approval'
  AND t.description = 'Normal changes require technical lead and change manager approval'
  AND t.nodes::jsonb = '[{"type":"approval","name":"Technical Lead Approval","approver_type":"role","role":"it_admin","timeout":480},{"type":"approval","name":"Change Manager Approval","approver_type":"role","role":"change_manager","timeout":480}]'::jsonb
  AND t.ticket_type = 'change'
  AND t.priority = 'medium,high'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '普通变更审批' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = '服务请求审批', description = '涉及资源、权限或费用的服务请求需要部门负责人审批', nodes = '[{"type":"approval","name":"部门负责人审批","approver_type":"role","role":"manager","timeout":480}]'::jsonb
WHERE t.name = 'Service Request Approval'
  AND t.description = 'Service requests involving resources, access, or expenses require department head approval'
  AND t.nodes::jsonb = '[{"type":"approval","name":"Department Head Approval","approver_type":"role","role":"manager","timeout":480}]'::jsonb
  AND t.ticket_type = 'service_request'
  AND t.priority = ''
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务请求审批' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Change-紧急', description = '紧急变更快速评审与实施SLA'
WHERE t.name = 'Change-Emergency'
  AND t.description = 'Emergency change review and implementation SLA'
  AND t.service_type = 'change'
  AND t.priority = 'high'
  AND t.response_time = 60
  AND t.resolution_time = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Change-紧急' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Change-普通', description = '普通变更评审与实施SLA'
WHERE t.name = 'Change-Normal'
  AND t.description = 'Normal change review and implementation SLA'
  AND t.service_type = 'change'
  AND t.priority = 'medium'
  AND t.response_time = 480
  AND t.resolution_time = 10080
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Change-普通' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'ServiceRequest-标准', description = '标准服务请求交付SLA'
WHERE t.name = 'ServiceRequest-Standard'
  AND t.description = 'Standard service request delivery SLA'
  AND t.service_type = 'service_request'
  AND t.priority = 'medium'
  AND t.response_time = 480
  AND t.resolution_time = 4320
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ServiceRequest-标准' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Incident-P3-低', description = '咨询、轻微影响或非紧急问题'
WHERE t.name = 'Incident-P3-Low'
  AND t.description = 'Inquiries, minor impact, or non-urgent issues'
  AND t.service_type = 'incident'
  AND t.priority = 'low'
  AND t.response_time = 240
  AND t.resolution_time = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Incident-P3-低' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Incident-P2-中', description = '局部功能受影响，有替代方案'
WHERE t.name = 'Incident-P2-Medium'
  AND t.description = 'Limited functionality affected with a workaround available'
  AND t.service_type = 'incident'
  AND t.priority = 'medium'
  AND t.response_time = 120
  AND t.resolution_time = 480
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Incident-P2-中' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Incident-P1-高', description = '重要业务受影响，需要快速恢复'
WHERE t.name = 'Incident-P1-High'
  AND t.description = 'Important business services affected; rapid recovery required'
  AND t.service_type = 'incident'
  AND t.priority = 'high'
  AND t.response_time = 30
  AND t.resolution_time = 240
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Incident-P1-高' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'Incident-P0-紧急', description = '核心业务中断或重大安全事件'
WHERE t.name = 'Incident-P0-Urgent'
  AND t.description = 'Critical business outage or major security incident'
  AND t.service_type = 'incident'
  AND t.priority = 'urgent'
  AND t.response_time = 15
  AND t.resolution_time = 120
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Incident-P0-紧急' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '变更委员会', description = '变更评审、发布窗口和风险控制'
WHERE t.name = 'Change Advisory Board'
  AND t.description = 'Change reviews, release windows, and risk management'
  AND t.code = '变更委员会'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '变更委员会' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '安全运营', description = '安全事件响应、漏洞和合规运营'
WHERE t.name = 'Security Operations'
  AND t.description = 'Security incident response, vulnerability management, and compliance'
  AND t.code = '安全运营'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全运营' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '应用支持', description = '企业应用、接口和中间件支持'
WHERE t.name = 'Application Support'
  AND t.description = 'Enterprise application, API, and middleware support'
  AND t.code = '应用支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '应用支持' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '云平台运维', description = '公有云、私有云和容器平台运维'
WHERE t.name = 'Cloud Operations'
  AND t.description = 'Public cloud, private cloud, and container platform operations'
  AND t.code = '云平台运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '云平台运维' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '数据库运维', description = '数据库实例、备份和性能运维'
WHERE t.name = 'Database Operations'
  AND t.description = 'Database instance, backup, and performance operations'
  AND t.code = '数据库运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '数据库运维' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '网络运维', description = '网络设备、链路和安全域运维'
WHERE t.name = 'Network Operations'
  AND t.description = 'Network device, connectivity, and security zone operations'
  AND t.code = '网络运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '网络运维' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '服务器运维', description = '服务器、虚拟化和操作系统运维'
WHERE t.name = 'Server Operations'
  AND t.description = 'Server, virtualization, and operating system operations'
  AND t.code = '服务器运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务器运维' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '服务台-L3', description = '三线技术专家，处理疑难问题和根因分析'
WHERE t.name = 'Service Desk L3'
  AND t.description = 'Third-line specialists for complex problems and root cause analysis'
  AND t.code = '服务台-l3'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务台-L3' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '服务台-L2', description = '二线技术支持，处理复杂事件和升级工单'
WHERE t.name = 'Service Desk L2'
  AND t.description = 'Second-line technical support for complex incidents and escalated tickets'
  AND t.code = '服务台-l2'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务台-L2' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '服务台-L1', description = '一线服务支持，处理咨询、账号和标准请求'
WHERE t.name = 'Service Desk L1'
  AND t.description = 'First-line support for inquiries, accounts, and standard requests'
  AND t.code = '服务台-l1'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务台-L1' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '安全运营', description = '安全监控与响应'
WHERE t.name = 'Security Operations'
  AND t.description = 'Security monitoring and response'
  AND t.code = 'OPS-SEC'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全运营' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT项目管理', description = 'IT项目管理办公室'
WHERE t.name = 'IT Project Management'
  AND t.description = 'IT project management office'
  AND t.code = 'IT-PMO'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT项目管理' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = '已关闭工单', description = '已完成的工单'
WHERE t.name = 'Closed Tickets'
  AND t.description = 'Completed tickets'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","closed_at"]'::jsonb
  AND t.is_shared = false
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '已关闭工单' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = '未分配工单', description = '尚未分配的工单'
WHERE t.name = 'Unassigned Tickets'
  AND t.description = 'Tickets awaiting assignment'
  AND t.columns::jsonb = '["id","title","priority","status","creator","created_at"]'::jsonb
  AND t.is_shared = true
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '未分配工单' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = '紧急工单', description = '紧急和高优先级工单'
WHERE t.name = 'Urgent Tickets'
  AND t.description = 'Urgent and high priority tickets'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","created_at"]'::jsonb
  AND t.is_shared = true
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '紧急工单' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = '我创建的工单', description = '我提交的工单'
WHERE t.name = 'Tickets I Created'
  AND t.description = 'Tickets submitted by me'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","created_at"]'::jsonb
  AND t.is_shared = false
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '我创建的工单' AND sibling.id <> t.id);

UPDATE ticket_views AS t
SET name = '我的待办工单', description = '分配给我的未关闭工单'
WHERE t.name = 'My Pending Tickets'
  AND t.description = 'Open tickets assigned to me'
  AND t.columns::jsonb = '["id","title","priority","status","assignee","created_at"]'::jsonb
  AND t.is_shared = false
  AND NOT EXISTS (SELECT 1 FROM ticket_views AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '我的待办工单' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = '服务请求审批', description = '高价值服务请求需要审批', nodes = '[{"approver_type":"manager","name":"服务审批","timeout":120,"type":"approval"}]'::jsonb
WHERE t.name = 'Service Request Approval'
  AND t.description = 'High value service requests require approval'
  AND t.nodes::jsonb = '[{"approver_type":"manager","name":"Service Approval","timeout":120,"type":"approval"}]'::jsonb
  AND t.ticket_type = 'service_request'
  AND t.priority = 'high'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务请求审批' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = '变更审批', description = '所有变更请求需要多级审批', nodes = '[{"approver_type":"role","name":"技术审批","role":"engineer","timeout":240,"type":"approval"},{"approver_type":"role","name":"经理审批","role":"manager","timeout":480,"type":"approval"}]'::jsonb
WHERE t.name = 'Change Approval'
  AND t.description = 'All change requests require multilevel approval'
  AND t.nodes::jsonb = '[{"approver_type":"role","name":"Technical Approval","role":"engineer","timeout":240,"type":"approval"},{"approver_type":"role","name":"Manager Approval","role":"manager","timeout":480,"type":"approval"}]'::jsonb
  AND t.ticket_type = 'change'
  AND t.priority = ''
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '变更审批' AND sibling.id <> t.id);

UPDATE approval_workflows AS t
SET name = 'P0/P1事件审批', description = '紧急和高优先级事件需要主管审批', nodes = '[{"approver_type":"manager","name":"主管审批","timeout":60,"type":"approval"}]'::jsonb
WHERE t.name = 'P0/P1 Incident Approval'
  AND t.description = 'Urgent and high priority incidents require supervisor approval'
  AND t.nodes::jsonb = '[{"approver_type":"manager","name":"Supervisor Approval","timeout":60,"type":"approval"}]'::jsonb
  AND t.ticket_type = 'incident'
  AND t.priority = 'urgent,high'
  AND NOT EXISTS (SELECT 1 FROM approval_workflows AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'P0/P1事件审批' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '安全渗透测试', description = '申请系统安全渗透测试'
WHERE t.name = 'Security penetration testing'
  AND t.description = 'Request a security penetration test'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 5
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全渗透测试' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Web 漏洞扫描', description = '申请 Web 应用安全漏洞扫描'
WHERE t.name = 'Web Vulnerability Scan'
  AND t.description = 'Request a web application vulnerability scan'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Web 漏洞扫描' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '测试环境申请', description = '申请预发布测试环境'
WHERE t.name = 'Test Environment Request'
  AND t.description = 'Request a staging test environment'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 2
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '测试环境申请' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'Git 仓库创建', description = '申请创建新的 Git 代码仓库'
WHERE t.name = 'Git Repository Creation'
  AND t.description = 'Request a new Git repository'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 0
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'Git 仓库创建' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '内部域名注册', description = '注册内部系统域名'
WHERE t.name = 'Internal domain registration'
  AND t.description = 'Register an internal system domain'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 3
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '内部域名注册' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '办公网络接入', description = '申请办公网络接入权限'
WHERE t.name = 'Office Network Access'
  AND t.description = 'Request office network access'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 2
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '办公网络接入' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '业务权限变更', description = '业务系统、数据库或基础设施权限调整'
WHERE t.name = 'Business Access Change'
  AND t.description = 'Change application, database, or infrastructure access'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 2
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '业务权限变更' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '新员工账户开通', description = '新员工 IT 账户与权限开通'
WHERE t.name = 'New Employee Account Setup'
  AND t.description = 'Set up IT accounts and access for new employees'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '新员工账户开通' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '标准软件安装', description = '安装公司授权的标准软件'
WHERE t.name = 'Standard Software Installation'
  AND t.description = 'Install approved corporate software'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '标准软件安装' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '密码重置', description = '账户密码重置服务'
WHERE t.name = 'Password Reset'
  AND t.description = 'Account password reset service'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 0
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '密码重置' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'IT 问题咨询', description = 'IT 相关问题咨询与解答'
WHERE t.name = 'IT Consultation'
  AND t.description = 'IT questions and assistance'
  AND t.business_sub_type = 'ticket'
  AND t.process_definition_key = 'ticket_general_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 0
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT 问题咨询' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = '数据库备份恢复', description = '申请数据库备份恢复服务'
WHERE t.name = 'Database Backup Recovery'
  AND t.description = 'Request database backup recovery'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 2
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '数据库备份恢复' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'RDS 实例申请', description = '申请新建数据库实例'
WHERE t.name = 'RDS Instance Request'
  AND t.description = 'Request a new database instance'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'RDS 实例申请' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'ECS 故障报修', description = 'ECS 实例运行异常报修'
WHERE t.name = 'ECS Incident Report'
  AND t.description = 'Report an ECS instance incident'
  AND t.business_sub_type = 'incident'
  AND t.process_definition_key = 'incident_emergency_flow'
  AND t.requires_approval = false
  AND t.estimated_days = 0
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ECS 故障报修' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'ECS 配置变更', description = '变更现有 ECS 实例规格'
WHERE t.name = 'ECS Configuration Change'
  AND t.description = 'Change the specifications of an existing ECS instance'
  AND t.business_sub_type = 'change'
  AND t.process_definition_key = 'change_normal_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ECS 配置变更' AND sibling.id <> t.id);

UPDATE service_catalog_items AS t
SET name = 'ECS 实例申请', description = '申请新建弹性云服务器实例'
WHERE t.name = 'ECS Instance Request'
  AND t.description = 'Request a new elastic cloud server instance'
  AND t.business_sub_type = 'service_request'
  AND t.process_definition_key = 'service_request_flow'
  AND t.requires_approval = true
  AND t.estimated_days = 1
  AND NOT EXISTS (SELECT 1 FROM service_catalog_items AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ECS 实例申请' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'API网关', description = 'API接口管理'
WHERE t.name = 'API Gateway'
  AND t.description = 'API management'
  AND t.category = '开发'
  AND t.service_type = 'custom'
  AND t.delivery_time = 3
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'API网关' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '测试环境', description = '预发布测试环境'
WHERE t.name = 'Test Environment'
  AND t.description = 'Staging test environment'
  AND t.category = '开发'
  AND t.service_type = 'custom'
  AND t.delivery_time = 2
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '测试环境' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'CI/CD流水线', description = '自动化部署'
WHERE t.name = 'CI/CD Pipeline'
  AND t.description = 'Automated deployment'
  AND t.category = '开发'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'CI/CD流水线' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '代码仓库', description = 'Git代码仓库'
WHERE t.name = 'Code Repository'
  AND t.description = 'Git code repository'
  AND t.category = '开发'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '代码仓库' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '域名申请', description = '内部域名注册'
WHERE t.name = 'Domain Request'
  AND t.description = 'Internal domain registration'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 3
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '域名申请' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '网络接入', description = '网络接入申请'
WHERE t.name = 'Network Access'
  AND t.description = 'Network access request'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 2
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '网络接入' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '账户申请', description = '新员工账户开通'
WHERE t.name = 'Account Request'
  AND t.description = 'New Employee Account Setup'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '账户申请' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '软件安装', description = '标准软件安装'
WHERE t.name = 'Software Installation'
  AND t.description = 'Standard Software Installation'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 1
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '软件安装' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'IT服务台', description = 'IT问题咨询支持'
WHERE t.name = 'IT Service Desk'
  AND t.description = 'IT consultation and support'
  AND t.category = '支持'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT服务台' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '等保合规', description = '等级保护咨询'
WHERE t.name = 'Security Compliance Assessment'
  AND t.description = 'Classified protection compliance consulting'
  AND t.category = '安全'
  AND t.service_type = 'security'
  AND t.delivery_time = 30
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '等保合规' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '渗透测试', description = '安全渗透测试'
WHERE t.name = 'Penetration Testing'
  AND t.description = 'Security penetration testing'
  AND t.category = '安全'
  AND t.service_type = 'security'
  AND t.delivery_time = 5
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '渗透测试' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '漏洞扫描', description = 'Web漏洞扫描'
WHERE t.name = 'Vulnerability Scanning'
  AND t.description = 'Web vulnerability scanning'
  AND t.category = '安全'
  AND t.service_type = 'security'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '漏洞扫描' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '企业IM', description = '即时通讯工具'
WHERE t.name = 'Enterprise Messaging'
  AND t.description = 'Instant messaging'
  AND t.category = '通讯'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '企业IM' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '视频会议', description = '高清视频会议'
WHERE t.name = 'Video Conferencing'
  AND t.description = 'HD video conferencing'
  AND t.category = '通讯'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '视频会议' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '企业网盘', description = '文件存储共享'
WHERE t.name = 'Enterprise File Storage'
  AND t.description = 'File storage and sharing'
  AND t.category = '协作'
  AND t.service_type = 'custom'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '企业网盘' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '企业邮箱', description = '企业域名邮箱'
WHERE t.name = 'Enterprise Email'
  AND t.description = 'Corporate domain email'
  AND t.category = '通讯'
  AND t.service_type = 'custom'
  AND t.delivery_time = 1
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '企业邮箱' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'VPN 网关', description = 'VPN加密通道'
WHERE t.name = 'VPN Gateway'
  AND t.description = 'Encrypted VPN tunnel'
  AND t.category = '安全'
  AND t.service_type = 'security'
  AND t.delivery_time = 2
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'VPN 网关' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '负载均衡 SLB', description = '流量分发服务'
WHERE t.name = 'SLB Load Balancer'
  AND t.description = 'Traffic distribution service'
  AND t.category = '网络'
  AND t.service_type = 'network'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '负载均衡 SLB' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = 'CDN 加速', description = '内容分发加速'
WHERE t.name = 'CDN Acceleration'
  AND t.description = 'Content delivery acceleration'
  AND t.category = '网络'
  AND t.service_type = 'network'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'CDN 加速' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '对象存储 OSS', description = '海量云存储'
WHERE t.name = 'OSS Object Storage'
  AND t.description = 'Scalable cloud storage'
  AND t.category = '存储'
  AND t.service_type = 'oss'
  AND t.delivery_time = 0
  AND t.requires_approval = false
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '对象存储 OSS' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '云数据库 RDS', description = 'MySQL/PostgreSQL数据库'
WHERE t.name = 'RDS Cloud Database'
  AND t.description = 'MySQL/PostgreSQL database'
  AND t.category = '数据库'
  AND t.service_type = 'rds'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '云数据库 RDS' AND sibling.id <> t.id);

UPDATE service_catalogs AS t
SET name = '云服务器 ECS', description = '弹性云服务器'
WHERE t.name = 'ECS Cloud Server'
  AND t.description = 'Elastic cloud server'
  AND t.category = '云计算'
  AND t.service_type = 'vm'
  AND t.delivery_time = 1
  AND t.requires_approval = true
  AND NOT EXISTS (SELECT 1 FROM service_catalogs AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '云服务器 ECS' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-变更', description = '变更请求SLA'
WHERE t.name = 'SLA-Change'
  AND t.description = 'Change request SLA'
  AND t.service_type = 'change'
  AND t.priority = 'high'
  AND t.response_time = 60
  AND t.resolution_time = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-变更' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-服务请求', description = '服务请求标准SLA'
WHERE t.name = 'SLA-Service Request'
  AND t.description = 'Standard service request SLA'
  AND t.service_type = 'service_request'
  AND t.priority = 'medium'
  AND t.response_time = 480
  AND t.resolution_time = 4320
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-服务请求' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-P3-低', description = 'P3低级别SLA'
WHERE t.name = 'SLA-P3-Low'
  AND t.description = 'P3 low priority SLA'
  AND t.service_type = 'incident'
  AND t.priority = 'low'
  AND t.response_time = 240
  AND t.resolution_time = 1440
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-P3-低' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-P2-中', description = 'P2中级别SLA'
WHERE t.name = 'SLA-P2-Medium'
  AND t.description = 'P2 medium priority SLA'
  AND t.service_type = 'incident'
  AND t.priority = 'medium'
  AND t.response_time = 120
  AND t.resolution_time = 480
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-P2-中' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-P1-高', description = 'P1高级别SLA'
WHERE t.name = 'SLA-P1-High'
  AND t.description = 'P1 high priority SLA'
  AND t.service_type = 'incident'
  AND t.priority = 'high'
  AND t.response_time = 30
  AND t.resolution_time = 240
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-P1-高' AND sibling.id <> t.id);

UPDATE sla_definitions AS t
SET name = 'SLA-P0-紧急', description = 'P0紧急级别SLA'
WHERE t.name = 'SLA-P0-Urgent'
  AND t.description = 'P0 urgent SLA'
  AND t.service_type = 'incident'
  AND t.priority = 'urgent'
  AND t.response_time = 15
  AND t.resolution_time = 120
  AND NOT EXISTS (SELECT 1 FROM sla_definitions AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'SLA-P0-紧急' AND sibling.id <> t.id);

UPDATE groups AS t
SET description = '变更审批组：从 change_manager / manager 角色人员中拉人组成（变更委员会）'
WHERE t.description = 'Change advisory board: add members with change_manager or manager roles'
  AND t.name = 'approvers-change';

UPDATE groups AS t
SET description = '安全审批组：从 security_admin 角色人员中拉人组成'
WHERE t.description = 'Security approval group: add members with the security_admin role'
  AND t.name = 'approvers-security';

UPDATE groups AS t
SET description = '管理审批组：从 manager 角色人员中拉人组成'
WHERE t.description = 'Management approval group: add members with the manager role'
  AND t.name = 'approvers-managers';

UPDATE groups AS t
SET description = '三线审批组：从 it_admin / security_admin 角色人员中拉人组成'
WHERE t.description = 'Third-line approval group: add members with it_admin or security_admin roles'
  AND t.name = 'approvers-l3';

UPDATE groups AS t
SET description = '二线审批组：从 technician / it_admin 角色人员中拉人组成'
WHERE t.description = 'Second-line approval group: add members with technician or it_admin roles'
  AND t.name = 'approvers-l2';

UPDATE groups AS t
SET description = '一线审批组：从 agent 角色人员中拉人组成'
WHERE t.description = 'First-line approval group: add members with the agent role'
  AND t.name = 'approvers-l1';

UPDATE roles AS t
SET name = 'MSP管理员', description = 'MSP 全量协作管理'
WHERE t.name = 'MSP Administrator'
  AND t.description = 'Full MSP collaboration management'
  AND t.code = 'msp_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP管理员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP经理', description = 'MSP 交付与资源分配管理'
WHERE t.name = 'MSP Manager'
  AND t.description = 'MSP delivery and resource allocation'
  AND t.code = 'msp_manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP经理' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP专家', description = 'MSP 专家级处理与客户维护'
WHERE t.name = 'MSP Specialist'
  AND t.description = 'MSP specialist support and customer management'
  AND t.code = 'msp_specialist'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP专家' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP技术员', description = 'MSP 派单处理协作'
WHERE t.name = 'MSP Technician'
  AND t.description = 'MSP ticket assignment and handling'
  AND t.code = 'msp_tech'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP技术员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = 'MSP查看者', description = 'MSP 客户工单只读协作'
WHERE t.name = 'MSP Viewer'
  AND t.description = 'Read-only collaboration on MSP customer tickets'
  AND t.code = 'msp_viewer'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'MSP查看者' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '服务目录管理员', description = '服务目录、请求模板与工单模板配置'
WHERE t.name = 'Service Catalog Administrator'
  AND t.description = 'Service catalogs, request templates, and ticket templates'
  AND t.code = 'service_catalog_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务目录管理员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '配置管理员', description = 'CI 类型/实例/关系与对账'
WHERE t.name = 'Configuration Administrator'
  AND t.description = 'CI types, instances, relationships, and reconciliation'
  AND t.code = 'cmdb_admin'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '配置管理员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '知识管理员', description = '知识文章生命周期与评审发布'
WHERE t.name = 'Knowledge Administrator'
  AND t.description = 'Knowledge article lifecycle, review, and publishing'
  AND t.code = 'knowledge_manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '知识管理员' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '问题经理', description = '根因分析、已知错误与变通方案'
WHERE t.name = 'Problem Manager'
  AND t.description = 'Root cause analysis, known errors, and workarounds'
  AND t.code = 'problem_manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '问题经理' AND sibling.id <> t.id);

UPDATE roles AS t
SET name = '变更经理', description = '变更生命周期、CAB 与发布联动'
WHERE t.name = 'Change Manager'
  AND t.description = 'Change lifecycle, CAB reviews, and release coordination'
  AND t.code = 'change_manager'
  AND NOT EXISTS (SELECT 1 FROM roles AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '变更经理' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '技术支持', description = '客户服务技术支持'
WHERE t.name = 'Technical Support'
  AND t.description = 'Customer technical support'
  AND t.code = '技术支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '技术支持' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '客户成功', description = '客户成功管理'
WHERE t.name = 'Customer Success'
  AND t.description = 'Customer success management'
  AND t.code = '客户成功'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '客户成功' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '测试团队', description = '测试与质量保证'
WHERE t.name = 'Testing Team'
  AND t.description = 'Testing and quality assurance'
  AND t.code = '测试团队'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '测试团队' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '移动开发', description = '移动端开发团队'
WHERE t.name = 'Mobile Development'
  AND t.description = 'Mobile development team'
  AND t.code = '移动开发'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '移动开发' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '前端开发', description = '前端开发团队'
WHERE t.name = 'Frontend Development'
  AND t.description = 'Frontend development team'
  AND t.code = '前端开发'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '前端开发' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '后端开发', description = '后端开发团队'
WHERE t.name = 'Backend Development'
  AND t.description = 'Backend development team'
  AND t.code = '后端开发'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '后端开发' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '安全合规', description = '安全合规管理'
WHERE t.name = 'Security Compliance'
  AND t.description = 'Security compliance management'
  AND t.code = '安全合规'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全合规' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '安全运营', description = '安全监控与响应'
WHERE t.name = 'Security Operations'
  AND t.description = 'Security monitoring and response'
  AND t.code = '安全运营'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '安全运营' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'OA支持', description = 'OA办公系统支持'
WHERE t.name = 'Office Applications Support'
  AND t.description = 'Office application support'
  AND t.code = 'oa支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'OA支持' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'CRM支持', description = 'CRM系统支持'
WHERE t.name = 'CRM Support'
  AND t.description = 'CRM system support'
  AND t.code = 'crm支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'CRM支持' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = 'ERP支持', description = 'ERP系统支持'
WHERE t.name = 'ERP Support'
  AND t.description = 'ERP system support'
  AND t.code = 'erp支持'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'ERP支持' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '云平台运维', description = '云计算平台运维'
WHERE t.name = 'Cloud Operations'
  AND t.description = 'Cloud platform operations'
  AND t.code = '云平台运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '云平台运维' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '数据库运维', description = '数据库运维管理'
WHERE t.name = 'Database Operations'
  AND t.description = 'Database operations management'
  AND t.code = '数据库运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '数据库运维' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '网络运维', description = '网络设备运维'
WHERE t.name = 'Network Operations'
  AND t.description = 'Network device operations'
  AND t.code = '网络运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '网络运维' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '服务器运维', description = '服务器运维管理'
WHERE t.name = 'Server Operations'
  AND t.description = 'Server operations management'
  AND t.code = '服务器运维'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务器运维' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '服务台-L3', description = '三线技术专家'
WHERE t.name = 'Service Desk L3'
  AND t.description = 'Third-line technical specialists'
  AND t.code = '服务台-l3'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务台-L3' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '服务台-L2', description = '二线技术支持'
WHERE t.name = 'Service Desk L2'
  AND t.description = 'Second-line technical support'
  AND t.code = '服务台-l2'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务台-L2' AND sibling.id <> t.id);

UPDATE teams AS t
SET name = '服务台-L1', description = '一线服务支持'
WHERE t.name = 'Service Desk L1'
  AND t.description = 'First-line support'
  AND t.code = '服务台-l1'
  AND NOT EXISTS (SELECT 1 FROM teams AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务台-L1' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '行政部', description = '行政管理'
WHERE t.name = 'Administration'
  AND t.description = 'Administration management'
  AND t.code = 'ADMIN'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '行政部' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '财务部', description = '财务管理'
WHERE t.name = 'Finance'
  AND t.description = 'Financial management'
  AND t.code = 'FIN'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '财务部' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '人力资源部', description = '人力资源管理'
WHERE t.name = 'Human Resources'
  AND t.description = 'Human resources management'
  AND t.code = 'HR'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '人力资源部' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '测试部', description = '质量保证'
WHERE t.name = 'Quality Assurance'
  AND t.description = 'Quality assurance'
  AND t.code = 'QA'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '测试部' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '研发部', description = '产品研发'
WHERE t.name = 'Research and Development'
  AND t.description = 'Product development'
  AND t.code = 'RD'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '研发部' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '客户服务', description = '客户服务体验'
WHERE t.name = 'Customer Service'
  AND t.description = 'Customer service experience'
  AND t.code = 'OPS-CS'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '客户服务' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '运维中心', description = '7x24运维监控'
WHERE t.name = 'Operations Center'
  AND t.description = '24/7 operations monitoring'
  AND t.code = 'OPS-NOC'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '运维中心' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '服务台', description = '一线服务支持'
WHERE t.name = 'Service Desk'
  AND t.description = 'First-line support'
  AND t.code = 'OPS-SD'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '服务台' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '运营管理部', description = 'IT运营管理'
WHERE t.name = 'Operations'
  AND t.description = 'IT operations management'
  AND t.code = 'OPS'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '运营管理部' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT项目管理', description = 'IT项目管理'
WHERE t.name = 'IT Project Management'
  AND t.description = 'IT Project Management'
  AND t.code = 'IT-PMO'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT项目管理' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT安全', description = '信息安全管理'
WHERE t.name = 'IT Security'
  AND t.description = 'Information security management'
  AND t.code = 'IT-SEC'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT安全' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT应用服务', description = '应用系统运维'
WHERE t.name = 'IT Applications'
  AND t.description = 'Application operations'
  AND t.code = 'IT-APP'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT应用服务' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = 'IT基础架构', description = '基础设施运维'
WHERE t.name = 'IT Infrastructure'
  AND t.description = 'Infrastructure operations'
  AND t.code = 'IT-INFRA'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = 'IT基础架构' AND sibling.id <> t.id);

UPDATE departments AS t
SET name = '信息技术部', description = 'IT整体管理'
WHERE t.name = 'Information Technology'
  AND t.description = 'IT management'
  AND t.code = 'IT'
  AND NOT EXISTS (SELECT 1 FROM departments AS sibling WHERE sibling.tenant_id = t.tenant_id AND sibling.name = '信息技术部' AND sibling.id <> t.id);
