import tsParser from '@typescript-eslint/parser';
import tsPlugin from '@typescript-eslint/eslint-plugin';
import reactHooksPlugin from 'eslint-plugin-react-hooks';
import reactPlugin from 'eslint-plugin-react';

export default [
  {
    ignores: [
      '.next/**',
      'coverage/**',
      'node_modules/**',
      'playwright-report/**',
      'test-results/**',
      'tests/e2e/venv/**',
      'output/**',
      '.jest-cache/**',
      '.storybook/**',
      'public/**',
      '*.config.js',
      '*.setup.js',
      'screenshot.js',
      'next-env.d.ts',
    ],
  },
  {
    files: ['**/*.{js,jsx,ts,tsx}'],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: 2020,
        sourceType: 'module',
        ecmaFeatures: {
          jsx: true,
        },
        project: './tsconfig.json',
      },
      globals: {
        window: 'readonly',
        document: 'readonly',
        console: 'readonly',
        Blob: 'readonly',
        URL: 'readonly',
        FileReader: 'readonly',
        ResizeObserver: 'readonly',
        requestAnimationFrame: 'readonly',
        cancelAnimationFrame: 'readonly',
        KeyboardEvent: 'readonly',
        HTMLElement: 'readonly',
        MouseEvent: 'readonly',
        React: 'readonly',
        process: 'readonly',
        jest: 'readonly',
        describe: 'readonly',
        it: 'readonly',
        test: 'readonly',
        expect: 'readonly',
        beforeEach: 'readonly',
        afterEach: 'readonly',
      },
    },
    plugins: {
      '@typescript-eslint': tsPlugin,
      'react-hooks': reactHooksPlugin,
      react: reactPlugin,
    },
    settings: {
      react: {
        version: 'detect',
      },
    },
    rules: {
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'off', // 渐进式改进，逐步开启
      'react/jsx-uses-react': 'off',
      'react/react-in-jsx-scope': 'off',
      'react/prop-types': 'off',
      'no-console': 'off', // 开发阶段允许 console
      'no-debugger': 'error',
      'no-alert': 'off', // 开发阶段允许 alert
      'no-var': 'error',
      'prefer-const': 'error',
      'no-unused-vars': 'off', // 使用 @typescript-eslint 版本
      '@typescript-eslint/no-unused-vars': 'off', // 渐进式改进，逐步开启
      '@typescript-eslint/no-explicit-any': 'off', // 渐进式改进，逐步开启
      '@typescript-eslint/consistent-type-imports': 'off', // 渐进式改进，逐步开启
      '@typescript-eslint/no-non-null-assertion': 'off', // 渐进式改进，逐步开启
    },
  },
  {
    files: ['tests/**/*.{js,jsx,ts,tsx}', '**/*.test.{ts,tsx}', '**/*.spec.{ts,tsx}'],
    languageOptions: {
      parserOptions: {
        project: null,
      },
    },
    rules: {
      'no-console': 'off',
      'react-hooks/rules-of-hooks': 'off',
      'react-hooks/exhaustive-deps': 'off',
    },
  },
  // 禁止在业务层直接 fetch()，统一走 src/lib/api/httpClient
  // 例外：src/lib/api/**（httpClient 自身实现）、src/app/api/**（Next.js route handler, Node 侧）
  //       pwa.ts(SW 注册)、security.ts(CSRF token 获取先于 httpClient)、NetworkStatus.tsx(健康探测)
  {
    files: ['src/app/**/*.tsx', 'src/lib/services/**/*.ts', 'src/components/**/*.tsx'],
    ignores: ['src/app/api/**'],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='fetch']",
          message:
            '禁止直接调用 fetch()，请使用 src/lib/api/httpClient 统一封装的 API client。' +
            '如确需例外（SW 注册、CSRF token 获取、健康探测），' +
            '在调用上方加 // eslint-disable-next-line no-restricted-syntax 并注明理由。',
        },
      ],
    },
  },
];
