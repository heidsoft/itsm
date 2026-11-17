/**
 * CMDB 注册页「缺失条件」的展示口径守卫。
 *
 * 背景：后端 /cmdb/capabilities 下发的 missingRequirements 是稳定机器名
 * （discoveryAdapter / cloudAccountWithDiscoveryAdapter / ITSM_CMDB_DISCOVERY_READY …）。
 * 早前页面直接 `join('、')` 拼进中文句子，结果是「缺失条件：tenantSecretResolver、
 * discoveryWorker」——运维既读不懂，也看不出该去改什么。
 *
 * 本轮域内能力面开始下发部署申报变量名（ITSM_CMDB_DISCOVERY_READY），如果不翻译，
 * 云发现的门槛信息会以变量名形式暴露给使用者。本守卫锁两件事：
 *   1) 页面用翻译函数渲染，而不是裸 join；
 *   2) 每个已知缺失键都有中文说明，且未知键必须原样透出，不能被静默吞掉。
 */

import fs from 'node:fs';
import path from 'node:path';

const SRC = path.join(process.cwd(), 'src');
const PAGE = path.join(SRC, 'app/(main)/cmdb/registry/page.tsx');

describe('cmdb registry missingRequirements wording', () => {
  const source = fs.readFileSync(PAGE, 'utf8');

  it('缺失条件必须经翻译函数渲染，不得裸 join 机器名', () => {
    expect(source).toContain('describeMissingRequirements');
    // 裸 join 会把 cloudAccountWithDiscoveryAdapter 直接摆到中文句子里
    expect(source).not.toContain('missingRequirements.join(');
  });

  it('已知缺失键必须有中文说明', () => {
    // 后端 GetDiscoveryCapability / GetDiscoveryHealth 可能下发的全部键
    const knownKeys = [
      'discoveryAdapter',
      'tenantSecretResolver',
      'discoveryWorker',
      'cloudAccountWithDiscoveryAdapter',
      'ITSM_CMDB_DISCOVERY_READY',
    ];
    for (const key of knownKeys) {
      expect(source).toContain(key);
    }
    // 中文说明不能是空的占位
    expect(source).toContain('未安装任何厂商发现适配器');
    expect(source).toContain('部署未申报');
  });

  it('未声明的缺失键必须原样透出，不得静默吞掉', () => {
    // 新增判定条件时若忘记加文案，至少要让人看到原始键，而不是少一条原因
    expect(source).toContain('MISSING_REQUIREMENT_LABEL[key] ?? key');
  });
});
