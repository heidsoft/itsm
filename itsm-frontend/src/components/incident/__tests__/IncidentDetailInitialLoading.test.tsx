import { useLayoutEffect } from 'react';
import { render, screen } from '@testing-library/react';
import { IncidentAPI } from '@/lib/api/';
import IncidentDetail from '../IncidentDetail';

jest.mock('next/navigation', () => ({
  useParams: () => ({}),
  useRouter: () => ({ push: jest.fn() }),
}));
jest.mock('@/lib/api/', () => ({ IncidentAPI: { getIncident: jest.fn() } }));
jest.mock('@/lib/hooks/useUserListQuery', () => ({
  useUserPickerQuery: () => ({ users: [], isLoading: false }),
}));
jest.mock('@/lib/hooks/useErrorHandler', () => ({
  useErrorHandler: () => ({ handleError: jest.fn() }),
}));

it('请求 pending 时首帧即显示加载指示，不闪未找到事件', () => {
  jest.mocked(IncidentAPI.getIncident).mockReturnValue(new Promise(() => {}));
  let initialText = '';
  let initialLoading = false;
  function FirstFrame() {
    // layout 阶段先于加载请求的 passive effect，才能捕获用户实际看到的首帧。
    useLayoutEffect(() => {
      initialText = document.body.textContent || '';
      initialLoading = !!document.querySelector('.ant-skeleton');
    }, []);
    return <IncidentDetail id='7' />;
  }
  const { container } = render(<FirstFrame />);
  expect(IncidentAPI.getIncident).toHaveBeenCalledWith(7);
  expect(initialText).not.toContain('未找到事件');
  expect(initialLoading).toBe(true);
  expect(screen.queryByText('未找到事件')).not.toBeInTheDocument();
  expect(container.querySelector('.ant-skeleton')).toBeInTheDocument();
});
