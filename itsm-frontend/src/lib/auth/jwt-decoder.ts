/**
 * JWT 轻量解析工具（不验签，仅做格式/过期检查）
 *
 * 为何需要此工具：
 * - Next.js middleware 默认运行在 Edge Runtime，Edge Runtime 禁用了
 *   `Buffer.from()` / `eval` / `Function` 等基于代码生成的 API，
 *   在 middleware 中直接 `Buffer.from(part, 'base64').toString('utf-8')`
 *   会抛出 "Code generation from strings disallowed for this context"。
 * - 解决方案：使用 Web 标准的 `atob()` + TextDecoder 还原
 *   JWT payload（base64url → UTF-8 → JSON 对象）。
 *
 * 安全说明：
 * - 本工具**不做签名校验**，仅做 token 格式轻校验（exp 字段过期判断）。
 *   真正的鉴权仍由后端接口（或 Next.js API 代理）执行。
 * - 该策略与原 middleware 实现保持一致，避免引入新的安全断言。
 */

/**
 * 解码 base64/base64url 字符串为 UTF-8 文本
 * Edge Runtime 不提供 Buffer，所以走 atob + TextDecoder 路径。
 * 优先使用 TextDecoder（现代 Edge Runtime / 浏览器 / Node 18+ 上可用），
 * 不可用时退到 escape + decodeURIComponent（V8 / 充分兼容）。
 */
function decodeBase64ToUtf8(b64: string): string {
  const normalized = b64.replace(/-/g, '+').replace(/_/g, '/');
  const binary = atob(normalized);
  // atob 返回的 binary 是 Latin-1 编码（每字节 = 一个 code point 0-255），
  // 当原始是 UTF-8 字节时，需要重组成 UTF-8 bytes 再解码。
  if (typeof TextDecoder !== 'undefined') {
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i);
    }
    return new TextDecoder().decode(bytes);
  }
  // 兜底：escape() 把 Latin-1 转为 %XX，decodeURIComponent 反解码。
  // escape 在 V8 上多年保持可用，Edge Runtime 中同样可用。
  let utf8 = '';
  for (let i = 0; i < binary.length; i += 1) {
    utf8 += '%' + binary.charCodeAt(i).toString(16).padStart(2, '0');
  }
  return decodeURIComponent(utf8);
}

export function decodeJwtPayload(token: string | null): Record<string, unknown> | null {
  if (!token) return null;
  const parts = token.split('.');
  if (parts.length !== 3) return null;
  try {
    const json = decodeBase64ToUtf8(parts[1]);
    return JSON.parse(json) as Record<string, unknown>;
  } catch {
    return null;
  }
}

/**
 * 验证 token 格式是否有效（JWT 3 段结构 + 未过期）
 */
export function isValidJwtToken(token: string | null): boolean {
  const payload = decodeJwtPayload(token);
  if (!payload) return false;
  if (payload.exp) {
    const currentTime = Math.floor(Date.now() / 1000);
    if (typeof payload.exp === 'number' && payload.exp < currentTime) {
      return false;
    }
  }
  return true;
}
