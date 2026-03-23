import type { Page } from 'playwright';

import type { PreScanAction } from '../core/types';
import type { AgentGoal, InteractiveElement, PagePerception, SuggestedAction } from './types';
import type { VisionClient } from './vision-client';

import { parseFirstJsonObject } from './json';

export class PageAnalyzer {
	constructor(private readonly visionClient: VisionClient) {}

	async analyze(page: Page, goal?: AgentGoal): Promise<PagePerception> {
		const interactiveElements = await this.extractInteractiveElements(page);
		const url = page.url();
		const title = await page.title();
		const screenshot = await page.screenshot({ type: 'png' });

		const prompt = this.buildAnalysisPrompt(url, title, interactiveElements, goal);
		const response = await this.visionClient.analyze(screenshot, prompt);

		return this.parseAnalysisResponse(response.content, url, title, interactiveElements);
	}

	private async extractInteractiveElements(page: Page): Promise<InteractiveElement[]> {
		const elements = await page.evaluate(() => {
			const interactiveSelectors = [
				'a[href]',
				'button',
				'input',
				'select',
				'textarea',
				'[role="button"]',
				'[role="link"]',
				'[onclick]',
				'[tabindex]:not([tabindex="-1"])'
			];

			const maxElements = 75;
			const results: {
				selector: string;
				tagName: string;
				role?: string;
				accessibleName?: string;
				text?: string;
				isVisible: boolean;
				isEnabled: boolean;
				boundingBox?: { x: number; y: number; width: number; height: number };
			}[] = [];

			const seen = new Set<Element>();

			const safeSnippet = (value: string, maxLen: number): string => {
				const cleaned = value.replace(/\s+/g, ' ').trim();
				return cleaned.length > maxLen ? cleaned.slice(0, maxLen) : cleaned;
			};

			const escapeSelectorText = (value: string): string => {
				return value.replaceAll('\\', '\\\\').replaceAll('"', '\\"');
			};

			const makeSelector = (el: Element): string => {
				const element = el as HTMLElement;

				if (element.id) {
					return `#${element.id}`;
				}

				const tag = element.tagName.toLowerCase();
				const classes = Array.from(element.classList).slice(0, 2).join('.');
				const base = classes ? `${tag}.${classes}` : tag;

				const text = safeSnippet(element.textContent || '', 20);
				if (!text) {
					return base;
				}

				return `${base}:has-text("${escapeSelectorText(text)}")`;
			};

			for (const selector of interactiveSelectors) {
				for (const el of document.querySelectorAll(selector)) {
					if (results.length >= maxElements) {
						return results;
					}

					if (seen.has(el)) {
						continue;
					}

					seen.add(el);

					const rect = (el as HTMLElement).getBoundingClientRect();
					const style = window.getComputedStyle(el as HTMLElement);
					const isVisible =
						rect.width > 0 &&
						rect.height > 0 &&
						style.visibility !== 'hidden' &&
						style.display !== 'none';

					if (!isVisible) {
						continue;
					}

					const tagName = (el as HTMLElement).tagName.toLowerCase();
					const role = (el as HTMLElement).getAttribute('role')?.trim() ?? undefined;
					const ariaLabel = (el as HTMLElement).getAttribute('aria-label')?.trim() ?? undefined;
					const titleAttr = (el as HTMLElement).getAttribute('title')?.trim() ?? undefined;
					const placeholder = (el as HTMLInputElement).placeholder || undefined;
					const text = safeSnippet((el as HTMLElement).textContent || '', 100) || undefined;

					const accessibleName =
						ariaLabel ?? titleAttr ?? placeholder ?? (text ? safeSnippet(text, 50) : undefined);

					const isEnabled = !(el as HTMLInputElement).disabled;
					const selectorStr = makeSelector(el);

					results.push({
						selector: selectorStr,
						tagName,
						isVisible: true,
						isEnabled,
						boundingBox: {
							x: Math.round(rect.x),
							y: Math.round(rect.y),
							width: Math.round(rect.width),
							height: Math.round(rect.height)
						},
						...(role !== undefined ? { role } : {}),
						...(accessibleName !== undefined ? { accessibleName } : {}),
						...(text !== undefined ? { text } : {})
					});
				}
			}

			return results;
		});

		return elements.map((el) => ({
			selector: el.selector,
			tagName: el.tagName,
			isVisible: el.isVisible,
			isEnabled: el.isEnabled,
			...(el.role !== undefined ? { role: el.role } : {}),
			...(el.accessibleName !== undefined ? { accessibleName: el.accessibleName } : {}),
			...(el.text !== undefined ? { text: el.text } : {}),
			...(el.boundingBox !== undefined ? { boundingBox: el.boundingBox } : {})
		}));
	}

