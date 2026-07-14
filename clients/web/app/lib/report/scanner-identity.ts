import { Bot, CheckCircle, Globe, Link2, Search, Shield, Zap } from 'lucide-react';

export interface ScannerMeta {
	icon: typeof Shield;
	label: string;
	description: string;
}

export const SCANNER_META: Record<string, ScannerMeta> = {
	axe: {
		icon: Shield,
		label: 'Axe Accessibility',
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
 * Canonical user-facing scanner name. Prefers the SCANNER_META label so the
 * homepage, configure page, and report all say the same thing; falls back to
 * the API-provided name with any trailing " Scanner" suffix dropped.
 */
export function scannerLabel(id: string, apiName?: string | null): string {
	const meta = SCANNER_META[id];
	if (meta) return meta.label;
	if (apiName) return apiName.replace(/\s+Scanner$/i, '');
	return id;
}

export function getScannerTileClass(isSelected: boolean): string {
	if (isSelected) {
		return 'scanner-tile selected';
	}
	return 'scanner-tile';
}

export function getScannerIconClass(isSelected: boolean): string {
	if (isSelected) {
		return 'scanner-icon-container selected';
	}
	return 'scanner-icon-container';
}
