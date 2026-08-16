"use strict";

// 自定义 Jest 测试环境:让 jsdom 的样式级联遍历对非法选择器"跳过"而非抛错
//
// 背景:antd v6 Card 无条件注册 `:not(:has(> .ant-card-head))` 样式规则。
// getComputedStyle(testing-library 的 getByRole 会通过 isHidden 触发)会让 jsdom
// 用 nwsapi 逐条匹配样式规则。nwsapi 把 `:has(...)` 编译为 element.querySelector(
// ":scope ..."),并把元素的第一个 class 代入 `:scope` —— Tailwind 任意值类(如
// bg-[var(--color-surface-primary)])代入后选择器非法,nwsapi 直接抛 SyntaxError,
// 而浏览器只会跳过该规则。
//
// 为什么不能用 jest.mock('nwsapi') 或 jest.setup.js 拦截:jsdom 在环境构造时(测试
// 文件注册表之外)已经捕获了真实的 nwsapi 引用,且 document impl 与全局 document
// wrapper 是两个对象(jsdom 惰性读取 impl 上的 `_nwsapiDontThrow` / `_nwsapi`)。
// 因此必须在本文件(与 jsdom 同属环境模块注册表,可拿到 impl)的 setup() 里,把这两个
// 惰性实例替换为"匹配失败返回 false"的安全包装 —— 等价于"规则不生效",与浏览器行为一致。

const JSDOMEnvironment = require('jest-environment-jsdom').default;
const { implForWrapper } = require('jsdom/lib/jsdom/living/generated/utils');
const nwsapi = require('nwsapi');

const withSafeMatch = instance => {
  const match = instance.match.bind(instance);
  instance.match = (selector, element, callback) => {
    try {
      return match(selector, element, callback);
    } catch {
      return false;
    }
  };
  return instance;
};

const createSafeInstance = (document, DOMException, configureOptions) => {
  const instance = nwsapi({ document, DOMException });
  instance.configure(configureOptions);
  return withSafeMatch(instance);
};

class SafeNwsapiJSDOMEnvironment extends JSDOMEnvironment {
  async setup() {
    await super.setup();

    const window = this.global;
    if (!window || !window.document) return;

    const docImpl = implForWrapper(window.document);
    if (!docImpl) return;

    // 无条件覆盖:jsdom 在环境构造期间可能已创建未包装实例
    docImpl._nwsapiDontThrow = createSafeInstance(window.document, window.DOMException, {
      LOGERRORS: false,
      VERBOSITY: false,
      IDS_DUPES: true,
      MIXEDCASE: true,
    });
    docImpl._nwsapi = createSafeInstance(window.document, window.DOMException, {
      LOGERRORS: false,
      IDS_DUPES: true,
      MIXEDCASE: true,
    });
  }
}

module.exports = SafeNwsapiJSDOMEnvironment;
