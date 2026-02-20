'use client';

import { App } from 'antd';

/**
 * 提供 antd 的 message, notification, modal 实例
 * 解决静态方法无法消费上下文的问题
 */
export const useAntdApp = () => {
  const { message, notification, modal } = App.useApp();
  return { message, notification, modal };
};
