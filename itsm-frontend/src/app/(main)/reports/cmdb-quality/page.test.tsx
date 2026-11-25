import { render, screen, waitFor } from '@/lib/test-utils';

import CMDBQualityReport from './page';

import { CMDBApi } from '@/lib/api/cmdb-api';

// 本页的核心契约：**所有指标都来自后端治理看板，页面不得自算**。
//
// 改造前这里用硬编码 9 个字段在前端算完整度，与后端 owner 覆盖率口径并存，
// 同一租户在两处读到两个数字。这些用例锁三件事：
//   1. 页面确实调用 getGovernanceReport（而不是拉 CI 列表自己算）；
//   2. assessable=0 时显示「不可评估」而不是 0% 完整；
//   3. 重复组/缺失属性真的呈现在页面上，不是只存进 state。

// 🔴 jest.config.js 开了 restoreMocks: true —— 每个 test 之前会把 spied 方法
// 还原到原始实现。所以 spy 必须在 beforeEach 里重建，不能提到模块顶层
// （提到顶层会被还原，真实 httpClient 被真的调起来 → fetch is not defined）。
let governanceSpy: jest.SpyInstance;
let listSpy: jest.SpyInstance;

const baseReport = {
  quality: {
    total: 10,
    active: 8,
    retired: 2,
    stale: 1,
    orphan: 3,
    incomplete: 2,
    completenessPct: 75,
    assessable: 8,
    attributeDefinedTypes: 1,
    missingByAttribute: [
      { ciTypeId: 1, attribute: 'vendor', missing: 2, applicable: 8 },
    ],
  },
  diff: { add: 1, noop: 2, retire_confirm: 0, duplicate: 0 },
  cloudResources: 3,
  cis: 10,
  duplicateGroups: [
    {
      key: 'cloud:aliyun:i-dup',
      reason: 'cloud_identity' as const,
      ciIds: [1, 2],
      ciNumbers: ['CI-1', 'CI-2'],
    },
  ],
  duplicateCiCount: 2,
  duplicateRatePct: 25,
};

describe('CMDB 质量报表页', () => {
  beforeEach(() => {
    // 用 spyOn 而不是 jest.mock 工厂：页面 import 的是同一个类对象，
    // 替换静态方法即可拦截；工厂在 restoreMocks 下行为更不可控。
    governanceSpy = jest.spyOn(CMDBApi, 'getGovernanceReport');
    // 若页面回退到「拉列表自算」，这条断言会暴露出来。
    listSpy = jest.spyOn(CMDBApi, 'getAllCIs');
  });

  it('只调用治理看板接口，不拉 CI 列表在前端自算', async () => {
    governanceSpy.mockResolvedValue(baseReport);
    // 反向验证用：让「拉列表」也成功返回，确保失败只可能来自 listSpy 断言，
    // 而不是「未 mock 导致 undefined 报错」这种偶然红。
    listSpy.mockResolvedValue([] as never);

    render(<CMDBQualityReport />);

    await waitFor(() => {
      expect(governanceSpy).toHaveBeenCalledTimes(1);
    });
    expect(listSpy).not.toHaveBeenCalled();
  });

  it('呈现后端下发的完整率与可评估基数', async () => {
    governanceSpy.mockResolvedValue(baseReport);

    render(<CMDBQualityReport />);

    // 完整率 75%
    expect(await screen.findByText('75')).toBeInTheDocument();
    // 分母说明必须出现（区分「完整率低」与「没配规则」）
    expect(await screen.findByText('可评估 8 项')).toBeInTheDocument();
    // 重复率 25%
    expect(await screen.findByText('25')).toBeInTheDocument();
    expect(await screen.findByText('2 项参与重复组')).toBeInTheDocument();
  });

  it('🔴 assessable=0 时显示不可评估而不是 0% 完整', async () => {
    governanceSpy.mockResolvedValue({
      ...baseReport,
      quality: {
        ...baseReport.quality,
        assessable: 0,
        incomplete: 0,
        completenessPct: 0,
        attributeDefinedTypes: 0,
        missingByAttribute: [],
      },
      duplicateGroups: [],
      duplicateCiCount: 0,
      duplicateRatePct: 0,
    });

    render(<CMDBQualityReport />);

    // 必须给出明确告知
    expect(
      await screen.findByText('尚未配置必填属性，完整率不可评估')
    ).toBeInTheDocument();
    // 数值位显示「—」而不是 0（后者会被读成「数据烂透了」）
    expect(await screen.findByText('—')).toBeInTheDocument();
    expect(screen.queryByText('整体完整度')).not.toBeInTheDocument();
    expect(screen.getByText('整体完整度（不可评估）')).toBeInTheDocument();
  });

  it('呈现重复组成员与判重依据', async () => {
    governanceSpy.mockResolvedValue(baseReport);

    render(<CMDBQualityReport />);

    expect(await screen.findByText('云身份相同')).toBeInTheDocument();
    expect(await screen.findByText('CI-1')).toBeInTheDocument();
    expect(await screen.findByText('CI-2')).toBeInTheDocument();
    expect(await screen.findByText('cloud:aliyun:i-dup')).toBeInTheDocument();
  });

  it('呈现最缺的必填属性（可行动产出）', async () => {
    governanceSpy.mockResolvedValue(baseReport);

    render(<CMDBQualityReport />);

    // recharts 在 jsdom 下不把 Y 轴类目名渲染成可查询文本节点，
    // 因此断言卡片标题与图例（"缺失数"/"适用数" 是 Bar 的 name），
    // 并确认没有落到「必填属性均已填写」的空态。
    expect(await screen.findByText('最缺的必填属性')).toBeInTheDocument();
    expect(screen.queryByText('必填属性均已填写')).not.toBeInTheDocument();
    expect(screen.queryByText('未配置必填属性定义')).not.toBeInTheDocument();
  });

  it('接口失败时给出可识别状态，不静默空白', async () => {
    governanceSpy.mockRejectedValue(new Error('boom'));

    render(<CMDBQualityReport />);

    expect(await screen.findByText('无法获取 CMDB 治理数据')).toBeInTheDocument();
  });
});
