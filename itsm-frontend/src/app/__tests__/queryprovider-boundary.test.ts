/**
 * QueryProvider 边界守卫（QA-001 / QA-004 回归锁）
 *
 * RecentVisitTracker 内部调用 useUserMenusQuery()（底层 useQuery），对挂载位置有两条硬约束：
 *
 *   QA-001（P1）：必须位于根布局 <QueryProvider> 子树内。放在 Provider 之外时，
 *     Next.js SSG 预渲染期无 QueryClient context，`--no-cache` 构建在
 *     `Generating static pages` 阶段抛 "No QueryClient set" 并整体失败（148 页中断）。
 *     增量缓存构建掩盖此问题，只有全新构建会暴露。
 *
 *   QA-004（P3）：必须只在认证后的 (main) 布局渲染。挂在全局根布局时，
 *     未登录页面（如 /login）也会拉 /auth/menus → 401 → http-client 401 拦截器
 *     无条件续签 → 后端无 refresh cookie → 400「refreshToken 不能为空」console error。
 *
 * 结论：唯一正确挂载点是 (main) 布局的 ready 态子树——它在根布局 QueryProvider 之内，
 * 且 checking/anonymous/unavailable 三态均提前 return，认证后才到达挂载点。
 *
 * 反向验证：把 <RecentVisitTracker /> 放回根布局，或从 (main) 布局删除，本套件必须转红。
 * 剥注释后扫描，避免把说明文字里的组件名算作真实挂载。
 */
import fs from 'node:fs';
import path from 'node:path';

const srcRoot = path.join(process.cwd(), 'src');
const rootLayout = path.join(srcRoot, 'app', 'layout.tsx');
const mainLayout = path.join(srcRoot, 'app', '(main)', 'layout.tsx');

/** 去掉 // 与 /* *\/ 注释，避免注释中的组件名造成假绿/假红 */
function stripComments(source: string): string {
  return source
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/(^|[^:])\/\/.*$/gm, '$1');
}

const trackerMount = /<RecentVisitTracker\s*\/?>/;

describe('QueryProvider 边界：RecentVisitTracker 挂载点', () => {
  const rootSrc = stripComments(fs.readFileSync(rootLayout, 'utf8'));
  const mainSrc = stripComments(fs.readFileSync(mainLayout, 'utf8'));

  it('QA-001：根布局不得挂载 RecentVisitTracker（在 QueryProvider 之外会导致 SSG 崩溃）', () => {
    expect(trackerMount.test(rootSrc)).toBe(false);
  });

  it('QA-004：(main) 认证布局必须挂载 RecentVisitTracker', () => {
    expect(trackerMount.test(mainSrc)).toBe(true);
  });

  it('QA-004：tracker 挂载点必须位于 ready 态 return 之后（未认证三态提前 return）', () => {
    const mountIdx = mainSrc.search(trackerMount);
    const anonymousReturn = mainSrc.indexOf("sessionView === 'anonymous'");
    const unavailableReturn = mainSrc.indexOf("sessionView === 'unavailable'");
    const readyReturn = mainSrc.lastIndexOf('return (', mountIdx);

    expect(mountIdx).toBeGreaterThan(-1);
    // 挂载点必须在 anonymous/unavailable 早退分支之后，避免在未认证态拉菜单
    expect(mountIdx).toBeGreaterThan(anonymousReturn);
    expect(mountIdx).toBeGreaterThan(unavailableReturn);
    // 且位于其所属 return 语句内部（ready 态的 ThemedMainLayout 分支）
    expect(readyReturn).toBeGreaterThan(unavailableReturn);
  });

  it('QA-001：(main) 布局在根布局 QueryProvider 子树内——根布局仍保留 QueryProvider 包裹 children', () => {
    expect(rootSrc).toMatch(/<QueryProvider>[\s\S]*?<ErrorBoundary>\{children\}<\/ErrorBoundary>[\s\S]*?<\/QueryProvider>/);
  });

  /**
   * 反向验证（测试图可区分性）：把修复前的错误源码喂给同样的判据，必须判为违规。
   * 若两条路径得出同一结论，说明守卫是假绿。
   */
  it('反向验证：修复前源码（tracker 在根布局、不在 (main)）应被判违规', () => {
    const buggyRoot = stripComments(
      rootSrc.replace('<ThemeHtmlClassSync />', '<RecentVisitTracker />\n<ThemeHtmlClassSync />')
    );
    const buggyMain = stripComments(mainSrc.replace(trackerMount, ''));
    expect(trackerMount.test(buggyRoot)).toBe(true); // 错误挂载被检出
    expect(trackerMount.test(buggyMain)).toBe(false); // 缺失挂载被检出
  });
});
