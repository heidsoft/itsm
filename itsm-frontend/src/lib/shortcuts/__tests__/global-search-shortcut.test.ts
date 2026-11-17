/**
 * 全局搜索快捷键回归锁（缺陷 O6-B）。
 *
 * 修复前：三处各自注册 window keydown，其中两处只 preventDefault 不 open；
 * 且都用 `e.key === 'k'` 严格比较，Shift+K / CapsLock 下 `e.key` 为 `'K'` 不命中，
 * 表现为「顶栏标注 Ctrl K，但按了没反应」。
 */
import { isGlobalSearchShortcut } from '../global-search-shortcut';

const keyEvent = (
  overrides: Partial<KeyboardEvent> & { key: string }
): KeyboardEvent =>
  ({
    ctrlKey: false,
    metaKey: false,
    altKey: false,
    shiftKey: false,
    isComposing: false,
    ...overrides,
  }) as KeyboardEvent;

describe('isGlobalSearchShortcut', () => {
  it('Ctrl+K 命中', () => {
    expect(isGlobalSearchShortcut(keyEvent({ key: 'k', ctrlKey: true }))).toBe(true);
  });

  it('Cmd+K（macOS）命中', () => {
    expect(isGlobalSearchShortcut(keyEvent({ key: 'k', metaKey: true }))).toBe(true);
  });

  // 修复前失败的用例：严格比较 'k' 时 'K' 不命中。
  it('大写 K（Shift+K / CapsLock / 部分输入法）同样命中', () => {
    expect(isGlobalSearchShortcut(keyEvent({ key: 'K', ctrlKey: true }))).toBe(true);
    expect(isGlobalSearchShortcut(keyEvent({ key: 'K', ctrlKey: true, shiftKey: true }))).toBe(
      true
    );
    expect(isGlobalSearchShortcut(keyEvent({ key: 'K', metaKey: true }))).toBe(true);
  });

  it('没有修饰键的裸 k 不命中（避免在输入框里打字就弹窗）', () => {
    expect(isGlobalSearchShortcut(keyEvent({ key: 'k' }))).toBe(false);
    expect(isGlobalSearchShortcut(keyEvent({ key: 'K' }))).toBe(false);
  });

  it('Ctrl+Alt+K 不命中（AltGr 组合 / 被其他功能占用）', () => {
    expect(isGlobalSearchShortcut(keyEvent({ key: 'k', ctrlKey: true, altKey: true }))).toBe(
      false
    );
  });

  it('Ctrl+其他键不命中', () => {
    expect(isGlobalSearchShortcut(keyEvent({ key: 'j', ctrlKey: true }))).toBe(false);
    expect(isGlobalSearchShortcut(keyEvent({ key: 'b', ctrlKey: true }))).toBe(false);
  });

  it('输入法合成态不命中（中文输入过程中不抢键）', () => {
    expect(
      isGlobalSearchShortcut(keyEvent({ key: 'k', ctrlKey: true, isComposing: true }))
    ).toBe(false);
  });
});
