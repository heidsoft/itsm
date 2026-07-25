import { renderHook, act } from '@testing-library/react';
import { useFeedback } from '../useFeedback';

// Mock antd App.useApp
const mockMessage = {
  success: jest.fn(),
  error: jest.fn(),
  warning: jest.fn(),
  info: jest.fn(),
  loading: jest.fn(),
  destroy: jest.fn(),
};

const mockNotification = {
  success: jest.fn(),
  error: jest.fn(),
  warning: jest.fn(),
};

jest.mock('antd', () => ({
  App: {
    useApp: () => ({
      message: mockMessage,
      notification: mockNotification,
    }),
  },
}));

describe('useFeedback', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return all feedback functions', () => {
    const { result } = renderHook(() => useFeedback());

    expect(typeof result.current.showSuccess).toBe('function');
    expect(typeof result.current.showError).toBe('function');
    expect(typeof result.current.showWarning).toBe('function');
    expect(typeof result.current.showInfo).toBe('function');
    expect(typeof result.current.showLoading).toBe('function');
    expect(typeof result.current.hideLoading).toBe('function');
    expect(typeof result.current.showSuccessNotification).toBe('function');
    expect(typeof result.current.showErrorNotification).toBe('function');
    expect(typeof result.current.showWarningNotification).toBe('function');
  });

  describe('showSuccess', () => {
    it('should call message.success with default duration', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showSuccess('Operation succeeded');
      });

      expect(mockMessage.success).toHaveBeenCalledWith({
        content: 'Operation succeeded',
        duration: 2,
      });
    });

    it('should call message.success with custom duration', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showSuccess('Done', 5);
      });

      expect(mockMessage.success).toHaveBeenCalledWith({
        content: 'Done',
        duration: 5,
      });
    });
  });

  describe('showError', () => {
    it('should call message.error with default duration', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showError('Something went wrong');
      });

      expect(mockMessage.error).toHaveBeenCalledWith({
        content: 'Something went wrong',
        duration: 3,
      });
    });

    it('should call message.error with custom duration', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showError('Error!', 10);
      });

      expect(mockMessage.error).toHaveBeenCalledWith({
        content: 'Error!',
        duration: 10,
      });
    });
  });

  describe('showWarning', () => {
    it('should call message.warning with default duration', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showWarning('Be careful');
      });

      expect(mockMessage.warning).toHaveBeenCalledWith({
        content: 'Be careful',
        duration: 2,
      });
    });
  });

  describe('showInfo', () => {
    it('should call message.info with default duration', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showInfo('FYI');
      });

      expect(mockMessage.info).toHaveBeenCalledWith({
        content: 'FYI',
        duration: 2,
      });
    });
  });

  describe('showLoading', () => {
    it('should call message.loading with key and duration 0', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showLoading('Loading...', 'load-key');
      });

      expect(mockMessage.loading).toHaveBeenCalledWith({
        content: 'Loading...',
        key: 'load-key',
        duration: 0,
      });
    });
  });

  describe('hideLoading', () => {
    it('should call message.destroy with key', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.hideLoading('load-key');
      });

      expect(mockMessage.destroy).toHaveBeenCalledWith('load-key');
    });
  });

  describe('showSuccessNotification', () => {
    it('should call notification.success with default params', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showSuccessNotification('Success Title');
      });

      expect(mockNotification.success).toHaveBeenCalledWith({
        message: 'Success Title',
        description: undefined,
        duration: 4.5,
        placement: 'topRight',
      });
    });

    it('should call notification.success with description and custom duration', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showSuccessNotification('Title', 'Detailed description', 10);
      });

      expect(mockNotification.success).toHaveBeenCalledWith({
        message: 'Title',
        description: 'Detailed description',
        duration: 10,
        placement: 'topRight',
      });
    });
  });

  describe('showErrorNotification', () => {
    it('should call notification.error', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showErrorNotification('Error Title', 'Error details');
      });

      expect(mockNotification.error).toHaveBeenCalledWith({
        message: 'Error Title',
        description: 'Error details',
        duration: 4.5,
        placement: 'topRight',
      });
    });
  });

  describe('showWarningNotification', () => {
    it('should call notification.warning', () => {
      const { result } = renderHook(() => useFeedback());

      act(() => {
        result.current.showWarningNotification('Warning Title', 'Warning details', 7);
      });

      expect(mockNotification.warning).toHaveBeenCalledWith({
        message: 'Warning Title',
        description: 'Warning details',
        duration: 7,
        placement: 'topRight',
      });
    });
  });
});
