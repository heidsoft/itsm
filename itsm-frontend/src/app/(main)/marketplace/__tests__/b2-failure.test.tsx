import { fireEvent, render, screen } from '@testing-library/react';
import Page from '../page';
import { httpClient } from '@/lib/api/http-client';
jest.mock('@/lib/api/http-client', () => ({ httpClient: { get: jest.fn() } }));
beforeEach(() => { jest.spyOn(console, 'error').mockImplementation(() => {}); });
it('失败文案可见且可重试，成功空列表才显示空态', async () => {
  jest.mocked(httpClient.get).mockRejectedValueOnce(new Error('市场暂不可用')).mockResolvedValue({ items: [] });
  render(<Page />);
  expect(await screen.findByText('市场暂不可用')).toBeInTheDocument();
  expect(screen.queryByText('没有找到匹配的应用')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
  expect(await screen.findByText('没有找到匹配的应用')).toBeInTheDocument();
});
