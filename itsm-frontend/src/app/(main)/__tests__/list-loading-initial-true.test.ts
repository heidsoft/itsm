/**
 * D56 回归（B 类扩展）：手写列表页的加载标志必须初值为 true
 *
 * 背景：`useState(false)` 的单一 loading 标志会让首帧（mount effect 尚未执行）
 * 渲染出「暂无数据 / 共 0 条 / 统计卡全 0」的假空态，再被真实数据覆盖。
 *
 * 本守卫锁定三条不变量：
 * 1. 列表加载标志初值必须是 true（不能是 false）；
 * 2. 统计卡区块必须被 `!listLoading`（或等价）门控，不能在加载期渲染 0 值；
 * 3. 空态（Empty / Alert / 三元表达式）不能在数据到达前被判定为「空」。
 *
 * 采用静态源码断言而非渲染断言：本仓库多数路由页依赖 Next.js App Router、
 * 认证 store 与后端 API，渲染级测试需要大量 mock 且脆弱；静态守卫直接在
 * 源码层面锁定不变量，回退时立即失败。
 */

import fs from 'node:fs';
import path from 'node:path';

const SRC = path.join(process.cwd(), 'src');

type ListFlagCase = {
  /** 相对 src 的文件路径 */
  file: string;
  /** 列表加载标志的声明，必须包含初值 true */
  declaration: string;
  /** 禁止出现的旧写法（单标志 useState(false)） */
  forbidden: string[];
};

type GatedCase = {
  file: string;
  /** 统计卡门控表达式，必须出现在源码中 */
  gate: string;
};

