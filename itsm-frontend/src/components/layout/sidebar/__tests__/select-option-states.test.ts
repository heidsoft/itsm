import fs from 'node:fs';
import path from 'node:path';

const repo = path.resolve(__dirname, '../../../../..');

/**
 * AUDIT-041 回归锁：禁用态 Select 选项必须和可用态有明显区别。
 * 这两条规则此前分别被亮色/暗色的 catch-all 覆盖，测试用于防止再次被顺序调整冲掉。
 */
function rulesAfterMarker(css: string): string {
  const marker = '/* 禁用态 Select 选项必须与可用态明显不同。';
  const start = css.indexOf(marker);
  return start < 0 ? '' : css.slice(start);
}

describe('Select 禁用态与菜单模块一致性', () => {
  const globals = fs.readFileSync(path.join(repo, 'src/app/globals.css'), 'utf8');
  const menuItems = fs.readFileSync(
    path.join(repo, 'src/components/layout/sidebar/MenuItems.tsx'),
    'utf8',
  );

  it('亮色下禁用选项有独立颜色与 not-allowed 光标', () => {
    const block = rulesAfterMarker(globals);
    expect(block).toContain('color: #94a3b8 !important');
    expect(block).toContain('cursor: not-allowed !important');
  });

  it('暗色下禁用选项使用主题变量而不是 #fff', () => {
    const block = rulesAfterMarker(globals);
    expect(block).toContain('html.dark .ant-select-item-option-disabled');
    expect(block).toContain('color: var(--color-text-disabled) !important');
    expect(block).toContain('cursor: not-allowed !important');
  });

  it('菜单项渲染组件与 Sidebar 使用同一个 CSS Module', () => {
    expect(menuItems).toContain("import styles from './Sidebar.module.css'");
    expect(menuItems).not.toContain("import styles from '../Sidebar.module.css'");
  });
});
