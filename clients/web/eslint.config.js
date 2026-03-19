import js from "@eslint/js";
import vitest from "@vitest/eslint-plugin";
import perfectionist from "eslint-plugin-perfectionist";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import tseslint from "typescript-eslint";
import svelteConfig from "./svelte.config.js";

import path from "node:path";
import { fileURLToPath } from "node:url";

const tsconfigRootDir = path.dirname(fileURLToPath(import.meta.url));

const tsAndSvelteFiles = [
	"**/*.{ts,tsx,cts,mts}",
	"**/*.svelte",
	"**/*.svelte.ts",
	"**/*.svelte.js",
];

const testFiles = [
	"tests/**/*.{ts,tsx,js,svelte}",
	"src/**/*.{test,spec}.{ts,tsx,js,svelte}",
	"**/__tests__/**/*.{ts,tsx,js,svelte}",
	".storybook/**/*.{ts,tsx,js,svelte}",
	"vitest.config.ts",
];

const unusedVarsOptions = {
	argsIgnorePattern: "^_",
	varsIgnorePattern: "^_",
	caughtErrorsIgnorePattern: "^_",
};

const commonTypeRules = {
	"perfectionist/sort-imports": ["error", { type: "natural", order: "asc" }],

	"@typescript-eslint/consistent-type-imports": [
		"error",
		{
			prefer: "type-imports",
			fixStyle: "inline-type-imports",
			disallowTypeAnnotations: false,
		},
	],
	"@typescript-eslint/consistent-type-exports": [
		"error",
		{ fixMixedExportsWithInlineTypeSpecifier: true },
	],

	"@typescript-eslint/no-unused-vars": ["error", unusedVarsOptions],
	"@typescript-eslint/no-explicit-any": "error",
	"@typescript-eslint/no-non-null-assertion": "error",

	"@typescript-eslint/no-unsafe-assignment": "error",
	"@typescript-eslint/no-unsafe-member-access": "error",
	"@typescript-eslint/no-unsafe-call": "error",
	"@typescript-eslint/no-unsafe-argument": "error",
	"@typescript-eslint/no-unsafe-return": "error",

	"@typescript-eslint/no-unnecessary-condition": "error",
	"@typescript-eslint/no-confusing-void-expression": "error",
	"@typescript-eslint/no-base-to-string": "error",
	"@typescript-eslint/use-unknown-in-catch-callback-variable": "error",
	"@typescript-eslint/non-nullable-type-assertion-style": "error",
	"@typescript-eslint/unbound-method": "error",

	"@typescript-eslint/no-floating-promises": [
		"error",
		{ ignoreVoid: true, ignoreIIFE: true },
	],
	"@typescript-eslint/switch-exhaustiveness-check": "error",
	"@typescript-eslint/only-throw-error": "error",

	"@typescript-eslint/prefer-nullish-coalescing": [
		"error",
		{
			ignoreConditionalTests: true,
			ignoreMixedLogicalExpressions: true,
		},
	],
	"@typescript-eslint/restrict-template-expressions": [
		"error",
		{
			allowNumber: true,
			allowBoolean: true,
			allowNullish: true,
		},
	],

	"no-console": ["error", { allow: ["warn", "error"] }],
	"no-debugger": "error",
	"no-var": "error",
	"prefer-const": "error",
	eqeqeq: ["error", "always", { null: "ignore" }],
	curly: ["error", "all"],
	"no-duplicate-imports": "error",
};

export default tseslint.config(
	{
		ignores: [
			"build/**",
			".svelte-kit/**",
			"coverage/**",
			"dist/**",
			"storybook-static/**",
			"node_modules/**",
		],
	},

	{
		linterOptions: {
			reportUnusedDisableDirectives: "error",
		},
	},

	js.configs.recommended,
	...tseslint.configs.strictTypeChecked,
	...tseslint.configs.stylisticTypeChecked,
	...svelte.configs.recommended,

	{
		languageOptions: {
			globals: {
				...globals.browser,
				...globals.node,
			},
			parserOptions: {
				projectService: {
					allowDefaultProject: ["eslint.config.js", "svelte.config.js"],
				},
				tsconfigRootDir,
			},
		},
	},

	{
		files: ["**/*.svelte", "**/*.svelte.ts", "**/*.svelte.js"],
		languageOptions: {
			parserOptions: {
				projectService: {
					allowDefaultProject: ["eslint.config.js", "svelte.config.js"],
				},
				tsconfigRootDir,
				extraFileExtensions: [".svelte"],
				parser: tseslint.parser,
				svelteConfig,
			},
		},
	},
	{
		files: ["eslint.config.js"],
		rules: {
			"@typescript-eslint/no-deprecated": "off",
		},
	},

	{
		files: tsAndSvelteFiles,
		plugins: {
			"@typescript-eslint": tseslint.plugin,
			perfectionist,
		},
		rules: {
			...commonTypeRules,

			"no-unused-vars": "off",

			"svelte/no-navigation-without-resolve": "off",
			"svelte/require-each-key": "error",
			"svelte/no-at-html-tags": "error",
			"svelte/prefer-svelte-reactivity": "error",
		},
	},

	{
		files: testFiles,
		plugins: {
			vitest,
		},
		languageOptions: {
			globals: {
				...vitest.environments.env.globals,
			},
		},
		rules: {
			...vitest.configs.recommended.rules,

			"@typescript-eslint/unbound-method": "off",
			"@typescript-eslint/no-explicit-any": "off",
			"@typescript-eslint/no-non-null-assertion": "off",
			"@typescript-eslint/no-unsafe-assignment": "off",
			"@typescript-eslint/no-unsafe-member-access": "off",
			"@typescript-eslint/no-unsafe-call": "off",
			"@typescript-eslint/no-unsafe-argument": "off",
			"@typescript-eslint/no-unsafe-return": "off",

			"no-console": "off",

			"vitest/no-disabled-tests": "error",
			"vitest/no-focused-tests": "error",
		},
	},
);
