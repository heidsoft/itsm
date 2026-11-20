/**
 * 回归锁：A2UI 动态表单的「提交」按钮必须防重复建单。
 *
 * 缺陷：按钮 disabled 只绑数据模型 isEnabled；handleAction 的 loading 只喂外层 Spin
 * （Spin 不阻断点击），网络延迟窗口内可多次 POST handleTicketAction('submit')。
 */
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { App } from 'antd';
import A2UIFormRenderer from '../A2UIFormRenderer';

const mockGenerate = jest.fn();
const mockHandleAction = jest.fn();
const mockIsAuthenticated = { current: true };

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: Object.assign(
    () => ({ isAuthenticated: mockIsAuthenticated.current }),
    { getState: () => ({ isAuthenticated: mockIsAuthenticated.current }) }
  ),
}));

jest.mock('@/lib/api/a2ui-api', () => ({
  A2UIApi: {
    generateTicketForm: (...args: unknown[]) => mockGenerate(...args),
    handleTicketAction: (...args: unknown[]) => mockHandleAction(...args),
  },
}));

jest.mock('@/lib/i18n/useI18n', () => ({ useI18n: () => ({ t: (k: string) => k }) }));

const surfaceUpdate = JSON.stringify({
  surfaceUpdate: {
    surfaceId: 'surf-1',
    components: [
      { id: 'root', component: { Column: { children: { explicitList: ['submitBtn'] } } } },
      { id: 'submitBtn', component: { Button: {
        child: 'btnText',
        action: { name: 'submit', context: [] },
      } } },
      { id: 'btnText', component: { Text: { text: { literalString: '提交' } } } },
    ],
  },
});

describe('A2UI 表单提交防重复', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockIsAuthenticated.current = true;
    mockGenerate.mockResolvedValue({ messages: [surfaceUpdate] });
  });

  async function renderGeneratedForm() {
    render(<App><A2UIFormRenderer /></App>);
    fireEvent.change(screen.getByPlaceholderText(/ThinkPad/), { target: { value: '打印机坏了' } });
    // Antd 把两个汉字的「发送」渲染为「发 送」
    const sendBtn = screen.getAllByRole('button').find(b => /发\s*送/.test(b.textContent || ''));
    if (!sendBtn) throw new Error('send button not found');
    fireEvent.click(sendBtn);
    await waitFor(() => expect(mockGenerate).toHaveBeenCalled());
    return await screen.findByRole('button', { name: /提\s*交/ });
  }

  it('提交在途时按钮 disabled，双击只发一次 handleTicketAction', async () => {
    let resolveAction: () => void = () => {};
    mockHandleAction.mockImplementation(
      () => new Promise<{ messages: string[] }>(resolve => { resolveAction = () => resolve({ messages: [] }); })
    );

    const submitBtn = await renderGeneratedForm();

    fireEvent.click(submitBtn);
    await waitFor(() => expect(mockHandleAction).toHaveBeenCalledTimes(1));

    expect(submitBtn).toBeDisabled();

    fireEvent.click(submitBtn);
    fireEvent.click(submitBtn);
    expect(mockHandleAction).toHaveBeenCalledTimes(1);

    resolveAction();
    await waitFor(() => expect(mockHandleAction).toHaveBeenCalledTimes(1));
  });
});
