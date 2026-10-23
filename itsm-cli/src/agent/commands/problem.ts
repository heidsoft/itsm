import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString, body, stringFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';

/** Problem management + known error DB — bodies mirror dto/problem_dto.go. */

export function problemCommands() {
  return [
    cmd('problem.list', 'List problems', 'itsm problem list [--status s] [--page N] [--pageSize N]', ['GET /api/v1/problems'], async (ctx) =>
      apiClient.listProblems({
        ...withPagination(ctx.args),
        status: optionalString(ctx.args.flags, 'status'),
      })
    ),
    cmd('problem.get', 'Get problem detail', 'itsm problem get <id>', ['GET /api/v1/problems/:id'], async (ctx) => apiClient.getProblem(requirePositional(ctx.args, 0, 'problemId'))),
    cmd('problem.stats', 'Problem aggregate counters (total/open/inProgress/resolved/closed)', 'itsm problem stats', ['GET /api/v1/problems/stats'], async () => apiClient.getProblemStats()),
    cmd('problem.trend', 'Problem trend over a window', 'itsm problem trend [--startDate d] [--endDate d]', ['GET /api/v1/problems/trend'], async (ctx) =>
      apiClient.getProblemTrends({
        startDate: optionalString(ctx.args.flags, 'startDate'),
        endDate: optionalString(ctx.args.flags, 'endDate'),
      })
    ),
    cmd('problem.hotspots', 'Problem hotspots over a window', 'itsm problem hotspots [--startDate d] [--endDate d]', ['GET /api/v1/problems/hotspots'], async (ctx) =>
      apiClient.getProblemHotspots({
        startDate: optionalString(ctx.args.flags, 'startDate'),
        endDate: optionalString(ctx.args.flags, 'endDate'),
      })
    ),
    cmd('problem.create', 'Create a problem (title/description/priority required)', 'itsm problem create <title> --description <d> --priority <p> | --data <json>', ['POST /api/v1/problems'], async (ctx) => {
      const data = body(ctx.args);
      if (data) return apiClient.createProblem(data);
      // Flat-argument form: agents should not have to hand-escape JSON for the
      // three fields the backend marks `binding:"required"` (title/description/priority).
      const title = requirePositional(ctx.args, 0, 'title');
      const description = optionalString(ctx.args.flags, 'description');
      const priority = optionalString(ctx.args.flags, 'priority');
      if (!description || !priority) {
        throw usageError('Missing --description (min 10 chars) and/or --priority (low|medium|high|critical|urgent)');
      }
      const flat: Record<string, unknown> = { title, description, priority };
      const category = optionalString(ctx.args.flags, 'category');
      if (category) flat['category'] = category;
      return apiClient.createProblem(flat);
    }),
    cmd('problem.update', 'Pointer-field update', 'itsm problem update <id> --data <json>', ['PUT /api/v1/problems/:id'], async (ctx) => apiClient.updateProblem(requirePositional(ctx.args, 0, 'problemId'), body(ctx.args) ?? {})),
    cmd('problem.delete', 'Delete a problem', 'itsm problem delete <id>', ['DELETE /api/v1/problems/:id'], async (ctx) => apiClient.deleteProblem(requirePositional(ctx.args, 0, 'problemId'))),
    cmd('problem.investigate', 'Start investigation (state transition only, no body)', 'itsm problem investigate <id>', ['POST /api/v1/problems/:id/investigate'], async (ctx) => apiClient.problemLifecycle(requirePositional(ctx.args, 0, 'problemId'), 'investigate')),
    cmd('problem.resolve', 'Move a problem to resolved (required before close; no dedicated /resolve route)', 'itsm problem resolve <id>', ['PUT /api/v1/problems/:id'], async (ctx) => apiClient.updateProblem(requirePositional(ctx.args, 0, 'problemId'), { status: 'resolved' })),
    cmd('problem.root-cause', 'Update root cause (required)', 'itsm problem root-cause <id> --rootCause <text>', ['PUT /api/v1/problems/:id/root-cause'], async (ctx) => {
      const rootCause = stringFlag(ctx.args.flags, 'rootCause') ?? body(ctx.args)?.rootCause;
      if (typeof rootCause !== 'string' || !rootCause) throw usageError('Missing required --rootCause');
      return apiClient.updateProblemRootCause(requirePositional(ctx.args, 0, 'problemId'), { rootCause });
    }),
    cmd('problem.solution', 'Update workaround/solution', 'itsm problem solution <id> --data <json>', ['PUT /api/v1/problems/:id/solution'], async (ctx) => {
      const data = body(ctx.args);
      if (!data) throw usageError('Provide --data \'{"solution":"...","workaround":"..."}\'');
      return apiClient.updateProblemSolution(requirePositional(ctx.args, 0, 'problemId'), data);
    }),
    cmd('problem.close', 'Close with resolution', 'itsm problem close <id> [--resolution text]', ['POST /api/v1/problems/:id/close'], async (ctx) => {
      const resolution = optionalString(ctx.args.flags, 'resolution');
      return apiClient.problemLifecycle(requirePositional(ctx.args, 0, 'problemId'), 'close', resolution ? { resolution } : undefined);
    }),
    cmd('problem.comments.list', 'Problem comments', 'itsm problem comments list <id>', ['GET /api/v1/problems/:id/comments'], async (ctx) => apiClient.listProblemComments(requirePositional(ctx.args, 0, 'problemId'))),
    cmd('problem.comments.add', 'Add problem comment', 'itsm problem comments add <id> --data <json>', ['POST /api/v1/problems/:id/comments'], async (ctx) => apiClient.addProblemComment(requirePositional(ctx.args, 0, 'problemId'), body(ctx.args) ?? {})),
    cmd('problem.known-errors.list', 'Known error database', 'itsm problem known-errors list [--q q] [--page N] [--pageSize N]', ['GET /api/v1/known-errors'], async (ctx) =>
      apiClient.listKnownErrors({
        ...withPagination(ctx.args),
        q: optionalString(ctx.args.flags, 'q'),
      })
    ),
    cmd('problem.known-errors.get', 'Known error detail', 'itsm problem known-errors get <id>', ['GET /api/v1/known-errors/:id'], async (ctx) => apiClient.getKnownError(requirePositional(ctx.args, 0, 'knownErrorId'))),
    cmd('problem.known-errors.search', 'Search known errors', 'itsm problem known-errors search <q>', ['GET /api/v1/known-errors/search'], async (ctx) => apiClient.searchKnownErrors(requirePositional(ctx.args, 0, 'query'))),
  ];
}
