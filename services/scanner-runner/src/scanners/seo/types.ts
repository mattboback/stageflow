import type { IssueSeverity } from "../../core/types";

export interface SEOCheck {
	id: string;
	title: string;
	check: (data: PageSEOData) => {
		passed: boolean;
		message: string;
		details?: Record<string, unknown>;
	} | null;
	severity: IssueSeverity;
	category: string;
	helpUrl?: string;
}

export interface PageSEOData {
	title: string | null;
	description: string | null;
	canonical: string | null;
	robots: string | null;
	viewport: string | null;
	charset: string | null;
	language: string | null;
	headings: HeadingData[];
	images: ImageData[];
	links: LinkData[];
	ogTags: Record<string, string>;
	twitterTags: Record<string, string>;
	structuredData: unknown[];
	wordCount: number;
	url: string;
}

export interface HeadingData {
	level: number;
	text: string;
}

export interface ImageData {
	src: string;
	alt: string | null;
	width: number | null;
	height: number | null;
}

export interface LinkData {
	href: string;
	text: string;
	isInternal: boolean;
	hasNofollow: boolean;
}