	private buildAnalysisPrompt(
		url: string,
		title: string,
		elements: InteractiveElement[],
		goal?: AgentGoal
	): string {
		const elementList = elements
			.slice(0, 25)
			.map((e, i) => `${i + 1}. [${e.tagName}] ${e.accessibleName ?? e.text ?? e.selector}`)
			.join('\n');

		const goalSection = goal
			? `\n\nGoal: "${goal.objective}"\nInclude "goalRelevance": 0.0-1.0 in your JSON response.`
			: '';

		return [
			'Analyze this webpage screenshot.',
			'',
			`URL: ${url}`,
			`Title: ${title}`,
			'',
			'Interactive elements found:',
			elementList || '(none)',
			'',
			'Respond with JSON:',
			'{',
			'  "pageType": "homepage|product|article|form|login|search|cart|checkout|error|other",',
			'  "description": "Brief description of what this page is",',
			'  "suggestedActions": [',
			'    {',
			'      "elementIndex": 1,',
			'      "action": "click|fill|select",',
			'      "value": "optional value for fill/select",',
			'      "reasoning": "why this action"',
			'    }',
			'  ]',
			'}',
			goalSection
		].join('\n');
	}

	private parseAnalysisResponse(
		response: string,
		url: string,
		title: string,
		elements: InteractiveElement[]
	): PagePerception {
		const parsed = parseFirstJsonObject(response);
		if (!parsed) {
			return {
				url,
				title,
				pageType: 'other',
				description: '',
				interactiveElements: elements
			};
		}

		const pageType = typeof parsed.pageType === 'string' ? parsed.pageType : 'other';
		const description = typeof parsed.description === 'string' ? parsed.description : '';
		const goalRelevance =
			typeof parsed.goalRelevance === 'number' ? parsed.goalRelevance : undefined;

		const suggestedActions = this.parseSuggestedActions(parsed.suggestedActions, elements);

		return {
			url,
			title,
			pageType,
			description,
			interactiveElements: elements,
			...(goalRelevance !== undefined ? { goalRelevance } : {}),
			...(suggestedActions.length > 0 ? { suggestedActions } : {})
		};
	}

	private parseSuggestedActions(raw: unknown, elements: InteractiveElement[]): SuggestedAction[] {
		if (!Array.isArray(raw)) {
			return [];
		}

		const actions: SuggestedAction[] = [];

		for (const item of raw) {
			if (!item || typeof item !== 'object' || Array.isArray(item)) {
				continue;
			}

			const record = item as Record<string, unknown>;
			const index = record.elementIndex;
			if (typeof index !== 'number' || index < 1 || index > elements.length) {
				continue;
			}

			const element = elements[index - 1];
			if (!element) {
				continue;
			}

			const kind = record.action;
			if (typeof kind !== 'string') {
				continue;
			}

			const action = this.toPreScanAction(kind, element.selector, record.value);
			if (!action) {
				continue;
			}

			const reasoning = typeof record.reasoning === 'string' ? record.reasoning : '';
			const confidence = typeof record.confidence === 'number' ? record.confidence : 0.7;

			actions.push({ action, reasoning, confidence });
		}

		return actions;
	}

	private toPreScanAction(
		kind: string,
		selector: string,
		value: unknown
	): PreScanAction | undefined {
		if (kind === 'click') {
			return { type: 'click', selector };
		}

		if (kind === 'fill') {
			const rawValue = typeof value === 'string' ? value : '';
			return { type: 'fill', selector, value: rawValue };
		}

		if (kind === 'select') {
			const rawValue = typeof value === 'string' ? value : '';
			return { type: 'select', selector, value: rawValue };
		}

		return undefined;
	}
}
