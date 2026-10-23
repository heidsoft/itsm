import { cmd } from '../spec.js';
import { requirePositional, withPagination, body, stringFlag, numberFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';

/** Knowledge base with version state + permission-filtered retrieval. */

export function knowledgeCommands() {
  return [
    cmd('knowledge.list', 'List articles', 'itsm knowledge list [--page N] [--pageSize N]', ['GET /api/v1/knowledge/articles'], async (ctx) => apiClient.listKnowledge(withPagination(ctx.args) as never)),
    cmd('knowledge.get', 'Article detail (with source attribution)', 'itsm knowledge get <id>', ['GET /api/v1/knowledge/articles/:id'], async (ctx) => apiClient.getKnowledgeArticle(requirePositional(ctx.args, 0, 'articleId'))),
    cmd('knowledge.create', 'Create article (draft)', 'itsm knowledge create --data <json>', ['POST /api/v1/knowledge/articles'], async (ctx) => apiClient.createKnowledgeArticle(body(ctx.args) ?? {})),
    cmd('knowledge.update', 'Update article', 'itsm knowledge update <id> [--data <json>]', ['PUT /api/v1/knowledge/articles/:id'], async (ctx) => apiClient.updateKnowledgeArticle(requirePositional(ctx.args, 0, 'articleId'), body(ctx.args) ?? {})),
    cmd('knowledge.publish', 'Publish draft article (requires an approved review)', 'itsm knowledge publish <id>', ['POST /api/v1/knowledge/articles/:id/publish'], async (ctx) => apiClient.knowledgeLifecycle(requirePositional(ctx.args, 0, 'articleId'), 'publish')),
    cmd('knowledge.review-submit', 'Submit an article for editorial review', 'itsm knowledge review-submit <id> [--comment text]', ['POST /api/v1/knowledge/articles/:id/review/submit'], async (ctx) =>
      apiClient.knowledgeReviewAction(requirePositional(ctx.args, 0, 'articleId'), 'submit', body(ctx.args) ?? {})
    ),
    cmd('knowledge.review-approve', 'Approve a pending review (publish gate)', 'itsm knowledge review-approve <id> [--comment text]', ['POST /api/v1/knowledge/articles/:id/review/approve'], async (ctx) =>
      apiClient.knowledgeReviewAction(requirePositional(ctx.args, 0, 'articleId'), 'approve', body(ctx.args) ?? {})
    ),
    cmd('knowledge.review-reject', 'Reject a pending review (comment required)', 'itsm knowledge review-reject <id> --data \'{"comment":"..."}\'', ['POST /api/v1/knowledge/articles/:id/review/reject'], async (ctx) => {
      const data = body(ctx.args);
      if (!data || typeof data['comment'] !== 'string' || !data['comment']) throw usageError('Missing required comment: --data \'{"comment":"..."}\'');
      return apiClient.knowledgeReviewAction(requirePositional(ctx.args, 0, 'articleId'), 'reject', data);
    }),
    cmd('knowledge.review-withdraw', 'Withdraw a submitted review', 'itsm knowledge review-withdraw <id> [--comment text]', ['POST /api/v1/knowledge/articles/:id/review/withdraw'], async (ctx) =>
      apiClient.knowledgeReviewAction(requirePositional(ctx.args, 0, 'articleId'), 'withdraw', body(ctx.args) ?? {})
    ),
    cmd('knowledge.review-republish', 'Republish after review changes', 'itsm knowledge review-republish <id> [--comment text]', ['POST /api/v1/knowledge/articles/:id/review/republish'], async (ctx) =>
      apiClient.knowledgeReviewAction(requirePositional(ctx.args, 0, 'articleId'), 'republish', body(ctx.args) ?? {})
    ),
    cmd('knowledge.reviews', 'Review records for an article', 'itsm knowledge reviews <id>', ['GET /api/v1/knowledge/articles/:id/reviews'], async (ctx) => apiClient.listKnowledgeReviews(requirePositional(ctx.args, 0, 'articleId'))),
    cmd('knowledge.unpublish', 'Unpublish article', 'itsm knowledge unpublish <id>', ['POST /api/v1/knowledge/articles/:id/unpublish'], async (ctx) => apiClient.knowledgeLifecycle(requirePositional(ctx.args, 0, 'articleId'), 'unpublish')),
    cmd('knowledge.search', 'Search articles (RAG retrieval respects permissions/time window)', 'itsm knowledge search <q> [--category c] [--pageSize N]', ['POST /api/v1/knowledge/search'], async (ctx) => apiClient.searchKnowledge(requirePositional(ctx.args, 0, 'query'), {
      category: stringFlag(ctx.args.flags, 'category'),
      pageSize: numberFlag(ctx.args.flags, 'pageSize'),
    })),
    cmd('knowledge.versions', 'Article version history', 'itsm knowledge versions <id>', ['GET /api/v1/knowledge/articles/:id/versions'], async (ctx) => apiClient.listKnowledgeVersions(requirePositional(ctx.args, 0, 'articleId'))),
    cmd('knowledge.categories', 'Category tree', 'itsm knowledge categories', ['GET /api/v1/knowledge/categories'], async () => apiClient.listKnowledgeCategories()),
    cmd('knowledge.stats', 'Knowledge statistics', 'itsm knowledge stats', ['GET /api/v1/knowledge/stats'], async () => apiClient.getKnowledgeStats()),
  ];
}
