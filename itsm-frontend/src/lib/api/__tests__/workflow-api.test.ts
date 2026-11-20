import { WorkflowApi } from '@/lib/api/workflow-api';
import type { BpmnProcessVersion } from '@/lib/api/workflow-api';
import { httpClient } from '@/lib/api/http-client';

// Mock httpClient methods directly
jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    getTenantId: jest.fn().mockReturnValue(null),
    setToken: jest.fn(),
    getAuthToken: jest.fn().mockReturnValue(null),
  },
}));

// Mock console methods to avoid noise in tests
const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

describe('WorkflowApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  /** service.ProcessVersion 的完整 camelCase 镜像 */
  const bpmnVersion = (overrides: Partial<BpmnProcessVersion> = {}): BpmnProcessVersion => ({
    id: '42',
    processDefinitionKey: 'wf1',
    version: 2,
    name: 'WF',
    description: '',
    bpmnXml: '<definitions />',
    deploymentId: 'dep-1',
    isActive: true,
    createdAt: '2024-01-01T10:00:00Z',
    updatedAt: '2024-01-02T10:00:00Z',
    createdBy: '1',
    tenantId: 7,
    changeLog: '调整审批人',
    compatibilityNotes: '',
    ...overrides,
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  describe('getWorkflows', () => {
    it('should fetch workflows successfully', async () => {
      const mockData = [
        {
          id: 1,
          key:'approvalWorkflow',
          name: 'Approval Workflow',
          description: 'Test workflow',
          version: 1,
          createdAt: '2024-01-01T10:00:00Z',
          updatedAt: '2024-01-01T10:00:00Z',
        },
      ];

      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: mockData, total: 1 });

      const result = await WorkflowApi.getWorkflows();

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions'),
        expect.any(Object)
      );

      expect(result.workflows).toHaveLength(1);
      expect(result.workflows[0].name).toBe('Approval Workflow');
    });

    it('should handle empty workflow list', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: [], total: 0 });

      const result = await WorkflowApi.getWorkflows();

      expect(result.workflows).toHaveLength(0);
      expect(result.total).toBe(0);
    });

    it('should pass pagination parameters correctly', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: [], total: 0 });

      await WorkflowApi.getWorkflows({ page: 2, pageSize: 10 });

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ page: 2, pageSize: 10 })
      );
    });
  });

  describe('getWorkflow', () => {
    it('should fetch single workflow successfully', async () => {
      const mockData = {
        id: 1,
        key:'approvalWorkflow',
        name: 'Approval Workflow',
        description: 'Test workflow',
        version: 1,
        createdAt: '2024-01-01T10:00:00Z',
        updatedAt: '2024-01-01T10:00:00Z',
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getWorkflow('approval_workflow');

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions/approval_workflow')
      );

      expect(result.name).toBe('Approval Workflow');
    });

    it('should handle workflow not found', async () => {
      (httpClient.get as jest.Mock).mockRejectedValueOnce(new Error('Workflow not found'));

      await expect(WorkflowApi.getWorkflow('nonexistent')).rejects.toBeDefined();
    });
  });

  describe('getProcessDefinition (backward compatible)', () => {
    it('should fetch process definition using key', async () => {
      const mockData = {
        id: 1,
        key:'testProcess',
        name: 'Test Process',
        version: 1,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getProcessDefinition('test_process');

      expect(result.name).toBe('Test Process');
    });
  });

  describe('getProcessVersions', () => {
    it('should fetch all versions of a process', async () => {
      const mockData = [
        bpmnVersion({ id: '2', version: 2 }),
        bpmnVersion({ id: '1', version: 1, isActive: false }),
      ];

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getProcessVersions('test_process');

      // 后端 ListVersions 只读 ctx.Query("processKey")，缺失时直接 400，
      // 因此必须断言查询参数而不只是 URL 前缀。
      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/versions', {
        processKey: 'test_process',
      });

      expect(result).toHaveLength(2);
      expect(result.map(item => item.id)).toEqual(['2', '1']);
      expect(result.map(item => item.status)).toEqual(['active', 'inactive']);
    });
  });

  describe('getWorkflowVersions', () => {
    it('每个版本保留唯一 id、按 isActive 映射状态并沿用真实时间', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([
        bpmnVersion({ id: '42', version: 2, isActive: true }),
        bpmnVersion({ id: '41', version: 1, isActive: false, bpmnXml: '<old />', changeLog: '' }),
      ]);

      const result = await WorkflowApi.getWorkflowVersions('wf1');

      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/versions', { processKey: 'wf1' });
      expect(result.map(item => item.id)).toEqual(['42', '41']);
      expect(result.map(item => item.code)).toEqual(['wf1', 'wf1']);
      expect(result.map(item => item.status)).toEqual(['active', 'inactive']);
      expect(result[0].createdAt).toEqual(new Date('2024-01-01T10:00:00Z'));
      expect(result[1].bpmnXml).toBe('<old />');
    });
  });

  describe('getVersion', () => {
    it('按 key + version 路径段取单个版本', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce(bpmnVersion());

      const result = await WorkflowApi.getVersion('wf 1', 2);

      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf%201/2');
      expect(result.bpmnXml).toBe('<definitions />');
      expect(result.isActive).toBe(true);
    });
  });

	describe('workflow persistence', () => {
	  it('returns the already-unwrapped definition after update', async () => {
		const updated = { id: '12', code: 'incident_flow', name: 'Incident Flow', version: 2 };
		(httpClient.put as jest.Mock).mockResolvedValueOnce(updated);

		await expect(
		  WorkflowApi.updateWorkflow('incident_flow', { name: 'Incident Flow' }, '2')
		).resolves.toBe(updated);
		expect(httpClient.put).toHaveBeenCalledWith(
		  '/api/v1/bpmn/process-definitions/incident_flow?version=2',
		  expect.objectContaining({ name: 'Incident Flow' })
		);
	  });

	  it('persists a new version through the backend version endpoint', async () => {
		(httpClient.post as jest.Mock).mockResolvedValueOnce({ version: 2 });
		const payload = {
		  processDefinitionKey: 'incident_flow',
		  name: 'Incident Flow',
		  bpmnXml: '<definitions />',
		  changeLog: '创建新版本',
		};

		await WorkflowApi.createWorkflowVersion(payload);
		expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/versions', payload);
	  });
	});

  describe('listWorkflowInstances', () => {
    it('should fetch workflow instances', async () => {
      const mockData = {
        items: [
          {
            id: '1',
            instanceId: '1',
            processDefinitionKey: 'approval_workflow',
            businessKey: 'ticket-123',
            status: 'running',
            startTime: '2024-01-01T10:00:00Z',
          },
        ],
        total: 1,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.listWorkflowInstances({});

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances'),
        expect.any(Object)
      );

      expect(result.instances).toHaveLength(1);
    });

    // 回归锁：实例列表必须带出 currentActivityId / currentActivityName。
    // 事故背景（2026-11-15 prod 实测）：后端 GET /bpmn/process-instances 一直返回
    // currentActivityId="Activity_Assign" / currentActivityName="任务分配"，
    // 但前端 map 里把这两个字段丢掉了，导致「这条单子卡在哪」只能点进详情、
    // 再从任务表反推。字段被丢弃属于静默回归——接口调用正常、页面不报错，
    // 只是用户永远看不到这一步。
    it('keeps currentActivityId / currentActivityName from the backend payload', async () => {
      const mockData = {
        items: [
          {
            id: '1',
            instanceId: 'PI-ticket_general_flow-1',
            processDefinitionKey: 'ticket_general_flow',
            businessKey: 'ticket:1',
            status: 'running',
            startTime: '2024-01-01T10:00:00Z',
            currentActivityId: 'Activity_Assign',
            currentActivityName: '任务分配',
          },
        ],
        total: 1,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getInstances({});

      expect(result.instances[0].currentActivityId).toBe('Activity_Assign');
      expect(result.instances[0].currentActivityName).toBe('任务分配');
    });

    it('leaves currentActivity fields undefined when the backend omits them', async () => {
      // 已结束的实例没有当前节点，不应变成空串或 'undefined' 字面量，
      // 否则列表列会把 '-' 渲染成 "undefined"。
      const mockData = {
        items: [
          {
            id: '1',
            instanceId: 'PI-1',
            processDefinitionKey: 'proc',
            businessKey: 'biz:1',
            status: 'completed',
            startTime: '2024-01-01T10:00:00Z',
            endTime: '2024-01-02T10:00:00Z',
          },
        ],
        total: 1,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getInstances({});

      expect(result.instances[0].currentActivityId).toBeUndefined();
      expect(result.instances[0].currentActivityName).toBeUndefined();
    });
  });

  describe('getInstance currentActivity', () => {
    // 同一条链路的详情分支。详情页此前也没有接这两个字段，
    // 它和列表一样把后端已返回的字段丢在 map 里。
    it('keeps currentActivity fields on the single-instance payload', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({
        id: '1',
        instanceId: 'PI-ticket_general_flow-1',
        processDefinitionKey: 'ticket_general_flow',
        businessKey: 'ticket:1',
        status: 'running',
        startTime: '2024-01-01T10:00:00Z',
        currentActivityId: 'Activity_Assign',
        currentActivityName: '任务分配',
      });

      const result = await WorkflowApi.getInstance('PI-ticket_general_flow-1');

      expect(result.currentActivityId).toBe('Activity_Assign');
      expect(result.currentActivityName).toBe('任务分配');
    });
  });

  describe('startWorkflow', () => {
    it('should start a new workflow', async () => {
      const mockData = {
        id: '1',
        processInstanceId: '1',
        processDefinitionKey: 'approval_workflow',
        businessKey: 'BIZ-123',
        status: 'running',
        startTime: '2024-01-01T10:00:00Z',
      };

      (httpClient.post as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.startWorkflow({
        workflowId: 'approval_workflow',
        variables: { ticketId: 123 },
      });

      expect(httpClient.post).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances'),
        expect.any(Object)
      );

      expect(result.status).toBe('running');
    });
  });

  describe('suspendWorkflow', () => {
    it('should suspend a running workflow', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);

      await WorkflowApi.suspendWorkflow('1');

      expect(httpClient.put).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/suspend'),
        expect.any(Object)
      );
    });
  });

  describe('resumeWorkflow', () => {
    it('should resume a suspended workflow', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);

      await WorkflowApi.resumeWorkflow('1');

      expect(httpClient.put).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/resume')
      );
    });
  });

  describe('terminateWorkflow', () => {
    it('should terminate a running workflow', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);

      await WorkflowApi.terminateWorkflow('1', 'Terminated by user');

      expect(httpClient.put).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/terminate'),
        expect.any(Object)
      );
    });
  });

  describe('deleteWorkflow', () => {
    it('should delete a workflow', async () => {
      (httpClient.delete as jest.Mock).mockResolvedValueOnce(undefined);

      await WorkflowApi.deleteWorkflow('approval_workflow');

      expect(httpClient.delete).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions/approval_workflow')
      );
    });
  });

  describe('createWorkflow', () => {
    it('should create a workflow with code/name mapped to key/category', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 1, key: 'new_wf', name: 'New' });
      const result = await WorkflowApi.createWorkflow({ code: 'new_wf', name: 'New', type: 'ticket' as any });
      expect(httpClient.post).toHaveBeenCalledWith(
        '/api/v1/bpmn/process-definitions',
        expect.objectContaining({ key: 'new_wf', name: 'New', category: 'ticket' })
      );
      // Tenant is injected by the backend from the JWT context; the frontend must
      // NOT forge it, so the payload must stay tenant-free (see createWorkflow in workflow-api.ts).
      expect(httpClient.post).toHaveBeenCalledWith(
        '/api/v1/bpmn/process-definitions',
        expect.not.objectContaining({ tenantId: expect.anything() })
      );
    });

    it('should omit tenantId from the payload (backend-issued tenant)', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 1 });
      await WorkflowApi.createWorkflow({ name: 'Test' } as any);
      const [, body] = (httpClient.post as jest.Mock).mock.calls[0];
      expect(body).not.toHaveProperty('tenantId');
    });
  });

  describe('cloneWorkflow', () => {
    it('should clone by getting original and creating new', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ id: 1, key: 'orig', name: 'Original', version: 1, createdAt: '2024-01-01', updatedAt: '2024-01-01' });
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 2, key: 'orig_copy' });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(1);
      await WorkflowApi.cloneWorkflow('orig', 'Clone');
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ name: 'Clone' }));
    });
  });

  describe('activateWorkflow', () => {
    it('should activate with default version', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.activateWorkflow('wf1');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/wf1/active?version=1.0.0', { active: true });
    });

    it('should activate with specified version', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.activateWorkflow('wf1', '2.0.0');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/wf1/active?version=2.0.0', { active: true });
    });
  });

  describe('deactivateWorkflow', () => {
    it('should deactivate workflow', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.deactivateWorkflow('wf1', '1.0.0');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/wf1/active?version=1.0.0', { active: false });
    });
  });

  describe('validateWorkflow', () => {
    it('should validate complete workflow without calling an unavailable endpoint', async () => {
      const result = await WorkflowApi.validateWorkflow({
        id: 'wf1',
        name: 'Test',
        nodes: [{ id: 'n1' }],
        connections: [{ id: 'c1' }],
      } as any);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
      expect(httpClient.post).not.toHaveBeenCalled();
    });

    it('should warn when workflow has nodes but no connections', async () => {
      const result = await WorkflowApi.validateWorkflow({ id: 'wf1', name: 'Test', nodes: [{ id: 'n1' }] } as any);
      expect(result.isValid).toBe(true);
      expect(result.warnings.length).toBeGreaterThan(0);
    });

    it('should report errors for invalid workflow', async () => {
      const result = await WorkflowApi.validateWorkflow({} as any);
      expect(result.isValid).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
    });
  });

  describe('exportWorkflow', () => {
    it('should export workflow', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ id: 1, key: 'wf1', name: 'WF', version: 1, bpmnXml: '<xml/>', createdAt: '2024-01-01', updatedAt: '2024-01-01' });
      const result = await WorkflowApi.exportWorkflow('wf1');
      expect(result.version).toBe('1.0');
      expect(result.workflow).toBeDefined();
    });
  });

  describe('importWorkflow', () => {
    it('should import workflow', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 1 });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(1);
      await WorkflowApi.importWorkflow({ version: '1.0', exportedAt: new Date(), exportedBy: 'sys', workflow: { code: 'imp', name: 'Imported', type: 'ticket' } as any });
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ name: 'Imported' }));
    });

    it('should throw on invalid data', async () => {
      await expect(WorkflowApi.importWorkflow({ version: '1.0', exportedAt: new Date(), exportedBy: 'sys' } as any)).rejects.toThrow('无效的导入数据');
    });
  });

  describe('publishVersion', () => {
    it('取当前已激活版本', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([
        bpmnVersion({ id: '43', version: 3, isActive: false }),
        bpmnVersion({ id: '42', version: 2, isActive: true }),
      ]);
      const result = await WorkflowApi.publishVersion('wf1');
      expect(result.version).toBe(2);
      expect(result.status).toBe('active');
    });

    it('没有任何激活版本时不得返回草稿', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([bpmnVersion({ isActive: false })]);
      await expect(WorkflowApi.publishVersion('wf1')).rejects.toThrow('未找到可发布的版本');
    });

    it('should throw if no versions', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([]);
      await expect(WorkflowApi.publishVersion('wf1')).rejects.toThrow('未找到可发布的版本');
    });
  });

  describe('activateVersion', () => {
    it('should activate a specific version', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.activateVersion('wf1', 2);
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf1/2/activate', {});
    });

    it('流程 key 含特殊字符时放入路径段前转义', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.activateVersion('wf 1/2', 3);
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf%201%2F2/3/activate', {});
    });
  });

  describe('rollbackVersion', () => {
    it('should rollback to version', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.rollbackVersion('wf1', 1, 'bug fix');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf1/1/rollback', { reason: 'bug fix' });
    });

    it('should use default reason', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.rollbackVersion('wf1', 1);
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf1/1/rollback', { reason: '回滚操作' });
    });
  });

  describe('deleteVersion', () => {
    it('should delete a version', async () => {
      (httpClient.delete as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.deleteVersion('wf1', 2);
      expect(httpClient.delete).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/wf1?version=2');
    });
  });

  describe('compareVersions', () => {
    it('透传后端 VersionComparison 结构', async () => {
      const comparison = {
        baseVersion: bpmnVersion({ version: 1, isActive: false }),
        targetVersion: bpmnVersion({ version: 2, isActive: true }),
        changes: [
          {
            type: 'element',
            changeType: 'added',
            elementType: 'bpmn:UserTask',
            elementId: 'Task_1',
            elementName: '审批',
            description: 'Task count changed',
            impact: 'high',
          },
        ],
        breakingChanges: ['Task count changed'],
        compatibility: 'risky',
      };
      (httpClient.get as jest.Mock).mockResolvedValueOnce(comparison);

      const result = await WorkflowApi.compareVersions('wf1', 1, 2);

      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf1/compare', {
        baseVersion: '1',
        targetVersion: '2',
      });
      expect(result?.compatibility).toBe('risky');
      expect(result?.breakingChanges).toEqual(['Task count changed']);
      expect(result?.changes).toHaveLength(1);
      expect(result?.baseVersion?.version).toBe(1);
      expect(result?.targetVersion?.bpmnXml).toBe('<definitions />');
    });

    it('相同版本号直接返回 null，不发请求', async () => {
      const result = await WorkflowApi.compareVersions('wf1', 2, 2);
      expect(result).toBeNull();
      expect(httpClient.get).not.toHaveBeenCalled();
    });

    it('后端返回空 data 时按“无需比较”处理，不能伪装成零差异', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce(null);
      await expect(WorkflowApi.compareVersions('wf1', 1, 2)).resolves.toBeNull();
    });

    it('流程 key 含特殊字符时转义路径段', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce(null);
      await WorkflowApi.compareVersions('wf 1', 1, 2);
      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf%201/compare', {
        baseVersion: '1',
        targetVersion: '2',
      });
    });
  });

  describe('getInstance', () => {
    it('should get single instance', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ id: 'inst1', instanceId: 'inst1', processDefinitionKey: 'wf1', status: 'running', startTime: '2024-01-01' });
      const result = await WorkflowApi.getInstance('inst1');
      expect(result.id).toBe('inst1');
      expect(result.status).toBe('running');
    });
  });

  describe('retryInstance', () => {
    it('should retry by resuming', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.retryInstance('inst1');
      expect(httpClient.put).toHaveBeenCalledWith(expect.stringContaining('/resume'));
    });
  });

  describe('completeNode', () => {
    it('should complete a node', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.completeNode({ instanceId: 'inst1', nodeId: 'task1', output: { approved: true } });
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/complete', { variables: { approved: true } });
    });
  });

  describe('skipNode', () => {
    it('should skip a node by completing with _skipped flag', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.skipNode('inst1', 'task1');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/complete', { variables: { _skipped: true } });
    });
  });

  describe('retryNode', () => {
    it('should retry a node', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.retryNode('inst1', 'task1');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/complete', { variables: { _retried: true } });
    });
  });

  describe('getTemplates', () => {
    it('should return empty array (mock)', async () => {
      const result = await WorkflowApi.getTemplates();
      expect(result).toEqual([]);
    });
  });

  describe('getTemplate', () => {
    it('should return mock template', async () => {
      const result = await WorkflowApi.getTemplate('t1');
      expect(result.id).toBe('t1');
      expect(result.name).toBe('未命名模板');
    });
  });

  describe('createFromTemplate', () => {
    it('should create from template', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 1 });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(1);
      await WorkflowApi.createFromTemplate('t1', 'New WF');
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ name: 'New WF' }));
    });
  });

  describe('saveAsTemplate', () => {
    it('should return mock template data', async () => {
      const result = await WorkflowApi.saveAsTemplate('wf1', { name: 'My Template', category: 'approval', tags: ['tag1'] });
      expect(result.name).toBe('My Template');
      expect(result.category).toBe('approval');
      expect(result.tags).toEqual(['tag1']);
    });
  });

  describe('getWorkflowStats', () => {
    it('returns a zeroed snapshot without hitting the unregistered legacy endpoint', async () => {
      // Legacy /api/v1/workflows/:id/stats is not registered; the method is neutralized so
      // callers fall back to BPMN stats sources instead of triggering a 404.
      const result = await WorkflowApi.getWorkflowStats('wf1');
      expect(result.workflowId).toBe('wf1');
      expect(result.totalInstances).toBe(0);
      expect(result.bottlenecks).toEqual([]);
      expect(httpClient.get).not.toHaveBeenCalledWith(
        '/api/v1/workflows/wf1/stats',
        expect.anything()
      );
    });
  });

  describe('getInstanceStats', () => {
    it('should get instance stats from backend', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ total: 10, running: 3, completed: 5, suspended: 1, terminated: 1 });
      const result = await WorkflowApi.getInstanceStats({ processDefinitionKey: 'wf1' });
      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/stats/instances', expect.objectContaining({ processDefinitionKey: 'wf1' }));
      expect(result.total).toBe(10);
    });

    it('should return defaults when null', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce(null);
      const result = await WorkflowApi.getInstanceStats();
      expect(result.total).toBe(0);
    });
  });

  describe('getTaskStats', () => {
    it('should get task stats from backend', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ totalTasks: 20, completedTasks: 15, pendingTasks: 5, overdueTasks: 2, averageCompletion: 3.5 });
      const result = await WorkflowApi.getTaskStats({ assignee: 'user1' });
      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/stats/tasks', expect.objectContaining({ assignee: 'user1' }));
      expect(result.totalTasks).toBe(20);
    });

    it('should return defaults when null', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce(null);
      const result = await WorkflowApi.getTaskStats();
      expect(result.totalTasks).toBe(0);
    });
  });

  describe('createCounterSignTasks', () => {
    it('should create counter-sign tasks', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ data: [{ taskId: 't1', assignee: 'user1', status: 'pending' }] });
      const result = await WorkflowApi.createCounterSignTasks('task1', ['user1', 'user2'], 'parallel', 2);
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/counter-sign', { approvalType: 'parallel', approvers: ['user1', 'user2'], threshold: 2 });
      expect(result).toHaveLength(1);
    });

    it('should handle empty response', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({});
      const result = await WorkflowApi.createCounterSignTasks('task1', ['user1']);
      expect(result).toEqual([]);
    });
  });

  describe('getCounterSignStatus', () => {
    it('should get counter-sign status', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ data: { parentTaskId: 't1', total: 3, completed: 1, approved: 1, rejected: 0, pending: 2, status: 'pending' } });
      const result = await WorkflowApi.getCounterSignStatus('t1');
      expect(result.total).toBe(3);
    });

    it('should return defaults when no data', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({});
      const result = await WorkflowApi.getCounterSignStatus('t1');
      expect(result.total).toBe(0);
      expect(result.status).toBe('pending');
    });
  });

  describe('vote', () => {
    it('should vote on a task', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.vote('task1', true, 'Approved');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/vote', { approved: true, comment: 'Approved' });
    });
  });

  describe('getNodeStats', () => {
    it('should return empty array (mock)', async () => {
      const result = await WorkflowApi.getNodeStats('wf1');
      expect(result).toEqual([]);
    });
  });

  describe('getBottleneckAnalysis', () => {
    it('should return empty bottlenecks (mock)', async () => {
      const result = await WorkflowApi.getBottleneckAnalysis('wf1');
      expect(result.bottlenecks).toEqual([]);
    });
  });

  describe('listMyTasks', () => {
    it('should list my tasks with paginated response', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: [{ id: '1', taskName: 'Review', status: 'pending', assignee: '10' }], total: 1 });
      const result = await WorkflowApi.listMyTasks({ page: 1, pageSize: 20 });
      expect(result.items).toHaveLength(1);
      expect(result.items[0].nodeName).toBe('Review');
    });

    it('should handle paginated response', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: [{ id: '1', taskName: 'T1' }], total: 50, page: 2, size: 10 });
      const result = await WorkflowApi.listMyTasks({ page: 2, pageSize: 10 });
      expect(result.total).toBe(50);
      expect(result.page).toBe(2);
    });
  });

  describe('claimMyTask', () => {
    it('should claim a task', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.claimMyTask('task1');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/claim');
    });
  });

  describe('listWorkflowTasks', () => {
    it('should list tasks from the canonical items field', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: [{ id: '1', taskName: 'Step 1', status: 'completed' }] });
      const result = await WorkflowApi.listWorkflowTasks('inst1');
      expect(result).toHaveLength(1);
      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/tasks', {
        processInstanceKey: 'inst1',
        page: 1,
        pageSize: 100,
      });
    });

    it('does not read deprecated list fields', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ list: [{ id: '1', taskName: 'Legacy task' }] });
      await expect(WorkflowApi.listWorkflowTasks('inst1')).rejects.toThrow();
    });

    it('should throw error on failure', async () => {
      (httpClient.get as jest.Mock).mockRejectedValueOnce(new Error('fail'));
      await expect(WorkflowApi.listWorkflowTasks('inst1')).rejects.toThrow('fail');
    });
  });

  describe('deployWorkflow', () => {
    it('should deploy by activating', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.deployWorkflow('wf1');
      expect(httpClient.put).toHaveBeenCalledWith(expect.stringContaining('/active'), expect.any(Object));
    });

    it('should propagate error on deploy failure', async () => {
      (httpClient.put as jest.Mock).mockRejectedValueOnce(new Error('Deploy failed'));
      await expect(WorkflowApi.deployWorkflow('wf1')).rejects.toThrow('Deploy failed');
    });
  });

  describe('getNodeInstances', () => {
    it('should get node instances from tasks', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({
        items: [{ id: 't1', taskId: 'task1', name: 'Review', status: 'pending', assignee: '5' }],
      });
      const result = await WorkflowApi.getNodeInstances('inst1');
      expect(result).toHaveLength(1);
      expect(result[0].nodeName).toBe('Review');
      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/tasks', {
        processInstanceKey: 'inst1',
        page: 1,
        pageSize: 100,
      });
    });

    /**
     * nodeId 必须是 BPMN 元素 ID，不能是 taskId。
     *
     * prod 实测（ticket_general_flow 实例的真实任务行）：
     *   {'taskId': 'TASK-Activity_Assign-1794579980116057665',
     *    'taskDefinitionKey': 'Activity_Assign', 'taskName': '任务分配'}
     * taskId 是任务流水号（TASK-<元素ID>-<雪花>），把它当 nodeId 去 BPMN 图上
     * 打标记会一个都打不中 —— 图上元素的 id 是 'Activity_Assign'。
     */
    it('nodeId 优先取 taskDefinitionKey（BPMN 元素 ID），而非 taskId 流水号', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({
        items: [
          {
            id: 't1',
            taskId: 'TASK-Activity_Assign-1794579980116057665',
            taskDefinitionKey: 'Activity_Assign',
            taskName: '任务分配',
            name: '任务分配',
            status: 'completed',
            assignee: '5',
          },
        ],
      });
      const result = await WorkflowApi.getNodeInstances('inst1');
      expect(result[0].nodeId).toBe('Activity_Assign');
      expect(result[0].nodeId).not.toBe('TASK-Activity_Assign-1794579980116057665');
      expect(result[0].nodeName).toBe('任务分配');
    });

    it('缺 taskDefinitionKey 的历史数据退回 taskId，至少详情表还能显示一行', async () => {
      // 老数据/系统任务可能没有 taskDefinitionKey；此时不能让 nodeId 变空，
      // 否则详情表格整行丢失。退回 taskId 后仍无法在 BPMN 图上定位，
      // 但预览组件打标记前有 elementRegistry 存在性判断会自然跳过。
      (httpClient.get as jest.Mock).mockResolvedValueOnce({
        items: [{ id: 't2', taskId: 'task-legacy', status: 'completed' }],
      });
      const result = await WorkflowApi.getNodeInstances('inst1');
      expect(result[0].nodeId).toBe('task-legacy');
    });

    it('should return empty on error', async () => {
      (httpClient.get as jest.Mock).mockRejectedValueOnce(new Error('fail'));
      const result = await WorkflowApi.getNodeInstances('inst1');
      expect(result).toEqual([]);
    });
  });

  describe('exportReport', () => {
    it('should generate CSV blob', async () => {
      (httpClient.get as jest.Mock)
        .mockResolvedValueOnce({ id: 1, key: 'wf1', name: 'Report WF', version: 1, createdAt: '2024-01-01', updatedAt: '2024-01-01' })
        .mockResolvedValueOnce({ total: 5, running: 2, completed: 2, suspended: 0, terminated: 1 })
        .mockResolvedValueOnce({ totalTasks: 10, completedTasks: 8, pendingTasks: 2, overdueTasks: 0, averageCompletion: 2.5 });
      const result = await WorkflowApi.exportReport({ workflowId: 'wf1', format: 'excel', startDate: '2024-01-01', endDate: '2024-12-31' });
      expect(result).toBeInstanceOf(Blob);
    });
  });

  describe('getWorkflows with paginated response', () => {
    it('should read total from the canonical flat envelope', async () => {
      // 规范契约：common.SuccessWithPagination 只产出平铺
      // {items,total,page,pageSize,totalPages}；嵌套 pagination 与领域名别名已删除。
      (httpClient.get as jest.Mock).mockResolvedValueOnce({
        items: [{ id: 1, key: 'wf', name: 'WF', version: 1, createdAt: '2024-01-01', updatedAt: '2024-01-01' }],
        total: 5,
        page: 1,
        pageSize: 1,
        totalPages: 5,
      });
      const result = await WorkflowApi.getWorkflows();
      expect(result.workflows).toHaveLength(1);
      expect(result.total).toBe(5);
    });

    it('rejects legacy {data: [...]} shape (single list envelope)', async () => {
      // 只接受规范 {items,total,...}：response.data ?? response.items ?? response.list
      // 这种多字段兜底会让两套集合同源共存，data 不是集合来源。
      (httpClient.get as jest.Mock).mockResolvedValueOnce({
        data: [{ id: 1, key: 'wf', name: 'WF', version: 1, createdAt: '2024-01-01', updatedAt: '2024-01-01' }],
      });
      const result = await WorkflowApi.getWorkflows();
      expect(result.workflows).toHaveLength(0);
      expect(result.total).toBe(0);
    });
  });
});
