import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

it('pins the image browser installer to the scanner Playwright runtime', () => {
	const manifest = JSON.parse(
		readFileSync(new URL('../../package.json', import.meta.url), 'utf8')
	) as { dependencies: { playwright: string; 'playwright-core': string } };
	const dockerfile = readFileSync(new URL('../../Dockerfile', import.meta.url), 'utf8');
	const browserVersion = /^ARG PLAYWRIGHT_VERSION=(.+)$/m.exec(dockerfile)?.[1];

	expect(browserVersion).toBe(manifest.dependencies.playwright);
	expect(manifest.dependencies['playwright-core']).toBe(manifest.dependencies.playwright);
});
