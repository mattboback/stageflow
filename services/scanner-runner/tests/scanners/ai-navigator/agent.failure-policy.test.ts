/**
 * Failure-policy guard tests for the ai-navigator agent loop. The guards are
 * the single useful idea ported from the legacy agent-harness turn loop.
 */

import type { Page } from 'playwright';

import { describe, expect, it, vi } from 'vitest';

import type {
	ActionDecider,
	ActionDecision,
	AgentGoal,
	PageAnalyzer,
	PagePerception
} from '../../../src/ai';
import type { ScreenshotService } from '../../../src/core/screenshots';
import type { PreScanAction, ScannerLogger } from '../../../src/core/types';

import { runAiNavigatorAgent } from '../../../src/scanners/ai-navigator/agent';

const writeFileMock = vi.hoisted(() => vi.fn().mockResolvedValue(undefined));

vi.mock('node:fs/promises', async () => {
	const actual = await vi.importActual<typeof import('node:fs/promises')>('node:fs/promises');
	return {
		...actual,
		writeFile: writeFileMock
	};
});

function makeLogger(): ScannerLogger {
	return {
		info: vi.fn(),
		warn: vi.fn(),
		error: vi.fn(),
		debug: vi.fn()
	};
}

interface FakePageState {
	url: string;
	domLength: number;
}

function makePage(state: FakePageState): Page {
	return {
		url: () => state.url,
		title: () => Promise.resolve('title'),
		screenshot: () => Promise.resolve(Buffer.from([])),
		waitForTimeout: vi.fn().mockResolvedValue(undefined),
		evaluate: vi.fn().mockImplementation(() => Promise.resolve(state.domLength)),
		// goal-checker may call locator/getByText, but these tests don't set
		// successCriteria so checkGoal returns confidence 0.2 without touching
		// the page.
		locator: vi.fn(),
		getByText: vi.fn()
	} as unknown as Page;
}

function makeScreenshotService(): ScreenshotService {
	return {
		captureFullPage: vi.fn().mockResolvedValue({ buffer: Buffer.from([]), height: 1, width: 1 }),
		captureWithHighlights: vi
			.fn()
			.mockResolvedValue({ buffer: Buffer.from([]), height: 1, width: 1 })
	};
}

function makeAnalyzer(): PageAnalyzer {
	const perception: PagePerception = {
		url: 'https://app.example.com/x',
		title: 'x',
		pageType: 'other',
		description: '',
		interactiveElements: []
	};
	return { analyze: vi.fn().mockResolvedValue(perception) } as unknown as PageAnalyzer;
}

function makeDecider(action: ActionDecision['action']): ActionDecider {
	const decision: ActionDecision = {
		action,
		reasoning: 'forward',
		confidence: 0.9
	};
	return { decide: vi.fn().mockResolvedValue(decision) } as unknown as ActionDecider;
}

interface RunOptions {
	goal: AgentGoal;
	page: Page;
	executor: { executePreScanActions: ReturnType<typeof vi.fn> };
	action?: ActionDecision['action'];
}

async function runAgent(opts: RunOptions): ReturnType<typeof runAiNavigatorAgent> {
	const action: PreScanAction = opts.action ?? { type: 'click', selector: '#go' };
	return runAiNavigatorAgent(opts.page, opts.goal, {
		screenshotsDir: '/tmp/screens',
		pageAnalyzer: makeAnalyzer(),
		actionDecider: makeDecider(action),
		screenshotService: makeScreenshotService(),
		logger: makeLogger(),
		preScanExecutor: { executePreScanActions: opts.executor.executePreScanActions }
	});
}

