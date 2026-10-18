/**
 * 附件响应类型契约守卫（后端 UPGRADE §1.43 / B7.2 的前端镜像）。
 *
 * 运行时无法内省 interface 要素，改用 TS 条件类型在编译期完成断言：
 * 若有人把 `filePath`（服务端磁盘路径泄漏面）加回 Attachment，
 * 条件类型翻转导致 const 初始化类型不匹配，tsc 报错拦截。
 */
import type { Attachment } from '../types';

describe('Attachment 类型契约', () => {
  it('不得包含 filePath（服务端磁盘路径不进 API 契约）', () => {
    const probe: 'filePath' extends keyof Attachment ? { leaked: true } : { leaked: false } = {
      leaked: false,
    };
    expect(probe.leaked).toBe(false);
  });

  it('保留下载契约键 fileUrl', () => {
    const probe: 'fileUrl' extends keyof Attachment ? { present: true } : { present: false } = {
      present: true,
    };
    expect(probe.present).toBe(true);
  });
});
