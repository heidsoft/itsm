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
];
