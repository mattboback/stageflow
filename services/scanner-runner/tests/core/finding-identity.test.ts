import { describe, expect, it } from 'vitest';
import {
	hasMultipleOrigins,
	stablePageId,
	issueFingerprint,
	occurrenceFingerprint
} from '../../src/core/finding-identity';
import { collectPageOverviewTargets } from '../../src/screenshots/axe/page-overview';

describe('finding identity', () => {
	it('compares deployments by case-sensitive path and query, ignoring fragments', () => {
		expect(stablePageId('https://preview.example/About?q=1#top')).toBe(
			stablePageId('https://live.example/About?q=1')
		);
		expect(stablePageId('https://live.example/About')).not.toBe(
			stablePageId('https://live.example/about')
		);
		expect(stablePageId('https://live.example/about?q=1')).not.toBe(
			stablePageId('https://live.example/about?q=2')
		);
	});
	it('keeps origins when a scan spans domains and stays independent of URL order', () => {
		const urls = ['https://a.example/about', 'https://b.example/about'];
		expect(hasMultipleOrigins(urls)).toBe(true);
		expect(urls.map((url) => stablePageId(url, hasMultipleOrigins(urls)))).toEqual(
			[...urls]
				.reverse()
				.map((url) => stablePageId(url, hasMultipleOrigins(urls)))
				.reverse()
		);
		expect(stablePageId(urls[0]!, true)).not.toBe(stablePageId(urls[1]!, true));
	});
	it('uses identical case-sensitive fingerprints for report issues and screenshot markers', () => {
		const [marker] = collectPageOverviewTargets(
			[{ id: 'image-alt', impact: 'serious', nodes: [{ target: ['#Logo'] }] }],
			5,
			{ scanner: 'axe', pageId: 'page-v2-root' }
		);
		expect(marker?.issueId).toBe(issueFingerprint('axe', 'image-alt', 'page-v2-root', '#Logo'));
		expect(issueFingerprint('axe', 'image-alt', 'page-v2-root', '#Logo')).not.toBe(
			issueFingerprint('axe', 'image-alt', 'page-v2-root', '#logo')
		);
		expect(occurrenceFingerprint('axe', 'image-alt', 'page-v2-root', ['#Logo'])).not.toBe(
			occurrenceFingerprint('axe', 'image-alt', 'page-v2-root', ['#logo'])
		);
	});
});
