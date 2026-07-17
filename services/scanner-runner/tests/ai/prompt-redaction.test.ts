import type { Page } from 'playwright';

import { describe, expect, it, vi } from 'vitest';

import type { AgentGoal, PagePerception } from '../../src/ai/types';
import type { VisionClient } from '../../src/ai/vision-client';

import { ActionDecider } from '../../src/ai/action-decider';
import { PageAnalyzer } from '../../src/ai/page-analyzer';

const INPUT_CANARY = 'vision p@ss word+1';
const URI_ENCODED_CANARY = encodeURIComponent(INPUT_CANARY);
const FORM_ENCODED_CANARY = new URLSearchParams([['value', INPUT_CANARY]])
	.toString()
	.slice('value='.length);

function expectNoInputCanary(value: string): void {
	expect(value).not.toContain(INPUT_CANARY);
	expect(value).not.toContain(URI_ENCODED_CANARY);
	expect(value).not.toContain(FORM_ENCODED_CANARY);
}

function makeVisionClient(responseContent: string): {
	client: VisionClient;
	analyze: ReturnType<typeof vi.fn>;
} {
	const analyze = vi.fn().mockResolvedValue({ content: responseContent });
	return {
		client: { analyze } as unknown as VisionClient,
		analyze
	};
}

function makeGoal(): AgentGoal {
	return {
		objective: `Submit ${INPUT_CANARY}`,
		inputValues: { privateValue: INPUT_CANARY }
	};
}

describe('AI prompt redaction', () => {
	it('removes configured input values from page-analysis prompts', async () => {
		const { client, analyze } = makeVisionClient('{"pageType":"form","description":"A form"}');
		const page = {
			url: () => `https://example.com/form?draft=${FORM_ENCODED_CANARY}`,
			title: vi.fn().mockResolvedValue(`Editing ${INPUT_CANARY}`),
			screenshot: vi.fn().mockResolvedValue(Buffer.from('screenshot')),
			evaluate: vi.fn().mockResolvedValue([
				{
					selector: '#private-field',
					tagName: 'input',
					accessibleName: `Draft ${INPUT_CANARY}`,
					text: INPUT_CANARY,
					isVisible: true,
					isEnabled: true
				}
			])
		} as unknown as Page;

		await new PageAnalyzer(client).analyze(page, makeGoal());

		const prompt = analyze.mock.calls[0]?.[1] as string;
		expectNoInputCanary(prompt);
		expect(prompt).toContain('[REDACTED]');
	});

	it('removes configured input values from action-decision prompts', async () => {
		const { client, analyze } = makeVisionClient(
			'{"action":{"type":"done"},"reasoning":"complete","confidence":0.9}'
		);
		const page = {
			screenshot: vi.fn().mockResolvedValue(Buffer.from('screenshot')),
			url: () => `https://example.com/form?draft=${URI_ENCODED_CANARY}`
		} as unknown as Page;
		const perception: PagePerception = {
			url: `https://example.com/form?draft=${FORM_ENCODED_CANARY}`,
			title: `Editing ${INPUT_CANARY}`,
			pageType: 'form',
			description: `Contains ${INPUT_CANARY}`,
			interactiveElements: [
				{
					selector: '#private-field',
					tagName: 'input',
					accessibleName: `Draft ${INPUT_CANARY}`,
					text: INPUT_CANARY,
					isVisible: true,
					isEnabled: true
				}
			]
		};

		await new ActionDecider(client).decide(page, perception, makeGoal(), []);

		const prompt = analyze.mock.calls[0]?.[1] as string;
		expectNoInputCanary(prompt);
		expect(prompt).toContain('[REDACTED]');
	});
});
