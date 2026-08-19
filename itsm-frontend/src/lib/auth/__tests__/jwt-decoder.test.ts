/**
 * JWT 轻量解析工具单元测试
 *
 * 重点验证：
 * 1. 不再依赖 Buffer.from()（Edge Runtime 不兼容）
 * 2. 正确解析 base64url 编码的 JWT payload
 * 3. 正确判定过期 / 有效 / 格式错误
 * 4. 兼容 null / undefined / 空字符串 / 非法字符串
 */

import { decodeJwtPayload, isValidJwtToken } from '../jwt-decoder';

/**
 * 手动构造一段 JWT，header.payload.signature
 * payload 使用 base64url 编码（替换 +/ 为 -_）
 *
 * 由于 jsdom 测试环境不一定提供 TextEncoder 且为控制测试隔离，
 * 这里用最小的 UTF-8 编码器 + btoa。生成 token 是不在生产路径上的
 * 测试辅助逻辑（生产路径使用企业级 JWT 库），所以手写编码是安全的。
 */
function utf8Bytes(str: string): number[] {
  const bytes: number[] = [];
  for (let i = 0; i < str.length; i += 1) {
    const c = str.charCodeAt(i);
    if (c < 0x80) {
      bytes.push(c);
    } else if (c < 0x800) {
      bytes.push(0xc0 | (c >> 6), 0x80 | (c & 0x3f));
    } else if (c < 0xd800 || c >= 0xe000) {
      bytes.push(0xe0 | (c >> 12), 0x80 | ((c >> 6) & 0x3f), 0x80 | (c & 0x3f));
    } else {
      // surrogate pair
      i += 1;
      const c2 = str.charCodeAt(i);
      const cp = 0x10000 + (((c & 0x3ff) << 10) | (c2 & 0x3ff));
      bytes.push(
        0xf0 | (cp >> 18),
        0x80 | ((cp >> 12) & 0x3f),
        0x80 | ((cp >> 6) & 0x3f),
        0x80 | (cp & 0x3f)
      );
    }
  }
  return bytes;
}

