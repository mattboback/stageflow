import type { Page } from "playwright";

/**
 * Configuration for context snippet extraction.
 */
export interface ContextSnippetConfig {
	/** Maximum total characters for contextHtml (default: 2000) */
	maxContextChars: number;
	/** Maximum ancestor depth to include (default: 5) */
	maxAncestorDepth: number;
	/** Maximum siblings before/after to include (default: 1) */
	maxSiblings: number;
	/** Maximum characters per sibling element (default: 200) */
	maxSiblingChars: number;
}

export const DEFAULT_CONTEXT_CONFIG: ContextSnippetConfig = {
	maxContextChars: 2000,
	maxAncestorDepth: 5,
	maxSiblings: 1,
	maxSiblingChars: 200,
};

/**
 * Result of context snippet extraction.
 */
export interface ContextSnippetResult {
	/** The element's outer HTML with surrounding context */
	contextHtml: string;
	/** Breadcrumb path of ancestors (e.g., "html > body > main > div.container") */
	ancestorPath: string;
}

/**
 * Extract DOM context around an element for debugging purposes.
 *
 * Returns:
 * - contextHtml: The element's HTML wrapped in its parent, with truncated siblings
 * - ancestorPath: Breadcrumb-style path showing the element's location in the DOM
 *
 * The extraction is bounded to prevent excessively large snippets:
 * - Limited ancestor depth
 * - Limited siblings (before/after the element)
 * - Truncated sibling content
 * - Overall character limit
 */
export async function extractContextSnippet(
	page: Page,
	selector: string,
	config: Partial<ContextSnippetConfig> = {},
): Promise<ContextSnippetResult | undefined> {
	const cfg: ContextSnippetConfig = { ...DEFAULT_CONTEXT_CONFIG, ...config };

	try {
		const locator = page.locator(selector).first();

		const result = await locator.evaluate(
			(el: Element, opts: ContextSnippetConfig) => {
				/**
				 * Build a short descriptor for an element (tag + id/class).
				 */
				function describeElement(elem: Element): string {
					const tag = elem.tagName.toLowerCase();
					const id = elem.id ? `#${elem.id}` : "";
					const classes =
						elem.className && typeof elem.className === "string"
							? elem.className
									.split(/\s+/)
									.filter(Boolean)
									.slice(0, 2)
									.map((c) => `.${c}`)
									.join("")
							: "";
					return `${tag}${id}${classes}`;
				}

				/**
				 * Build the ancestor path breadcrumb.
				 */
				function buildAncestorPath(elem: Element, maxDepth: number): string {
					const parts: string[] = [];
					let current: Element | null = elem;
					let depth = 0;

					while (current && depth < maxDepth) {
						parts.unshift(describeElement(current));
						current = current.parentElement;
						depth++;
					}

					// If we didn't reach the root, prepend "..."
					if (current && current !== document.documentElement) {
						parts.unshift("...");
					}

					return parts.join(" > ");
				}

				/**
				 * Truncate HTML string to a max length, trying to close tags gracefully.
				 */
				function truncateHtml(html: string, maxLen: number): string {
					if (html.length <= maxLen) {
						return html;
					}
					// Simple truncation - just cut and add ellipsis
					// A more sophisticated approach would try to close open tags
					const truncated = html.slice(0, maxLen - 3);
					// Try to avoid cutting in the middle of a tag
					const lastOpenBracket = truncated.lastIndexOf("<");
					const lastCloseBracket = truncated.lastIndexOf(">");
					if (lastOpenBracket > lastCloseBracket) {
						// We're in the middle of a tag, cut before it
						return truncated.slice(0, lastOpenBracket) + "...";
					}
					return truncated + "...";
				}

				/**
				 * Get a truncated outer HTML representation of an element.
				 */
				function getTruncatedHtml(elem: Element, maxLen: number): string {
					const html = elem.outerHTML;
					return truncateHtml(html, maxLen);
				}

				/**
				 * Build context HTML with the element, its parent wrapper, and limited siblings.
				 */
				function buildContextHtml(
					elem: Element,
					maxSiblings: number,
					maxSiblingChars: number,
					maxTotal: number,
				): string {
					const parent = elem.parentElement;
					if (!parent) {
						// No parent - just return element's HTML
						return truncateHtml(elem.outerHTML, maxTotal);
					}

					// Get siblings
					const siblings = Array.from(parent.children);
					const elemIndex = siblings.indexOf(elem);

					// Calculate range of siblings to include
					const startIdx = Math.max(0, elemIndex - maxSiblings);
					const endIdx = Math.min(siblings.length - 1, elemIndex + maxSiblings);

					// Build the content pieces
					const pieces: string[] = [];
					let totalLen = 0;

					// Opening tag of parent
					const parentTag = parent.tagName.toLowerCase();
					const parentId = parent.id ? ` id="${parent.id}"` : "";
					const parentClass =
						parent.className && typeof parent.className === "string"
							? ` class="${parent.className.split(/\s+/).slice(0, 3).join(" ")}"`
							: "";
					const openTag = `<${parentTag}${parentId}${parentClass}>`;
					const closeTag = `</${parentTag}>`;
					totalLen += openTag.length + closeTag.length + 10; // Reserve space for tags and ellipsis

					// Add ellipsis if we're not starting at the beginning
					if (startIdx > 0) {
						pieces.push("  <!-- ... -->");
						totalLen += 15;
					}

					// Add siblings and the element
					for (let i = startIdx; i <= endIdx && totalLen < maxTotal; i++) {
						const sibling = siblings[i];
						if (!sibling) {
							continue;
						}
						if (sibling === elem) {
							// This is our target element - include it fully (or as much as fits)
							const remainingSpace = maxTotal - totalLen - 100; // Reserve space for closing
							const elemHtml = truncateHtml(
								elem.outerHTML,
								Math.max(500, remainingSpace),
							);
							pieces.push(`  ${elemHtml} <!-- TARGET -->`);
							totalLen += elemHtml.length + 20;
						} else {
							// This is a sibling - truncate it aggressively
							const siblingHtml = getTruncatedHtml(sibling, maxSiblingChars);
							if (totalLen + siblingHtml.length + 10 < maxTotal) {
								pieces.push(`  ${siblingHtml}`);
								totalLen += siblingHtml.length + 2;
							}
						}
					}

					// Add ellipsis if we're not ending at the end
					if (endIdx < siblings.length - 1) {
						pieces.push("  <!-- ... -->");
					}

					return `${openTag}\n${pieces.join("\n")}\n${closeTag}`;
				}

				// Build the ancestor path
				const ancestorPath = buildAncestorPath(el, opts.maxAncestorDepth);

				// Build the context HTML
				const contextHtml = buildContextHtml(
					el,
					opts.maxSiblings,
					opts.maxSiblingChars,
					opts.maxContextChars,
				);

				return { contextHtml, ancestorPath };
			},
			cfg,
		);

		return result;
	} catch {
		return undefined;
	}
}