const listFlagCases: ListFlagCase[] = [
  // ── 第一批（D56 B 类首批参考实现）──────────────────────────────
  {
    file: 'app/(main)/admin/users/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/workflow/automation/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },

  // ── 第二批：含统计卡门控的页面 ─────────────────────────────────
  {
    file: 'app/(main)/admin/menus/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/marketplace/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/capabilities/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/sla-templates/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/roles/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/tenants/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/groups/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/process-routing/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/workflows/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/sla-monitor/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/workflow/sla/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/workflow/bottlenecks/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/workflow/dashboard/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/department-processes/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/service-catalogs/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/escalation-rules/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/sla-definitions/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/msp/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },

  // ── 第二批：无统计卡、仅需初值为 true 的页面 ─────────────────────
  {
    file: 'app/(main)/admin/connectors/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/audit-logs/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/notifications/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/change-review/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/standard-changes/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/ai/approval/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/knowledge/reviews/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/skills/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/ticket-categories/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/config-inheritance/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },

  // ── 批次四：NOC / 报表 / 模板 ─────────────────────────────────────
  {
    file: 'app/(main)/noc/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/problems/trends/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/reports/cmdb-quality/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/reports/sla-performance/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/reports/service-catalog-usage/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/tickets/templates/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },

  // ── 批次五：列表页（customers / source-organizations / email-intake 等）──
  {
    file: 'app/(main)/customers/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/source-organizations/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },

  // ── 批次六：其余 admin / workflow / 应用类列表页 ──────────────────
  {
    file: 'app/(main)/support-contracts/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/msp/management/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/vector-store/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/approval-chains/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/tickets/assignment-rules/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/admin/tickets/automation-rules/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/on-call/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/problems/known-errors/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/problems/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/changes/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/changes/pirs/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/changes/[id]/pir/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/service-catalog/approvals/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/workflow/instances/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/workflow/versions/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/workflow/audit/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/improvements/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/tickets/cc/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/email-intake/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/workbench/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
  {
    file: 'app/(main)/profile/page.tsx',
    declaration: 'useState(true)',
    forbidden: ['useState(false)'],
  },
];

const gatedCases: GatedCase[] = [
  { file: 'app/(main)/admin/users/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/workflow/automation/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/admin/menus/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/admin/marketplace/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/admin/capabilities/page.tsx', gate: '!loading &&' },
  { file: 'app/(main)/admin/sla-templates/page.tsx', gate: '!loading &&' },
  { file: 'app/(main)/admin/roles/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/admin/tenants/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/admin/groups/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/admin/process-routing/page.tsx', gate: '!loading &&' },
  { file: 'app/(main)/admin/workflows/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/sla-monitor/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/workflow/sla/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/workflow/bottlenecks/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/admin/department-processes/page.tsx', gate: '!listLoading &&' },
  { file: 'app/(main)/admin/service-catalogs/page.tsx', gate: '!loading &&' },
  { file: 'app/(main)/admin/escalation-rules/page.tsx', gate: '!loading &&' },
  { file: 'app/(main)/admin/sla-definitions/page.tsx', gate: '!loading &&' },
  { file: 'app/(main)/admin/vector-store/page.tsx', gate: '!status && !listLoading' },
  { file: 'app/(main)/admin/approval-chains/page.tsx', gate: 'loading={listLoading}' },
  { file: 'app/(main)/admin/tickets/assignment-rules/page.tsx', gate: '!listLoading' },
  { file: 'app/(main)/admin/tickets/automation-rules/page.tsx', gate: '!listLoading' },
  { file: 'app/(main)/problems/known-errors/page.tsx', gate: 'loading={listLoading}' },
  { file: 'app/(main)/problems/page.tsx', gate: '!listLoading' },
  { file: 'app/(main)/changes/page.tsx', gate: '!listLoading' },
  { file: 'app/(main)/changes/pirs/page.tsx', gate: 'loading={listLoading}' },
  { file: 'app/(main)/workflow/instances/page.tsx', gate: "listLoading ? 'loading'" },
  { file: 'app/(main)/workflow/versions/page.tsx', gate: "listLoading ? 'loading'" },
  { file: 'app/(main)/improvements/page.tsx', gate: '!listLoading ?' },
  { file: 'app/(main)/email-intake/page.tsx', gate: 'loading={listLoading}' },
  { file: 'app/(main)/workbench/page.tsx', gate: '!listLoading' },
  { file: 'app/(main)/profile/page.tsx', gate: 'stats && (' },
];

describe('B 类列表页首帧加载态（D56 回归）', () => {
  describe('列表加载标志初值必须为 true', () => {
    it.each(listFlagCases)('$file', ({ file, declaration }) => {
      const source = fs.readFileSync(path.join(SRC, file), 'utf8');
      expect(source).toContain(declaration);
    });
  });

  describe('统计卡必须被加载态门控', () => {
    it.each(gatedCases)('$file', ({ file, gate }) => {
      const source = fs.readFileSync(path.join(SRC, file), 'utf8');
      expect(source).toContain(gate);
    });
  });

  describe('空态不得在数据到达前被判定为空', () => {
    it('admin/connectors 的连接器空态必须等加载完成', () => {
      const source = fs.readFileSync(
        path.join(SRC, 'app/(main)/admin/connectors/page.tsx'),
        'utf8',
      );
      // 原来 `market.length === 0 ? <Empty/> : <Table/>` 在首帧直接渲染空态，
      // Table 分支不渲染连 loading 都看不到；现在加载期渲染骨架表格。
      expect(source).toContain('listLoading ? (');
      expect(source).toContain('<Empty description={t(\'connectors.marketEmpty\')} />');
    });

    it('knowledge/reviews 的文章空态必须等加载完成', () => {
      const source = fs.readFileSync(
        path.join(SRC, 'app/(main)/knowledge/reviews/page.tsx'),
        'utf8',
      );
      expect(source).toContain('articles.length > 0 || listLoading');
    });

    it('workflow/sla 的违规空态必须等加载完成', () => {
      const source = fs.readFileSync(
        path.join(SRC, 'app/(main)/workflow/sla/page.tsx'),
        'utf8',
      );
      // 「暂无SLA违规」是 success Alert，首帧误报会把「还没拉到」说成「没有违规」。
      expect(source).toContain('listLoading ? (');
      expect(source).toContain('<Skeleton active paragraph={{ rows: 3 }} />');
    });
  });

  describe('提交动作不得与列表加载共用同一标志', () => {
    const actionSplitCases = [
      {
        file: 'app/(main)/admin/roles/page.tsx',
        tokens: ['useState(true)', 'useState(false)', 'setActionLoading'],
      },
      {
        file: 'app/(main)/admin/tenants/page.tsx',
        tokens: ['useState(true)', 'useState(false)', 'setActionLoading'],
      },
      {
        file: 'app/(main)/admin/groups/page.tsx',
        tokens: ['useState(true)', 'useState(false)', 'setActionLoading'],
      },
      {
        file: 'app/(main)/admin/workflows/page.tsx',
        tokens: ['useState(true)', 'useState(false)', 'setActionLoading'],
      },
      {
        file: 'app/(main)/admin/menus/page.tsx',
        tokens: ['useState(true)', 'useState(false)', 'setActionLoading'],
      },
    ];

    it.each(actionSplitCases)('$file 区分列表加载与提交动作', ({ file, tokens }) => {
      const source = fs.readFileSync(path.join(SRC, file), 'utf8');
      for (const token of tokens) {
        expect(source).toContain(token);
      }
    });
  });

  describe('表单页提交态必须与列表加载标志分离', () => {
    // 表单页首帧是空表单，没有「假空态」问题；但提交动作标志必须与数据加载标志
    // 区分命名，避免后续维护时误把初值改成 true 导致提交按钮常驻转圈。
    const actionOnlyCases = [
      'app/(main)/applications/page.tsx',
      'app/(main)/admin/departments/page.tsx',
      'app/(main)/projects/page.tsx',
      'app/(main)/service-catalog/request/[id]/page.tsx',
      'app/(main)/service-catalog/edit/[id]/page.tsx',
      'app/(main)/knowledge/articles/new/page.tsx',
      'app/(main)/knowledge/articles/[id]/edit/page.tsx',
      'app/(main)/improvements/new/page.tsx',
      'app/(main)/problems/new/page.tsx',
      'app/(main)/problems/[id]/edit/page.tsx',
      'app/(main)/changes/new/page.tsx',
      'app/(main)/changes/[id]/edit/page.tsx',
      'app/(main)/incidents/create/page.tsx',
      'app/(main)/incidents/[id]/edit/page.tsx',
    ];

    it.each(actionOnlyCases)('$file 使用 actionLoading 且初值为 false', file => {
      const source = fs.readFileSync(path.join(SRC, file), 'utf8');
      expect(source).toContain('useState(false)');
      expect(source).toContain('setActionLoading');
      expect(source).toContain('actionLoading');
      // 旧的单标志写法不得残留。
      expect(source).not.toMatch(/const \[loading, setLoading\] = useState\(false\)/);
    });
  });

  describe('详情页不得在数据到达前渲染错态结论', () => {
    it('changes/[id]/pir 加载期不得渲染表单', () => {
      const source = fs.readFileSync(
        path.join(SRC, 'app/(main)/changes/[id]/pir/page.tsx'),
        'utf8',
      );
      // existingPIR 决定标题是「创建实施后审查」还是「编辑实施后审查」，
      // 首帧渲染表单必然先显示「创建」再翻转。
      expect(source).toContain('listLoading ? (');
      expect(source).toContain('<Spin');
    });

    it('profile 统计卡必须等 stats 到达才渲染', () => {
      const source = fs.readFileSync(path.join(SRC, 'app/(main)/profile/page.tsx'), 'utf8');
      // 4 个 Statistic 读 stats，stats 初始为 null，直接渲染会先闪一组 0。
      expect(source).toContain('stats && (');
      expect(source).not.toMatch(/value=\{stats\?\.totalTickets \|\| 0\}/);
    });

    it('profile 身份与详情区块必须等 profile 到达才渲染', () => {
      const source = fs.readFileSync(path.join(SRC, 'app/(main)/profile/page.tsx'), 'utf8');
      // 「用户 / user@example.com / 未设置」是兜底占位，资料未返回时会把
      // 「尚未加载」呈现为「账号没填资料」，必须由 listLoading 门控。
      expect(source).toContain('listLoading ? (');
      expect(source).toContain('<Skeleton');
      // 兜底占位只能在 listLoading 为 false 的分支里出现。
      const gatedIndex = source.indexOf('listLoading ? (');
      expect(gatedIndex).toBeGreaterThan(-1);
      expect(source.indexOf("departmentLabel(profile?.department) || 'Not set'")).toBeGreaterThan(gatedIndex);
      expect(source.indexOf('profile?.name || profile?.username || "Users"')).toBeGreaterThan(
        gatedIndex,
      );
      // 头像首字母兜底（'U'）同样是占位：未加载时不能渲染成真实头像。
      expect(source).toContain('<Skeleton.Avatar active');
      const avatarGatedIndex = source.indexOf('<Skeleton.Avatar active');
      const userInitialUsage = source.indexOf('{userInitial}');
      expect(avatarGatedIndex).toBeGreaterThan(-1);
      expect(userInitialUsage).toBeGreaterThan(avatarGatedIndex);
      // 角色标签兜底成「用户」同理：不能渲染成「这个账号就是普通用户」。
      const roleTagIndex = source.indexOf(': "Users"}');
      expect(roleTagIndex).toBeGreaterThan(gatedIndex);
    });
  });
});
