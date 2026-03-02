/**
 * 工单类型预设配置
 * 企业实际使用中的常见工单场景
 */

export interface TicketTypePreset {
  id: string;
  code: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  category: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  // 关联的工作流模板ID（对应工作流定义的code）
  workflowTemplateId?: string;
  // 预设的表单字段配置
  fields?: {
    name: string;
    label: string;
    type: 'text' | 'textarea' | 'select' | 'number' | 'date';
    required?: boolean;
    placeholder?: string;
    options?: { label: string; value: string }[];
  }[];
}

export const ticketTypePresets: TicketTypePreset[] = [
  // ==================== 运维类 ====================
  {
    id: 'k8s-scale',
    code: 'k8s_scale',
    name: 'K8S扩缩容',
    description: 'Kubernetes容器集群扩容或缩容请求',
    icon: 'Container',
    color: '#1890ff',
    category: '技术支持',
    priority: 'high',
    workflowTemplateId: 'ticket_urgent_flow',
    fields: [
      {
        name: 'cluster',
        label: '集群名称',
        type: 'select',
        required: true,
        placeholder: '请选择集群',
        options: [
          { label: '生产集群', value: 'prod' },
          { label: '测试集群', value: 'test' },
          { label: '开发集群', value: 'dev' },
        ],
      },
      {
        name: 'namespace',
        label: '命名空间',
        type: 'text',
        required: true,
        placeholder: '例如: default',
      },
      {
        name: 'current_replicas',
        label: '当前副本数',
        type: 'number',
        required: true,
      },
      {
        name: 'target_replicas',
        label: '目标副本数',
        type: 'number',
        required: true,
      },
      {
        name: 'reason',
        label: '扩缩容原因',
        type: 'textarea',
        required: true,
        placeholder: '请说明扩缩容的业务原因',
      },
    ],
  },
  {
    id: 'ddl-execute',
    code: 'ddl_execute',
    name: 'DDL执行',
    description: '数据库表结构变更、索引创建等DDL操作',
    icon: 'Database',
    color: '#722ed1',
    category: '技术支持',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'db_type',
        label: '数据库类型',
        type: 'select',
        required: true,
        options: [
          { label: 'MySQL', value: 'mysql' },
          { label: 'PostgreSQL', value: 'postgresql' },
          { label: 'Redis', value: 'redis' },
          { label: 'MongoDB', value: 'mongodb' },
        ],
      },
      {
        name: 'db_name',
        label: '数据库名称',
        type: 'text',
        required: true,
      },
      {
        name: 'table_name',
        label: '表名',
        type: 'text',
        required: true,
      },
      {
        name: 'ddl_content',
        label: 'DDL语句',
        type: 'textarea',
        required: true,
        placeholder: '请输入完整的DDL语句',
      },
      {
        name: 'execute_time',
        label: '执行时间',
        type: 'select',
        required: true,
        options: [
          { label: '立即执行', value: 'now' },
          { label: '维护窗口(02:00-04:00)', value: 'maintenance' },
          { label: '指定时间', value: 'scheduled' },
        ],
      },
      {
        name: 'backup_required',
        label: '是否需要备份',
        type: 'select',
        required: true,
        options: [
          { label: '是', value: 'yes' },
          { label: '否', value: 'no' },
        ],
      },
    ],
  },
  {
    id: 'data-export',
    code: 'data_export',
    name: '数据导出',
    description: '从数据库或系统导出数据',
    icon: 'Download',
    color: '#13c2c2',
    category: '技术支持',
    priority: 'low',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'data_source',
        label: '数据源',
        type: 'select',
        required: true,
        options: [
          { label: 'MySQL', value: 'mysql' },
          { label: 'PostgreSQL', value: 'postgresql' },
          { label: 'MongoDB', value: 'mongodb' },
          { label: '业务系统', value: 'business' },
        ],
      },
      {
        name: 'table_name',
        label: '表/视图名',
        type: 'text',
        required: true,
      },
      {
        name: 'columns',
        label: '导出字段',
        type: 'text',
        placeholder: '用逗号分隔，如: id,name,email',
      },
      {
        name: 'condition',
        label: '筛选条件',
        type: 'textarea',
        placeholder: 'WHERE条件，如: status=1 AND created_at > "2024-01-01"',
      },
      {
        name: 'export_format',
        label: '导出格式',
        type: 'select',
        required: true,
        options: [
          { label: 'Excel', value: 'xlsx' },
          { label: 'CSV', value: 'csv' },
          { label: 'JSON', value: 'json' },
        ],
      },
      {
        name: 'email',
        label: '接收邮箱',
        type: 'text',
        required: true,
        placeholder: '导出完成后发送到此邮箱',
      },
    ],
  },
  {
    id: 'vm-apply',
    code: 'vm_apply',
    name: '虚拟机申请',
    description: '申请新的虚拟机资源',
    icon: 'Desktop',
    color: '#2f54eb',
    category: '技术支持',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'vm_name',
        label: '虚拟机名称',
        type: 'text',
        required: true,
        placeholder: '建议使用英文名称',
      },
      {
        name: 'os_type',
        label: '操作系统',
        type: 'select',
        required: true,
        options: [
          { label: 'CentOS 7.x', value: 'centos7' },
          { label: 'CentOS 8.x', value: 'centos8' },
          { label: 'Ubuntu 20.04', value: 'ubuntu20' },
          { label: 'Ubuntu 22.04', value: 'ubuntu22' },
          { label: 'Windows Server 2019', value: 'win2019' },
          { label: 'Windows Server 2022', value: 'win2022' },
        ],
      },
      {
        name: 'cpu_cores',
        label: 'CPU核心数',
        type: 'select',
        required: true,
        options: [
          { label: '1核', value: '1' },
          { label: '2核', value: '2' },
          { label: '4核', value: '4' },
          { label: '8核', value: '8' },
          { label: '16核', value: '16' },
        ],
      },
      {
        name: 'memory',
        label: '内存大小',
        type: 'select',
        required: true,
        options: [
          { label: '1GB', value: '1' },
          { label: '2GB', value: '2' },
          { label: '4GB', value: '4' },
          { label: '8GB', value: '8' },
          { label: '16GB', value: '16' },
          { label: '32GB', value: '32' },
        ],
      },
      {
        name: 'disk_size',
        label: '磁盘大小(GB)',
        type: 'number',
        required: true,
        placeholder: '40',
      },
      {
        name: 'network',
        label: '网络类型',
        type: 'select',
        required: true,
        options: [
          { label: '办公网', value: 'office' },
          { label: '生产网', value: 'prod' },
          { label: 'DMZ', value: 'dmz' },
        ],
      },
      {
        name: 'purpose',
        label: '用途说明',
        type: 'textarea',
        required: true,
      },
    ],
  },

  // ==================== 账号类 ====================
  {
    id: 'account-apply',
    code: 'account_apply',
    name: '账号申请',
    description: '申请系统账号、VPN账号、堡垒机账号等',
    icon: 'User',
    color: '#52c41a',
    category: '账户问题',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'account_type',
        label: '账号类型',
        type: 'select',
        required: true,
        options: [
          { label: 'VPN账号', value: 'vpn' },
          { label: '堡垒机账号', value: 'bastion' },
          { label: 'GitLab账号', value: 'gitlab' },
          { label: 'Jenkins账号', value: 'jenkins' },
          { label: 'Jira账号', value: 'jira' },
          { label: '阿里云账号', value: 'aliyun' },
          { label: 'AWS账号', value: 'aws' },
          { label: '其他', value: 'other' },
        ],
      },
      {
        name: 'username',
        label: '申请用户名',
        type: 'text',
        required: true,
        placeholder: '建议使用工号或姓名拼音',
      },
      {
        name: 'department',
        label: '所属部门',
        type: 'text',
        required: true,
      },
      {
        name: 'employment_type',
        label: '人员类型',
        type: 'select',
        required: true,
        options: [
          { label: '正式员工', value: 'fulltime' },
          { label: '外包员工', value: 'outsourced' },
          { label: '实习生', value: 'intern' },
          { label: '合作伙伴', value: 'partner' },
        ],
      },
      {
        name: 'access_duration',
        label: '有效期限',
        type: 'select',
        required: true,
        options: [
          { label: '3个月', value: '3m' },
          { label: '6个月', value: '6m' },
          { label: '1年', value: '1y' },
          { label: '长期', value: 'permanent' },
        ],
      },
      {
        name: 'reason',
        label: '申请原因',
        type: 'textarea',
        required: true,
      },
      {
        name: 'approver',
        label: '审批人',
        type: 'text',
        required: true,
        placeholder: '部门负责人或项目负责人',
      },
    ],
  },
  {
    id: 'gitlab-repo-apply',
    code: 'gitlab_repo_apply',
    name: 'GitLab代码仓库申请',
    description: '申请创建新的GitLab代码仓库',
    icon: 'Code',
    color: '#fa541c',
    category: '账户问题',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'repo_name',
        label: '仓库名称',
        type: 'text',
        required: true,
        placeholder: '建议使用英文名称，如: backend-api',
      },
      {
        name: 'repo_type',
        label: '仓库类型',
        type: 'select',
        required: true,
        options: [
          { label: '公开(Public)', value: 'public' },
          { label: '内部(Internal)', value: 'internal' },
          { label: '私有(Private)', value: 'private' },
        ],
      },
      {
        name: 'description',
        label: '仓库描述',
        type: 'textarea',
        placeholder: '简要描述项目用途',
      },
      {
        name: 'group_path',
        label: '所属组',
        type: 'text',
        required: true,
        placeholder: '如: backend-team',
      },
      {
        name: 'initial_members',
        label: '初始成员',
        type: 'textarea',
        required: true,
        placeholder: '每行一个成员用户名',
      },
      {
        name: 'has_ci',
        label: '需要CI/CD',
        type: 'select',
        required: true,
        options: [
          { label: '是', value: 'yes' },
          { label: '否', value: 'no' },
        ],
      },
      {
        name: 'has_k8s_deploy',
        label: '需要K8S部署',
        type: 'select',
        options: [
          { label: '是', value: 'yes' },
          { label: '否', value: 'no' },
        ],
      },
    ],
  },

  // ==================== 网络类 ====================
  {
    id: 'domain-apply',
    code: 'domain_apply',
    name: '域名申请',
    description: '申请新的域名或域名解析变更',
    icon: 'Global',
    color: '#eb2f96',
    category: '系统故障',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'domain_name',
        label: '申请域名',
        type: 'text',
        required: true,
        placeholder: '如: api.example.com',
      },
      {
        name: 'domain_type',
        label: '域名类型',
        type: 'select',
        required: true,
        options: [
          { label: '新域名申请', value: 'new' },
          { label: '解析变更', value: 'change' },
          { label: 'SSL证书申请', value: 'ssl' },
          { label: '域名续费', value: 'renew' },
        ],
      },
      {
        name: 'target_ip',
        label: '指向IP',
        type: 'text',
        required: true,
        placeholder: '域名需要解析到的IP地址',
      },
      {
        name: 'record_type',
        label: '记录类型',
        type: 'select',
        required: true,
        options: [
          { label: 'A记录', value: 'A' },
          { label: 'CNAME', value: 'CNAME' },
          { label: 'MX记录', value: 'MX' },
          { label: 'TXT记录', value: 'TXT' },
        ],
      },
      {
        name: 'purpose',
        label: '用途说明',
        type: 'textarea',
        required: true,
      },
      {
        name: 'cert_needed',
        label: '是否需要SSL证书',
        type: 'select',
        options: [
          { label: '是', value: 'yes' },
          { label: '否', value: 'no' },
        ],
      },
    ],
  },
  {
    id: 'firewall-apply',
    code: 'firewall_apply',
    name: '防火墙规则申请',
    description: '申请开放或变更防火墙端口规则',
    icon: 'Safety',
    color: '#fa8c16',
    category: '系统故障',
    priority: 'high',
    workflowTemplateId: 'ticket_urgent_flow',
    fields: [
      {
        name: 'rule_type',
        label: '规则类型',
        type: 'select',
        required: true,
        options: [
          { label: '开放端口', value: 'open' },
          { label: '限制端口', value: 'restrict' },
          { label: 'IP白名单', value: 'whitelist' },
          { label: 'IP黑名单', value: 'blacklist' },
        ],
      },
      {
        name: 'source_ip',
        label: '源IP/网段',
        type: 'text',
        required: true,
        placeholder: '如: 10.0.0.0/8 或特定IP',
      },
      {
        name: 'target_ip',
        label: '目标IP/网段',
        type: 'text',
        required: true,
      },
      {
        name: 'port',
        label: '端口',
        type: 'text',
        required: true,
        placeholder: '如: 8080 或 8000-9000',
      },
      {
        name: 'protocol',
        label: '协议',
        type: 'select',
        required: true,
        options: [
          { label: 'TCP', value: 'tcp' },
          { label: 'UDP', value: 'udp' },
          { label: 'TCP/UDP', value: 'both' },
        ],
      },
      {
        name: 'validity',
        label: '有效期',
        type: 'select',
        required: true,
        options: [
          { label: '临时(7天)', value: '7d' },
          { label: '1个月', value: '1m' },
          { label: '3个月', value: '3m' },
          { label: '6个月', value: '6m' },
          { label: '长期', value: 'permanent' },
        ],
      },
      {
        name: 'reason',
        label: '申请原因',
        type: 'textarea',
        required: true,
      },
    ],
  },

  // ==================== 应用类 ====================
  {
    id: 'app-apply',
    code: 'app_apply',
    name: '应用申请',
    description: '申请在K8S集群中部署新应用服务',
    icon: 'Appstore',
    color: '#1890ff',
    category: '技术支持',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'app_name',
        label: '应用名称',
        type: 'text',
        required: true,
        placeholder: '英文名称，如: order-service',
      },
      {
        name: 'app_type',
        label: '应用类型',
        type: 'select',
        required: true,
        options: [
          { label: 'Web服务', value: 'web' },
          { label: '后端API服务', value: 'api' },
          { label: '定时任务', value: 'cronjob' },
          { label: '消息消费者', value: 'consumer' },
          { label: '前端应用', value: 'frontend' },
        ],
      },
      {
        name: 'language',
        label: '开发语言',
        type: 'select',
        options: [
          { label: 'Java', value: 'java' },
          { label: 'Go', value: 'go' },
          { label: 'Node.js', value: 'nodejs' },
          { label: 'Python', value: 'python' },
          { label: '其他', value: 'other' },
        ],
      },
      {
        name: 'image_url',
        label: 'Docker镜像地址',
        type: 'text',
        required: true,
        placeholder: '如: registry.example.com/app:v1.0.0',
      },
      {
        name: 'replicas',
        label: '实例数',
        type: 'select',
        required: true,
        options: [
          { label: '1', value: '1' },
          { label: '2', value: '2' },
          { label: '3', value: '3' },
          { label: '5', value: '5' },
        ],
      },
      {
        name: 'cpu_request',
        label: 'CPU请求',
        type: 'select',
        required: true,
        options: [
          { label: '100m', value: '100m' },
          { label: '200m', value: '200m' },
          { label: '500m', value: '500m' },
          { label: '1000m', value: '1000m' },
          { label: '2000m', value: '2000m' },
        ],
      },
      {
        name: 'memory_request',
        label: '内存请求',
        type: 'select',
        required: true,
        options: [
          { label: '128Mi', value: '128Mi' },
          { label: '256Mi', value: '256Mi' },
          { label: '512Mi', value: '512Mi' },
          { label: '1Gi', value: '1Gi' },
          { label: '2Gi', value: '2Gi' },
          { label: '4Gi', value: '4Gi' },
        ],
      },
      {
        name: 'env_vars',
        label: '环境变量',
        type: 'textarea',
        placeholder: '每行一个，格式: KEY=VALUE',
      },
      {
        name: 'port',
        label: '服务端口',
        type: 'number',
        required: true,
        placeholder: '如: 8080',
      },
    ],
  },
  {
    id: 'project-apply',
    code: 'project_apply',
    name: '项目申请',
    description: '申请创建新项目或项目空间',
    icon: 'Project',
    color: '#722ed1',
    category: '技术支持',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'project_name',
        label: '项目名称',
        type: 'text',
        required: true,
      },
      {
        name: 'project_type',
        label: '项目类型',
        type: 'select',
        required: true,
        options: [
          { label: '研发项目', value: 'development' },
          { label: '运维项目', value: 'operation' },
          { label: '数据分析项目', value: 'data' },
          { label: '测试项目', value: 'testing' },
        ],
      },
      {
        name: 'description',
        label: '项目描述',
        type: 'textarea',
        required: true,
      },
      {
        name: 'team_members',
        label: '团队成员',
        type: 'textarea',
        required: true,
        placeholder: '每行一个成员，格式: 用户名(角色)',
      },
      {
        name: 'start_date',
        label: '开始日期',
        type: 'text',
        required: true,
        placeholder: 'YYYY-MM-DD',
      },
      {
        name: 'end_date',
        label: '预计结束日期',
        type: 'text',
        placeholder: 'YYYY-MM-DD',
      },
      {
        name: 'resources',
        label: '需要资源',
        type: 'textarea',
        placeholder: '如: 2台4核8G服务器、GitLab仓库、Jenkins流水线等',
      },
    ],
  },

  // ==================== 数据库类 ====================
  {
    id: 'db-account-apply',
    code: 'db_account_apply',
    name: '数据库账号申请',
    description: '申请数据库读写账号、只读账号等',
    icon: 'Key',
    color: '#faad14',
    category: '系统故障',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
    fields: [
      {
        name: 'db_type',
        label: '数据库类型',
        type: 'select',
        required: true,
        options: [
          { label: 'MySQL', value: 'mysql' },
          { label: 'PostgreSQL', value: 'postgresql' },
          { label: 'Redis', value: 'redis' },
          { label: 'MongoDB', value: 'mongodb' },
        ],
      },
      {
        name: 'db_name',
        label: '数据库名称',
        type: 'text',
        required: true,
      },
      {
        name: 'account_type',
        label: '账号类型',
        type: 'select',
        required: true,
        options: [
          { label: '读写账号', value: 'readwrite' },
          { label: '只读账号', value: 'readonly' },
          { label: '运维账号', value: 'admin' },
        ],
      },
      {
        name: 'username',
        label: '申请账号',
        type: 'text',
        required: true,
      },
      {
        name: 'host_access',
        label: '允许访问主机',
        type: 'text',
        required: true,
        placeholder: '如: % 或具体IP',
      },
      {
        name: 'tables',
        label: '需要访问的表(可选)',
        type: 'textarea',
        placeholder: '如需限制到具体表，请在此填写',
      },
      {
        name: 'reason',
        label: '申请原因',
        type: 'textarea',
        required: true,
      },
      {
        name: 'validity',
        label: '有效期',
        type: 'select',
        required: true,
        options: [
          { label: '3个月', value: '3m' },
          { label: '6个月', value: '6m' },
          { label: '1年', value: '1y' },
          { label: '长期', value: 'permanent' },
        ],
      },
    ],
  },

  // ==================== 其他常见类型 ====================
  {
    id: 'general',
    code: 'general',
    name: '其他工单',
    description: '通用工单类型，用于不属于以上分类的请求',
    icon: 'FileText',
    color: '#8c8c8c',
    category: '系统故障',
    priority: 'medium',
    workflowTemplateId: 'ticket_general_flow',
  },
];

// 按分类分组
export const ticketTypeCategories = [
  { key: 'all', label: '全部' },
  { key: '技术支持', label: '技术支持' },
  { key: '账户问题', label: '账户问题' },
  { key: '系统故障', label: '系统故障' },
];

// 获取预设类型
export function getTicketTypePreset(code: string): TicketTypePreset | undefined {
  return ticketTypePresets.find(t => t.code === code);
}

// 按分类筛选
export function getTicketTypesByCategory(category: string): TicketTypePreset[] {
  if (category === 'all') {
    return ticketTypePresets;
  }
  return ticketTypePresets.filter(t => t.category === category);
}

// 根据工单类型获取关联的工作流模板ID
export function getWorkflowTemplateIdByTicketType(ticketTypeCode: string): string | undefined {
  const preset = ticketTypePresets.find(t => t.code === ticketTypeCode);
  return preset?.workflowTemplateId;
}
