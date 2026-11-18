import type { IssueDetail, IssueOccurrence } from '../types/unified-report';

import { getContrastData } from './contrast-verify';

/** One block of an axe failure summary: every check must pass, or any one of them. */
export interface FailureCheckGroup {
	mode: 'all' | 'any';
	checks: string[];
}

const FAILURE_GROUP_HEADER = /^fix (all|any) of the following:?\s*(.*)$/i;

/*
 * axe failure summaries arrive as machine output: a "Fix all of the following:"
 * or "Fix any of the following:" header, then one failed check per line, and
 * sometimes both blocks in one summary. Returns the blocks, or null when the
 * text does not have that shape (curated howToFix prose from other scanners).
 */
export function parseFailureSummary(
	summary: string | null | undefined
): FailureCheckGroup[] | null {
	if (!summary) return null;
	const groups: FailureCheckGroup[] = [];
	for (const raw of summary.split(/\n/)) {
		const line = raw.trim();
		if (!line) continue;
		const header = FAILURE_GROUP_HEADER.exec(line);
		if (header) {
			const mode = header[1]?.toLowerCase() === 'all' ? 'all' : 'any';
			const inline = header[2]?.trim();
			groups.push({ mode, checks: inline ? [inline] : [] });
			continue;
		}
		const current = groups[groups.length - 1];
		if (!current) return null;
		current.checks.push(line);
	}
	const nonEmpty = groups.filter((group) => group.checks.length > 0);
	return nonEmpty.length > 0 ? nonEmpty : null;
}

export function asSentence(line: string): string {
	return /[.!?]$/.test(line) ? line : `${line}.`;
}

/*
 * The lead-in for one block. An "any" block with a single check is just as
 * required as an "all" block, and "resolves it" only holds when the block
 * stands alone: link-name pairs a required check with a choice of four.
 */
export function failureGroupLabel(group: FailureCheckGroup, index: number, total: number): string {
	const first = index === 0;
	if (group.mode === 'any' && group.checks.length > 1) {
		if (total === 1) return 'Fixing any one of these resolves it:';
		return first ? 'One of these needs fixing:' : 'And one of these:';
	}
	if (group.checks.length === 1) return first ? 'This needs fixing:' : 'And this:';
	return first ? 'All of these need fixing:' : 'And all of these:';
}

/*
 * The same summary as plain sentences, for places that only have room for
 * text. Falls back to the raw text when the shape is unfamiliar.
 */
export function humanizeFailureSummary(summary: string | null | undefined): string | null {
	if (!summary) return null;
	const groups = parseFailureSummary(summary);
	if (!groups) {
		const text = summary
			.split(/\n+/)
			.map((line) => line.trim())
			.filter(Boolean);
		return text.length > 0 ? text.map(asSentence).join(' ') : null;
	}
	const only = groups.length === 1 ? groups[0] : undefined;
	if (only?.checks.length === 1) return asSentence(only.checks[0] ?? '');
	return groups
		.map(
			(group, index) =>
				`${failureGroupLabel(group, index, groups.length)} ${group.checks.map(asSentence).join(' ')}`
		)
		.join(' ');
}

/*
 * Rule-specific guidance first, then the scanner's own howToFix. Rule ids are
 * matched exactly: once this text wins over howToFix, a loose match such as
 * includes('label') would hand form-field-multiple-labels the wrong advice.
 */
