import { Bot, CheckCircle, Globe, Link2, Search, Shield, Zap } from 'lucide-svelte';

export interface ScannerMeta {
	icon: typeof Shield;
	label: string;
	description: string;
}

export const SCANNER_META: Record<string, ScannerMeta> = {
	axe: {
		icon: Shield,
		label: 'Axe',
		description: 'WCAG accessibility testing with axe-core engine'
	},
	lighthouse: {
		icon: Zap,
		label: 'Lighthouse',
		description: 'Performance, SEO, and best practices audits'
	},
	'link-checker': {
		icon: Link2,
		label: 'Link Checker',
		description: 'Detect broken links and redirect chains'
	},
	'security-headers': {
		icon: Shield,
		label: 'Security Headers',
		description: 'HTTP security header analysis and scoring'
	},
	seo: {
		icon: Search,
		label: 'SEO',
		description: 'Meta tags, headings, and SEO optimization'
	},
	'ai-navigator': {
		icon: Bot,
		label: 'AI Navigator',
		description: 'LLM-powered agent that navigates toward a goal'
	},
	'open-graph': {
		icon: Globe,
		label: 'Open Graph',
		description: 'Social preview metadata validation for OG tags and Twitter cards'
	},
	'spelling-grammar': {
		icon: CheckCircle,
		label: 'Spelling & Grammar',
		description: 'AI-powered spelling, grammar, and content quality checks'
	}
};

/**
 * Canonical display label for a scanner id (e.g. `seo` → "SEO",
 * `spelling-grammar` → "Spelling & Grammar"). Falls back to a title-cased slug
 * so newly-added scanners still render a sensible name without a client release.
 */
export function getScannerLabel(scannerId: string): string {
	const meta = SCANNER_META[scannerId];
	if (meta) return meta.label;
	return scannerId
		.split('-')
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(' ');
}

/**
 * Returns Tailwind classes for a scanner tile.
 * Neutral slate/white base; teal accent ring when selected.
 */
export function getScannerTileClass(isSelected: boolean): string {
	if (isSelected) {
		return 'border-accent/60 bg-accent/5 ring-2 ring-accent/20';
	}
	return 'border-line bg-surface hover:border-accent/30 hover:bg-surface-muted';
}

/**
 * Returns Tailwind classes for a scanner icon container.
 * Teal when selected, neutral slate otherwise.
 */
export function getScannerIconClass(isSelected: boolean): string {
	if (isSelected) {
		return 'bg-accent text-white';
	}
	return 'bg-surface-muted text-ink-muted';
}
