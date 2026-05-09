// @ts-check
import eslint from "@eslint/js";
import vitest from "@vitest/eslint-plugin";
import { defineConfig } from "eslint/config";
import perfectionist from "eslint-plugin-perfectionist";
import globals from "globals";
import tseslint from "typescript-eslint";

const typedFiles = ["src/**/*.ts", "tests/**/*.ts", "vitest.config.ts"];

const testFiles = [
	"tests/**/*.{ts,tsx}",
	"**/*.{test,spec}.{ts,tsx}",
	"**/__tests__/**/*.{ts,tsx}",
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
	"@typescript-eslint/prefer-nullish-coalescing": "error",
	"@typescript-eslint/non-nullable-type-assertion-style": "error",

	"@typescript-eslint/no-floating-promises": [
		"error",
		{ ignoreVoid: true, ignoreIIFE: true },
	],
	"@typescript-eslint/switch-exhaustiveness-check": "error",
	"@typescript-eslint/only-throw-error": "error",

	"@typescript-eslint/restrict-template-expressions": [
		"error",
		{ allowNumber: true, allowBoolean: true, allowNullish: true },
	],

	"@typescript-eslint/naming-convention": [
		"error",
		{
			selector: "default",
			format: ["camelCase", "PascalCase"],
			leadingUnderscore: "allow",
		},
		{
			selector: "variable",
			format: ["camelCase", "UPPER_CASE", "PascalCase"],
			leadingUnderscore: "allow",
		},
		{
			selector: "parameter",
			format: ["camelCase"],
			leadingUnderscore: "allow",
		},
		{ selector: "typeLike", format: ["PascalCase"] },
		{ selector: "enumMember", format: ["UPPER_CASE", "PascalCase"] },
		{ selector: "property", format: null },
	],

	"no-console": "off",
	"no-debugger": "error",
	"no-var": "error",
	"prefer-const": "error",
	eqeqeq: ["error", "always", { null: "ignore" }],
	curly: ["error", "all"],
	"no-duplicate-imports": "error",
	"no-eval": "error",
	"no-implied-eval": "error",
	"no-throw-literal": "error",
	"prefer-promise-reject-errors": "error",

	complexity: ["warn", 40],
	"max-depth": ["warn", 5],
	"max-nested-callbacks": ["warn", 4],
};

export default defineConfig(
	{
		ignores: ["dist/**", "node_modules/**", "coverage/**"],
	},

	{
		linterOptions: {
			reportUnusedDisableDirectives: "error",
			reportUnusedInlineConfigs: "error",
		},
	},

	eslint.configs.recommended,
	...tseslint.configs.strictTypeChecked,
	...tseslint.configs.stylisticTypeChecked,

	{
		languageOptions: {
			globals: {
				...globals.node,
			},
			parserOptions: {
				projectService: {
					allowDefaultProject: ["eslint.config.mjs", "vitest.config.ts"],
				},
				tsconfigRootDir: import.meta.dirname,
			},
		},
	},

	{
		files: typedFiles,
		plugins: {
			"@typescript-eslint": tseslint.plugin,
			perfectionist,
		},
		rules: commonTypeRules,
	},
	{
		files: ["eslint.config.mjs"],
		rules: {
			"@typescript-eslint/no-deprecated": "off",
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

			"vitest/no-disabled-tests": "error",
			"vitest/no-focused-tests": "error",
		},
	},
);
