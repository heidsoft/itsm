/**
 * 全局搜索快捷键判定 —— 单点定义，避免多处注册各自写一份比较逻辑。
 *
 * 缺陷 O6-B：顶栏标注 `Ctrl K`，但 Ctrl+K / Meta+K 实测不唤起搜索。代码级成因有三处：
 *   1. `GlobalShortcutProvider` 也注册了 `ctrl+k`，handler 只写一个无消费者的状态，
 *      却同样 `preventDefault`，与 Header 的注册竞争；
 *   2. `GlobalSearch` 自己也挂了一个 window keydown，只 `preventDefault` 不 open；
 *   3. 三处都用 `e.key === 'k'`，Shift+K / CapsLock 下 `e.key` 是 `'K'`，直接不命中。
 *
 * 本模块把「是不是这个组合键」收成唯一实现；注册点只留 Header 一处。
 */

/**
 * 判断键盘事件是否为「打开全局搜索」的组合键（Ctrl+K / Meta+K，大小写不敏感）。
 *
 * 注意：不判断 `repeat`，按住不放会重复设置 `open=true`（幂等，无副作用）；
 * 但要跳过组合键作用期间的输入法合成态，避免中文输入过程中误触发。
 */
export function isGlobalSearchShortcut(
  event: Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'metaKey'> & {
    altKey?: boolean;
    isComposing?: boolean;
  }
): boolean {
  if (event.isComposing) return false;
  if (event.altKey) return false;
  if (!event.ctrlKey && !event.metaKey) return false;
  return event.key.toLowerCase() === 'k';
}
