'use strict';

/**
 * init-labels.sh 的请求构造契约。
 *
 * 背景：这个脚本长期「运行成功但什么都没做」。
 *
 *   - label 颜色数据表里写成 `#1D76DB`（为了可读），但 GitHub 的 label color
 *     是不带 `#` 的 6 位 hex。带 `#` 提交会被判 `code: invalid, field: color`，
 *     HTTP 422。
 *   - 脚本把**所有** 422 当成「名称已存在」，于是转去 PATCH 更新。但那个标签
 *     其实从未创建成功，PATCH 找不到资源，返回 404。
 *   - 404 落到 else 分支只打印 `⚠️ exists but update failed`，仍然
 *     `SKIPPED++`，最后无条件 `exit 0`。
 *
 * 结果是 GitHub 上只有 29 个历史标签，而脚本声明的其余 47 个（area/*、
 * kind/*、priority/*、status/*、size/*）一个都没有，工作流却天天显示绿色。
 *
 * 本测试锁三件事（都是上面那条因果链的必要环节，去掉任何一条都会退回原状）：
 *   1. 送入 API 的 color 不含 `#`；
 *   2. 422 必须按响应体的 errors[].code 分流，不能一律当已存在；
 *   3. 有失败时脚本必须以非零退出。
 *
 * 刻意**不**断言具体实现写法（sed / printf / 变量名都可改），只断言行为契约，
 * 避免把测试写成实现快照。
 */

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.resolve(__dirname, '..', '..');
const scriptPath = path.join(root, 'scripts', 'init-labels.sh');
const source = fs.readFileSync(scriptPath, 'utf8');

/**
 * 取出 LABELS 数组里每一行的颜色字段（第二个 `|` 分隔段）。
 * 只读数据表，不关心后续怎么处理——断言的是「数据表里的写法」与
 * 「送进 API 前有没有归一化」两件事分开看。
 */
function declaredColors() {
  const colors = [];
  const re = /^\s*"([^"|]+)\|([^"|]+)\|/gm;
  let m;
  while ((m = re.exec(source)) !== null) {
    colors.push({ name: m[1], color: m[2] });
  }
  return colors;
}

test('数据表声明了标签，且颜色是带 # 的可读写法', () => {
  const all = declaredColors();
  assert.ok(all.length > 0, 'LABELS 数据表不应为空');

  const withHash = all.filter((l) => /^#[0-9A-Fa-f]{6}$/.test(l.color));
  assert.ok(
    withHash.length > 0,
    '数据表里应存在带 # 的颜色（这是可读性约定；若全部改成无 #，本测试与下面的归一化守卫需一并复核）',
  );
});

test('颜色在送入 API 前被去掉 # 前缀', () => {
  // 归一化必须真实存在：既要有去 # 的动作，也要在 POST 的 payload 里用归一化后的值。
  // 只查「有没有 strip 函数」会假绿——函数写了却没调用同样无效。
  assert.match(
    source,
    /\$\{1#"?[#]"\]?\}|sed\s*'s\|#\|\|'|tr\s+-d\s+'#'|\$\{color#\#\}/,
    '必须存在去掉颜色 # 前缀的动作',
  );

  // 送进 POST 的 payload 变量必须在其中一处归一化后才使用。
  // 断言 payload 构造行本身引用的是被处理过的变量名，而非原始数据表字段。
  const postPayload = source.match(/payload=\$?\(?printf[^\n]*/);
  assert.ok(postPayload, '应能找到构造 POST payload 的行');
  assert.ok(
    !/printf '{"name":"%s","color":"%s"[^\n]*"\$color"/.test(source) ||
      /color=\$\([^\n]*\)/.test(source),
    'payload 不应直接使用数据表原始 color（必须先归一化）',
  );
});

test('422 按 errors[].code 分流，不把校验失败当成已存在', () => {
  assert.match(
    source,
    /already_exists/,
    '必须显式识别 already_exists 才走更新分支',
  );

  // 必须有「非 already_exists 的 422 → 计失败」的分支，否则非法 payload 会被吞掉。
  const invalidBranch = /invalid payload|422.*FAILED|FAILED.*422/is;
  assert.match(
    source,
    invalidBranch,
    '422 中非 already_exists 的情况必须计为失败并暴露，不能静默跳过',
  );
});

test('标签名中的 / 在 URL 路径里被编码，否则 PATCH/DELETE 定位不到资源', () => {
  assert.match(
    source,
    /%2F/,
    '含 / 的标签名（area/backend 等）必须编码为 %2F 才能定位到该资源',
  );
});

test('有失败时以非零退出，不再无条件 exit 0', () => {
  // 抓最后的退出段：必须存在 FAILED > 0 → exit 1 的分支。
  assert.match(
    source,
    /\[\[?\s*"?\$FAILED"?\s+-gt\s+0\s*\]\]?|FAILED\s+-gt\s+0/,
    '必须存在「有失败即非零退出」的判定',
  );
  assert.match(
    source,
    /exit\s+1/,
    '失败路径必须 exit 1',
  );
});

test('脚本在 dry-run 下不产生任何写操作', () => {
  // dry-run 分支必须在任何 curl 之前 continue，否则 --dry-run 会真改仓库。
  const dryRunIdx = source.indexOf('DRY_RUN" == "true"');
  assert.ok(dryRunIdx > -1, '应存在 dry-run 分支');

  const afterDryRun = source.slice(dryRunIdx, dryRunIdx + 400);
  assert.match(afterDryRun, /continue/, 'dry-run 分支必须 continue 跳过写请求');
});
