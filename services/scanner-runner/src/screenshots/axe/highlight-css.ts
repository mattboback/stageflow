import type { Page } from "playwright";

import type { AxeScreenshotConfig } from "./types";

import { normalizeSelectors } from "./targets";

export function buildHighlightCSSRules(
	selectors: string[],
	cfg: Pick<AxeScreenshotConfig, "highlightStyle">,
): string | null {
	const cleanSelectors = normalizeSelectors(selectors);
	if (!cleanSelectors.length) {
		return null;
	}

	let outline: string;
	let shadow: string;
	let bg: string;

	if (cfg.highlightStyle === "dashed") {
		outline = "4px dashed #ff2d55";
		shadow =
			"0 0 0 4px rgba(255,45,85,0.35) inset, 0 0 0 4px rgba(255,45,85,0.3)";
		bg = "rgba(255,45,85,0.1)";
	} else {
		outline = "5px solid #ff0000";
		shadow = "0 0 0 4px rgba(255, 0, 0, 0.4)";
		bg = "rgba(255, 0, 0, 0.1)";
	}

	return cleanSelectors
		.map(
			(sel) =>
				`${sel} { outline: ${outline} !important; outline-offset: 2px !important; box-shadow: ${shadow} !important; background-color: ${bg} !important; position: relative !important; z-index: 2147483647 !important; }`,
		)
		.join("\n");
}

export async function injectHighlightCSS(
	page: Page,
	selectors: string[],
	cfg: Pick<AxeScreenshotConfig, "highlightStyle">,
): Promise<void> {
	const validSelectors = await filterValidSelectors(page, selectors);
	const rules = buildHighlightCSSRules(validSelectors, cfg);
	if (!rules) {
		return;
	}

	const script = `
      (() => {
        const styleId = 'a11y-axe-highlight-style';
        let s = document.getElementById(styleId);
        if (s) s.remove();
        s = document.createElement('style');
        s.id = styleId;
        s.textContent = \`${rules}\`;
        document.head.appendChild(s);
      })();
    `;

	try {
		await page.evaluate(script);
		// Wait for styles to apply and paint.
		await page.waitForTimeout(100);
	} catch {
		// ignore
	}
}

async function filterValidSelectors(
	page: Page,
	selectors: string[],
): Promise<string[]> {
	try {
		return await page.evaluate((rawSelectors) => {
			if (!Array.isArray(rawSelectors)) {
				return [];
			}

			const seen = new Set<string>();
			const valid: string[] = [];

			for (const raw of rawSelectors) {
				if (typeof raw !== "string") {
					continue;
				}
				const selector = raw.trim();
				if (!selector || seen.has(selector)) {
					continue;
				}
				try {
					document.querySelector(selector);
					valid.push(selector);
					seen.add(selector);
				} catch {
					// Ignore invalid selectors that would break CSS injection.
				}
			}

			return valid;
		}, selectors);
	} catch {
		return normalizeSelectors(selectors);
	}
}

export async function removeHighlightCSS(page: Page): Promise<void> {
	const script = `
      (() => {
        const s = document.getElementById('a11y-axe-highlight-style');
        if (s) s.remove();
      })();
    `;

	try {
		await page.evaluate(script);
	} catch {
		// ignore
	}
}
