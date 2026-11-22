import { act, fireEvent, render, screen } from '@testing-library/react';
import { App, Modal } from 'antd';
import CIList from '../CIList';

const mockDelete = jest.fn();
jest.mock('@/lib/hooks/useCMDB', () => ({
  useCIsQuery: () => ({
    data: { items: [{ id: 7, name: '服务器', status: 'active' }], total: 1 },
    refetch: jest.fn(),
  }),
  useCITypesQuery: () => ({ data: [] }),
  useDeleteCIMutation: () => ({ mutateAsync: mockDelete }),
}));
jest.mock('@/lib/hooks/use-permissions', () => ({
  useOperationPermissions: () => ({
    cmdb: { canCreate: () => true, canUpdate: () => true, canDelete: () => true },
  }),
}));
jest.mock('@/lib/i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));

it.each([
  { label: 'Error.message', failure: new Error('删除被拒绝') },
  { label: '非 Error 回退文案', failure: 'unknown' },
])('单删失败提示且确认回调不遗留 rejection：$label', async ({ failure }) => {
  const error = jest.fn();
  jest
    .spyOn(App, 'useApp')
    .mockReturnValue({ message: { error } } as unknown as ReturnType<typeof App.useApp>);
  const confirm = jest
    .spyOn(Modal, 'confirm')
    .mockImplementation(() => ({ destroy: jest.fn(), update: jest.fn() }));
  mockDelete.mockRejectedValueOnce(failure);
  render(<CIList />);
  fireEvent.click(screen.getByRole('button', { name: /^删除$/ }));
  const onOk = confirm.mock.calls[0][0].onOk;
  if (!onOk) throw new Error('缺少确认回调');
  await act(async () => {
    await expect(onOk()).resolves.toBeUndefined();
  });
  expect(mockDelete).toHaveBeenCalledWith('7');
  expect(error).toHaveBeenCalledTimes(1);
  expect(error).toHaveBeenCalledWith(failure instanceof Error ? failure.message : '删除配置项失败');
});
