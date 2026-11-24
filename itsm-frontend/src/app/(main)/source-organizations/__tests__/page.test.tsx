/**
 * 来源组织页 Modal 重复提交 / 状态锁 回归锁。
 *
 * ── 缺陷本体（已实证，不是推测） ────────────────────────────────
 * `<Modal onOk={async fn}>` 不会自动 loading。antd v6 的 `Modal.js:111`：
 *   const handleOk = e => { onOk?.(e); onClose?.(); };
 * 它调用 onOk 之后**立刻** onClose，从不等待返回的 Promise。
 * 自动 loading 只属于 `Modal.confirm` / `useModal`。
 *
 * 后果：请求挂起（真实网络延迟）期间弹窗仍然开着、确定按钮仍可点，
 * 用户连点两次 → 提交两次 → 建两条来源组织记录。
 * 实测：双防线全拆时 CALL COUNT = 2。
 *
 * ── 两道防线（各自独立有效，已分别实测） ──────────────────────────
 *   ① 入口同步守卫 `if (submitBusy) return;` —— 二次进入直接返回
 *   ② `confirmLoading={submitBusy}` —— antd Button 在 loading 时
 *      `e.preventDefault()` 丢弃点击（button.js:185），且按钮转圈给出反馈
 * 只留任一一道都能把 2 次压回 1 次；两道都拆就回到 2 次。
 *
 * ── 测试为什么必须用「挂起 Promise」 ──────────────────────────────
 * 用 mockResolvedValue 立即成功时，第一次保存就把弹窗关掉了，
 * 第二次根本点不到按钮 → 永远只有 1 次调用 → 假绿。
 * 必须让请求一直挂起，弹窗才保持打开，竞态窗口才存在。
 */

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { emailIntakeService } from '@/lib/services/emailIntakeService';
import SourcesPage from '../page';

jest.mock('@/lib/services/emailIntakeService', () => ({
  emailIntakeService: {
    sourceOrganizations: jest.fn(),
    createSourceOrganization: jest.fn(),
    updateSourceOrganization: jest.fn(),
    disableSourceOrganization: jest.fn(),
  },
}));

const svc = emailIntakeService as jest.Mocked<typeof emailIntakeService>;

// antd Table + Modal 在 jsdom 下的挂载远超默认 5s 预算。
jest.setTimeout(40000);

function renderPage() {
  return render(
    // 必须带 zh_CN：真实装配由 AntdProvider 注入中文 locale，
    // 裸用 <App> 时 Modal 默认按钮是 OK / Cancel，点不到「确定」。
    <ConfigProvider locale={zhCN}>
      <App>
        <SourcesPage />
      </App>
    </ConfigProvider>,
  );
}

/** 永不 resolve 的 Promise —— 让弹窗在整个断言期内保持打开。 */
function never(): Promise<never> {
  return new Promise<never>(() => {});
}

async function openCreateModal() {
  fireEvent.click(await screen.findByRole('button', { name: /新增组织/ }));
  fireEvent.change(await screen.findByLabelText('组织名称'), {
    target: { value: '某某科技IT部门' },
  });
  return screen.findByRole('button', { name: /确\s*定/ });
}

describe('来源组织页 Modal 重复提交', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    svc.sourceOrganizations.mockResolvedValue({ items: [] } as never);
  });

  it('🔴 请求挂起期间连点两次「确定」只提交一次', async () => {
    svc.createSourceOrganization.mockImplementation(() => never() as never);

    renderPage();
    const ok = await openCreateModal();

    fireEvent.click(ok);
    // 给第一次点击真正进入 async，再点第二次 —— 这就是竞态窗口
    await waitFor(() => expect(svc.createSourceOrganization).toHaveBeenCalledTimes(1));

    const okAgain = screen.queryByRole('button', { name: /确\s*定/ });
    // 弹窗此刻必须仍然开着，否则本用例没有测到竞态窗口（守卫会白测）
    expect(okAgain).toBeTruthy();
    fireEvent.click(okAgain!);
    await new Promise((r) => setTimeout(r, 150));

    expect(svc.createSourceOrganization).toHaveBeenCalledTimes(1);
  });

  it('🔴 校验失败后按钮必须复活，不能被永久锁死', async () => {
    // 不填必填项 → validateFields reject。若 finally 缺失，submitBusy 会停
    // 在 true，按钮从此点不动 —— 这是补 finally 时最容易引入的自伤缺陷。
    renderPage();
    fireEvent.click(await screen.findByRole('button', { name: /新增组织/ }));
    // 留空直接确定：校验失败
    fireEvent.click(await screen.findByRole('button', { name: /确\s*定/ }));

    // 填好必填项，再一次确定必须能真正走到接口调用 —— 证明守卫已复位
    fireEvent.change(await screen.findByLabelText('组织名称'), {
      target: { value: '某某科技IT部门' },
    });
    svc.createSourceOrganization.mockResolvedValue({ id: 1 } as never);
    fireEvent.click(await screen.findByRole('button', { name: /确\s*定/ }));

    await waitFor(() => expect(svc.createSourceOrganization).toHaveBeenCalledTimes(1));
  });

  it('邮箱字段用中文全角逗号分隔也必须正确切分（不能整串成一个）', async () => {
    svc.createSourceOrganization.mockResolvedValue({ id: 1 } as never);

    renderPage();
    fireEvent.click(await screen.findByRole('button', { name: /新增组织/ }));
    fireEvent.change(await screen.findByLabelText('组织名称'), {
      target: { value: '某某科技IT部门' },
    });
    fireEvent.change(await screen.findByLabelText('邮箱地址（逗号分隔）'), {
      target: { value: 'support@example.com，noc@example.com' },
    });
    fireEvent.change(await screen.findByLabelText('邮箱域名（逗号分隔）'), {
      target: { value: 'example.com，example.org' },
    });

    fireEvent.click(await screen.findByRole('button', { name: /确\s*定/ }));
    await waitFor(() => expect(svc.createSourceOrganization).toHaveBeenCalledTimes(1));

    const [payload] = svc.createSourceOrganization.mock.calls[0];
    expect(payload.emailAddresses).toEqual([
      'support@example.com',
      'noc@example.com',
    ]);
    expect(payload.emailDomains).toEqual(['example.com', 'example.org']);
  });
});
