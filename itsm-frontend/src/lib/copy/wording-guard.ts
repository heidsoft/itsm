/**
 * 用户可见文案的装饰性措辞检查。
 *
 * 产品语言治理的目标是「说清这页能干什么」，而不是给普通业务功能套一层宏大叙事。
 * 曾经的典型问题是页面头部写成「重大事件作战室：实时跟踪企业级重大事件处置进展」
 * ——「作战室」既不描述功能（跟踪进展），也不描述入口位置，只是给 NOC 页镀了一层
 * 军事化氛围；同类还有「可持续的图谱治理闭环」「逾期的待办会持续提醒」（后者更是
 * 能力夸大：逾期并不会主动提醒任何人，只在列表里标出来）。
 *
 * 这里给出**可执行**的判定函数，让测试能对真实文案跑断言，而不是在文件里搜字面量
 * （那种写法只要删掉一行注释就会假绿）。纯函数便于反向验证：把某个词加进返回
 * 命中列表的日志，对应断言即红。
 */

/** 仅用于营销包装、不描述任何可操作行为的词。 */
export const DECORATIVE_TERMS = [
  '作战室', '闭环', '赋能', '中枢', '智慧', '全域', '生态',
  '数字员工', '全链路', '超级', '一站式', '赋能客户',
] as const;

/** 能力夸大：宣称系统会主动做某件它在代码里并不做的事。 */
export const OVERCLAIM_PATTERNS = [
  /持续提醒/,
  /自动提醒你没/,
  /实时预警一切/,
] as const;

export type WordingHit = { text: string; term: string };

/**
 * 返回文案里命中的装饰性词与夸大表达。空数组表示通过。
 *
 * 只扫「用户可见」的位置（页面标题/描述/占位符/表单标签），不扫代码注释：
 * 注释里写「沉淀闭环」解释设计意图是合理的，把它当违规会让团队被迫写得更含糊。
 */
export function findDecorativeWording(texts: readonly string[]): WordingHit[] {
  const hits: WordingHit[] = [];
  for (const text of texts) {
    for (const term of DECORATIVE_TERMS) {
      if (text.includes(term)) hits.push({ text, term });
    }
    for (const re of OVERCLAIM_PATTERNS) {
      if (re.test(text)) hits.push({ text, term: re.source });
    }
  }
  return hits;
}
