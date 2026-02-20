/**
 * 模态框 Hook
 * 使用 Ant Design App 组件提供的 modal API，支持动态主题
 */
import { App } from 'antd';

export const useModal = () => {
  const { modal } = App.useApp();
  return modal;
};

export default useModal;
