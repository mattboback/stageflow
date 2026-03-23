import type { Preview } from '@storybook/svelte';

import '@fontsource-variable/jetbrains-mono';
import '@fontsource-variable/source-sans-3';
import '@fontsource-variable/source-serif-4';

import '../src/app.css';

const preview: Preview = {
	parameters: {
		layout: 'centered',
		options: {
			storySort: {
				order: ['Introduction', 'UI', 'Playground', 'Scan Status', 'Report']
			}
		},
		actions: {
			argTypesRegex: '^on[A-Z].*'
		},
		controls: {
			expanded: true,
			matchers: {
				color: /(background|color)$/i,
				date: /Date$/i
			}
		},
		docs: {
			toc: true
		},
		a11y: {
			test: 'error'
		}
	}
};

export default preview;
