/**
 * 统一交互提示出口（AI-Native 友好）
 *
 * 设计目标
 * 1. 统一出口：收敛散落在 200+ 文件中的裸 `message.*` 调用，避免同一错误在不同
 *    页面提示不一致（历史问题见 output/ux-audit-2026-09-12.md）。
 * 2. 可行动文案：提示必须回答三件事——发生了什么 / 为什么 / 怎么办。
 *    现有 `getFriendlyErrorMessage` 只给一句话（如 403 → "没有权限执行此操作"），
 *    用户知道失败了但不知道下一步做什么，本模块在其之上补「建议动作」层。
 * 3. AI 降级友好：AI 是增强能力而非主链路，AI 失败不应表现为"操作失败"，
 *    而要明确告知"已降级，你的手动输入不受影响"，避免用户以为数据丢了。
 *
 * 使用方式
 *   notify.success('变更已提交')
 *   notify.error(error)                        // 自动提取可行动文案
 *   notify.error(error, { context: '提交变更' }) // 带操作上下文
 *   notify.error(error, {
 *     action: { label: '重试', onClick: retry },
 *   })
 *   notify.aiUnavailable()                     // AI 能力降级专用
 *
 * 迁移约定
 *   - 新代码一律使用本模块，不再直接 `import { message } from 'antd'`。
 *   - 存量 `message.*` 可按页面渐进替换，行为保持兼容。
 */

import { Button, message, notification } from 'antd';
import { getFriendlyErrorMessage } from '@/lib/api/base-api-handler';

/** 提示中的「建议动作」，用于把用户从"知道失败"引导到"知道做什么" */
export interface NotifyAction {
  label: string;
  onClick: () => void | Promise<void>;
}

export interface NotifyOptions {
  /** 操作上下文，用于把泛化错误具体化，如「提交变更失败」而非「操作失败」 */
  context?: string;
  /** 兜底文案，无法识别错误类型时使用 */
  fallback?: string;
  /** 建议动作；提供时改用 notification 承载按钮（message 不支持交互） */
  action?: NotifyAction;
  /** 停留时长（秒），默认 3；错误类默认 4.5 以便阅读建议动作 */
  duration?: number;
}

/** 一条可行动的错误提示：发生了什么 / 为什么 / 怎么办 */
interface ErrorGuidance {
  /** 发生了什么 */
  title: string;
  /** 为什么（可空，避免编造原因） */
  reason?: string;
  /** 怎么办 */
  action?: string;
}

/**
 * 错误码 → 可行动引导
 *
 * 键为 HTTP 状态码或错误特征串，与 base-api-handler 的 ERROR_MESSAGES 对齐，
 * 但额外提供「为什么 / 怎么办」。不要在此编造业务原因，只写确实成立的通用解释。
 */
const ERROR_GUIDANCE: Record<string, ErrorGuidance> = {
  '400': {
    title: '提交的内容有误',
    reason: '请求参数未通过服务端校验',
    action: '请检查标红字段后重新提交',
  },
  '401': {
    title: '登录状态已失效',
    reason: '登录凭证已过期或已被注销',
    action: '请重新登录后再试',
  },
  '403': {
    title: '没有权限执行此操作',
    reason: '当前角色未被授予该资源的访问权限',
    action: '请联系管理员申请权限，或切换到有权限的角色',
  },
  '404': {
    title: '找不到该资源',
    reason: '资源可能已被删除，或你无权查看',
    action: '请返回列表刷新后重试',
  },
  '409': {
    title: '数据已被其他人修改',
    reason: '并发提交时版本校验未通过，为避免覆盖他人改动已拒绝本次提交',
    action: '请刷新页面查看最新内容后重新提交',
  },
  '422': {
    title: '数据校验未通过',
    reason: '部分字段不符合业务规则',
    action: '请检查表单中标红的字段',
  },
  '429': {
    title: '操作过于频繁',
    reason: '已触发服务端的频率限制',
    action: '请稍等片刻后重试',
  },
  '500': {
    title: '服务暂时不可用',
    reason: '服务端处理时发生内部错误',
    action: '请稍后重试；若持续出现请联系管理员',
  },
  '502': {
    title: '网关异常',
    action: '请稍后重试；若持续出现请联系管理员',
  },
  '503': {
    title: '服务正在维护或过载',
    action: '请稍后重试',
  },
  '504': {
    title: '服务响应超时',
    reason: '后端处理时间超过了网关上限',
    action: '请稍后重试，或缩小查询范围后再试',
  },
  // 网络层
  'Network Error': {
    title: '网络连接失败',
    reason: '无法连接到服务器',
    action: '请检查网络后重试',
  },
  timeout: {
    title: '请求超时',
    action: '请稍后重试，或缩小查询范围后再试',
  },
  'Request timeout': {
    title: '请求超时',
    action: '请稍后重试，或缩小查询范围后再试',
  },
};

