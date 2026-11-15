import {
  DEMO_LOGIN_PARAM,
  readDemoCredentials,
  shouldAutoDemoLogin,
  readBuiltinDemoCredentials,
  shouldAutoDemoLoginBuiltin,
} from '../demo-login';

const fullEnv: Record<string, string | undefined> = {
  NEXT_PUBLIC_DEMO_ENABLED: 'true',
  NEXT_PUBLIC_DEMO_USER: 'demo',
  NEXT_PUBLIC_DEMO_PASSWORD: 'test-only-password',
  NEXT_PUBLIC_DEMO_TENANT: 'default',
};

describe('demo-login (一键直达演示)', () => {
  describe('readDemoCredentials', () => {
    it('开关开启且凭据齐全时返回完整凭据', () => {
      expect(readDemoCredentials(fullEnv)).toEqual({
        username: 'demo',
        password: 'test-only-password',
        tenantCode: 'default',
      });
    });

    it('开关未开启时返回 null（开源版默认关闭）', () => {
      expect(readDemoCredentials({ ...fullEnv, NEXT_PUBLIC_DEMO_ENABLED: 'false' })).toBeNull();
      expect(readDemoCredentials({})).toBeNull();
    });

    it('用户名或密码为空时返回 null，绝不用空凭据撞登录', () => {
      expect(readDemoCredentials({ ...fullEnv, NEXT_PUBLIC_DEMO_USER: '  ' })).toBeNull();
      expect(readDemoCredentials({ ...fullEnv, NEXT_PUBLIC_DEMO_PASSWORD: '' })).toBeNull();
    });

    it('租户代码缺省时回退为 default', () => {
      const creds = readDemoCredentials({ ...fullEnv, NEXT_PUBLIC_DEMO_TENANT: '  ' });
      expect(creds?.tenantCode).toBe('default');
    });
  });

  describe('shouldAutoDemoLogin', () => {
    it('URL 带开关且已配置凭据时触发', () => {
      const params = new URLSearchParams(`${DEMO_LOGIN_PARAM}=1`);
      expect(shouldAutoDemoLogin(params, fullEnv)).toBe(true);
    });

    it('URL 无开关时不触发，即使凭据已配置', () => {
      expect(shouldAutoDemoLogin(null, fullEnv)).toBe(false);
      expect(shouldAutoDemoLogin(new URLSearchParams('foo=bar'), fullEnv)).toBe(false);
    });

    it('开关值不是 1 时不触发', () => {
      expect(shouldAutoDemoLogin(new URLSearchParams('demo=true'), fullEnv)).toBe(false);
      expect(shouldAutoDemoLogin(new URLSearchParams('demo=0'), fullEnv)).toBe(false);
    });

    it('开源版（有开关但无凭据）不触发，回退普通登录页', () => {
      const params = new URLSearchParams('demo=1');
      expect(shouldAutoDemoLogin(params, {})).toBe(false);
      expect(shouldAutoDemoLogin(params, { NEXT_PUBLIC_DEMO_ENABLED: 'false' })).toBe(false);
    });
  });

  /**
   * 构建期绑定版本：客户端组件实际调用的函数。
   * 这组词守护「Next.js DefinePlugin 内联」这条生产路径 —— 纯函数版本即使全绿，
   * 也不能证明浏览器能拿到凭据；只有这组直接从 process.env 字面量取值的函数
   * 在 build 阶段被内联，才是真正决定线上能不能一键直达的实现。
   */
  describe('builtin 版本（客户端实际调用 / 构建期内联路径）', () => {
    const KEYS = [
      'NEXT_PUBLIC_DEMO_ENABLED',
      'NEXT_PUBLIC_DEMO_USER',
      'NEXT_PUBLIC_DEMO_PASSWORD',
      'NEXT_PUBLIC_DEMO_TENANT',
    ] as const;
    const original: Record<string, string | undefined> = {};

    beforeEach(() => {
      KEYS.forEach(k => { original[k] = process.env[k]; });
    });
    afterEach(() => {
      KEYS.forEach(k => {
        if (original[k] === undefined) delete process.env[k];
        else process.env[k] = original[k];
      });
    });

    function setEnv(values: Record<string, string>) {
      KEYS.forEach(k => {
        if (values[k] === undefined) delete process.env[k];
        else process.env[k] = values[k];
      });
    }

    it('从 process.env 字面量读取完整凭据', () => {
      setEnv({
        NEXT_PUBLIC_DEMO_ENABLED: 'true',
        NEXT_PUBLIC_DEMO_USER: 'demo',
        NEXT_PUBLIC_DEMO_PASSWORD: 'test-only-password',
        NEXT_PUBLIC_DEMO_TENANT: 'default',
      });
      expect(readBuiltinDemoCredentials()).toEqual({
        username: 'demo',
        password: 'test-only-password',
        tenantCode: 'default',
      });
      expect(shouldAutoDemoLoginBuiltin(new URLSearchParams('demo=1'))).toBe(true);
    });

    it('未配置环境变量时返回 null（开源版默认关闭）', () => {
      setEnv({});
      expect(readBuiltinDemoCredentials()).toBeNull();
      expect(shouldAutoDemoLoginBuiltin(new URLSearchParams('demo=1'))).toBe(false);
    });

    it('反向验证：凭据存在但 URL 无 demo=1 时不触发', () => {
      setEnv({
        NEXT_PUBLIC_DEMO_ENABLED: 'true',
        NEXT_PUBLIC_DEMO_USER: 'demo',
        NEXT_PUBLIC_DEMO_PASSWORD: 'test-only-password',
      });
      expect(shouldAutoDemoLoginBuiltin(null)).toBe(false);
      expect(shouldAutoDemoLoginBuiltin(new URLSearchParams('foo=bar'))).toBe(false);
    });
  });
});
