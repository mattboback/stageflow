import type { Page } from 'playwright';

import type { FriendlyNodeInfo } from './types';

/**
 * Build a human-readable description of an element.
 */
export async function buildFriendlyNodeInfo(
	page: Page,
	selector: string
): Promise<FriendlyNodeInfo | undefined> {
	try {
		const locator = page.locator(selector).first();
		const info = await locator.evaluate((el: Element) => {
			const tagName = el.tagName.toLowerCase();
			const role = el.getAttribute('role')?.trim() ?? undefined;

			// Compute accessible name approximation.
			const ariaLabel =
				el.getAttribute('aria-label')?.trim() ?? el.getAttribute('aria-labelledby')?.trim() ?? '';
			const alt = (el as HTMLImageElement).alt.trim();
			const title = el.getAttribute('title')?.trim() ?? '';
			const placeholder = (el as HTMLInputElement).placeholder.trim();
			const textContent = (el.textContent || '').trim();

			// Use first non-empty name source, truncated.
			const rawName =
				ariaLabel || alt || title || placeholder || textContent.split(/\s+/).slice(0, 8).join(' ');
			const name = rawName.length > 60 ? `${rawName.slice(0, 57)}...` : rawName;

			// Find nearest landmark/section for context.
			const regionEl =
				el.closest('main, header, footer, nav, aside, section, article, form, dialog') ??
				document.body;
			let region: string | undefined;
			if (regionEl === document.body) {
				region = 'Page body';
			} else {
				const regionTag = regionEl.tagName.toLowerCase();
				const regionLabels: Record<string, string> = {
					main: 'Main content',
					header: 'Header',
					footer: 'Footer',
					nav: 'Navigation',
					aside: 'Sidebar',
					section: 'Section',
					article: 'Article',
					form: 'Form',
					dialog: 'Dialog'
				};
				region = regionLabels[regionTag] ?? regionTag;

				// Try to get aria-label of the region for more context.
				const regionLabel = regionEl.getAttribute('aria-label');
				if (regionLabel) {
					region = `${region} "${regionLabel}"`;
				}
			}

			const textSnippet = textContent ? textContent.slice(0, 80) : undefined;

			return { tagName, role, name: name || undefined, region, textSnippet };
		});

		const typeLabels: Record<string, string> = {
			button: 'Button',
			a: 'Link',
			input: 'Input field',
			img: 'Image',
			select: 'Dropdown',
			textarea: 'Text area',
			table: 'Table',
			form: 'Form',
			nav: 'Navigation',
			ul: 'List',
			ol: 'Numbered list',
			li: 'List item',
			h1: 'Heading (H1)',
			h2: 'Heading (H2)',
			h3: 'Heading (H3)',
			h4: 'Heading (H4)',
			h5: 'Heading (H5)',
			h6: 'Heading (H6)',
			div: 'Container',
			span: 'Text element',
			p: 'Paragraph',
			label: 'Label',
			iframe: 'Embedded frame',
			video: 'Video',
			audio: 'Audio'
		};

		const typeWord =
			info.role === 'button' || info.tagName === 'button'
				? 'Button'
				: info.role === 'link' || info.tagName === 'a'
					? 'Link'
					: info.role === 'img' || info.tagName === 'img'
						? 'Image'
						: info.role
							? `${info.role.charAt(0).toUpperCase() + info.role.slice(1)} element`
							: (typeLabels[info.tagName] ?? `<${info.tagName}> element`);

		let label = typeWord;
		if (info.name) {
			label = `${typeWord} "${info.name}"`;
		}
		if (info.region) {
			label += ` in ${info.region}`;
		}

		return {
			label,
			tagName: info.tagName,
			region: info.region,
			...(info.role !== undefined ? { role: info.role } : {}),
			...(info.name !== undefined ? { name: info.name } : {}),
			...(info.textSnippet !== undefined ? { textSnippet: info.textSnippet } : {}),
			selector
		};
	} catch {
		return undefined;
	}
}
