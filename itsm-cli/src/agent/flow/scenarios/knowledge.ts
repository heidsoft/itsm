import { apiClient } from '../../../lib/api-client.js';
import type { FlowScenario } from '../types.js';
import { idOf, requireNumber } from '../vars.js';
import { listContains } from '../assertions.js';

/**
 * Knowledge base: draft → publish → retrievable by search → unpublish.
 *
 * Search is the step that matters. A drafted article leaking into retrieval is a
 * permission/visibility defect, and a published article failing to appear is a
 * knowledge-reuse outage. Both are invisible if you only test CRUD.
 */
export const knowledgeLifecycle: FlowScenario = {
  name: 'knowledge-lifecycle',
  summary: 'Knowledge: create draft → publish → searchable → unpublish, with version history',
  domain: 'knowledge',
  tags: ['knowledge', 'rag', 'write'],
  proves: [
    'articles are created as drafts',
    'publish makes the article retrievable through search',
    'version history is recorded on mutation',
    'unpublish removes it from the published state',
  ],
  steps: [
    {
      name: 'create draft article',
      endpoint: 'POST /api/v1/knowledge/articles',
      run: (ctx) =>
        apiClient.createKnowledgeArticle({
          title: `[${ctx.tag}] Connection pool exhaustion runbook`,
          content: 'Symptom: latency spikes. Action: check pool saturation, then restart the worker.',
          category: 'incident-response',
          tags: ['flow', ctx.tag],
        }),
      expect: [
        { path: 'status', eq: 'draft' },
        { path: 'title', contains: 'Connection pool' },
      ],
      capture: (result) => ({ articleId: idOf(result, 'knowledge create') }),
    },
    {
      name: 'submit article for review',
      endpoint: 'POST /api/v1/knowledge/articles/:id/review/submit',
      run: (ctx) => apiClient.knowledgeReviewAction(requireNumber(ctx, 'articleId'), 'submit', { comment: 'Flow: ready for review' }),
    },
    {
      name: 'approve review (publish gate)',
      endpoint: 'POST /api/v1/knowledge/articles/:id/review/approve',
      run: (ctx) => apiClient.knowledgeReviewAction(requireNumber(ctx, 'articleId'), 'approve', { comment: 'Flow: approved' }),
      // Four-eyes governance: the author cannot approve their own article. With a
      // single CLI identity this flow is genuinely unverifiable, so report
      // `blocked` instead of pretending the publish path passed.
      blockedWhen: {
        messageIncludes: '自己撰写',
        reason: 'the review gate forbids self-approval; run this flow with a second reviewer identity',
      },
    },
    {
      name: 'publish article',
      endpoint: 'POST /api/v1/knowledge/articles/:id/publish',
      run: (ctx) => apiClient.knowledgeLifecycle(requireNumber(ctx, 'articleId'), 'publish'),
    },
    {
      name: 'read published article',
      endpoint: 'GET /api/v1/knowledge/articles/:id',
      run: (ctx) => apiClient.getKnowledgeArticle(requireNumber(ctx, 'articleId')),
      expect: [{ path: 'status', eq: 'published' }],
    },
    {
      name: 'search finds the published article',
      endpoint: 'POST /api/v1/knowledge/search',
      run: (ctx) => apiClient.searchKnowledge(ctx.tag),
      verify: (result, ctx) => {
        const articleId = ctx.vars['articleId'];
        return [
          {
            check: 'published article is retrievable by its run tag',
            ok: listContains(result, 'id', articleId),
            expected: `article ${String(articleId)} present in search results`,
            actual: (result as { items?: unknown[] }).items?.length ?? 0,
          },
        ];
      },
    },
    {
      name: 'version history exists',
      endpoint: 'GET /api/v1/knowledge/articles/:id/versions',
      run: (ctx) => apiClient.listKnowledgeVersions(requireNumber(ctx, 'articleId')),
      verify: (result) => {
        const items = Array.isArray(result) ? result : ((result as { items?: unknown[] }).items ?? []);
        return [{ check: 'at least one version is recorded', ok: items.length >= 1, expected: '>=1', actual: items.length }];
      },
    },
    {
      name: 'unpublish article',
      endpoint: 'POST /api/v1/knowledge/articles/:id/unpublish',
      run: (ctx) => apiClient.knowledgeLifecycle(requireNumber(ctx, 'articleId'), 'unpublish'),
    },
    {
      name: 'read unpublished article',
      endpoint: 'GET /api/v1/knowledge/articles/:id',
      run: (ctx) => apiClient.getKnowledgeArticle(requireNumber(ctx, 'articleId')),
      expect: [{ path: 'status', eq: 'draft' }],
    },
  ],
  teardown: [
    {
      name: 'delete flow article',
      endpoint: 'DELETE /api/v1/knowledge/articles/:id',
      run: async (ctx) => {
        const id = ctx.vars['articleId'];
        if (typeof id === 'number') await apiClient.deleteKnowledgeArticle(id);
      },
    },
  ],
};
