/**
 * 通知类型词表回归锁（缺陷 O6-A）。
 *
 * 修复前：通知列表页用一份不完整的局部字典（键写 `statusChanged` / `slaWarning`，
 * 而后端落库的是 `status_changed` / `sla_warning`），未命中即 `return type`，
 * 于是英文枚举直接上屏；且标题渲染让 `title` 优先，字典沦为几乎不生效的兜底。
 *
 * 本测试锁住三件事：
 *   1. 后端真实落库的全部枚举都能解析到词表键（不留透传缺口）
 *   2. 未知值返回 null，由调用方显式回退，而不是静默透传
 *   3. 「裸枚举」判定可区分人工文案，避免把正常标题吞掉
 */
import {
  notificationTypeKey,
  isNotificationTypeToken,
  resolveNotificationTitle,
} from '../notification-type';

describe('notification-type 词表', () => {
  // 后端 dto.TicketNotification.Type 的完整值域 + normalizeNotificationEventType
  // 输出的偏好长词。这份清单是「词表覆盖率」的断言依据，新增后端枚举时必须同步。
  const BACKEND_SHORT_ENUMS = [
    'created',
    'assigned',
    'updated',
    'status_changed',
    'commented',
    'sla_warning',
    'sla_alert',
    'sla_breached',
    'resolved',
    'closed',
    'change_approval_required',
    'change_approval_decided',
    'info',
  ] as const;

  const PREFERENCE_LONG_TERMS = [
    'ticket_created',
    'ticket_assigned',
    'ticket_updated',
    'ticket_commented',
    'comment_added',
    'ticket_resolved',
    'ticket_closed',
    'sla_violated',
    'approval_required',
    'approval_completed',
  ] as const;

  describe('后端枚举全部可解析（无透传缺口）', () => {
    it.each([...BACKEND_SHORT_ENUMS, ...PREFERENCE_LONG_TERMS])(
      '%s 必须命中词表键',
      type => {
        expect(notificationTypeKey(type)).not.toBeNull();
      }
    );

    it('短枚举与偏好长词指向同一语义（防止两套写法各自漂移）', () => {
      expect(notificationTypeKey('created')).toBe(notificationTypeKey('ticket_created'));
      expect(notificationTypeKey('assigned')).toBe(notificationTypeKey('ticket_assigned'));
      expect(notificationTypeKey('status_changed')).toBe(notificationTypeKey('ticket_updated'));
      expect(notificationTypeKey('commented')).toBe(notificationTypeKey('comment_added'));
      expect(notificationTypeKey('resolved')).toBe(notificationTypeKey('ticket_resolved'));
      expect(notificationTypeKey('closed')).toBe(notificationTypeKey('ticket_closed'));
      expect(notificationTypeKey('sla_breached')).toBe(notificationTypeKey('sla_violated'));
      expect(notificationTypeKey('change_approval_required')).toBe(
        notificationTypeKey('approval_required')
      );
      expect(notificationTypeKey('change_approval_decided')).toBe(
        notificationTypeKey('approval_completed')
      );
    });

    // 反向验证：修复前的局部字典键（camelCase）在真实后端枚举下必然不命中。
    // 若有人把词表改回旧写法，此断言变红。
    it('旧局部字典的 camelCase 键不是后端真实取值', () => {
      expect(notificationTypeKey('statusChanged')).toBeNull();
      expect(notificationTypeKey('slaWarning')).toBeNull();
    });
  });

  describe('未知值不静默透传', () => {
    it.each(['', '   ', 'some_future_enum', 'https://example.com', '0'])(
      '未知值 %p 返回 null',
      value => {
        expect(notificationTypeKey(value)).toBeNull();
      }
    );

    it('null / undefined 安全返回 null', () => {
      expect(notificationTypeKey(null)).toBeNull();
      expect(notificationTypeKey(undefined)).toBeNull();
    });

    it('前后空白被裁剪后仍可命中', () => {
      expect(notificationTypeKey('  created  ')).toBe('created');
    });
  });

  describe('裸枚举判定可区分人工文案', () => {
    it('枚举原文被判为 token（应翻译）', () => {
      expect(isNotificationTypeToken('created')).toBe(true);
      expect(isNotificationTypeToken('status_changed')).toBe(true);
      expect(isNotificationTypeToken('sla_warning')).toBe(true);
    });

    it('人工文案不被判为 token（应原样保留）', () => {
      expect(isNotificationTypeToken('工单 #123 已分配给您')).toBe(false);
      expect(isNotificationTypeToken('SLA 即将超时，请尽快处理')).toBe(false);
      expect(isNotificationTypeToken('')).toBe(false);
      expect(isNotificationTypeToken(null)).toBe(false);
    });
  });

  describe('标题展示优先级（O6-A 修复点）', () => {
    // 用带前缀的假翻译器，断言「走的是词表」而不是「恰好等于原文」。
    const label = (key: string) => `T:${key}`;

    it('title 是 event_type 裸枚举时必须翻译，不能原样上屏', () => {
      // 修复前：titleText = notification.title → 屏幕上就是 "created"
      const resolved = resolveNotificationTitle(
        { title: 'created', content: '工单 #8 已创建', type: 'created' },
        label
      );
      expect(resolved).toBe('T:created');
      expect(resolved).not.toBe('created');
    });

    it('status_changed / sla_warning 这类下划线枚举同样被翻译（旧字典的漏网值）', () => {
      expect(
        resolveNotificationTitle({ title: 'status_changed', type: 'status_changed' }, label)
      ).toBe('T:statusChanged');
      expect(resolveNotificationTitle({ title: 'sla_warning', type: 'sla_warning' }, label)).toBe(
        'T:slaWarning'
      );
    });

    it('人工标题原样保留（不得被词表吞掉）', () => {
      const humanTitle = '变更审批已由张三通过';
      expect(resolveNotificationTitle({ title: humanTitle, type: 'created' }, label)).toBe(
        humanTitle
      );
    });

    it('title 为空时退回 content', () => {
      expect(resolveNotificationTitle({ title: '', content: '正文内容' }, label)).toBe('正文内容');
      expect(resolveNotificationTitle({ title: '   ', content: '正文内容' }, label)).toBe('正文内容');
    });

    it('title/content 都空时用 type 兜底；type 未知则回退 info', () => {
      expect(resolveNotificationTitle({ type: 'closed' }, label)).toBe('T:closed');
      expect(resolveNotificationTitle({ type: 'unknown_enum' }, label)).toBe('T:info');
      expect(resolveNotificationTitle({}, label)).toBe('T:info');
    });
  });
});