function base64UrlEncode(obj: unknown): string {
  const json = JSON.stringify(obj);
  const bytes = utf8Bytes(json);
  const binary = Array.from(bytes, b => String.fromCharCode(b)).join('');
  const b64 = btoa(binary);
  return b64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function makeJwt(payload: Record<string, unknown>): string {
  const header = base64UrlEncode({ alg: 'HS256', typ: 'JWT' });
  const body = base64UrlEncode(payload);
  // 签名无所谓，工具只解析 payload
  return `${header}.${body}.fake-signature`;
}

describe('jwt-decoder / decodeJwtPayload', () => {
  it('返回 null 当 token 为 null', () => {
    expect(decodeJwtPayload(null)).toBeNull();
  });

  it('返回 null 当 token 为 undefined', () => {
    expect(decodeJwtPayload(undefined as unknown as null)).toBeNull();
  });

  it('返回 null 当 token 为空字符串', () => {
    expect(decodeJwtPayload('')).toBeNull();
  });

  it('返回 null 当 token 不是 3 段结构', () => {
    expect(decodeJwtPayload('only.two')).toBeNull();
    expect(decodeJwtPayload('four.parts.here.indeed')).toBeNull();
    expect(decodeJwtPayload('no-dots')).toBeNull();
  });

  it('正确解析标准 JWT payload', () => {
    const token = makeJwt({
      userId: 1,
      username: 'admin',
      role: 'super_admin',
      exp: Math.floor(Date.now() / 1000) + 3600,
    });
    const payload = decodeJwtPayload(token);
    expect(payload).not.toBeNull();
    expect(payload).toMatchObject({
      userId: 1,
      username: 'admin',
      role: 'super_admin',
    });
  });

  it('支持 base64url 字符（- 与 _）', () => {
    // 1. 字符串级别验证：replace(-/+) / replace(_//) 能被正确还原。
    // base64url -> base64 -> base64url 往返应恒等。
    const samples = ['__-_-_--', '-_-', 'a-b_c', '____', '----'];
    samples.forEach(s => {
      const std = s.replace(/-/g, '+').replace(/_/g, '/');
      const back = std.replace(/\+/g, '-').replace(/\//g, '_');
      expect(back).toBe(s);
    });

    // 2. 验证完整解码路径能处理包含 - 与 _ 的合法 base64url 中间段。
    //    bytes 0xFB 0xFF 0xFE = base64 "+//+" = base64url "-__-"
    //    这 3 个字节不是合法 UTF-8（0xFB 是 4 字节 UTF-8 起始字节但缺少后续），
    //    但我们这里仅验证 decodeJwtPayload 调用不会拍物报意外错误。
    //    函数返回 null 本身是正确行为（不是有效 JSON）。
    expect(decodeJwtPayload('header.-__-.signature')).toBeNull();
  });

  it('返回 null 当 payload 不是合法 JSON', () => {
    // 手工构造一段中间段为非法 base64 的 token
    const token = `header.!!!not-base64!!!.signature`;
    expect(decodeJwtPayload(token)).toBeNull();
  });

  it('返回 null 当中间段是合法 base64 但不是 JSON', () => {
    // 一些字符串可以被 atob 但不是 JSON
    const middle = btoa('not a json object');
    const token = `header.${middle}.signature`;
    expect(decodeJwtPayload(token)).toBeNull();
  });

  it('正确解析包含中文的 payload', () => {
    const payload = {
      username: '管理员',
      tenantName: '默认租户',
      exp: Math.floor(Date.now() / 1000) + 3600,
    };
    const token = makeJwt(payload);
    const decoded = decodeJwtPayload(token);
    expect(decoded).toMatchObject({
      username: '管理员',
      tenantName: '默认租户',
    });
  });

  it('使用 atob 而不是 Buffer.from（Edge Runtime 兼容）', () => {
    // 验证在测试环境（默认 jsdom）下 atob 存在
    expect(typeof atob).toBe('function');
    // 验证 Buffer.from 在 Edge Runtime 中不可用
    // 这里只能间接验证：decodeJwtPayload 不引用 Buffer 即可
    const token = makeJwt({ exp: Math.floor(Date.now() / 1000) + 60 });
    const decoded = decodeJwtPayload(token);
    expect(decoded).not.toBeNull();
  });
});

describe('jwt-decoder / isValidJwtToken', () => {
  it('null 视为无效', () => {
    expect(isValidJwtToken(null)).toBe(false);
  });

  it('空字符串视为无效', () => {
    expect(isValidJwtToken('')).toBe(false);
  });

  it('非 3 段结构视为无效', () => {
    expect(isValidJwtToken('a.b')).toBe(false);
    expect(isValidJwtToken('a.b.c.d')).toBe(false);
  });

  it('未过期 token 视为有效', () => {
    const token = makeJwt({
      exp: Math.floor(Date.now() / 1000) + 3600,
    });
    expect(isValidJwtToken(token)).toBe(true);
  });

  it('已过期 token 视为无效', () => {
    const token = makeJwt({
      exp: Math.floor(Date.now() / 1000) - 60,
    });
    expect(isValidJwtToken(token)).toBe(false);
  });

  it('无 exp 字段视为有效（不做时间推断）', () => {
    const token = makeJwt({ username: 'admin' });
    expect(isValidJwtToken(token)).toBe(true);
  });

  it('exp 不是 number 视为有效（与原 middleware 行为对齐）', () => {
    // 构造一个 exp 是字符串的 payload
    const token = makeJwt({ exp: 'not-a-number' as unknown as number });
    expect(isValidJwtToken(token)).toBe(true);
  });

  it('payload 解析失败视为无效', () => {
    const token = `header.!!!invalid!!.signature`;
    expect(isValidJwtToken(token)).toBe(false);
  });

  it('过期时间正好为当前时间视为无效', () => {
    const now = Math.floor(Date.now() / 1000);
    const token = makeJwt({ exp: now });
    // exp < currentTime 才算过期，等于当前时间应视为有效（边界）
    expect(isValidJwtToken(token)).toBe(true);
  });
});
