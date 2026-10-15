import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import js from '@eslint/js';
import eslintComments from '@eslint-community/eslint-plugin-eslint-comments';
import globals from 'globals';
import jsxA11y from 'eslint-plugin-jsx-a11y';
import importX from 'eslint-plugin-import-x';
import promise from 'eslint-plugin-promise';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import sonarjs from 'eslint-plugin-sonarjs';
import unicorn from 'eslint-plugin-unicorn';
import tseslint from 'typescript-eslint';

const __dirname = import.meta.dirname ?? dirname(fileURLToPath(import.meta.url));

export default tseslint.config(
  // Global ignores
  { ignores: ['dist', 'node_modules', 'coverage', '*.config.js', '*.config.ts', '.pnp.*', '.yarn/', 'src/vite-env.d.ts'] },

  // Base configs for all TS/TSX files
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      ...tseslint.configs.strictTypeChecked,
      ...tseslint.configs.stylisticTypeChecked,
    ],
    languageOptions: {
      ecmaVersion: 2022,
      globals: {
        ...globals.browser,
        ...globals.es2022,
      },
      parserOptions: {
        projectService: true,
        tsconfigRootDir: __dirname,
        ecmaFeatures: {
          jsx: true,
        },
      },
    },
    settings: {
      react: {
        version: 'detect',
      },
      'import-x/resolver': {
        typescript: {
          alwaysTryTypes: true,
        },
      },
    },
    plugins: {
      react,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      'jsx-a11y': jsxA11y,
      'import-x': importX,
      unicorn,
      sonarjs,
      promise,
      '@eslint-community/eslint-comments': eslintComments,
    },
    rules: {
      // ============================================
      // React
      // ============================================
      ...react.configs.recommended.rules,
      ...react.configs['jsx-runtime'].rules,
      'react/prop-types': 'off', // Using TypeScript
      'react/jsx-no-target-blank': 'error',
      'react/jsx-no-useless-fragment': 'error',
      'react/jsx-curly-brace-presence': ['error', { props: 'never', children: 'never' }],
      'react/self-closing-comp': 'error',
      'react/jsx-boolean-value': ['error', 'never'],
      'react/jsx-no-constructed-context-values': 'error',
      'react/hook-use-state': 'error',
      'react/no-array-index-key': 'warn',
      'react/no-unstable-nested-components': 'error',
      // React security rules
      'react/no-danger': 'warn',
      'react/no-danger-with-children': 'error',
      'react/no-children-prop': 'error',

      // React Hooks
      ...reactHooks.configs.recommended.rules,

      // React Refresh (HMR)
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],

      // ============================================
      // Accessibility (jsx-a11y)
      // ============================================
      ...jsxA11y.configs.strict.rules,

      // ============================================
      // TypeScript
      // ============================================
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      '@typescript-eslint/consistent-type-imports': [
        'error',
        { prefer: 'type-imports', fixStyle: 'inline-type-imports' },
      ],
      '@typescript-eslint/consistent-type-definitions': ['error', 'interface'],
      '@typescript-eslint/no-confusing-void-expression': ['error', { ignoreArrowShorthand: true }],
      '@typescript-eslint/no-misused-promises': [
        'error',
        { checksVoidReturn: { attributes: false } },
      ],
      '@typescript-eslint/restrict-template-expressions': [
        'error',
        { allowNumber: true, allowBoolean: true },
      ],
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/await-thenable': 'error',
      '@typescript-eslint/no-unnecessary-condition': 'error',
      '@typescript-eslint/prefer-nullish-coalescing': 'error',
      '@typescript-eslint/prefer-optional-chain': 'error',
      '@typescript-eslint/strict-boolean-expressions': [
        'error',
        {
          allowString: false,
          allowNumber: false,
          allowNullableObject: true,
          allowNullableBoolean: true,
          allowNullableString: false,
          allowNullableNumber: false,
          allowAny: false,
        },
      ],
      // Additional strict TypeScript rules
      '@typescript-eslint/ban-ts-comment': [
        'error',
        {
          'ts-check': false,
          'ts-expect-error': 'allow-with-description',
          'ts-ignore': 'allow-with-description',
          'ts-nocheck': true,
        },
      ],
      '@typescript-eslint/no-unnecessary-type-constraint': 'error',
      // Relax some overly strict rules for React patterns
      '@typescript-eslint/no-non-null-assertion': 'warn',
      '@typescript-eslint/unbound-method': 'off', // Conflicts with React event handlers

      // ============================================
      // Imports (import-x)
      // ============================================
      'import-x/no-unresolved': 'error',
      'import-x/named': 'error',
      'import-x/no-named-as-default': 'warn',
      'import-x/no-named-as-default-member': 'warn',
      'import-x/no-duplicates': 'error',
      'import-x/no-cycle': ['error', { maxDepth: 4 }],
      'import-x/no-self-import': 'error',
      'import-x/no-useless-path-segments': 'error',
      'import-x/first': 'error',
      'import-x/newline-after-import': 'error',
      'import-x/no-mutable-exports': 'error',
      'import-x/order': [
        'error',
        {
          groups: ['builtin', 'external', 'internal', 'parent', 'sibling', 'index', 'type'],
          'newlines-between': 'never',
          alphabetize: { order: 'asc', caseInsensitive: true },
        },
      ],

      // ============================================
      // Unicorn (modern JS best practices)
      // ============================================
      ...unicorn.configs['flat/recommended'].rules,
      'unicorn/filename-case': [
        'error',
        {
          cases: {
            camelCase: true,
            pascalCase: true,
          },
        },
      ],
      'unicorn/prevent-abbreviations': 'off', // Too aggressive for React (props, ref, etc.)
      'unicorn/no-null': 'off', // React uses null for conditional rendering
      'unicorn/no-useless-undefined': 'off', // Conflicts with explicit return types
      'unicorn/prefer-module': 'off', // Vite handles this
      'unicorn/no-array-reduce': 'off', // Sometimes reduce is cleaner
      'unicorn/no-array-for-each': 'off', // forEach is fine for side effects
      'unicorn/prefer-spread': 'warn', // Suggest but don't error
      'unicorn/prefer-ternary': 'off', // Ternary isn't always more readable
      'unicorn/switch-case-braces': ['error', 'avoid'],
      'unicorn/prefer-top-level-await': 'off', // Not applicable in React components
      'unicorn/no-array-sort': 'off', // Using [...arr].sort() for broader browser compatibility (ES2023 toSorted not supported in Chrome <110)

      // ============================================
      // SonarJS (code smells & complexity)
      // ============================================
      ...sonarjs.configs.recommended.rules,
      'sonarjs/cognitive-complexity': ['error', 15],
      'sonarjs/no-duplicate-string': ['error', { threshold: 3 }],
      'sonarjs/no-identical-functions': 'error',
      'sonarjs/prefer-immediate-return': 'error',

      // ============================================
      // Promise
      // ============================================
      ...promise.configs['flat/recommended'].rules,
      'promise/always-return': 'off', // Too strict for React patterns
      'promise/catch-or-return': ['error', { allowFinally: true }],

      // ============================================
      // ESLint Comments
      // ============================================
      '@eslint-community/eslint-comments/no-unused-disable': 'error',
      '@eslint-community/eslint-comments/no-unlimited-disable': 'error',
      '@eslint-community/eslint-comments/no-duplicate-disable': 'error',
      '@eslint-community/eslint-comments/require-description': [
        'error',
        { ignore: ['eslint-enable'] },
      ],

      // ============================================
      // General JavaScript
      // ============================================
      // Security rules
      'no-eval': 'error',
      'no-implied-eval': 'error',
      'no-new-func': 'error',
      // General rules
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      eqeqeq: ['error', 'always'],
      'no-else-return': 'error',
      'no-lonely-if': 'error',
      'no-unneeded-ternary': 'error',
      'prefer-const': 'error',
      'object-shorthand': 'error',
      'prefer-arrow-callback': 'error',
      'prefer-template': 'error',
      'no-nested-ternary': 'error',
      curly: ['error', 'all'],
    },
  },

  // Test file overrides
  {
    files: ['**/*.test.{ts,tsx}', '**/*.spec.{ts,tsx}', '**/test/**/*.{ts,tsx}'],
    rules: {
      'react-refresh/only-export-components': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unsafe-assignment': 'off',
      '@typescript-eslint/no-unsafe-member-access': 'off',
      '@typescript-eslint/no-unsafe-call': 'off',
      '@typescript-eslint/no-unsafe-return': 'off',
      '@typescript-eslint/ban-ts-comment': 'off',
      '@typescript-eslint/no-empty-function': 'off',
      '@typescript-eslint/no-floating-promises': 'off',
      'sonarjs/no-duplicate-string': 'off',
      'sonarjs/no-hardcoded-passwords': 'off',
      'sonarjs/no-hardcoded-ip': 'off',
      'sonarjs/cognitive-complexity': 'off',
      '@eslint-community/eslint-comments/require-description': 'off',
    },
  }
);
