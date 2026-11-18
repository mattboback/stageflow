export interface RuleBehavior {
	summary?: string;
	owner?: string;
	tags?: string[];
	displayMode?: 'expanded' | 'collapsed';
	references?: string[];
}

interface RuleBehaviorFile {
	defaults?: RuleBehavior;
	rules?: Record<string, RuleBehavior>;
}

const BUILT_IN_BEHAVIORS: RuleBehaviorFile = {
	defaults: {
		displayMode: 'expanded'
	},
	rules: {
		'color-contrast': {
			summary: 'Ensure text/background contrast meets WCAG AA ratios.',
			owner: 'Design',
			tags: ['WCAG 1.4.3', 'Level AA']
		},
		'color-contrast-enhanced': {
			summary: 'Ensure text/background contrast meets WCAG AAA ratios.',
			owner: 'Design',
			tags: ['WCAG 1.4.6', 'Level AAA']
		},
		'image-alt': {
			summary: 'Images must have alternative text describing their content.',
			owner: 'Content',
			tags: ['WCAG 1.1.1', 'Level A']
		},
		'focus-visible': {
			summary: 'Interactive elements must have visible focus indicators.',
			owner: 'Design / Frontend',
			tags: ['WCAG 2.4.7', 'Level AA']
		},
		'target-size': {
			summary: 'Touch targets should be at least 44x44 CSS pixels.',
			owner: 'Design',
			tags: ['WCAG 2.5.5', 'Level AAA']
		},
		'link-in-text-block': {
			summary: 'Links within text must be distinguishable without relying on color alone.',
			owner: 'Design',
			tags: ['WCAG 1.4.1', 'Level A']
		},

		'language-of-page': {
			summary: 'Add a lang attribute to the <html> tag.',
			owner: 'Content / Localization',
			tags: ['WCAG 3.1.1', 'Level A'],
			displayMode: 'collapsed'
		},
		'html-has-lang': {
			summary: 'Set the default page language via <html lang="...">.',
			owner: 'Content / Localization',
			tags: ['WCAG 3.1.1'],
			displayMode: 'collapsed'
		},
		'valid-lang': {
			summary: 'The lang attribute must use a valid BCP 47 language tag.',
			owner: 'Content',
			tags: ['WCAG 3.1.1'],
			displayMode: 'collapsed'
		},
		'html-lang-valid': {
			summary: 'The lang attribute must use a valid language code.',
			owner: 'Content',
			tags: ['WCAG 3.1.1'],
			displayMode: 'collapsed'
		},
		keyboard: {
			summary: 'Every interactive element must be reachable via keyboard.',
			owner: 'Frontend / Interaction',
			tags: ['WCAG 2.1.1', 'Level A'],
			displayMode: 'collapsed'
		},
		'landmark-one-main': {
			summary: 'Each page should contain exactly one <main> landmark.',
			owner: 'Frontend / IA',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'landmark-unique': {
			summary: 'Landmarks must have unique labels when the same role appears multiple times.',
			owner: 'Frontend / IA',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		region: {
			summary: 'Wrap major page areas in landmark roles.',
			owner: 'Frontend / IA',
			tags: ['WCAG 2.4.1'],
			displayMode: 'collapsed'
		},
		'multiple-ways': {
			summary: 'Provide multiple navigation mechanisms.',
			owner: 'Product / IA',
			tags: ['WCAG 2.4.5', 'Level AA'],
			displayMode: 'collapsed'
		},
		'link-name': {
			summary: 'Links need discernible text or aria-labels.',
			owner: 'Frontend',
			tags: ['WCAG 2.4.4']
		},
		'button-name': {
			summary: 'Buttons must have discernible accessible names.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2']
		},
		label: {
			summary: 'Form inputs must have associated labels.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1', 'Level A']
		},
		'form-field-multiple-labels': {
			summary: 'Form fields should not have multiple labels.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'aria-valid-attr': {
			summary: 'ARIA attributes must be valid.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'aria-valid-attr-value': {
			summary: 'ARIA attribute values must be valid.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'aria-allowed-attr': {
			summary: "ARIA attributes must be allowed for the element's role.",
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'aria-required-attr': {
			summary: 'Required ARIA attributes must be present.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'aria-required-children': {
			summary: 'ARIA roles must contain required child roles.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'aria-required-parent': {
			summary: 'ARIA roles must be contained by required parent roles.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'aria-roles': {
			summary: 'ARIA roles must be valid.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'aria-hidden-focus': {
			summary: 'Elements with aria-hidden should not contain focusable elements.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'aria-hidden-body': {
			summary: 'aria-hidden should not be set on the document body.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'duplicate-id': {
			summary: 'Element IDs must be unique.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.1'],
			displayMode: 'collapsed'
		},
		'duplicate-id-active': {
			summary: 'Active element IDs must be unique.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.1'],
			displayMode: 'collapsed'
		},
		'duplicate-id-aria': {
			summary: 'IDs used in ARIA must be unique.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.1'],
			displayMode: 'collapsed'
		},
		'heading-order': {
			summary: 'Heading levels should increase by one.',
			owner: 'Content / IA',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'empty-heading': {
			summary: 'Headings must have discernible text.',
			owner: 'Content',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		bypass: {
			summary: 'Page must have a way to bypass repeated content.',
			owner: 'Frontend / IA',
			tags: ['WCAG 2.4.1', 'Level A'],
			displayMode: 'collapsed'
		},
		'skip-link': {
			summary: 'Skip links should be provided.',
			owner: 'Frontend',
			tags: ['WCAG 2.4.1'],
			displayMode: 'collapsed'
		},
		tabindex: {
			summary: 'tabindex values should not be greater than 0.',
			owner: 'Frontend',
			tags: ['WCAG 2.4.3'],
			displayMode: 'collapsed'
		},
		'document-title': {
			summary: 'Documents must have a <title> element.',
			owner: 'Content',
			tags: ['WCAG 2.4.2', 'Level A'],
			displayMode: 'collapsed'
		},
		'meta-viewport': {
			summary: 'Zooming and scaling should not be disabled.',
			owner: 'Frontend',
			tags: ['WCAG 1.4.4'],
			displayMode: 'collapsed'
		},
		'meta-refresh': {
			summary: 'Pages should not auto-refresh.',
			owner: 'Frontend',
			tags: ['WCAG 2.2.1'],
			displayMode: 'collapsed'
		},
		'html-xml-lang-mismatch': {
			summary: 'HTML lang and xml:lang attributes must match.',
			owner: 'Content',
			tags: ['WCAG 3.1.1'],
			displayMode: 'collapsed'
		},
		list: {
			summary: 'Lists must be marked up correctly.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		listitem: {
			summary: 'List items must be contained in a list.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'definition-list': {
			summary: 'Definition lists must be marked up correctly.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		dlitem: {
			summary: 'Definition list items must be contained in a definition list.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'table-fake-caption': {
			summary: 'Tables should use <caption> instead of fake captions.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'td-headers-attr': {
			summary: 'Table header references must be valid.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'th-has-data-cells': {
			summary: 'Table headers must be associated with data cells.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'scope-attr-valid': {
			summary: 'The scope attribute must be valid.',
			owner: 'Frontend',
			tags: ['WCAG 1.3.1'],
			displayMode: 'collapsed'
		},
		'video-caption': {
			summary: 'Videos must have captions.',
			owner: 'Content',
			tags: ['WCAG 1.2.2', 'Level A'],
			displayMode: 'collapsed'
		},
		'audio-caption': {
			summary: 'Audio must have captions or transcripts.',
			owner: 'Content',
			tags: ['WCAG 1.2.1'],
			displayMode: 'collapsed'
		},
		'frame-title': {
			summary: 'Frames must have accessible names.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'frame-title-unique': {
			summary: 'Frame titles should be unique.',
			owner: 'Frontend',
			tags: ['WCAG 4.1.2'],
			displayMode: 'collapsed'
		},
		'server-side-image-map': {
			summary: 'Server-side image maps should not be used.',
			owner: 'Frontend',
			tags: ['WCAG 2.1.1'],
			displayMode: 'collapsed'
		},
		'no-autoplay-audio': {
			summary: 'Audio should not autoplay.',
			owner: 'Frontend',
			tags: ['WCAG 1.4.2'],
			displayMode: 'collapsed'
		},
		blink: {
			summary: 'Blinking content should be avoided.',
			owner: 'Design',
			tags: ['WCAG 2.2.2'],
			displayMode: 'collapsed'
		},
		marquee: {
			summary: 'Marquee elements should not be used.',
			owner: 'Frontend',
			tags: ['WCAG 2.2.2'],
			displayMode: 'collapsed'
		}
	}
};

export function getRuleBehavior(ruleId?: string): RuleBehavior {
	const rule = ruleId ? BUILT_IN_BEHAVIORS.rules?.[ruleId] : undefined;
	return { ...BUILT_IN_BEHAVIORS.defaults, ...rule };
}
