import type { BrowserContext, Page } from 'playwright';

import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { ActionDecider, AgentResult, PageAnalyzer } from '../../../src/ai';
import type { BrowserManager } from '../../../src/core/browser-manager';
import type { ScreenshotService } from '../../../src/core/screenshots';
import type { ScanContext, ScannerConfig, ScannerLogger } from '../../../src/core/types';
import type { AiNavigatorOptions } from '../../../src/scanners/ai-navigator/options';

const runAgentMock = vi.hoisted(() => vi.fn());

vi.mock('../../../src/scanners/ai-navigator/agent', () => ({
	runAiNavigatorAgent: runAgentMock
}));

import { AiNavigatorScanner } from '../../../src/scanners/ai-navigator';

const URL_CANARY = 'persisted p@ss word+1';
const FORM_ENCODED_URL_CANARY = new URLSearchParams([['token', URL_CANARY]])
	.toString()
	.slice('token='.length);

interface AiNavigatorInternals {
	options: AiNavigatorOptions;
	browserManager: BrowserManager;
	screenshotService: ScreenshotService;
	pageAnalyzer: PageAnalyzer;
	actionDecider: ActionDecider;
}

function makeLogger(): ScannerLogger {
	return {
		info: vi.fn(),
		warn: vi.fn(),
		error: vi.fn(),
		debug: vi.fn()
	};
}

describe('AiNavigatorScanner', () => {
	let resultsDir: string;

	beforeEach(async () => {
		resultsDir = await mkdtemp(path.join(tmpdir(), 'stageflow-ai-navigator-'));
		runAgentMock.mockReset();
	});

	afterEach(async () => {
		await rm(resultsDir, { recursive: true, force: true });
	});

	it('persists and publishes the sanitized agent final URL', async () => {
		const rawUrl = `https://example.com/complete?token=${FORM_ENCODED_URL_CANARY}`;
		const sanitizedUrl = 'https://example.com/complete?token=[REDACTED]';
		const agentResult: AgentResult = {
			success: true,
			goal: { objective: 'Complete the flow', inputValueKeys: ['token'] },
			startUrl: 'https://example.com/start',
			finalUrl: sanitizedUrl,
			steps: [],
			totalSteps: 0,
			totalDurationMs: 1
		};
		runAgentMock.mockResolvedValue(agentResult);

		const scanner = new AiNavigatorScanner();
		const internals = scanner as unknown as AiNavigatorInternals;
		internals.options = {
			goal: { objective: 'Complete the flow', inputValues: { token: URL_CANARY } },
			vision: { provider: 'openrouter', model: 'test/model' }
		};
		internals.browserManager = {} as BrowserManager;
		internals.screenshotService = {} as ScreenshotService;
		internals.pageAnalyzer = {} as PageAnalyzer;
		internals.actionDecider = {} as ActionDecider;

		const context: ScanContext = {
			page: { url: () => rawUrl } as unknown as Page,
			context: {} as BrowserContext,
			pageEntry: {
				id: 'complete',
				path: '/complete',
				url: rawUrl
			},
			resultsDir,
			config: {} as ScannerConfig,
			logger: makeLogger()
		};

		const result = await scanner.scanPage(context);
		const persistedTrace = await readFile(path.join(resultsDir, 'ai-trace.json'), 'utf8');

		expect(result.url).toBe(sanitizedUrl);
		expect(JSON.stringify(result)).not.toContain(URL_CANARY);
		expect(JSON.stringify(result)).not.toContain(FORM_ENCODED_URL_CANARY);
		expect(persistedTrace).not.toContain(URL_CANARY);
		expect(persistedTrace).not.toContain(FORM_ENCODED_URL_CANARY);
		expect(persistedTrace).toContain('[REDACTED]');
	});
});