describe('runAiNavigatorAgent failure-policy guards', () => {
	it('executes configured input values but never returns them in traces or errors', async () => {
		const secret = 'agent p@ss word+1';
		const uriEncodedSecret = encodeURIComponent(secret);
		const formEncodedSecret = new URLSearchParams([['value', secret]])
			.toString()
			.slice('value='.length);
		const executor = {
			executePreScanActions: vi
				.fn<(page: Page, actions: PreScanAction[]) => Promise<void>>()
				.mockRejectedValue(new Error(`could not submit ${secret}`))
		};

		const result = await runAgent({
			goal: {
				objective: `Submit the configured value, never ${secret}`,
				maxSteps: 1,
				maxConsecutiveFailures: 1,
				inputValues: { accountPassword: secret }
			},
			page: makePage({
				url: `https://app.example.com/form?draft=${formEncodedSecret}`,
				domLength: 100
			}),
			executor,
			action: {
				type: 'fill',
				selector: '#password',
				value: secret,
				valueKey: 'accountPassword'
			}
		});

		expect(executor.executePreScanActions).toHaveBeenCalledWith(
			expect.anything(),
			[
				expect.objectContaining({
					type: 'fill',
					value: secret,
					valueKey: 'accountPassword'
				})
			],
			undefined,
			{ maskInputValues: true }
		);
		expect(result.goal).toMatchObject({
			objective: 'Submit the configured value, never [REDACTED]',
			inputValueKeys: ['accountPassword']
		});
		expect(result.goal).not.toHaveProperty('inputValues');
		expect(result.steps[0]?.action).toEqual({
			type: 'fill',
			selector: '#password',
			value: '[REDACTED]',
			valueKey: 'accountPassword'
		});
		expect(result.finalUrl).toBe('https://app.example.com/form?draft=[REDACTED]');
		expect(JSON.stringify(result)).not.toContain(secret);
		expect(JSON.stringify(result)).not.toContain(uriEncodedSecret);
		expect(JSON.stringify(result)).not.toContain(formEncodedSecret);
		expect(JSON.stringify(result)).toContain('[REDACTED]');
	});

	it('keeps going after one failure but stops once consecutive failures hit the threshold', async () => {
		writeFileMock.mockResolvedValue(undefined);
		const goal: AgentGoal = { objective: 'demo', maxSteps: 10, maxConsecutiveFailures: 3 };
		const executor = {
			executePreScanActions: vi
				.fn<(page: Page, actions: PreScanAction[]) => Promise<void>>()
				.mockRejectedValue(new Error('transient click failure'))
		};

		const result = await runAgent({
			goal,
			page: makePage({ url: 'https://app.example.com/x', domLength: 100 }),
			executor
		});

		expect(executor.executePreScanActions).toHaveBeenCalledTimes(3);
		expect(result.steps.filter((s) => !s.success)).toHaveLength(3);
		expect(result.steps.every((s) => !s.success)).toBe(true);
		expect(result.stuckReason).toContain('Stopped after 3 consecutive failed action attempts');
	});

	it('stops when N successful turns produced no observable URL or DOM signature change', async () => {
		writeFileMock.mockResolvedValue(undefined);
		const goal: AgentGoal = { objective: 'demo', maxSteps: 10, maxNoProgressTurns: 3 };
		const pageState: FakePageState = { url: 'https://app.example.com/x', domLength: 42 };
		const page = makePage(pageState);

		const executor = {
			executePreScanActions: vi.fn().mockResolvedValue(undefined)
		};

		const result = await runAgent({
			goal,
			page,
			executor
		});

		// Each successful turn produces the same signature, so noProgressTurns
		// reaches 3 on the fourth identical turn.
		expect(result.stuckReason).toContain('No observable progress in 3 successful turns');
		expect(result.steps.length).toBeLessThan(10);
		expect(result.steps.every((s) => s.success)).toBe(true);
	});

	it('continues iterating when each turn changes the URL or DOM signature', async () => {
		writeFileMock.mockResolvedValue(undefined);
		const goal: AgentGoal = {
			objective: 'demo',
			maxSteps: 4,
			maxNoProgressTurns: 3
		};
		const pageState: FakePageState = { url: 'https://app.example.com/x', domLength: 100 };
		const page = makePage(pageState);

		// page.url() and page.evaluate(domLength) read live state; mutate per turn.
		const executor = {
			executePreScanActions: vi.fn().mockImplementation(() => {
				pageState.url = `${pageState.url}/next`;
				pageState.domLength += 50;
				return Promise.resolve();
			})
		};

		const result = await runAgent({
			goal,
			page,
			executor
		});

		expect(result.steps.every((s) => s.success)).toBe(true);
		// Neither failure guard fires; the loop completes its maxSteps budget. The
		// stuckReason set after the loop comes from goal-checker (no success
		// criteria), not from the guards we just ported.
		expect(result.stuckReason).not.toContain('No observable progress');
		expect(result.stuckReason).not.toContain('consecutive failed action attempts');
		expect(result.steps).toHaveLength(4);
	});
});
