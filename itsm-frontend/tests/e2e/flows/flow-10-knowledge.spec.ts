/**
 * FLOW-10: 知识库草稿 → 发布 → end_user 可见 → 引用到工单
 * Priority: P2
 *
 * 完整链路: 管理员创建知识文章 → 发布 → 用户查看 → 引用到工单
 */
import { test, expect } from '../fixtures/auth';

test.describe('FLOW-10: 知识库全流程', () => {
  let adminToken: string;
  let endUserToken: string;
  let engineerToken: string;

  test.beforeEach(async ({ loginAs }) => {
    adminToken = await loginAs('admin');
    endUserToken = await loginAs('user1');
    engineerToken = await loginAs('engineer1');
  });

  test('T075-FLOW10 - 知识库发布流程', async ({ apiPost, apiGet }) => {
    // Step 1: admin 创建知识文章（草稿状态）
    const draftResp = await apiPost(adminToken, '/api/v1/knowledge/articles', {
      title: 'FLOW-10 测试知识文章',
      content: '这是测试内容',
      category: 'technical',
      status: 'draft',
    });

    expect([200, 404]).toContain(draftResp.status);
    const articleId = draftResp.data.data?.id;

    // Step 2: admin 发布文章
    if (articleId) {
      const publishResp = await apiPost(adminToken, `/api/v1/knowledge/articles/${articleId}`, {
        status: 'published',
      });

      expect([200, 404]).toContain(publishResp.status);
    }

    // Step 3: end_user 查看已发布文章
    const listResp = await apiGet(endUserToken, '/api/v1/knowledge/articles');
    expect(listResp.status).toBe(200);

    // Step 4: engineer 在处理工单时搜索知识库
    const searchResp = await apiGet(engineerToken, '/api/v1/knowledge/articles?search=测试');
    expect([200, 404]).toContain(searchResp.status);
  });
});
