/**
 * `<html lang>` 同步回归锁（缺陷 O6-C）。
 *
 * 修复前：`app/layout.tsx` 硬编码 `<html lang="zh-CN">`，全站没有任何客户端补偿同步，
 * 切到 English 后 `document.documentElement.lang` 永远是 `zh-CN`。
 */
import { render } from '@testing-library/react';

const mockLanguage = { current: 'zh-CN' as 'zh-CN' | 'en-US' };
const mockSubscribers: Array<(prefs: { language: string }) => void> = [];

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({ language: mockLanguage.current }),
}));

import { HtmlLangSync } from '../HtmlLangSync';

describe('HtmlLangSync', () => {
  beforeEach(() => {
    document.documentElement.lang = 'zh-CN';
    mockLanguage.current = 'zh-CN';
    mockSubscribers.length = 0;
  });

  it('中文偏好下 lang 为 zh-CN', () => {
    render(<HtmlLangSync />);
    expect(document.documentElement.lang).toBe('zh-CN');
  });

  // 修复前失败的用例：没有同步点，lang 恒为 SSR 初值。
  it('英文偏好下 lang 切换为 en-US', () => {
    mockLanguage.current = 'en-US';
    render(<HtmlLangSync />);
    expect(document.documentElement.lang).toBe('en-US');
  });

    it('语言变化后重新渲染会再次对齐（不是只在挂载时跑一次）', () => {
    const { rerender } = render(<HtmlLangSync />);
    expect(document.documentElement.lang).toBe('zh-CN');

    mockLanguage.current = 'en-US';
    rerender(<HtmlLangSync />);
    expect(document.documentElement.lang).toBe('en-US');

    mockLanguage.current = 'zh-CN';
    rerender(<HtmlLangSync />);
    expect(document.documentElement.lang).toBe('zh-CN');
  });

  it('组件不渲染任何 DOM 内容（纯副作用）', () => {
    const { container } = render(<HtmlLangSync />);
    expect(container.firstChild).toBeNull();
  });
});
