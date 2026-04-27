import type { Preview } from '@storybook/react';
import { AntdRegistry } from '@ant-design/nextjs-registry';
import '../src/styles/globals.css';

const preview: Preview = {
  decorators: [
    (Story) => (
      <AntdRegistry>
        <Story />
      </AntdRegistry>
    ),
  ],
  parameters: {
    actions: { argTypesRegex: '^on[A-Z].*' },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    backgrounds: {
      default: 'light',
      values: [
        { name: 'light', value: '#ffffff' },
        { name: 'dark', value: '#1f2937' },
        { name: 'gray', value: '#f3f4f6' },
      ],
    },
  },
};

export default preview;
