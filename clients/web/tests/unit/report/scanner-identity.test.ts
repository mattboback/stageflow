import {
	SCANNER_META,
	getScannerIconClass,
	getScannerTileClass
} from '$lib/report/scanner-identity';
import { describe, expect, it } from 'vitest';

describe('scanner-identity utilities', () => {
	describe('SCANNER_META', () => {
		it('contains entries for all known scanners', () => {
			const ids = Object.keys(SCANNER_META);
			expect(ids).toContain('axe');
			expect(ids).toContain('lighthouse');
			expect(ids).toContain('link-checker');
			expect(ids).toContain('security-headers');
			expect(ids).toContain('seo');
			expect(ids).toContain('ai-navigator');
			expect(ids).toContain('open-graph');
			expect(ids).toContain('spelling-grammar');
		});

		it('each entry has icon, label, and description', () => {
			for (const [, meta] of Object.entries(SCANNER_META)) {
				expect(meta.icon).toBeDefined();
				expect(meta.label).toBeTruthy();
				expect(meta.description).toBeTruthy();
			}
		});

		it('has no color fields', () => {
			for (const [, meta] of Object.entries(SCANNER_META)) {
				expect(meta).not.toHaveProperty('color');
				expect(meta).not.toHaveProperty('borderColor');
			}
		});
	});

	describe('getScannerTileClass', () => {
		it('returns teal accent classes when selected', () => {
			const cls = getScannerTileClass(true);
			expect(cls).toContain('border-accent');
			expect(cls).toContain('ring-accent');
		});

		it('returns neutral classes when not selected', () => {
			const cls = getScannerTileClass(false);
			expect(cls).toContain('border-line');
			expect(cls).toContain('bg-surface');
			expect(cls).not.toContain('ring-accent');
		});
	});

	describe('getScannerIconClass', () => {
		it('returns teal bg + white text when selected', () => {
			const cls = getScannerIconClass(true);
			expect(cls).toContain('bg-accent');
			expect(cls).toContain('text-white');
		});

		it('returns neutral muted classes when not selected', () => {
			const cls = getScannerIconClass(false);
			expect(cls).toContain('bg-surface-muted');
			expect(cls).toContain('text-ink-muted');
		});
	});
});
