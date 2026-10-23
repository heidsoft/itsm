import { cmd } from '../spec.js';
import { requirePositional, withPagination, booleanFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
/** Notifications + delivery audit (outbox surface). */
export function notificationCommands() {
    return [
        cmd('notification.list', 'List current-user notifications', 'itsm notification list [--unread] [--page N]', ['GET /api/v1/notifications'], async (ctx) => apiClient.listNotifications({
            ...withPagination(ctx.args),
            unread: booleanFlag(ctx.args.flags, 'unread'),
        })),
        cmd('notification.unread-count', 'Unread count only', 'itsm notification unread-count', ['GET /api/v1/notifications/unread-count'], async () => apiClient.getUnreadNotificationCount()),
        cmd('notification.read', 'Mark one notification read', 'itsm notification read <id>', ['PUT /api/v1/notifications/:id/read'], async (ctx) => apiClient.markNotificationRead(requirePositional(ctx.args, 0, 'notificationId'))),
        cmd('notification.read-all', 'Mark all read', 'itsm notification read-all', ['PUT /api/v1/notifications/read-all'], async () => apiClient.markAllNotificationsRead()),
        cmd('notification.delete', 'Delete a notification', 'itsm notification delete <id>', ['DELETE /api/v1/notifications/:id'], async (ctx) => apiClient.deleteNotification(requirePositional(ctx.args, 0, 'notificationId'))),
        cmd('notification.deliveries', 'Delivery audit trail (per recipient+channel)', 'itsm notification deliveries', ['GET /api/v1/notification-deliveries'], async (ctx) => apiClient.listNotificationDeliveries(withPagination(ctx.args))),
        cmd('notification.delivery-stats', 'Delivery aggregate stats', 'itsm notification delivery-stats', ['GET /api/v1/notification-deliveries/stats'], async () => apiClient.getDeliveryStats()),
        cmd('notification.retry-delivery', 'Retry a failed delivery', 'itsm notification retry-delivery <id>', ['POST /api/v1/notification-deliveries/:id/retry'], async (ctx) => apiClient.retryDelivery(requirePositional(ctx.args, 0, 'deliveryId'))),
    ];
}
