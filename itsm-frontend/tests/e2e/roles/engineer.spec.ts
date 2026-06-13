/**
 * US2: engineer 处理人全生命周期验证
 * Priority: P1
 *
 * 用户故事: 作为工程师/处理人，我能够接收指派工单、更新工单状态、填写处理记录、关闭工单
 */
import { test, expect } from '../fixtures/auth';

test.describe('US2: engineer 处理人全生命周期', () => {
  let token: string;

  test.beforeEach(async ({ loginAs }) => {
    // 登录为工程师
    token = await loginAs('engineer1');
  });

  test('T021 - 工程师登录后能访问工单列表', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/tickets');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);
  });

  test('T022 - 能查看指派给我的工单', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/tickets?assignee=engineer1');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);
  });

  test('T023 - 能更新工单状态 open -> in_progress', async ({ apiPost }) => {
    // 创建工单
    const createResp = await apiPost(token, '/api/v1/tickets', {
      title: '工程师测试工单 - 状态更新',
      description: 'This is a test ticket for status update testing',
      priority: 'high',
    });

    const ticketId = createResp.data.data?.id;
    expect(ticketId).toBeDefined();

    // 更新为处理中
    const updateResp = await apiPost(token, `/api/v1/tickets/${ticketId}/status`, {
      status: 'in_progress',
    });

    expect(updateResp.status).toBe(200);
    expect(updateResp.data).toHaveProperty('code', 0);
  });

  test('T024 - 能填写处理记录', async ({ apiPost }) => {
    const createResp = await apiPost(token, '/api/v1/tickets', {
      title: '工程师测试工单 - 处理记录',
      description: 'This is a test ticket for handling notes',
      priority: 'medium',
    });

    const ticketId = createResp.data.data?.id;
    expect(ticketId).toBeDefined();

    // 添加处理记录（使用 update status + note）
    const noteResp = await apiPost(token, `/api/v1/tickets/${ticketId}/status`, {
      status: 'in_progress',
      handling_note: '已定位问题原因，正在修复中',
    });

    expect(noteResp.status).toBe(200);
  });

  test('T025 - 能将工单标记为已解决', async ({ apiPost }) => {
    const createResp = await apiPost(token, '/api/v1/tickets', {
      title: '工程师测试工单 - 已解决',
      description: 'This is a test ticket for resolution',
      priority: 'low',
    });

    const ticketId = createResp.data.data?.id;
    expect(ticketId).toBeDefined();

    // 标记为已解决
    const resolveResp = await apiPost(token, `/api/v1/tickets/${ticketId}/resolve`, {
      resolution: '问题已修复',
    });

    expect(resolveResp.status).toBe(200);
    expect(resolveResp.data).toHaveProperty('code', 0);
  });

  test('T026 - 能查看工单历史记录', async ({ apiGet, apiPost }) => {
    // 创建工单
    const createResp = await apiPost(token, '/api/v1/tickets', {
      title: '工程师测试工单 - 历史',
      description: 'This is a test ticket for history viewing',
      priority: 'low',
    });

    const ticketId = createResp.data.data?.id;
    expect(ticketId).toBeDefined();

    // 查看历史
    const historyResp = await apiGet(token, `/api/v1/tickets/${ticketId}/workflow-history`);

    expect(historyResp.status).toBe(200);
  });

  test('T027 - 工程师可创建工单', async ({ apiPost }) => {
    // 验证工程师有工单创建权限
    const createResp = await apiPost(token, '/api/v1/tickets', {
      title: '工程师测试工单 - 创建验证',
      description: 'This is a test ticket to verify engineer create permission',
      priority: 'low',
    });
    expect(createResp.status).toBe(200);
    expect(createResp.data).toHaveProperty('code', 0);
  });

  test('T028 - 能访问知识库进行问题排查', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/knowledge/articles');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);
  });
});