/** 从异常中提取 HTTP 状态码或错误特征键 */
function resolveGuidanceKey(error: unknown): string | undefined {
  const candidates: Array<unknown> = [];

  if (error && typeof error === 'object') {
    const e = error as Record<string, unknown>;
    candidates.push(e.code, e.status, e.statusCode);
    const resp = e.response as Record<string, unknown> | undefined;
    if (resp) candidates.push(resp.status, (resp.data as Record<string, unknown>)?.code);
  }

  for (const c of candidates) {
    if (c === undefined || c === null) continue;
    const key = String(c);
    if (ERROR_GUIDANCE[key]) return key;
  }

  // 退化为消息文本匹配（后端有时把状态码写进 message）
  const text = error instanceof Error ? error.message : typeof error === 'string' ? error : '';
  for (const key of Object.keys(ERROR_GUIDANCE)) {
    if (text.includes(key)) return key;
  }
  return undefined;
}

/** 组装最终提示文案：发生了什么 / 为什么 / 怎么办 */
function composeMessage(guidance: ErrorGuidance, context?: string): string {
  const title = context ? `${context}失败：${guidance.title}` : guidance.title;
  const parts = [title];
  if (guidance.reason) parts.push(guidance.reason);
  if (guidance.action) parts.push(guidance.action);
  return parts.join('，');
}

function isBrowser(): boolean {
  return typeof window !== 'undefined';
}

/** 需要用户操作的提示用 notification（支持按钮），否则用 message（更轻量） */
function showError(text: string, options: NotifyOptions): void {
  if (!isBrowser()) return;

  if (options.action) {
    // 注意：antd v6 已废弃 notification 的 `btn`，改用 `actions`
    notification.error({
      message: options.context ? `${options.context}失败` : '操作失败',
      description: text,
      duration: options.duration ?? 6,
      actions: (
        <Button
          size="small"
          danger
          onClick={() => {
            void options.action?.onClick();
            notification.destroy();
          }}
        >
          {options.action.label}
        </Button>
      ),
    });
    return;
  }

  message.error(text, options.duration ?? 4.5);
}

export const notify = {
  success(text: string, options: Omit<NotifyOptions, 'fallback' | 'action'> = {}): void {
    if (!isBrowser()) return;
    message.success(text, options.duration ?? 3);
  },

  info(text: string, options: Omit<NotifyOptions, 'fallback' | 'action'> = {}): void {
    if (!isBrowser()) return;
    message.info(text, options.duration ?? 3);
  },

  warning(text: string, options: Omit<NotifyOptions, 'fallback' | 'action'> = {}): void {
    if (!isBrowser()) return;
    message.warning(text, options.duration ?? 4);
  },

  /**
   * 错误提示。接受任意异常，自动补全「为什么 / 怎么办」。
   *
   * 解析顺序：结构化引导表 → base-api-handler 的友好文案 → 调用方兜底 → 通用兜底。
   * 始终优先给出可行动信息，绝不把后端原始错误串直接抛给用户。
   */
  error(error: unknown, options: NotifyOptions = {}): void {
    if (!isBrowser()) return;

    const key = resolveGuidanceKey(error);
    const guidance = key ? ERROR_GUIDANCE[key] : undefined;

    let text: string;
    if (guidance) {
      text = composeMessage(guidance, options.context);
    } else {
      const friendly = getFriendlyErrorMessage(error);
      text = options.context ? `${options.context}失败：${friendly}` : friendly;
    }

    // 兜底：解析不出任何有用信息时不暴露原始错误对象
    if (!text || text === '操作失败，请稍后重试') {
      text = options.fallback
        ? options.fallback
        : options.context
          ? `${options.context}失败，请稍后重试`
          : '操作失败，请稍后重试';
    }

    showError(text, options);
  },

  /**
   * AI 能力降级提示。
   *
   * AI 属于增强能力，失败不应表现为"操作失败"——用户最担心的是自己填的内容丢失，
   * 因此明确说明已降级且手动输入不受影响，避免误导为数据问题。
   */
  aiUnavailable(context = 'AI 助手', options: NotifyOptions = {}): void {
    if (!isBrowser()) return;
    const text = `${context}暂时不可用，已为你保留手动填写的内容，不影响正常提交`;
    if (options.action) {
      showError(text, options);
      return;
    }
    message.warning(text, options.duration ?? 5);
  },

  /** AI 建议可用但置信度偏低时的提示，避免用户盲目采信 */
  aiLowConfidence(context = 'AI 建议'): void {
    if (!isBrowser()) return;
    message.info(`${context}置信度较低，请结合实际情况判断后再采纳`, 4);
  },
};

export default notify;
