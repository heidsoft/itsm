/**
 * 统一的交互提示出口。错误提示除「发生了什么」外还要给出原因与建议动作；
 * AI 能力失败表现为「已降级」而不是「操作失败」。
 *
 *   notify.success('变更已提交')
 *   notify.error(error)                        // 自动提取可行动文案
 *   notify.error(error, { context: '提交变更' }) // 带操作上下文
 *   notify.error(error, {
 *     action: { label: '重试', onClick: retry },
 *   })
 *   notify.aiUnavailable()                     // AI 能力降级专用
 *
 * 新代码一律走本模块，不再直接 `import { message } from 'antd'`；
 * 存量 `message.*` 可按页面渐进替换。
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
    title: 'Wrong content submitted',
    reason: 'Request parameters failed server-side verification',
    action: 'Please check the red fields and resubmit.',
  },
  '401': {
    title: 'Login status has expired',
    reason: 'Login credentials have expired or been logged out',
    action: 'Please log in again and try again',
  },
  '403': {
    title: 'No permission to perform this operation',
    reason: 'The current role is not granted access to this resource',
    action: 'Please contact the administrator to apply for permissions, or switch to a role with permissions',
  },
  '404': {
    title: 'The resource cannot be found',
    reason: 'The resource may have been deleted or you do not have permission to view it.',
    action: 'Please return to the list to refresh and try again.',
  },
  '409': {
    title: 'Data has been modified by someone else',
    reason: 'The version verification failed during concurrent submission. To avoid overwriting other people\'s changes, this submission has been rejected.',
    action: 'Please refresh the page to view the latest content and resubmit.',
  },
  '422': {
    title: 'Data verification failed',
    reason: 'Some fields do not comply with business rules',
    action: 'Please check the fields marked red in the form',
  },
  '429': {
    title: 'Operation too frequent',
    reason: 'Server-side frequency limit has been triggered',
    action: 'Please wait for a while and try again.',
  },
  '500': {
    title: 'Service is temporarily unavailable',
    reason: 'An internal error occurred during server-side processing',
    action: 'Please try again later; if it continues to appear, please contact the administrator.',
  },
  '502': {
    title: 'Gateway exception',
    action: 'Please try again later; if it continues to appear, please contact the administrator.',
  },
  '503': {
    title: 'The service is under maintenance or overloaded',
    action: 'Please try again later',
  },
  '504': {
    title: 'Service response timed out',
    reason: 'The backend processing time exceeds the gateway limit',
    action: 'Please try again later, or narrow down the query scope and try again.',
  },
  // 网络层
  'Network Error': {
    title: 'Network connection failed',
    reason: 'Unable to connect to server',
    action: 'Please check the network and try again',
  },
  timeout: {
    title: 'Request timeout',
    action: 'Please try again later, or narrow down the query scope and try again.',
  },
  'Request timeout': {
    title: 'Request timeout',
    action: 'Please try again later, or narrow down the query scope and try again.',
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
  const title = context ? `${context} Failure:${guidance.title}` : guidance.title;
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
      message: options.context ? `${options.context} Failed` : 'Operation failed',
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
      text = options.context ? `${options.context} Failure:${friendly}` : friendly;
    }

    // 兜底：解析不出任何有用信息时不暴露原始错误对象
    if (!text || text === "Operation failed, please try again later.") {
      text = options.fallback
        ? options.fallback
        : options.context
          ? `${options.context} Failed, please try again later`
          : 'Operation failed, please try again later.';
    }

    showError(text, options);
  },

  /**
   * AI 能力降级提示。
   *
   * AI 属于增强能力，失败不应表现为"操作失败"——用户最担心的是自己填的内容丢失，
   * 因此明确说明已降级且手动输入不受影响，避免误导为数据问题。
   */
  aiUnavailable(context = 'AI Assistant', options: NotifyOptions = {}): void {
    if (!isBrowser()) return;
    const text = `${context} Temporarily unavailable. The manually filled content has been reserved for you and will not affect normal submission.`;
    if (options.action) {
      showError(text, options);
      return;
    }
    message.warning(text, options.duration ?? 5);
  },

  /** AI 建议可用但置信度偏低时的提示，避免用户盲目采信 */
  aiLowConfidence(context = 'AI suggestion'): void {
    if (!isBrowser()) return;
    message.info(`${context} The confidence level is low. Please judge based on the actual situation before adopting it.`, 4);
  },
};

export default notify;
