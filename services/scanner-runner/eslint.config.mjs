// @ts-check
import eslint from '@eslint/js';
import perfectionist from 'eslint-plugin-perfectionist';
import vitest from 'eslint-plugin-vitest';
import tseslint from 'typescript-eslint';

const tsFiles = ['**/*.{ts,tsx,mts,cts}'];
const testFiles = ['**/*.{test,spec}.{ts,tsx}', '**/__tests__/**'];

export default tseslint.config(
	// ------------------------------------------
	// Global ignores (standalone => truly global)
	// ------------------------------------------
	{
		ignores: ['dist/**', 'node_modules/**', 'coverage/**', '**/*.config.*']
	},

	{
		linterOptions: {
			reportUnusedDisableDirectives: 'error'
		}
	},

	// Base ESLint recommended rules
	eslint.configs.recommended,

	// TypeScript strict type-checked rules (don’t remap; keep upstream file targeting)
	...tseslint.configs.strictTypeChecked,
	...tseslint.configs.stylisticTypeChecked,

	// ------------------------------------------
	// TypeScript project service + repo root
	// ------------------------------------------
	{
		files: tsFiles,
		languageOptions: {
			parserOptions: {
				projectService: {
					// Add more “out of project” root files here as needed
					allowDefaultProject: [
						'vitest.config.ts',
						'vite.config.ts',
						'playwright.config.ts',
						'eslint.config.*'
					]
				},
				tsconfigRootDir: import.meta.dirname
			}
		},
		plugins: {
			perfectionist
		},
		rules: {
			'perfectionist/sort-imports': ['error', { type: 'natural', order: 'asc' }],

			// Allow unused vars with underscore prefix
			'@typescript-eslint/no-unused-vars': [
				'error',
				{
					argsIgnorePattern: '^_',
					varsIgnorePattern: '^_',
					caughtErrorsIgnorePattern: '^_'
				}
			],

			'@typescript-eslint/consistent-type-imports': [
				'error',
				{
					prefer: 'type-imports',
					fixStyle: 'inline-type-imports',
					disallowTypeAnnotations: false
				}
			],
			'@typescript-eslint/consistent-type-exports': [
				'error',
				{ fixMixedExportsWithInlineTypeSpecifier: true }
			],

			'@typescript-eslint/no-explicit-any': 'error',
			'@typescript-eslint/no-non-null-assertion': 'warn',

			'@typescript-eslint/restrict-template-expressions': [
				'error',
				{ allowNumber: true, allowBoolean: true, allowNullish: true }
			],

			// Temporarily relaxed rules
			'@typescript-eslint/no-unnecessary-condition': 'warn',
			'@typescript-eslint/no-unsafe-assignment': 'warn',
			'@typescript-eslint/no-unsafe-member-access': 'warn',
			'@typescript-eslint/no-unsafe-call': 'warn',
			'@typescript-eslint/no-unsafe-argument': 'warn',
			'@typescript-eslint/no-unsafe-return': 'warn',
			'@typescript-eslint/no-confusing-void-expression': 'warn',
			'@typescript-eslint/no-base-to-string': 'warn',
			'@typescript-eslint/use-unknown-in-catch-callback-variable': 'warn',
			'@typescript-eslint/prefer-nullish-coalescing': 'warn',

			'@typescript-eslint/no-floating-promises': ['error', { ignoreVoid: true, ignoreIIFE: true }],

			'@typescript-eslint/naming-convention': [
				'error',
				{ selector: 'default', format: ['camelCase', 'PascalCase'], leadingUnderscore: 'allow' },
				{
					selector: 'variable',
					format: ['camelCase', 'UPPER_CASE', 'PascalCase'],
					leadingUnderscore: 'allow'
				},
				{ selector: 'parameter', format: ['camelCase'], leadingUnderscore: 'allow' },
				{ selector: 'typeLike', format: ['PascalCase'] },
				{ selector: 'enumMember', format: ['UPPER_CASE', 'PascalCase'] },
				{ selector: 'property', format: null }
			],

			// General best practices
			'no-console': 'off',
			'prefer-const': 'error',
			'no-var': 'error',
			eqeqeq: ['error', 'always', { null: 'ignore' }],
			curly: ['error', 'all'],
			'no-debugger': 'error',
			'no-eval': 'error',
			'no-implied-eval': 'error',
			'no-throw-literal': 'error',
			'prefer-promise-reject-errors': 'error',

			// Code quality metrics
			complexity: ['warn', 30],
			'max-depth': ['warn', 6],
			'max-lines-per-function': 'off',
			'max-nested-callbacks': ['warn', 5],

			// Import organization
			'no-duplicate-imports': 'error'
		}
	},

	// ------------------------------------------
	// Tests: Vitest recommended + globals + relaxations
	// ------------------------------------------
	{
		files: testFiles,
		plugins: { vitest },
		rules: {
			...vitest.configs.recommended.rules,

			// your relaxations
			'@typescript-eslint/unbound-method': 'off',
			'@typescript-eslint/no-explicit-any': 'off',
			'@typescript-eslint/no-non-null-assertion': 'off',
			'@typescript-eslint/no-unsafe-assignment': 'off',
			'@typescript-eslint/no-unsafe-member-access': 'off',
			'@typescript-eslint/no-unsafe-call': 'off',
			'@typescript-eslint/no-unsafe-argument': 'off',
			'max-lines-per-function': 'off',

			// keep these explicitly if you like the “hard fail”
			'vitest/no-disabled-tests': 'error',
			'vitest/no-focused-tests': 'error'
		},
		languageOptions: {
			globals: {
				...vitest.environments.env.globals
			}
		}
	}
);
