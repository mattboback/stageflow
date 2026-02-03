import js from '@eslint/js';
import perfectionist from 'eslint-plugin-perfectionist';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import ts from 'typescript-eslint';
import svelteConfig from './svelte.config.js';

import path from 'node:path';
import { fileURLToPath } from 'node:url';

// Safer than import.meta.dirname across Node/tooling combos
const tsconfigRootDir = path.dirname(fileURLToPath(import.meta.url));

const tsFiles = ['**/*.{ts,tsx,cts,mts}'];

const perfectionistSortImports = [
	'error',
	{
		type: 'natural',
		order: 'asc'
	}
];

const unusedVarsOptions = {
	argsIgnorePattern: '^_',
	varsIgnorePattern: '^_',
	caughtErrorsIgnorePattern: '^_'
};

export default ts.config(
	{
		ignores: ['build/**', '.svelte-kit/**', 'coverage/**', 'dist/**', 'node_modules/**']
	},
	{
		linterOptions: {
			reportUnusedDisableDirectives: 'error'
		}
	},

	// Base JS recommended
	js.configs.recommended,

	// TypeScript typed + stylistic typed (apply only to TS files)
	...ts.configs.recommendedTypeChecked.map((c) => ({ ...c, files: tsFiles })),
	...ts.configs.stylisticTypeChecked.map((c) => ({ ...c, files: tsFiles })),

	// Svelte recommended (flat config compatible per plugin docs)
	...svelte.configs.recommended,

	// Globals once
	{
		languageOptions: {
			globals: {
				...globals.browser,
				...globals.node
			}
		}
	},

	/**
	 * IMPORTANT: Centralize TS parser options used for typed linting.
	 * - projectService is the recommended typed linting mode.
	 * - extraFileExtensions + svelteConfig per eslint-plugin-svelte TypeScript example.
	 */
	{
		files: [...tsFiles, '**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
		languageOptions: {
			parserOptions: {
				projectService: true,
				tsconfigRootDir,
				extraFileExtensions: ['.svelte'],
				parser: ts.parser,
				svelteConfig
			}
		}
	},

	// Ensure embedded TS in .svelte.* gets parsed by TS parser
	{
		files: ['**/*.svelte.ts', '**/*.svelte.js'],
		languageOptions: {
			parser: ts.parser
		}
	},

	// TS-only rules/plugins
	{
		files: tsFiles,
		plugins: {
			'@typescript-eslint': ts.plugin,
			perfectionist
		},
		rules: {
			'perfectionist/sort-imports': perfectionistSortImports,

			// ==========================================
			// Type safety - imports/exports
			// ==========================================
			'@typescript-eslint/consistent-type-imports': [
				'error',
				{ prefer: 'type-imports', fixStyle: 'inline-type-imports' }
			],
			'@typescript-eslint/consistent-type-exports': [
				'error',
				{ fixMixedExportsWithInlineTypeSpecifier: true }
			],

			// ==========================================
			// Type safety - any handling
			// ==========================================
			'@typescript-eslint/no-explicit-any': 'error',
			'@typescript-eslint/no-unsafe-assignment': 'warn',
			'@typescript-eslint/no-unsafe-member-access': 'warn',
			'@typescript-eslint/no-unsafe-call': 'warn',
			'@typescript-eslint/no-unsafe-argument': 'warn',
			'@typescript-eslint/no-unsafe-return': 'warn',

			// ==========================================
			// Code quality
			// ==========================================
			'@typescript-eslint/no-unnecessary-condition': 'warn',
			'@typescript-eslint/no-confusing-void-expression': 'warn',
			'@typescript-eslint/no-base-to-string': 'warn',
			'@typescript-eslint/use-unknown-in-catch-callback-variable': 'warn',

			// ==========================================
			// Best practices
			// ==========================================
			'@typescript-eslint/prefer-nullish-coalescing': [
				'error',
				{ ignoreConditionalTests: true, ignoreMixedLogicalExpressions: true }
			],
			'@typescript-eslint/non-nullable-type-assertion-style': 'warn',
			'@typescript-eslint/prefer-regexp-exec': 'off',
			'@typescript-eslint/unbound-method': 'warn',

			// Allow numbers/booleans/nullish in template literals
			'@typescript-eslint/restrict-template-expressions': [
				'error',
				{
					allowNumber: true,
					allowBoolean: true,
					allowNullish: true
				}
			],

			// ==========================================
			// Variables and naming
			// ==========================================
			'@typescript-eslint/no-unused-vars': ['error', unusedVarsOptions],

			// ==========================================
			// Console usage
			// ==========================================
			'no-console': ['error', { allow: ['warn', 'error'] }]
		}
	},

	// Svelte rules/plugins (template-aware)
	{
		files: ['**/*.svelte'],
		plugins: {
			'@typescript-eslint': ts.plugin,
			perfectionist
		},
		rules: {
			'perfectionist/sort-imports': perfectionistSortImports,

			// Prefer TS version (and avoid duplicate reports)
			'no-unused-vars': 'off',
			'@typescript-eslint/no-unused-vars': ['error', unusedVarsOptions],

			// Disable for static routes - resolve() only needed for dynamic param routes
			'svelte/no-navigation-without-resolve': 'off',

			// Each blocks must have keys for performance
			'svelte/require-each-key': 'error',

			// Allow @html for CMS/markdown content (review usage for XSS)
			'svelte/no-at-html-tags': 'warn',

			// SvelteSet/SvelteMap are nice but not strictly required
			'svelte/prefer-svelte-reactivity': 'warn'
		}
	},

	// Avoid type-linting on config file itself
	{
		files: ['svelte.config.js'],
		languageOptions: {
			parserOptions: {
				project: null
			}
		}
	}
);
