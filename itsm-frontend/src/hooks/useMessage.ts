/**
 * 消息提示 Hook
 * 使用 Ant Design App 组件提供的 message API，支持动态主题
 */
import { App } from 'antd';

export const useMessage = () => {
  const { message } = App.useApp();
  return message;
};

export default useMessage;
