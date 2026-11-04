import { cmd } from '../spec.js';
import { requirePositional, stringFlag, booleanFlag, body, withPagination } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';

/**
 * Marketplace (tenant face). Covers the routes a tenant installs/uninstalls/
 * upgrades extensions through. The admin catalog CRUD face
 * (/marketplace/catalog/*) is deliberately not exposed — see api-client.ts.
 *
 * install/uninstall inherit high-risk-write from the verb (the /install/
 * substring in deriveRisk matches both), so they go through the confirmation
 * gate like every other destructive command.
 */

export function marketplaceCommands() {
  return [
    cmd(
      'marketplace.items',
      'Browse the marketplace catalog visible to this tenant',
      'itsm marketplace items [--type t] [--category c] [--search kw] [--official] [--page N] [--pageSize N]',
      ['GET /api/v1/marketplace/items'],
      async (ctx) => {
        const flags = ctx.args.flags;
        return apiClient.listMarketplaceItems({
          type: stringFlag(flags, 'type'),
          category: stringFlag(flags, 'category'),
          search: stringFlag(flags, 'search'),
          isOfficial: booleanFlag(flags, 'official'),
          ...withPagination(ctx.args),
        });
      }
    ),
    cmd(
      'marketplace.item',
      'Marketplace item detail',
      'itsm marketplace item <id>',
      ['GET /api/v1/marketplace/items/:id'],
      async (ctx) => apiClient.getMarketplaceItem(requirePositional(ctx.args, 0, 'id'))
    ),
    cmd(
      'marketplace.install',
      'Install a marketplace item into this tenant',
      'itsm marketplace install <id>',
      ['POST /api/v1/marketplace/items/:id/install'],
      async (ctx) => apiClient.installMarketplaceItem(requirePositional(ctx.args, 0, 'id'))
    ),
    cmd(
      'marketplace.uninstall',
      'Remove an installed marketplace item from this tenant',
      'itsm marketplace uninstall <id>',
      ['POST /api/v1/marketplace/items/:id/uninstall'],
      async (ctx) => apiClient.uninstallMarketplaceItem(requirePositional(ctx.args, 0, 'id'))
    ),
    cmd(
      'marketplace.upgrade',
      'Upgrade an installed item to a target version (default: catalog latest)',
      'itsm marketplace upgrade <id> [--targetVersion v]',
      ['POST /api/v1/marketplace/items/:id/upgrade'],
      async (ctx) => {
        const id = requirePositional(ctx.args, 0, 'id');
        const targetVersion = stringFlag(ctx.args.flags, 'targetVersion');
        // Backend contract: an empty string is an explicit 400, never "latest".
        if (targetVersion === '') {
          throw usageError('--targetVersion 不能是空串；省略该 flag 即升级到目录当前最新版本。');
        }
        return apiClient.upgradeMarketplaceItem(id, targetVersion);
      },
      // 'upgrade' matches neither risk regex, so without this it would read as
      // an un-gated no-op. Re-loading the runtime is a platform write.
      'write' as const
    ),
    cmd(
      'marketplace.auto-upgrade',
      "Switch an installed item's auto-upgrade preference on or off",
      'itsm marketplace auto-upgrade <id> <on|off>',
      ['PATCH /api/v1/marketplace/items/:id/auto-upgrade'],
      async (ctx) => {
        const id = requirePositional(ctx.args, 0, 'id');
        const raw = requirePositional(ctx.args, 1, 'on|off');
        if (raw !== 'on' && raw !== 'off') throw usageError(`auto-upgrade 取值必须是 'on' 或 'off'，收到 '${raw}'。`);
        return apiClient.setMarketplaceAutoUpgrade(id, raw === 'on');
      },
      'write' as const
    ),
    cmd(
      'marketplace.installations',
      'List installed items for this tenant',
      'itsm marketplace installations [--status s]',
      ['GET /api/v1/marketplace/installations'],
      async (ctx) => apiClient.listMarketplaceInstallations(stringFlag(ctx.args.flags, 'status'))
    ),
    cmd(
      'marketplace.installation',
      'Installation detail and tenant config for one installed item',
      'itsm marketplace installation <itemId>',
      ['GET /api/v1/marketplace/installations/:itemId'],
      async (ctx) => apiClient.getMarketplaceInstallation(requirePositional(ctx.args, 0, 'itemId'))
    ),
    cmd(
      'marketplace.installation-config',
      'Update the tenant-level config of an installed item',
      'itsm marketplace installation-config <itemId> --data <json>',
      ['PUT /api/v1/marketplace/installations/:itemId/config'],
      async (ctx) => {
        const itemId = requirePositional(ctx.args, 0, 'itemId');
        const config = body(ctx.args);
        // The backend takes a free-form JSON map; sending nothing is a
        // parameter error, not a no-op save.
        if (!config) throw usageError('安装配置更新需要 --data <json>（自由 JSON 对象）。');
        return apiClient.updateMarketplaceInstallationConfig(itemId, config);
      },
      // /install/ substring would pull this into high-risk-write; a config
      // update is a normal write, declared explicitly to keep that decided.
      'write' as const
    ),
    cmd(
      'marketplace.dependency-status',
      'Read-only dependency alignment report for every installed item',
      'itsm marketplace dependency-status',
      ['GET /api/v1/marketplace/dependency-status'],
      async () => apiClient.getMarketplaceDependencyStatus()
    ),
  ];
}
