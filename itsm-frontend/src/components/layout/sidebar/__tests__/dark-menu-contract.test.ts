/**
 * 深色模式 + 菜单交互回归锁（文本级契约测试）
 *
 * 这些修复大多是 CSS 变量 / inline script / JSX 属性的改动，mock DOM 无法稳定复现
 * 浏览器级联与 portal 行为。参照项目 C.6.4 基线与 icon-map-parity 的做法，用源文件
 * 文本断言锁定关键不变量——每条断言都对应一个曾真实复现的缺陷：
 *
 *  1. 浮层深底深字：tooltip/popover/dropdown 文字误用 --color-text-inverse，
 *     深色下为 #0f172a 配 #1e293b 背景（对比度约 1:1，几乎不可见）。
 *  2. 菜单白块：Menu 写死 theme="light"，覆盖 ConfigProvider 的 darkAlgorithm，
 *     收缩态 submenu popup portal 到 body 后渲染成白底。
 *  3. FOUC 闪白：SSR 首帧无 .dark，ThemeHtmlClassSync 用 useEffect 同步晚于首帧。
 *  4. 侧栏用户名：亮色下硬编码深色字，dark 覆盖块遗漏 userName/userRole。
 */

import { readFileSync } from 'fs';
import { join } from 'path';

const ROOT = join(__dirname, '../../../../../');
const read = (rel: string) => readFileSync(join(ROOT, rel), 'utf-8');

const GLOBALS = 'src/app/globals.css';
const MENU_ITEMS = 'src/components/layout/sidebar/MenuItems.tsx';
const LAYOUT = 'src/app/layout.tsx';
const SIDEBAR_CSS = 'src/components/layout/sidebar/Sidebar.module.css';
const ANTD_OVERRIDES = 'src/styles/antd-layout-overrides.css';

describe('深色模式 + 菜单交互回归锁', () => {
  describe('① 浮层文字对比度', () => {
    it('浮层文字使用 --color-text-primary 而非 --color-text-inverse', () => {
      const css = read(GLOBALS);
      // tooltip/popover/dropdown 三个浮层的文字色必须跟随主文字色：
      // 亮色 primary=#0f172a（配白底）、深色 primary=#f8fafc（配深底）。
      // 旧代码用 text-inverse，深色下=#0f172a，与深底叠加成深底深字。
      const overlaySelectors = [
        '.ant-tooltip-inner',
        '.ant-popover-inner',
        '.ant-dropdown-menu',
      ];
      for (const selector of overlaySelectors) {
        const block = css.slice(css.indexOf(selector), css.indexOf(selector) + 300);
        expect(block).toContain('color: var(--color-text-primary) !important');
        expect(block).not.toContain('color: var(--color-text-inverse) !important');
      }
      // 全局不应再有浮层误用 text-inverse（防止新浮层复制旧代码）
      expect(css).not.toMatch(/color:\s*var\(--color-text-inverse\)\s*!important/);
    });
  });

  describe('② 菜单主题跟随暗色', () => {
    it('Menu 组件不再写死 theme="light"', () => {
      const tsx = read(MENU_ITEMS);
      expect(tsx).not.toMatch(/theme=["']light["']/);
      // 保留 ConfigProvider 驱动的说明，防止后续又加回硬编码
      expect(tsx).toContain('ConfigProvider');
    });

    it('深色展开和收缩菜单共享选中底色，覆盖亮色渐变', () => {
      const css = read(ANTD_OVERRIDES);
      expect(css).toMatch(
        /html\.dark \.ant-menu \.ant-menu-item-selected[^{}]*\{[^}]*background: var\(--color-primary-soft\) !important/
      );
      expect(css).toMatch(
        /html\.dark \.ant-menu-sub\.ant-menu-inline \{[^}]*background: transparent !important/
      );
    });

    it('收缩态 submenu popup 有暗色背景与文字覆盖', () => {
      const css = read(ANTD_OVERRIDES);
      // 必须存在一个 html.dark .ant-menu-submenu-popup 规则块，块内同时设暗底与浅字。
      // 只断言全文包含变量会导致 hover 等其他规则让它假绿，因此精确切出规则块。
      const blockStart = css.indexOf('html.dark .ant-menu-submenu-popup .ant-menu,');
      expect(blockStart).toBeGreaterThan(-1);
      const block = css.slice(blockStart, css.indexOf('}', blockStart));
      expect(block).toContain('background-color: var(--color-bg-elevated) !important');
      expect(block).toContain('color: var(--color-text-primary) !important');
    });
  });

  describe('③ FOUC 首帧主题同步', () => {
    it('layout head 内有 beforeInteractive 主题预同步脚本', () => {
      const tsx = read(LAYOUT);
      expect(tsx).toContain('dangerouslySetInnerHTML');
      // 脚本必须与 ThemeProvider 使用同一个 storageKey
      expect(tsx).toContain("localStorage.getItem('itsm-theme')");
      expect(tsx).toContain("classList.add('dark')");
      // 必须处理 system 模式跟随系统偏好
      expect(tsx).toContain('prefers-color-scheme: dark');
    });
  });

  describe('④ 侧栏用户名深色可读', () => {
    it('dark 覆盖块包含 userName / userRole / userAvatar', () => {
      const css = read(SIDEBAR_CSS);
      expect(css).toContain(':global(html.dark) .userName');
      expect(css).toContain(':global(html.dark) .userRole');
      expect(css).toContain(':global(html.dark) .userAvatar');
      // 暗色下必须走主题变量，不能残留硬编码 #1f2937 / #6b7280
      const darkBlock = css.slice(css.indexOf(':global(html.dark) .userName'));
      expect(darkBlock).toContain('var(--color-text-primary)');
      expect(darkBlock).toContain('var(--color-text-muted)');
    });
  });
});
