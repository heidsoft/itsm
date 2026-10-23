/**
 * MCP resources — read-only context the model can pull without spending a tool
 * call on discovery.
 *
 * Two kinds:
 *   · Live capability map (`itsm://capabilities`), derived from the command
 *     registry: what this instance can do, and through which command.
 *   · Product documentation (`itsm://docs/*`), read from the repository. This is
 *     the "improve the product" half of the connector: an agent that can also
 *     read ROADMAP/CHANGELOG/API reference can propose a change that fits the
 *     existing contract instead of inventing a parallel one.
 *
 * Documentation files are optional at runtime (an installed CLI may sit outside
 * the repo). A missing file is reported as a normal resource error, never a
 * crash.
 */
import { readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { registry } from '../agent/registry.js';
import { resolveRuntimeConfig } from '../lib/runtime.js';
/** Cap on a single document read: a 400KB CHANGELOG is not context, it is noise. */
const MAX_DOC_BYTES = 64 * 1024;
const DOC_RESOURCES = [
    {
        uri: 'itsm://docs/roadmap',
        name: '产品路线图',
        description: 'ROADMAP.md：已落地能力与下一阶段收敛项。规划改动前先读，避免与路线图重复或冲突。',
        file: 'ROADMAP.md',
    },
    {
        uri: 'itsm://docs/changelog',
        name: '变更记录',
        description: 'CHANGELOG.md：按版本记录的能力/修复/破坏性变更，判断某行为是“设计如此”还是“回归”的依据。',
        file: 'CHANGELOG.md',
    },
    {
        uri: 'itsm://docs/api-reference',
        name: 'API 参考',
        description: 'docs/api-reference.md：后端 HTTP 契约（路径、方法、请求/响应、错误码）。',
        file: 'docs/api-reference.md',
    },
    {
        uri: 'itsm://docs/architecture',
        name: '架构总览',
        description: 'docs/architecture/overview.md：系统边界、分层规则、扩展点。',
        file: 'docs/architecture/overview.md',
    },
];
/**
 * Locate the repository root without assuming an install layout: walk up from
 * this module until a directory that looks like the repo is found.
 */
function findRepoRoot() {
    const override = (process.env['ITSM_REPO_ROOT'] || '').trim();
    if (override)
        return override;
    let dir = import.meta.dirname;
    for (let i = 0; i < 6; i++) {
        try {
            if (statSync(path.join(dir, 'ROADMAP.md')).isFile() && statSync(path.join(dir, 'itsm-cli')).isDirectory()) {
                return dir;
            }
        }
        catch {
            // keep walking
        }
        const parent = path.dirname(dir);
        if (parent === dir)
            break;
        dir = parent;
    }
    return undefined;
}
let cachedRoot = null;
function repoRoot() {
    if (cachedRoot === null)
        cachedRoot = findRepoRoot();
    return cachedRoot;
}
export function listResources() {
    const docs = DOC_RESOURCES.map((d) => ({
        uri: d.uri,
        name: d.name,
        description: d.description,
        mimeType: 'text/markdown',
    }));
    return [
        {
            uri: 'itsm://capabilities',
            name: '实例能力清单',
            description: '当前实例可执行的域、命令数与身份/端点信息（实时由命令注册表派生）。',
            mimeType: 'application/json',
        },
        ...docs,
    ];
}
function capabilities() {
    const specs = registry();
    const byDomain = new Map();
    for (const spec of specs) {
        const domain = spec.name.split('.')[0];
        const entry = byDomain.get(domain) ?? { commands: 0, reads: 0, writes: 0, highRisk: 0 };
        entry.commands += 1;
        if (spec.risk === 'normal')
            entry.reads += 1;
        else if (spec.risk === 'write')
            entry.writes += 1;
        else
            entry.highRisk += 1;
        byDomain.set(domain, entry);
    }
    const runtime = resolveRuntimeConfig();
    const payload = {
        surface: 'itsm-mcp',
        commands: specs.length,
        domains: [...byDomain.entries()]
            .map(([domain, stats]) => ({ domain, ...stats }))
            .sort((a, b) => b.commands - a.commands),
        runtime: {
            baseURL: runtime.baseURL,
            authSource: runtime.authSource,
            repoRoot: repoRoot() ?? null,
        },
        howToCall: {
            curated: 'itsm_ticket_list / itsm_cmdb_impact 等精选工具直接调用',
            discovery: 'itsm_discover 检索 → itsm_describe 取契约 → itsm_call 执行',
            risk: '高风险写操作（删除/终止/回滚/批量）需 confirm=true',
        },
    };
    return { uri: 'itsm://capabilities', mimeType: 'application/json', text: JSON.stringify(payload, null, 2) };
}
export function readResource(uri) {
    if (uri === 'itsm://capabilities')
        return capabilities();
    const doc = DOC_RESOURCES.find((d) => d.uri === uri);
    if (!doc)
        throw new Error(`未知资源 '${uri}'。可用资源：${listResources().map((r) => r.uri).join(', ')}`);
    const root = repoRoot();
    if (!root) {
        throw new Error(`未找到仓库根目录，无法读取 ${doc.file}。设置 ITSM_REPO_ROOT 指向 ITSM 仓库（包含 ROADMAP.md 与 itsm-cli/）。`);
    }
    const abs = path.join(root, doc.file);
    let raw;
    try {
        raw = readFileSync(abs, 'utf8');
    }
    catch (err) {
        throw new Error(`无法读取 ${doc.file}：${err instanceof Error ? err.message : String(err)}`);
    }
    if (Buffer.byteLength(raw, 'utf8') <= MAX_DOC_BYTES) {
        return { uri: doc.uri, mimeType: 'text/markdown', text: raw };
    }
    // Byte-slicing can land mid-codepoint; drop the resulting replacement chars
    // so a truncated Chinese document stays readable instead of ending in garbage.
    const truncated = Buffer.from(raw, 'utf8').subarray(0, MAX_DOC_BYTES).toString('utf8').replace(/\uFFFD+$/, '');
    return {
        uri: doc.uri,
        mimeType: 'text/markdown',
        text: `${truncated}\n\n> [已截断] ${doc.file} 超出 ${Math.round(MAX_DOC_BYTES / 1024)}KB，仅返回前 ${Math.round(MAX_DOC_BYTES / 1024)}KB。完整内容见仓库文件 ${doc.file}。`,
    };
}
