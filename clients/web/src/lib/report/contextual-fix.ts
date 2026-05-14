import type { IssueDetail, IssueOccurrence } from '$lib/types/unified-report';

/**
 * Generate a rule-specific, context-aware fix instruction for an issue occurrence.
 *
 * For known rule IDs (image-alt, label, color-contrast, document-title,
 * heading-order, link-name), inspect the occurrence's HTML snippet and produce
 * a sharper instruction than the generic `issue.howToFix` text.
 *
 * Falls back to `issue.howToFix` when no rule-specific generator matches or
 * when DOM parsing is unavailable (e.g., SSR).
 */
export function generateContextualFix(
	issue: IssueDetail,
	occurrence: IssueOccurrence | null
): string | null {
	const ruleId = (issue.ruleId ?? '').toLowerCase();
	const html = occurrence?.html?.trim() ?? '';

	const element = parseFirstElement(html);

	if (ruleId.includes('image-alt')) {
		const src = element?.getAttribute('src');
		if (src) {
			return `Add a descriptive alt attribute to the <img src="${truncate(src, 80)}"> element. If the image is decorative, set alt="" so assistive tech can skip it.`;
		}
		return 'Add a descriptive alt attribute to this image, or set alt="" if the image is decorative.';
	}

	if (ruleId.includes('label') && !ruleId.includes('link') && !ruleId.includes('aria')) {
		const id = element?.getAttribute('id');
		const name = element?.getAttribute('name');
		const tag = element?.tagName?.toLowerCase();
		if (id) {
			return `Associate this ${tag ?? 'input'} with a <label for="${id}">…</label>, or wrap it in a <label> element. An aria-label is acceptable when a visible label is undesirable.`;
		}
		if (name) {
			return `This ${tag ?? 'input'} (name="${name}") needs an associated <label>. Add a matching id and reference it with <label for="…">, or wrap the field in a <label>.`;
		}
		return 'Add a <label> that references this field by id, wrap the field in a <label>, or apply an aria-label.';
	}

	if (ruleId.includes('color-contrast') || ruleId === 'contrast') {
		const raw = occurrence as (IssueOccurrence & { scannerData?: unknown }) | null;
		const data = (raw?.scannerData ?? null) as {
			fgColor?: string;
			bgColor?: string;
			contrastRatio?: number | string;
			expectedContrastRatio?: number | string;
		} | null;
		if (data?.contrastRatio && data?.expectedContrastRatio) {
			const fg = data.fgColor ? ` (foreground ${data.fgColor})` : '';
			const bg = data.bgColor ? ` against ${data.bgColor}` : '';
			return `Current contrast ratio is ${data.contrastRatio}:1${fg}${bg}. Adjust foreground or background to reach at least ${data.expectedContrastRatio}:1.`;
		}
		return 'Adjust foreground or background colors to meet WCAG contrast minimums (4.5:1 for normal text, 3:1 for large text).';
	}

	if (ruleId.includes('document-title')) {
		return 'Add a non-empty <title> element to the document <head>. It should concisely describe the page.';
	}

	if (ruleId.includes('heading-order')) {
		const tag = element?.tagName?.toLowerCase();
		if (tag && /^h[1-6]$/.test(tag)) {
			return `This <${tag}> skips a level in the heading hierarchy. Use the next sequential level (or restructure surrounding headings) so the outline stays well-formed.`;
		}
		return 'Use heading levels (h1–h6) sequentially without skipping levels so screen readers can navigate the document outline.';
	}

	if (ruleId.includes('link-name')) {
		const href = element?.getAttribute('href');
		const tag = element?.tagName?.toLowerCase() ?? 'link';
		if (href) {
			return `The <${tag} href="${truncate(href, 60)}"> has no accessible name. Add visible text content, an aria-label, or aria-labelledby so assistive tech can announce its purpose.`;
		}
		return 'Give this link an accessible name via visible text content, aria-label, or aria-labelledby.';
	}

	if (ruleId.includes('button-name')) {
		return 'Provide an accessible name for this button: add text content, an aria-label, or aria-labelledby.';
	}

	if (ruleId.includes('html-has-lang') || ruleId.includes('html-lang')) {
		return 'Set a valid lang attribute on the root <html> element (e.g., lang="en") so assistive tech can use correct pronunciation rules.';
	}

	if (ruleId.includes('meta-viewport')) {
		return 'Remove user-scalable=no from the viewport meta tag and avoid maximum-scale values below 5 so users can zoom.';
	}

	return issue.howToFix?.trim() || null;
}

function parseFirstElement(html: string): Element | null {
	if (!html) return null;
	if (typeof DOMParser === 'undefined') return null;
	try {
		const doc = new DOMParser().parseFromString(html, 'text/html');
		return doc.body.firstElementChild ?? null;
	} catch {
		return null;
	}
}

function truncate(value: string, max: number): string {
	if (value.length <= max) return value;
	return `${value.slice(0, max - 1)}…`;
}
