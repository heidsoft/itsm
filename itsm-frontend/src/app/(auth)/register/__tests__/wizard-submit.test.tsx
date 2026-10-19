import React from 'react';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { render } from '@/lib/test-utils';
import RegisterPage from '../page';

const mockRegister = jest.fn();
const mockPush = jest.fn();

jest.mock('@/lib/services/auth-service', () => ({
  AuthService: {
    register: (...args: unknown[]) => mockRegister(...args),
  },
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
}));

jest.mock('@/lib/env', () => ({
  logger: { info: jest.fn(), warn: jest.fn(), error: jest.fn() },
}));

// antd Form 每次 change 都重渲染整张表，--runInBand 下每条链路会接近默认 10s 上限。
jest.setTimeout(30_000);

function setValue(label: string, value: string) {
  // userEvent.type 在 antd Form 上每个按键都触发一次校验渲染，整条链路会超过
  // Jest 默认 10s；这里只需要字段最终值，用 change 事件一次写入。
  const control = screen.getByLabelText(label) as HTMLInputElement;
  fireEvent.change(control, { target: { value } });
}

async function completeStepOne(user: ReturnType<typeof userEvent.setup>, username = 'newbie') {
  setValue('用户名', username);
  setValue('邮箱', 'newbie@example.com');
  setValue('密码', 'password123');
  setValue('确认密码', 'password123');
  await user.click(screen.getByRole('button', { name: /下一步/ }));
}

async function submitStepTwo(user: ReturnType<typeof userEvent.setup>) {
  setValue('姓名', '新员工');
  setValue('手机号', '13900001018');
  await user.click(screen.getByRole('button', { name: /注册/ }));
}

describe('注册页两步向导', () => {
  beforeEach(() => {
    mockRegister.mockReset();
    mockPush.mockClear();
  });

  it('第二步提交时携带第一步的账号与密码', async () => {
    // 历史上每一步各自包了一个 <Form>，切到第二步后第一步的字段被卸载，
    // 提交体只剩姓名/手机号，后端因缺少必填项返回 400，新用户永远注册不成功。
    const user = userEvent.setup();
    mockRegister.mockResolvedValue({ ok: true });

    render(<RegisterPage />);
    await completeStepOne(user);
    await submitStepTwo(user);

    await waitFor(() => expect(mockRegister).toHaveBeenCalledTimes(1));
    expect(mockRegister).toHaveBeenCalledWith(
      expect.objectContaining({
        username: 'newbie',
        email: 'newbie@example.com',
        password: 'password123',
        fullName: '新员工',
        phone: '13900001018',
      })
    );
  });

  it('用户名含下划线等非法字符时先在前端拦住', async () => {
    // 后端 binding 是 alphanum；前端只校验 required+min=3，用户要提交后才知道
    // 用户名不合法，而且拿到的是一句通用「参数错误」。
    const user = userEvent.setup();

    render(<RegisterPage />);
    await completeStepOne(user, 'new_user');

    expect(await screen.findByText('用户名只能用字母和数字')).toBeInTheDocument();
    // 仍停留在第一步：第二步字段容器保持 hidden
    expect(screen.getByLabelText('姓名').closest('[hidden]')).not.toBeNull();
    expect(mockRegister).not.toHaveBeenCalled();
  });

  it('服务端返回的失败原因直接展示给用户', async () => {
    const user = userEvent.setup();
    mockRegister.mockResolvedValue({ ok: false, message: '用户名已被注册' });

    render(<RegisterPage />);
    await completeStepOne(user);
    await submitStepTwo(user);

    expect(await screen.findByText('用户名已被注册')).toBeInTheDocument();
  });

  it('注册成功后跳转登录页', async () => {
    const user = userEvent.setup();
    mockRegister.mockResolvedValue({ ok: true });

    render(<RegisterPage />);
    await completeStepOne(user);
    await submitStepTwo(user);

    await waitFor(() => expect(mockPush).toHaveBeenCalledWith('/login'));
  });
});
