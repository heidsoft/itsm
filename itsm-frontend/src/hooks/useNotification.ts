/**
 * 通知提醒 Hook
 * 使用 Ant Design App 组件提供的 notification API，支持动态主题
 */
import { App } from 'antd';

export const useNotification = () => {
  const { notification } = App.useApp();
  return notification;
};

export default useNotification;