export function generateContextualFix(
	issue: IssueDetail,
	occurrence: IssueOccurrence | null
): string | null {
	const ruleId = (issue.ruleId ?? '').toLowerCase();
	const html = occurrence?.html?.trim() ?? '';

	if (ruleId === 'image-alt') {
		const src = parseFirstElement(html)?.attr('src');
		if (src) {
			return `Add a descriptive alt attribute to the <img src="${truncate(src, 80)}"> element. If the image is decorative, set alt="" so assistive tech can skip it.`;
		}
		return 'Add a descriptive alt attribute to this image, or set alt="" if the image is decorative.';
	}

	if (ruleId === 'label' || ruleId === 'select-name') {
		const field = parseFirstElement(html);
		const id = field?.attr('id');
		const name = field?.attr('name');
		const tag = field?.tag;
		if (id) {
			return `Associate this ${tag ?? 'input'} with a <label for="${id}">…</label>, or wrap it in a <label> element. An aria-label is acceptable when a visible label is undesirable.`;
		}
		if (name) {
			return `This ${tag ?? 'input'} (name="${name}") needs an associated <label>. Add a matching id and reference it with <label for="…">, or wrap the field in a <label>.`;
		}
		return 'Add a <label> that references this field by id, wrap the field in a <label>, or apply an aria-label.';
	}

	if (ruleId.includes('color-contrast') || ruleId === 'contrast') {
		const data = getContrastData(issue);
		if (data?.contrastRatio && data?.expectedContrastRatio) {
			const fg = data.fgColor ? ` (foreground ${data.fgColor})` : '';
			const bg = data.bgColor ? ` against ${data.bgColor}` : '';
			return `Current contrast ratio is ${data.contrastRatio}:1${fg}${bg}. Adjust foreground or background to reach at least ${data.expectedContrastRatio}:1.`;
		}
		return 'Adjust foreground or background colors to meet WCAG contrast minimums (4.5:1 for normal text, 3:1 for large text).';
	}

	if (ruleId === 'document-title') {
		return 'Add a non-empty <title> element to the document <head>. It should concisely describe the page.';
	}

	if (ruleId === 'heading-order') {
		const tag = parseFirstElement(html)?.tag;
		if (tag && /^h[1-6]$/.test(tag)) {
			return `This <${tag}> skips a level in the heading hierarchy. Use the next sequential level (or restructure surrounding headings) so the outline stays well-formed.`;
		}
		return 'Use heading levels (h1–h6) sequentially without skipping levels so screen readers can navigate the document outline.';
	}

	if (ruleId === 'link-name') {
		const link = parseFirstElement(html);
		const href = link?.attr('href');
		const tag = link?.tag ?? 'link';
		if (href) {
			return `The <${tag} href="${truncate(href, 60)}"> has no accessible name. Add visible text content, an aria-label, or aria-labelledby so assistive tech can announce its purpose.`;
		}
		return 'Give this link an accessible name via visible text content, aria-label, or aria-labelledby.';
	}

	if (ruleId === 'button-name') {
		return 'Provide an accessible name for this button: add text content, an aria-label, or aria-labelledby.';
	}

	if (ruleId === 'html-has-lang' || ruleId === 'html-lang-valid') {
		return 'Set a valid lang attribute on the root <html> element (e.g., lang="en") so assistive tech can use correct pronunciation rules.';
	}

	if (ruleId === 'meta-viewport' || ruleId === 'meta-viewport-large') {
		return 'Remove user-scalable=no from the viewport meta tag and avoid maximum-scale values below 5 so users can zoom.';
	}

	return issue.howToFix?.trim() || null;
}

interface OpeningTag {
	tag: string;
	attr: (name: string) => string | null;
}

const OPENING_TAG =
	/^\s*<([a-z][\w-]*)((?:\s+[^\s"'>/=]+(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s"'=<>`]+))?)*)\s*\/?>/i;
const ATTRIBUTE = /([^\s"'>/=]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+)))?/g;
const NAMED_ENTITIES: Record<string, string> = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'" };

/*
 * The first element's tag and attributes, read from its opening tag. Not
 * DOMParser: this also runs while /demo is prerendered, where there is none,
 * and guidance that differed between the two broke hydration. Values are
 * decoded when read, so an attribute no rule asks for is never touched.
 */
function parseFirstElement(html: string): OpeningTag | null {
	const open = OPENING_TAG.exec(html);
	if (!open?.[1]) return null;
	const attributes = new Map<string, string>();
	for (const match of (open[2] ?? '').matchAll(ATTRIBUTE)) {
		const name = match[1]?.toLowerCase();
		if (name && !attributes.has(name)) {
			attributes.set(name, match[2] ?? match[3] ?? match[4] ?? '');
		}
	}
	return {
		tag: open[1].toLowerCase(),
		attr: (name) => {
			const raw = attributes.get(name);
			return raw === undefined ? null : decodeEntities(raw);
		}
	};
}

function decodeEntities(value: string): string {
	return value.replace(
		/&(?:#(\d+)|#x([\da-f]+)|([a-z]+));/gi,
		(entity, decimal?: string, hex?: string, named?: string) => {
			if (decimal) return fromCharacterReference(Number(decimal));
			if (hex) return fromCharacterReference(parseInt(hex, 16));
			return NAMED_ENTITIES[named?.toLowerCase() ?? ''] ?? entity;
		}
	);
}

/*
 * What a browser makes of a numeric reference. The markup is the scanned
 * site's, so &#99999999; or &#xD800; can arrive, and String.fromCodePoint
 * throws on both; the browser shows U+FFFD.
 */
function fromCharacterReference(codePoint: number): string {
	const valid =
		Number.isInteger(codePoint) &&
		codePoint > 0 &&
		codePoint <= 0x10ffff &&
		(codePoint < 0xd800 || codePoint > 0xdfff);
	return valid ? String.fromCodePoint(codePoint) : '\uFFFD';
}

function truncate(value: string, max: number): string {
	if (value.length <= max) return value;
	return `${value.slice(0, max - 1)}…`;
}
