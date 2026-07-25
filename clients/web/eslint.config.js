import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';
import { defineConfig, globalIgnores } from 'eslint/config';

/**
 * Type-aware linting, matching services/scanner-runner's posture.
 *
 * `recommendedTypeChecked` rather than `strictTypeChecked`: the strict preset adds
 * ~325 further reports here, and roughly 190 of those are no-confusing-void-expression
 * plus restrict-template-expressions — stylistic rules whose flagged pattern
 * (`onClick={() => setOpen(true)}`) is idiomatic React. Adopting it would mean a
 * wall of rule disables, which reads worse than the honest smaller rule set. The
 * few genuinely load-bearing strict rules are cherry-picked below instead.
 */
export default defineConfig([
	globalIgnores(['dist', 'build', 'coverage', '.react-router', 'node_modules', 'test-results']),

	// Type-aware pass over application, test, and Playwright sources.
	{
		files: ['**/*.{ts,tsx}'],
		extends: [
			js.configs.recommended,
			tseslint.configs.recommendedTypeChecked,
			reactHooks.configs.flat.recommended,
			reactRefresh.configs.vite
		],
		languageOptions: {
			globals: globals.browser,
			parserOptions: {
				projectService: true,
				tsconfigRootDir: import.meta.dirname
			}
		},
		rules: {
			'react-refresh/only-export-components': [
				'error',
				{
					allowConstantExport: true,
					allowExportNames: ['meta', 'links', 'headers', 'loader', 'action']
				}
			],

			// Cherry-picked from strictTypeChecked. Each one catches a defect class
			// rather than a formatting preference.
			'@typescript-eslint/no-non-null-assertion': 'error',
			'@typescript-eslint/use-unknown-in-catch-callback-variable': 'error',
			'@typescript-eslint/no-deprecated': 'error'
		}
	},

	// Plain JS/MJS: the build and QA helper scripts. These were previously linted
	// by nothing at all, because every rule block was scoped to ts/tsx.
	{
		files: ['**/*.{js,mjs}'],
		extends: [js.configs.recommended],
		languageOptions: {
			globals: { ...globals.node },
			sourceType: 'module',
			ecmaVersion: 'latest'
		}
	},

	// The QA capture scripts drive Playwright, so their page.evaluate callbacks are
	// serialized into the browser and legitimately reference document/window.
	{
		files: ['qa/**/*.{js,mjs}'],
		languageOptions: {
			globals: { ...globals.node, ...globals.browser }
		}
	},

	// Test files may assert an element exists with `!` instead of restating the
	// guard: if the assumption is wrong the test throws, which is the intended
	// outcome. Product code keeps the rule, matching how services/scanner-runner
	// relaxes a small set of rules for its own tests.
	{
		files: ['**/*.{test,spec}.{ts,tsx}', 'e2e/**/*.ts'],
		rules: {
			'@typescript-eslint/no-non-null-assertion': 'off'
		}
	}
]);
