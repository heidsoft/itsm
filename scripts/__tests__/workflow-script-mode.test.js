'use strict';

/**
 * 直接执行面的执行位契约。
 *
 * `label-sync.yml` 曾写 `run: ./scripts/init-labels.sh`，而该脚本在版本库里是
 * 100644（无执行位）。GitHub Actions 直接执行时不会回落到 `bash`，结果是
 * `line 1: ./scripts/init-labels.sh: Permission denied` / exit 126 —— 这条
 * 自动化从 2026-11-14 落地起四次触发全部失败，label 同步一直靠人工。
 *
 * 失败是静默的：本地 macOS 工作树里文件是 `-rwx------`（chmod 过、但从未
 * 入库），所以 `ls -l` 看起来"没问题"；只有干净 checkout 才暴露出 644。
 * 因此本测试不读工作区权限位（Windows/macOS checkout 都不可靠），
 * 而是查 **git index 里的 mode**，那才是 clone 后真正生效的东西。
 *
 * 断言语义（不是文本匹配）：
 *   1. 任何工作流里以 `./scripts/x` 或 `scripts/x` 开头、且未显式调用解释器
 *      （bash/sh/node/python）的命令，其目标脚本在索引中的 mode 必须是 100755；
 *   2. 用 `bash scripts/x` 形式调用的不受此限（解释器不要求执行位）；
 *   3. 扫描面覆盖全部 workflow 文件，新增工作流自动纳入，不需要改本测试。
 */

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { execFileSync } = require('node:child_process');

const root = path.resolve(__dirname, '..', '..');
const workflowsDir = path.join(root, '.github', 'workflows');

/**
 * 从工作流 YAML 里抽出所有"直接执行脚本"的位置。
 *
 * 只认命令起始位置的脚本调用：`run: ./scripts/a.sh`、`run: scripts/a.sh`、
 * 以及 run 块（`run: |`）内独立成行的同形写法。刻意不匹配
 * `bash scripts/a.sh`、`node scripts/a.js`（解释器显式调用不需要执行位），
 * 也不匹配出现在参数中间或注释里的路径（`# ./scripts/a.sh` 只是说明文字）。
 */
function collectDirectExecutions() {
  const entries = [];
  const files = fs
    .readdirSync(workflowsDir)
    .filter((f) => f.endsWith('.yml') || f.endsWith('.yaml'))
    .sort();

  for (const file of files) {
    const lines = fs.readFileSync(path.join(workflowsDir, file), 'utf8').split('\n');
    for (let i = 0; i < lines.length; i += 1) {
      const line = lines[i];

      // 注释行整行跳过：文档里引用脚本路径很常见，不是执行面。
      if (/^\s*#/.test(line)) continue;

      // 去掉 `run:` 前缀与可选的 `- `（step 列表项），剩下命令本体。
      const m = line.match(/^\s*(?:-\s*)?run:\s*(.*)$/);
      if (!m) continue;

      const command = m[1].trim();

      // `run: |` / `run: >` 之后的块内容逐行扫描。
      if (command === '|' || command === '>' || command === '|-' || command === '>-') {
        // 块内缩进必须比 `run:` 更深，否则属于下一个 key。
        const baseIndent = line.search(/[a-z]/i);
        for (let j = i + 1; j < lines.length; j += 1) {
          const body = lines[j];
          if (body.trim() === '') continue;
          const indent = body.search(/\S/);
          if (indent <= baseIndent) break;
          const candidate = body.trim();
          if (/^\s*#/.test(candidate)) continue;
          const direct = matchDirectExec(candidate);
          if (direct) entries.push({ file, line: j + 1, script: direct });
        }
        continue;
      }

      const direct = matchDirectExec(command);
      if (direct) entries.push({ file, line: i + 1, script: direct });
    }
  }

  return entries;
}

/**
 * 判断一条命令是否"直接执行脚本"。返回脚本仓库相对路径，否则 null。
 *
 * 只接受命令开头就是路径的形式。带解释器前缀（bash/sh/node/python/deno/uv）
 * 的调用返回 null —— 那些不依赖执行位，把它们算进来会制造无关红。
 */
function matchDirectExec(command) {
  // 取命令的第一个 token；忽略 `sudo` 之类前缀（当前仓库没有，但别假设）。
  const first = command.split(/\s+/)[0];
  if (!first) return null;
  if (!/^(?:\.\/)?scripts\//.test(first)) return null;
  return first.replace(/^\.\//, '');
}

/** 读 git index 里的 mode（clone 后真正生效的那个），不是工作区权限位。 */
function indexMode(relPath) {
  const out = execFileSync('git', ['ls-files', '-s', '--', relPath], {
    cwd: root,
    encoding: 'utf8',
  }).trim();
  if (!out) return null;
  // 格式：<mode> <object> <stage>\t<path>
  return out.split(/\s+/)[0];
}

test('工作流里直接执行的脚本在 git index 中必须带执行位', () => {
  const executions = collectDirectExecutions();

  // 扫描面自检：找不到任何直接执行点说明解析失效了（例如 YAML 形态变了），
  // 那种情况下测试会假绿。这里用 >= 1 而不是精确值，避免新增工作流就要改测试。
  assert.ok(
    executions.length >= 1,
    '未扫到任何直接执行的脚本调用：解析可能已失效（会假绿），请检查 run: 语法'
  );

  const offenders = [];
  for (const { file, line, script } of executions) {
    if (!fs.existsSync(path.join(root, script))) {
      offenders.push(`${file}:${line} 指向不存在的脚本 ${script}`);
      continue;
    }
    const mode = indexMode(script);
    if (mode === null) {
      offenders.push(`${file}:${line} 执行的 ${script} 未被 git 跟踪`);
      continue;
    }
    if (mode !== '100755') {
      offenders.push(`${file}:${line} 直接执行 ${script}，但索引 mode=${mode}（应为 100755）`);
    }
  }

  assert.deepEqual(
    offenders,
    [],
    `工作流直接执行无执行位的脚本会在干净 checkout 上 exit 126：\n${offenders.join('\n')}\n` +
      '修复：git update-index --chmod=+x <脚本>（仅 chmod 工作区不入库，换机器仍复发）'
  );
});

test('扫描器本身能识别解释器调用与直接执行的区别', () => {
  // 反向断言解析器语义：解释器前缀不算直接执行，裸路径算。
  // 否则上面那条测试可能因为"什么都没扫到"而永久绿。
  assert.equal(matchDirectExec('bash scripts/a.sh'), null);
  assert.equal(matchDirectExec('node scripts/__tests__/a.test.js'), null);
  assert.equal(matchDirectExec('./scripts/a.sh'), 'scripts/a.sh');
  assert.equal(matchDirectExec('scripts/a.sh --flag'), 'scripts/a.sh');
  assert.equal(matchDirectExec('echo hello'), null);
});
