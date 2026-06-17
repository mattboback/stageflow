import { Bot, CheckCircle, Globe, Link2, Search, Shield, Zap } from 'lucide-react';

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
